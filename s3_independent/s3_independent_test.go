package chaincode

// S3 harness: runs the UNMODIFIED deployed decision functions offline against a list of cases, with an in-memory
// stub replacing the ledger (only GetState/PutState are used by the code under test).
// env: S3_CASES = input JSON, S3_OUT = output JSON. Skipped when S3_CASES is unset.

import (
	"bytes"
	"encoding/json"
	"math/big"
	"os"
	"testing"

	"github.com/hyperledger/fabric-chaincode-go/v2/shim"
	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
)

type memStub struct {
	shim.ChaincodeStubInterface
	m map[string][]byte
}

func (s *memStub) GetState(k string) ([]byte, error) { return s.m[k], nil }
func (s *memStub) PutState(k string, v []byte) error { s.m[k] = v; return nil }

type s3Case struct {
	ID                     string      `json:"id"`
	DecisionID             string      `json:"decisionId"`
	SourceDomain           string      `json:"sourceDomain"`
	Role                   string      `json:"role"`
	Amount                 json.Number `json:"amount"`
	SourceLimit            json.Number `json:"sourceLimit"`
	SemBound               bool        `json:"semBound"`
	EvAssuranceOK          bool        `json:"evAssuranceOK"`
	SecondApprovalObtained bool        `json:"secondApprovalObtained"`
}

type s3Out struct {
	ID         string `json:"id"`
	Class      string `json:"class"`
	Err        string `json:"err"`
	ClassBasic string `json:"class_basic"`
	ErrBasic   string `json:"err_basic"`
	Skipped    string `json:"skipped,omitempty"`
}

func toInt64(n json.Number) (int64, bool) {
	b, ok := new(big.Int).SetString(n.String(), 10)
	if !ok || !b.IsInt64() {
		return 0, false
	}
	return b.Int64(), true
}

func TestS3Independent(t *testing.T) {
	inPath, outPath := os.Getenv("S3_CASES"), os.Getenv("S3_OUT")
	if inPath == "" {
		t.Skip("S3_CASES not set")
	}
	raw, err := os.ReadFile(inPath)
	if err != nil {
		t.Fatal(err)
	}
	var cases []s3Case
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	if err := dec.Decode(&cases); err != nil {
		t.Fatal(err)
	}
	full := &memStub{m: map[string][]byte{}}
	basic := &memStub{m: map[string][]byte{}}
	ctxFull, ctxBasic := &contractapi.TransactionContext{}, &contractapi.TransactionContext{}
	ctxFull.SetStub(full)
	ctxBasic.SetStub(basic)
	c := &AuthContract{}
	outs := make([]s3Out, 0, len(cases))
	for _, cs := range cases {
		o := s3Out{ID: cs.ID}
		amt, ok1 := toInt64(cs.Amount)
		lim, ok2 := toInt64(cs.SourceLimit)
		if !ok1 || !ok2 {
			o.Skipped = "int64-overflow"
			outs = append(outs, o)
			continue
		}
		cls, err := c.SubmitDecisionFull(ctxFull, cs.DecisionID, cs.SourceDomain, cs.Role, amt, lim, cs.SemBound, cs.EvAssuranceOK, cs.SecondApprovalObtained)
		o.Class = cls
		if err != nil {
			o.Err = err.Error()
		}
		cls2, err2 := c.SubmitDecision(ctxBasic, cs.DecisionID, cs.SourceDomain, cs.Role, amt, lim, cs.SemBound, cs.EvAssuranceOK)
		o.ClassBasic = cls2
		if err2 != nil {
			o.ErrBasic = err2.Error()
		}
		outs = append(outs, o)
	}
	b, _ := json.MarshalIndent(outs, "", " ")
	if err := os.WriteFile(outPath, b, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Logf("ran %d cases", len(outs))
}
