// lightclient.go — destination-side verification of Besu (IBFT2) source-domain state, closing the
// EvAssuranceOK half of Limitations item 1: instead of trusting a relay-supplied boolean, the
// destination re-derives it from (a) a Besu block header whose committed seals it recovers and
// checks against a validator set pinned at deployment, and (b) Merkle-Patricia proofs
// (eth_getProof) of the RegistryContract storage slots under that header's stateRoot.
//
// Trust anchor: the pinned validator set below (the source network's genesis validators) and the
// registry address. Everything else in the request is untrusted and re-derived.
//
// Scope, stated plainly: this verifies the header of ONE block against a STATIC pinned validator
// set. It does not follow validator-set changes (votes) across blocks; a header whose validator
// set differs from the pinned one is rejected (fail-closed), not re-anchored.
package chaincode

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"

	secpecdsa "github.com/decred/dcrd/dcrec/secp256k1/v4/ecdsa"
	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
	"golang.org/x/crypto/sha3"
)

// Pinned trust anchor for the source domain (Besu IBFT2, chainId 1337): genesis validators.
var pinnedValidatorHex = []string{
	"3ff00c0960daf07ed230c4a789db66e43fd77600",
	"ab3b96e0cb3594fe092fdadf10a53d8033174552",
	"0250812142577b93a70190d88403db730d0a16de",
	"496eeed75e79102238176c56206ca727e2fafc3a",
}

// Pinned source registries: the Stage-3 instance (two entries) and the Stage-6 instance that holds
// the whole 24-case corpus. A proof for any other address is rejected.
var sourceRegistryAddresses = []string{
	"0x93254ACd9089a3fC9c17A7DD8f4b310A5Ff1ad08",
	"0x4BDF7d0CC77F907FeDfc1Ab6B39fa9f4336869c7",
}

const (
	slotMlScanPassed  = 3 // RegistryContract: mapping(bytes32=>bool) mlScanPassed
	slotApprovalCount = 5 // RegistryContract: mapping(bytes32=>uint8) approvalCount
	minApprovals      = 2 // 2-of-3 admin quorum == EvAssuranceOK at the source
	// maxHeaderAgeSeconds bounds how stale the proven header may be relative to the destination's
	// (deterministic) transaction timestamp. 24 h; v1.6 used 30 days to admit the halted network's
	// last block (2026-09-15), which this bound now rejects.
	maxHeaderAgeSeconds = 24 * 3600
)

func pinnedRegistry(addr string) (string, bool) {
	for _, a := range sourceRegistryAddresses {
		if strings.EqualFold(a, addr) {
			return a, true
		}
	}
	return "", false
}

func keccak256(b ...[]byte) []byte {
	h := sha3.NewLegacyKeccak256()
	for _, x := range b {
		h.Write(x)
	}
	return h.Sum(nil)
}

func unhexBytes(s string) ([]byte, error) {
	s = strings.TrimPrefix(s, "0x")
	if len(s)%2 == 1 {
		s = "0" + s
	}
	return hex.DecodeString(s)
}

// ---------- minimal RLP ----------

type rlpItem struct {
	isList bool
	data   []byte
	list   []rlpItem
	raw    []byte
}

func rlpDecode(b []byte) (rlpItem, []byte, error) {
	if len(b) == 0 {
		return rlpItem{}, nil, errors.New("rlp: empty input")
	}
	p := b[0]
	switch {
	case p < 0x80:
		return rlpItem{data: b[:1], raw: b[:1]}, b[1:], nil
	case p <= 0xb7:
		n := int(p - 0x80)
		if len(b) < 1+n {
			return rlpItem{}, nil, errors.New("rlp: short string")
		}
		return rlpItem{data: b[1 : 1+n], raw: b[:1+n]}, b[1+n:], nil
	case p <= 0xbf:
		ll := int(p - 0xb7)
		if len(b) < 1+ll {
			return rlpItem{}, nil, errors.New("rlp: short length")
		}
		n := int(new(big.Int).SetBytes(b[1 : 1+ll]).Int64())
		if n < 0 || len(b) < 1+ll+n {
			return rlpItem{}, nil, errors.New("rlp: short long-string")
		}
		return rlpItem{data: b[1+ll : 1+ll+n], raw: b[:1+ll+n]}, b[1+ll+n:], nil
	case p <= 0xf7:
		n := int(p - 0xc0)
		if len(b) < 1+n {
			return rlpItem{}, nil, errors.New("rlp: short list")
		}
		items, err := rlpDecodeList(b[1 : 1+n])
		return rlpItem{isList: true, list: items, raw: b[:1+n]}, b[1+n:], err
	default:
		ll := int(p - 0xf7)
		if len(b) < 1+ll {
			return rlpItem{}, nil, errors.New("rlp: short list length")
		}
		n := int(new(big.Int).SetBytes(b[1 : 1+ll]).Int64())
		if n < 0 || len(b) < 1+ll+n {
			return rlpItem{}, nil, errors.New("rlp: short long-list")
		}
		items, err := rlpDecodeList(b[1+ll : 1+ll+n])
		return rlpItem{isList: true, list: items, raw: b[:1+ll+n]}, b[1+ll+n:], err
	}
}

