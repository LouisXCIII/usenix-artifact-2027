package chaincode

import (
	"strings"
	"testing"

	"github.com/hyperledger/fabric-chaincode-go/v2/shim"
	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
)

type guardStub struct {
	shim.ChaincodeStubInterface
	written int
}

func (s *guardStub) GetState(key string) ([]byte, error) { return nil, nil }

func (s *guardStub) PutState(key string, value []byte) error {
	s.written++
	return nil
}

type guardCtx struct {
	contractapi.TransactionContextInterface
	stub *guardStub
}

func (c *guardCtx) GetStub() shim.ChaincodeStubInterface { return c.stub }

func TestAmountGuardRejectsNonPositive(t *testing.T) {
	c := &AuthContract{}
	cases := map[string]func(ctx *guardCtx, amount int64) (string, error){
		"SubmitDecision": func(ctx *guardCtx, amount int64) (string, error) {
			return c.SubmitDecision(ctx, "d", "src", "manager", amount, 1000000, true, true)
		},
		"SubmitDecisionFull": func(ctx *guardCtx, amount int64) (string, error) {
			return c.SubmitDecisionFull(ctx, "d", "src", "manager", amount, 1000000, true, true, true)
		},
		"SubmitDecisionObligationV2": func(ctx *guardCtx, amount int64) (string, error) {
			return c.SubmitDecisionObligationV2(ctx, "d", "src", "manager", amount, 1000000, true, true, "p", "m", true, true)
		},
		"SubmitDecisionAssuranceV2": func(ctx *guardCtx, amount int64) (string, error) {
			return c.SubmitDecisionAssuranceV2(ctx, "d", "src", "manager", amount, 1000000, true, true, "classical", "classical", "classical")
		},
		"SubmitDecisionAssuranceV3": func(ctx *guardCtx, amount int64) (string, error) {
			return c.SubmitDecisionAssuranceV3(ctx, "d", "src", "manager", amount, 1000000, "00", "", "", "", "", "", "classical", "classical", "classical")
		},
		"SubmitDecisionTrustlessV7": func(ctx *guardCtx, amount int64) (string, error) {
			return c.SubmitDecisionTrustlessV7(ctx, "d", "src", "manager", amount, 1000000, "00", "", "", "", "", "")
		},
		"SubmitDecisionTrustlessV7Obligation": func(ctx *guardCtx, amount int64) (string, error) {
			return c.SubmitDecisionTrustlessV7Obligation(ctx, "d", "src", "manager", amount, 1000000, "00", "", "", "", "", "", "p", "m", true, true)
		},
	}
	for name, call := range cases {
		for _, amount := range []int64{0, -1, -100000} {
			stub := &guardStub{}
			_, err := call(&guardCtx{stub: stub}, amount)
			if err == nil || !strings.Contains(err.Error(), "amount must be positive") {
				t.Errorf("%s amount=%d: want amount-positive error, got err=%v", name, amount, err)
			}
			if stub.written != 0 {
				t.Errorf("%s amount=%d: wrote state before rejecting", name, amount)
			}
		}
	}
}

func TestICS27BaselineAcceptsWideningByType(t *testing.T) {
	c := &AuthContract{}
	stub := &guardStub{}
	class, err := c.SubmitDecisionICS27(&guardCtx{stub: stub}, "d-ics27", "src", "transfer", 900000000)
	if err != nil || class != "SAFE" {
		t.Fatalf("want SAFE for allowed type regardless of amount, got %q err=%v", class, err)
	}
	stub2 := &guardStub{}
	class, err = c.SubmitDecisionICS27(&guardCtx{stub: stub2}, "d-ics27b", "src", "delete", 1)
	if err != nil || class != "REJECT" {
		t.Fatalf("want REJECT for disallowed type, got %q err=%v", class, err)
	}
}
