# E5 PQ-migration profile sweep (P0-P3), real on-chain, closing the Corollary cor:pq-scope gap

**Date:** 2026-10-11. **Trigger:** comparison against the advisor's
`ADP_r3_Experimental_Execution_Guide.docx` (section 7) found that only 2 of 5 PQ-migration
evidence configurations had been exercised as on-chain transactions, and specifically that P1 --
the Guide's own "critical negative control" (a valid classical decision signature travelling over
a PQ-secure channel must not satisfy a PQ decision-authentication requirement) -- had never been
run on-chain. The manuscript's own text already disclosed this honestly (lines ~3717-3721:
"We have not run that specific case as an on-chain transaction... closing this with a dedicated
on-chain case remains open work"). This evidence file closes that gap.

## The Guide's P0-P4 profile table (read directly from the .docx, not paraphrased)

| Profile | Decision evidence | Transport | Destination requirement | Expected interpretation |
|---|---|---|---|---|
| P0 | Classical decision signature | Classical transport | Classical decision auth accepted | Positive control |
| P1 | Classical decision signature | PQ/hybrid transport | PQ decision auth required | Must not be Safe solely because channel is PQ |
| P2 | Classical + independent accepted ML-DSA decision evidence | PQ/hybrid transport | Hybrid/dual required | Can become Safe if all other conditions hold |
| P3 | PQ decision signature | PQ transport | PQ decision auth required | PQ positive control |
| P4 | Classical decision signature | Hybrid transport | Classical decision auth accepted | Separates transport migration from policy requirement |

"Critical negative control: P1 must never satisfy a destination requirement for PQ decision
authentication. PQ-secure transport is not PQ-authenticated authorization."

## No new chaincode was needed

`~/fabric-network/authcc/chaincode/assurance_a6.go` (added 2026-09-24 for the A6 ablation) already
implements exactly the mechanism these profiles need: `SubmitDecisionWithAssurance` takes a real
`decisionSigTier` ("classical"/"hybrid"/"pq") and a `destAssuranceRequirement` string, and gates
the decision on `claimsFor(decisionSigTier)[destAssuranceRequirement]` -- `claimsFor` maps
classical->{ClassicalDecisionAuth}, hybrid->{ClassicalDecisionAuth, PQDecisionAuth},
pq->{PQDecisionAuth}, mirroring `e1_classifier.py`'s `_claims()` exactly. This is deployed and
live: chaincode v2.42/seq54 on `authchannel` (confirmed via `peer lifecycle chaincode
querycommitted` before running anything below).

One real on-chain case matching P1 exactly (`decisionSigTier=classical`,
`destAssuranceRequirement=PQDecisionAuth` -> REJECT) was in fact already run on 2026-09-24, under
the "A6 ablation" framing (`evidence/e8_a6_and_a2_families_2026-09-24.md`), not labeled or
reported as the Guide's "P1" profile. What was missing was (a) a dedicated run under the Guide's
own profile naming/table so the gap-closure is legible against the Guide directly, and (b) P3 (the
PQ positive control), which had never been run at all -- the 2026-09-24 pass only exercised
"classical" and "hybrid" tiers, never "pq".

## What was run (real, committed Fabric transactions, chaincode v2.42/seq54, 2026-10-11)

Fixed inputs across all four profiles (role/amount held constant so only the signature-tier /
requirement pair varies, per the Guide's own step 1: "Generate one semantically identical Safe
case so only assurance evidence changes"): `role="manager"`, `amount=200000` (well under
`RoleThreshold["manager"]=500000`), `semBound=true`, `evAssuranceOK=true`.

| Profile | `decisionSigTier` | `destAssuranceRequirement` | Expected | Real result | Decision ID |
|---|---|---|---|---|---|
| P0 | classical | ClassicalDecisionAuth | EXACT (accepted) | **EXACT** | `p1sweep-P0-1791723134` |
| **P1** | classical | **PQDecisionAuth** | **REJECT** | **REJECT** | `p1sweep-P1-1791723134` |
| P2 | hybrid | PQDecisionAuth | EXACT (accepted) | **EXACT** | `p1sweep-P2-1791723134` |
| P3 | pq | PQDecisionAuth | EXACT (accepted) | **EXACT** | `p1sweep-P3-1791723134` |

**4/4 matched expectation.** All four are real, committed transactions on the live `authchannel`:
channel height was 5074 before this run and 5078 after (confirmed via `peer channel getinfo`),
exactly +4 blocks for the 4 transactions -- not simulated, not `eth_estimateGas`-style dry runs.
Each result was read back fresh via `ReadDecision` (not trusted from the invoke's own stdout).
Full records, including every field of each `PortableDecisionRecord`, in
`p1_profile_sweep_result_2026-10-11.json`.

**P1 specifically** (the Guide's mandatory negative control): a decision carrying only a classical
signature-tier claim, submitted against a destination requiring `PQDecisionAuth`, was correctly
**REJECTed** -- confirming on real, committed chain state (not just the formal benchmark's 385-case
slice, which was the only prior evidence for Corollary~cor:pq-scope) that `claimsFor("classical")`
does not grant `PQDecisionAuth`, i.e. that transport-level PQ-security claims cannot be laundered
into decision-level PQ-authentication by this destination.

## What this does and does not establish

**Does establish**: the specific property Corollary cor:pq-scope states (a classical-signature
decision cannot satisfy a PQ decision-authentication requirement) now has a real on-chain
transaction as evidence, not only the formal benchmark's 385-case slice. P3 (PQ positive control)
is now also demonstrated on-chain for the first time.

**Does not establish**: a genuine PQ-secure TRANSPORT channel distinct from the decision's own
signature tier. `SubmitDecisionWithAssurance` takes `decisionSigTier` as a caller-supplied string
parameter; it does not itself establish or verify an actual PQ/hybrid transport channel (e.g. an
ML-KEM768-wrapped connection) separately from the decision's signature. The Guide's P1/P4
distinction specifically turns on separating transport-level migration from decision-level policy
requirements; this run demonstrates the decision-level claim-gating correctly resists the
confusion in the abstract (exactly what Corollary cor:pq-scope formalizes), but does not run the
transaction over an actual distinct PQ transport channel to show the distinction operationally.
**P4 was not run**, for the same reason: it differs from P0 only in the Guide's "transport" column,
which this entry point does not model as a separate parameter -- running it would currently
produce output identical to P0's, which would not be a meaningful test. Closing this fully (a real
PQ-transport-vs-decision-signature distinction, operationally) remains open work; disclosed here
rather than silently treated as done.

## Files

- `p1_profile_sweep.py` (invocation script, reused the project's established `peer chaincode
  invoke`/`ReadDecision` pattern from `s6_t0_multihop.py` and prior invoke scripts)
- `p1_profile_sweep_result_2026-10-11.json` (full records for all 4 transactions)
