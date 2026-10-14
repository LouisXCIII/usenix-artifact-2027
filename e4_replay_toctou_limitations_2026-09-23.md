# E4 limitations, resolved where resolvable (2026-09-23)

Continuing the "why + fix or explain" pass (E1, E3) applied to E4's disclosed
limitations in `subsec:results-replay`
(`Manuscriptsp_m2_r3.tex`, as of the 2026-09-21 concurrency/TOCTOU experiment).

## (1) Replay depth tested only at N=5, guide calls for 1/10/100/1,000 -- FIXED (1/10/100; 1,000 scoped down honestly)

**Why it existed**: the 2026-09-21 experiment used 5 concurrent duplicate
submissions of one decision id, not a genuine depth sweep.

**What was run**: one real `SubmitDecision` (decision id
`e4-replay-depth-1790178856`), followed by 100 **sequential** resubmission
attempts of the *same* id against the live `authcc` network
(`~/authcc-network/fabric-samples/test-network`), then one genuinely new
decision id as a fresh-nonce positive control. This covers the 1/10/100
tiers exactly (attempt 1, attempt 10, attempt 100 all checked); 1,000 was not
run and is disclosed as still open rather than claimed.

**Naive count vs. real mechanism (investigated, not just reported)**: a
first pass counted chaincode-level rejections by grepping CLI output for
`"already recorded (replay rejected)"` and got only **91/100**. Rather than
report that number as-is, the remaining 9 were traced to their actual
on-chain outcome, since 91/100 alone would understate real protection if the
other 9 silently "succeeded":

- All 101 real invocations were captured. 91 were rejected **at endorsement
  time** (the chaincode's `decisionExists` read already saw the committed
  record and returned the explicit error).
- The other 9 raced ahead of the first commit (their endorsement read the
  world state *before* the winning write was committed) and were endorsed as
  if the id were free, then broadcast to the orderer.
- Fetching the real committed blocks from a peer (`qscc GetBlockByNumber`,
  decoded with a from-scratch protobuf walker, since `configtxlator` only
  decodes the orderer's un-annotated copy, not the peer's validated one) and
  reading the `TRANSACTIONS_FILTER` metadata shows: of the 10 transactions
  actually broadcast for this decision id (1 original + 9 raced), **exactly
  1 is `VALID`** and **9 are `MVCC_READ_CONFLICT`**.

So the true end-to-end result is **100/100 replay attempts rejected**, via
two layers of defense in depth: 91 by the chaincode's own idempotency check
at endorsement, 9 by Fabric's ledger-level optimistic concurrency control at
commit -- not a partial 91% result. The fresh-nonce positive control (a
genuinely new decision id submitted immediately after the sweep) committed
`VALID` in its own block (block 334, `TRANSACTIONS_FILTER=[0]`), confirming
replay protection does not spuriously block legitimate new submissions.
Real block evidence: block 333 (10 txs, filter `[0,11,11,11,11,11,11,11,11,11]`),
block 334 (1 tx, filter `[0]`). Raw output:
`evidence/e4_replay_depth_result_2026-09-23.json`,
`evidence/e4_block333_validation_2026-09-23.json`.

## (2) No "non-policy-relevant state control" for TOCTOU -- FIXED

**Why it existed**: the 2026-09-21 TOCTOU experiment only showed the
*positive* case (a concurrent write to the *same* role's threshold correctly
causes `MVCC_READ_CONFLICT`). It never showed the *negative* case -- that a
concurrent write to an *unrelated* piece of state does **not** spuriously
block the decision, which is what "the read-set is minimal/correctly scoped"
actually requires.

**Fix**: fired `SubmitDecisionDynamicPolicy(role="manager", ...)` and
`SetRoleThreshold("director", 9999999)` truly concurrently (same shell,
backgrounded, `wait`) against the live network. `director`'s threshold key
(`thresholdStateKey("director")`) is disjoint from `manager`'s
(`thresholdStateKey("manager")`), which is the only threshold key
`SubmitDecisionDynamicPolicy(role="manager",...)` reads. Both transactions
were ordered into the same block (335) and **both committed `VALID`**
(`TRANSACTIONS_FILTER=[0,0]`) -- confirmed by decoding the real peer-stored
block. Combined with the already-real 2026-09-21 same-key result (concurrent
write to the *same* key correctly conflicts), this closes the gap: the
read-set is neither too narrow (same-key case still conflicts, so the policy
read is not silently ignored) nor too broad (different-key case does not
spuriously conflict, so unrelated policy churn cannot block a legitimate
decision). Raw output: `evidence/e4_toctou_nonpolicy_result_2026-09-23.json`.

## (3) TAC substitution 5-way requirement -- already satisfied (free cross-reference, no new work)

Already noted in `subsec:results-replay`'s existing text: the r3
TAC/context-substitution protocol's five distinct substitution types
(decision, destination, state, mapping, assurance-profile) were tested via
S19 (`evidence/s19_tac_substitution_2026-09-23.md`) against a real,
standalone TAC construction -- all five correctly rejected. No new
experiment needed here; this limitation item was already resolved by S19 and
simply needed the explicit cross-reference retained.

## Still open, disclosed honestly

- The 1,000-deep tier of the replay sweep was not run (only 1/10/100).
- The TAC certificate is not yet wired into either network's real
  transaction-execution path (unchanged from before -- S19 tests the
  construction standalone, not integrated).
