// risk_tier.go -- Tru 4 proper: a real, genuinely non-trivial R_P instance, correcting the
// 2026-10-11 same-day finding (evidence/rp_checker_zk_2026-10-11.md) that the SemBound/
// feature-binding fragment (zkverify.go/featurebind.go) is NOT one -- its psi_s and gamma_d are
// stated over the identical public-signal vocabulary the circuit itself outputs, so M_I is the
// identity map and gamma_d===psi_s by construction (a tautology-under-identity, same category
// as the obligation mechanism-strength fragment). What that mechanism tests is AUTHENTICITY,
// not a semantic relation between two independently-chosen predicate languages.
//
// This file is different by construction. The v6/v7 circuit relation does not output the
// achieved margin as a continuous public signal -- minMargin is a PARAMETER supplied at proof
// generation time, and witness generation genuinely fails if the real margin falls short
// (x16_risk_tier_proofs.py, real, measured). A destination that requests real proofs at several
// different floors for the same contract gets a genuine, real, SOURCE-side risk TIER (0-3),
// expressed purely on the circuit's own margin scale -- entirely distinct from this
// destination's own dollar scale (RoleThreshold). M_I (tierFractionBps, below) is a real
// governance table translating one into the other: a genuine cross-vocabulary mapping a
// governance body could transcribe wrong, unlike an identity pass-through. gamma_d (admit iff
// amount <= cap(role) * fraction(tier)) is NOT definitionally equal to psi_s (achieved tier
// alone says nothing about role or amount) -- see rp_checker_tier.py for the Z3 formalization
// and 4/4 genuine cross-vocabulary naive-mapping variants found widening.
package chaincode

import (
	"encoding/json"
	"fmt"
	"math/big"

	secpecdsa "github.com/decred/dcrd/dcrec/secp256k1/v4/ecdsa"
	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
)

// Real floors used by x16_risk_tier_proofs.py (quantized logits, SCALE2=4096*4096): T1 is the
// unchanged real deployed floor (requiredMinMargin in zkverify.go); T2/T3 are new, destination-
// chosen governance thresholds for this experiment, not tuned to any specific corpus contract.
const (
	tierT1Floor = 54445422 // ceil(3.2452 * SCALE2), same floor as requiredMinMargin
	tierT2Floor = 67108864 // 4.0 * SCALE2
	tierT3Floor = 83886080 // 5.0 * SCALE2
)

// tierFractionBps is M_I: achieved risk tier -> fraction of the destination role's real cap
// (RoleThreshold) allowed, in basis points. A real governance table, independent of both the
// source's margin vocabulary and the destination's own role/amount vocabulary.
var tierFractionBps = map[int64]int64{0: 0, 1: 1000, 2: 5000, 3: 10000} // 0%, 10%, 50%, 100%

// tierFromFloorSignal derives the achieved tier from the submitted proof's own declared
// minMargin floor (public[4]) -- the highest band that floor falls at or above. The pairing
// check (verifyGroth16) already guarantees the real achieved margin was >= that declared floor;
// this function never sees the achieved margin itself, only which floor the prover chose to
// prove against.
func tierFromFloorSignal(floorStr string) (int64, error) {
	f, ok := new(big.Int).SetString(floorStr, 10)
	if !ok {
		return 0, fmt.Errorf("malformed minMargin public signal %q", floorStr)
	}
	switch {
	case f.Cmp(big.NewInt(tierT3Floor)) >= 0:
		return 3, nil
	case f.Cmp(big.NewInt(tierT2Floor)) >= 0:
		return 2, nil
	case f.Cmp(big.NewInt(tierT1Floor)) >= 0:
		return 1, nil
	default:
		return 0, nil
	}
}

