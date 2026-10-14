# S5 protocol: larger real relation, prover-cost scaling, and quantified disclosure (deep dive after S4)

Written 2026-09-28 BEFORE any S5 measurement. Amendments are appended below with dates. Follows `s4_fair_b1_protocol_2026-09-28.md`.

## Why
S4 showed that for the last-layer relation (3,072 MACs) re-execution is cheaper than Groth16 verification, and it left three gaps: (1) the break-even was an
unmeasured extrapolation from one small relation; (2) the cost that ADP moves to the prover was measured at one circuit size only; (3) the claimed benefit
"x is not disclosed" was structural, not quantified. S5 closes them with real circuits and a real corpus.

## S5-A. Larger real relation and break-even
- Real relation: `circuits_v4/ml_scan_proof_v4.circom`, the WHOLE classifier head (576->512 LayerNorm+ReLU ->256 LayerNorm+ReLU ->12), about 429,000 MACs, 678,462 constraints,
  4 public inputs. (v4 predates the feature and weights commitments; it is used for relation-size scaling only.)
- B1-head: exact fixed-point re-execution in Go (linear layers, constant rescale with truncation toward zero, LayerNorm with integer mean/variance/square root/division,
  gamma/beta rescale, ReLU, argmax), mirroring the circuit.
- Parity oracle: the circuit's own witness (its `inferenceResult` and last-layer logits, read from the exported witness) on every available real input (A, B, ATTACK, EXPV4_*),
  plus randomly perturbed inputs; B1-head must agree exactly on logits and argmax.
- Timing (same process, same host, interleaved, >= 30 repetitions, CPU governor `performance`): Groth16 verification of a real v4 proof vs B1-head re-execution.
- Synthetic scaling (LABELLED SYNTHETIC): random dense layers of m x n with 3e3 .. 6.4e7 MACs to locate the size where re-execution equals the measured Groth16 verification time.
  Reported as a measurement of the Go loop, not of any deployed model.

## S5-B. Prover-cost scaling
- Real circuits already built in the repository: v3 (4,652 constraints), v5_n160 (273,368), v5 (435,604), v6 (436,454), v7 (474,110), v4 (678,462).
- For each: snarkjs Groth16 prove wall time and peak RSS (>= 3 runs), zkey size, and Go verification time. Circuits without a ready input are listed as not measured.
- Fit prove time against constraints only as a description of these points; no extrapolation beyond the largest measured circuit is presented as measured.

## S5-C. Quantified disclosure
Question: how much does giving the destination x (256 values) reveal about the source contract beyond what ADP already publishes (verdict class, codeHash, digests)?
- Corpus (all sources found on this machine, deduplicated by content hash; near-duplicates grouped): SmartBugs-curated, the real deployed sources collected for the S1 survey,
  the S1 pinned sources, the e7 contracts, the WeakAccessMod set, and the 22 test contracts. Real scanner run to capture x for every contract with >= 1 token.
- Attributes (fixed now, all reported, none dropped after seeing results): uses delegatecall; uses selfdestruct; uses tx.origin; uses inline assembly; has a low-level call
  (`.call(`/`.call{`); uses block.timestamp; has a payable function; has a for/while loop; uses ecrecover; is an ERC-20-like token (transfer + balanceOf); inherits another contract;
  uses `send`/`transfer`; pragma is 0.4.x; and two size attributes: line count above the corpus median, external/public function count above the corpus median.
- Predictors: (i) x (256 standardised values) with L2 logistic regression; (ii) verdict only (the scanner's predicted class, one-hot); (iii) prior only (AUC 0.5).
  ADP discloses (ii) plus digests. Metric: ROC AUC on held-out groups (GroupKFold, 5 folds; groups = near-duplicate clusters and source project), with a bootstrap 95% CI over groups.
  Leakage of x beyond the verdict for an attribute = AUC(x) - AUC(verdict). Regularisation strength fixed in advance (C=1.0), no tuning on the test folds.
- Also reported: exact-match fingerprinting (x is a deterministic function of the source, so a candidate source can be confirmed against a disclosed x; the same is true of the public
  codeHash and of featureHash, so this is NOT claimed as an extra leak of x).
- Not claimed: that the leaked attributes are sensitive in any particular deployment, or that x can be inverted to source code.

## Pre-declared reading
Reported however it falls. If x reveals no more than the verdict for most attributes, the non-disclosure argument for ADP is weak for this workload and the paper must say so.
If break-even is far above any relation we can build a proof for, the paper must say that proofs pay off only where re-execution is impossible or the model is not shared.

## Amendments
(none yet)

- A1 (2026-09-28, before any S5-C result): grouping made concrete. Groups = connected components of near-duplicate edges (Jaccard >= 0.6 over 5-token shingles, whitespace-
  normalised, comments stripped); for the S1 deployed sources the project directory is ALSO merged into the group (same project = same group). SmartBugs contracts are grouped only
  by near-duplicate clusters (their directories are vulnerability categories, not projects). Attributes with fewer than 10 positives or 10 negatives are listed but not scored.
  Folds that end up highly unbalanced because of giant components are reported as such.

- A2 (2026-09-28, AFTER the first S5-C run, dated after seeing data): the first run compared x only with the verdict and with the prior. Several verdict-only AUCs came out BELOW 0.5
  (the verdict's association with an attribute is unstable across held-out groups), which makes "AUC(x) - AUC(verdict)" overstate the leakage, and x could be a proxy for the corpus a
  contract came from (e.g. old-pragma contracts come from SmartBugs). Added controls: (iv) corpus-of-origin one-hot alone and (v) verdict + corpus. The conservative leakage is now
  AUC(x) - max(0.5, AUC(verdict + corpus)). The first run is kept as `s5_disclosure_results_run1_noCorpusControl.json`. The attribute list, folds, groups and C are unchanged.

- A3 (2026-09-28, after run 2): with the corpus control most of the apparent leakage disappears, but the one-hot corpus control is itself debatable (an attacker may not know the origin;
  the destination does know the source domain). Added a within-corpus analysis (the corpus is then constant): the same probes restricted to the `deployed` corpus (real deployed contracts, the
  most realistic one) and to `smartbugs`. Both readings are reported: unconditional (x vs verdict) and conditional on origin (within-corpus and the conservative measure). No change to attributes, folds, C.
