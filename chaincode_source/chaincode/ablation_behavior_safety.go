// ablation_behavior_safety.go -- closes the "behavior-safety beyond policy-safety" gap
// identified 2026-10-08: SubmitDecisionObligationV2 (obligation_v2.go) and
// SubmitDecisionTrustlessV7Obligation (v7_obligation.go) both take obligationDischarged and
// enforcementWitnessVerified as RAW CALLER-SUPPLIED booleans, fed straight into oblCoveredV2 with
// no cross-check against any real on-chain discharge event. The only real discharge mechanism
// that exists, DischargeObligation (authchaincode.go:480), requires the referenced record's own
// TransferClass to be CONDITIONAL and flips that SAME record's ObligationDischarged field to true
// -- but obligation_v2.go/v7_obligation.go never read that field back. A caller can therefore
// claim obligationDischarged=true for a transfer whose obligation was never discharged anywhere
// on-chain, and the destination classifies it SAFE purely on real semBound/evOK/role/amount --
// i.e. policy-safe (every existing check passes) but not behavior-safe (the real-world
// precondition the SAFE label is supposed to certify is unverified). This is the exact Skalka-style
// gap: authorization alone is not sufficient if the state transition it licenses was never shown
// to be safe.
//
// SubmitDecisionTrustlessV7ObligationVerifiedDischarge is additive: it changes neither
// SubmitDecisionTrustlessV7Obligation nor SubmitDecisionObligationV2. It re-runs V7Obligation's
// exact evidence-chain and obligation-model pipeline unchanged, but replaces the caller-supplied
// obligationDischarged boolean with a real GetState lookup of a referenced decision record, and
// requires that record to (a) actually have ObligationDischarged=true (set only by the real
// DischargeObligation state transition) and (b) be bound to the SAME transfer (sourceDomain,
// role, amount) being submitted here -- closing both the "claim was never true" gap and the
// "claim was true for a different, unrelated transfer" confused-deputy variant of it.
// enforcementWitnessVerified remains a caller-supplied boolean: this codebase has no independent
// enforcement-witness submission mechanism to verify it against, so that narrower claim stays an
// explicitly disclosed, unclosed limitation (S7 laundering is unchanged by this fix).
package chaincode

import (
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
)

