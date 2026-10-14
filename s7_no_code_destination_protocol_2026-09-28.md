# S7 protocol: a destination that never receives the code or the feature vector x

(Roadmap item "S7", not the attack scenario S7 of the paper's S0-S19 suite.)
Written 2026-09-28 BEFORE any S7 transaction was run. Amendments are appended at the bottom with dates.

## Why
A reviewer pointed out that the only workload evaluated re-evaluates a contract the destination would deploy, so the destination could hold the code and
"not disclosing x" buys nothing there; the scenario in which it does buy something (the destination authorizes an action about an artifact deployed
elsewhere and the developer does not release the code, e.g. a compliance, insurance or credit attestation) was not tested. S7 tests that deployment class
on the real Besu -> Fabric substrate.

## Scenario (what is modeled, and what is not)
- Source: Besu chain (chainId 1337) with the ML scan registry and two FeatureCommitRegistry contracts holding, for each of the 22 S4 files, the scanner's
  pass bit, approvals and feature commitments. Real Groth16 proofs (circuits_v7) exist for the 22 files (S4 fixtures).
- Destination: Fabric channel `authchannel`, chaincode `authcc`, the DEPLOYED and UNMODIFIED entry point `SubmitDecisionTrustlessV7`. The destination
  records an attestation (class SAFE/EXACT/CONDITIONAL/REEVALUATE/REJECT) for a `codeHash` under the destination's own role/amount policy
  (role `manager`, amount = attested coverage/limit, threshold 500,000).
- The destination is given only: decisionID, sourceDomain, role, amount, sourceLimit, codeHash, the Groth16 proof and 7 public signals, one Besu header, the
  Besu registry Merkle-Patricia proofs (ML registry slots and the Poseidon feature commitment). It is never given x (256 features), the model weights, or code.
- Modeled: "the developer does not release code or x" is modeled by not sending them. NOT modeled: a real developer, a real second deployment of the
  contract, any legal or business process. The .sol files are not deployed anywhere. That the destination trusts the source scanner's verdict is the
  same trust assumption as everywhere in the paper (A1-A2); ADP transports source errors and does not correct them.

## Measures (pre-declared; none changed after seeing data)
M1 Correctness without x. All 22 files are submitted as real invokes (endorsed by Org1 and Org2, ordered, committed). Pass: the committed class equals the class
   of the S4 committed B4 run for the same file (6 SAFE, 16 REJECT), 22/22.
M2 Non-receipt of x. (a) Client side: no argument of any treatment transaction contains the JSON of x or of the model weights. (b) Channel side: after the run,
   fetch every Fabric block that could contain a treatment transaction and search the raw block bytes for, per file, the compact JSON of its first 16 quantised
   features (`x_q[0:16]` joined by commas) and for the first 8 entries of the pinned weights. The search is validated by a POSITIVE CONTROL: one deliberate
   `SubmitDecisionB1Store` invoke (which takes x) for one file must be found by the same search in its block. Pass: 0 hits in all treatment blocks and >= 1 hit
   for the control. This shows what a channel member can read in the ledger; it does not show that x cannot be inferred from digests.
M3 Bytes. Per treatment transaction, bytes of all arguments; bytes of the same call in the B1-store form (x instead of proof and signals); difference.
M4 Negative controls with x withheld, on the 6 files whose strict relation is satisfiable (03_dos_safe, 03_dos_vuln, 04_call_to_unknown_safe,
   06_unused_return_safe, 08_timestamp_vuln, 09_mishandled_exception_vuln), one real invoke per (category, file):
   - N1 codeHash argument replaced by another file's codeHash (proof-substitution).
   - N2 proof tampered (last digit of pi_a[0] changed).
   - N3 public signal featureHash (index 6) replaced by another file's.
   - N4 feature-commitment MPT proof of another file with this file's proof.
   - N5 amount 600,000 (> manager threshold 500,000) with a valid proof (destination-policy widening).
   - N6 replay: the same decisionID resubmitted after its accepted commit.
   "Accepted" means the returned/committed class is SAFE or EXACT (or CONDITIONAL); an endorsement error or a REJECT/REEVALUATE is "not accepted".
   Pass: 0 accepted in every category. Bounds: one-sided 95% Clopper-Pearson per category (n = 6 each, so 39.3% for 0/6); they are reported per category,
   not pooled across different mechanisms.
M5 Contrast controls (real invokes, same 6 files): the same N1 and N5 inputs sent to the deployed proof-only baseline `SubmitDecisionB0`
   (no codeHash binding, no role/amount check). Expected by construction: accepted. Purpose: show that in this deployment class the destination-side
   binding and policy checks still decide the outcome. And C3: one `SubmitDecisionB1Store` QUERY with x omitted, to record how the re-evaluation
   baseline behaves when x is not provided (expected: it cannot produce a decision).
M6 Client submit-to-commit wall time of the treatment invokes (informational; the ordering batch timeout dominates).

## Setup
Ledger clock: `SetClock(now+300)` by the Org1 admin (monotonic), then one fresh Besu header and fresh MPT proofs are fetched into
`s7_no_code_destination/besu_state.json` (the S4 file is NOT overwritten). Fabric height is read before and after; blocks in between are fetched with
`peer channel fetch`. The pilot (one query per category on one file) runs first to check that none of the negative inputs crashes the chaincode process.

## Not claimed
- That x cannot be inferred from what the destination receives (public codeHash and featureHash allow a dictionary check of a candidate source; S5 quantifies
  what x reveals, not what the digests reveal).
- That any real developer would release proofs but not code; that the destination's compliance decision has legal meaning.
- Anything about freshness (a stale header is not tested here: the ledger clock is monotonic and cannot be moved forward without affecting every other test).
- Deployment with several administrative domains; one host, one channel.

## Pre-declared reading of outcomes
Reported whichever way they fall. Any accepted negative control, any class mismatch in M1, or any hit for x in a treatment block is a failure and is reported
as such; a missing positive-control hit means the block search is invalid and M2 is reported as inconclusive.

## Amendments
- A1 (2026-09-29, AFTER the original run, dated after seeing data): the original run's premise ("the destination is given
  no code, no features and no model weights") was violated at the packaging level, not the ledger-data level: an
  independent review (S9 Measure C, evidence/s9_dictionary_attack_results_2026-09-29.md) found that the installed
  chaincode package at the time of the original run (authcc_2.29/seq41) embedded the raw pinned model weights
  (`b1_model_weights.json`, via `//go:embed` in `b1fair.go`, added for the unrelated S4 same-relation baseline) in the
  same Go package as every production entry point, so Fabric shipped the raw weights to every installing peer
  administrator regardless of which entry points that peer's organisation called. This was verified live (package hash
  matched `queryinstalled`/`querycommitted` on the real channel), then fixed: the four B1-only files
  (`b1fair.go`, `poseidon_opt.go`, `s4_b1_entrypoints.go`, `b1_model_weights.json`) were removed from the package,
  which was rebuilt and redeployed as `authcc` v2.30/seq42. M1 (22 treatment invokes), M4 (30 negative controls + 6
  replay attempts), and M5 (12 B0 contrasts) were re-run in full on this cleaned package
  (`s7_rerun_v230.py`, `s7_rerun_v230_summary.json`); the block-content search (M2b) was also re-run on this run's own
  64 new blocks. The B1Store positive control could not be repeated (that entry point no longer exists in this
  package); the search script's own correctness was already established by the original run's positive control
  (block 3422, `s7_run.json`), which validates the search logic, not the chaincode version, so it was not repeated.
  Results below are from this re-run, on a package independently verified to contain no code, no features, and no
  model weights.
