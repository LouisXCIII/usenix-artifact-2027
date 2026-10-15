package chaincode

// S4 harness. env: S4_FIXTURES = fixtures.json (from make_fixtures.py); S4_OUT = timings.json; S4_REPS (default 30).
// Skipped when S4_FIXTURES is unset.

import (
	"encoding/json"
	"os"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

type s4Fixture struct {
	Name                    string                 `json:"name"`
	SrcBytes                int                    `json:"src_bytes"`
	X                       []int64                `json:"x_q"`
	Pred                    int                    `json:"pred"`
	Margin                  int64                  `json:"margin"`
	StrictCircuitSatisfiabl bool                   `json:"strict_circuit_satisfiable"`
	FeatureHash             string                 `json:"feature_hash"`
	Proof                   map[string]interface{} `json:"proof"`
	Public                  []string               `json:"public"`
	SnarkjsOK               bool                   `json:"snarkjs_verify_ok"`
}

func s4Load(t testing.TB) ([]s4Fixture, string) {
	p := os.Getenv("S4_FIXTURES")
	if p == "" {
		t.Skip("S4_FIXTURES not set")
	}
	raw, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	var f struct {
		Files []s4Fixture `json:"files"`
	}
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatal(err)
	}
	wb, err := os.ReadFile("b1_model_weights.json")
	if err != nil {
		t.Fatal(err)
	}
	return f.Files, string(wb)
}

func s4JSON(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}

// Correctness first: B1 (both variants) must accept exactly when the strict circuit relation is satisfiable,
// must reproduce the fixture's Poseidon feature hash, and the fixtures' Groth16 proofs must pass the pairing check.
func TestS4Parity(t *testing.T) {
	fx, wb := s4Load(t)
	accept, reject := 0, 0
	for _, f := range fx {
		xJSON := s4JSON(f.X)
		okS, fhS, dS, err := b1Reevaluate(xJSON, "", 0, requiredMinMargin)
		if err != nil {
			t.Fatalf("%s store: %v", f.Name, err)
		}
		okN, fhN, dN, err := b1Reevaluate(xJSON, wb, 0, requiredMinMargin)
		if err != nil {
			t.Fatalf("%s nostore: %v", f.Name, err)
		}
		if okS != f.StrictCircuitSatisfiabl || okN != f.StrictCircuitSatisfiabl {
			t.Errorf("%s: B1 store=%v (%s) nostore=%v (%s), strict circuit satisfiable=%v", f.Name, okS, dS, okN, dN, f.StrictCircuitSatisfiabl)
		}
		if fhS != f.FeatureHash || fhN != f.FeatureHash {
			t.Errorf("%s: recomputed featureHash %s != fixture %s", f.Name, fhS, f.FeatureHash)
		}
		if okA, _, err := b1ReevaluateSHA(xJSON, 0, requiredMinMargin); err != nil || okA != f.StrictCircuitSatisfiabl {
			t.Errorf("%s: B1-sha=%v err=%v, strict circuit satisfiable=%v", f.Name, okA, err, f.StrictCircuitSatisfiabl)
		}
		if f.StrictCircuitSatisfiabl {
			accept++
		} else {
			reject++
		}
		if f.Proof != nil {
			ok, detail, err := verifySemBoundProof(s4JSON(f.Proof), s4JSON(f.Public))
			if err != nil || (!ok && !strings.HasPrefix(detail, "proof verifies, but")) {
				t.Errorf("%s: Groth16 fixture does not pass the pairing check: ok=%v detail=%q err=%v", f.Name, ok, detail, err)
			}
		}
	}
	t.Logf("parity over %d real files: %d accepted, %d rejected, all equal to the circuit's own satisfiability", len(fx), accept, reject)
}

