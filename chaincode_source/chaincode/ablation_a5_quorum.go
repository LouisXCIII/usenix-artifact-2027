// ablation_a5_quorum.go -- E8 limitation fix: real on-chain analog for A5 (finality check).
// verifyIbft2Header (lightclient.go, UNCHANGED, reused verbatim by every other real entry point)
// requires at least ceil(2n/3) distinct pinned-validator commit seals before treating a header as
// finalized. verifyIbft2HeaderCustomQuorum below is an exact copy of that same verification (same
// RLP decoding, same block-hash reconstruction, same ECDSA seal recovery -- nothing about how a
// seal is checked is weakened) except the REQUIRED NUMBER of distinct seals is a parameter instead
// of the fixed ceil(2n/3) formula -- A5 ablated when called with a weaker requirement. Additive
// only: verifyIbft2Header itself is never modified or called by this file.
package chaincode

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"

	secpecdsa "github.com/decred/dcrd/dcrec/secp256k1/v4/ecdsa"
	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
)

// verifyIbft2HeaderCustomQuorum: identical logic to verifyIbft2Header, with the quorum threshold
// parameterized instead of fixed at ceil(2n/3).
func verifyIbft2HeaderCustomQuorum(h besuHeader, claimedHash string, pinned [][]byte, requiredQuorum int) (*verifiedHeader, error) {
	extra, err := unhexBytes(h.ExtraData)
	if err != nil {
		return nil, err
	}
	root, rest, err := rlpDecode(extra)
	if err != nil || len(rest) != 0 || !root.isList {
		return nil, fmt.Errorf("malformed IBFT2 extraData: %v", err)
	}
	if len(root.list) != 5 {
		return nil, errors.New("extraData is not the plain EC IBFT2 layout (vanity, validators, vote, round, seals)")
	}
	vanity, validators, vote, round, seals := root.list[0], root.list[1], root.list[2], root.list[3], root.list[4]
	if !validators.isList || !seals.isList {
		return nil, errors.New("extraData: validators/seals must be lists")
	}
	if len(validators.list) != len(pinned) {
		return nil, errors.New("header validator set size differs from the pinned set")
	}
	valSet := map[string]bool{}
	for _, v := range validators.list {
		valSet[string(v.data)] = true
	}
	for _, p := range pinned {
		if !valSet[string(p)] {
			return nil, errors.New("header validator set differs from the pinned set")
		}
	}

	onchainExtra := rlpEncList(append(append(append([]byte{}, vanity.raw...), validators.raw...), vote.raw...))
	hdr1, err := encodeBesuHeader(h, onchainExtra)
	if err != nil {
		return nil, err
	}
	blockHash := keccak256(hdr1)
	want, err := unhexBytes(claimedHash)
	if err != nil || !bytes.Equal(blockHash, want) {
		return nil, fmt.Errorf("reconstructed block hash %x does not match the claimed hash", blockHash)
	}

	sealExtra := rlpEncList(append(append(append(append([]byte{}, vanity.raw...), validators.raw...), vote.raw...), round.raw...))
	hdr2, err := encodeBesuHeader(h, sealExtra)
	if err != nil {
		return nil, err
	}
	sealHash := keccak256(hdr2)

	signers := map[string]bool{}
	for _, s := range seals.list {
		if len(s.data) != 65 {
			return nil, fmt.Errorf("commit seal has %d bytes, expected 65", len(s.data))
		}
		compact := make([]byte, 65)
		compact[0] = 27 + s.data[64]
		copy(compact[1:33], s.data[0:32])
		copy(compact[33:65], s.data[32:64])
		pub, _, err := secpecdsa.RecoverCompact(compact, sealHash)
		if err != nil {
			return nil, fmt.Errorf("commit seal recovery failed: %v", err)
		}
		addr := keccak256(pub.SerializeUncompressed()[1:])[12:]
		if valSet[string(addr)] {
			signers[string(addr)] = true
		}
	}
	// A5 ablated when requiredQuorum < ceil(2n/3): the same real signature-recovery check above,
	// just a weaker acceptance threshold on how many distinct real signers are required.
	if len(signers) < requiredQuorum {
		return nil, fmt.Errorf("only %d distinct validator commit seals, quorum is %d", len(signers), requiredQuorum)
	}
	stateRoot, _ := unhexBytes(h.StateRoot)
	ts, ok := new(big.Int).SetString(trim0x(h.Timestamp), 16)
	if !ok {
		return nil, errors.New("bad header timestamp")
	}
	return &verifiedHeader{BlockHash: blockHash, StateRoot: stateRoot, Timestamp: ts.Int64(), Number: h.Number, Committers: len(signers)}, nil
}

func trim0x(s string) string {
	if len(s) >= 2 && s[0] == '0' && (s[1] == 'x' || s[1] == 'X') {
		return s[2:]
	}
	return s
}

// verifySourceStateWeakQuorum is verifySourceState using verifyIbft2HeaderCustomQuorum(quorum=1)
// instead of verifyIbft2Header -- A5 ablated. Freshness and the registry Merkle-Patricia proofs
// are still fully, genuinely checked; only the finality (commit-seal count) requirement is weak.
func verifySourceStateWeakQuorum(txSeconds int64, codeHash []byte, headerJSON, registryProofJSON string, requiredQuorum int) sourceState {
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
	vh, err := verifyIbft2HeaderCustomQuorum(h, hr.Hash, pinned, requiredQuorum)
	if err != nil {
		return fail("header not verified against the pinned validator set: %v", err)
	}
	if err := checkFreshness(vh.Timestamp, txSeconds, maxHeaderAgeSeconds); err != nil {
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
		Detail: fmt.Sprintf("A5 ablated: quorum required=%d (header of block %s reproduced from raw fields with %d/%d distinct pinned-validator commit seals; registry slots proven by Merkle-Patricia proofs under its stateRoot)",
			requiredQuorum, vh.Number, vh.Committers, len(pinned)),
	}
}

// SubmitDecisionTrustlessWeakQuorum is SubmitDecisionTrustless with A5 ablated: requiredQuorum is
// caller-supplied instead of the real ceil(2n/3) (see verifySourceStateWeakQuorum).
func (c *AuthContract) SubmitDecisionTrustlessWeakQuorum(ctx contractapi.TransactionContextInterface,
	decisionID string, sourceDomain string, role string, amount int64, sourceLimit int64,
	codeHash string, proofJSON string, publicSignalsJSON string,
	headerJSON string, registryProofJSON string, requiredQuorum int64) (string, error) {

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
	src := verifySourceStateWeakQuorum(nowSec, ch, headerJSON, registryProofJSON, int(requiredQuorum))
	return submitDecisionTrustlessAblated(ctx, decisionID, sourceDomain, role, amount, sourceLimit,
		codeHash, proofJSON, publicSignalsJSON, headerJSON, registryProofJSON, src, "A5 ablated")
}
