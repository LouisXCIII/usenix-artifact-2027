# E8, full value sweeps for A6 and A2's extended families (2026-09-24)

Per the user's explicit request to sweep value ranges for the cases already
done (`evidence/e8_a6_and_a2_families_2026-09-24.md`, which tested one
representative point per case) and go deeper still. All 32 transactions
below are real, on-chain, against the already-deployed chaincode
v2.10/seq22 -- no redeploy needed, only new invocations.

## A6: the full signature-tier x destination-requirement truth table (8 real transactions)

Rather than one laundering case (classical/PQ-required), all 3x2=6
combinations of `decisionSigTier` x `destAssuranceRequirement` were run
against the real `SubmitDecisionWithAssurance`, reproducing the *entire*
`_claims()` truth table on-chain:

| Signature tier | Requirement | Real result | Correct? |
|---|---|---|---|
| classical | ClassicalDecisionAuth | EXACT (accepted) | Yes |
| classical | PQDecisionAuth | **REJECT** | Yes |
| hybrid | ClassicalDecisionAuth | EXACT | Yes |
| hybrid | PQDecisionAuth | EXACT | Yes |
| pq | ClassicalDecisionAuth | **REJECT** | Yes |
| pq | PQDecisionAuth | EXACT | Yes |

**6/6 correct** -- this is a second, independent laundering direction beyond
the one tested previously: a `pq`-tier signature does **not** automatically
satisfy a `ClassicalDecisionAuth` requirement either (an "overqualified"
signature is not implicitly compatible with every weaker requirement --
the destination's actual required claim must be explicitly among the
tier's granted claims, matching `_claims()` exactly, not a simple ordering
assumption). The ablated twin (`SubmitDecisionWithAssuranceNoCheck`) was
run for both real-REJECT ("laundering") cells: **2/2 incorrectly accepted**
(`EXACT`), confirming both laundering directions are caught by the real
check and missed by the ablated one.

## A2-action: full sweep, actionLevel 0-4 (cap=2) (10 real transactions)

| Level | Real | Ablated |
|---:|---|---|
| 0 | EXACT | EXACT |
| 1 | EXACT | EXACT |
| 2 (at cap) | EXACT | EXACT |
| 3 (cap+1) | **REJECT** | EXACT (incorrect) |
| 4 (cap+2) | **REJECT** | EXACT (incorrect) |

**10/10 correct.** Sweeping two points past the cap (not just one) confirms
the real check's rejection is not an isolated boundary artifact -- every
level above the cap is rejected, and the ablated twin accepts every level
regardless, including levels 3 and 4 alike.

## A2-resource: full sweep, resourceID in {0,1,2,3,99} (10 real transactions)

| Resource ID | Real | Ablated |
|---:|---|---|
| 0 | **REJECT** | EXACT (incorrect) |
| 1 | **REJECT** | EXACT (incorrect) |
| 2 (pinned) | EXACT | EXACT |
| 3 | **REJECT** | EXACT (incorrect) |
| 99 | **REJECT** | EXACT (incorrect) |

**10/10 correct.** Since the resource family's real check is an *equality*
test (not an inequality/threshold), a single wrong value was not sufficient
to rule out an accidental off-by-one or a second silently-accepted value;
sweeping four distinct wrong IDs (0, 1, 3, 99 -- spanning small, adjacent,
and far-out values) confirms only the pinned value 2 is ever accepted, and
the ablated twin accepts every one of them.

## A2-context: full 2x2 (4 real transactions)

| contextOK | Real | Ablated |
|---|---|---|
| true | EXACT | EXACT |
| false | **REJECT** | EXACT (incorrect) |

**4/4 correct.** Context is boolean, so this is definitionally the complete
value space (not a sample of it) -- both the true and false cases are now
directly confirmed for both the real and ablated entry points, completing
the matrix that the previous pass had only partially covered (3 of 4 cells).

## Totals

**32/32 real on-chain transactions correct**, all against the
already-deployed chaincode v2.10/seq22 (no new infra issues this pass --
the build fix from the previous two deployments held). Combined with
`evidence/e8_a2_real_ablation_2026-09-23.md` and
`evidence/e8_a6_and_a2_families_2026-09-24.md`, this project now has
**49 real on-chain transactions** demonstrating A2 (all 4 families) and A6
specifically. Raw output: `evidence/e8_value_sweep_result_2026-09-24.json`.

## Still open, disclosed honestly

- A1, A3-A5, A7-A9 remain entirely formal-only; no real analog attempted.
- A6's sweep varies signature tier and requirement, but still uses one
  fixed (role, amount) = (manager, \$200,000); it does not cross-sweep
  against role/amount boundaries the way E7's boundary sweep did.
- The resource family's sweep (0,1,3,99) is a representative sample of the
  "wrong value" space, not exhaustive (an int64 domain cannot be swept
  exhaustively) -- chosen to span small/adjacent/far-out values rather than
  being arbitrary.
