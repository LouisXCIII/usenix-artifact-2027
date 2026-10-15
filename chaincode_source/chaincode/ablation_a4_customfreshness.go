// ablation_a4_customfreshness.go -- Tuần 2, bổ sung: biến thể n=60 nhanh cho Ab4 (freshness),
// theo ĐÚNG mẫu đã dùng cho A5 (ablation_a5_quorum.go, requiredQuorum tham số hoá) nhưng tham số
// hoá maxAge thay cho requiredQuorum. checkFreshness (lightclient.go, KHÔNG sửa) đã nhận maxAge
// như một THAM SỐ trong chữ ký hàm của chính nó -- file này không đổi checkFreshness, không đổi
// verifyIbft2Header (full, finality thật đầy đủ, không ablate), chỉ đổi SỐ NGƯỡNG được truyền vào.
//
// Mục đích: minh hoạ ĐÚNG cơ chế checkFreshness (real path từ chối header cũ hơn ngưỡng, ablated
// path NoFreshnessCheck bỏ qua hoàn toàn) với một ngưỡng NHỎ, có thể đạt được trong vài chục giây
// thay vì 24 giờ thật -- bổ sung cho (không thay thế) bằng chứng n=8 trên entry point thật dùng
// đúng ngưỡng 24h production (maxHeaderAgeSeconds) với header tự nhiên đã cũ.
//
// Additive only: SubmitDecisionTrustless/checkFreshness/maxHeaderAgeSeconds không bị sửa/gọi khác
// đi; đây là một entry point MỚI, độc lập.
package chaincode

import (
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
)

// verifySourceStateCustomFreshness is verifySourceState with the SAME real, full IBFT2 header
// verification (verifyIbft2Header, unmodified -- finality/quorum NOT weakened) and the SAME
// checkFreshness function, but maxAge is caller-supplied instead of the fixed maxHeaderAgeSeconds
// constant -- lets the identical freshness mechanism be exercised against a short, disclosed
// threshold instead of the real 24h production bound.
func verifySourceStateCustomFreshness(txSeconds int64, codeHash []byte, headerJSON, registryProofJSON string, maxAge int64) sourceState {
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
	vh, err := verifyIbft2Header(h, hr.Hash, pinned) // real, full finality check -- unablated
	if err != nil {
		return fail("header not verified against the pinned validator set: %v", err)
	}
	if err := checkFreshness(vh.Timestamp, txSeconds, maxAge); err != nil { // same checkFreshness, custom maxAge
		return fail("header rejected: %v", err)
	}
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
		Detail: fmt.Sprintf("Ab4 custom-threshold variant: maxAge=%ds (header of block %s reproduced from raw fields with %d/%d distinct pinned-validator commit seals; registry slots proven by Merkle-Patricia proofs under its stateRoot)",
			maxAge, vh.Number, vh.Committers, len(pinned)),
	}
}

// SubmitDecisionTrustlessCustomFreshness is SubmitDecisionTrustless with maxAge parameterized
// instead of the fixed 24h maxHeaderAgeSeconds (see verifySourceStateCustomFreshness). Finality
// and the registry Merkle-Patricia proofs are still fully, genuinely checked; only the freshness
// THRESHOLD is caller-supplied, to demonstrate the same checkFreshness mechanism at a short,
// disclosed timescale instead of waiting out the real 24h production bound.
func (c *AuthContract) SubmitDecisionTrustlessCustomFreshness(ctx contractapi.TransactionContextInterface,
	decisionID string, sourceDomain string, role string, amount int64, sourceLimit int64,
	codeHash string, proofJSON string, publicSignalsJSON string,
	headerJSON string, registryProofJSON string, maxAgeSeconds int64) (string, error) {

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
	src := verifySourceStateCustomFreshness(nowSec, ch, headerJSON, registryProofJSON, maxAgeSeconds)
	return submitDecisionTrustlessAblated(ctx, decisionID, sourceDomain, role, amount, sourceLimit,
		codeHash, proofJSON, publicSignalsJSON, headerJSON, registryProofJSON, src,
		fmt.Sprintf("Ab4 custom-threshold variant (maxAge=%ds)", maxAgeSeconds))
}
