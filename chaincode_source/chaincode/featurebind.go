// featurebind.go — P4-minimal feature-binding (companion to circuits_v7/ml_scan_proof_v7.circom and
// FeatureCommitRegistry.sol). Every entry point before this one leaves the feature vector x unbound: a
// prover can certify a vulnerable contract's hash with a clean contract's features (Section eval-attacks,
// attack A2), and this holds through v6 (Table tab:secprops, property P4). v7 makes a Poseidon digest of x
// a public signal (featureHash); this file proves, from the same verified Besu state SubmitDecisionTrustless
// already establishes, that featureHash equals the value ONE pinned scanner authority committed on-chain
// for this exact codeHash. A forged-feature proof still verifies (the circuit only checks internal
// consistency) but its featureHash will not match the honest scanner's commitment, so SemBound is false.
//
// What this does NOT do: it does not prove the scanner authority itself was honest at commit time. It
// narrows the forged-feature attack surface from "any prover, at proof time" to "one pinned key, at scan
// time" (FeatureCommitRegistry.sol); a compromised scanner authority defeats it exactly as a compromised
// policy authority defeats SubmitDecisionCommitted (Section t7-residual).
package chaincode

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
)

const (
	// Redeployed with an independent scanner authority (a freshly generated, non-validator wallet),
	// replacing the original instance whose scannerAuthority was one of the 4 pinned IBFT2 validators.
	featureCommitRegistryAddress = "0x7Cb4419EebEE62Cd823DD119407C424e835537AE"
	slotFeatureHash              = 1 // FeatureCommitRegistry: mapping(bytes32=>bytes32) featureHash
)

// verifyFeatureBinding proves featureHash[codeHash] under stateRoot and checks it equals the v7 proof's
// public featureHash. It never returns an error for evidence that merely fails to bind; it returns
// (false, reason) so the caller can still record a fail-closed REJECT.
func verifyFeatureBinding(stateRoot []byte, codeHash []byte, featureProofJSON string, claimedFeatureHash string) (bool, string) {
	var fp registryProof
	if err := json.Unmarshal([]byte(featureProofJSON), &fp); err != nil {
		return false, fmt.Sprintf("malformed feature-proof JSON: %v", err)
	}
	if !strings.EqualFold(fp.Address, featureCommitRegistryAddress) {
		return false, "proof is for a contract that is not the pinned feature-commit registry"
	}
	v, err := provenMappingAt(stateRoot, fp, featureCommitRegistryAddress, codeHash, slotFeatureHash)
	if err != nil {
		return false, fmt.Sprintf("featureHash not proven: %v", err)
	}
	if v.Sign() == 0 {
		return false, "no scanner commitment exists for this exact codeHash (never honestly scanned, or not yet committed)"
	}
	claimed, ok := new(big.Int).SetString(claimedFeatureHash, 10)
	if !ok {
		return false, "malformed public featureHash"
	}
	if v.Cmp(claimed) != 0 {
		return false, "proof's public featureHash does not match the pinned scanner authority's on-chain commitment for this codeHash (forged-feature attack rejected)"
	}
	return true, "featureHash proven to match the pinned scanner authority's on-chain commitment"
}

// SubmitDecisionTrustlessV7 is SubmitDecisionTrustless plus the P4-minimal feature-binding condition: a
// v7 proof (7 public signals) is required, and its public featureHash must be proven, by Merkle-Patricia
// proof under the same verified header, to equal FeatureCommitRegistry.featureHash[codeHash]. Every other
// SubmitDecisionTrustless condition still applies unchanged.
func (c *AuthContract) SubmitDecisionTrustlessV7(ctx contractapi.TransactionContextInterface,
	decisionID string, sourceDomain string, role string, amount int64, sourceLimit int64,
	codeHash string, proofJSON string, publicSignalsJSON string,
	headerJSON string, registryProofJSON string, featureProofJSON string) (string, error) {

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
		zkDetail = fmt.Sprintf("SubmitDecisionTrustlessV7 requires a v7 proof (7 public signals: adds featureHash), got %d", len(pub))
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

	threshold, known := RoleThreshold[role]
	if !known {
		threshold = 0
	}
	class := classify(semBound, evOK, amount, threshold, known)

	rec := PortableDecisionRecord{
		DecisionID: decisionID, SourceDomain: sourceDomain, Role: role, Amount: amount,
		SourceLimit: sourceLimit, SemBound: semBound, EvAssuranceOK: evOK, TransferClass: class,
		DestThreshold: threshold, ZKVerified: true, ZKVerificationDetail: detail,
		CodeHash: codeHash, LCVerified: src.Verified, LCDetail: src.Detail,
		SourceBlockNumber: src.BlockNumber, SourceBlockHash: src.BlockHash,
		ProvenApprovals: src.Approvals, ProvenMlScanPassed: src.MlPassed,
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