// Tamper checks: each must be rejected or produce a different digest (which verifyFeatureBinding would refuse).
func TestS4Tamper(t *testing.T) {
	fx, wb := s4Load(t)
	var f s4Fixture
	for _, c := range fx {
		if c.StrictCircuitSatisfiabl {
			f = c
			break
		}
	}
	if f.Name == "" {
		t.Skip("no strict-satisfiable fixture")
	}
	x2 := append([]int64(nil), f.X...)
	x2[0]++
	_, fh, _, _ := b1Reevaluate(s4JSON(x2), "", 0, requiredMinMargin)
	if fh == f.FeatureHash {
		t.Error("flipping one feature did not change the digest")
	}
	var m struct {
		W [][]int64 `json:"W_q"`
		B []int64   `json:"b_q"`
	}
	_ = json.Unmarshal([]byte(wb), &m)
	m.W[0][0]++
	ok, _, d, _ := b1Reevaluate(s4JSON(f.X), s4JSON(m), 0, requiredMinMargin)
	if ok || !strings.Contains(d, "not the pinned model") {
		t.Errorf("altered weights accepted or wrong reason: ok=%v %q", ok, d)
	}
	if ok, _, _, _ := b1Reevaluate(s4JSON(f.X), "", 0, f.Margin+1); ok {
		t.Error("margin above the real margin was accepted")
	}
	if ok, _, _, _ := b1Reevaluate(s4JSON(f.X[:255]), "", 0, requiredMinMargin); ok {
		t.Error("short feature vector accepted")
	}
}

type s4Timing struct {
	Name         string  `json:"name"`
	SrcBytes     int     `json:"src_bytes"`
	B4Groth16    []int64 `json:"b4_groth16_ns"`
	B1Store      []int64 `json:"b1_store_ns"`
	B1NoStore    []int64 `json:"b1_nostore_ns"`
	B1SHA        []int64 `json:"b1_sha_ns"`
	XJSONBytes   int     `json:"x_json_bytes"`
	WbJSONBytes  int     `json:"wb_json_bytes"`
	ProofBytes   int     `json:"proof_json_bytes"`
	PublicBytes  int     `json:"public_json_bytes"`
	StrictAccept bool    `json:"strict_accept"`
}

