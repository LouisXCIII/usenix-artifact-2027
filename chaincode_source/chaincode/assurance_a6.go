// assurance_a6.go -- E8 limitation fix: A6 (assurance-claim continuity, the LARGEST FAR
// contributor in the offline ablation, 4.59%) previously had no real on-chain analog -- the base
// deployed classify() checks only the role/amount threshold and the boolean semBound/
// evAssuranceOK flags, not an assurance-TIER claim (classical/hybrid/pq) against a destination
// requirement. This file adds a real "assure_ok" gate, matching e1_classifier.py's _claims()
// exactly (decisionSigTier -> the set of decision-authentication claims it grants), plus an
// ablated twin that skips only that check -- mirroring ablation_a2.go's pattern for A6 instead
// of A2. Additive only: does not modify any existing entry point.
package chaincode

import (
	"encoding/json"
	"fmt"

	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
)

// claimsFor mirrors e1_classifier.py's _claims(decision_sig_tier) exactly:
//   classical -> {ClassicalDecisionAuth}
//   hybrid    -> {ClassicalDecisionAuth, PQDecisionAuth}
//   pq        -> {PQDecisionAuth}
func claimsFor(decisionSigTier string) map[string]bool {
	switch decisionSigTier {
	case "classical":
		return map[string]bool{"ClassicalDecisionAuth": true}
	case "hybrid":
		return map[string]bool{"ClassicalDecisionAuth": true, "PQDecisionAuth": true}
	case "pq":
		return map[string]bool{"PQDecisionAuth": true}
	default:
		return map[string]bool{}
	}
}

// SubmitDecisionWithAssurance is SubmitDecision's role/amount check plus a real assure_ok gate:
// the destination's required decision-authentication claim (destAssuranceRequirement, e.g.
// "PQDecisionAuth") must be among the claims the decision's real signature tier
// (decisionSigTier: "classical"/"hybrid"/"pq") grants. A failure here is non-repairable (matches
// e8_ablation.py: `not assure_ok` is always in `non_repairable_failure`) -- REJECT, not
// CONDITIONAL.
func (c *AuthContract) SubmitDecisionWithAssurance(ctx contractapi.TransactionContextInterface,
	decisionID string, sourceDomain string, role string, amount int64,
	sourceLimit int64, semBound bool, evAssuranceOK bool,
	decisionSigTier string, destAssuranceRequirement string) (string, error) {

	exists, err := c.decisionExists(ctx, decisionID)
	if err != nil {
		return "", err
	}
	if exists {
		return "", fmt.Errorf("decision %s already recorded (replay rejected)", decisionID)
	}
	if !semBound {
		return c.recordAssurance(ctx, decisionID, sourceDomain, role, amount, sourceLimit,
			semBound, evAssuranceOK, decisionSigTier, destAssuranceRequirement, "REJECT", 0)
	}
	if !evAssuranceOK {
		return c.recordAssurance(ctx, decisionID, sourceDomain, role, amount, sourceLimit,
			semBound, evAssuranceOK, decisionSigTier, destAssuranceRequirement, "REJECT", 0)
	}
	destRole := role
	identityMapping := true
	if mapped, needsMapping := RoleMapping[role]; needsMapping {
		destRole = mapped
		identityMapping = false
	}
	threshold, roleKnown := RoleThreshold[destRole]
	if !roleKnown {
		return c.recordAssurance(ctx, decisionID, sourceDomain, role, amount, sourceLimit,
			semBound, evAssuranceOK, decisionSigTier, destAssuranceRequirement, "REEVALUATE", 0)
	}
	if amount > threshold {
		return c.recordAssurance(ctx, decisionID, sourceDomain, role, amount, sourceLimit,
			semBound, evAssuranceOK, decisionSigTier, destAssuranceRequirement, "REJECT", threshold)
	}
	// assure_ok: the destination's required claim must be granted by the real signature tier.
	if !claimsFor(decisionSigTier)[destAssuranceRequirement] {
		return c.recordAssurance(ctx, decisionID, sourceDomain, role, amount, sourceLimit,
			semBound, evAssuranceOK, decisionSigTier, destAssuranceRequirement, "REJECT", threshold)
	}
	class := "SAFE"
	if identityMapping {
		class = "EXACT"
	}
	return c.recordAssurance(ctx, decisionID, sourceDomain, role, amount, sourceLimit,
		semBound, evAssuranceOK, decisionSigTier, destAssuranceRequirement, class, threshold)
}

// SubmitDecisionWithAssuranceNoCheck is SubmitDecisionWithAssurance with the assure_ok claim
// check removed (A6 ablated) -- everything else identical.
func (c *AuthContract) SubmitDecisionWithAssuranceNoCheck(ctx contractapi.TransactionContextInterface,
	decisionID string, sourceDomain string, role string, amount int64,
	sourceLimit int64, semBound bool, evAssuranceOK bool,
	decisionSigTier string, destAssuranceRequirement string) (string, error) {

	exists, err := c.decisionExists(ctx, decisionID)
	if err != nil {
		return "", err
	}
	if exists {
		return "", fmt.Errorf("decision %s already recorded (replay rejected)", decisionID)
	}
	if !semBound {
		return c.recordAssurance(ctx, decisionID, sourceDomain, role, amount, sourceLimit,
			semBound, evAssuranceOK, decisionSigTier, destAssuranceRequirement, "REJECT", 0)
	}
	if !evAssuranceOK {
		return c.recordAssurance(ctx, decisionID, sourceDomain, role, amount, sourceLimit,
			semBound, evAssuranceOK, decisionSigTier, destAssuranceRequirement, "REJECT", 0)
	}
	destRole := role
	identityMapping := true
	if mapped, needsMapping := RoleMapping[role]; needsMapping {
		destRole = mapped
		identityMapping = false
	}
	threshold, roleKnown := RoleThreshold[destRole]
	if !roleKnown {
		return c.recordAssurance(ctx, decisionID, sourceDomain, role, amount, sourceLimit,
			semBound, evAssuranceOK, decisionSigTier, destAssuranceRequirement, "REEVALUATE", 0)
	}
	if amount > threshold {
		return c.recordAssurance(ctx, decisionID, sourceDomain, role, amount, sourceLimit,
			semBound, evAssuranceOK, decisionSigTier, destAssuranceRequirement, "REJECT", threshold)
	}
	// A6 ablated: assure_ok (claimsFor(decisionSigTier)[destAssuranceRequirement]) NOT checked here.
	class := "SAFE"
	if identityMapping {
		class = "EXACT"
	}
	return c.recordAssurance(ctx, decisionID, sourceDomain, role, amount, sourceLimit,
		semBound, evAssuranceOK, decisionSigTier, destAssuranceRequirement, class, threshold)
}

func (c *AuthContract) recordAssurance(ctx contractapi.TransactionContextInterface,
	decisionID, sourceDomain, role string, amount, sourceLimit int64,
	semBound, evAssuranceOK bool, decisionSigTier, destAssuranceRequirement, class string,
	threshold int64) (string, error) {
	rec := PortableDecisionRecord{
		DecisionID:               decisionID,
		SourceDomain:             sourceDomain,
		Role:                     role,
		Amount:                   amount,
		SourceLimit:              sourceLimit,
		SemBound:                 semBound,
		EvAssuranceOK:            evAssuranceOK,
		TransferClass:            class,
		DestThreshold:            threshold,
		DecisionSigTier:          decisionSigTier,
		DestAssuranceRequirement: destAssuranceRequirement,
		AssuranceChecked:         true,
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
