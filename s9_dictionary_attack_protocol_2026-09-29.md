# S9 protocol: a real dictionary-attack success rate against the live registry, a side-channel check, and an independent red-team pass

Written 2026-09-29 BEFORE any query or analysis was run. Amendments are appended at the bottom with dates.

## Why (the gap this closes)
S5/S8 showed that the published digests (codeHash, featureHash) confirm only an unedited exact copy, and that raw x/W do not appear in the Fabric ledger bytes. Neither measured what an attacker with a realistic candidate pool actually achieves: (a) how often codeHash alone uniquely identifies the true source with zero ambiguity, versus how often featureHash alone puts an attacker into a same-x collision group with more than one real candidate (a quantified identification-precision number, not just a collision count); (b) whether anything OTHER than x/W leaks through the real transaction metadata already collected (argument byte sizes, gas, wall time) that could narrow down x indirectly; (c) an independent read-only review of the chaincode and protocol by an agent not involved in designing S7/S8, specifically tasked with finding a leakage or bypass path the author did not consider.

No AI-generated adversarial source code is used anywhere in this experiment (mechanical, deterministic transforms and real registry queries only), consistent with the user's standing instruction against AI participation in adversarial content generation.

## Population and prior state (reused, not regenerated)
- The 643-contract corpus and its `x_q` (`evidence/s5/s5_corpus_features.json`).
- The 189 real codeHash/featureHash commitments already on the live Besu ML registry (22 from S4 + 167 from S8; `evidence/s4_fair_b1/besu_registration.json`, `evidence/s8_population/s8_registration.json`). No new registrations are made.
- The Fabric argument-byte and timing records already collected in S7 (`evidence/s7_no_code_destination/s7_run.json`) and S8 (`evidence/s8_population/s8_run.json`).

## Measure A: dictionary-attack identification precision (offline hash computation + real read-only on-chain queries)
A1 (offline). For all 643 corpus contracts, compute pairwise equality of (i) codeHash (SHA-256 of raw source, as the chaincode defines it) and (ii) featureHash (Poseidon of quantised x, as S4/S8 define it). Report: the number of codeHash collision groups (expected 0, since SHA-256 over distinct byte strings) and the number of featureHash collision groups (already known from S8: 25 groups, 61 contracts) with their sizes.
A2 (offline). For each of the 189 really-committed decisions, define the attacker's task as: given only the on-chain featureHash (not the codeHash, which is a stronger and separate identifier), and a candidate pool of the full 643-contract corpus, how many candidates share that featureHash? Report the distribution of group sizes for the 189 committed decisions specifically (not all 643, since only committed ones are attackable at all), and the resulting identification precision 1/(group size) averaged and worst-case.
A3 (real, read-only on-chain queries; no state change). For a random sample of 50 of the 643 corpus contracts (seed 20260929) NOT among the 189 already committed, compute their codeHash and featureHash and query the live `RegistryContract.mlScanPassed`/`FeatureCommitRegistry.featureHash` view functions on the real Besu chain to confirm they return false/zero (i.e., an attacker scanning uncommitted candidates gets no false hit from the live contract state). This is a real dictionary-attack simulation against the live registry, using only `eth_call` (no transaction, no gas, no state change).

## Measure B: side-channel check on already-collected real transaction data (no new transactions)
For the S7 (n=22) and S8 (n=173) treatment calls, using the already-recorded `argBytes`/timing records: test whether the total argument byte size or the client wall-time of a treatment call correlates with any property of that file's x (its class, its margin, or an index into the corpus) in a way that could let an observer narrow down x without seeing it. Report the correlation (Spearman) between argument bytes and (i) file's real Solidity source byte length, (ii) predicted margin. A near-zero or a fully explained (proof/header/MPT-proof size only) correlation supports "no side-channel found in what we measured"; a non-trivial residual correlation is reported as a finding, not hidden.

## Measure C: independent read-only red-team pass
A fresh general-purpose agent, given only the S7/S8 protocol files and the chaincode source (not this file, so it does not see the measures above stated first), is asked to identify any way the destination, a channel member, or an external observer could learn something about x, W, or the source code beyond what S5/S7/S8 already disclose, given the deployed chaincode and the real on-chain data structures. It reports findings as needs-validation leads (source-grounded, not executed), the same standard used by the earlier security-audit pass in this project. Its report is included verbatim in the results file; nothing is filtered out because it is inconvenient.

## What is not claimed
- That codeHash collisions are impossible for SHA-256 in general (only that none exist in this specific 643-contract corpus).
- That the red-team pass is exhaustive; it is one fresh reviewer, source-only, no execution.
- Anything about a real developer's actual toolchain; Measure B/C analyse only the artifacts already produced by this project's own experiments.

## Pre-declared reading
Reported whichever way it falls. A non-trivial Measure B correlation, or any credible Measure C finding, is reported as a limitation, not smoothed over.

## Amendments
(none yet)