func TestS4Timing(t *testing.T) {
	fx, wb := s4Load(t)
	reps := 30
	if v := os.Getenv("S4_REPS"); v != "" {
		reps, _ = strconv.Atoi(v)
	}
	if err := b1Provision(); err != nil {
		t.Fatal(err)
	}
	var sink int
	type prep struct{ x, pf, pub string }
	var res []s4Timing
	for _, f := range fx {
		if f.Proof == nil {
			continue
		}
		p := prep{s4JSON(f.X), s4JSON(f.Proof), s4JSON(f.Public)}
		run := []func() int64{
			func() int64 { // B4: the destination's Groth16 verification, argument decoding included (pairing runs before policy checks)
				t0 := time.Now()
				ok, d, _ := verifySemBoundProof(p.pf, p.pub)
				el := time.Since(t0).Nanoseconds()
				if ok || !strings.HasPrefix(d, "proof verifies, but") {
					t.Fatalf("%s: unexpected Groth16 outcome %v %q", f.Name, ok, d)
				}
				sink += len(d)
				return el
			},
			func() int64 { // B1-store
				t0 := time.Now()
				ok, fh, _, _ := b1Reevaluate(p.x, "", 0, requiredMinMargin)
				el := time.Since(t0).Nanoseconds()
				sink += len(fh)
				if ok {
					sink++
				}
				return el
			},
			func() int64 { // B1-nostore
				t0 := time.Now()
				ok, fh, _, _ := b1Reevaluate(p.x, wb, 0, requiredMinMargin)
				el := time.Since(t0).Nanoseconds()
				sink += len(fh)
				if ok {
					sink++
				}
				return el
			},
			func() int64 { // B1-sha (amendment A1): SHA-256 commitment to x, stored model
				t0 := time.Now()
				ok, dg, _ := b1ReevaluateSHA(p.x, 0, requiredMinMargin)
				el := time.Since(t0).Nanoseconds()
				sink += int(dg[0])
				if ok {
					sink++
				}
				return el
			},
		}
		tm := s4Timing{Name: f.Name, SrcBytes: f.SrcBytes, XJSONBytes: len(p.x), WbJSONBytes: len(wb),
			ProofBytes: len(p.pf), PublicBytes: len(p.pub), StrictAccept: f.StrictCircuitSatisfiabl}
		for w := 0; w < 10; w++ { // warm-up, not recorded
			for _, r := range run {
				r()
			}
		}
		for i := 0; i < reps; i++ {
			runtime.GC()
			// rotate the order so no variant is always first (temporal-order artefact seen earlier in this project)
			for k := 0; k < len(run); k++ {
				idx := (i + k) % len(run)
				el := run[idx]()
				switch idx {
				case 0:
					tm.B4Groth16 = append(tm.B4Groth16, el)
				case 1:
					tm.B1Store = append(tm.B1Store, el)
				case 2:
					tm.B1NoStore = append(tm.B1NoStore, el)
				case 3:
					tm.B1SHA = append(tm.B1SHA, el)
				}
			}
		}
		res = append(res, tm)
	}
	out := os.Getenv("S4_OUT")
	if out == "" {
		t.Fatal("S4_OUT not set")
	}
	b, _ := json.MarshalIndent(map[string]interface{}{"reps": reps, "files": res, "sink": sink, "goos": runtime.GOOS, "gomaxprocs": runtime.GOMAXPROCS(0)}, "", " ")
	if err := os.WriteFile(out, b, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Logf("timed %d files x %d reps x 4 variants -> %s", len(res), reps, out)
}

// Component breakdown on the first fixture (medians over S4_COMP_REPS, default 300), so the totals can be explained.
func TestS4Components(t *testing.T) {
	fx, wb := s4Load(t)
	f := fx[0]
	reps := 300
	if v := os.Getenv("S4_COMP_REPS"); v != "" {
		reps, _ = strconv.Atoi(v)
	}
	_ = b1Provision()
	xJSON, pf, pub := s4JSON(f.X), s4JSON(f.Proof), s4JSON(f.Public)
	med := func(fn func()) float64 {
		for i := 0; i < 20; i++ {
			fn()
		}
		s := make([]int64, reps)
		for i := range s {
			t0 := time.Now()
			fn()
			s[i] = time.Since(t0).Nanoseconds()
		}
		for i := 1; i < len(s); i++ {
			for j := i; j > 0 && s[j-1] > s[j]; j-- {
				s[j-1], s[j] = s[j], s[j-1]
			}
		}
		return float64(s[len(s)/2]) / 1e6
	}
	var x []int64
	_ = json.Unmarshal([]byte(xJSON), &x)
	var pfS groth16Proof
	_ = json.Unmarshal([]byte(pf), &pfS)
	var pubS []string
	_ = json.Unmarshal([]byte(pub), &pubS)
	var vk groth16VKey
	res := map[string]float64{
		"b1_parse_x_json_ms":        med(func() { var v []int64; _ = json.Unmarshal([]byte(xJSON), &v) }),
		"b1_poseidon_feature_ms":    med(func() { _ = b1FeatureDigest(x) }),
		"b1_sha256_feature_ms":      med(func() { _, _, _ = b1ReevaluateSHA(xJSON, 0, requiredMinMargin) }),
		"b1_relation_ms":            med(func() { _, _ = b1Relation(x, &b1Store, 0, requiredMinMargin) }),
		"b1_parse_wb_json_ms":       med(func() { _, _ = b1ParseModel(wb) }),
		"b1_poseidon_weights_ms":    med(func() { _ = b1WeightsDigest(&b1Store) }),
		"b4_parse_proof_public_ms":  med(func() { var a groth16Proof; var b []string; _ = json.Unmarshal([]byte(pf), &a); _ = json.Unmarshal([]byte(pub), &b) }),
		"b4_parse_vk_ms":            med(func() { _ = json.Unmarshal(verificationKeyV7JSON, &vk) }),
		"b4_verifyGroth16_core_ms":  med(func() { _, _ = verifyGroth16(vk, pfS, pubS) }),
		"b4_verifySemBoundProof_ms": med(func() { _, _, _ = verifySemBoundProof(pf, pub) }),
	}
	b, _ := json.MarshalIndent(res, "", " ")
	if out := os.Getenv("S4_COMP_OUT"); out != "" {
		_ = os.WriteFile(out, b, 0o644)
	}
	t.Logf("%s", b)
}