func (c *AuthContract) SubmitDecisionTrustlessV7ObligationVerifiedDischarge(ctx contractapi.TransactionContextInterface,
	decisionID string, sourceDomain string, role string, amount int64, sourceLimit int64,
	codeHash string, proofJSON string, publicSignalsJSON string,
	headerJSON string, registryProofJSON string, featureProofJSON string,
	obligationProfile string, mappedMechanism string, obligationDischargeRef string, enforcementWitnessVerified bool) (string, error) {

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
	ch, err := unhexBytes(codeHash)
	if err != nil || len(ch) != 32 {
		return "", fmt.Errorf("codeHash must be 32 bytes of hex")
	}

	// -- real evidence chain, identical to SubmitDecisionTrustlessV7Obligation (v7_obligation.go) --
	zkOK, zkDetail, err := verifySemBoundProof(proofJSON, publicSignalsJSON)
	if err != nil {
		return "", fmt.Errorf("malformed verification request: %v", err)
	}
	var pub []string
	_ = json.Unmarshal([]byte(publicSignalsJSON), &pub)
	if zkOK && len(pub) != 7 {
		zkOK = false
		zkDetail = fmt.Sprintf("SubmitDecisionTrustlessV7ObligationVerifiedDischarge requires a v7 proof (7 public signals: adds featureHash), got %d", len(pub))
	}
	if zkOK {
		bound, ok := new(big.Int).SetString(pub[1], 10)
		if !ok || bound.Cmp(new(big.Int).SetBytes(ch)) != 0 {
			zkOK = false
			zkDetail = "proof verifies but its public solidityCodeHash differs from this decision's codeHash (proof substitution rejected)"
		}
	}

	nowSec, err := ledgerNow(ctx)
	if err != nil {
		return "", err
	}
	src := verifySourceState(nowSec, ch, headerJSON, registryProofJSON)

	featureOK, featureDetail := false, "source header not verified, so no feature commitment could be proven"
	if src.Verified {
		if len(pub) == 7 {
			featureOK, featureDetail = verifyFeatureBinding(src.StateRoot, ch, featureProofJSON, pub[6])
		} else {
			featureDetail = "no v7 public featureHash signal present"
		}
	}

	semBound := zkOK && src.Verified && src.MlPassed && featureOK
	evOK := src.Verified && src.Approvals >= minApprovals
	detail := zkDetail
	if zkOK && src.Verified && !src.MlPassed {
		detail = "proof is valid and bound, but the registry's proven mlScanPassed[codeHash] is false"
	} else if zkOK && src.Verified && src.MlPassed {
		detail = featureDetail
	}

	// -- real, on-chain-verified discharge, replacing the caller-supplied obligationDischarged
	// boolean that SubmitDecisionTrustlessV7Obligation and SubmitDecisionObligationV2 both trust
	// unverified --
	obligationDischarged := false
	dischargeDetail := "no obligationDischargeRef supplied (treated as not discharged)"
	if obligationDischargeRef != "" {
		refData, refErr := ctx.GetStub().GetState(obligationDischargeRef)
		if refErr != nil {
			return "", fmt.Errorf("failed to read referenced obligation record: %v", refErr)
		}
		if refData == nil {
			dischargeDetail = fmt.Sprintf("referenced obligation record %q does not exist on-chain", obligationDischargeRef)
		} else {
			var ref PortableDecisionRecord
			if err := json.Unmarshal(refData, &ref); err != nil {
				return "", fmt.Errorf("referenced obligation record %q is malformed: %v", obligationDischargeRef, err)
			}
			usedKey := "discharge-used:" + obligationDischargeRef
			usedData, usedErr := ctx.GetStub().GetState(usedKey)
			if usedErr != nil {
				return "", fmt.Errorf("failed to read discharge-use marker: %v", usedErr)
			}
			switch {
			case !ref.ObligationDischarged:
				dischargeDetail = fmt.Sprintf("referenced record %q exists but was never actually discharged on-chain (ObligationDischarged=false)", obligationDischargeRef)
			case ref.SourceDomain != sourceDomain || ref.Role != role || ref.Amount != amount:
				dischargeDetail = fmt.Sprintf("referenced record %q was discharged, but for a different transfer (sourceDomain/role/amount mismatch) -- cannot cover this decision (confused-deputy rejected)", obligationDischargeRef)
			case usedData != nil:
				// Fix 2026-10-13: this entry point was missing the reuse/double-citation guard
				// that SubmitDecisionTrustlessV7ObligationDomainVerified (q_d_domain_attestation.go)
				// already had since the 2026-10-08 red-team finding -- found live while generating
				// a fresh real on-chain test batch for the independent-verification artifact (a
				// second citation of a genuinely-discharged record returned SAFE instead of REJECT).
				dischargeDetail = fmt.Sprintf("referenced record %q was already consumed by decision %q (one discharge covers one decision; re-citation rejected)", obligationDischargeRef, string(usedData))
			default:
				obligationDischarged = true
				dischargeDetail = fmt.Sprintf("referenced record %q verified on-chain: genuinely discharged for this same sourceDomain/role/amount", obligationDischargeRef)
			}
		}
	}

	oblCovered, oblDetail := oblCoveredV2(obligationProfile, mappedMechanism, obligationDischarged, enforcementWitnessVerified)
	oblDetail = oblDetail + " [discharge: " + dischargeDetail + "]"

	threshold, known := RoleThreshold[role]
	if !known {
		threshold = 0
	}

	var class string
	switch {
	case !semBound:
		class = "REJECT" // Theorem T0: unbound semantic relation, no evidence/obligation level compensates
	case !evOK:
		class = "REJECT"
	case !known:
		class = "REEVALUATE"
	case amount > threshold:
		class = "REJECT" // widening: amount exceeds what the destination permits
	case !oblCovered:
		class = "REJECT" // S5/S6/S7, or an unverified/mismatched discharge claim
	default:
		class = "SAFE"
	}

	rec := PortableDecisionRecord{
		DecisionID: decisionID, SourceDomain: sourceDomain, Role: role, Amount: amount,
		SourceLimit: sourceLimit, SemBound: semBound, EvAssuranceOK: evOK, TransferClass: class,
		DestThreshold: threshold, ZKVerified: true, ZKVerificationDetail: detail,
		CodeHash: codeHash, LCVerified: src.Verified, LCDetail: src.Detail,
		SourceBlockNumber: src.BlockNumber, SourceBlockHash: src.BlockHash,
		ProvenApprovals: src.Approvals, ProvenMlScanPassed: src.MlPassed,
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
	if obligationDischarged && obligationDischargeRef != "" {
		if err := ctx.GetStub().PutState("discharge-used:"+obligationDischargeRef, []byte(decisionID)); err != nil {
			return "", fmt.Errorf("failed to write discharge-use marker: %v", err)
		}
	}
	return class, nil
}
