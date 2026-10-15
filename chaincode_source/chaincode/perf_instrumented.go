// perf_instrumented.go — additive, timing-instrumented copy of SubmitDecisionTrustless for
// Manuscriptsp_m2_r3.tex's tab:perf-components component-latency decomposition
// (subsec:perf-latency). Real motivation: measuring component cost via separate
// `peer chaincode query` invocations of the existing ablated-vs-unablated function pairs
// (A1/A4/A2/A3/A6) showed the deltas are dominated by CLI/gRPC per-invocation overhead
// (~45-50ms), swamping the actual in-chaincode computation -- so this file adds real
// time.Now()-based instrumentation INSIDE a single chaincode invocation, isolating the real
// wall-clock cost of each internal step without any CLI-overhead noise. It duplicates
// SubmitDecisionTrustless's exact logic (same calls: verifySemBoundProof, ledgerNow,
// verifySourceState, classify, PutState) rather than modifying it, so the reported real
// systems evidence (Section results-baselines, results-multihop, etc.) is untouched -- this
// is a read-adjacent measurement tool, not a change to any security-relevant path. Not called
// by any other chaincode file; not part of any existing test suite.
package chaincode

import (
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
)

type componentTiming struct {
	Class                 string `json:"class"`
	DecisionExistsCheckNs int64  `json:"decisionExistsCheckNs"`
	ZkProofVerifyNs       int64  `json:"zkProofVerifyNs"`
	LedgerNowReadNs       int64  `json:"ledgerNowReadNs"`
	SourceStateVerifyNs   int64  `json:"sourceStateVerifyNs"`
	ClassifyNs            int64  `json:"classifyNs"`
	PutStateNs            int64  `json:"putStateNs"`
	TotalNs               int64  `json:"totalNs"`
}

// SubmitDecisionTrustlessTimed: identical real logic and identical real security checks as
// SubmitDecisionTrustless (never modified), but wraps each internal step with time.Now() and
// returns the per-step wall-clock cost as JSON instead of just the classification string. Used
// only by the performance-measurement harness (/tmp/e9perf_component_timed.py); does not write
// a PortableDecisionRecord under the SAME decisionID scheme as SubmitDecisionTrustless (still
// calls the real PutState so the real cost is measured, but callers should use disposable
// decisionIDs for repeated timing runs, exactly like SubmitDecisionTrustless itself).
func (c *AuthContract) SubmitDecisionTrustlessTimed(ctx contractapi.TransactionContextInterface,
	decisionID string, sourceDomain string, role string, amount int64, sourceLimit int64,
	codeHash string, proofJSON string, publicSignalsJSON string,
	headerJSON string, registryProofJSON string) (string, error) {

	tStart := time.Now()

	t0 := time.Now()
	exists, err := c.decisionExists(ctx, decisionID)
	tExists := time.Since(t0)
	if err != nil {
		return "", err
	}
	if exists {
		return "", fmt.Errorf("decision %s already recorded (replay rejected)", decisionID)
	}
	ch, err := unhexBytes(codeHash)
	if err != nil || len(ch) != 32 {
		return "", fmt.Errorf("codeHash must be 32 bytes of hex")
	}

	t1 := time.Now()
	zkOK, zkDetail, err := verifySemBoundProof(proofJSON, publicSignalsJSON)
	tZk := time.Since(t1)
	if err != nil {
		return "", fmt.Errorf("malformed verification request: %v", err)
	}
	if zkOK {
		var pub []string
		_ = json.Unmarshal([]byte(publicSignalsJSON), &pub)
		bound, ok := new(big.Int).SetString(pub[1], 10)
		if !ok || bound.Cmp(new(big.Int).SetBytes(ch)) != 0 {
			zkOK = false
			zkDetail = "proof verifies but its public solidityCodeHash differs from this decision's codeHash (proof substitution rejected)"
		}
	}

	t2 := time.Now()
	nowSec, err := ledgerNow(ctx)
	tLedgerNow := time.Since(t2)
	if err != nil {
		return "", err
	}

	t3 := time.Now()
	src := verifySourceState(nowSec, ch, headerJSON, registryProofJSON)
	tSrc := time.Since(t3)

	semBound := zkOK && src.Verified && src.MlPassed
	evOK := src.Verified && src.Approvals >= minApprovals
	detail := zkDetail
	if zkOK && src.Verified && !src.MlPassed {
		detail = "proof is valid and bound, but the registry's proven mlScanPassed[codeHash] is false"
	}

	t4 := time.Now()
	threshold, known := RoleThreshold[role]
	if !known {
		threshold = 0
	}
	class := classify(semBound, evOK, amount, threshold, known)
	tClassify := time.Since(t4)

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
	t5 := time.Now()
	putErr := ctx.GetStub().PutState(decisionID, recJSON)
	tPut := time.Since(t5)
	if putErr != nil {
		return "", fmt.Errorf("failed to write state: %v", putErr)
	}

	out := componentTiming{
		Class:                 class,
		DecisionExistsCheckNs: tExists.Nanoseconds(),
		ZkProofVerifyNs:       tZk.Nanoseconds(),
		LedgerNowReadNs:       tLedgerNow.Nanoseconds(),
		SourceStateVerifyNs:   tSrc.Nanoseconds(),
		ClassifyNs:            tClassify.Nanoseconds(),
		PutStateNs:            tPut.Nanoseconds(),
		TotalNs:               time.Since(tStart).Nanoseconds(),
	}
	outJSON, _ := json.Marshal(out)
	return string(outJSON), nil
}

