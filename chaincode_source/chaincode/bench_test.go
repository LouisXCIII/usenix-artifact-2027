package chaincode

import (
	"encoding/json"
	"os"
	"strconv"
	"strings"
	"testing"
)

func benchTD(b *testing.B, name string) string {
	b.Helper()
	d, err := os.ReadFile("testdata/" + name)
	if err != nil {
		b.Fatal(err)
	}
	return strings.TrimSpace(string(d))
}

// Header: rebuild two header encodings, recover the commit seals, check the pinned quorum, and prove
// two registry slots by Merkle-Patricia proofs (verifySourceState does all of it).
func BenchmarkSourceState_HeaderAndTwoSlots(b *testing.B) {
	hdr, rp := benchTD(b, "besu_header_0x5fc0c.json"), benchTD(b, "registry_proof_honest.json")
	ch, _ := unhexBytes(benchTD(b, "codehash_honest.txt"))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if st := verifySourceState(1789435861+60, ch, hdr, rp); !st.Verified {
			b.Fatal(st.Detail)
		}
	}
}

// Groth16 pairing check of the v6 relation plus the destination's policy checks.
func BenchmarkGroth16_V6(b *testing.B) {
	pf, pub := benchTD(b, "v6_honest_proof.json"), benchTD(b, "v6_honest_public.json")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if ok, d, _ := verifySemBoundProof(pf, pub); !ok {
			b.Fatal(d)
		}
	}
}

// Recompute hD and prove decisionCommit[hD] under the header's stateRoot.
func BenchmarkCommitProof(b *testing.B) {
	var s struct {
		Header       map[string]interface{} `json:"header"`
		CommitProofs map[string]struct {
			Fields map[string]string `json:"fields"`
			Proof  registryProof     `json:"proof"`
		} `json:"commitProofs"`
	}
	if err := json.Unmarshal([]byte(benchTD(b, "x3_state.json")), &s); err != nil {
		b.Fatal(err)
	}
	root, _ := unhexBytes(s.Header["stateRoot"].(string))
	cp := s.CommitProofs["D1_honest"]
	f := cp.Fields
	ch, _ := unhexBytes(f["codeHash"])
	nonce, _ := unhexBytes(f["nonce"])
	n := func(k string) int64 { v, _ := strconv.ParseInt(f[k], 10, 64); return v }
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h := decisionHash(destinationDomain, ch, f["role"], n("amount"), n("sourceLimit"), n("epoch"), n("expiry"), nonce)
		if ok, d := commitStatus(root, cp.Proof, h); !ok {
			b.Fatal(d)
		}
	}
}
