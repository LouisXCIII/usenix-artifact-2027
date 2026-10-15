# Systematic mutation testing extension of classify() (2026-09-29)

Closes the "mutation-testing extension" open item from review-4/5. Extends the
existing hand-picked 4-defect mutation-testing pass (`e1_mutation_check.py`,
already in the paper as the H1 evidence) to a full, mechanical operator-mutation
sweep of the same real classifier function (`e1_classifier.py`'s `classify()`,
the standalone reimplementation scored against the formal benchmark).

## Method (fully mechanical, no AI-generated mutants or judgment)
`evidence/systematic_mutation_2026-09-29.py` parses `classify()`'s AST and
enumerates every node eligible for a standard textbook mutation operator:
comparison-operator flips (`<`<->`>=`, `<=`<->`>`, `==`<->`!=`, `in`<->`not in`),
boolean-operator flips (`and`<->`or`), and unary-`not` removal. Each mutant
flips exactly one such node, is executed against the same 2000 generated
cases (`e1_generator.generate_cases`, seed 20260923) and the same independent
ground-truth oracle (`e1_ground_truth.ground_truth_class`) used by the main
formal benchmark. A mutant is "killed" if it disagrees with ground truth on at
least one case (nonzero mismatches/FAR/FRR) -- the same detection convention
`e1_mutation_check.py` already established for its 4 hand-picked bugs.

## Result
38 mutation sites found in `classify()`; all 38 mutants killed (mutation score
100.0%, 0 survivors). Per-mutant mismatch counts range 8-1018 out of 2000
cases (well above single digits for almost all, i.e. most mutations are not
near-misses). Full per-mutant detail: `systematic_mutation_result_2026-09-29.json`.

## What this adds over the existing 4-defect pass
The existing pass hand-picked 4 defects chosen to look like plausible real
bugs (skip a specific named check). This pass instead enumerates *every*
comparison/boolean-operator mutation site mechanically, without picking which
ones "should" be interesting -- a stronger, harder-to-game form of the same
independence argument (Section "Threats to Validity": mutation testing is
what supplies independent evidence, since the classifier and the Z3
ground-truth oracle share the same specification).

## What this does NOT show
- Only `classify()` (the role/amount-threshold fragment) was mutated;
  `e1_ground_truth.py` and `e1_oracle.py` (the ground-truth/Z3 side) were not
  mutated, since the point is to test whether ground truth catches classifier
  defects, not the reverse.
- Constant-value mutations (e.g. off-by-one on `ROLE_THRESHOLD`,
  `DEST_ACTION_CAP`) and higher-arity multi-node mutants were not attempted;
  this is single-operator mutation only, the standard baseline form.
- This is the Python formal-benchmark classifier, not the deployed Go
  chaincode; the existing cross-implementation checks (independent
  `snarkjs` Groth16 re-verification, dual liboqs/circl PQ oracles) remain the
  independence evidence for the deployed code path.