// obligationTiming / assuranceTiming: per-step real timing for the non-trustless
// SubmitDecisionFull/SubmitDecisionWithAssurance paths (T_obligation, T_assurance), measured
// directly rather than inferred from code inspection, per explicit follow-up request.
type obligationTiming struct {
	Class                 string `json:"class"`
	DecisionExistsCheckNs int64  `json:"decisionExistsCheckNs"`
	RoleMapThresholdNs    int64  `json:"roleMapThresholdNs"`
	ObligationCheckNs     int64  `json:"obligationCheckNs"`
	PutStateNs            int64  `json:"putStateNs"`
	TotalNs               int64  `json:"totalNs"`
}

// SubmitDecisionFullTimed: identical real logic to SubmitDecisionFull (never modified), timed
// per internal step. Isolates the real cost of the obligation check
// (requiresObligation := amount > threshold/2; if requiresObligation && !secondApprovalObtained)
// on its own, separately from role-mapping/threshold resolution and the state write.
func (c *AuthContract) SubmitDecisionFullTimed(ctx contractapi.TransactionContextInterface,
	decisionID string, sourceDomain string, role string, amount int64,
	sourceLimit int64, semBound bool, evAssuranceOK bool,
	secondApprovalObtained bool) (string, error) {

	tStart := time.Now()

	t0 := time.Now()
	exists, err := c.decisionExists(ctx, decisionID)
	tExists := time.Since(t0)
	if err != nil {
		return "", err
	}
	if exists {
		return "", fmt.Errorf("decision %s already recorded (replay rejected)", decisionID)
	}
	if !semBound || !evAssuranceOK {
		return "", fmt.Errorf("timing harness requires semBound=true, evAssuranceOK=true to exercise the real obligation-check branch")
	}

	t1 := time.Now()
	destRole := role
	identityMapping := true
	if mapped, needsMapping := RoleMapping[role]; needsMapping {
		destRole = mapped
		identityMapping = false
	}
	threshold, roleKnown := RoleThreshold[destRole]
	tRoleMap := time.Since(t1)
	if !roleKnown {
		return "", fmt.Errorf("timing harness requires a known destination role")
	}
	if amount > threshold {
		return "", fmt.Errorf("timing harness requires amount <= threshold to reach the obligation check")
	}

	t2 := time.Now()
	requiresObligation := amount > threshold/2
	obligationOK := !requiresObligation || secondApprovalObtained
	tObligation := time.Since(t2)

	class := "REJECTED_BY_HARNESS"
	if obligationOK {
		class = "SAFE"
		if identityMapping {
			class = "EXACT"
		}
	}

	rec := PortableDecisionRecord{
		DecisionID: decisionID, SourceDomain: sourceDomain, Role: role, Amount: amount,
		SourceLimit: sourceLimit, SemBound: semBound, EvAssuranceOK: evAssuranceOK,
		TransferClass: class, DestThreshold: threshold,
		RequiresObligation: requiresObligation, ObligationDischarged: requiresObligation && secondApprovalObtained,
	}
	recJSON, err := json.Marshal(rec)
	if err != nil {
		return "", err
	}
	t3 := time.Now()
	putErr := ctx.GetStub().PutState(decisionID, recJSON)
	tPut := time.Since(t3)
	if putErr != nil {
		return "", fmt.Errorf("failed to write state: %v", putErr)
	}

	out := obligationTiming{
		Class: class, DecisionExistsCheckNs: tExists.Nanoseconds(),
		RoleMapThresholdNs: tRoleMap.Nanoseconds(), ObligationCheckNs: tObligation.Nanoseconds(),
		PutStateNs: tPut.Nanoseconds(), TotalNs: time.Since(tStart).Nanoseconds(),
	}
	outJSON, _ := json.Marshal(out)
	return string(outJSON), nil
}

