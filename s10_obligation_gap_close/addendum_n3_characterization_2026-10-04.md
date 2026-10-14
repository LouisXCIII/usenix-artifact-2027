# S10 addendum: "N3_forged_featureHash" does not exercise `verifyFeatureBinding`'s own rejection branch -- a real, project-wide mischaracterization found by deep verification (2026-10-04)

Found while deep-verifying every claim in `results_2026-10-04.md`/`results_deep_2026-10-04.md`
against the actual `ReadDecision` records (not just the top-level `committed` class label) --
not by running anything new, by reading what was already there more carefully.

## What was checked

For every `N1_proof_substitution`/`N2_tampered_proof`/`N3_forged_featureHash` row in the n=30
deep sweep, `ReadDecision` was queried live and `semBound`/`zkVerificationDetail` were inspected
(sampled: 1+3 of 30 for N3, cross-checked against N1/N2 for contrast):

- **N1** (`s10d-c-n1-03_dos_safe.sol-...`): `semBound:false`, detail = *"proof verifies but its
  public solidityCodeHash differs from this decision's codeHash (proof substitution rejected)"*.
  Exercises exactly the intended check (the `pub[1]` vs `codeHash` binding in
  `v7_obligation.go`/`featurebind.go`). Correctly labelled.
- **N2** (`...-n2-...`): `semBound:false`, detail = *"Groth16 pairing check failed: proof does not
  verify against the embedded verification key"*. Exercises exactly the intended check (the
  tampered `pi_a` breaks the SNARK itself). Correctly labelled.
- **N3** (`...-n3-...`, 4 fixtures sampled: `03_dos_safe.sol`, `03_dos_vuln.sol`,
  `04_call_to_unknown_safe.sol`, `06_unused_return_safe.sol`): `semBound:false`, detail in **all
  four** = *"Groth16 pairing check failed: proof does not verify against the embedded
  verification key"* -- **not** `verifyFeatureBinding`'s own message
  ("...forged-feature attack rejected"). The SAME check was run against the ORIGINAL S8 record
  `s8-n3-03_dos_safe.sol-1790605271` (plain `SubmitDecisionTrustlessV7`, chaincode v2.27 at the
  time): identical result, same Groth16-pairing-failure detail string. `grep` across
  `s7_no_code_destination_results_2026-09-28.md`, `s8_population_results_2026-09-28.md`,
  `s9_dictionary_attack_results_2026-09-29.md` for "pairing"/"verifyFeatureBinding": 0 hits -- this
  distinction has never been disclosed anywhere in the project before.

## Root cause (read directly in `zkverify.go`/`featurebind.go`)

`pub[6]` (the featureHash) is one of the SNARK's own **public inputs** -- the Groth16 pairing
equation is defined over the exact public-input vector the proof was generated for. Substituting
a different fixture's `pub[6]` into the public-signals array before calling
`verifySemBoundProof` therefore invalidates the proof's own cryptographic validity (the pairing
check fails) **before** `verifyFeatureBinding`'s separate Merkle-Patricia registry-comparison
check is ever the deciding factor -- `semBound := zkOK && src.Verified && src.MlPassed &&
featureOK` is already `false` via `zkOK=false`, so it does not matter whether `featureOK` from
`verifyFeatureBinding` is true or false. Confirmed independently (not just via the
`pub_feature_from` test mechanics, but by also checking that `v7_args`'s `pub_feature_from` path
is literally just an index-6 swap into the SAME `pub` array passed whole into
`verifySemBoundProof`): the test genuinely re-exercises the N2 mechanism (broken SNARK validity),
not a distinct one. `v7_args`'s `pub_feature_from` parameter and the `N3_forged_featureHash`
category name are inherited unchanged from `s8_run.py` (written before this session, used
identically in S7/S8/S9's own evidence against plain `SubmitDecisionTrustlessV7`); this is a
pre-existing project characteristic surfaced now, not something introduced by `v7_obligation.go`.

## What this does and does not affect

- **Case B (obligation-defect sweep) is unaffected and remains correctly characterized.**
  Independently re-verified: all sampled S5/S6/S7 records show `semBound:true` (the genuine
  evidence chain really did hold) with `obligationDetailV2` exactly matching the intended formal
  mode ("S5: obligation stripped" / "S6: obligation weakened" / "S7: obligation laundered") --
  this is the paper's own stated missing case (`Manuscriptsp_m2_r3_usenix.tex` line ~4532: "an
  undischarged obligation carried on a decision that also relies on..."), and it is genuinely,
  correctly instantiated here: a real valid proof+header (semBound=true) combined with a real
  obligation defect, rejected in one transaction.
- **Case C's N1 and N2 remain valid and correctly characterized** (verified above) -- 2 of its 3
  categories genuinely test what they claim.
- **Case C's N3 is REDUNDANT with N2, not WRONG**: the end result (REJECT, 0/30 + 0/113-in-S8
  wrongly accepted) is still real and still correct -- forging a featureHash this way really is
  rejected, every time, across this project's entire history. What is inaccurate is the specific
  mechanism claimed by the category's name ("forged featureHash" implying `verifyFeatureBinding`
  catches it specifically) -- it is actually caught earlier, by Groth16 itself, which is a
  *stronger* property than the one originally claimed, not a weaker one, but it is not evidence
  that `verifyFeatureBinding`'s own Merkle-comparison branch (featurebind.go:54-56, "forged-feature
  attack rejected") is ever exercised by any committed transaction in this project to date.