func rlpDecodeList(b []byte) ([]rlpItem, error) {
	var out []rlpItem
	for len(b) > 0 {
		it, rest, err := rlpDecode(b)
		if err != nil {
			return nil, err
		}
		out = append(out, it)
		b = rest
	}
	return out, nil
}

func rlpEncBytes(b []byte) []byte {
	if len(b) == 1 && b[0] < 0x80 {
		return []byte{b[0]}
	}
	return append(rlpLenPrefix(0x80, len(b)), b...)
}

func rlpEncList(payload []byte) []byte {
	return append(rlpLenPrefix(0xc0, len(payload)), payload...)
}

func rlpLenPrefix(base byte, n int) []byte {
	if n < 56 {
		return []byte{base + byte(n)}
	}
	lb := new(big.Int).SetInt64(int64(n)).Bytes()
	return append([]byte{base + 55 + byte(len(lb))}, lb...)
}

func rlpEncUint(hexStr string) ([]byte, error) {
	b, err := unhexBytes(hexStr)
	if err != nil {
		return nil, err
	}
	return rlpEncBytes(bytes.TrimLeft(b, "\x00")), nil
}

// ---------- IBFT2 header verification ----------

// besuHeader holds the raw header fields exactly as eth_getBlockByNumber returns them.
type besuHeader struct {
	ParentHash, Sha3Uncles, Miner, StateRoot, TransactionsRoot, ReceiptsRoot string
	LogsBloom, Difficulty, Number, GasLimit, GasUsed, Timestamp              string
	ExtraData, MixHash, Nonce                                                string
}

func encodeBesuHeader(h besuHeader, extraData []byte) ([]byte, error) {
	var payload []byte
	for _, f := range []string{h.ParentHash, h.Sha3Uncles, h.Miner, h.StateRoot, h.TransactionsRoot, h.ReceiptsRoot, h.LogsBloom} {
		b, err := unhexBytes(f)
		if err != nil {
			return nil, err
		}
		payload = append(payload, rlpEncBytes(b)...)
	}
	for _, f := range []string{h.Difficulty, h.Number, h.GasLimit, h.GasUsed, h.Timestamp} {
		e, err := rlpEncUint(f)
		if err != nil {
			return nil, err
		}
		payload = append(payload, e...)
	}
	payload = append(payload, rlpEncBytes(extraData)...)
	for _, f := range []string{h.MixHash, h.Nonce} {
		b, err := unhexBytes(f)
		if err != nil {
			return nil, err
		}
		payload = append(payload, rlpEncBytes(b)...)
	}
	return rlpEncList(payload), nil
}

type verifiedHeader struct {
	BlockHash  []byte
	StateRoot  []byte
	Timestamp  int64
	Number     string
	Committers int
}

// verifyIbft2Header reproduces the claimed block hash from the raw fields, recovers every commit
// seal's signer over the committed-seal hash (Besu BftBlockHashing.calculateDataHashForCommittedSeal:
// header with extraData = [vanity, validators, vote, round], no seals), and requires at least
// ceil(2n/3) distinct signers from the pinned validator set. Only plain 65-byte EC seals are
// accepted; a header carrying non-EC seals (sealSenders field) is rejected, not partially verified.
func verifyIbft2Header(h besuHeader, claimedHash string, pinned [][]byte) (*verifiedHeader, error) {
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
	n := len(validators.list)
	quorum := (2*n + 2) / 3
	if len(signers) < quorum {
		return nil, fmt.Errorf("only %d distinct validator commit seals, quorum is %d", len(signers), quorum)
	}
	stateRoot, _ := unhexBytes(h.StateRoot)
	ts, ok := new(big.Int).SetString(strings.TrimPrefix(h.Timestamp, "0x"), 16)
	if !ok {
		return nil, errors.New("bad header timestamp")
	}
	return &verifiedHeader{BlockHash: blockHash, StateRoot: stateRoot, Timestamp: ts.Int64(), Number: h.Number, Committers: len(signers)}, nil
}

