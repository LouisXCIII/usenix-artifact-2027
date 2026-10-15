package chaincode

import (
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
)

// SubmitDecisionAssuranceV3 closes the caller-trust gap in SubmitDecisionAssuranceV2, which takes
// semBound and evAssuranceOK as caller booleans. Here both are derived from the real evidence chain
// (ZK proof, light-client header, proven mlScanPassed/approvals, feature binding), exactly as in
// SubmitDecisionTrustlessV7Obligation, and the assurance tier credit still comes from verifiedTier only.
// V2 is kept unchanged because the measured baselines depend on it.
func (c *AuthContract) SubmitDecisionAssuranceV3(ctx contractapi.TransactionContextInterface,
	decisionID string, sourceDomain string, role string, amount int64, sourceLimit int64,
	codeHash string, proofJSON string, publicSignalsJSON string,
	headerJSON string, registryProofJSON string, featureProofJSON string,
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
	ch, err := unhexBytes(codeHash)
	if err != nil || len(ch) != 32 {
		return "", fmt.Errorf("codeHash must be 32 bytes of hex")
	}

	zkOK, zkDetail, err := verifySemBoundProof(proofJSON, publicSignalsJSON)
	if err != nil {
		return "", fmt.Errorf("malformed verification request: %v", err)
	}
	var pub []string
	_ = json.Unmarshal([]byte(publicSignalsJSON), &pub)
	if zkOK && len(pub) != 7 {
		zkOK = false
		zkDetail = fmt.Sprintf("SubmitDecisionAssuranceV3 requires a v7 proof (7 public signals: adds featureHash), got %d", len(pub))
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

	assureOK := claimsFor(verifiedTier)[destAssuranceRequirement]
	overClaim := claimedTier != verifiedTier

	class := "REJECT"
	var threshold int64
	identityMapping := true
	if !semBound || !evOK {
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
		SourceLimit: sourceLimit, SemBound: semBound, EvAssuranceOK: evOK,
		TransferClass: class, DestThreshold: threshold, ZKVerified: true, ZKVerificationDetail: detail,
		CodeHash: codeHash, LCVerified: src.Verified, LCDetail: src.Detail,
		SourceBlockNumber: src.BlockNumber, SourceBlockHash: src.BlockHash,
		ProvenApprovals: src.Approvals, ProvenMlScanPassed: src.MlPassed,
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
