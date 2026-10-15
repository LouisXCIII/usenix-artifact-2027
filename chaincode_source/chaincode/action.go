// action.go — E1 limitation (3) fix, second family: a real, on-chain guard for the "action"
// mapping family (action_restriction/action_expansion in E1's formal benchmark). Additive only:
// does not modify any existing entry point.
//
// Matches e1_classifier.py's action-family SemCovered exactly: destination action levels are
// 0=view, 1=discharge, 2=approve, 3=transfer; the destination cap is 2 (approve). A claimed
// action level outside [0,2] is a real widening (action_expansion) and REJECTs, matching the
// offline formal classifier's treatment of sem_covered=false as non-repairable.
package chaincode

import (
	"encoding/json"
	"fmt"

	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
)

// DestActionCap mirrors e1_classifier.py's DEST_ACTION_CAP (approve=2).
const DestActionCap int64 = 2

func (c *AuthContract) SubmitDecisionWithAction(ctx contractapi.TransactionContextInterface,
	decisionID string, sourceDomain string, role string, amount int64,
	sourceLimit int64, semBound bool, evAssuranceOK bool, actionLevel int64) (string, error) {

	exists, err := c.decisionExists(ctx, decisionID)
	if err != nil {
		return "", err
	}
	if exists {
		return "", fmt.Errorf("decision %s already recorded (replay rejected)", decisionID)
	}

	if !semBound {
		return c.recordAction(ctx, decisionID, sourceDomain, role, amount, sourceLimit,
			semBound, evAssuranceOK, actionLevel, "REJECT", 0)
	}
	if !evAssuranceOK {
		return c.recordAction(ctx, decisionID, sourceDomain, role, amount, sourceLimit,
			semBound, evAssuranceOK, actionLevel, "REJECT", 0)
	}

	destRole := role
	identityMapping := true
	if mapped, needsMapping := RoleMapping[role]; needsMapping {
		destRole = mapped
		identityMapping = false
	}

	threshold, roleKnown := RoleThreshold[destRole]
	if !roleKnown {
		return c.recordAction(ctx, decisionID, sourceDomain, role, amount, sourceLimit,
			semBound, evAssuranceOK, actionLevel, "REEVALUATE", 0)
	}
	if amount > threshold {
		return c.recordAction(ctx, decisionID, sourceDomain, role, amount, sourceLimit,
			semBound, evAssuranceOK, actionLevel, "REJECT", threshold)
	}

	// SemCovered for the action family: 0 <= actionLevel <= DestActionCap, exactly matching
	// e1_classifier.py. A claimed level outside this range is a non-repairable widening -- REJECT.
	if actionLevel < 0 || actionLevel > DestActionCap {
		return c.recordAction(ctx, decisionID, sourceDomain, role, amount, sourceLimit,
			semBound, evAssuranceOK, actionLevel, "REJECT", threshold)
	}

	class := "SAFE"
	if identityMapping {
		class = "EXACT"
	}
	return c.recordAction(ctx, decisionID, sourceDomain, role, amount, sourceLimit,
		semBound, evAssuranceOK, actionLevel, class, threshold)
}

func (c *AuthContract) recordAction(ctx contractapi.TransactionContextInterface,
	decisionID, sourceDomain, role string, amount, sourceLimit int64,
	semBound, evAssuranceOK bool, actionLevel int64, class string, threshold int64) (string, error) {
	rec := PortableDecisionRecord{
		DecisionID:    decisionID,
		SourceDomain:  sourceDomain,
		Role:          role,
		Amount:        amount,
		SourceLimit:   sourceLimit,
		SemBound:      semBound,
		EvAssuranceOK: evAssuranceOK,
		TransferClass: class,
		DestThreshold: threshold,
		ActionLevel:   actionLevel,
		ActionChecked: true,
	}
	recJSON, err := json.Marshal(rec)
	if err != nil {
		return "", err
	}
	if err := ctx.GetStub().PutState(decisionID, recJSON); err != nil {
		return "", fmt.Errorf("failed to write state: %v", err)
	}
	return class, nil
}
