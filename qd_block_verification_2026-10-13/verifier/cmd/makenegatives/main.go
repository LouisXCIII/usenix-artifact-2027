// Builds 3 tampered block variants from real blocks, for the verifier's negative-test suite.
// Each writes a .block file; the caller then runs `qdblockverify verify <dir> <certs>` on each
// and checks that the correct, specific check fails (not just "some check fails").
package main

import (
	"crypto/sha256"
	"fmt"
	"os"

	"google.golang.org/protobuf/proto"

	cb "github.com/hyperledger/fabric-protos-go-apiv2/common"
	"github.com/hyperledger/fabric-protos-go-apiv2/msp"
	pb "github.com/hyperledger/fabric-protos-go-apiv2/peer"
)

func readBlock(path string) *cb.Block {
	raw, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	blk := &cb.Block{}
	if err := proto.Unmarshal(raw, blk); err != nil {
		panic(err)
	}
	return blk
}

func write(blk *cb.Block, path string) {
	raw, err := proto.Marshal(blk)
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		panic(err)
	}
}

func recomputeDataHash(blk *cb.Block) {
	var concat []byte
	for _, e := range blk.Data.Data {
		concat = append(concat, e...)
	}
	h := sha256.Sum256(concat)
	blk.Header.DataHash = h[:]
}

func main() {
	srcA := os.Args[1] // a block with >=1 ENDORSER_TRANSACTION, e.g. b5082.block
	srcB := os.Args[2] // a DIFFERENT such block, e.g. b5085.block, for the splice case
	outDir := os.Args[3]
	os.MkdirAll(outDir, 0o755)

	// Case A: flip a byte inside the RWSet (ProposalResponsePayload), leave DataHash stale.
	{
		blk := readBlock(srcA)
		idx := len(blk.Data.Data[0]) / 2
		b := []byte(blk.Data.Data[0])
		b[idx] ^= 0xFF
		blk.Data.Data[0] = b
		write(blk, outDir+"/caseA_rwset_tamper_stale_datahash.block")
	}

	// Case B: same tamper as A, but recompute DataHash to match -- isolates the endorsement
	// signature check (must fail) independently of DataHash (must now PASS), and orderer sig
	// (must fail, since orderer signed the OLD header with the OLD DataHash).
	{
		blk := readBlock(srcA)
		idx := len(blk.Data.Data[0]) / 2
		b := []byte(blk.Data.Data[0])
		b[idx] ^= 0xFF
		blk.Data.Data[0] = b
		recomputeDataHash(blk)
		write(blk, outDir+"/caseB_rwset_tamper_fixed_datahash.block")
	}

	// Case C: splice tx 0's ENTIRE raw envelope bytes from block B into block A's tx slot 0
	// (a whole, genuinely-signed-elsewhere envelope pasted into a different block) --
	// DataHash recomputed to match (so DataHash alone would PASS); orderer sig must still FAIL
	// because the real orderer never signed a header whose DataHash matches THIS content.
	{
		blkA := readBlock(srcA)
		blkB := readBlock(srcB)
		blkA.Data.Data[0] = blkB.Data.Data[0]
		recomputeDataHash(blkA)
		write(blkA, outDir+"/caseC_spliced_envelope_fixed_datahash.block")
	}

	// Case D: substitute the first endorsement's identity cert with an untrusted self-signed
	// cert (same Mspid claimed, e.g. Org1MSP) -- the ECDSA signature itself was made with the
	// REAL endorser's key, so it will no longer verify against the fake cert's public key
	// either, but the intent is to isolate the cert-chain-to-root failure specifically.
	{
		blk := readBlock(srcA)
		fakeCertPEM, err := os.ReadFile(os.Args[4]) // path to a self-signed cert not in certs/
		if err != nil {
			panic(err)
		}
		env := &cb.Envelope{}
		proto.Unmarshal(blk.Data.Data[0], env)
		payload := &cb.Payload{}
		proto.Unmarshal(env.Payload, payload)
		txn := &pb.Transaction{}
		proto.Unmarshal(payload.Data, txn)
		cap := &pb.ChaincodeActionPayload{}
		proto.Unmarshal(txn.Actions[0].Payload, cap)
		sid := &msp.SerializedIdentity{}
		proto.Unmarshal(cap.Action.Endorsements[0].Endorser, sid)
		sid.IdBytes = fakeCertPEM
		newEndorser, _ := proto.Marshal(sid)
		cap.Action.Endorsements[0].Endorser = newEndorser
		newCapBytes, _ := proto.Marshal(cap)
		txn.Actions[0].Payload = newCapBytes
		newTxnBytes, _ := proto.Marshal(txn)
		payload.Data = newTxnBytes
		newPayloadBytes, _ := proto.Marshal(payload)
		env.Payload = newPayloadBytes
		newEnvBytes, _ := proto.Marshal(env)
		blk.Data.Data[0] = newEnvBytes
		recomputeDataHash(blk)
		write(blk, outDir+"/caseD_untrusted_cert_fixed_datahash.block")
	}

	fmt.Println("wrote 4 tampered block variants to", outDir)
}