// verifyRiskTierProof re-verifies the Groth16 pairing (reusing verifyGroth16/groth16Proof/
// groth16VKey from zkverify.go, same embedded v6 verification key) and the destination's own
// policy on the non-margin signals (scanPassed, maxVulnClass, modelWeightsHash -- same checks
// and same constants as verifySemBoundProof), then derives the achieved tier from the proof's
// own declared floor rather than requiring one fixed floor.
func verifyRiskTierProof(proofJSON, publicJSON string) (bool, int64, string, error) {
	var pf groth16Proof
	if err := json.Unmarshal([]byte(proofJSON), &pf); err != nil {
		return false, 0, "", fmt.Errorf("malformed proof JSON: %v", err)
	}
	var public []string
	if err := json.Unmarshal([]byte(publicJSON), &public); err != nil {
		return false, 0, "", fmt.Errorf("malformed public-signals JSON: %v", err)
	}
	if len(public) != 6 {
		return false, 0, fmt.Sprintf("expected 6 public signals (v6 relation), got %d", len(public)), nil
	}
	var vk groth16VKey
	if err := json.Unmarshal(verificationKeyV6JSON, &vk); err != nil {
		return false, 0, "", fmt.Errorf("embedded v6 verification key is corrupt: %v", err)
	}
	pairingOK, err := verifyGroth16(vk, pf, public)
	if err != nil {
		return false, 0, fmt.Sprintf("pairing check error: %v", err), nil
	}
	if !pairingOK {
		return false, 0, "Groth16 pairing check failed: proof does not verify against the embedded v6 verification key", nil
	}
	if public[0] != "1" {
		return false, 0, "proof verifies, but its own scanPassed output signal is 0", nil
	}
	if public[3] != requiredMaxVulnClass {
		return false, 0, fmt.Sprintf("proof verifies, but was produced for maxVulnClass=%s; destination policy requires exactly %s", public[3], requiredMaxVulnClass), nil
	}
	if public[5] != pinnedModelWeightsHash {
		return false, 0, "proof verifies, but its modelWeightsHash is not the pinned model", nil
	}
	tier, terr := tierFromFloorSignal(public[4])
	if terr != nil {
		return false, 0, terr.Error(), nil
	}
	return true, tier, fmt.Sprintf("Groth16 v6 proof verified; achieved risk tier %d from declared floor %s", tier, public[4]), nil
}

