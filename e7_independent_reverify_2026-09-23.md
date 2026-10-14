# E7 deep investigation: independent re-verification of the original 24-case corpus (2026-09-23)

The user asked to investigate E7 more deeply ("điều tra sâu e7") rather than
accept `evidence/e7_boundary_sweep_2026-09-23.md`'s limitation (2) at face
value.

## Correction of a false claim in the previous evidence file

`evidence/e7_boundary_sweep_2026-09-23.md` stated: *"the original corpus's
exact real proof files (`/tmp/corpus_proofs/*.json`) no longer exist
(ephemeral `/tmp` state from the 2026-09-21 session), so independently
re-verifying the original 24 cases is not currently possible."* **This was
wrong.** A deeper search (not limited to `/tmp`) found a permanent,
untouched copy at
`~/pqc-hardhat/experiments/portability/drivers/corpus_v5_proofs/` (24 files
`00.json`-`23.json` + `summary.json`), which exactly matches the original
corpus's `i`/`file`/`ownProof` fields
(`evidence/trustless_corpus_c1_results_2026-09-21.json`) index-for-index.
This correction matters and is recorded here rather than silently fixed.

## What "independent" means here

The original real result relies on the Fabric chaincode's own Go
implementation (`zkverify.go`'s `verifySemBoundProof`) to decide `semBound`.
A ground truth built by trusting that same code's verdict would be
circular. This investigation re-derives the same facts using **four
separate methods, none of which call or reuse the Go verifier**, for all 13
of the corpus's real (non-stub) proofs -- the "own proof" cases where the
scanner's class was `NonVulnerable` with confidence $\geq 0.70$
(`03_dos_vuln.sol` and `08_timestamp_vuln.sol` and
`09_mishandled_exception_vuln.sol` are real scanner **false negatives**:
genuinely vulnerable contracts the ML scanner rated safe with high
confidence -- correctly included as real, not filtered out, since this is
exactly the honest behavior a real scanner-based semBound mechanism has):

1. **Independent Groth16 verification via `snarkjs`** (a different,
   separately-implemented Groth16 verifier from the Go chaincode's),
   against the real `circuits_v5/verification_key_v5.json`: `snarkjs
   groth16 verify` on each of the 13 real proof/public-signal pairs.
   **13/13 `OK`.**
2. **codeHash binding, proof vs. registry**: the codeHash each proof's
   public signal[1] is bound to, compared against
   `stage6_registry_result.json`'s independently recorded `codeHash` for
   the same file/index (two separately-generated artifacts, cross-checked).
   **13/13 match exactly.**
3. **codeHash, independently recomputed from the real source files on
   disk**: found the exact, real definition
   (`code_hash(src) = SHA256(src) mod 2^253`, `x1_attacks.py:37-38`) and
   recomputed it directly from each contract's real `.sol` source text
   (`contracts/SafeBank.sol`, `test_contracts/*.sol`) -- the strongest
   check, since it traces back to the actual source artifact rather than
   cross-referencing two derived JSON files. **13/13 match the proof's
   bound codeHash exactly.**
4. **Destination policy, recomputed from raw public signals**: independently
   checked `public[3] == "0"` (maxVulnClass, i.e. only the safe verdict is
   accepted) and `public[4] >= 7000` (confidence floor, basis points)
   against the real policy constants read directly from `zkverify.go`
   (`requiredMaxVulnClass="0"`, `requiredMinConfidence=7000`). **13/13
   pass.** (All 13 proofs' confidence signal is exactly `7000`, the floor
   itself, not the contract's individual scanner confidence -- by design:
   the circuit proves "confidence exceeds the floor" and only the floor is
   public, per `zkverify.go`'s own comment on replacing a confidence floor
   with an implied margin floor; this is correct circuit behavior, not a
   bug.)
5. **Registry state (`mlScanPassed`/`approvalCount`), live direct RPC
   query** (no Merkle-Patricia proof involved at all, a different
   verification path from the light-client mechanism entirely): called the
   real, deployed `RegistryContract.mlScanPassed(bytes32)` and
   `.approvalCount(bytes32)` getters live, today, for all 13 codeHashes.
   **13/13 match** `stage6_registry_result.json`'s recorded on-chain values
   exactly -- including reproducing `SafeBank.sol`'s real `approvalCount=1`
   (below `minApprovals=2`), which independently confirms the exact real
   case the paper already cites for the $B_3$-vs-$B_4$ contrast
   (`semBound=true`, `evOK=false`).

## Result

**All four independent checks, across all 13 real proofs, agree with the
original real on-chain result with zero discrepancies.** This closes E7's
limitation (2) for real: `semBound`/`evOK`'s ground truth is not merely "the
chaincode's own real cryptographic checks" (already true, but circular as
evidence) -- it is now independently reproduced end to end, from the actual
source file through the actual proof through the actual live registry
state, by code that shares nothing with the Go chaincode being evaluated.

Raw output: `evidence/e7_independent_reverify_result_2026-09-23.json` (all
13 rows), `evidence/e7_independent_reverify_manifest_2026-09-23.json`.

## Still open, disclosed honestly

- The header/light-client commit-seal verification itself (that the proven
  block was genuinely finalized by 3-of-4 pinned validators) was not
  independently re-implemented this pass -- only the registry *state*
  values were cross-checked, via a live direct call rather than a
  Merkle-Patricia proof replay (a different, not a strictly stronger,
  independence axis).
- The 11 "stub" (substitution-proof) cases are covered by this
  verification only indirectly, through `SafeBank.sol`'s proof (case
  i=1, one of the 13) which they reuse -- the same proof was independently
  re-verified once, not once per reusing case.
- This re-verifies the ORIGINAL 09-21 corpus's cryptographic evidence, not
  the two-proof boundary-sweep run from earlier today
  (`evidence/e7_boundary_sweep_2026-09-23.md`), which used a different
  (v7) proof already exercised live today.
