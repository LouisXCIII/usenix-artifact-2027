// External-system baseline: does a real, independently published capability/delegation library
// (github.com/ucan-wg/go-ucan v1.2.0) reproduce the failure mode of baseline B2 (Section
// subsec:eval-baselines of Manuscriptsp2_Full_v1.tex)? We encode the paper's own running example
// (Section subsec:motivation-auth-vs-admissibility): a source policy lets a manager approve up to
// $1,000,000; the destination permits the same role only up to $500,000; a $700,000 decision is
// cryptographically authentic (validly signed, validly delegated) but not admissible at the
// destination. UCAN's own verifier (Token.ExecutionAllowed) checks the delegation chain, signatures,
// time bounds and the delegation's OWN policy -- nothing about a second, independently governed
// destination policy. It has no notion of conservative refinement across two different policies.
package main

import (
	"fmt"
	"log"

	"github.com/ipfs/go-cid"

	"github.com/ucan-wg/go-did-it/didtest"

	"github.com/ucan-wg/go-ucan/pkg/command"
	"github.com/ucan-wg/go-ucan/pkg/policy"
	"github.com/ucan-wg/go-ucan/pkg/policy/literal"
	"github.com/ucan-wg/go-ucan/token/delegation"
	"github.com/ucan-wg/go-ucan/token/invocation"
)

// mapLoader is the simplest possible delegation.Loader: an in-memory map from CID to Token, exactly
// what a real verifier would do after fetching/receiving the referenced delegation(s).
type mapLoader map[cid.Cid]*delegation.Token

func (m mapLoader) GetDelegation(c cid.Cid) (*delegation.Token, error) {
	t, ok := m[c]
	if !ok {
		return nil, delegation.ErrDelegationNotFound
	}
	return t, nil
}

func run(amount int64, label string) {
	// Real Ed25519 keypairs from go-did-it's own test fixtures, not values we invented.
	source := didtest.PersonaAlice   // the source policy authority (issuer)
	relay := didtest.PersonaBob      // the audience: whoever holds/presents the capability at the destination
	sourceDID, sourcePriv := source.DID(), source.PrivKey()
	relayDID := relay.DID()

	cmd := command.MustParse("/approve/deployment-decision")

	// Source policy: this role may approve amounts up to $1,000,000 -- a real UCAN policy predicate,
	// not a comment. This is the delegation the source authority actually signs.
	sourcePolicy, err := policy.Construct(
		policy.LessThanOrEqual(".amount", must(literal.Any(int64(1000000)))),
	)
	if err != nil {
		log.Fatalf("construct policy: %v", err)
	}

	del, err := delegation.Root(sourceDID, relayDID, cmd, sourcePolicy)
	if err != nil {
		log.Fatalf("build delegation: %v", err)
	}
	sealed, delCid, err := del.ToSealed(sourcePriv)
	if err != nil {
		log.Fatalf("seal delegation: %v", err)
	}
	_ = sealed

	loader := mapLoader{delCid: del}

	// The relay/holder invokes the capability for a concrete $amount decision. This is the destination
	// side: it presents the invocation plus the delegation as proof, and the verifier decides using ONLY
	// UCAN's own machinery -- the delegation chain, signatures, time bounds, and the delegation's policy.
	inv, err := invocation.New(relayDID, cmd, sourceDID, []cid.Cid{delCid},
		invocation.WithArgument("amount", amount),
	)
	if err != nil {
		log.Fatalf("build invocation: %v", err)
	}

	verifyErr := inv.ExecutionAllowed(loader)
	status := "ALLOWED (UCAN's own verifier accepts this invocation)"
	if verifyErr != nil {
		status = fmt.Sprintf("REJECTED by UCAN: %v", verifyErr)
	}

	fmt.Printf("%s: amount=$%d, source policy <=$1,000,000, destination's OWN rule (not known to UCAN) <=$500,000\n", label, amount)
	fmt.Printf("  go-ucan v1.2.0 Token.ExecutionAllowed() -> %s\n", status)
}

func must[T any](v T, err error) T {
	if err != nil {
		log.Fatal(err)
	}
	return v
}

func main() {
	fmt.Println("UCAN baseline (real library: github.com/ucan-wg/go-ucan v1.2.0)")
	fmt.Println("Running example from Manuscriptsp2_Full_v1.tex, subsec:motivation-auth-vs-admissibility")
	fmt.Println()
	run(300000, "IN-BOUNDS case ($300K, within both source and destination limits)")
	run(700000, "WIDENING case ($700K, within source's $1M limit, exceeds destination's $500K limit)")
	run(1500000, "CONTROL: exceeds even the SOURCE's own $1M limit (checks the verifier is not a rubber stamp)")
}
