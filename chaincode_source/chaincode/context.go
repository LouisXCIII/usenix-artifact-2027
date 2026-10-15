// context.go — E1 limitation (3) fix: a real, on-chain guard for the "context" mapping family
// that the ADP paper's formal benchmark (E1) previously only tested at the formal-model level.
// Additive only: does not modify SubmitDecision/SubmitDecisionFull or any existing entry point.
//
// SemCovered (eq:semantic-coverage) requires every mandatory destination guard to be covered by
// the source predicate under an accepted mapping. context_preservation/context_loss in the
// generated benchmark model exactly this: a boolean context flag that must survive the mapping
// unchanged for the destination guard to be satisfiable. This entry point makes that a real
// on-chain check (role/amount, exactly as SubmitDecisionFull, PLUS a genuine contextOK guard)
// rather than only a Python-side simulation.
package chaincode

import (
	"encoding/json"
	"fmt"

	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
)

// SubmitDecisionWithContext is SubmitDecisionFull's role/amount check plus a real destination
// context guard: contextOK must be true (the mapped context flag matches what the destination
// guard requires) in addition to semBound/evAssuranceOK/role/amount, matching the "context"
// family of eq:five-class-classifier's SemCovered condition. A context failure is NOT treated as
// repairable (matching e1_classifier.py's Satisfiable(Delta) definition, which only admits
// obligation/staleness/finality-deficit as repairable gaps) -- it REJECTs rather than going
// CONDITIONAL, exactly as the offline formal classifier does for a sem_covered=false case.
func (c *AuthContract) SubmitDecisionWithContext(ctx contractapi.TransactionContextInterface,
	decisionID string, sourceDomain string, role string, amount int64,
	sourceLimit int64, semBound bool, evAssuranceOK bool, contextOK bool) (string, error) {

	exists, err := c.decisionExists(ctx, decisionID)
	if err != nil {
		return "", err
	}
	if exists {
		return "", fmt.Errorf("decision %s already recorded (replay rejected)", decisionID)
	}

	if !semBound {
		return c.recordContext(ctx, decisionID, sourceDomain, role, amount, sourceLimit,
			semBound, evAssuranceOK, contextOK, "REJECT", 0)
	}
	if !evAssuranceOK {
		return c.recordContext(ctx, decisionID, sourceDomain, role, amount, sourceLimit,
			semBound, evAssuranceOK, contextOK, "REJECT", 0)
	}

	destRole := role
	identityMapping := true
	if mapped, needsMapping := RoleMapping[role]; needsMapping {
		destRole = mapped
		identityMapping = false
	}

	threshold, roleKnown := RoleThreshold[destRole]
	if !roleKnown {
		return c.recordContext(ctx, decisionID, sourceDomain, role, amount, sourceLimit,
			semBound, evAssuranceOK, contextOK, "REEVALUATE", 0)
	}
	if amount > threshold {
		return c.recordContext(ctx, decisionID, sourceDomain, role, amount, sourceLimit,
			semBound, evAssuranceOK, contextOK, "REJECT", threshold)
	}

	// SemCovered fails if the mapped context flag does not satisfy the destination's context
	// requirement. This is a non-repairable semantic gap, not a Conditional case, so it REJECTs
	// -- matching how the offline formal classifier (e1_classifier.py) treats sem_covered=false.
	if !contextOK {
		return c.recordContext(ctx, decisionID, sourceDomain, role, amount, sourceLimit,
			semBound, evAssuranceOK, contextOK, "REJECT", threshold)
	}

	class := "SAFE"
	if identityMapping {
		class = "EXACT"
	}
	return c.recordContext(ctx, decisionID, sourceDomain, role, amount, sourceLimit,
		semBound, evAssuranceOK, contextOK, class, threshold)
}

func (c *AuthContract) recordContext(ctx contractapi.TransactionContextInterface,
	decisionID, sourceDomain, role string, amount, sourceLimit int64,
	semBound, evAssuranceOK, contextOK bool, class string, threshold int64) (string, error) {
	rec := PortableDecisionRecord{
		DecisionID:     decisionID,
		SourceDomain:   sourceDomain,
		Role:           role,
		Amount:         amount,
		SourceLimit:    sourceLimit,
		SemBound:       semBound,
		EvAssuranceOK:  evAssuranceOK,
		TransferClass:  class,
		DestThreshold:  threshold,
		ContextOK:      contextOK,
		ContextChecked: true,
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
