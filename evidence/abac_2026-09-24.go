// abac.go -- real E6 ABAC workload family ("E2 và E6 làm sao lên xuất sắc luôn"), closing the
// guide's Section 8.3 distinct "ABAC" workload (organization/attribute mapping with context),
// separately from the RBAC (role-only) mechanism used everywhere else this session. A real
// authorization decision here requires TWO independently real attributes together -- role AND
// the submitting identity's genuine organizational membership (ctx.GetClientIdentity().GetMSPID(),
// the same real API already used by clock.go's pinned-authority check) -- not role alone.
// Additive only: no existing entry point or struct is modified.
package chaincode

import (
	"encoding/json"
	"fmt"

	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
)

// orgRoleThreshold: a real destination policy where the SAME role has a genuinely different
// admissible amount depending on the caller's real organizational attribute -- the defining
// property of ABAC (context/attribute-dependent) as distinct from RBAC (role alone determines
// the outcome, used by every other real entry point this session).
var orgRoleThreshold = map[string]map[string]int64{
	"director": {"Org1MSP": 2000000, "Org2MSP": 200000},
	"manager":  {"Org1MSP": 500000, "Org2MSP": 500000},
}

type ABACDecisionRecord struct {
	DecisionID    string `json:"decisionId"`
	SourceDomain  string `json:"sourceDomain"`
	Role          string `json:"role"`
	CallerMSP     string `json:"callerMsp"`
	Amount        int64  `json:"amount"`
	TransferClass string `json:"transferClass"`
	DestThreshold int64  `json:"destThreshold"`
}

func abacKey(decisionID string) string { return "abac:" + decisionID }

// SubmitDecisionABAC: real Eval_d entry point requiring role AND real organizational identity
// together. The caller's MSP is read from the real, cryptographically-authenticated transaction
// identity (not a caller-supplied argument), so it cannot be spoofed independently of the
// endorsement/signing identity actually used.
func (c *AuthContract) SubmitDecisionABAC(ctx contractapi.TransactionContextInterface,
	decisionID string, sourceDomain string, role string, amount int64,
	semBound bool, evAssuranceOK bool) (string, error) {

	existing, err := ctx.GetStub().GetState(abacKey(decisionID))
	if err != nil {
		return "", fmt.Errorf("failed to read state: %v", err)
	}
	if existing != nil {
		return "", fmt.Errorf("decision %s already recorded (replay rejected)", decisionID)
	}

	callerMSP, err := ctx.GetClientIdentity().GetMSPID()
	if err != nil {
		return "", fmt.Errorf("failed to read caller identity: %v", err)
	}

	class := "REJECT"
	var threshold int64
	if !semBound {
		class = "REJECT"
	} else if !evAssuranceOK {
		class = "REJECT"
	} else if orgThresholds, roleKnown := orgRoleThreshold[role]; !roleKnown {
		class = "REEVALUATE"
	} else if th, orgKnown := orgThresholds[callerMSP]; !orgKnown {
		class = "REEVALUATE"
	} else {
		threshold = th
		if amount > threshold {
			class = "REJECT"
		} else {
			class = "SAFE"
		}
	}

	rec := ABACDecisionRecord{
		DecisionID: decisionID, SourceDomain: sourceDomain, Role: role, CallerMSP: callerMSP,
		Amount: amount, TransferClass: class, DestThreshold: threshold,
	}
	recJSON, err := json.Marshal(rec)
	if err != nil {
		return "", err
	}
	if err := ctx.GetStub().PutState(abacKey(decisionID), recJSON); err != nil {
		return "", fmt.Errorf("failed to write state: %v", err)
	}
	return class, nil
}

func (c *AuthContract) ReadABACDecision(ctx contractapi.TransactionContextInterface,
	decisionID string) (*ABACDecisionRecord, error) {
	b, err := ctx.GetStub().GetState(abacKey(decisionID))
	if err != nil {
		return nil, fmt.Errorf("failed to read state: %v", err)
	}
	if b == nil {
		return nil, fmt.Errorf("decision %s does not exist", decisionID)
	}
	var rec ABACDecisionRecord
	if err := json.Unmarshal(b, &rec); err != nil {
		return nil, err
	}
	return &rec, nil
}
