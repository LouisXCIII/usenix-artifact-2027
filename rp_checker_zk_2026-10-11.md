# Tru 4: ZK proof-authenticity ablation (SemBound/feature-binding) -- NOT an R_P instance

> **CORRECTED same day, after external review.** This file's original title and framing
> ("the real non-trivial $R_P$ instance was already deployed") was itself wrong. $\psi_s$ and
> $\gamma_d$ for this fragment are stated over the IDENTICAL public-signal vocabulary the
> circuit outputs -- $M_I$ is the identity map and $\gamma_d\equiv\psi_s$ by construction, the
> same tautology-under-identity category as the obligation fragment in `rp_checker.py`, not a
> translation between two independently-chosen predicate languages. What the Groth16 proof and
> Merkle-Patricia feature-binding actually establish is AUTHENTICITY of $\psi_s$ (that the
> source genuinely computed what it claims) -- a precondition for applying any $R_P$ relation,
> not an instance of one. The "7/7 widening" results below are real and worth keeping, but
> under their correct name: a ZK proof-authenticity ablation (every signal the real destination
> policy checks is load-bearing against a dishonest prover), not an $R_P$ result. **The
> deployed system still has no non-trivial $R_P$ instance.** The manuscript has been corrected
> to match (Appendix A, Table 8, Open Science index). The rest of this file is kept as the
> honest record of what was built and measured; read every "$R_P$ instance" claim below as
> superseded by this banner.

## Correction this file makes

`evidence/rp_checker_2026-10-10.md` and the manuscript's appendix (around the sentence
"Real system has no non-trivial $R_P$ instance yet") stated that the deployed system has no
destination guard whose safety genuinely depends on an unverified source-trust relation, and
that a genuinely non-trivial instance would need "a zero-knowledge-proven private source
predicate the destination must accept via a verified mapping rather than re-derive -- which
the current workload does not yet have."

