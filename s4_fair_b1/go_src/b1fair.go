// b1fair.go -- B1 ("full destination re-evaluation") implemented in the SAME language and runtime as the B4
// verification path, re-executing exactly the relation that ml_scan_proof_v7.circom proves (last layer of the
// source classifier): Poseidon digest of x == featureHash, W,b == the pinned model, logits = W x + b, argmax,
// class <= maxVulnClass, margin >= minMargin. Protocol: evidence/s4_fair_b1_protocol_2026-09-28.md.
//
// Scope: these are the pure functions used by the measurement harness. They are NOT registered as chaincode
// entry points here (an entry point needs fresh Besu MPT fixtures to be exercised end to end).
package chaincode

import (
	"crypto/sha256"
	_ "embed"
	"encoding/json"
	"fmt"
	"sync"
)

//go:embed b1_model_weights.json
var b1ModelJSON []byte

const (
	b1N       = 256
	b1C       = 12
	b1ChunkX  = 18  // ceil(256/15)
	b1ChunkW  = 206 // ceil(3084/15)
	b1MaxAbsX = int64(1) << 24
	b1MaxAbsW = int64(1) << 24
	b1Shift   = int64(1) << 40 // circuit compares logits+2^40 in 64 bits: |logit| must stay below 2^40
)

type b1Model struct {
	W [b1C][b1N]int64
	B [b1C]int64
}

var (
	b1Once  sync.Once
	b1Store b1Model // the destination's provisioned copy of the pinned model
	b1Err   error
)

// b1Provision parses the destination-side stored model once (deployment-time cost, not per call).
func b1Provision() error {
	b1Once.Do(func() {
		var m struct {
			W [][]int64 `json:"W_q"`
			B []int64   `json:"b_q"`
		}
		if err := json.Unmarshal(b1ModelJSON, &m); err != nil {
			b1Err = err
			return
		}
		if len(m.W) != b1C || len(m.B) != b1C {
			b1Err = fmt.Errorf("stored model has wrong shape")
			return
		}
		for j := 0; j < b1C; j++ {
			if len(m.W[j]) != b1N {
				b1Err = fmt.Errorf("stored model row %d has wrong length", j)
				return
			}
			copy(b1Store.W[j][:], m.W[j])
		}
		copy(b1Store.B[:], m.B)
	})
	return b1Err
}

// b1ParseModel decodes a caller-supplied (W,b) argument, as the no-store variant receives per call.
func b1ParseModel(wbJSON string) (*b1Model, error) {
	var m struct {
		W [][]int64 `json:"W_q"`
		B []int64   `json:"b_q"`
	}
	if err := json.Unmarshal([]byte(wbJSON), &m); err != nil {
		return nil, fmt.Errorf("malformed model JSON: %v", err)
	}
	if len(m.W) != b1C || len(m.B) != b1C {
		return nil, fmt.Errorf("model has wrong shape")
	}
	out := new(b1Model)
	for j := 0; j < b1C; j++ {
		if len(m.W[j]) != b1N {
			return nil, fmt.Errorf("model row %d has wrong length", j)
		}
		copy(out.W[j][:], m.W[j])
	}
	copy(out.B[:], m.B)
	return out, nil
}

func b1WeightsFlat(m *b1Model) []int64 {
	flat := make([]int64, 0, b1C*b1N+b1C)
	for j := 0; j < b1C; j++ {
		flat = append(flat, m.W[j][:]...)
	}
	return append(flat, m.B[:]...)
}

func b1FeatureDigest(x []int64) string {
	d := poseidonChain(x, b1ChunkX)
	return d.String()
}

func b1WeightsDigest(m *b1Model) string {
	d := poseidonChain(b1WeightsFlat(m), b1ChunkW)
	return d.String()
}

