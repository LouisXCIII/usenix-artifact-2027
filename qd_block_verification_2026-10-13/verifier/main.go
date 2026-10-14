// Independent offline verifier for real Hyperledger Fabric blocks exported from the
// authchannel/authcc network of the ADP ("Authorization-Decision Portability") research
// project. Does NOT trust configtxlator, NOT trust any JSON summary the paper's author
// produced, and does NOT import any of the project's own chaincode code. It reads the raw
// .block files directly with the public Fabric protobuf schema and verifies, using only
// standard Go crypto against the channel's own root CA certificates. Every check below runs
// and is reported INDEPENDENTLY (the tool never stops at the first failure), so a FAIL can
// always be attributed to the exact check that caught it:
//
//  1. DataHash: block.Header.DataHash == sha256(concat(raw envelope bytes)).
//  2. Every endorsement signature on every transaction, chained to the correct org's root CA.
//  3. Endorsement policy: the channel's committed default is "MAJORITY Endorsement" with 2
//     orgs (Org1MSP, Org2MSP), i.e. equivalent to AND(Org1MSP, Org2MSP) -- checked as "the set
//     of orgs with a verified endorsement on this transaction contains both".
//  4. The orderer's signature over each block's metadata+header, chained to the orderer
//     org's root CA.
//  5. TRANSACTIONS_FILTER: Fabric still writes invalid transactions (MVCC conflict, failed
//     endorsement policy) into the block; metadata index TRANSACTIONS_FILTER carries the real
//     per-transaction validation code. Only VALID transactions are trusted for RWSet recovery.
//  6. Hash-chain continuity between consecutive blocks (PreviousHash).
//
// Usage:
//   go run main.go verify <blocks-dir> <certs-dir>
//   go run main.go recover <blocks-dir> <decisionId...>
//
// certs-dir must contain: ca.org1.example.com-cert.pem, ca.org2.example.com-cert.pem,
// ca.example.com-orderer-cert.pem (root CAs for Org1MSP, Org2MSP, OrdererMSP respectively).
package main

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/asn1"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"

	"google.golang.org/protobuf/proto"

	cb "github.com/hyperledger/fabric-protos-go-apiv2/common"
	"github.com/hyperledger/fabric-protos-go-apiv2/ledger/rwset"
	"github.com/hyperledger/fabric-protos-go-apiv2/ledger/rwset/kvrwset"
	"github.com/hyperledger/fabric-protos-go-apiv2/msp"
	pb "github.com/hyperledger/fabric-protos-go-apiv2/peer"
)

// asn1Header matches Fabric's own protoutil.BlockHeaderBytes encoding exactly: field order
// (Number, PreviousHash, DataHash) governs the ASN.1 SEQUENCE order, not protobuf.
type asn1Header struct {
	Number       int64
	PreviousHash []byte
	DataHash     []byte
}

func blockHeaderASN1Bytes(h *cb.BlockHeader) []byte {
	out, err := asn1.Marshal(asn1Header{Number: int64(h.Number), PreviousHash: h.PreviousHash, DataHash: h.DataHash})
	if err != nil {
		panic(err)
	}
	return out
}

func readBlock(path string) *cb.Block {
	raw, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	blk := &cb.Block{}
	if err := proto.Unmarshal(raw, blk); err != nil {
		panic(fmt.Sprintf("%s: %v", path, err))
	}
	return blk
}

func loadRootCA(path string) *x509.Certificate {
	raw, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	p, _ := pem.Decode(raw)
	if p == nil {
		panic("no PEM block in " + path)
	}
	cert, err := x509.ParseCertificate(p.Bytes)
	if err != nil {
		panic(err)
	}
	return cert
}

func certFromIdBytes(idBytes []byte) (*x509.Certificate, error) {
	p, _ := pem.Decode(idBytes)
	if p == nil {
		return nil, fmt.Errorf("identity is not PEM")
	}
	return x509.ParseCertificate(p.Bytes)
}

// verifyChain checks leaf was signed by root (direct issuance, as used by the test-network's
// single-tier CA) using only crypto/x509 -- no reliance on any Fabric MSP validation code.
func verifyChain(leaf, root *x509.Certificate) error {
	pool := x509.NewCertPool()
	pool.AddCert(root)
	_, err := leaf.Verify(x509.VerifyOptions{Roots: pool, KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageAny}})
	return err
}

