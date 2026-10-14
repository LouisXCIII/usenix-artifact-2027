> **Re-run on the cleaned chaincode package, 2026-09-29** (see `s9_dictionary_attack_results_2026-09-29.md` and
> `s7_no_code_destination_protocol_2026-09-28.md` amendment A1): this file's original run used `authcc_2.29/seq41`,
> which embedded the raw pinned model weights in the same package as every production entry point (found by an
> independent review). The premise "no code, no features, no model weights" did not hold for the peer's installed
> package during the run below, only for the ledger data L4 covers. **Full re-run on `authcc_2.30/seq42`** (weights
> file and its two dependent source files confirmed absent from the installed package):
> `s8_population/s8_rerun_v230.py`/`s8_rerun_v230.json`/`s8_rerun_v230_blocks.json`/`s8_rerun_v230_summary.json`.
> Result: **identical to the numbers below in every respect** -- L1 173/173 committed classes match (113 SAFE, 60
> REJECT); L2 all five negative categories and replay at 0/113 accepted except N3 (3/113 accepted, the same three
> no-op controls already explained below, unchanged since the partner pairing is deterministic); L3 B0 226/226
> accepted; L4 searched 243 new blocks, **0 hits for x, 0 hits for W** (this re-run additionally searched for the
> model weights, which the original L4 did not). The B1Store positive control was not repeated (removed with the
> fix); the search methodology's validity carries over from this file's own positive controls (block 3510, below).
> These are now the premise-clean, authoritative numbers for the population-scale no-code destination result.

# S8 results: S7 at population scale, digest confirmation surface, scanner pass-through (2026-09-28)

Protocol: `s8_population_protocol_2026-09-28.md` (frozen before the first proof or transaction; amendments A1 and A2 are dated after seeing data and explained there).
Files: `s8_population/` (`s8_prepare.py`, `s8_make_fixtures.py`, `s8_register_besu.js`, `s8_fetch.js`, `s8_run.py`, `s8_rerun_n3.py`, `s8_confirm.py`, outputs `*.json`, logs, `blocks_tmp/` = the 244 raw blocks searched).
Destination: the DEPLOYED, UNMODIFIED `SubmitDecisionTrustlessV7`; no x, no weights, no code sent. Fabric height 3466 -> 3710; Besu header at block 541661; run stamp in `s8_run.json`.
Real work: 167 new Groth16 v7 proofs (107 strict, 60 relaxed; all 167 verify with snarkjs), 167 registrations on Besu (record/approve x2/commit; on-chain state matched expectation for all 167).

## P. Scanner pass-through (offline; strict policy, corpora known to be vulnerable)
| Corpus | passed / n | rate | 95% CP |
|---|---|---|---|
| SmartBugs-curated (each contract annotated vulnerable) | 12 / 142 | 8.5% | 4.4%-14.3% |
| weakaccess (built to have weak access control) | 3 / 111 | 2.7% | 0.6%-7.7% |
| pooled | 15 / 253 | 5.9% | 3.4%-9.6% |
| test22 by file name | 3/10 "vuln" and 3/10 "safe" pass | -- | -- |
Reading: the destination inherits the scanner's misses. Caveats: possible training overlap; "vulnerable" is the dataset's annotation; test22's strict pass rate is the same for names marked vuln and safe.

## L. Population correctness and controls (no x, no code)
- L1: 173/173 committed classes equal the offline prediction (113 SAFE, 60 REJECT; 107 new strictly satisfiable + 60 sampled non-satisfiable + 6 satisfiable S4 files). Median submit-to-commit 2,097 ms.
- L2, n = 113 satisfiable contracts per category, one real invoke each, no valid control accepted; bound for 0/113 = 2.6% (one-sided 95% CP, per category, not pooled):
  N1 proof substitution 0/113; N2 tampered proof 0/113; N3 forged featureHash 0/113 valid forgeries (see A2: 3 of the first 113 were accepted because the partner had the SAME featureHash, so the "forged" signal was
  unchanged and the control a no-op; re-run with a partner whose featureHash differs: 3/3 REJECT); N4 wrong feature-commitment proof 0/113; N5 amount 600,000 > threshold 500,000: 0/113; N6 replay: 113/113 refused at endorsement, records unchanged.
- L3: the proof-only baseline `SubmitDecisionB0` accepted 226/226 (113 proof substitutions, 113 amount widenings).
- L4: 244 Fabric blocks searched (x: first 16 quantised features of each of the 173 contracts of the run; W: first 8 entries). Positive controls: 3 deliberate `SubmitDecisionB1Store` calls (03_dos_safe, c377, c617) share block 3510; the search finds their x
  there (and c616, whose x is identical to c617's). **The other 243 blocks: 0 hits for x and 0 for W.**

## D. Digest confirmation surface (offline, real scanner; 100 random contracts, 4 cosmetic edits)
- Determinism control: unedited sources give identical x in 100/100.
- After ANY of the four edits the quantised x differs (identical x: 0/79 for strip-comments (79 of the 100 have comments), 0/100 whitespace normalisation, 0/89 contract-identifier rename, 0/100 appended comment); 120-132 of the 256 features change (median);
  the SHA-256 source hash differs every time. So the featureHash/codeHash confirm an unedited copy, not a cosmetic variant.
- But distinct sources can share x: 61 of the 643 corpus contracts (9.5%) have the same quantised x (hence the same featureHash) as at least one other contract (25 groups, largest 5; 47 in the e7 set, whose members are identifier-renamed variants; c616/c617 differ by `==0` versus `!=2`).
  So the featureHash is not a unique fingerprint of a source.
- Scanner robustness (amendment A1, added after seeing that every edit changes x): of the 88 sampled contracts that fail the strict policy, 9 (10.2%, 95% CP 4.8%-18.5%) have at least one cosmetic variant that PASSES it (strip comments 7, whitespace 6, rename 0, append 0); 1 of 12 passing contracts (c100) fails under some variant (rename and append).
  A source-side developer can therefore change the scanner's decision by cosmetic edits; ADP transports the decision of whatever text is submitted.

## What this does NOT show
- Still a simulation: the code is not sent, not a real developer or second deployment; the corpus is our own and is not production traffic.
- The confirmation-surface test covers 4 cosmetic edits on 100 contracts, not an adaptive attacker; it does not measure what can be inferred from x-digests beyond confirmation of a candidate.
- Stale headers, several administrative domains, latency: not tested. The scanner's training data may overlap the vulnerable corpora used in P.
