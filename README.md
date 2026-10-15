# Artifact: signature-verified block evidence, independent re-implementation, and timestamp proof

This repository holds reproducibility artifacts for a security evaluation of a cross-domain
authorization-decision reuse system, submitted for double-blind peer review. It contains no
author-identifying information by design.

## Contents

- `qd_block_verification_2026-10-13.md` — signature-verified Fabric block evidence: raw signed
  blocks, a standalone Go verifier (public schema only, no project source), and isolated
  negative/tamper tests. See `qd_block_verification_2026-10-13/`.
- `qd3_differential_test_2026-10-13.md` — an independent second implementation of the
  obligation-coverage logic, written from a prose specification only, differential-tested
  against the real system. See `qd3_differential_test_2026-10-13/`.
- `qd_block_verification_2026-10-13/manifest_combined_2026-10-13.sha256(.ots)` — a SHA-256
  manifest of this artifact, anchored via [OpenTimestamps](https://opentimestamps.org) to a real
  Bitcoin block. This proves the artifact existed by a given date without revealing any identity
  (hash-only commitment to public calendar servers) — see `OPENTIMESTAMPS.md` for how to verify.
- `chaincode_source/` — the full chaincode source for every baseline and ablation entry point
  (`chaincode/`, a single Go package), the Fabric chaincode entry point (`main.go`), `go.mod`/
  `go.sum`, the Groth16 verification keys and other fixtures it reads (`chaincode/testdata/`),
  and the standalone MapRec-signing tool used for the Ab7 ablation (`tools/maprec_sign/`).
  `chaincode/b1_model_weights.json` is the baseline-only model-weights file documented
  separately as removed from the deployed package after the S9 finding; it is kept here only to
  reproduce the (non-deployed) same-relation baseline build the paper describes.
- `e1_formal_benchmark_code/` — the generator, independent Z3 oracle, classifier under test, and
  mutation-check scripts behind the paper's formal benchmark (`e1_generator.py`, `e1_oracle.py`,
  `e1_ground_truth.py`, `e1_ground_truth_smt.py`, `e1_classifier.py`, `e1_run_campaign.py`,
  `e1_run_campaign_smt.py`, `e1_mutation_check.py`). See `e1_formal_benchmark_2026-09-23.md` for
  the protocol and results these scripts produced.

## How to verify

Each subdirectory's own `.md` file documents exact build/run instructions and expected results.
No step requires trusting this repository's author: the block verifier is built from the public
`fabric-protos-go-apiv2` schema only, and the timestamp proof is checkable against the public
Bitcoin blockchain independently of this repository.
