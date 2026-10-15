// ablation_lightclient.go -- E8 limitation fix: real on-chain analogs for A1 (state
// correspondence) and A4 (freshness/revocation), both ablating verifySourceState
// (lightclient.go) at exactly one point each. Additive only: verifyIbft2Header, checkFreshness,
// provenRegistryValue, and every other existing function are reused completely unchanged; nothing
// here modifies an existing entry point.
package chaincode

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
)

// verifySourceStateNoHeaderVerification is verifySourceState with the ENTIRE cryptographic header
// verification step (verifyIbft2Header: block-hash reconstruction + pinned-validator commit-seal
// recovery) removed -- A1 ablated. The caller-supplied StateRoot/Timestamp/Number/Hash fields are
// trusted directly, with no proof this header was ever produced or signed by the real validators;
// only the registry Merkle-Patricia proof (still real) and freshness check are kept. This is the
// real consequence of A1 (R_Sigma state correspondence) being dropped: a state root can be
// presented that never came from a genuinely validator-attested block, as long as an attacker
// happens to hold a real, valid registry proof under that root (e.g. one legitimately fetched
// earlier, for a different, real block).
func verifySourceStateNoHeaderVerification(txSeconds int64, codeHash []byte, headerJSON, registryProofJSON string) sourceState {
	fail := func(f string, a ...interface{}) sourceState {
		return sourceState{Detail: fmt.Sprintf(f, a...)}
	}
	var hr headerRequest
	if err := json.Unmarshal([]byte(headerJSON), &hr); err != nil {
		return fail("malformed header JSON: %v", err)
	}
	var rp registryProof
	if err := json.Unmarshal([]byte(registryProofJSON), &rp); err != nil {
		return fail("malformed registry-proof JSON: %v", err)
	}
	// A1 ablated: verifyIbft2Header is NOT called here.
	stateRoot, err := unhexBytes(hr.StateRoot)
	if err != nil {
		return fail("bad stateRoot: %v", err)
	}
	ts, ok := new(big.Int).SetString(strings.TrimPrefix(hr.Timestamp, "0x"), 16)
	if !ok {
		return fail("bad header timestamp")
	}
	if err := checkFreshness(ts.Int64(), txSeconds, maxHeaderAgeSeconds); err != nil {
		return fail("header rejected: %v", err)
	}
	approvals, err := provenRegistryValue(stateRoot, rp, codeHash, slotApprovalCount)
	if err != nil {
		return fail("approvalCount not proven: %v", err)
	}
	ml, err := provenRegistryValue(stateRoot, rp, codeHash, slotMlScanPassed)
	if err != nil {
		return fail("mlScanPassed not proven: %v", err)
	}
	return sourceState{
		Verified: true, StateRoot: stateRoot, BlockNumber: hr.Number, BlockHash: hr.Hash,
		Approvals: approvals.Int64(), MlPassed: ml.Sign() != 0,
		Detail: "A1 ablated: header NOT cryptographically verified against the pinned validator set -- StateRoot trusted as claimed; registry slots proven by Merkle-Patricia proofs under that unverified root",
	}
}

