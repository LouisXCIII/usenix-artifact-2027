// maprec_sign: standalone tool to generate the mapping-governance authority key (once) and to
// sign MapRec digests for the Ab7-as-formally-defined ablation (chaincode/maprec_governance.go).
// Not part of the deployed chaincode package -- excluded from the rsync copy used for
// `network.sh deployCC`, same as every other tools/test-only file in this module.
//
// Usage:
//
//	maprec_sign keygen <keyfile>
//	    Generates a new secp256k1 key, writes the raw 32-byte private key (hex) to <keyfile>,
//	    and prints the corresponding Ethereum-style address
//	    (keccak256(uncompressed pubkey)[12:]) to paste into
//	    chaincode/maprec_governance.go's mappingGovernanceAuthorityAddress constant.
//
//	maprec_sign sign <keyfile> <id> <scope> <role1>=<cap1>[,<role2>=<cap2>,...]
//	    Signs the MapRec digest (same byte layout as chaincode/maprec_governance.go's
//	    mapRecDigest) with the key in <keyfile>, prints the 65-byte compact signature as hex.
package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	secpecdsa "github.com/decred/dcrd/dcrec/secp256k1/v4/ecdsa"
	"golang.org/x/crypto/sha3"
)

func keccak256(parts ...[]byte) []byte {
	h := sha3.NewLegacyKeccak256()
	for _, p := range parts {
		h.Write(p)
	}
	return h.Sum(nil)
}

func word(v int64) []byte {
	b := make([]byte, 32)
	for i := 0; i < 8; i++ {
		b[31-i] = byte(v >> (8 * i))
	}
	return b
}

func mapRecDigest(id, scope string, caps map[string]int64) []byte {
	roles := make([]string, 0, len(caps))
	for r := range caps {
		roles = append(roles, r)
	}
	sort.Strings(roles)
	parts := [][]byte{[]byte(id), []byte(scope)}
	for _, r := range roles {
		parts = append(parts, []byte(r), word(caps[r]))
	}
	return keccak256(parts...)
}

func addressOf(pub *secp256k1.PublicKey) []byte {
	return keccak256(pub.SerializeUncompressed()[1:])[12:]
}

func parseCaps(spec string) (map[string]int64, error) {
	caps := map[string]int64{}
	for _, pair := range strings.Split(spec, ",") {
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) != 2 {
			return nil, fmt.Errorf("bad role=cap pair %q", pair)
		}
		v, err := strconv.ParseInt(kv[1], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("bad cap value in %q: %v", pair, err)
		}
		caps[kv[0]] = v
	}
	return caps, nil
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: maprec_sign keygen <keyfile> | maprec_sign sign <keyfile> <id> <scope> <role1>=<cap1>[,...]")
		os.Exit(1)
	}
	switch os.Args[1] {
	case "keygen":
		if len(os.Args) != 3 {
			fmt.Fprintln(os.Stderr, "usage: maprec_sign keygen <keyfile>")
			os.Exit(1)
		}
		var raw [32]byte
		if _, err := rand.Read(raw[:]); err != nil {
			fmt.Fprintln(os.Stderr, "rand.Read failed:", err)
			os.Exit(1)
		}
		priv := secp256k1.PrivKeyFromBytes(raw[:])
		if err := os.WriteFile(os.Args[2], []byte(hex.EncodeToString(raw[:])), 0600); err != nil {
			fmt.Fprintln(os.Stderr, "write keyfile failed:", err)
			os.Exit(1)
		}
		addr := addressOf(priv.PubKey())
		fmt.Printf("private key written to %s (mode 0600)\n", os.Args[2])
		fmt.Printf("mapping-governance authority address: 0x%x\n", addr)

	case "sign":
		if len(os.Args) != 6 {
			fmt.Fprintln(os.Stderr, "usage: maprec_sign sign <keyfile> <id> <scope> <role1>=<cap1>[,...]")
			os.Exit(1)
		}
		keyHex, err := os.ReadFile(os.Args[2])
		if err != nil {
			fmt.Fprintln(os.Stderr, "read keyfile failed:", err)
			os.Exit(1)
		}
		rawKey, err := hex.DecodeString(strings.TrimSpace(string(keyHex)))
		if err != nil || len(rawKey) != 32 {
			fmt.Fprintln(os.Stderr, "keyfile must contain exactly 64 hex chars (32 bytes)")
			os.Exit(1)
		}
		priv := secp256k1.PrivKeyFromBytes(rawKey)
		id, scope := os.Args[3], os.Args[4]
		caps, err := parseCaps(os.Args[5])
		if err != nil {
			fmt.Fprintln(os.Stderr, "parse caps failed:", err)
			os.Exit(1)
		}
		digest := mapRecDigest(id, scope, caps)
		sig := secpecdsa.SignCompact(priv, digest, false)
		fmt.Printf("digest: 0x%x\n", digest)
		fmt.Printf("signer address: 0x%x\n", addressOf(priv.PubKey()))
		fmt.Printf("compact signature (65 bytes): 0x%x\n", sig)

	default:
		fmt.Fprintln(os.Stderr, "unknown subcommand:", os.Args[1])
		os.Exit(1)
	}
}
