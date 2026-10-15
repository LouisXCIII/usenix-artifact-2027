// ablation_a2_families.go -- extends E8's A2 real on-chain analog (ablation_a2.go, "role" family
// only) to the three other mapping families that already have a real, deployed sem_covered gate:
// action (action.go), resource (resource.go), context (context.go). Each ablated twin is
// identical to its real counterpart except the ONE family-specific sem_covered check is skipped
// (forced to pass) -- the role/amount threshold check, semBound/evAssuranceOK ordering, and
// REEVALUATE-on-unknown-role branch are all left untouched, isolating exactly one conjunct the
// same way e8_ablation.py's classify_ablation() and ablation_a2.go already do for the role family.
// Additive only: does not modify action.go, resource.go, context.go, or any other existing file.
// Reuses recordAction/recordResource/recordContext (defined in those files) unchanged for state
// writes, so no new JSON schema is introduced here.
package chaincode

import (
	"fmt"

	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
)

// SubmitDecisionWithActionNoSemCheck is SubmitDecisionWithAction with the actionLevel<=DestActionCap
// check removed (A2 ablated for the "action" family).
func (c *AuthContract) SubmitDecisionWithActionNoSemCheck(ctx contractapi.TransactionContextInterface,
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
	// A2 ablated: actionLevel<=DestActionCap NOT checked here.
	class := "SAFE"
	if identityMapping {
		class = "EXACT"
	}
	return c.recordAction(ctx, decisionID, sourceDomain, role, amount, sourceLimit,
		semBound, evAssuranceOK, actionLevel, class, threshold)
}

// SubmitDecisionWithResourceNoSemCheck is SubmitDecisionWithResource with the
// resourceID==PinnedResourceID check removed (A2 ablated for the "resource" family).
func (c *AuthContract) SubmitDecisionWithResourceNoSemCheck(ctx contractapi.TransactionContextInterface,
	decisionID string, sourceDomain string, role string, amount int64,
	sourceLimit int64, semBound bool, evAssuranceOK bool, resourceID int64) (string, error) {

	exists, err := c.decisionExists(ctx, decisionID)
	if err != nil {
		return "", err
	}
	if exists {
		return "", fmt.Errorf("decision %s already recorded (replay rejected)", decisionID)
	}
	if !semBound {
		return c.recordResource(ctx, decisionID, sourceDomain, role, amount, sourceLimit,
			semBound, evAssuranceOK, resourceID, "REJECT", 0)
	}
	if !evAssuranceOK {
		return c.recordResource(ctx, decisionID, sourceDomain, role, amount, sourceLimit,
			semBound, evAssuranceOK, resourceID, "REJECT", 0)
	}
	destRole := role
	identityMapping := true
	if mapped, needsMapping := RoleMapping[role]; needsMapping {
		destRole = mapped
		identityMapping = false
	}
	threshold, roleKnown := RoleThreshold[destRole]
	if !roleKnown {
		return c.recordResource(ctx, decisionID, sourceDomain, role, amount, sourceLimit,
			semBound, evAssuranceOK, resourceID, "REEVALUATE", 0)
	}
	if amount > threshold {
		return c.recordResource(ctx, decisionID, sourceDomain, role, amount, sourceLimit,
			semBound, evAssuranceOK, resourceID, "REJECT", threshold)
	}
	// A2 ablated: resourceID==PinnedResourceID NOT checked here.
	class := "SAFE"
	if identityMapping {
		class = "EXACT"
	}
	return c.recordResource(ctx, decisionID, sourceDomain, role, amount, sourceLimit,
		semBound, evAssuranceOK, resourceID, class, threshold)
}

// SubmitDecisionWithContextNoSemCheck is SubmitDecisionWithContext with the contextOK check
// removed (A2 ablated for the "context" family).
func (c *AuthContract) SubmitDecisionWithContextNoSemCheck(ctx contractapi.TransactionContextInterface,
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
	// A2 ablated: contextOK NOT checked here.
	class := "SAFE"
	if identityMapping {
		class = "EXACT"
	}
	return c.recordContext(ctx, decisionID, sourceDomain, role, amount, sourceLimit,
		semBound, evAssuranceOK, contextOK, class, threshold)
}
