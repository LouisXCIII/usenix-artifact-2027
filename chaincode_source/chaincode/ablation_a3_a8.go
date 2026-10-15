// ablation_a3_a8.go -- E8 limitation fix: real on-chain analogs for A3 (obligation preservation)
// and A8 (replay protection). Additive only: does not modify any existing entry point.
//
// A3: SubmitDecisionFull (authchaincode.go) already has a REAL obligation gate --
// requiresObligation := amount > threshold/2; if requiresObligation && !secondApprovalObtained,
// the result is CONDITIONAL, not SAFE/EXACT. This is exactly the formal model's obl_covered
// conjunct in real form. SubmitDecisionFullNoObligationCheck below is identical except that gate
// is skipped, so a high-value decision with no second approval is (incorrectly) immediately
// accepted -- the real consequence of A3.
//
// A8: SubmitDecision (authchaincode.go) already has a REAL replay gate (decisionExists), already
// exercised at N=100 real depth (E4, evidence/e4_replay_toctou_limitations_2026-09-23.md).
// SubmitDecisionNoReplayCheck below is identical except that gate is skipped, so a second
// submission under the SAME decisionID silently overwrites the first record instead of being
// rejected -- the real consequence of A8.
package chaincode

import (
	"encoding/json"
	"fmt"

	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
)

// SubmitDecisionFullNoObligationCheck is SubmitDecisionFull with the
// requiresObligation/secondApprovalObtained gate removed (A3 ablated): a decision that should be
// held CONDITIONAL pending a second approval is instead accepted immediately.
func (c *AuthContract) SubmitDecisionFullNoObligationCheck(ctx contractapi.TransactionContextInterface,
	decisionID string, sourceDomain string, role string, amount int64,
	sourceLimit int64, semBound bool, evAssuranceOK bool,
	secondApprovalObtained bool) (string, error) {

	exists, err := c.decisionExists(ctx, decisionID)
	if err != nil {
		return "", err
	}
	if exists {
		return "", fmt.Errorf("decision %s already recorded (replay rejected)", decisionID)
	}
	if !semBound {
		return c.recordFull(ctx, decisionID, sourceDomain, role, amount, sourceLimit,
			semBound, evAssuranceOK, "REJECT", 0, false, false)
	}
	if !evAssuranceOK {
		return c.recordFull(ctx, decisionID, sourceDomain, role, amount, sourceLimit,
			semBound, evAssuranceOK, "REJECT", 0, false, false)
	}
	destRole := role
	identityMapping := true
	if mapped, needsMapping := RoleMapping[role]; needsMapping {
		destRole = mapped
		identityMapping = false
	}
	threshold, roleKnown := RoleThreshold[destRole]
	if !roleKnown {
		return c.recordFull(ctx, decisionID, sourceDomain, role, amount, sourceLimit,
			semBound, evAssuranceOK, "REEVALUATE", 0, false, false)
	}
	if amount > threshold {
		return c.recordFull(ctx, decisionID, sourceDomain, role, amount, sourceLimit,
			semBound, evAssuranceOK, "REJECT", threshold, false, false)
	}
	// A3 ablated: requiresObligation/secondApprovalObtained NOT checked here -- always proceeds.
	requiresObligation := amount > threshold/2
	class := "SAFE"
	if identityMapping {
		class = "EXACT"
	}
	return c.recordFull(ctx, decisionID, sourceDomain, role, amount, sourceLimit,
		semBound, evAssuranceOK, class, threshold, requiresObligation, requiresObligation)
}

// SubmitDecisionNoReplayCheck is SubmitDecision with the decisionExists replay gate removed (A8
// ablated): a second submission under the same decisionID silently overwrites the first record.
func (c *AuthContract) SubmitDecisionNoReplayCheck(ctx contractapi.TransactionContextInterface,
	decisionID string, sourceDomain string, role string, amount int64,
	sourceLimit int64, semBound bool, evAssuranceOK bool) (string, error) {

	// A8 ablated: decisionExists NOT checked here -- no replay guard at all.
	threshold, known := RoleThreshold[role]
	if !known {
		threshold = 0
	}
	class := classify(semBound, evAssuranceOK, amount, threshold, known)

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
