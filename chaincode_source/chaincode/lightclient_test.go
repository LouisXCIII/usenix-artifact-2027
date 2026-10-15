package chaincode

import (
	"encoding/json"
	"math/big"
	"os"
	"strings"
	"testing"
)

const fixtureHeaderTs = 1789435861 // timestamp of block 0x5fc0c in testdata/besu_header_0x5fc0c.json

func readTD(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(string(b))
}

func codeHashBytes(t *testing.T, tag string) []byte {
	t.Helper()
	b, err := unhexBytes(readTD(t, "codehash_"+tag+".txt"))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestSourceState_HonestProven(t *testing.T) {
	st := verifySourceState(fixtureHeaderTs+60, codeHashBytes(t, "honest"),
		readTD(t, "besu_header_0x5fc0c.json"), readTD(t, "registry_proof_honest.json"))
	if !st.Verified {
		t.Fatalf("expected verified, got: %s", st.Detail)
	}
	if st.Approvals != 2 || !st.MlPassed {
		t.Fatalf("expected approvals=2 mlPassed=true, got %d %v", st.Approvals, st.MlPassed)
	}
}

func TestSourceState_DishonestProvenFalse(t *testing.T) {
	st := verifySourceState(fixtureHeaderTs+60, codeHashBytes(t, "dishonest"),
		readTD(t, "besu_header_0x5fc0c.json"), readTD(t, "registry_proof_dishonest.json"))
	if !st.Verified {
		t.Fatalf("expected verified, got: %s", st.Detail)
	}
	if st.Approvals != 2 || st.MlPassed {
		t.Fatalf("expected approvals=2 mlPassed=false, got %d %v", st.Approvals, st.MlPassed)
	}
}

func TestSourceState_ProofForWrongCodeHashRejected(t *testing.T) {
	// honest proof material presented for the dishonest codeHash: the storage keys are derived
	// from the codeHash inside the chaincode, so the supplied proofs no longer match.
	st := verifySourceState(fixtureHeaderTs+60, codeHashBytes(t, "dishonest"),
		readTD(t, "besu_header_0x5fc0c.json"), readTD(t, "registry_proof_honest.json"))
	if st.Verified {
		t.Fatal("proof for a different codeHash must not verify")
	}
}

func mutateHeader(t *testing.T, field, value string) string {
	t.Helper()
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(readTD(t, "besu_header_0x5fc0c.json")), &m); err != nil {
		t.Fatal(err)
	}
	m[field] = value
	b, _ := json.Marshal(m)
	return string(b)
}

func TestSourceState_TamperedHeaderRejected(t *testing.T) {
	for _, f := range []struct{ field, val string }{
		{"timestamp", "0x6aa89fd6"},
		{"stateRoot", "0x" + strings.Repeat("00", 31) + "01"},
		{"number", "0x5fc0d"},
	} {
		st := verifySourceState(fixtureHeaderTs+60, codeHashBytes(t, "honest"),
			mutateHeader(t, f.field, f.val), readTD(t, "registry_proof_honest.json"))
		if st.Verified {
			t.Fatalf("header with tampered %s must be rejected", f.field)
		}
	}
}

func TestSourceState_StaleHeaderRejected(t *testing.T) {
	st := verifySourceState(fixtureHeaderTs+maxHeaderAgeSeconds+10, codeHashBytes(t, "honest"),
		readTD(t, "besu_header_0x5fc0c.json"), readTD(t, "registry_proof_honest.json"))
	if st.Verified || !strings.Contains(st.Detail, "freshness") {
		t.Fatalf("stale header must be rejected for freshness, got verified=%v detail=%s", st.Verified, st.Detail)
	}
}

func TestSourceState_CorruptedProofNodeRejected(t *testing.T) {
	var rp registryProof
	if err := json.Unmarshal([]byte(readTD(t, "registry_proof_honest.json")), &rp); err != nil {
		t.Fatal(err)
	}
	for i := range rp.StorageProof {
		n := rp.StorageProof[i].Proof
		last := []byte(n[len(n)-1])
		if last[len(last)-1] == '0' {
			last[len(last)-1] = '1'
		} else {
			last[len(last)-1] = '0'
		}
		n[len(n)-1] = string(last)
	}
	b, _ := json.Marshal(rp)
	st := verifySourceState(fixtureHeaderTs+60, codeHashBytes(t, "honest"),
		readTD(t, "besu_header_0x5fc0c.json"), string(b))
	if st.Verified {
		t.Fatal("corrupted storage proof must be rejected")
	}
}

func TestSourceState_MalformedInputs(t *testing.T) {
	if verifySourceState(fixtureHeaderTs, codeHashBytes(t, "honest"), "{", "{").Verified {
		t.Fatal("malformed JSON must not verify")
	}
}

func TestFreshness(t *testing.T) {
	if checkFreshness(1000, 1100, 200) != nil {
		t.Fatal("within bound should pass")
	}
	if checkFreshness(1000, 1300, 200) == nil {
		t.Fatal("beyond bound should fail")
	}
	if checkFreshness(2000, 1000, 200) == nil {
		t.Fatal("future header should fail")
	}
}

func TestSafeBankProof_VerifiesAndIsBoundToRegistryCodeHash(t *testing.T) {
	proof := readTD(t, "safebank_proof_v5.json")
	public := readTD(t, "safebank_public_v5.json")
	ok, detail, err := verifySemBoundProof(proof, public)
	if err != nil || !ok {
		t.Fatalf("real SafeBank proof must verify: ok=%v err=%v detail=%s", ok, err, detail)
	}
	var pub []string
	if err := json.Unmarshal([]byte(public), &pub); err != nil {
		t.Fatal(err)
	}
	bound, _ := new(big.Int).SetString(pub[1], 10)
	if bound.Cmp(new(big.Int).SetBytes(codeHashBytes(t, "honest"))) != 0 {
		t.Fatal("public solidityCodeHash must equal the registry's HONEST codeHash")
	}
	if bound.Cmp(new(big.Int).SetBytes(codeHashBytes(t, "dishonest"))) == 0 {
		t.Fatal("proof must not be bound to the dishonest codeHash")
	}
}