func verifyECDSA(cert *x509.Certificate, signedBytes, sig []byte) error {
	pub, ok := cert.PublicKey.(*ecdsa.PublicKey)
	if !ok {
		return fmt.Errorf("certificate public key is not ECDSA")
	}
	digest := sha256.Sum256(signedBytes)
	if !ecdsa.VerifyASN1(pub, digest[:], sig) {
		return fmt.Errorf("ECDSA signature does not verify")
	}
	return nil
}

type rootCAs struct{ org1, org2, orderer *x509.Certificate }

func (r rootCAs) byMSP(mspid string) (*x509.Certificate, error) {
	switch mspid {
	case "Org1MSP":
		return r.org1, nil
	case "Org2MSP":
		return r.org2, nil
	case "OrdererMSP":
		return r.orderer, nil
	default:
		return nil, fmt.Errorf("unknown MSP id %q", mspid)
	}
}

// checkResult reports every independent check's own PASS/FAIL, never conflated with another
// check's result -- so a reader (or a negative test) can attribute a FAIL to the exact step.
type checkResult struct {
	DataHashOK           bool
	DataHashErrs         []string
	EndorsementsOK       bool
	EndorsementErrs      []string
	NEndorsements        int
	PolicyOK             bool
	PolicyErrs           []string
	OrdererSigOK         bool
	OrdererSigErrs       []string
	ValidationFilterOK     bool   // false only for a genuine FAIL; "N/A" (structurally absent) counts as true
	ValidationFilterStatus string // "PASS" / "N/A" / "FAIL", for display
	ValidationFilterErrs   []string
	TxValidity           []bool // per-transaction, in Data order; true = VALID
}

func (r *checkResult) AllOK() bool {
	return r.DataHashOK && r.EndorsementsOK && r.PolicyOK && r.OrdererSigOK && r.ValidationFilterOK
}

const requiredEndorsingOrgs = 2 // channel's committed default: MAJORITY of 2 orgs = both

// computeTxValidity reads TRANSACTIONS_FILTER independently of any signature verification, so
// cmdRecover can call it without needing root CAs loaded at all.
//
// IMPORTANT, confirmed empirically (2026-10-13) against this network: `peer channel fetch`
// pulls blocks via the ORDERER's deliver service. Fabric's orderer only cuts and orders
// blocks; it never validates transactions, so an orderer-delivered block's copy of
// TRANSACTIONS_FILTER is always an empty byte slice (confirmed on all 25 blocks in an earlier
// pass of this artifact). A PEER's own ledger copy, retrieved instead via the `qscc` system
// chaincode (`peer chaincode query -n qscc -c '{"Args":["GetBlockByNumber",<channel>,<num>]}'`),
// DOES carry the real per-tx codes -- confirmed directly: header, Data, and all signatures in
// the qscc copy are byte-identical to the orderer-delivered copy; only TRANSACTIONS_FILTER
// (now populated) and COMMIT_HASH (a peer-only metadata slot this verifier does not check)
// differ. This artifact's blocks/ now holds the qscc-fetched copies for exactly this reason;
// the orderer-fetched originals are kept, superseded, under blocks/_orderer_fetched_superseded/.
// The three-way return below still exists because it is the honest way to report whichever
// kind of block a caller points this verifier at (not every caller of this tool will have
// re-fetched via qscc):
//   - length == n (tx count): real per-tx codes, checked normally.
//   - length == 0 and n > 0: structurally absent (orderer-sourced block) -- reported as
//     "not applicable", NOT as a FAIL, and NOT silently treated as evidence of validity either
//     (TxValidity defaults to all-true here as a best-effort assumption, explicitly caveated).
//   - any other length: a genuine anomaly (inconsistent with either case above) -- FAIL.
func computeTxValidity(blk *cb.Block) ([]bool, string, []string) {
	n := len(blk.Data.Data)
	valid := make([]bool, n)
	for i := range valid {
		valid[i] = true
	}
	if len(blk.Metadata.Metadata) <= int(cb.BlockMetadataIndex_TRANSACTIONS_FILTER) {
		if n > 0 {
			return valid, "FAIL", []string{"block has no TRANSACTIONS_FILTER metadata slot at all"}
		}
		return valid, "PASS", nil
	}
	filter := blk.Metadata.Metadata[cb.BlockMetadataIndex_TRANSACTIONS_FILTER]
	switch {
	case len(filter) == n:
		for i, code := range filter {
			valid[i] = pb.TxValidationCode(code) == pb.TxValidationCode_VALID
		}
		return valid, "PASS", nil
	case len(filter) == 0 && n > 0:
		return valid, "N/A", []string{"TRANSACTIONS_FILTER not populated -- block was fetched via the orderer's deliver service (peer channel fetch), which never validates transactions; only a committing peer's own ledger copy carries real per-tx validation codes. TxValidity defaults to all-true here, NOT independently confirmed."}
	default:
		return valid, "FAIL", []string{fmt.Sprintf("TRANSACTIONS_FILTER length %d matches neither tx count %d nor the known 'orderer-sourced, unpopulated' pattern (0) -- genuine anomaly", len(filter), n)}
	}
}

