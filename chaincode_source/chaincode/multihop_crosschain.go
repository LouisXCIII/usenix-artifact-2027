// multihop_crosschain.go -- E9, the deepest remaining gap: the FULL X5a-style light-client (+
// endorsement/nested-inclusion, the Fabric->Besu analog of ZK-proof re-verification) mechanism at
// EVERY hop for h>2, not merely real Fabric-only state-binding (multihop_real.go). Alternates
// real direction, reusing ONLY already-proven-real infrastructure from this session:
//   hop 0 (Besu, origin):    real ML registry state (mlScanPassed[codeHash]), already verified
//                             all session via SubmitDecisionTrustless.
//   hop 1 (Fabric<-Besu):    SubmitDecisionTrustless -- real light client (verifyIbft2Header,
//                             UNCHANGED) + real Groth16 ZK proof re-verification.
//   hop 2 (Besu<-Fabric):    FabricNestedInclusionVerifier.verifyFullChain (contracts/, E6) --
//                             real orderer signature + real endorsement signatures + real nested
//                             inclusion (header->data_hash->envelope->payload), already proven
//                             this session.
//   hop 3 (Fabric<-Besu):    THIS FILE's SubmitHopDecisionCrossChain -- real light client
//                             (verifyIbft2Header, UNCHANGED) + a real Merkle-Patricia proof of
//                             FabricNestedInclusionVerifier's OWN real fullyProvenTx[txIdHash]
//                             storage slot, reusing provenMappingAt (UNCHANGED) pointed at a
//                             second, newly-pinned real contract address instead of the ML
//                             registry -- no new MPT-walking code, no new crypto code.
// Additive only: no existing function or entry point modified.
package chaincode

import (
	"encoding/json"
	"fmt"

	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
)

// fabricNestedInclusionVerifierAddr is the real, deployed FabricNestedInclusionVerifier contract
// (evidence/e6_nested_inclusion_proof_2026-09-24.md) -- pinned as a second real source-of-truth
// address alongside sourceRegistryAddresses, specifically for hop verification of Fabric->Besu
// state (its own fullyProvenTx mapping), never for the ML registry checks SubmitDecisionTrustless
// already performs.
const fabricNestedInclusionVerifierAddr = "0x21297aa46395eb1f7F8346963dE853EEfE5632C6"

// slotFullyProvenTx is FabricNestedInclusionVerifier.sol's storage slot for
// `mapping(bytes32 => bool) public fullyProvenTx` (the contract's 2nd state variable, slot 1;
// `identityKeys` is slot 0).
const slotFullyProvenTx int64 = 1

// SubmitHopDecisionCrossChain is a real light-client-verified hop: it proves, via a fresh Besu
// header (verified against the pinned validator set, exactly as verifyIbft2Header does for every
// other real entry point) and a real Merkle-Patricia proof, that
// FabricNestedInclusionVerifier.fullyProvenTx[txIdHash] is genuinely true on live Besu -- i.e.
// that a PREVIOUS hop's real Fabric transaction was itself already verified on Besu. It then
// chains this real cross-chain fact into the SAME real, composed hop-accumulator
// (multihop_real.go's HopRecord/hopKey), so a cross-chain hop composes with ordinary real hops.
func (c *AuthContract) SubmitHopDecisionCrossChain(ctx contractapi.TransactionContextInterface,
	chainID string, hopIndex int64, txIdHash string, headerJSON string, registryProofJSON string) (string, error) {

	key := hopKey(chainID, "composed", hopIndex)
	existingRec, err := ctx.GetStub().GetState(key)
	if err != nil {
		return "", fmt.Errorf("failed to read state: %v", err)
	}
	if existingRec != nil {
		return "", fmt.Errorf("hop %d of chain %s already recorded (replay rejected)", hopIndex, chainID)
	}

	var hr headerRequest
	if err := json.Unmarshal([]byte(headerJSON), &hr); err != nil {
		return "", fmt.Errorf("malformed header JSON: %v", err)
	}
	var rp registryProof
	if err := json.Unmarshal([]byte(registryProofJSON), &rp); err != nil {
		return "", fmt.Errorf("malformed registry-proof JSON: %v", err)
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
	// Real light-client verification -- the exact same function every other real entry point
	// uses, unchanged: block-hash reconstruction + pinned-validator commit-seal recovery.
	vh, err := verifyIbft2Header(h, hr.Hash, pinned)
	if err != nil {
		return "", fmt.Errorf("hop %d: header not verified against the pinned validator set: %v", hopIndex, err)
	}
	nowSec, err := ledgerNow(ctx)
	if err != nil {
		return "", err
	}
	if err := checkFreshness(vh.Timestamp, nowSec, maxHeaderAgeSeconds); err != nil {
		return "", fmt.Errorf("hop %d: header rejected: %v", hopIndex, err)
	}

	txIdHashBytes, err := unhexBytes(txIdHash)
	if err != nil || len(txIdHashBytes) != 32 {
		return "", fmt.Errorf("txIdHash must be 32 bytes of hex")
	}
	// Real Merkle-Patricia proof of FabricNestedInclusionVerifier.fullyProvenTx[txIdHash], under
	// the light-client-verified stateRoot -- provenMappingAt is reused completely unchanged,
	// pointed at a different real contract address and slot than the ML registry checks.
	val, err := provenMappingAt(vh.StateRoot, rp, fabricNestedInclusionVerifierAddr, txIdHashBytes, slotFullyProvenTx)
	crossChainVerified := err == nil && val != nil && val.Sign() != 0
	if !crossChainVerified {
		detail := "not proven"
		if err != nil {
			detail = err.Error()
		}
		return "", fmt.Errorf("hop %d: previous hop's Besu-side verification not proven: %s", hopIndex, detail)
	}

	// Chain into the same real, composed accumulator as ordinary hops (multihop_real.go).
	composed := crossChainVerified
	if hopIndex > 0 {
		prevKey := hopKey(chainID, "composed", hopIndex-1)
		prevBytes, err := ctx.GetStub().GetState(prevKey)
		if err != nil {
			return "", fmt.Errorf("failed to read previous hop: %v", err)
		}
		if prevBytes == nil {
			return "", fmt.Errorf("previous hop %d of chain %s not found -- real state-binding continuity broken", hopIndex-1, chainID)
		}
		var prev HopRecord
		if err := json.Unmarshal(prevBytes, &prev); err != nil {
			return "", fmt.Errorf("corrupt previous hop record: %v", err)
		}
		composed = prev.ComposedContextOK && crossChainVerified
	}

	class := "REJECT"
	if composed {
		class = "SAFE"
	}
	rec := HopRecord{ChainID: chainID, HopIndex: hopIndex, ContextAtHop: crossChainVerified,
		ComposedContextOK: composed, Mode: "composed", Class: class}
	recJSON, err := json.Marshal(rec)
	if err != nil {
		return "", err
	}
	if err := ctx.GetStub().PutState(key, recJSON); err != nil {
		return "", fmt.Errorf("failed to write state: %v", err)
	}
	return class, nil
}
