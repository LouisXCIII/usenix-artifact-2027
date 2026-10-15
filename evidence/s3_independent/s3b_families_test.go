package chaincode

// S3b harness: runs the UNMODIFIED deployed decision functions of the six other decision families offline with an
// in-memory stub replacing the ledger. env: S3B_CASES = input JSON, S3B_OUT = output JSON. Skipped when S3B_CASES is unset.

import (
	"bytes"
	"encoding/json"
	"math/big"
	"os"
	"testing"

	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
)

type s3bCase struct {
	ID            string                     `json:"id"`
	Target        string                     `json:"target"`
	DecisionID    string                     `json:"decisionId"`
	SourceDomain  string                     `json:"sourceDomain"`
	Role          string                     `json:"role"`
	Amount        json.Number                `json:"amount"`
	SourceLimit   json.Number                `json:"sourceLimit"`
	SemBound      bool                       `json:"semBound"`
	EvAssuranceOK bool                       `json:"evAssuranceOK"`
	Args          map[string]json.RawMessage `json:"args"`
}

type s3bOut struct {
	ID      string `json:"id"`
	Class   string `json:"class"`
	Err     string `json:"err"`
	Skipped string `json:"skipped,omitempty"`
}

func s3bInt(n json.Number) (int64, bool) {
	b, ok := new(big.Int).SetString(n.String(), 10)
	if !ok || !b.IsInt64() {
		return 0, false
	}
	return b.Int64(), true
}

func argInt(a map[string]json.RawMessage, k string) (int64, bool) {
	raw, ok := a[k]
	if !ok {
		return 0, false
	}
	return s3bInt(json.Number(string(raw)))
}

func argBool(a map[string]json.RawMessage, k string) (bool, bool) {
	raw, ok := a[k]
	if !ok {
		return false, false
	}
	var v bool
	if err := json.Unmarshal(raw, &v); err != nil {
		return false, false
	}
	return v, true
}

func argStr(a map[string]json.RawMessage, k string) (string, bool) {
	raw, ok := a[k]
	if !ok {
		return "", false
	}
	var v string
	if err := json.Unmarshal(raw, &v); err != nil {
		return "", false
	}
	return v, true
}

func TestS3bFamilies(t *testing.T) {
	inPath, outPath := os.Getenv("S3B_CASES"), os.Getenv("S3B_OUT")
	if inPath == "" {
		t.Skip("S3B_CASES not set")
	}
	raw, err := os.ReadFile(inPath)
	if err != nil {
		t.Fatal(err)
	}
	var cases []s3bCase
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	if err := dec.Decode(&cases); err != nil {
		t.Fatal(err)
	}
	c := &AuthContract{}
	outs := make([]s3bOut, 0, len(cases))
	for _, cs := range cases {
		o := s3bOut{ID: cs.ID}
		amt, ok1 := s3bInt(cs.Amount)
		lim, ok2 := s3bInt(cs.SourceLimit)
		if !ok1 || !ok2 {
			o.Skipped = "int64-overflow"
			outs = append(outs, o)
			continue
		}
		// fresh in-memory ledger per case: no replay interaction between cases
		ctx := &contractapi.TransactionContext{}
		ctx.SetStub(&memStub{m: map[string][]byte{}})
		var cls string
		var e error
		bad := ""
		switch cs.Target {
		case "action":
			lvl, ok := argInt(cs.Args, "actionLevel")
			if !ok {
				bad = "args.actionLevel"
			} else {
				cls, e = c.SubmitDecisionWithAction(ctx, cs.DecisionID, cs.SourceDomain, cs.Role, amt, lim, cs.SemBound, cs.EvAssuranceOK, lvl)
			}
		case "resource":
			id, ok := argInt(cs.Args, "resourceID")
			if !ok {
				bad = "args.resourceID"
			} else {
				cls, e = c.SubmitDecisionWithResource(ctx, cs.DecisionID, cs.SourceDomain, cs.Role, amt, lim, cs.SemBound, cs.EvAssuranceOK, id)
			}
		case "context":
			v, ok := argBool(cs.Args, "contextOK")
			if !ok {
				bad = "args.contextOK"
			} else {
				cls, e = c.SubmitDecisionWithContext(ctx, cs.DecisionID, cs.SourceDomain, cs.Role, amt, lim, cs.SemBound, cs.EvAssuranceOK, v)
			}
		case "obligation_v2":
			p, ok1 := argStr(cs.Args, "obligationProfile")
			m, ok2 := argStr(cs.Args, "mappedMechanism")
			d, ok3 := argBool(cs.Args, "obligationDischarged")
			w, ok4 := argBool(cs.Args, "enforcementWitnessVerified")
			if !(ok1 && ok2 && ok3 && ok4) {
				bad = "args(obligation_v2)"
			} else {
				cls, e = c.SubmitDecisionObligationV2(ctx, cs.DecisionID, cs.SourceDomain, cs.Role, amt, lim, cs.SemBound, cs.EvAssuranceOK, p, m, d, w)
			}
		case "assurance_v2":
			cl, ok1 := argStr(cs.Args, "claimedTier")
			ve, ok2 := argStr(cs.Args, "verifiedTier")
			rq, ok3 := argStr(cs.Args, "destAssuranceRequirement")
			if !(ok1 && ok2 && ok3) {
				bad = "args(assurance_v2)"
			} else {
				cls, e = c.SubmitDecisionAssuranceV2(ctx, cs.DecisionID, cs.SourceDomain, cs.Role, amt, lim, cs.SemBound, cs.EvAssuranceOK, cl, ve, rq)
			}
		case "assurance_a6":
			st, ok1 := argStr(cs.Args, "decisionSigTier")
			rq, ok2 := argStr(cs.Args, "destAssuranceRequirement")
			if !(ok1 && ok2) {
				bad = "args(assurance_a6)"
			} else {
				cls, e = c.SubmitDecisionWithAssurance(ctx, cs.DecisionID, cs.SourceDomain, cs.Role, amt, lim, cs.SemBound, cs.EvAssuranceOK, st, rq)
			}
		default:
			bad = "unknown target " + cs.Target
		}
		if bad != "" {
			o.Skipped = "bad-args: " + bad
		}
		o.Class = cls
		if e != nil {
			o.Err = e.Error()
		}
		outs = append(outs, o)
	}
	b, _ := json.MarshalIndent(outs, "", " ")
	if err := os.WriteFile(outPath, b, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Logf("ran %d cases", len(outs))
}