That statement is **wrong**. It was written after reading only the `RoleThreshold`/amount
guard (`authchaincode.go`, destination-local) and the obligation mechanism-strength mapping
(`obligation_v2.go`, an identity-mapping tautology) -- it did not re-check
`authcc/chaincode/zkverify.go` or `featurebind.go`. Those files implement exactly the
situation the disclosure said was missing, and have since 2026-09-21 (`SubmitDecisionVerified`)
and since the v7/feature-binding work (`SubmitDecisionTrustlessV7`, the production entry point
the paper's main security claims already rest on).

## Why this is a genuine, non-trivial $R_P$ instance (not destination-local, not tautological)

Section~sec:semantics's own $\Phi_s=\{\texttt{mlScanPassed},\texttt{approvalCount}{\geq}k\}$
(line ~531) is established at the source by running a private ML model over private features.
The destination never has those features or weights and, in `SubmitDecisionVerified`/
`SubmitDecisionTrustlessV7`, never re-runs the scan. It only ever sees:

- a Groth16 proof ($\pi_a,\pi_b,\pi_c$) and public signals
  `[scanPassed, solidityCodeHash, commitmentHash, maxVulnClass, minMargin, modelWeightsHash]`
  (v6; v7 appends `featureHash`), and
- (for `SubmitDecisionTrustlessV7`) a Merkle-Patricia proof, under an independently-verified
  Besu header, that the proof's `featureHash` equals
  `FeatureCommitRegistry.featureHash[codeHash]` -- a value one pinned scanner authority
  committed on-chain at scan time, independent of the prover.

and accepts `mlScanPassed`'s stand-in (`semBound`) only if both verify. This is literally "a
guard the destination cannot evaluate from its own state alone... accepted via a verified
mapping rather than re-derived" -- the exact description the earlier disclosure said the
workload lacked.

## Formalization

- $R_\Sigma$: the Groth16 relation itself, for fragment A (binds public signals to the hidden
  witness $x,W,b$); treated as a trusted cryptographic oracle in this Z3 encoding, the same
  abstraction boundary the project already uses for the Coq-mechanization-vs-chaincode gap.
  This module does **not** re-verify the pairing (`zkverify_test.go` already does, in Go,
  against real proof/key files); it checks the policy layer applied to whatever public signals
  a verified proof carries.
- $M_I$: identity -- the public signals are the intent payload that crosses.
- $\psi_s$ (fragment A): scanPassed=1 $\land$ maxVulnClass=0 $\land$ minMargin$\geq$required
  $\land$ modelWeightsHash=pinned -- a genuine safe verdict from the real pinned model with
  sufficient margin.
- $\gamma_d$ (fragment A, real deployed policy, `verifySemBoundProof` lines 236-263 of
  `zkverify.go`, ported faithfully): the same four conjuncts.
- $\gamma_d$ (fragment B, real deployed, `SubmitDecisionTrustlessV7`): fragment A **AND**
  `featureHash` proven equal to the registry's committed value for this exact `codeHash`
  (`verifyFeatureBinding`, `featurebind.go`).

Because $\gamma_d$ is, by construction, exactly what $\psi_s$ demands, `check_entailment`'s
UNSAT result on the real policy (confirmed, `sound=True`) is close to a restatement -- a
self-consistency check on the Z3 transcription, not new information. The new information is in
the **naive variants**.

## Naive variants: 7 of 7 are widening, 2 of them with real measured attack data

| Variant | Mechanism | Result | Evidence |
|---|---|---|---|
| NV1 | Legacy v5 relation (no modelWeightsHash public signal; minConfidence prover-declared, not circuit-bound) | **WIDENING (real)** | `x1_attacks.py`, 2026-09-21: A2 (forged features), A3 (substituted weights), A2d (forged confidence) all produce `valid_proof=true` against the real snarkjs prover + real circuit |
| NV2 | Omit `modelWeightsHash==pinned` check | **WIDENING (symbolic)** | Z3 witness: `modelHashMatchesPinned=False` accepted |
| NV3 | Omit `minMargin>=required` check | **WIDENING (symbolic)** | Z3 witness: below-floor margin accepted |
| NV4 | `maxVulnClass<=1` instead of `==0` (off-by-one) | **WIDENING (symbolic)** | Z3 witness: class-1 verdict accepted |
| NV5 | Margin comparison reversed (`<=` instead of `>=`) | **WIDENING (symbolic)** | Z3 witness: margin direction flipped |
| NV6 | Omit `scanPassed==1` check | **WIDENING (symbolic)** | Z3 witness: scanPassed=False still accepted |
| NV_FB | Omit feature-binding (fragment B); accept on fragment A alone | **WIDENING (symbolic + real)** | Z3: structural (dropping a required AND-conjunct can only widen). Real: `x7_forged_features_corpus_scale_2026-09-21.json` -- **11/11** corpus-scale forged-feature proofs classify SAFE under the real v6 policy, **9 of 11** for contracts whose honest classification is REJECT. `x9_summary.json` -- **11/11** forged v7 proofs are valid under the circuit's own internal self-consistency check alone (the registry comparison is what actually catches them) |

All 7 of 7 mechanically-constructed naive variants of this fragment are widening -- unlike the
`RoleMapping`/obligation fragments in `rp_checker.py` (where 3 of 6 were widening and 3 were
only over-conservative), every single-conjunct omission here is a real safety violation, not
just an availability bug. Two of the seven (NV1, NV_FB) are not hypothetical: they reuse
already-existing real measurements (X1, X7, X9) from dishonest-prover experiments run against
the real snarkjs prover and real circuits earlier in this project, rather than new synthetic
data manufactured for this check.

## What this does and does not establish

- Does not re-verify the Groth16 pairing itself (cryptographic primitive, out of SMT scope,
  same boundary disclosed for every other fragment in this project).
- Does not model the light-client/Merkle-Patricia proof mechanics of fragment B symbolically --
  `feature_hash_matches_registry` is a single Bool standing in for "the MPT proof verifies and
  the value matches," not a re-derivation of MPT/RLP correctness.
- Each family's correspondence between its Z3 encoding and the real Go guard is itself an
  unverified assumption (same category of gap as the Coq-mechanization-vs-chaincode
  correspondence disclosed elsewhere in this project), not something this checker establishes.
- Scope is `SubmitDecisionVerified`/`SubmitDecisionTrustlessV7`'s SemBound-acceptance logic
  only; `EvAssuranceOK`/finality (the light-client header+quorum check) is a separate guard,
  not covered here, and not privacy-sensitive in the same sense (it is about public on-chain
  state authenticity, not a private source predicate).
- This closes exactly the gap the previous round's disclosure named as missing, but it was
  *already deployed* -- "starting Tru 4" here means formalizing and checking an existing
  real workload, not building a brand-new one. No new chaincode, circuit, or infrastructure was
  built for this file; `rp_checker_zk.py` is a pure read of already-deployed, already-tested
  code plus already-measured experiment data.

## Files

- `evidence/rp_checker/rp_checker_zk.py` (checker; canonical copy also at
  `~/pqc-hardhat/experiments/portability/rp_checker_zk.py`)
- `evidence/rp_checker/rp_checker_zk_results_2026-10-11.json` (full output)
- Reused, not modified: `authcc/chaincode/zkverify.go`, `authcc/chaincode/featurebind.go`,
  `experiments/portability/x1_out/x1_summary.json`,
  `evidence/x7_forged_features_corpus_scale_2026-09-21.json`, `evidence/x9_summary.json`