// b1Relation re-executes logits/argmax/margin/class exactly as the circuit does (strict '>' keeps the first
// maximum; every non-winner must trail the winner by at least minMargin; class <= maxVulnClass).
func b1Relation(x []int64, m *b1Model, maxClass int, minMargin int64) (bool, string) {
	var logits [b1C]int64
	for j := 0; j < b1C; j++ {
		s := m.B[j]
		row := &m.W[j]
		for i := 0; i < b1N; i++ {
			s += row[i] * x[i]
		}
		if s >= b1Shift || s <= -b1Shift {
			return false, "logit outside the circuit's comparator domain (|logit| >= 2^40)"
		}
		logits[j] = s
	}
	best, cur := 0, logits[0]
	for j := 1; j < b1C; j++ {
		if logits[j] > cur {
			best, cur = j, logits[j]
		}
	}
	for j := 0; j < b1C; j++ {
		if j != best && cur-logits[j] < minMargin {
			return false, "margin below the required minimum"
		}
	}
	if best > maxClass {
		return false, fmt.Sprintf("predicted class %d exceeds the accepted maximum %d", best, maxClass)
	}
	return true, fmt.Sprintf("class %d, relation satisfied", best)
}

// b1ReevaluateSHA is amendment A1 (added AFTER the first timing run, see the protocol): the strongest B1 for
// this relation. If the scanner authority also commits SHA-256(x) on-chain (one more registry slot), the
// destination binds x with SHA-256 instead of the circuit-friendly Poseidon digest, and the stored model needs
// no digest. Everything else (argument decoding, relation) is identical to the store variant.
func b1ReevaluateSHA(xJSON string, maxClass int, minMargin int64) (ok bool, digest [32]byte, err error) {
	var x []int64
	if err = json.Unmarshal([]byte(xJSON), &x); err != nil {
		return false, digest, fmt.Errorf("malformed feature vector JSON: %v", err)
	}
	if len(x) != b1N {
		return false, digest, fmt.Errorf("feature vector must have %d entries", b1N)
	}
	buf := make([]byte, 0, 8*b1N)
	for _, v := range x {
		if v >= b1MaxAbsX || v <= -b1MaxAbsX {
			return false, digest, nil
		}
		u := uint64(v)
		buf = append(buf, byte(u>>56), byte(u>>48), byte(u>>40), byte(u>>32), byte(u>>24), byte(u>>16), byte(u>>8), byte(u))
	}
	digest = sha256.Sum256(buf)
	if err = b1Provision(); err != nil {
		return false, digest, err
	}
	ok, _ = b1Relation(x, &b1Store, maxClass, minMargin)
	return ok, digest, nil
}

// b1Reevaluate is the whole per-call B1 recompute. xJSON is the feature vector as it would arrive in a
// chaincode argument; wbJSON is empty for the store variant (the provisioned model is used, no digest is
// recomputed) and non-empty for the no-store variant (the digest of the supplied model is recomputed and
// compared with the pinned hash). It returns the recomputed feature digest so the caller can bind it to the
// on-chain commitment exactly as verifyFeatureBinding does for the proof's public featureHash.
func b1Reevaluate(xJSON, wbJSON string, maxClass int, minMargin int64) (ok bool, featureHash string, detail string, err error) {
	var x []int64
	if err = json.Unmarshal([]byte(xJSON), &x); err != nil {
		return false, "", "", fmt.Errorf("malformed feature vector JSON: %v", err)
	}
	if len(x) != b1N {
		return false, "", "", fmt.Errorf("feature vector must have %d entries", b1N)
	}
	for _, v := range x {
		if v >= b1MaxAbsX || v <= -b1MaxAbsX {
			return false, "", "feature value outside the accepted magnitude", nil
		}
	}
	featureHash = b1FeatureDigest(x)
	var m *b1Model
	if wbJSON == "" {
		if err = b1Provision(); err != nil {
			return false, "", "", err
		}
		m = &b1Store
	} else {
		m, err = b1ParseModel(wbJSON)
		if err != nil {
			return false, "", "", err
		}
		for j := 0; j < b1C; j++ {
			for i := 0; i < b1N; i++ {
				if v := m.W[j][i]; v >= b1MaxAbsW || v <= -b1MaxAbsW {
					return false, featureHash, "model weight outside the accepted magnitude", nil
				}
			}
		}
		if got := b1WeightsDigest(m); got != pinnedModelWeightsHash {
			return false, featureHash, "supplied model is not the pinned model (digest mismatch)", nil
		}
	}
	ok, detail = b1Relation(x, m, maxClass, minMargin)
	return ok, featureHash, detail, nil
}
