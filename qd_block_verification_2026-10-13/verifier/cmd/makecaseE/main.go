package main

import (
	"fmt"
	"os"

	"google.golang.org/protobuf/proto"
	cb "github.com/hyperledger/fabric-protos-go-apiv2/common"
)

func main() {
	raw, err := os.ReadFile(os.Args[1])
	if err != nil {
		panic(err)
	}
	blk := &cb.Block{}
	if err := proto.Unmarshal(raw, blk); err != nil {
		panic(err)
	}
	filter := blk.Metadata.Metadata[2]
	fmt.Println("before:", filter)
	// Flip the single VALID (0x00) byte to an arbitrary non-VALID code (1 = INVALID_OTHER_REASON)
	newFilter := make([]byte, len(filter))
	copy(newFilter, filter)
	newFilter[0] = 1
	blk.Metadata.Metadata[2] = newFilter
	fmt.Println("after:", blk.Metadata.Metadata[2])
	out, err := proto.Marshal(blk)
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile(os.Args[2], out, 0o644); err != nil {
		panic(err)
	}
	fmt.Println("wrote", os.Args[2])
}