// SubmitDecisionRiskTierVerified is the real destination entry point for the genuine two-
// vocabulary R_P instance. amount is admitted iff amount <= RoleThreshold[role] *
// tierFractionBps[achievedTier] / 10000 (M_I applied), tier 0 (no valid proof at any real
// floor) is always REJECT regardless of role/amount.
func (c *AuthContract) SubmitDecisionRiskTierVerified(ctx contractapi.TransactionContextInterface,
	decisionID string, role string, amount int64, proofJSON string, publicSignalsJSON string) (string, error) {

	exists, err := c.decisionExists(ctx, decisionID)
	if err != nil {
		return "", err
	}
	if exists {
		return "", fmt.Errorf("decision %s already recorded (replay rejected)", decisionID)
	}

	proofOK, tier, detail, err := verifyRiskTierProof(proofJSON, publicSignalsJSON)
	if err != nil {
		return "", fmt.Errorf("malformed verification request: %v", err)
	}

	var class string
	fraction := int64(0)
	if !proofOK || tier == 0 {
		class = "REJECT"
		if proofOK && tier == 0 {
			detail = detail + " -- tier 0 is always REJECT regardless of role/amount"
		}
	} else {
		cap, known := RoleThreshold[role]
		if !known {
			class = "REEVALUATE"
			detail = detail + fmt.Sprintf("; unknown role %q", role)
		} else {
			fraction = tierFractionBps[tier]
			adjustedCap := cap * fraction / 10000
			if amount <= adjustedCap {
				class = "SAFE"
			} else {
				class = "REJECT"
			}
			detail = detail + fmt.Sprintf("; role=%s real cap=%d, tier fraction=%d bps, adjusted cap=%d, amount=%d", role, cap, fraction, adjustedCap, amount)
		}
	}

	rec := PortableDecisionRecord{
		DecisionID: decisionID, Role: role, Amount: amount, TransferClass: class,
		RiskTierAchieved: tier, RiskTierFractionBps: fraction, RiskTierDetail: detail, RiskTierChecked: true,
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

// riskTierAwsAuthorityAddress is the pinned address of a secp256k1 key generated by
// `risktier_aws keygen` on the AWS domain (2026-10-11), distinct from
// awsDomainAuthorityAddress (q_d_domain_attestation.go, a separate attestation scheme/key) to
// keep the two experiments' digests cleanly unrelated. The private key never leaves that AWS
// account.
const riskTierAwsAuthorityAddress = "0x731c7cef6ba0b375f6ae6df77f29058d73d6bd97"

// word8 is an 8-byte big-endian encoding, deliberately local to this file and NOT the
// package-level word() (commit.go, 32 bytes) -- a real bug caught during this experiment's
// first on-chain run: calling word() here silently resolved to the 32-byte version (same
// package, same name), so the digest computed here did not match what risktier_aws actually
// signs, and a genuine AWS signature recovered to the wrong address. Fixed by giving this
// file its own distinctly-named 8-byte helper, matching risktier_aws's own encoding exactly
// (same pattern q_d_domain_attestation.go already uses locally, for the same reason).
func word8(v int64) []byte {
	b := make([]byte, 8)
	for i := 0; i < 8; i++ {
		b[7-i] = byte(v >> (8 * i))
	}
	return b
}

// riskTierAwsDigest reproduces exactly the digest risktier_aws's verify-sign subcommand signs:
// keccak256(decisionID || role || amount(8B BE) || achievedTier(8B BE) || adjustedCap(8B BE) ||
// passedByte). Both sides must agree on this byte layout or recovery fails even for a genuine
// signature.
func riskTierAwsDigest(decisionID, role string, amount, achievedTier, adjustedCap int64, passed bool) []byte {
	passedByte := byte(0x00)
	if passed {
		passedByte = 0x01
	}
	return keccak256([]byte(decisionID), []byte(role), word8(amount), word8(achievedTier), word8(adjustedCap), []byte{passedByte})
}

// verifyRiskTierAwsAttestation recovers the signer of a 65-byte compact signature (same layout
// as lightclient.go's commit-seal recovery and q_d_domain_attestation.go's AWS attestation)
// over digest, and reports whether it matches the pinned risk-tier domain authority.
func verifyRiskTierAwsAttestation(compactSigHex string, digest []byte) (bool, string, error) {
	sig, err := unhexBytes(compactSigHex)
	if err != nil || len(sig) != 65 {
		return false, "", fmt.Errorf("risk-tier AWS attestation signature must be 65 bytes of hex, got %d bytes", len(sig))
	}
	pub, _, err := secpecdsa.RecoverCompact(sig, digest)
	if err != nil {
		return false, "", fmt.Errorf("signature recovery failed: %v", err)
	}
	addr := keccak256(pub.SerializeUncompressed()[1:])[12:]
	want, werr := unhexBytes(riskTierAwsAuthorityAddress)
	if werr != nil {
		return false, "", werr
	}
	recovered := fmt.Sprintf("0x%x", addr)
	if len(addr) != len(want) {
		return false, recovered, nil
	}
	for i := range addr {
		if addr[i] != want[i] {
			return false, recovered, nil
		}
	}
	return true, recovered, nil
}

// SubmitDecisionRiskTierDomainVerified is SubmitDecisionRiskTierVerified plus a second,
// independently-governed administrative domain (AWS) that must independently re-derive the
// SAME tier->fraction mapping and co-sign agreement before SAFE is possible -- genuine cross-
// domain governance over M_I itself, not just over obligation-discharge coverage
// (q_d_domain_attestation.go's existing scope). Both domains must agree; either one's absence,
// mismatch, or disagreement is REJECT.
func (c *AuthContract) SubmitDecisionRiskTierDomainVerified(ctx contractapi.TransactionContextInterface,
	decisionID string, role string, amount int64, proofJSON string, publicSignalsJSON string,
	awsPassed bool, awsAdjustedCap int64, awsCompactSigHex string) (string, error) {

	exists, err := c.decisionExists(ctx, decisionID)
	if err != nil {
		return "", err
	}
	if exists {
		return "", fmt.Errorf("decision %s already recorded (replay rejected)", decisionID)
	}

	proofOK, tier, detail, err := verifyRiskTierProof(proofJSON, publicSignalsJSON)
	if err != nil {
		return "", fmt.Errorf("malformed verification request: %v", err)
	}

	fabricPassed := false
	fraction := int64(0)
	fabricAdjustedCap := int64(0)
	if proofOK && tier >= 1 {
		if cap, known := RoleThreshold[role]; known {
			fraction = tierFractionBps[tier]
			fabricAdjustedCap = cap * fraction / 10000
			fabricPassed = amount <= fabricAdjustedCap
		}
	}

	digest := riskTierAwsDigest(decisionID, role, amount, tier, awsAdjustedCap, awsPassed)
	awsOK, awsSigner, awsErr := verifyRiskTierAwsAttestation(awsCompactSigHex, digest)
	awsDetail := ""
	switch {
	case awsErr != nil:
		awsOK = false
		awsDetail = fmt.Sprintf("aws domain: signature error: %v", awsErr)
	case !awsOK:
		awsDetail = fmt.Sprintf("aws domain: signature recovered to %s, not the pinned risk-tier domain authority %s", awsSigner, riskTierAwsAuthorityAddress)
	case awsAdjustedCap != fabricAdjustedCap:
		awsDetail = fmt.Sprintf("aws domain: genuine signature, but AWS's own adjustedCap=%d does not match Fabric's own adjustedCap=%d (governance table divergence)", awsAdjustedCap, fabricAdjustedCap)
		awsOK = false
	default:
		awsDetail = fmt.Sprintf("aws domain: genuine signature from %s, independently re-derived adjustedCap=%d, passed=%v", awsSigner, awsAdjustedCap, awsPassed)
	}

	class := "REJECT"
	if fabricPassed && awsOK && awsPassed {
		class = "SAFE"
	}

	rec := PortableDecisionRecord{
		DecisionID: decisionID, Role: role, Amount: amount, TransferClass: class,
		RiskTierAchieved: tier, RiskTierFractionBps: fraction,
		RiskTierDetail:      detail + fmt.Sprintf("; fabric adjustedCap=%d, fabric passed=%v; %s", fabricAdjustedCap, fabricPassed, awsDetail),
		RiskTierChecked:     true,
		RiskTierAwsVerified: awsOK && awsPassed, RiskTierAwsSigner: awsSigner,
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
