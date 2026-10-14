// Widening corpus-scale extension of the UCAN baseline (Manuscriptsp2_Full_v1.tex,
// subsec:eval-corpus's single running-example UCAN result). The running example only tested one role
// (manager, source limit $1,000,000, destination limit $500,000, per the paper's own running example
// in subsec:motivation-auth-vs-admissibility) at three amounts. The 24-case corpus used elsewhere in
// this paper (stage4_corpus_and_baselines.js) has NO per-role source limit distinct from the
// destination threshold -- it only varies SemBound/EvAssuranceOK, which go-ucan's verifier has no way
// to represent (it is not a delegation/policy/signature concept). Extending the UCAN comparison
// corpus-scale therefore means constructing NEW (source limit, destination limit, amount) triples, not
// replaying the 24 existing labeled cases.
//
// Methodology, stated explicitly so it is not mistaken for a measured fact: for each of the paper's
// three known destination role thresholds (manager <= $500K, director <= $2M, clerk <= $10K, from
// RoleThreshold in the destination chaincode), we set a source-side limit at exactly 2x the destination
// threshold -- the same ratio the paper's own running example already uses for manager ($1M source /
// $500K destination) -- and test three amounts per role at the same relative points the running example
// already uses (0.6x, 1.4x and 3.0x the destination threshold, i.e. $300K/$700K/$1.5M for manager).
// These source limits and multipliers are OUR OWN choice for this extension, not values published
// anywhere else in the paper or corpus; we did not tune them after seeing go-ucan's output.
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

type mapLoader map[cid.Cid]*delegation.Token

func (m mapLoader) GetDelegation(c cid.Cid) (*delegation.Token, error) {
	t, ok := m[c]
	if !ok {
		return nil, delegation.ErrDelegationNotFound
	}
	return t, nil
}

type roleCase struct {
	role                    string
	destThreshold, srcLimit int64
}

func must[T any](v T, err error) T {
	if err != nil {
		log.Fatal(err)
	}
	return v
}

// run builds a real UCAN delegation for sourceLimit, invokes it at amount, and calls go-ucan's own
// unmodified verifier. Returns whether it was ALLOWED and the raw error (nil if allowed).
func run(sourceLimit, amount int64) (bool, error) {
	source := didtest.PersonaAlice
	relay := didtest.PersonaBob
	sourceDID, sourcePriv := source.DID(), source.PrivKey()
	relayDID := relay.DID()

	cmd := command.MustParse("/approve/deployment-decision")
	sourcePolicy, err := policy.Construct(
		policy.LessThanOrEqual(".amount", must(literal.Any(sourceLimit))),
	)
	if err != nil {
		log.Fatalf("construct policy: %v", err)
	}
	del, err := delegation.Root(sourceDID, relayDID, cmd, sourcePolicy)
	if err != nil {
		log.Fatalf("build delegation: %v", err)
	}
	_, delCid, err := del.ToSealed(sourcePriv)
	if err != nil {
		log.Fatalf("seal delegation: %v", err)
	}
	loader := mapLoader{delCid: del}
	inv, err := invocation.New(relayDID, cmd, sourceDID, []cid.Cid{delCid}, invocation.WithArgument("amount", amount))
	if err != nil {
		log.Fatalf("build invocation: %v", err)
	}
	verifyErr := inv.ExecutionAllowed(loader)
	return verifyErr == nil, verifyErr
}

func main() {
	fmt.Println("UCAN widening corpus-scale extension (real library: github.com/ucan-wg/go-ucan v1.2.0)")
	fmt.Println("Source limit = 2x destination threshold (same ratio as the paper's manager running example);")
	fmt.Println("amounts at 0.6x / 1.4x / 3.0x the destination threshold (same relative points as the running example).")
	fmt.Println()

	cases := []roleCase{
		{"manager", 500000, 1000000},
		{"director", 2000000, 4000000},
		{"clerk", 10000, 20000},
	}

	nWideningAllowed, nWideningTotal := 0, 0
	nInBoundsCorrect, nInBoundsTotal := 0, 0
	nControlRejected, nControlTotal := 0, 0

	for _, c := range cases {
		points := []struct {
			bucket string
			mult   float64
		}{
			{"in-bounds (<=dest)", 0.6},
			{"widening (>dest, <=src)", 1.4},
			{"control (>src)", 3.0},
		}
		for _, p := range points {
			amount := int64(float64(c.destThreshold) * p.mult)
			allowed, verr := run(c.srcLimit, amount)
			status := "ALLOWED"
			if !allowed {
				status = fmt.Sprintf("REJECTED (%v)", verr)
			}
			fmt.Printf("role=%-9s bucket=%-24s amount=$%-9d dest<=$%-9d src<=$%-9d -> %s\n",
				c.role, p.bucket, amount, c.destThreshold, c.srcLimit, status)

			switch p.bucket {
			case "in-bounds (<=dest)":
				nInBoundsTotal++
				if allowed {
					nInBoundsCorrect++
				}
			case "widening (>dest, <=src)":
				nWideningTotal++
				if allowed {
					nWideningAllowed++
				}
			case "control (>src)":
				nControlTotal++
				if !allowed {
					nControlRejected++
				}
			}
		}
	}

	fmt.Println()
	fmt.Printf("in-bounds cases correctly ALLOWED by UCAN: %d/%d\n", nInBoundsCorrect, nInBoundsTotal)
	fmt.Printf("widening cases WRONGLY ALLOWED by UCAN (exceed destination's own limit, UCAN cannot see it): %d/%d\n", nWideningAllowed, nWideningTotal)
	fmt.Printf("control cases correctly REJECTED by UCAN (exceed even the source's own limit): %d/%d\n", nControlRejected, nControlTotal)
}