// checkBlock runs every check independently and returns their individual results.
func checkBlock(blk *cb.Block, roots rootCAs) *checkResult {
	r := &checkResult{DataHashOK: true, EndorsementsOK: true, PolicyOK: true, OrdererSigOK: true, ValidationFilterOK: true}

	// 1. DataHash: sha256 of the concatenated raw envelope bytes (confirmed empirically
	// against 4 real blocks: no Merkle tree, no separators).
	var concat []byte
	for _, e := range blk.Data.Data {
		concat = append(concat, e...)
	}
	want := sha256.Sum256(concat)
	if hex.EncodeToString(want[:]) != hex.EncodeToString(blk.Header.DataHash) {
		r.DataHashOK = false
		r.DataHashErrs = append(r.DataHashErrs, fmt.Sprintf("header claims %x, actual sha256(Data)=%x", blk.Header.DataHash, want))
	}

	// 5. TRANSACTIONS_FILTER: one byte per transaction, pb.TxValidationCode (0 = VALID).
	r.TxValidity, r.ValidationFilterStatus, r.ValidationFilterErrs = computeTxValidity(blk)
	r.ValidationFilterOK = r.ValidationFilterStatus != "FAIL"

	// 2 + 3. Endorsements + endorsement policy, per transaction. Checked regardless of a
	// transaction's own validity (an invalid tx's signatures can still be checked; its RWSet
	// just won't be trusted by recoverWrites below).
	for txIdx, txBytes := range blk.Data.Data {
		env := &cb.Envelope{}
		if err := proto.Unmarshal(txBytes, env); err != nil {
			r.EndorsementsOK = false
			r.EndorsementErrs = append(r.EndorsementErrs, fmt.Sprintf("tx %d: envelope unmarshal: %v", txIdx, err))
			continue
		}
		payload := &cb.Payload{}
		if err := proto.Unmarshal(env.Payload, payload); err != nil {
			r.EndorsementsOK = false
			r.EndorsementErrs = append(r.EndorsementErrs, fmt.Sprintf("tx %d: payload unmarshal: %v", txIdx, err))
			continue
		}
		chHdr := &cb.ChannelHeader{}
		_ = proto.Unmarshal(payload.Header.ChannelHeader, chHdr)
		if chHdr.Type != int32(cb.HeaderType_ENDORSER_TRANSACTION) {
			continue // config blocks / other types carry no chaincode endorsements
		}
		txn := &pb.Transaction{}
		if err := proto.Unmarshal(payload.Data, txn); err != nil {
			r.EndorsementsOK = false
			r.EndorsementErrs = append(r.EndorsementErrs, fmt.Sprintf("tx %d: transaction unmarshal: %v", txIdx, err))
			continue
		}
		for actIdx, action := range txn.Actions {
			cap := &pb.ChaincodeActionPayload{}
			if err := proto.Unmarshal(action.Payload, cap); err != nil {
				r.EndorsementsOK = false
				r.EndorsementErrs = append(r.EndorsementErrs, fmt.Sprintf("tx %d action %d: ccActionPayload unmarshal: %v", txIdx, actIdx, err))
				continue
			}
			// The channel's committed MAJORITY(2 orgs) default is the `authcc` user
			// chaincode's policy. `_lifecycle` system-chaincode calls (chaincode
			// approve/commit) have a different, expected pattern -- each org submits its
			// OWN single-endorsed approveformyorg transaction, so a bare "1 endorsement" on
			// one of those is normal, not a policy violation. Confirmed live: blocks 5099-5100
			// (the v2.43 chaincode upgrade's own lifecycle transactions) are legitimately
			// 1-endorsement each. Only enforce the 2-org check for the real "authcc" namespace.
			ccName := ""
			if prp := (&pb.ProposalResponsePayload{}); proto.Unmarshal(cap.Action.ProposalResponsePayload, prp) == nil {
				if ccAction := (&pb.ChaincodeAction{}); proto.Unmarshal(prp.Extension, ccAction) == nil && ccAction.ChaincodeId != nil {
					ccName = ccAction.ChaincodeId.Name
				}
			}
			verifiedOrgs := map[string]bool{}
			for eIdx, endorsement := range cap.Action.Endorsements {
				r.NEndorsements++
				sid := &msp.SerializedIdentity{}
				if err := proto.Unmarshal(endorsement.Endorser, sid); err != nil {
					r.EndorsementsOK = false
					r.EndorsementErrs = append(r.EndorsementErrs, fmt.Sprintf("tx %d action %d endorsement %d: identity unmarshal: %v", txIdx, actIdx, eIdx, err))
					continue
				}
				root, err := roots.byMSP(sid.Mspid)
				if err != nil {
					r.EndorsementsOK = false
					r.EndorsementErrs = append(r.EndorsementErrs, fmt.Sprintf("tx %d action %d endorsement %d: %v", txIdx, actIdx, eIdx, err))
					continue
				}
				leaf, err := certFromIdBytes(sid.IdBytes)
				if err != nil {
					r.EndorsementsOK = false
					r.EndorsementErrs = append(r.EndorsementErrs, fmt.Sprintf("tx %d action %d endorsement %d: parse endorser cert: %v", txIdx, actIdx, eIdx, err))
					continue
				}
				if err := verifyChain(leaf, root); err != nil {
					r.EndorsementsOK = false
					r.EndorsementErrs = append(r.EndorsementErrs, fmt.Sprintf("tx %d action %d endorsement %d: cert chain to %s root: %v", txIdx, actIdx, eIdx, sid.Mspid, err))
					continue
				}
				signedBytes := append(append([]byte{}, cap.Action.ProposalResponsePayload...), endorsement.Endorser...)
				if err := verifyECDSA(leaf, signedBytes, endorsement.Signature); err != nil {
					r.EndorsementsOK = false
					r.EndorsementErrs = append(r.EndorsementErrs, fmt.Sprintf("tx %d action %d endorsement %d (%s): %v", txIdx, actIdx, eIdx, sid.Mspid, err))
					continue
				}
				verifiedOrgs[sid.Mspid] = true
			}
			if ccName == "authcc" && len(verifiedOrgs) < requiredEndorsingOrgs {
				r.PolicyOK = false
				r.PolicyErrs = append(r.PolicyErrs, fmt.Sprintf("tx %d action %d (chaincode %q): only %d distinct org(s) with a verified endorsement (%v); channel default MAJORITY Endorsement requires %d of 2 (Org1MSP, Org2MSP)", txIdx, actIdx, ccName, len(verifiedOrgs), keysOf(verifiedOrgs), requiredEndorsingOrgs))
			}
		}
	}

	// 4. Orderer block signature: Metadata[SIGNATURES=0].
	if len(blk.Metadata.Metadata) <= int(cb.BlockMetadataIndex_SIGNATURES) {
		r.OrdererSigOK = false
		r.OrdererSigErrs = append(r.OrdererSigErrs, "block has no SIGNATURES metadata slot")
		return r
	}
	sigMeta := &cb.Metadata{}
	if err := proto.Unmarshal(blk.Metadata.Metadata[cb.BlockMetadataIndex_SIGNATURES], sigMeta); err != nil {
		r.OrdererSigOK = false
		r.OrdererSigErrs = append(r.OrdererSigErrs, fmt.Sprintf("block signature metadata unmarshal: %v", err))
		return r
	}
	if len(sigMeta.Signatures) == 0 {
		r.OrdererSigOK = false
		r.OrdererSigErrs = append(r.OrdererSigErrs, "no orderer signatures present on block")
		return r
	}
	for i, ms := range sigMeta.Signatures {
		sigHdr := &cb.SignatureHeader{}
		if err := proto.Unmarshal(ms.SignatureHeader, sigHdr); err != nil {
			r.OrdererSigOK = false
			r.OrdererSigErrs = append(r.OrdererSigErrs, fmt.Sprintf("orderer signature %d: signature header unmarshal: %v", i, err))
			continue
		}
		sid := &msp.SerializedIdentity{}
		if err := proto.Unmarshal(sigHdr.Creator, sid); err != nil {
			r.OrdererSigOK = false
			r.OrdererSigErrs = append(r.OrdererSigErrs, fmt.Sprintf("orderer signature %d: creator identity unmarshal: %v", i, err))
			continue
		}
		root, err := roots.byMSP(sid.Mspid)
		if err != nil {
			r.OrdererSigOK = false
			r.OrdererSigErrs = append(r.OrdererSigErrs, fmt.Sprintf("orderer signature %d: %v", i, err))
			continue
		}
		leaf, err := certFromIdBytes(sid.IdBytes)
		if err != nil {
			r.OrdererSigOK = false
			r.OrdererSigErrs = append(r.OrdererSigErrs, fmt.Sprintf("orderer signature %d: parse creator cert: %v", i, err))
			continue
		}
		if err := verifyChain(leaf, root); err != nil {
			r.OrdererSigOK = false
			r.OrdererSigErrs = append(r.OrdererSigErrs, fmt.Sprintf("orderer signature %d: cert chain to %s root: %v", i, sid.Mspid, err))
			continue
		}
		// Signed bytes = Metadata.Value || SignatureHeader(raw) || BlockHeader(ASN.1) -- the
		// exact recipe confirmed in common.pb.go's own field comment on
		// MetadataSignature.Signature: "The signature over the concatenation of the Metadata
		// value bytes, signatureHeader, and block header".
		signedBytes := append(append(append([]byte{}, sigMeta.Value...), ms.SignatureHeader...), blockHeaderASN1Bytes(blk.Header)...)
		if err := verifyECDSA(leaf, signedBytes, ms.Signature); err != nil {
			r.OrdererSigOK = false
			r.OrdererSigErrs = append(r.OrdererSigErrs, fmt.Sprintf("orderer signature %d (%s): %v", i, sid.Mspid, err))
		}
	}
	return r
}

