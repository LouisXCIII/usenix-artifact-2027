# S8 protocol: S7 at population scale, with a measured digest confirmation surface and the scanner pass-through rate

Written 2026-09-28 BEFORE any S8 proof was generated or any S8 transaction was run. Amendments are appended at the bottom with dates.

## Why (what S7 left weak)
S7 (22 files, n = 6 per control category) showed the deployed entry point works when the destination receives no code or x, but (a) each control category has a 39.3% bound,
(b) most controls are true by construction, and (c) it did not measure what the published digests reveal or how good the decision the destination inherits is. S8 addresses
(a) by scale, and (b)/(c) by two measurements that can come out badly for the design.

## Population
The 643 unique contracts of the S5 corpus (`evidence/s5/s5_corpus_features.json`: path, corpus, x_q from the real scanner). Strict satisfiability (predicted class 0 and margin >=
the destination floor) is computed offline from x_q and the pinned weights; the replication was checked against the circuit's own witness generator on the 22 S4 files (22/22 agree).
Result of that offline count, fixed before any chain work: 113 of 643 contracts are strictly satisfiable (smartbugs 12/142, deployed 40/263, s1_sources 0/29, e7 52/76, weakaccess 3/111, test22 6/22).

## Measure P: scanner pass-through (offline, no chain)
The destination's decision is only as good as the source scanner. On corpora whose members are known to be vulnerable by construction (SmartBugs-curated: every contract carries an annotated vulnerability;
weakaccess: contracts built to have weak access control), report the fraction the scanner passes at the destination's strict policy, with Clopper-Pearson 95% intervals. Caveats declared now: the
scanner's training data may overlap these corpora; "vulnerable" is the dataset's annotation, not an audit; the per-file `_vuln`/`_safe` names of test22 are reported separately.

## Measure L (population correctness and controls, real Besu -> Fabric, the deployed unmodified `SubmitDecisionTrustlessV7`, no x, no code, no weights sent)
- L1 Correctness: every registered contract is submitted; the committed class must equal the offline prediction (SAFE iff strictly satisfiable and registered as pass, else REJECT). Registered set: all
  107 new strictly satisfiable contracts (strict proofs), 60 non-satisfiable contracts drawn at random with seed 20260928 (relaxed-policy proofs, as in S4), plus the 22 S4 files already registered.
- L2 Negative controls on all 113 strictly satisfiable contracts (107 new + 6 from S4), one real invoke per (category, contract), categories exactly as S7: N1 proof substitution (proof of A with
  codeHash and registry proofs of B, B = the next satisfiable contract in a fixed cyclic order), N2 tampered proof, N3 forged featureHash, N4 feature-commitment proof of another contract, N5 amount
  600,000 above the manager threshold with a valid proof, N6 replay of an accepted decisionID. "Accepted" = committed class SAFE/EXACT/CONDITIONAL. Per-category one-sided 95% Clopper-Pearson bound
  (n = 113 gives 2.6% for 0/113); categories are different mechanisms and are not pooled.
- L3 Contrast: `SubmitDecisionB0` (proof-only) on N1 and N5 for all 113 (expected accepted).
- L4 Channel content: every Fabric block of the run is fetched and its raw bytes searched for the first 16 quantised features of every contract in the run and for the first 8 weight entries;
  three deliberate `SubmitDecisionB1Store` calls (which carry x) are the positive controls and must be found.
- Invokes are issued four at a time; decisionIDs are unique; the ledger clock is set once to now+300 before the header/proof fetch.

## Measure D: digest confirmation surface (offline, real scanner)
The published featureHash/commitment are digests of x, and codeHash is SHA-256 of the source. An observer with a candidate source can confirm it. The question here is how far beyond an exact copy the
confirmation reaches. For a random sample of 100 contracts (seed 20260928) apply four semantics-preserving edits: T1 strip all comments, T2 normalise whitespace (collapse runs of blanks, strip line
indentation, drop blank lines), T3 rename the contract identifier (word-boundary replace of the first declared contract name by a longer one), T4 append a comment and blank lines. Re-run the real scanner
and compare the quantised x with the original: report the fraction with IDENTICAL x (so the same featureHash), and the fraction with an identical codeHash (expected 0 except where the edit is a no-op).
A determinism control re-runs the scanner on the unedited sources (must give identical x). Nothing is claimed about inverting x or a digest.

## What is measured but not claimed
- Population correctness is population-scale on this corpus, not on production traffic; the .sol files are not deployed; "code not sent" is modeled.
- Whether contracts that are not in an attacker's candidate list can be identified from the digests (no: not measured, and not expected).
- Stale headers, several administrative domains, latency.

## Pre-declared reading
All results are reported whichever way they fall. Any accepted negative control, any class mismatch in L1, any x/W hit in a treatment block, or a missing positive-control hit (block search invalid) is a
failure and is reported as such. If T1-T4 leave x unchanged for most contracts, the paper must say the featureHash confirms cosmetic variants, not only exact copies.

## Amendments
(none before the first run.)

- A1 (2026-09-28, AFTER Measure D run 1, dated after seeing data): run 1 showed that NONE of the 100 x 4 edited variants kept x identical (the determinism control T0 was identical for 100/100),
  so the featureHash confirms only unedited copies. That raises a second question the protocol did not ask: does a cosmetic edit change the scanner's strict-satisfiability decision (a developer
  could re-submit cosmetic variants until one passes)? Run 2 of `s8_confirm.py` additionally records pred, margin and strict satisfiability of every variant. Run 1 outputs are kept as
  `s8_confirm_run1.json/.log`; x-identity results are recomputed identically in run 2 (same seed, same scanner).
- A2 (2026-09-28, AFTER the main run, dated after seeing data): in category N3 (forged featureHash) 3 of 113 controls were ACCEPTED. Investigation before any interpretation:
  in all three the partner contract had the identical quantised feature vector, hence the identical featureHash, so the "forged" public signal equalled the original and the control
  was a no-op rather than a forgery (c470/c473 and c500/c502 are identifier-renamed variants of one contract in the e7 set; c616/c617 differ by one comparison `==0` versus `!=2` and have identical x).
  The pre-declared rule (any accepted negative control is a failure) is kept and the 3/113 is reported as recorded. The validity rule "the forged value must differ from the original" was not
  in the protocol; A2 adds it and re-runs those 3 contracts with the next satisfiable partner whose featureHash differs (`s8_rerun_n3.py`). The original records stay in `s8_run.json`.
  N1 and N4 were checked for the same defect (partners with identical codeHash: 0); N2 always changes the proof.