// checkFreshness requires the proven header not to be older than maxAge relative to txSeconds.
func checkFreshness(headerTs, txSeconds, maxAge int64) error {
	if headerTs > txSeconds+300 {
		return errors.New("header timestamp is in the future relative to the transaction")
	}
	if txSeconds-headerTs > maxAge {
		return fmt.Errorf("header is %ds old, exceeds the %ds freshness bound", txSeconds-headerTs, maxAge)
	}
	return nil
}

// ---------- Merkle-Patricia proof verification ----------

func hexPrefixToNibbles(b []byte) (nibbles []byte, isLeaf bool) {
	if len(b) == 0 {
		return nil, false
	}
	flag := b[0] >> 4
	isLeaf = flag&2 != 0
	if flag&1 != 0 {
		nibbles = append(nibbles, b[0]&0x0f)
	}
	for _, x := range b[1:] {
		nibbles = append(nibbles, x>>4, x&0x0f)
	}
	return
}

func keyToNibbles(key []byte) []byte {
	var n []byte
	for _, x := range key {
		n = append(n, x>>4, x&0x0f)
	}
	return n
}

// verifyMPTProof walks proof nodes from root along keyHash (keccak256 of the trie key). It returns
// the raw leaf payload, or nil when the proof validly shows the key is absent from the trie.
func verifyMPTProof(root []byte, keyHash []byte, proof [][]byte) ([]byte, error) {
	path := keyToNibbles(keyHash)
	expect := root
	for i, node := range proof {
		if !bytes.Equal(keccak256(node), expect) {
			return nil, fmt.Errorf("proof node %d hash mismatch", i)
		}
		it, _, err := rlpDecode(node)
		if err != nil || !it.isList {
			return nil, fmt.Errorf("proof node %d is not an RLP list", i)
		}
	walk:
		for {
			switch len(it.list) {
			case 17:
				if len(path) == 0 {
					return it.list[16].data, nil
				}
				child := it.list[path[0]]
				path = path[1:]
				if child.isList {
					it = child
					continue walk
				}
				if len(child.data) == 0 {
					return nil, nil
				}
				expect = child.data
				break walk
			case 2:
				nib, leaf := hexPrefixToNibbles(it.list[0].data)
				if len(path) < len(nib) || !bytes.Equal(path[:len(nib)], nib) {
					return nil, nil
				}
				path = path[len(nib):]
				if leaf {
					if len(path) != 0 {
						return nil, nil
					}
					return it.list[1].data, nil
				}
				child := it.list[1]
				if child.isList {
					it = child
					continue walk
				}
				expect = child.data
				break walk
			default:
				return nil, fmt.Errorf("proof node %d has %d items", i, len(it.list))
			}
		}
	}
	return nil, errors.New("proof ended before reaching a leaf")
}

func decodeStorageValue(v []byte) (*big.Int, error) {
	if v == nil {
		return big.NewInt(0), nil
	}
	it, _, err := rlpDecode(v)
	if err != nil || it.isList {
		return nil, errors.New("bad storage value encoding")
	}
	return new(big.Int).SetBytes(it.data), nil
}

// ---------- registry state proof ----------

type storageProofEntry struct {
	Key   string   `json:"key"`
	Proof []string `json:"proof"`
}

type registryProof struct {
	Address      string              `json:"address"`
	AccountProof []string            `json:"accountProof"`
	StorageProof []storageProofEntry `json:"storageProof"`
}