// verifySourceStateNoFreshnessCheck is verifySourceState with the checkFreshness call removed --
// A4 ablated. Header validity (pinned-validator commit seals) and the registry Merkle-Patricia
// proofs are still fully, genuinely verified; only the header's age is no longer bounded, so an
// arbitrarily stale (but genuinely real, validator-signed) header is accepted.
func verifySourceStateNoFreshnessCheck(txSeconds int64, codeHash []byte, headerJSON, registryProofJSON string) sourceState {
	fail := func(f string, a ...interface{}) sourceState {
		return sourceState{Detail: fmt.Sprintf(f, a...)}
	}
	var hr headerRequest
	if err := json.Unmarshal([]byte(headerJSON), &hr); err != nil {
		return fail("malformed header JSON: %v", err)
	}
	var rp registryProof
	if err := json.Unmarshal([]byte(registryProofJSON), &rp); err != nil {
		return fail("malformed registry-proof JSON: %v", err)
	}
	h := besuHeader{
		ParentHash: hr.ParentHash, Sha3Uncles: hr.Sha3Uncles, Miner: hr.Miner, StateRoot: hr.StateRoot,
		TransactionsRoot: hr.TransactionsRoot, ReceiptsRoot: hr.ReceiptsRoot, LogsBloom: hr.LogsBloom,
		Difficulty: hr.Difficulty, Number: hr.Number, GasLimit: hr.GasLimit, GasUsed: hr.GasUsed,
		Timestamp: hr.Timestamp, ExtraData: hr.ExtraData, MixHash: hr.MixHash, Nonce: hr.Nonce,
	}
	var pinned [][]byte
	for _, v := range pinnedValidatorHex {
		b, _ := unhexBytes(v)
		pinned = append(pinned, b)
	}
	vh, err := verifyIbft2Header(h, hr.Hash, pinned)
	if err != nil {
		return fail("header not verified against the pinned validator set: %v", err)
	}
	// A4 ablated: checkFreshness is NOT called here.
	approvals, err := provenRegistryValue(vh.StateRoot, rp, codeHash, slotApprovalCount)
	if err != nil {
		return fail("approvalCount not proven: %v", err)
	}
	ml, err := provenRegistryValue(vh.StateRoot, rp, codeHash, slotMlScanPassed)
	if err != nil {
		return fail("mlScanPassed not proven: %v", err)
	}
	return sourceState{
		Verified: true, StateRoot: vh.StateRoot, BlockNumber: vh.Number, BlockHash: "0x" + hex.EncodeToString(vh.BlockHash),
		Approvals: approvals.Int64(), MlPassed: ml.Sign() != 0,
		Detail: fmt.Sprintf("A4 ablated: header age NOT checked (header of block %s reproduced from raw fields with %d/%d distinct pinned-validator commit seals; registry slots proven by Merkle-Patricia proofs under its stateRoot)",
			vh.Number, vh.Committers, len(pinned)),
	}
}

func submitDecisionTrustlessAblated(ctx contractapi.TransactionContextInterface,
	decisionID, sourceDomain, role string, amount, sourceLimit int64,
	codeHash, proofJSON, publicSignalsJSON, headerJSON, registryProofJSON string,
	src sourceState, ablationDetail string) (string, error) {

	zkOK, zkDetail, err := verifySemBoundProof(proofJSON, publicSignalsJSON)
	if err != nil {
		return "", fmt.Errorf("malformed verification request: %v", err)
	}
	if zkOK {
		ch, _ := unhexBytes(codeHash)
		var pub []string
		_ = json.Unmarshal([]byte(publicSignalsJSON), &pub)
		bound, ok := new(big.Int).SetString(pub[1], 10)
		if !ok || bound.Cmp(new(big.Int).SetBytes(ch)) != 0 {
			zkOK = false
			zkDetail = "proof verifies but its public solidityCodeHash differs from this decision's codeHash (proof substitution rejected)"
		}
	}
	semBound := zkOK && src.Verified && src.MlPassed
	evOK := src.Verified && src.Approvals >= minApprovals
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
		CodeHash: codeHash, LCVerified: src.Verified, LCDetail: ablationDetail + " | " + src.Detail,
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

// SubmitDecisionTrustlessNoHeaderVerification is SubmitDecisionTrustless with A1 ablated (see
// verifySourceStateNoHeaderVerification).
func (c *AuthContract) SubmitDecisionTrustlessNoHeaderVerification(ctx contractapi.TransactionContextInterface,
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
	nowSec, err := ledgerNow(ctx)
	if err != nil {
		return "", err
	}
	src := verifySourceStateNoHeaderVerification(nowSec, ch, headerJSON, registryProofJSON)
	return submitDecisionTrustlessAblated(ctx, decisionID, sourceDomain, role, amount, sourceLimit,
		codeHash, proofJSON, publicSignalsJSON, headerJSON, registryProofJSON, src, "A1 ablated")
}

// SubmitDecisionTrustlessNoFreshnessCheck is SubmitDecisionTrustless with A4 ablated (see
// verifySourceStateNoFreshnessCheck).
func (c *AuthContract) SubmitDecisionTrustlessNoFreshnessCheck(ctx contractapi.TransactionContextInterface,
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
	nowSec, err := ledgerNow(ctx)
	if err != nil {
		return "", err
	}
	src := verifySourceStateNoFreshnessCheck(nowSec, ch, headerJSON, registryProofJSON)
	return submitDecisionTrustlessAblated(ctx, decisionID, sourceDomain, role, amount, sourceLimit,
		codeHash, proofJSON, publicSignalsJSON, headerJSON, registryProofJSON, src, "A4 ablated")
}