type assuranceTiming struct {
	Class                 string `json:"class"`
	DecisionExistsCheckNs int64  `json:"decisionExistsCheckNs"`
	RoleMapThresholdNs    int64  `json:"roleMapThresholdNs"`
	AssuranceCheckNs      int64  `json:"assuranceCheckNs"`
	PutStateNs            int64  `json:"putStateNs"`
	TotalNs               int64  `json:"totalNs"`
}

// SubmitDecisionWithAssuranceTimed: identical real logic to SubmitDecisionWithAssurance (never
// modified), timed per internal step. Isolates the real cost of the assurance check
// (claimsFor(decisionSigTier)[destAssuranceRequirement]) on its own.
func (c *AuthContract) SubmitDecisionWithAssuranceTimed(ctx contractapi.TransactionContextInterface,
	decisionID string, sourceDomain string, role string, amount int64,
	sourceLimit int64, semBound bool, evAssuranceOK bool,
	decisionSigTier string, destAssuranceRequirement string) (string, error) {

	tStart := time.Now()

	t0 := time.Now()
	exists, err := c.decisionExists(ctx, decisionID)
	tExists := time.Since(t0)
	if err != nil {
		return "", err
	}
	if exists {
		return "", fmt.Errorf("decision %s already recorded (replay rejected)", decisionID)
	}
	if !semBound || !evAssuranceOK {
		return "", fmt.Errorf("timing harness requires semBound=true, evAssuranceOK=true to exercise the real assurance-check branch")
	}

	t1 := time.Now()
	destRole := role
	identityMapping := true
	if mapped, needsMapping := RoleMapping[role]; needsMapping {
		destRole = mapped
		identityMapping = false
	}
	threshold, roleKnown := RoleThreshold[destRole]
	tRoleMap := time.Since(t1)
	if !roleKnown {
		return "", fmt.Errorf("timing harness requires a known destination role")
	}
	if amount > threshold {
		return "", fmt.Errorf("timing harness requires amount <= threshold to reach the assurance check")
	}

	t2 := time.Now()
	assureOK := claimsFor(decisionSigTier)[destAssuranceRequirement]
	tAssurance := time.Since(t2)

	class := "REJECTED_BY_HARNESS"
	if assureOK {
		class = "SAFE"
		if identityMapping {
			class = "EXACT"
		}
	}

	rec := PortableDecisionRecord{
		DecisionID: decisionID, SourceDomain: sourceDomain, Role: role, Amount: amount,
		SourceLimit: sourceLimit, SemBound: semBound, EvAssuranceOK: evAssuranceOK,
		TransferClass: class, DestThreshold: threshold,
		DecisionSigTier: decisionSigTier, DestAssuranceRequirement: destAssuranceRequirement,
		AssuranceChecked: true,
	}
	recJSON, err := json.Marshal(rec)
	if err != nil {
		return "", err
	}
	t3 := time.Now()
	putErr := ctx.GetStub().PutState(decisionID, recJSON)
	tPut := time.Since(t3)
	if putErr != nil {
		return "", fmt.Errorf("failed to write state: %v", putErr)
	}

	out := assuranceTiming{
		Class: class, DecisionExistsCheckNs: tExists.Nanoseconds(),
		RoleMapThresholdNs: tRoleMap.Nanoseconds(), AssuranceCheckNs: tAssurance.Nanoseconds(),
		PutStateNs: tPut.Nanoseconds(), TotalNs: time.Since(tStart).Nanoseconds(),
	}
	outJSON, _ := json.Marshal(out)
	return string(outJSON), nil
}
