// risktier_aws is a standalone, independent re-implementation (not a call into the authcc Go
// module) of the risk-tier -> cap-fraction M_I mapping (Tru 4, risk_tier.go), run under a
// separate administrative domain (this AWS account, same account/key-generation pattern as
// qd2aws but a distinct key and a distinct experiment) to give genuine administrative
// independence over the governance table specifically, not just over obligation-discharge
// coverage (qd2aws's existing scope).
//
// Two subcommands:
//
//	risktier_aws keygen
//	    One-time: generates a secp256k1 keypair, writes the private key to
//	    ~/.risktier_domain_key (0600, never printed), prints the derived Ethereum-style
//	    address once -- the "risk-tier domain authority" identity to hardcode into the
//	    chaincode.
//
//	risktier_aws verify-sign < request.json
//	    Reads {decisionId, role, amount, achievedTier} from stdin, independently
//	    RE-DERIVES the fraction-of-cap allowed from its own copy of the governance table
//	    (re-typed from the paper's own description, not copy-pasted from risk_tier.go),
//	    computes adjustedCap and whether amount<=adjustedCap, and signs an attestation digest
//	    with the key from ~/.risktier_domain_key. Always prints a JSON result to stdout.
package main

import (
	"bufio"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/decred/dcrd/dcrec/secp256k1/v4/ecdsa"
	"golang.org/x/crypto/sha3"
)

func keyPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".risktier_domain_key")
}

func keccak256(b ...[]byte) []byte {
	h := sha3.NewLegacyKeccak256()
	for _, x := range b {
		h.Write(x)
	}
	return h.Sum(nil)
}

func addressFromPub(pub *secp256k1.PublicKey) []byte {
	return keccak256(pub.SerializeUncompressed()[1:])[12:]
}

func doKeygen() error {
	if _, err := os.Stat(keyPath()); err == nil {
		return fmt.Errorf("key already exists at %s -- refusing to overwrite; delete it first if you really want a new identity", keyPath())
	}
	priv, err := secp256k1.GeneratePrivateKey()
	if err != nil {
		return err
	}
	if err := os.WriteFile(keyPath(), []byte(hex.EncodeToString(priv.Serialize())), 0600); err != nil {
		return err
	}
	addr := addressFromPub(priv.PubKey())
	fmt.Printf("risk-tier domain authority address: 0x%x\n", addr)
	fmt.Printf("(private key written to %s, mode 0600, never transmitted)\n", keyPath())
	return nil
}

func loadKey() (*secp256k1.PrivateKey, error) {
	raw, err := os.ReadFile(keyPath())
	if err != nil {
		return nil, fmt.Errorf("no domain key at %s -- run 'risktier_aws keygen' first: %v", keyPath(), err)
	}
	b, err := hex.DecodeString(string(raw))
	if err != nil {
		return nil, err
	}
	return secp256k1.PrivKeyFromBytes(b), nil
}

// --- independent re-implementation of the tier -> fraction governance table ---
// Deliberately NOT imported from risk_tier.go: a from-scratch second implementation under
// separate administrative control, re-typed from the same values the paper's governance table
// specifies (0%, 10%, 50%, 100% for tiers 0-3), matching this project's established "dual
// independent implementation" methodology (liboqs/circl oracles, qd2aws's obligation model).
var roleThresholdIndependent = map[string]int64{
	"manager":  500000,
	"director": 2000000,
	"clerk":    10000,
}

var tierFractionBpsIndependent = map[int64]int64{0: 0, 1: 1000, 2: 5000, 3: 10000}

type request struct {
	DecisionID   string `json:"decisionId"`
	Role         string `json:"role"`
	Amount       int64  `json:"amount"`
	AchievedTier int64  `json:"achievedTier"`
}

type result struct {
	Passed      bool   `json:"passed"`
	AdjustedCap int64  `json:"adjustedCap"`
	Reasoning   string `json:"reasoning"`
	Signature   string `json:"signature,omitempty"`
	Signer      string `json:"signer,omitempty"`
	Error       string `json:"error,omitempty"`
}

func word(v int64) []byte {
	b := make([]byte, 8)
	for i := 0; i < 8; i++ {
		b[7-i] = byte(v >> (8 * i))
	}
	return b
}

// digest matches risk_tier.go's awsRiskTierDigest exactly: both sides must agree on this byte
// layout or recovery fails even for a genuine signature.
func digest(decisionID, role string, amount, achievedTier, adjustedCap int64, passed bool) []byte {
	passedByte := byte(0x00)
	if passed {
		passedByte = 0x01
	}
	return keccak256([]byte(decisionID), []byte(role), word(amount), word(achievedTier), word(adjustedCap), []byte{passedByte})
}

func doVerifySign() error {
	data, err := io.ReadAll(bufio.NewReader(os.Stdin))
	if err != nil {
		return err
	}
	var req request
	if err := json.Unmarshal(data, &req); err != nil {
		return err
	}
	cap, known := roleThresholdIndependent[req.Role]
	res := result{}
	if !known {
		res.Reasoning = fmt.Sprintf("unknown role %q in this domain's own role table", req.Role)
	} else if req.AchievedTier < 0 || req.AchievedTier > 3 {
		res.Reasoning = fmt.Sprintf("achievedTier %d outside the known 0-3 band", req.AchievedTier)
	} else {
		fraction := tierFractionBpsIndependent[req.AchievedTier]
		adjustedCap := cap * fraction / 10000
		res.AdjustedCap = adjustedCap
		res.Passed = req.AchievedTier >= 1 && req.Amount <= adjustedCap
		res.Reasoning = fmt.Sprintf("independent re-derivation: role=%s cap=%d tier=%d fraction=%d bps adjustedCap=%d amount=%d passed=%v",
			req.Role, cap, req.AchievedTier, fraction, adjustedCap, req.Amount, res.Passed)
	}
	priv, err := loadKey()
	if err != nil {
		res.Error = err.Error()
		json.NewEncoder(os.Stdout).Encode(res)
		return nil
	}
	d := digest(req.DecisionID, req.Role, req.Amount, req.AchievedTier, res.AdjustedCap, res.Passed)
	sig := ecdsa.SignCompact(priv, d, false)
	res.Signature = "0x" + hex.EncodeToString(sig)
	res.Signer = "0x" + hex.EncodeToString(addressFromPub(priv.PubKey()))
	return json.NewEncoder(os.Stdout).Encode(res)
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: risktier_aws keygen | risktier_aws verify-sign < request.json")
		os.Exit(1)
	}
	var err error
	switch os.Args[1] {
	case "keygen":
		err = doKeygen()
	case "verify-sign":
		err = doVerifySign()
	default:
		err = fmt.Errorf("unknown subcommand %q", os.Args[1])
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
