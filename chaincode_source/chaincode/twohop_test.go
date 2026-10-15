package chaincode

import (
	"encoding/json"
	"math/big"
	"os"
	"strings"
	"testing"
)

type x5bState struct {
	Header     map[string]interface{}   `json:"header"`
	Attester   string                   `json:"attester"`
	CodeHashes map[string]string        `json:"codeHashes"`
	Proofs     map[string]registryProof `json:"proofs"`
}

func loadX5b(t *testing.T) x5bState {
	t.Helper()
	b, err := os.ReadFile("testdata/x5b_state.json")
	if err != nil {
		t.Fatal(err)
	}
	var s x5bState
	if err := json.Unmarshal(b, &s); err != nil {
		t.Fatal(err)
	}
	return s
}

func b1Call(t *testing.T, s x5bState, which string, hdr map[string]interface{}, tx int64) b1Attestation {
	t.Helper()
	ch, _ := unhexBytes(s.CodeHashes[which])
	hj, _ := json.Marshal(hdr)
	pj, _ := json.Marshal(s.Proofs[which])
	return verifyB1Attestation(tx, ch, string(hj), string(pj))
}

func b1Ts(s x5bState) int64 {
	v, _ := new(big.Int).SetString(strings.TrimPrefix(s.Header["timestamp"].(string), "0x"), 16)
	return v.Int64()
}

func TestTwoHop_HonestAttestationProven(t *testing.T) {
	s := loadX5b(t)
	a := b1Call(t, s, "honest", s.Header, b1Ts(s)+30)
	if !a.Verified || !a.Attested {
		t.Fatalf("expected verified and attested, got %+v", a)
	}
}

func TestTwoHop_DishonestNotAttested(t *testing.T) {
	s := loadX5b(t)
	a := b1Call(t, s, "dishonest", s.Header, b1Ts(s)+30)
	if !a.Verified || a.Attested {
		t.Fatalf("dishonest entry must verify as NOT attested, got %+v", a)
	}
}

func TestTwoHop_B0HeaderIsNotAcceptedAsB1(t *testing.T) {
	// a header of the main chain (B0) does not verify against the pinned B1 validators
	s := loadX5b(t)
	var b0 map[string]interface{}
	d, _ := os.ReadFile("testdata/besu_header_0x5fc0c.json")
	_ = json.Unmarshal(d, &b0)
	if a := b1Call(t, s, "honest", b0, 1789435861+30); a.Verified {
		t.Fatal("a B0 header must not be accepted as B1")
	}
}

func TestTwoHop_TamperedAndStaleRejected(t *testing.T) {
	s := loadX5b(t)
	bad := map[string]interface{}{}
	for k, v := range s.Header {
		bad[k] = v
	}
	bad["stateRoot"] = "0x" + strings.Repeat("00", 31) + "01"
	if a := b1Call(t, s, "honest", bad, b1Ts(s)+30); a.Verified {
		t.Fatal("forged stateRoot must be rejected")
	}
	if a := b1Call(t, s, "honest", s.Header, b1Ts(s)+maxHeaderAgeSeconds+10); a.Verified {
		t.Fatal("stale B1 header must be rejected")
	}
}

func TestTwoHop_UnpinnedAttesterRejected(t *testing.T) {
	s := loadX5b(t)
	p := s.Proofs["honest"]
	p.Address = "0x0000000000000000000000000000000000000001"
	s.Proofs["honest"] = p
	if a := b1Call(t, s, "honest", s.Header, b1Ts(s)+30); a.Verified {
		t.Fatal("a contract that is not a pinned attester must be rejected")
	}
}
