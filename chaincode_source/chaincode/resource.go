// resource.go — E1 limitation (3) fix, third family: a real, on-chain guard for the "resource"
// mapping family (resource_substitution in E1's formal benchmark). Additive only: does not
// modify any existing entry point.
//
// Matches e1_classifier.py's resource-family model: the source always asserts a fixed resource
// id (2 in the formal benchmark), and the destination's accepted-equivalence table determines
// whether a mapped/substituted resource id is admissible. This chaincode pins the single
// identity-equivalence entry (2 -> 2), the "equivalent"/"source_stricter" case from the formal
// benchmark, as the real destination policy: any other resource id is a real resource
// substitution and REJECTs, matching sem_covered=false being non-repairable.
package chaincode

import (
	"encoding/json"
	"fmt"

	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
)

// PinnedResourceID mirrors the formal benchmark's source-asserted resource id and this
// destination's sole accepted equivalence entry (identity mapping 2 -> 2).
const PinnedResourceID int64 = 2

func (c *AuthContract) SubmitDecisionWithResource(ctx contractapi.TransactionContextInterface,
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

	// SemCovered for the resource family: the claimed resource id must be in this destination's
	// accepted equivalence set (here, the single pinned identity entry). A substituted resource
	// id is a non-repairable semantic gap -- REJECT.
	if resourceID != PinnedResourceID {
		return c.recordResource(ctx, decisionID, sourceDomain, role, amount, sourceLimit,
			semBound, evAssuranceOK, resourceID, "REJECT", threshold)
	}

	class := "SAFE"
	if identityMapping {
		class = "EXACT"
	}
	return c.recordResource(ctx, decisionID, sourceDomain, role, amount, sourceLimit,
		semBound, evAssuranceOK, resourceID, class, threshold)
}

func (c *AuthContract) recordResource(ctx contractapi.TransactionContextInterface,
	decisionID, sourceDomain, role string, amount, sourceLimit int64,
	semBound, evAssuranceOK bool, resourceID int64, class string, threshold int64) (string, error) {
	rec := PortableDecisionRecord{
		DecisionID:      decisionID,
		SourceDomain:    sourceDomain,
		Role:            role,
		Amount:          amount,
		SourceLimit:     sourceLimit,
		SemBound:        semBound,
		EvAssuranceOK:   evAssuranceOK,
		TransferClass:   class,
		DestThreshold:   threshold,
		ResourceID:      resourceID,
		ResourceChecked: true,
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
