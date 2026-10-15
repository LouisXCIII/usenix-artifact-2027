# S10 deep dive results: n=30 scaling of case B and case C (2026-10-04)

Follows `results_2026-10-04.md` (n=1 sanity pass). Run via `run_deep.py` against the same
deployed `authcc` v2.31/seq43, on `SAT[:30]` -- a deterministic 30-fixture subset of the
113-fixture S8 population (all genuinely satisfiable, real committed contracts, real proofs).
All 243 transactions were real `peer chaincode invoke --waitForEvent` calls, mined and endorsed
by both orgs (height 4035 -> 4097). No `eth_estimateGas`/simulation anywhere. Bound convention
(`cp_upper`, Clopper-Pearson one-sided 95%) matches the one already used throughout S7/S8/S9.

## B: obligation-defect sweep -- all three formal modes, not just S5

For each of the 30 fixtures, the SAME genuine proof+header was reused across 5 calls: one
`SubmitDecisionTrustlessV7` baseline, and four `SubmitDecisionTrustlessV7Obligation` calls
(healthy / S5 stripped / S6 weakened / S7 laundered).

| Condition | n | Expected | Matched | Wrong | Errors | 95% CP upper bound if 0 wrong |
|---|---|---|---|---|---|---|
| `v7_alone_baseline` (V7, blind to obligation) | 30 | SAFE | 30/30 | 0 | 0 | 9.50% |
| `healthy` (V7Obligation, good obligation) | 30 | SAFE | 30/30 | 0 | 0 | 9.50% |
| `S5_stripped` (`mappedMechanism=""`) | 30 | REJECT | 30/30 | 0 | 0 | 9.50% |
| `S6_weakened` (`combined`->`notification`, strength 1<4) | 30 | REJECT | 30/30 | 0 | 0 | 9.50% |
| `S7_laundered` (discharged claimed, no enforcement witness) | 30 | REJECT | 30/30 | 0 | 0 | 9.50% |

**Reading:** V7 alone is confirmed blind to obligation on every one of 30 real, distinct
decisions (always SAFE regardless of what the obligation sweep below says about the exact same
underlying contract) -- the n=1 finding in `results_2026-10-04.md` case B was not a fluke of that
one fixture. `SubmitDecisionTrustlessV7Obligation` correctly distinguishes all three formal
obligation-defect modes from S5/S6/S7 (`obligation_v2.go`'s `oblCoveredV2`), each independently,
across 30 real transactions each, with 0 wrong classifications and a 9.5% one-sided 95% upper
bound on the true miss rate for each mode.

## C: proof-chain robustness sweep -- all three established negative-control categories, under a genuinely good obligation claim

For each of the 30 fixtures, `SubmitDecisionTrustlessV7Obligation` was called with **healthy**
obligation parameters (audit/audit/discharged/witnessed -- the best possible obligation story)
but an attacked proof/header side, reusing the exact N1/N2/N3 mutation semantics already
validated against plain `SubmitDecisionTrustlessV7` in S7/S8:

| Category | n | Accepted (wrongly) | Rejected (correctly) | Errors | 95% CP upper bound if 0 accepted |
|---|---|---|---|---|---|
| N1 proof substitution (another fixture's genuine proof) | 30 | 0 | 30 | 0 | 9.50% |
| N2 tampered proof (1 digit flipped in `pi_a[0]`) | 30 | 0 | 30 | 0 | 9.50% |
| N3 forged featureHash (public signal from another fixture) | 30 | 0 | 30 | 0 | 9.50% |

**Reading:** a perfect obligation story never compensates for a broken evidence chain, across 90
real transactions covering all three previously-established attack categories, with 0 wrongly
accepted and a 9.5% bound per category. This is the mirror image of the B sweep: B shows a
perfect evidence chain does not compensate for a broken obligation, C shows a perfect obligation
does not compensate for a broken evidence chain. Together they cover both halves of the
completeness claim with real, bounded evidence rather than one anecdote per direction.

## C (claim-trust generality): `SubmitDecisionObligationV2`'s structural gap across all 3 roles

The n=1 finding (case C in `results_2026-10-04.md`: `SubmitDecisionObligationV2` has no proof
parameter at all, so a bare `semBound=true` claim is accepted at face value) was repeated at
each real compiled role's own threshold cap (`manager`=500000, `director`=2000000, `clerk`=10000),
good obligation parameters, amount = threshold: **3/3 fooled** (all three roles accept EXACT on a
pure claim with zero real proof attached). This confirms the gap is a structural property of the
function's signature (it literally cannot receive a proof to check), not a coincidence of the one
role tested originally.

## What this adds over the n=1 pass

The n=1 pass (`results_2026-10-04.md`) established the mechanism works as designed on one
representative case each. This pass replaces "it happened once" with a real, bounded, reproducible
result across 30 distinct real decisions per condition (243 new real transactions total,
heights 4035->4097), covering all three formal obligation-defect modes and all three established
proof-attack categories, with 0 failures anywhere and a uniform 9.5% one-sided 95% bound --
matching the statistical rigor already established for S7/S8's negative-control tables, not a
weaker ad hoc check.
