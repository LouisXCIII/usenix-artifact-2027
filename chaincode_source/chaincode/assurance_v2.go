// assurance_v2.go — real on-chain guard for the S15 (honest classical claim under PQ/hybrid
// transport)/S16 (fake tier upgrade) formal model, previously only tested by a standalone,
// local Python script (e3_assurance_claim_model.py) with no on-chain counterpart at all. Exact
// port of that script's assure_ok_v2(): the destination credits claims from the INDEPENDENTLY
// VERIFIED tier only, never the caller's separately-recorded claimed tier -- an over-claim
// (claimed != verified) is recorded for detection but never influences the security decision,
// exactly mirroring the real E5 result (a tampered/unverifiable PQ signature contributes no
// claim regardless of what tier was claimed for it). Reuses claimsFor() from assurance_a6.go
// completely unchanged (its "default" branch already returns an empty claim set for an
// unrecognized/unverifiable tier, i.e. Python's CLAIMS[None] = set()).
package chaincode

import (
	"encoding/json"
	"fmt"

	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
)

// SubmitDecisionAssuranceV2 is SubmitDecisionWithAssurance's real pipeline, extended with an
// explicit claimed-vs-verified split. assureOK is computed from verifiedTier ONLY (via the
// unchanged claimsFor()); claimedTier is recorded for over-claim detection but never gates the
// decision -- this is the real fix for S15/S16, which the original single-tier model could not
// represent at all (it always trusted the one tier it was given as already-verified).
func (c *AuthContract) SubmitDecisionAssuranceV2(ctx contractapi.TransactionContextInterface,
	decisionID string, sourceDomain string, role string, amount int64, sourceLimit int64,
	semBound bool, evAssuranceOK bool,
	claimedTier string, verifiedTier string, destAssuranceRequirement string) (string, error) {

	exists, err := c.decisionExists(ctx, decisionID)
	if err != nil {
		return "", err
	}
	if exists {
		return "", fmt.Errorf("decision %s already recorded (replay rejected)", decisionID)
	}
	if amount <= 0 {
		return "", fmt.Errorf("amount must be positive (got %d)", amount)
	}

	assureOK := claimsFor(verifiedTier)[destAssuranceRequirement]
	overClaim := claimedTier != verifiedTier

	class := "REJECT"
	var threshold int64
	identityMapping := true
	if !semBound || !evAssuranceOK {
		class = "REJECT"
	} else {
		destRole := role
		if mapped, needsMapping := RoleMapping[role]; needsMapping {
			destRole = mapped
			identityMapping = false
		}
		th, roleKnown := RoleThreshold[destRole]
		if !roleKnown {
			class = "REEVALUATE"
		} else {
			threshold = th
			if amount > threshold {
				class = "REJECT"
			} else if !assureOK {
				class = "REJECT"
			} else {
				class = "SAFE"
				if identityMapping {
					class = "EXACT"
				}
			}
		}
	}

	rec := PortableDecisionRecord{
		DecisionID: decisionID, SourceDomain: sourceDomain, Role: role, Amount: amount,
		SourceLimit: sourceLimit, SemBound: semBound, EvAssuranceOK: evAssuranceOK,
		TransferClass: class, DestThreshold: threshold,
		ClaimedTier: claimedTier, VerifiedTier: verifiedTier,
		OverClaimDetected: overClaim, AssuranceV2Checked: true,
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