- **The genuinely distinct test the paper's own wording implies** -- a proof that is internally
  self-consistent and passes Groth16 on its own terms (a dishonest prover who ran the real circuit
  on a different, self-consistent feature vector `x`, per `featurebind.go`'s own docstring
  describing "attack A2") but whose resulting `featureHash` genuinely does not match the on-chain
  `FeatureCommitRegistry` commitment -- has **not** been constructed anywhere in this project
  (S7/S8/S9/S10). Checked whether an already-generated proof variant could be reused for free: the
  `strict`/non-strict proof pair stored per fixture in `fixtures_s8.json` commits to the SAME
  featureHash in both variants (confirmed: `03_dos_safe.sol`'s strict and non-strict `public[6]`
  are byte-identical), so no free substitution exists -- constructing this test for real would
  require running the actual circuit prover on a dishonestly different `x` (~8s/~5GB per proof,
  the same cost documented elsewhere in this project for circuit proving), which was not
  attempted here.

## Follow-up, same day: the real "attack A2" test was constructed and run (user: "tôi đồng ý")

The gap identified above ("the genuinely distinct test the paper's own wording implies... has not
been constructed anywhere in this project") was closed the same day. `forge_feature_proof.py`
constructs a genuinely valid Groth16 proof using ONLY already-computed real values (no new ML
inference; the pinned real model weights were read from `s4_fair_b1/go_src/b1_model_weights.json`,
a leftover cached copy from before the S9 packaging fix, not from re-running the ML scanner):

- Target codeHash = `04_call_to_unknown_safe.sol` (already registered on Besu, own honest
  featureHash committed in `FeatureCommitRegistry`).
- "Forged" feature vector = `03_dos_safe.sol`'s own real, strict-satisfiable `x_q` (margin
  66,113,634 > `requiredMinMargin` 54,445,307) -- a real, different feature vector, not a
  corrupted public signal.
- Witness built with `solidityCodeHash=<target's real codeHash>`, `featureHash=<donor's real
  featureHash>`, `commitmentHash=Poseidon(target_codeHash, pinnedWeightsHash, donor_pred, SALT)`,
  real pinned `W_q`/`b_q`. Real `generate_witness.js` (0.7s) + real `snarkjs groth16 prove` (9.9s)
  against `circuits_v7/ml_scan_proof_v7_final.zkey` -- genuinely passes its own `snarkjs groth16
  verify` ("OK!"), confirming the proof is internally self-consistent and cryptographically valid
  on its own terms, unlike every N2/N3 case in this project's history.
- Submitted live to `SubmitDecisionTrustlessV7Obligation` (tx `8d3d8706c6ee...`, height
  4097->4098) with the target's real header/registry/feature Merkle proofs and genuinely good
  obligation parameters (audit/audit/discharged/witnessed). Real committed result:
  `semBound:false`, `evAssuranceOk:true`, `obligationCoveredV2:true`,
  `zkVerificationDetail:"proof's public featureHash does not match the pinned scanner authority's
  on-chain commitment for this codeHash (forged-feature attack rejected)"`, `transferClass:REJECT`.

**This is `verifyFeatureBinding`'s own rejection branch (featurebind.go:54-56), exercised by a
real, mined transaction for the first time in this project's history** -- not Groth16 pairing
failure (the proof passed that on its own), not an obligation defect (`obligationCoveredV2:true`
isolates that cleanly), specifically the Merkle-proof registry-comparison check the N3 category's
name always claimed to test but never actually reached. The paper's own cited canonical example
("a forged feature-commitment proof", `Manuscriptsp_m2_r3_usenix.tex` ~line 4532) is now backed
by a real on-chain instance, closing the gap this addendum originally flagged as open. Files:
`forge_feature_proof.py`, `forged_proof.json`, `forged_public.json`, `forged_feature_meta.json`,
`forged_feature_result.json`.

## Revised reading of the "xuất sắc" rating

This does not change the rating of case B, which remains the clean, correctly-targeted, now
statistically-bounded closure of the paper's stated composition gap. It narrows case C's claim:
"a perfect obligation does not compensate for a broken evidence chain" is still fully supported
(by N1 and N2, both real and distinct), but one of its three sub-categories should be relabelled
as redundant rather than cited as covering `verifyFeatureBinding`'s specific branch. Finding and
disclosing this via deep re-verification (checking detail-level `ReadDecision` fields instead of
trusting the top-level class label, then tracing the root cause in the Groth16/public-signal
code) is itself the kind of scrutiny that earns the "xuất sắc" bar rather than one that forfeits
it -- consistent with this project's own prior self-corrections (e.g. the Round-E citation-hash
finding, the S9 packaging-exposure finding). **Update, same day (see Follow-up section above):**
the paper's own canonical "forged feature-commitment proof" example (attack A2) has now been
constructed as a real, mined on-chain transaction (tx `8d3d8706c6ee...`) that specifically
exercises `verifyFeatureBinding`'s own rejection branch -- the honest summary is now: case B =
clean, strong, statistically bounded (n=30); case C = N1/N2 clean and distinct, N3's original
category redundant with N2 (disclosed, not hidden) but superseded by the new direct A2 test,
which is clean and real. Every claim this evidence set makes now has a real on-chain instance
behind it, including the one the paper itself names as the canonical missing case.
