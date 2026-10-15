# Artifact: signature-verified block evidence, independent re-implementation, and timestamp proof

This repository holds reproducibility artifacts for a security evaluation of a cross-domain
authorization-decision reuse system, submitted for double-blind peer review. It is maintained to
contain no author-identifying information; file paths and commit metadata are periodically
audited for this.

## Layout

- `evidence/` — every per-experiment evidence file and writeup the paper cites as
  `evidence/<name>` (raw transaction hashes, timing JSON, and markdown protocol/results files for
  every result in the paper, including the items below). This is a flat, dated-filename
  collection, not organized by subfolder beyond the few multi-file packages noted below — search
  by the filename the paper gives rather than expecting a deeper directory tree.
  - `evidence/qd_block_verification_2026-10-13.md` — signature-verified Fabric block evidence: raw
    signed blocks, a standalone Go verifier (public schema only, no project source), and isolated
    negative/tamper tests. See `evidence/qd_block_verification_2026-10-13/`.
  - `evidence/qd3_differential_test_2026-10-13.md` — an independent second implementation of the
    obligation-coverage logic, written from a prose specification only, differential-tested
    against the real system. See `evidence/qd3_differential_test_2026-10-13/`.
  - `evidence/qd_block_verification_2026-10-13/manifest_combined_2026-10-13.sha256(.ots)` — a
    SHA-256 manifest of this artifact, anchored via [OpenTimestamps](https://opentimestamps.org)
    to a real Bitcoin block. This proves the artifact existed by a given date without revealing
    any identity (hash-only commitment to public calendar servers) — see
    `evidence/qd_block_verification_2026-10-13/OPENTIMESTAMPS.md` for how it was produced, and
    "Verifying the manifest" below for how to check it.
- `chaincode_source/` — the full chaincode source for every baseline and ablation entry point
  (`chaincode/`, a single Go package), the Fabric chaincode entry point (`main.go`), `go.mod`/
  `go.sum`, the Groth16 verification keys and other fixtures it reads (`chaincode/testdata/`),
  and the standalone MapRec-signing tool used for the Ab7 ablation (`tools/maprec_sign/`).
  `chaincode/b1_model_weights.json` is the baseline-only model-weights file documented
  separately as removed from the deployed package after the S9 finding; it is kept here only to
  reproduce the (non-deployed) same-relation baseline build the paper describes. Builds clean
  from a fresh checkout (`cd chaincode_source && go build ./...`).
- `e1_formal_benchmark_code/` — the generator, independent Z3 oracle, classifier under test, and
  mutation-check scripts behind the paper's formal benchmark (`e1_generator.py`, `e1_oracle.py`,
  `e1_ground_truth.py`, `e1_ground_truth_smt.py`, `e1_classifier.py`, `e1_run_campaign.py`,
  `e1_run_campaign_smt.py`, `e1_mutation_check.py`). See
  `evidence/e1_formal_benchmark_2026-09-23.md` for the protocol and results these scripts
  produced. (This directory and `chaincode_source/` sit outside `evidence/` because the paper
  does not cite them with an `evidence/` prefix.)

## How to verify

Each subdirectory's own `.md` file documents exact build/run instructions and expected results.
No step requires trusting this repository's author: the block verifier is built from the public
`fabric-protos-go-apiv2` schema only, and the timestamp proof is checkable against the public
Bitcoin blockchain independently of this repository.

### Verifying the manifest

`manifest_combined_2026-10-13.sha256` was generated from two different source directories
(`qd_block_verification_2026-10-13/` and `qd3_differential_test_2026-10-13/`) and its 79 lines
use paths relative to each, not to a single common root — so it is checked in two passes, not
one `sha256sum -c` from a single directory:

```
cd evidence/qd_block_verification_2026-10-13
grep -E '  (blocks|certs|negative_tests|verifier)(/|$)' manifest_combined_2026-10-13.sha256 | sha256sum -c -

cd ../qd3_differential_test_2026-10-13
grep -vE '  (blocks|certs|negative_tests|verifier)(/|$)' ../qd_block_verification_2026-10-13/manifest_combined_2026-10-13.sha256 | sha256sum -c -
```

25 of the 79 lines (`blocks/_orderer_fetched_superseded/*.block`) will report "No such file or
directory": these reference an earlier, orderer-fetched copy of the same 25 blocks, superseded
same-day by the peer-fetched (`qscc`) copies actually shipped in `blocks/` (see
`qd_block_verification_2026-10-13.md`'s "TRANSACTIONS_FILTER" discussion) and kept only on the
authors' own machine, not part of this published package. The manifest was not edited to remove
them, since doing so would change its hash and invalidate the OpenTimestamps proof above. The
remaining 54 lines are expected to all report "OK".