func decodeNodes(nodes []string) ([][]byte, error) {
	out := make([][]byte, 0, len(nodes))
	for _, n := range nodes {
		b, err := unhexBytes(n)
		if err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, nil
}

// mappingSlotKey = keccak256(codeHash(32) || slot(32)), the Solidity storage key of m[codeHash].
func mappingSlotKey(codeHash []byte, slot int64) []byte {
	pad := make([]byte, 32)
	pad[31] = byte(slot)
	return keccak256(codeHash, pad)
}

// provenRegistryValue proves the value of registry.m[codeHash] at `slot` under stateRoot. The
// storage key is derived here from codeHash, never taken from the request.
func provenRegistryValue(stateRoot []byte, rp registryProof, codeHash []byte, slot int64) (*big.Int, error) {
	pinned, ok := pinnedRegistry(rp.Address)
	if !ok {
		return nil, errors.New("proof is for a contract that is not a pinned registry")
	}
	return provenMappingAt(stateRoot, rp, pinned, codeHash, slot)
}

// provenMappingAt proves the value of m[key] at mapping slot `slot` of the contract at `contract`
// under stateRoot. The storage key is derived here, never taken from the request.
func provenMappingAt(stateRoot []byte, rp registryProof, contract string, codeHash []byte, slot int64) (*big.Int, error) {
	addr, _ := unhexBytes(contract)
	accNodes, err := decodeNodes(rp.AccountProof)
	if err != nil {
		return nil, err
	}
	accRLP, err := verifyMPTProof(stateRoot, keccak256(addr), accNodes)
	if err != nil {
		return nil, fmt.Errorf("account proof: %v", err)
	}
	if accRLP == nil {
		return nil, errors.New("registry account absent from the proven state")
	}
	acc, _, err := rlpDecode(accRLP)
	if err != nil || !acc.isList || len(acc.list) != 4 {
		return nil, errors.New("malformed account record")
	}
	storageRoot := acc.list[2].data

	key := mappingSlotKey(codeHash, slot)
	for _, sp := range rp.StorageProof {
		k, err := unhexBytes(sp.Key)
		if err != nil {
			return nil, err
		}
		if new(big.Int).SetBytes(k).Cmp(new(big.Int).SetBytes(key)) != 0 {
			continue
		}
		nodes, err := decodeNodes(sp.Proof)
		if err != nil {
			return nil, err
		}
		raw, err := verifyMPTProof(storageRoot, keccak256(key), nodes)
		if err != nil {
			return nil, fmt.Errorf("storage proof: %v", err)
		}
		return decodeStorageValue(raw)
	}
	return nil, fmt.Errorf("no storage proof supplied for slot %d of this codeHash", slot)
}

// ---------- SubmitDecisionTrustless: both halves of Authentic_s re-derived at the destination ----------

type headerRequest struct {
	Hash             string `json:"hash"`
	ParentHash       string `json:"parentHash"`
	Sha3Uncles       string `json:"sha3Uncles"`
	Miner            string `json:"miner"`
	StateRoot        string `json:"stateRoot"`
	TransactionsRoot string `json:"transactionsRoot"`
	ReceiptsRoot     string `json:"receiptsRoot"`
	LogsBloom        string `json:"logsBloom"`
	Difficulty       string `json:"difficulty"`
	Number           string `json:"number"`
	GasLimit         string `json:"gasLimit"`
	GasUsed          string `json:"gasUsed"`
	Timestamp        string `json:"timestamp"`
	ExtraData        string `json:"extraData"`
	MixHash          string `json:"mixHash"`
	Nonce            string `json:"nonce"`
}

type sourceState struct {
	Verified    bool
	Detail      string
	BlockNumber string
	BlockHash   string
	Approvals   int64
	MlPassed    bool
	StateRoot   []byte // stateRoot of the verified header; set only when Verified
}

// verifySourceState never returns an error for evidence that merely fails verification: it returns
// Verified=false with the reason, so the caller records a fail-closed REJECT instead of aborting.
func verifySourceState(txSeconds int64, codeHash []byte, headerJSON, registryProofJSON string) sourceState {
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
		Detail: fmt.Sprintf("header of block %s reproduced from raw fields with %d/%d distinct pinned-validator commit seals; registry slots proven by Merkle-Patricia proofs under its stateRoot",
			vh.Number, vh.Committers, len(pinned)),
	}
}

// SubmitDecisionTrustless derives BOTH inputs of Theorem T0 at the destination. SemBound is true
// only if (a) the Groth16 proof verifies against the embedded key, (b) its public signals meet the
// destination's policy, (c) its public solidityCodeHash equals this decision's codeHash (a proof
// for some other contract cannot be substituted), and (d) the registry's mlScanPassed[codeHash],
// proven from Besu state, is true. EvAssuranceOK is true only if the Besu header is verified
// against the pinned validator set and the proven approvalCount[codeHash] >= minApprovals. There
// are no caller-supplied semBound/evAssuranceOk booleans.
func (c *AuthContract) SubmitDecisionTrustless(ctx contractapi.TransactionContextInterface,
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
	// 2026-10-14 guard fix (self-audit finding): this flagship entry point had no
	// amount<=0 check anywhere, unlike most sibling SubmitDecision* functions -- a
	// caller could get a real-proof-verified SAFE/EXACT record at amount=0 or negative,
	// since classification here only ever tests amount>threshold, never amount's sign.
	// Confirmed against the real evidence corpus before adding this: no reported
	// SubmitDecisionTrustless result used amount<=0, so no published number changes.
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
