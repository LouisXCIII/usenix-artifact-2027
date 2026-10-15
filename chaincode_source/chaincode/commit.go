// commit.go — X3: bind the CONTENT of a decision (role, amount, limit, epoch, expiry, nonce) to the source.
//
// SubmitDecisionTrustless derives SemBound and EvAssuranceOK but still takes role/amount from the caller,
// so a relay can rewrite what was authorized. SubmitDecisionCommitted recomputes
//
//	hD = keccak256(abi.encode(keccak(domain), keccak(intent), codeHash, keccak(role),
//	                          amount, sourceLimit, epoch, expiry, nonce))
//
// from the fields it is asked to act on and proves, from the verified Besu header, that
// DecisionCommitRegistry.decisionCommit[hD] equals the pinned policy authority. The domain tag is this
// destination's own (a decision committed for another destination hashes differently), and the replay key
// is hD, not a caller-chosen id.
package chaincode

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
)

const (
	decisionCommitAddress  = "0x891aFA35016D434782CA036fd6DBfc56A64666bc"
	policyAuthorityAddress = "0x0250812142577b93a70190D88403DB730D0A16De"
	slotDecisionCommit     = 1 // DecisionCommitRegistry: mapping(bytes32=>address) decisionCommit
	destinationDomain      = "fabric:authchannel:authcc"
	decisionIntent         = "approve-deployment-decision"
	currentPolicyEpoch     = 1
)

func word(v int64) []byte {
	b := make([]byte, 32)
	new(big.Int).SetInt64(v).FillBytes(b)
	return b
}

// decisionHash is the Go twin of the ethers/Solidity abi.encode used by the source.
func decisionHash(domain string, codeHash []byte, role string, amount, sourceLimit, epoch, expiry int64, nonce []byte) []byte {
	return keccak256(
		keccak256([]byte(domain)), keccak256([]byte(decisionIntent)), codeHash, keccak256([]byte(role)),
		word(amount), word(sourceLimit), word(epoch), word(expiry), nonce)
}

// commitStatus proves decisionCommit[hD] under stateRoot and reports whether it is the pinned authority.
func commitStatus(stateRoot []byte, cp registryProof, hD []byte) (bool, string) {
	if !strings.EqualFold(cp.Address, decisionCommitAddress) {
		return false, "proof is for a contract that is not the pinned decision-commit registry"
	}
	v, err := provenMappingAt(stateRoot, cp, decisionCommitAddress, hD, slotDecisionCommit)
	if err != nil {
		return false, fmt.Sprintf("commit slot not proven: %v", err)
	}
	if v.Sign() == 0 {
		return false, "no source commitment exists for this exact decision content"
	}
	auth, _ := unhexBytes(policyAuthorityAddress)
	if v.Cmp(new(big.Int).SetBytes(auth)) != 0 {
		return false, "committed by an address other than the pinned policy authority"
	}
	return true, "decision content recomputed into hD and proven committed on the source by the pinned policy authority"
}

// SubmitDecisionCommitted derives SemBound and EvAssuranceOK as SubmitDecisionTrustless does, and in
// addition requires that the exact decision content was committed at the source by the policy authority,
// is not expired, and belongs to the current policy epoch. Replay is keyed on hD.
func (c *AuthContract) SubmitDecisionCommitted(ctx contractapi.TransactionContextInterface,
	role string, amount int64, sourceLimit int64, epoch int64, expiry int64, nonceHex string,
	codeHash string, proofJSON string, publicSignalsJSON string,
	headerJSON string, registryProofJSON string, commitProofJSON string) (string, error) {

	if amount < 0 || sourceLimit < 0 || epoch < 0 || expiry < 0 {
		return "", fmt.Errorf("negative numeric field")
	}
	ch, err := unhexBytes(codeHash)
	if err != nil || len(ch) != 32 {
		return "", fmt.Errorf("codeHash must be 32 bytes of hex")
	}
	nonce, err := unhexBytes(nonceHex)
	if err != nil || len(nonce) != 32 {
		return "", fmt.Errorf("nonce must be 32 bytes of hex")
	}
	hD := decisionHash(destinationDomain, ch, role, amount, sourceLimit, epoch, expiry, nonce)
	key := "hd:" + hex.EncodeToString(hD)
	exists, err := c.decisionExists(ctx, key)
	if err != nil {
		return "", err
	}
	if exists {
		return "", fmt.Errorf("decision %s already recorded (replay rejected)", key)
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
	evOK := src.Verified && src.Approvals >= minApprovals

	commitOK, commitDetail := false, "source header not verified, so no commitment could be proven"
	if src.Verified {
		var cp registryProof
		if err := json.Unmarshal([]byte(commitProofJSON), &cp); err != nil {
			commitDetail = fmt.Sprintf("malformed commit-proof JSON: %v", err)
		} else {
			commitOK, commitDetail = commitStatus(src.StateRoot, cp, hD)
		}
	}
	if commitOK && nowSec > expiry {
		commitOK, commitDetail = false, fmt.Sprintf("decision expired at %d, transaction time %d", expiry, nowSec)
	}
	if commitOK && epoch != currentPolicyEpoch {
		commitOK, commitDetail = false, fmt.Sprintf("decision belongs to policy epoch %d, current epoch is %d", epoch, currentPolicyEpoch)
	}

	threshold, known := RoleThreshold[role]
	if !known {
		threshold = 0
	}
	class := "REJECT"
	if commitOK {
		class = classify(semBound, evOK, amount, threshold, known)
	}
	detail := zkDetail
	if zkOK && src.Verified && !src.MlPassed {
		detail = "proof is valid and bound, but the registry's proven mlScanPassed[codeHash] is false"
	}

	rec := PortableDecisionRecord{
		DecisionID: key, SourceDomain: "besu-main", Role: role, Amount: amount, SourceLimit: sourceLimit,
		SemBound: semBound, EvAssuranceOK: evOK, TransferClass: class, DestThreshold: threshold,
		ZKVerified: true, ZKVerificationDetail: detail, CodeHash: codeHash,
		LCVerified: src.Verified, LCDetail: src.Detail, SourceBlockNumber: src.BlockNumber,
		SourceBlockHash: src.BlockHash, ProvenApprovals: src.Approvals, ProvenMlScanPassed: src.MlPassed,
		HD: "0x" + hex.EncodeToString(hD), CommitVerified: commitOK, CommitDetail: commitDetail,
		Epoch: epoch, Expiry: expiry,
	}
	recJSON, err := json.Marshal(rec)
	if err != nil {
		return "", err
	}
	if err := ctx.GetStub().PutState(key, recJSON); err != nil {
		return "", fmt.Errorf("failed to write state: %v", err)
	}
	return class, nil
}
