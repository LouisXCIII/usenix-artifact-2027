package chaincode

import (
	"encoding/json"
	"os"
	"testing"
)

func loadTestFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	return string(b)
}

func TestVerifySemBoundProof_Genuine(t *testing.T) {
	proof := loadTestFile(t, "testdata/proof_A_v5.json")
	public := loadTestFile(t, "testdata/public_A_v5.json")

	ok, detail, err := verifySemBoundProof(proof, public)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatalf("expected genuine proof to verify, got false: %s", detail)
	}
}

func TestVerifySemBoundProof_TamperedProof(t *testing.T) {
	proof := loadTestFile(t, "testdata/proof_A_v5.json")
	public := loadTestFile(t, "testdata/public_A_v5.json")

	var p map[string]interface{}
	if err := json.Unmarshal([]byte(proof), &p); err != nil {
		t.Fatal(err)
	}
	piA := p["pi_a"].([]interface{})
	s := piA[0].(string)
	piA[0] = s[:len(s)-1] + "0"
	if s[len(s)-1] == '0' {
		piA[0] = s[:len(s)-1] + "1"
	}
	tampered, _ := json.Marshal(p)

	ok, detail, err := verifySemBoundProof(string(tampered), public)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatalf("expected tampered proof to be rejected, got valid=true (%s)", detail)
	}
}

func TestVerifySemBoundProof_PolicyMismatch(t *testing.T) {
	proof := loadTestFile(t, "testdata/proof_A_v5.json")
	public := loadTestFile(t, "testdata/public_A_v5.json")

	var pub []string
	if err := json.Unmarshal([]byte(public), &pub); err != nil {
		t.Fatal(err)
	}
	pub[3] = "5" // maxVulnClass tampered from the genuine 0 to 5
	tampered, _ := json.Marshal(pub)

	ok, detail, err := verifySemBoundProof(proof, string(tampered))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatalf("expected policy-mismatched public signal to be rejected, got valid=true (%s)", detail)
	}
}

func TestVerifySemBoundProof_MalformedJSON(t *testing.T) {
	_, _, err := verifySemBoundProof("not json", "also not json")
	if err == nil {
		t.Fatal("expected an error for malformed JSON, got nil")
	}
}
