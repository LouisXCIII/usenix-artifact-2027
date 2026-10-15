// twohop.go — X5b: the destination consumes an attestation made by an intermediate hop, and derives
// what it can itself.
//
// B0 (Besu main) -> B1 (Besu settlement) -> destination. B1 runs Ibft2RegistryAttester, whose only way to set
// attested[h] is a successful in-Solidity verification of B0's header (seals from B0's pinned validators) and
// Merkle-Patricia proofs of mlScanPassed[h] and approvalCount[h] >= 2. The destination verifies B1's header
// against B1's pinned validators and proves attested[h] under it, and verifies the Groth16 proof itself.
//
// Trust added compared with the direct path: B1's validator quorum (>= 3 of 4 honest, as for B0) and the correctness
// of the attester's bytecode. A hop whose attested[] could be written by an owner or caller (X5a) is not trusted:
// only the pinned attester addresses below are.
package chaincode

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
)

// Genesis validators of the settlement chain (B1, chainId 1338).
var pinnedB1ValidatorHex = []string{
	"b3ba5a1dfaf65d4f2471484edca5d02a8d53785b",
	"a2eb80a3b461c37d60dc0071567c71625371628f",
	"0b03b1899e6fe42ea0263af06af0959fdbebb766",
	"d4703609f64f9449f236e093b42177d996fae62c",
}

// Ibft2RegistryAttester deployments on B1 (slot 0 = mapping(bytes32=>bool) attested).
var attesterAddresses = []string{
	"0x8816CD3d17ccdFCE0d0f2ED6ae8419425560439A", // pinned to the Stage-3 registry
	"0x102d294E708c829547ED47B0cB7cBfE0f9C574f4", // pinned to the Stage-6 (corpus) registry
}

const slotAttested = 0

type b1Attestation struct {
	Verified    bool
	Attested    bool
	Detail      string
	BlockNumber string
	BlockHash   string
}

func pinnedAttester(addr string) (string, bool) {
	for _, a := range attesterAddresses {
		if strings.EqualFold(a, addr) {
			return a, true
		}
	}
	return "", false
}

// verifyB1Attestation verifies B1's header against the pinned B1 validators, checks freshness, and proves
// attested[codeHash] of a pinned attester under that header's stateRoot.
func verifyB1Attestation(txSeconds int64, codeHash []byte, headerJSON, proofJSON string) b1Attestation {
	fail := func(f string, a ...interface{}) b1Attestation { return b1Attestation{Detail: fmt.Sprintf(f, a...)} }
	var hr headerRequest
	if err := json.Unmarshal([]byte(headerJSON), &hr); err != nil {
		return fail("malformed B1 header JSON: %v", err)
	}
	var rp registryProof
	if err := json.Unmarshal([]byte(proofJSON), &rp); err != nil {
		return fail("malformed attestation-proof JSON: %v", err)
	}
	h := besuHeader{
		ParentHash: hr.ParentHash, Sha3Uncles: hr.Sha3Uncles, Miner: hr.Miner, StateRoot: hr.StateRoot,
		TransactionsRoot: hr.TransactionsRoot, ReceiptsRoot: hr.ReceiptsRoot, LogsBloom: hr.LogsBloom,
		Difficulty: hr.Difficulty, Number: hr.Number, GasLimit: hr.GasLimit, GasUsed: hr.GasUsed,
		Timestamp: hr.Timestamp, ExtraData: hr.ExtraData, MixHash: hr.MixHash, Nonce: hr.Nonce,
	}
	var pinned [][]byte
	for _, v := range pinnedB1ValidatorHex {
		b, _ := unhexBytes(v)
		pinned = append(pinned, b)
	}
	vh, err := verifyIbft2Header(h, hr.Hash, pinned)
	if err != nil {
		return fail("B1 header not verified against the pinned B1 validators: %v", err)
	}
	if err := checkFreshness(vh.Timestamp, txSeconds, maxHeaderAgeSeconds); err != nil {
		return fail("B1 header rejected: %v", err)
	}
	att, ok := pinnedAttester(rp.Address)
	if !ok {
		return fail("proof is for a contract that is not a pinned attester")
	}
	v, err := provenMappingAt(vh.StateRoot, rp, att, codeHash, slotAttested)
	if err != nil {
		return fail("attested[codeHash] not proven: %v", err)
	}
	return b1Attestation{Verified: true, Attested: v.Sign() != 0, BlockNumber: vh.Number,
		BlockHash: fmt.Sprintf("0x%x", vh.BlockHash),
		Detail: fmt.Sprintf("B1 header of block %s verified with %d/%d distinct pinned B1 validator seals; attested[codeHash] proven under its stateRoot",
			vh.Number, vh.Committers, len(pinned))}
}

// SubmitDecisionTwoHop: SemBound and EvAssuranceOK both require B1's attestation (which by construction means
// B0 showed mlScanPassed[h] true and >= 2 approvals, verified in Solidity on B1); SemBound additionally requires the
// Groth16 proof, verified here and bound to codeHash.
func (c *AuthContract) SubmitDecisionTwoHop(ctx contractapi.TransactionContextInterface,
	decisionID string, role string, amount int64, sourceLimit int64,
	codeHash string, proofJSON string, publicSignalsJSON string,
	b1HeaderJSON string, attestationProofJSON string) (string, error) {

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
	att := verifyB1Attestation(nowSec, ch, b1HeaderJSON, attestationProofJSON)
	attested := att.Verified && att.Attested
	semBound := zkOK && attested
	evOK := attested

	threshold, known := RoleThreshold[role]
	if !known {
		threshold = 0
	}
	class := classify(semBound, evOK, amount, threshold, known)
	rec := PortableDecisionRecord{
		DecisionID: decisionID, SourceDomain: "besu-b0-via-b1", Role: role, Amount: amount, SourceLimit: sourceLimit,
		SemBound: semBound, EvAssuranceOK: evOK, TransferClass: class, DestThreshold: threshold,
		ZKVerified: true, ZKVerificationDetail: zkDetail, CodeHash: codeHash,
		LCVerified: att.Verified, LCDetail: att.Detail, SourceBlockNumber: att.BlockNumber, SourceBlockHash: att.BlockHash,
		ProvenMlScanPassed: attested,
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

// TxTimestamp is a diagnostic: it returns the transaction timestamp the peer exposes to chaincode
// (ctx.GetStub().GetTxTimestamp), used to test whether a client can choose it. It changes no state.
func (c *AuthContract) TxTimestamp(ctx contractapi.TransactionContextInterface) (int64, error) {
	ts, err := ctx.GetStub().GetTxTimestamp()
	if err != nil {
		return 0, err
	}
	return ts.GetSeconds(), nil
}
