# S5 results: larger real relation, prover scaling, quantified disclosure (2026-09-28)

Protocol: `s5_deep_dive_protocol_2026-09-28.md` (frozen before measuring; amendments A1-A3 below, A2 and A3 dated after seeing data). Files: `s5/`.
Go code (scratch copy of the chaincode, not deployed): `b1head.go`, `b1head_test.go`, kept in `s5/go_src/`.

## S5-A. Larger real relation (the whole classifier head, circuits_v4)
Relation: 576 -> 512 (LayerNorm+ReLU) -> 256 (LayerNorm+ReLU) -> 12, **429,056 MACs**, 678,462 constraints, 5 public signals. B1-head is an exact fixed-point Go re-execution.
- Parity: the Go logits equal the independent Python fixed-point reference on **23/23** real circuit inputs (all 12 logits each), and the circuit's own witness (running maximum of the logits,
  read from the exported witness) on the **22** inputs for which the circuit is satisfiable (the 23rd is the circuit's own ATTACK input, unsatisfiable).
- Timing, Go, same process, interleaved, 22 inputs x 30 reps, 3 fresh-process replications (median ms, P95): Groth16 verification of the real v4 proof (incl. proof/signals/key decoding)
  **0.735 / 0.737 / 0.737** (P95 1.70); head re-execution incl. decoding x0 **0.206 / 0.209 / 0.209** (P95 0.37-0.38); the same plus a SHA-256 commitment to x0 0.206 / 0.209 / 0.211.
  Re-execution is **3.5x cheaper** than verifying the proof, even for a relation 140 times larger than the last layer.
- Synthetic scaling (LABELLED SYNTHETIC; one dense layer of random small integers, Go loop only), median: 3,072 MACs 0.002 ms; 32,768 0.011; 131,072 0.044; 294,912 0.194; 1,048,576 **0.349**;
  2,621,440 **1.081**; 4,194,304 2.02; 16,777,216 9.19; 67,108,864 35.1 ms (0.33-0.66 ns/MAC). With Groth16 verification at 0.735 ms the break-even lies between 1.05 and 2.62 million MACs;
  linear interpolation between those two measured points gives about 1.9 million MACs (about 4.4 times the head). This replaces the earlier unmeasured 2.6M extrapolation.

## S5-B. Prover cost against circuit size (real circuits, snarkjs, 3 runs each)
| Circuit | Constraints | Proving key | Prove (s) | Peak memory | Public signals |
|---|---|---|---|---|---|
| v3 last layer (light) | 4,652 | 3 MB | 0.5, 0.5, 0.5 | 0.6 GB | 5 |
| v5 last layer + weights digest | 435,604 | 186 MB | 8.0, 7.7, 7.7 | 4.9-5.6 GB | 5 |
| v7 last layer + weights + feature digests | 474,110 | 199 MB | 8.0, 8.0, 8.6 | 5.1-5.6 GB | 7 |
| v4 whole head | 678,462 | 488 MB | 15.6, 13.8, 12.3 | 10.1-10.9 GB | 5 |
All four proofs verify with snarkjs and pass the Go verifier. Witness generation 0.05-0.9 s. About 1.6 constraints per MAC for the head. At the measured break-even (about 1.9M MACs, roughly 3 million
constraints) linear extrapolation gives about 40-50 GB of prover memory and a minute or more of proving, above this machine's 31 GB: **the region where a proof would beat re-execution in destination
time is not provable on the host used here** (extrapolation, not measured). Groth16 verification stays roughly constant (0.735 ms with 5 public signals, 0.85 ms with 7).

## S5-C. Quantified disclosure (does giving the destination x reveal more than ADP already publishes?)
Corpus: 643 unique contracts (real scanner run on each, 0 skipped): SmartBugs-curated (142), 263 unique real deployed sources collected for S1 (384 files before deduplication), S1 pinned sources,
e7 contracts, the WeakAccessMod set and the 22 test contracts; 242 near-duplicate/project groups, the largest 255 contracts (39.7%); GroupKFold(5), so one test fold is that large group.
15 attributes (delegatecall, selfdestruct, tx.origin, assembly, low-level call, block.timestamp, payable, loop, ecrecover, ERC-20-like, inherits, send/transfer, pragma 0.4, two size attributes),
predicted by L2 logistic regression from x (256 values) against verdict-only, corpus-of-origin, verdict+corpus and the prior; ROC AUC with a bootstrap over groups (500 resamples).
1. Unconditional (x vs the verdict ADP already publishes), all 643: x predicts 11 of the 15 attributes with AUC 0.74-0.94 (ERC-20-like 0.94, size 0.91, selfdestruct 0.85, ecrecover 0.85,
   public-function count 0.84, low-level call 0.84, loop 0.83, block.timestamp 0.80, payable 0.80, send/transfer 0.79, inherits 0.74); verdict-only gives 0.26-0.66 (several below 0.5, i.e. unstable
   across groups). Rare or unpredictable attributes (delegatecall 21 positives, tx.origin 12, assembly 64 with AUC 0.50, pragma 0.4 0.68) are not reliably predicted.
2. Controlling for origin (amendment A2): corpus-of-origin alone reaches AUC 0.61-0.93 and explains most of it. Conservative leakage AUC(x) - max(0.5, AUC(verdict+corpus)) has a 95% interval excluding
   zero only for **block.timestamp (+0.155 [0.064, 0.332])**, **loops (+0.085 [0.032, 0.167])** and **size (lines above median +0.100 [0.053, 0.143])**; for pragma 0.4 it is significantly NEGATIVE
   (-0.244 [-0.357, -0.077]; the origin is a better predictor than x), for send/transfer negative with an interval reaching zero, and for the rest the interval includes zero.
3. Within one corpus (amendment A3): SmartBugs only (142 contracts, 99 groups, largest 7%): leakage is significant for 8 of 9 scored attributes, +0.23 to +0.34 AUC (loops +0.27 [0.07, 0.40], low-level call
   +0.23, payable +0.24, ERC-20-like +0.24, inherits +0.34, size +0.34, public-function count +0.28, block.timestamp +0.30; send/transfer +0.16 with an interval including zero). The deployed-sources
   corpus alone is NOT analysable with grouped folds (263 contracts, 6 groups, 94% in one group); its numbers in `s5_disclosure_results_within_deployed.json` are degenerate and are not used.
Reading: x is a rich descriptor of the source contract, so handing it to the destination discloses real structural information beyond the verdict; how much of that is *additional* depends on what the
destination already knows about the source. If it knows the source project, most of it is recoverable anyway and the remaining leakage in our pooled data is modest (three attributes); if it does not
(SmartBugs-like anonymous contracts), the leakage is broad. The public codeHash and featureHash already allow confirming a candidate source by a dictionary attack in both designs, so exact-match
fingerprinting is not counted as extra leakage of x. We do not claim that any leaked attribute is sensitive in a given deployment, or that x can be inverted to source code.
Limits: syntactic attributes chosen by us before the run (protocol); linear probes only (a stronger attacker could do better); one scanner; corpora of very different origin and a giant near-duplicate group;
attribute prevalence is unrelated to any real confidential workload.

## Amendments
- A1 (before results): grouping made concrete (near-duplicate components at Jaccard >= 0.6 over 5-token shingles; S1 projects merged; attributes with < 10 positives/negatives not scored).
- A2 (after run 1): corpus-of-origin and verdict+corpus controls added; conservative leakage defined; run 1 kept as `s5_disclosure_results_run1_noCorpusControl.json`.
- A3 (after run 2): within-corpus analyses added for `deployed` and `smartbugs`; `deployed` turned out not analysable (see above).

## Correction (2026-09-28, after external read of the manuscript)
The verdict-only AUC range quoted in S5-C item 1 ("0.26-0.66") was rounded by hand. From `s5/s5_disclosure_results.json` the minimum over the eleven predicted attributes is selfdestruct 0.2546, i.e. 0.25, and the manuscript table now uses the file's values (loop 0.5046 -> 0.50, selfdestruct 0.2546 -> 0.25). No analysis was re-run.
