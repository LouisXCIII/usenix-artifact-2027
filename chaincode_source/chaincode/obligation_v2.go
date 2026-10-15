// obligation_v2.go — real on-chain guard for the S5 (obligation stripping)/S6 (obligation
// weakening)/S7 (obligation laundering) formal model, previously only tested by a standalone,
// local Python script (e3_obligation_model.py) with no on-chain counterpart at all. This is an
// exact, line-for-line port of that script's obl_covered_v2() function -- same ordinal
// mechanism-strength tables, same four real inputs (obligation profile, mapped mechanism,
// discharged flag, enforcement-witness-verified flag), same five real failure/success paths --
// wired into the real destination decision pipeline (role/amount/semBound/evAssuranceOK), the
// same pattern already used for A3's simpler binary obligation gate in SubmitDecisionFull.
package chaincode

import (
	"encoding/json"
	"fmt"

	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
)

// mechanismStrengthV2 / requiredStrengthV2 mirror e3_obligation_model.py's MECHANISM_STRENGTH /
// REQUIRED_STRENGTH exactly. "" stands in for Python's None (mapped_mechanism stripped).
var mechanismStrengthV2 = map[string]int{
	"":              0,
	"notification":  1,
	"audit":         2,
	"dual_approval": 3,
	"combined":      4,
}

var requiredStrengthV2 = map[string]int{
	"empty":         0,
	"notification":  1,
	"audit":         2,
	"dual_approval": 3,
	"combined":      4,
}

// oblCoveredV2 is the exact real port of e3_obligation_model.py's obl_covered_v2().
func oblCoveredV2(obligationProfile, mappedMechanism string, obligationDischarged, enforcementWitnessVerified bool) (bool, string) {
	required, known := requiredStrengthV2[obligationProfile]
	if !known {
		return false, fmt.Sprintf("unknown obligation profile %q", obligationProfile)
	}
	if required == 0 {
		return true, "no source obligation required (empty profile)"
	}
	if mappedMechanism == "" {
		return false, "S5: obligation stripped -- destination maps the required obligation to no mechanism at all"
	}
	mapped, known := mechanismStrengthV2[mappedMechanism]
	if !known {
		return false, fmt.Sprintf("unknown mapped mechanism %q", mappedMechanism)
	}
	if mapped < required {
		return false, "S6: obligation weakened -- destination mechanism strength is below what the source requires"
	}
	if !obligationDischarged {
		return false, "obligation not yet discharged (ordinary Conditional path, not S5/S6/S7)"
	}
	if !enforcementWitnessVerified {
		return false, "S7: obligation laundered -- discharge is claimed but has no independently verifiable enforcement witness"
	}
	return true, "obligation genuinely covered: correct mechanism, sufficient strength, discharged, independently witnessed"
}

// SubmitDecisionObligationV2 is SubmitDecisionFull's real pipeline (semBound/evAssuranceOK/role/
// amount) with the binary obligation gate replaced by the real, richer oblCoveredV2 model above.
func (c *AuthContract) SubmitDecisionObligationV2(ctx contractapi.TransactionContextInterface,
	decisionID string, sourceDomain string, role string, amount int64, sourceLimit int64,
	semBound bool, evAssuranceOK bool,
	obligationProfile string, mappedMechanism string, obligationDischarged bool, enforcementWitnessVerified bool) (string, error) {

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

	oblCovered, oblDetail := oblCoveredV2(obligationProfile, mappedMechanism, obligationDischarged, enforcementWitnessVerified)

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
			} else if !oblCovered {
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
		ObligationProfile: obligationProfile, MappedMechanism: mappedMechanism,
		ObligationCoveredV2: oblCovered, ObligationDetailV2: oblDetail, ObligationV2Checked: true,
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