func keysOf(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

// recoverWrites scans every VALID transaction's RWSet in the given block for namespace
// "authcc" writes, returning key -> raw value bytes exactly as PutState wrote them. A
// transaction whose TRANSACTIONS_FILTER byte is not VALID is skipped entirely -- Fabric still
// stores invalid transactions' RWSets in the block, and trusting them would let a decision
// that the network actually REJECTED (MVCC conflict, failed endorsement policy) be
// misrecovered as if it had committed.
func recoverWrites(blk *cb.Block, txValidity []bool) map[string][]byte {
	out := map[string][]byte{}
	for txIdx, txBytes := range blk.Data.Data {
		if txIdx < len(txValidity) && !txValidity[txIdx] {
			continue
		}
		env := &cb.Envelope{}
		if proto.Unmarshal(txBytes, env) != nil {
			continue
		}
		payload := &cb.Payload{}
		if proto.Unmarshal(env.Payload, payload) != nil {
			continue
		}
		txn := &pb.Transaction{}
		if proto.Unmarshal(payload.Data, txn) != nil {
			continue
		}
		for _, action := range txn.Actions {
			cap := &pb.ChaincodeActionPayload{}
			if proto.Unmarshal(action.Payload, cap) != nil {
				continue
			}
			prp := &pb.ProposalResponsePayload{}
			if proto.Unmarshal(cap.Action.ProposalResponsePayload, prp) != nil {
				continue
			}
			ccAction := &pb.ChaincodeAction{}
			if proto.Unmarshal(prp.Extension, ccAction) != nil {
				continue
			}
			txrw := &rwset.TxReadWriteSet{}
			if proto.Unmarshal(ccAction.Results, txrw) != nil {
				continue
			}
			for _, ns := range txrw.NsRwset {
				if ns.Namespace != "authcc" {
					continue
				}
				kv := &kvrwset.KVRWSet{}
				if proto.Unmarshal(ns.Rwset, kv) != nil {
					continue
				}
				for _, w := range kv.Writes {
					if !w.IsDelete {
						out[w.Key] = w.Value
					}
				}
			}
		}
	}
	return out
}

func blockNumFromName(name string) (int, bool) {
	re := regexp.MustCompile(`b(\d+)\.block$`)
	m := re.FindStringSubmatch(name)
	if m == nil {
		return 0, false
	}
	n, _ := strconv.Atoi(m[1])
	return n, true
}

type numberedBlock struct {
	num  int
	path string
}

func listBlocks(blocksDir string) []numberedBlock {
	entries, err := os.ReadDir(blocksDir)
	if err != nil {
		panic(err)
	}
	var blocks []numberedBlock
	for _, e := range entries {
		if n, ok := blockNumFromName(e.Name()); ok {
			blocks = append(blocks, numberedBlock{n, filepath.Join(blocksDir, e.Name())})
		}
	}
	sort.Slice(blocks, func(i, j int) bool { return blocks[i].num < blocks[j].num })
	return blocks
}

func printCheck(name string, ok bool, errs []string) {
	status := "PASS"
	if !ok {
		status = "FAIL"
	}
	fmt.Printf("    %-20s %s\n", name+":", status)
	for _, e := range errs {
		fmt.Printf("        - %s\n", e)
	}
}

func cmdVerify(blocksDir, certsDir string) {
	roots := rootCAs{
		org1:    loadRootCA(filepath.Join(certsDir, "ca.org1.example.com-cert.pem")),
		org2:    loadRootCA(filepath.Join(certsDir, "ca.org2.example.com-cert.pem")),
		orderer: loadRootCA(filepath.Join(certsDir, "ca.example.com-orderer-cert.pem")),
	}
	blocks := listBlocks(blocksDir)

	totalOK, totalErrs := 0, 0
	var prevHeader *cb.BlockHeader
	for _, b := range blocks {
		blk := readBlock(b.path)
		r := checkBlock(blk, roots)

		chainOK := true
		chainMsg := "no adjacent predecessor loaded"
		if prevHeader != nil && blk.Header.Number == prevHeader.Number+1 {
			want := sha256.Sum256(blockHeaderASN1Bytes(prevHeader))
			chainOK = hex.EncodeToString(want[:]) == hex.EncodeToString(blk.Header.PreviousHash)
			chainMsg = fmt.Sprintf("previous_hash vs sha256(header[%d])", prevHeader.Number)
		}

		overallOK := r.AllOK() && chainOK
		status := "PASS"
		if !overallOK {
			status = "FAIL"
			totalErrs++
		} else {
			totalOK++
		}
		fmt.Printf("block %d: %s (%d endorsements seen)\n", blk.Header.Number, status, r.NEndorsements)
		printCheck("dataHash", r.DataHashOK, r.DataHashErrs)
		printCheck("endorsements", r.EndorsementsOK, r.EndorsementErrs)
		printCheck("endorsementPolicy", r.PolicyOK, r.PolicyErrs)
		printCheck("ordererSig", r.OrdererSigOK, r.OrdererSigErrs)
		fmt.Printf("    %-20s %s\n", "validationFilter:", r.ValidationFilterStatus)
		for _, e := range r.ValidationFilterErrs {
			fmt.Printf("        - %s\n", e)
		}
		hashChainErrs := []string{}
		if !chainOK {
			hashChainErrs = []string{chainMsg + ": MISMATCH"}
		}
		printCheck("hashChain", chainOK, hashChainErrs)
		prevHeader = blk.Header
	}
	fmt.Printf("\nSummary: %d/%d blocks PASS\n", totalOK, totalOK+totalErrs)
}

func cmdRecover(blocksDir string, keys []string) {
	blocks := listBlocks(blocksDir)
	want := map[string]bool{}
	for _, k := range keys {
		want[k] = true
	}
	found := map[string]string{}
	foundBlock := map[string]int{}
	for _, b := range blocks {
		blk := readBlock(b.path)
		txValidity, _, _ := computeTxValidity(blk) // recover only needs TxValidity, no root CAs
		writes := recoverWrites(blk, txValidity)
		for k, v := range writes {
			if want[k] {
				found[k] = string(v)
				foundBlock[k] = b.num
			}
		}
	}
	for _, k := range keys {
		if v, ok := found[k]; ok {
			fmt.Printf("=== %s (block %d) ===\n%s\n\n", k, foundBlock[k], v)
		} else {
			fmt.Printf("=== %s: NOT FOUND (or not VALID) in %s ===\n\n", k, blocksDir)
		}
	}
}

func main() {
	if len(os.Args) < 3 {
		fmt.Println("usage: go run main.go verify <blocks-dir> <certs-dir>")
		fmt.Println("       go run main.go recover <blocks-dir> <decisionId...>")
		os.Exit(1)
	}
	switch os.Args[1] {
	case "verify":
		if len(os.Args) != 4 {
			fmt.Println("usage: go run main.go verify <blocks-dir> <certs-dir>")
			os.Exit(1)
		}
		cmdVerify(os.Args[2], os.Args[3])
	case "recover":
		cmdRecover(os.Args[2], os.Args[3:])
	default:
		fmt.Println("unknown command:", os.Args[1])
		os.Exit(1)
	}
}
