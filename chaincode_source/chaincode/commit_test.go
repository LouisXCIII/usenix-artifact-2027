package chaincode

import (
	"encoding/hex"
	"encoding/json"
	"math/big"
	"os"
	"strings"
	"testing"
)

type x3State struct {
	Header        map[string]interface{} `json:"header"`
	CodeHash      string                 `json:"codeHash"`
	RegistryProof registryProof          `json:"registryProof"`
	CommitProofs  map[string]struct {
		HD     string            `json:"hD"`
		Fields map[string]string `json:"fields"`
		Proof  registryProof     `json:"proof"`
	} `json:"commitProofs"`
}

func loadX3(t *testing.T) x3State {
	t.Helper()
	b, err := os.ReadFile("testdata/x3_state.json")
	if err != nil {
		t.Fatal(err)
	}
	var s x3State
	if err := json.Unmarshal(b, &s); err != nil {
		t.Fatal(err)
	}
	return s
}

func i64(t *testing.T, s string) int64 {
	t.Helper()
	v, ok := new(big.Int).SetString(s, 10)
	if !ok {
		t.Fatalf("bad int %q", s)
	}
	return v.Int64()
}

func hashOf(t *testing.T, f map[string]string) []byte {
	t.Helper()
	ch, _ := unhexBytes(f["codeHash"])
	nonce, _ := unhexBytes(f["nonce"])
	return decisionHash(destinationDomain, ch, f["role"], i64(t, f["amount"]), i64(t, f["sourceLimit"]), i64(t, f["epoch"]), i64(t, f["expiry"]), nonce)
}

// The Go hash must equal the one ethers computed with abi.encode at the source.
func TestDecisionHash_MatchesSourceAbiEncode(t *testing.T) {
	var x3 struct {
		Decisions map[string]struct {
			HD     string            `json:"hD"`
			Fields map[string]string `json:"fields"`
		} `json:"decisions"`
	}
	b, _ := os.ReadFile("testdata/x3_result.json")
	if err := json.Unmarshal(b, &x3); err != nil {
		t.Fatal(err)
	}
	d1 := x3.Decisions["D1"]
	if got := "0x" + hex.EncodeToString(hashOf(t, d1.Fields)); got != d1.HD {
		t.Fatalf("Go hD %s != ethers hD %s", got, d1.HD)
	}
	// D_other was committed under a different domain tag; the same fields must hash differently here.
	do := x3.Decisions["Dother"]
	if got := "0x" + hex.EncodeToString(hashOf(t, do.Fields)); got == do.HD {
		t.Fatal("a decision committed for another destination domain must not hash to the same value here")
	}
}

func commitFor(t *testing.T, name string) (bool, string) {
	t.Helper()
	s := loadX3(t)
	root, _ := unhexBytes(s.Header["stateRoot"].(string))
	cp := s.CommitProofs[name]
	return commitStatus(root, cp.Proof, hashOf(t, cp.Fields))
}

func TestCommitStatus_HonestDecisionProven(t *testing.T) {
	if ok, d := commitFor(t, "D1_honest"); !ok {
		t.Fatalf("expected committed, got: %s", d)
	}
}

func TestCommitStatus_AlteredOrUncommittedRejected(t *testing.T) {
	for _, n := range []string{"relay_escalates_role_and_amount", "relay_raises_amount", "relay_changes_role",
		"Dother_presented_to_authcc", "stranger_uncommitted"} {
		if ok, d := commitFor(t, n); ok || !strings.Contains(d, "no source commitment") {
			t.Fatalf("%s: expected 'no source commitment', got ok=%v detail=%s", n, ok, d)
		}
	}
}

func TestCommitStatus_ProofForOtherKeyDoesNotTransfer(t *testing.T) {
	// present D1's commit proof for an altered decision: the derived key differs, so it must not prove it.
	s := loadX3(t)
	root, _ := unhexBytes(s.Header["stateRoot"].(string))
	honest := s.CommitProofs["D1_honest"]
	altered := s.CommitProofs["relay_raises_amount"]
	if ok, _ := commitStatus(root, honest.Proof, hashOf(t, altered.Fields)); ok {
		t.Fatal("a commit proof for one decision must not prove another")
	}
}

func TestCommitStatus_WrongContractRejected(t *testing.T) {
	s := loadX3(t)
	root, _ := unhexBytes(s.Header["stateRoot"].(string))
	cp := s.CommitProofs["D1_honest"]
	cp.Proof.Address = "0x93254ACd9089a3fC9c17A7DD8f4b310A5Ff1ad08"
	if ok, _ := commitStatus(root, cp.Proof, hashOf(t, cp.Fields)); ok {
		t.Fatal("proof labelled for a different contract must be rejected")
	}
}

func TestDecisionHash_EveryFieldMatters(t *testing.T) {
	s := loadX3(t)
	f := s.CommitProofs["D1_honest"].Fields
	base := hex.EncodeToString(hashOf(t, f))
	mut := map[string]string{"role": "manager", "amount": "5001", "sourceLimit": "10001", "epoch": "2", "expiry": "1", "nonce": "0x" + strings.Repeat("11", 32),
		"codeHash": "0x" + strings.Repeat("22", 32)}
	for k, v := range mut {
		g := map[string]string{}
		for kk, vv := range f {
			g[kk] = vv
		}
		g[k] = v
		if hex.EncodeToString(hashOf(t, g)) == base {
			t.Fatalf("changing %s did not change hD", k)
		}
	}
}
