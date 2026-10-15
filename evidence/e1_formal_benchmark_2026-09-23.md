# E1 — Generated formal benchmark + independent SMT oracle (2026-09-23)

Real, executed result for `Manuscriptsp_m2_r1.tex`'s `subsec:formal-benchmark` /
`subsec:results-formal` (Table `tab:formal-security-results`, macros `\ResFormalCases`
etc.), produced entirely offline (Z3 + pure Python, no blockchain infra required —
this experiment tests the formal classifier definition itself, eq:five-class-classifier,
not the deployed chaincode).

## What was built

- `~/pqc-hardhat/experiments/portability/e1_oracle.py` — independent SMT oracle
  (Z3 5.1.0, via a local venv `.venv_e1`). For each of the 50 `(pair_type x
  mapping_type)` combinations named in the paper's benchmark-generation
  description (5 pair types: equivalent/source-stricter/destination-stricter/
  incomparable/contradictory; 10 mapping types: identity, renaming, role
  refinement/widening, action restriction/expansion, resource substitution,
  context preservation/loss/strengthening), Z3 checks UNSAT of
  `(source predicate AND NOT mapped destination guard)` to decide whether that
  combination is **universally** safe to reuse directly, independent of any
  single instance. Result: **24/50 combinations Z3-verified sound, 26/50 unsound**
  (require per-instance re-verification — this is itself a real finding, not
  a rubber stamp; e.g. `destination_stricter|identity` and `incomparable|role_refinement`
  are unsound in general, while `source_stricter|*` is sound for 9/10 mapping
  types). Full table: `evidence/e1_oracle_table_2026-09-23.json`.
- `~/pqc-hardhat/experiments/portability/e1_generator.py` — generates concrete
  decision-transfer instances stratified over the 50 combinations (40 per
  combination, N=2000 total, seed=20260923), plus independently-drawn
  obligation (empty/audit/dual-approval/notification/combined), temporal
  (valid/not-yet-valid/expired/stale/epoch-mismatched/revoked), finality
  (above/at/below), and assurance-evidence (classical/hybrid/PQ, for decision
  signature and transport separately, encoding the non-amplification property
  of eq:pq-scope-example) dimensions. The role/amount fragment reuses the real
  chaincode's `RoleThreshold` values (`clerk=10000, manager=500000,
  director=2000000`), matching `authchaincode.go` and the real 7-transaction
  case study in `evidence/all_five_transfer_classes_2026-09-21.md`.
- `~/pqc-hardhat/experiments/portability/e1_ground_truth.py` — computes the
  TRUE five-class label per case by direct evaluation of the same deterministic
  destination guards (admission criterion (ii) in the paper's methodology).
- `~/pqc-hardhat/experiments/portability/e1_classifier.py` — the **system
  under test**: a standalone reimplementation of `eq:portable-final` /
  `eq:five-class-classifier`, written as a separate code path from the
  ground-truth module (no shared functions), scored against it.
- `~/pqc-hardhat/experiments/portability/e1_run_campaign.py` — runs the full
  pipeline and computes FAR/FRR (eq:far, eq:frr), five-class accuracy, and the
  full confusion matrix.
- `~/pqc-hardhat/experiments/portability/e1_mutation_check.py` — a validation
  step (not part of the reported numbers): injects 4 known bugs (skip mapping-
  validity check, skip replay check, skip finality check, skip assurance
  non-amplification check) into copies of the classifier and confirms the
  scoring harness detects every one of them (accuracy drops of 1.2–7.6
  percentage points, FAR jumps to 0.7–4.6%). This rules out the concern that a
  0-error result just means the harness can't detect errors.

## Real results (N=2000 generated cases, 50 combinations, seed=20260923)

| Metric | Value |
|---|---|
| Benchmark cases | 2000 |
| Ground-truth class totals | EXACT=13, SAFE=137, CONDITIONAL=95, REEVALUATE=95, REJECT=1660 |
| Unsafe cases (ground truth $\notin$ {Exact,Safe}) | 1850 |
| Safe direct-reuse cases (ground truth $\in$ {Exact,Safe}) | 150 |
| **FAR** (unsafe → {Exact,Safe}) | **0/1850 = 0** |
| **FRR** (safe → Reject) | **0/150 = 0** |
| Five-class accuracy | 2000/2000 = 1.0 |
| Confusion matrix | diagonal only — 0 off-diagonal entries |
| Mutation-test detection rate | 4/4 injected bugs detected (accuracy drop 1.2–7.6 pp) |
| Oracle class-level soundness | 24/50 combinations universally sound (Z3), 26/50 need per-instance check |

Raw output: `evidence/e1_formal_benchmark_results_2026-09-23.json` (includes the
full confusion matrix, per-combination breakdown, and the complete 50-row
oracle table).

## Honest scope notes (do not overclaim beyond this)

- **Why FAR=FRR=0 is expected, not vacuous**: `e1_classifier.py` and
  `e1_ground_truth.py` both implement the same formal definition
  (`eq:portable-final`/`eq:five-class-classifier`) by necessity — they evaluate
  the same deterministic destination guards. This experiment is a **formal
  correctness / regression test of the classifier's control-flow branching**
  (Fatal → Portable → BiRefine/Conditional/Reevaluate precedence) against
  independently-derived ground truth on a large, diverse generated corpus, not
  a test of an unrelated oracle. Its value is demonstrated by the mutation
  test: known bugs in that same branching logic are reliably caught.
- **What is independently verified by Z3, specifically**: the *class-level*
  soundness of each (pair_type, mapping_type) combination (whether the mapping
  rule is safe to trust for ALL instances of its declared predicate class) —
  this is the one place where "independent SMT agreement" (admission criterion
  (iii)) does real, non-circular work, and it produced a genuine, only
  moderately clean result (24/50, not 50/50).
- **What this does NOT cover yet**: this is an offline/logical validation of
  the formal classifier definition, not a further on-chain validation. The
  role/amount fragment specifically is cross-validated against real on-chain
  behavior separately (`evidence/all_five_transfer_classes_2026-09-21.md`, 7
  real transactions, chaincode v1.3/seq4). The action/resource/context
  mapping families modeled here have **no on-chain guard implementation yet**
  in `authchaincode.go` — they exist only in this offline formal model.
- Domain sizes (3 roles, 4 action levels, 5 resource ids, 1 boolean context
  flag) are deliberately small/finite, matching the paper's own description
  ("Boolean combinations of finite-domain ... predicates ... and linear
  numeric constraints"); this is not an exhaustive enumeration of a larger
  real policy language.
