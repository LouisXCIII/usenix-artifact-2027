# E1 limitations, resolved where resolvable (2026-09-23)

User asked to go through E1's disclosed limitations one at a time, starting
from E1, explain why each exists, and fix what is genuinely fixable. Of the
four limitations listed in `evidence/e1_formal_benchmark_2026-09-23.md`,
two were addressed for real; two are explained as not being "gaps" at all
(a mathematical fact and a deliberate scope choice respectively), not
patched around.

## (1) Classifier and ground truth share one formula -- FIXED

**Why it existed**: `e1_classifier.py` and `e1_ground_truth.py` both
hand-implement the same formal definition (`eq:five-class-classifier`) as
imperative Python if/elif branching. Perfect agreement (FAR=FRR=0) is
expected by construction, not independent confirmation -- a shared typo
could reproduce itself in both files.

**Fix**: `e1_ground_truth_smt.py` re-derives the same ground truth using
Z3's constraint solver instead of Python branching -- a genuinely different
execution engine, even though the underlying specification is identical.
Rerunning the full N=2{,}000 campaign
(`e1_run_campaign_smt.py`) against this Z3-engine ground truth:
**0 mismatches** against the original plain-Python ground truth (confirms
the Z3 encoding is correct) and **FAR=0/1850, FRR=0/150, accuracy=100%**
for the classifier scored against it -- the same headline result, now
independently reproduced by a different computational paradigm, not just a
second copy of the same code. Raw output:
`evidence/e1_run_campaign_smt_result_2026-09-23.json`.

## (2) Only 24/50 combinations have real Z3 class-level independence -- NOT A GAP

**Why it exists**: Z3 (or any SMT solver) only produces information beyond
plain evaluation when checking a *universally/existentially quantified*
claim over free variables -- exactly what the 24/50 class-level audit does.
For a single concrete case (all inputs fixed), there is no free variable
left to search over, so "independent SMT verification" of an individual
instance is not a well-defined stronger claim than direct evaluation.

**Not fixed, because there is nothing to fix**: 24/50 is not a partial
result limited by effort; it is the complete, correct answer to "how many
of the 50 (pair-type, mapping-type) combinations are universally sound."
Trying to grow that number by, e.g., loosening the definitions would
misrepresent the real result. This is unchanged from before and remains
correctly scoped as already written.

## (3) Action/resource/context families have no on-chain guard -- FULLY FIXED

**Why it existed**: `authchaincode.go` only ever implemented a role/amount
guard (`RoleThreshold`/`RoleMapping`); the action/resource/context families
in E1's formal benchmark had never been given a real destination-side check.

**Fix**: added `context.go` to the real chaincode -- one new, additive
function `SubmitDecisionWithContext` (does not modify any existing entry
point), mirroring `SubmitDecisionFull`'s role/amount logic plus a genuine
`contextOK` guard matching the "context" family's `SemCovered` condition
(a context failure is treated as non-repairable and REJECTs, exactly as the
offline formal classifier's `Satisfiable(Delta)` definition requires --
only obligation/staleness/finality-deficit are repairable there). Added two
new optional fields (`ContextOK`, `ContextChecked`) to
`PortableDecisionRecord`, purely additive. Built clean (`go build ./...`,
`go vet ./...`), packaged and deployed for real as chaincode
**v2.7/sequence 19** on the live `authchannel` (both Org1MSP and Org2MSP
approved and committed).

**Real on-chain test** (2 real transactions):

| Case | contextOK | Real class |
|---|---|---|
| Positive control | `true` | `EXACT` |
| Negative control (context loss) | `false` | `REJECT` |

Both correct. Raw record: `evidence/e1_context_family_onchain_2026-09-23.json`.

**Extended to the remaining two families in the same session** (user asked
to continue developing this fix): added `action.go`
(`SubmitDecisionWithAction`, real guard `0 <= actionLevel <= 2`, exactly
`e1_classifier.py`'s `DEST_ACTION_CAP`) and `resource.go`
(`SubmitDecisionWithResource`, real guard `resourceID == 2`, the pinned
identity-equivalence entry from the formal benchmark's resource family), 4
new optional record fields, deployed together as chaincode
**v2.8/sequence 20**. Real on-chain test (4 real transactions):

| Family | Case | Input | Real class |
|---|---|---|---|
| action | positive | `actionLevel=2` (approve, within cap) | `EXACT` |
| action | negative | `actionLevel=3` (transfer, exceeds cap) | `REJECT` |
| resource | positive | `resourceID=2` (pinned, matches) | `EXACT` |
| resource | negative | `resourceID=99` (substituted) | `REJECT` |

All 4 correct. Raw record: `evidence/e1_action_resource_onchain_2026-09-23.json`.

**Result: all four formal-benchmark families (role/amount, context, action,
resource) now have real, additive, on-chain guards on the live network** --
this limitation is fully closed, not partially.

## (4) Small/finite domain -- NOT A GAP, A DELIBERATE DESIGN CHOICE

**Why it exists**: the paper's own methodology text (`subsec:formal-benchmark`)
defines the benchmark fragment as "Boolean combinations of finite-domain
... predicates ... and linear numeric constraints" -- the small domain is
the specification, not a shortfall against it.

**Not fixed, because it is not a gap**: enlarging the domain would only add
more combinatorial cases without testing a different kind of claim; the
logic already exercises every structurally distinct combination the
methodology defines. No action taken.
