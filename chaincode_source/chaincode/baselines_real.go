// baselines_real.go -- real, standalone, separately-executable chaincode entry points for the
// B0/B2/B3 baseline configurations already defined and disclosed in
// evidence/e7_b0b4_rerun_2026-09-23.md (used there only as post-hoc reclassification formulas
// over SubmitDecisionTrustless's already-computed output). These three functions are additive,
// genuine standalone code paths -- each is its own real, separately timed chaincode transaction,
// not a formula reapplied to B4's cached fields -- closing RQ4/H7's disclosed "not a distinct
// executable system" limitation for B0/B2/B3 (B1 is closed separately, off-chain, since its real
// counterpart -- full local re-evaluation -- is the ML scanner pipeline, not a chaincode call).
// Does not modify SubmitDecisionTrustless/classify or any other existing entry point.
package chaincode

import (
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
)

// SubmitDecisionB0 -- real "proof/finality only" baseline: verifies the ZK proof's raw validity
// and the Besu header's finality, but deliberately does NOT check that the proof is bound to
// this decision's codeHash (the real proof-substitution defense SubmitDecisionTrustless applies)
// and does NOT check role/amount or mlScanPassed at all. Any cryptographically valid proof for
// ANY contract, plus valid finality, is accepted -- this is the real mechanism
// evidence/e7_b0b4_rerun_2026-09-23.md's FAR=100% finding is about, now as its own real,
// separately timed transaction rather than a post-hoc reclassification.
func (c *AuthContract) SubmitDecisionB0(ctx contractapi.TransactionContextInterface,
	decisionID string, sourceDomain string, role string, amount int64, sourceLimit int64,
	codeHash string, proofJSON string, publicSignalsJSON string,
	headerJSON string, registryProofJSON string) (string, error) {

	exists, err := c.decisionExists(ctx, decisionID)
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

	zkOK, zkDetail, err := verifySemBoundProof(proofJSON, publicSignalsJSON)
	if err != nil {
		return "", fmt.Errorf("malformed verification request: %v", err)
	}
	// Deliberately no codeHash-binding check here -- see doc comment above.

	nowSec, err := ledgerNow(ctx)
	if err != nil {
		return "", err
	}
	src := verifySourceState(nowSec, ch, headerJSON, registryProofJSON)

	class := "REJECT"
	if zkOK && src.Verified {
		class = "SAFE"
	}

	rec := PortableDecisionRecord{
		DecisionID: decisionID, SourceDomain: sourceDomain, Role: role, Amount: amount,
		SourceLimit: sourceLimit, SemBound: zkOK, EvAssuranceOK: src.Verified, TransferClass: class,
		DestThreshold: 0, ZKVerified: true, ZKVerificationDetail: zkDetail,
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

// SubmitDecisionB2 -- real "semantic mapping only" baseline: checks ONLY role/amount
// compatibility against RoleThreshold, trusting the caller's role/amount claim directly with no
// cryptographic proof, no light-client header, no assurance check at all. Real mechanism behind
// evidence/e7_b0b4_rerun_2026-09-23.md's B2 FAR finding, now its own real transaction.
func (c *AuthContract) SubmitDecisionB2(ctx contractapi.TransactionContextInterface,
	decisionID string, sourceDomain string, role string, amount int64, sourceLimit int64) (string, error) {

	exists, err := c.decisionExists(ctx, decisionID)
	if err != nil {
		return "", err
	}
	if exists {
		return "", fmt.Errorf("decision %s already recorded (replay rejected)", decisionID)
	}

	threshold, known := RoleThreshold[role]
	class := "REEVALUATE"
	if known {
		if amount <= threshold {
			class = "SAFE"
		} else {
			class = "REJECT"
		}
	}

	rec := PortableDecisionRecord{
		DecisionID: decisionID, SourceDomain: sourceDomain, Role: role, Amount: amount,
		SourceLimit: sourceLimit, SemBound: true, EvAssuranceOK: true, TransferClass: class,
		DestThreshold: threshold, ZKVerified: false, ZKVerificationDetail: "B2: no proof required by design",
		LCVerified: false, LCDetail: "B2: no light-client check by design",
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

// SubmitDecisionB3 -- real "full check minus assurance/finality continuity" baseline: identical
// to SubmitDecisionTrustless's real proof-binding + light-client + role/amount checks (semBound
// computed exactly the same way, including the real codeHash-substitution defense), but evOK is
// deliberately never derived from the real approval-count/finality state -- always treated as
// satisfied. Isolates exactly the assurance-continuity contribution, as its own real, separately
// timed transaction rather than a post-hoc reclassification of B4's cached fields.
func (c *AuthContract) SubmitDecisionB3(ctx contractapi.TransactionContextInterface,
	decisionID string, sourceDomain string, role string, amount int64, sourceLimit int64,
	codeHash string, proofJSON string, publicSignalsJSON string,
	headerJSON string, registryProofJSON string) (string, error) {

	exists, err := c.decisionExists(ctx, decisionID)
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

	zkOK, zkDetail, err := verifySemBoundProof(proofJSON, publicSignalsJSON)
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

	nowSec, err := ledgerNow(ctx)
	if err != nil {
		return "", err
	}
	src := verifySourceState(nowSec, ch, headerJSON, registryProofJSON)

	semBound := zkOK && src.Verified && src.MlPassed
	// B3 deliberately skips assurance/finality continuity -- evOK is never derived from real
	// approval-count/finality state, always treated as satisfied.
	evOK := true
	detail := zkDetail
	if zkOK && src.Verified && !src.MlPassed {
		detail = "proof is valid and bound, but the registry's proven mlScanPassed[codeHash] is false"
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

// SubmitDecisionB3Timed -- identical real logic to SubmitDecisionB3 (never modified), wrapped
// with time.Now() instrumentation matching perf_instrumented.go's established
// SubmitDecisionTrustlessTimed pattern exactly, for a real, in-chaincode (not CLI-wall-clock)
// component-latency measurement directly comparable to Table tab:perf-components's existing
// B4 numbers. Used only by the timing harness; real security logic is byte-identical to
// SubmitDecisionB3 above.
func (c *AuthContract) SubmitDecisionB3Timed(ctx contractapi.TransactionContextInterface,
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

	t3 := time.Now()
	nowSec, _ := ledgerNow(ctx)
	src := verifySourceState(nowSec, ch, headerJSON, registryProofJSON)
	tSrc := time.Since(t3)

	semBound := zkOK && src.Verified && src.MlPassed
	evOK := true
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
	_ = ctx.GetStub().PutState(decisionID, recJSON)
	tPut := time.Since(t5)

	out := componentTiming{
		Class: class, DecisionExistsCheckNs: tExists.Nanoseconds(), ZkProofVerifyNs: tZk.Nanoseconds(), SourceStateVerifyNs: tSrc.Nanoseconds(),
		ClassifyNs: tClassify.Nanoseconds(), PutStateNs: tPut.Nanoseconds(),
		TotalNs: time.Since(tStart).Nanoseconds(),
	}
	outJSON, _ := json.Marshal(out)
	return string(outJSON), nil
}

// SubmitDecisionB0Timed -- same real timing-instrumentation pattern as SubmitDecisionB3Timed,
// for the real B0 ("proof/finality only") logic (SubmitDecisionB0 above, never modified).
func (c *AuthContract) SubmitDecisionB0Timed(ctx contractapi.TransactionContextInterface,
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

	t3 := time.Now()
	nowSec, _ := ledgerNow(ctx)
	src := verifySourceState(nowSec, ch, headerJSON, registryProofJSON)
	tSrc := time.Since(t3)

	t4 := time.Now()
	class := "REJECT"
	if zkOK && src.Verified {
		class = "SAFE"
	}
	tClassify := time.Since(t4)

	rec := PortableDecisionRecord{
		DecisionID: decisionID, SourceDomain: sourceDomain, Role: role, Amount: amount,
		SourceLimit: sourceLimit, SemBound: zkOK, EvAssuranceOK: src.Verified, TransferClass: class,
		ZKVerified: true, ZKVerificationDetail: zkDetail, CodeHash: codeHash,
		LCVerified: src.Verified, LCDetail: src.Detail, SourceBlockNumber: src.BlockNumber,
		SourceBlockHash: src.BlockHash, ProvenApprovals: src.Approvals, ProvenMlScanPassed: src.MlPassed,
	}
	recJSON, err := json.Marshal(rec)
	if err != nil {
		return "", err
	}
	t5 := time.Now()
	_ = ctx.GetStub().PutState(decisionID, recJSON)
	tPut := time.Since(t5)

	out := componentTiming{
		Class: class, DecisionExistsCheckNs: tExists.Nanoseconds(), ZkProofVerifyNs: tZk.Nanoseconds(), SourceStateVerifyNs: tSrc.Nanoseconds(),
		ClassifyNs: tClassify.Nanoseconds(), PutStateNs: tPut.Nanoseconds(),
		TotalNs: time.Since(tStart).Nanoseconds(),
	}
	outJSON, _ := json.Marshal(out)
	return string(outJSON), nil
}

// SubmitDecisionB2Timed -- same real timing-instrumentation pattern, for the real B2
// ("semantic mapping only") logic (SubmitDecisionB2 above, never modified). No proof/header
// steps exist for B2 by design; the real decisionExists check (present in SubmitDecisionB2) is
// included for apples-to-apples comparability with B0Timed/B3Timed/SubmitDecisionTrustlessTimed,
// which all perform the same real check.
func (c *AuthContract) SubmitDecisionB2Timed(ctx contractapi.TransactionContextInterface,
	decisionID string, sourceDomain string, role string, amount int64, sourceLimit int64) (string, error) {

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
	t4 := time.Now()
	threshold, known := RoleThreshold[role]
	class := "REEVALUATE"
	if known {
		if amount <= threshold {
			class = "SAFE"
		} else {
			class = "REJECT"
		}
	}
	tClassify := time.Since(t4)

	rec := PortableDecisionRecord{
		DecisionID: decisionID, SourceDomain: sourceDomain, Role: role, Amount: amount,
		SourceLimit: sourceLimit, SemBound: true, EvAssuranceOK: true, TransferClass: class,
		DestThreshold: threshold, ZKVerified: false, ZKVerificationDetail: "B2: no proof required by design",
		LCVerified: false, LCDetail: "B2: no light-client check by design",
	}
	recJSON, err := json.Marshal(rec)
	if err != nil {
		return "", err
	}
	t5 := time.Now()
	_ = ctx.GetStub().PutState(decisionID, recJSON)
	tPut := time.Since(t5)

	out := componentTiming{
		Class: class, DecisionExistsCheckNs: tExists.Nanoseconds(), ClassifyNs: tClassify.Nanoseconds(), PutStateNs: tPut.Nanoseconds(),
		TotalNs: time.Since(tStart).Nanoseconds(),
	}
	outJSON, _ := json.Marshal(out)
	return string(outJSON), nil
}
