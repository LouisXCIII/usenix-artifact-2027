// v7_obligation.go -- closes the completeness gap flagged by the 2026-09-29 S9 red-team review:
// no existing entry point verifies BOTH the real evidence chain (ZK proof + light-client header,
// as in SubmitDecisionTrustlessV7, featurebind.go) AND the real obligation chain (oblCoveredV2,
// as in SubmitDecisionObligationV2, obligation_v2.go) in the same transaction.
//
// SubmitDecisionTrustlessV7 derives semBound/evAssuranceOK from a real Groth16 proof plus a
// Merkle-Patricia-proven Besu header, but classify() never consults any obligation model at all.
// SubmitDecisionObligationV2 checks oblCoveredV2 faithfully, but takes semBound/evAssuranceOK as
// caller-supplied booleans -- it trusts whatever the caller claims, never re-deriving them from a
// proof. Neither alone is sufficient: V7 can accept a transfer whose obligation was stripped or
// weakened (S5/S6), and ObligationV2 can accept a transfer whose "semBound=true" claim is a lie
// with no real proof behind it.
//
// SubmitDecisionTrustlessV7Obligation is additive: it changes neither existing function, and it
// does not invent any new verification primitive. It re-runs V7's exact proof-chain pipeline to
// compute REAL semBound/evOK, then gates on oblCoveredV2 using those real values instead of a
// caller claim -- a transfer is admissible only if the proof chain AND the obligation chain both
// hold, matching the "evidence/obligation" split already documented as two independent necessary
// conditions in the formal model.
package chaincode

import (
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
)

func (c *AuthContract) SubmitDecisionTrustlessV7Obligation(ctx contractapi.TransactionContextInterface,
	decisionID string, sourceDomain string, role string, amount int64, sourceLimit int64,
	codeHash string, proofJSON string, publicSignalsJSON string,
	headerJSON string, registryProofJSON string, featureProofJSON string,
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
	ch, err := unhexBytes(codeHash)
	if err != nil || len(ch) != 32 {
		return "", fmt.Errorf("codeHash must be 32 bytes of hex")
	}

	// -- real evidence chain, identical to SubmitDecisionTrustlessV7 (featurebind.go) --
	zkOK, zkDetail, err := verifySemBoundProof(proofJSON, publicSignalsJSON)
	if err != nil {
		return "", fmt.Errorf("malformed verification request: %v", err)
	}
	var pub []string
	_ = json.Unmarshal([]byte(publicSignalsJSON), &pub)
	if zkOK && len(pub) != 7 {
		zkOK = false
		zkDetail = fmt.Sprintf("SubmitDecisionTrustlessV7Obligation requires a v7 proof (7 public signals: adds featureHash), got %d", len(pub))
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

	// -- real obligation chain, identical model to SubmitDecisionObligationV2 (obligation_v2.go),
	// but gated on the REAL semBound/evOK just derived above, not on a caller-supplied claim --
	oblCovered, oblDetail := oblCoveredV2(obligationProfile, mappedMechanism, obligationDischarged, enforcementWitnessVerified)

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
		class = "REJECT" // S5/S6/S7: obligation stripped/weakened/laundered -- a genuine proof chain alone is not enough
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
	return class, nil
}
