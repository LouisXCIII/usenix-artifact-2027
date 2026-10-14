# E7 limitations, resolved where resolvable (2026-09-23)

Continuing the "why + fix or explain" pass (E1, E3, E4, E5, E6), applied to
E7's disclosed limitations (`evidence/e7_b0b4_rerun_2026-09-23.md`, "Honest
scope notes").

## (1)+(3) n=24, reuses one existing corpus without extending it -- FIXED (real boundary-precision extension, +10 new on-chain transactions)

**Why it existed**: the original 24-case corpus used one largely arbitrary
amount per contract file (whatever `stage4_results.json` happened to assign)
-- it never specifically tested a role's exact threshold boundary, so B0-B4
agreement/disagreement at the boundary itself was never directly observed.

**Fix (real, on-chain, not simulated)**: submitted 10 new real
`SubmitDecisionTrustless` transactions sweeping the exact threshold boundary
for all three real roles (`manager`=500,000, `director`=2,000,000,
`clerk`=10,000: below/at/above for each), plus one proof-substitution
negative control at a boundary amount.

**A real, useful complication surfaced during this fix, investigated rather
than routed around**: the first attempt reused `SafeBank.sol`'s real v5
Groth16 proof fixture (`safebank_proof_v5.json`, the same one
`relay_trustless.py`'s substitution cases already used) and got REJECT with
`zk: "v5 proofs are not accepted: in that relation the model hash and the
confidence are prover-declared"` for every single case, including the
honest one. Root-caused by reading `zkverify.go` directly:
`allowLegacyV5 = false` -- the deployed chaincode has been deliberately
hardened since the original corpus was submitted (2026-09-21) to reject the
v5 proof relation entirely (it let the prover privately declare the model
hash and confidence rather than binding them cryptographically -- a real
weakness the project's own `zkverify.go` comments cite attacks A2d/A3
against). This is a genuine, deliberate security improvement made elsewhere
in the project's timeline, not a bug introduced here, and it means
`SafeBank`'s v5 fixture can no longer be resubmitted as-is against the
current chaincode. Pivoted to a real, currently-valid v7 proof instead
(`x8_out/H_honest_own_features_{proof,public}.json`, the same real proof
already used and verified today for the Besu-side X8/X9 corpus), fetching a
fresh Besu header and a fresh registry Merkle-Patricia proof (`eth_getProof`
at slots 3/5 of registry `0x4BDF7d0CC77F907FeDfc1Ab6B39fa9f4336869c7`, the
same registry X14 already used) under the current chain state. A sanity
check (role=manager, amount=200,000, matching the known-good pattern)
confirmed `SAFE`, `semBound=true`, `evAssuranceOk=true` before running the
sweep.

**Real results (10/10 on-chain transactions, all correct)**:

| Role | Amount | Expected | Real result |
|---|---:|---|---|
| manager | 499,999 | SAFE | SAFE |
| manager | 500,000 | SAFE | SAFE |
| manager | 500,001 | REJECT | REJECT |
| director | 1,999,999 | SAFE | SAFE |
| director | 2,000,000 | SAFE | SAFE |
| director | 2,000,001 | REJECT | REJECT |
| clerk | 9,999 | SAFE | SAFE |
| clerk | 10,000 | SAFE | SAFE |
| clerk | 10,001 | REJECT | REJECT |
| manager (proof substituted from a different real contract) | 499,999 | REJECT | REJECT (`semBound=false` regardless of amount) |

**10/10 correct.** The corpus now has real, on-chain-confirmed coverage
exactly at every role threshold's boundary (not just an arbitrary interior
point), and confirms the assurance check (`semBound`) dominates the amount
decision even when the amount alone would otherwise pass -- directly
exercising Theorem T0's ordering. Combined with the original 24, this
project now has **n=34 real on-chain `SubmitDecisionTrustless`
transactions**. Raw output: `evidence/e7_boundary_sweep_result_2026-09-23.json`,
header: `evidence/e7_boundary_sweep_header_2026-09-23.json`.

## (2) Ground truth derived from the corpus's own fields, a different kind than E1's -- UPDATE: this section's claim was wrong; see the corrected, completed fix below

**This section originally said** a fully independent re-verification was
investigated but not completed because "the original corpus's exact real
proof files (`/tmp/corpus_proofs/*.json`) no longer exist." **That claim was
false** -- a deeper search (prompted by the user asking to investigate E7
further) found a permanent, untouched copy at
`~/pqc-hardhat/experiments/portability/drivers/corpus_v5_proofs/`, not
limited to the ephemeral `/tmp` path this file searched. The full
independent re-verification was then built and completed for real; see
`evidence/e7_independent_reverify_2026-09-23.md` for the corrected result
(four independent methods, 13/13 real proofs, zero discrepancies from the
original real on-chain result). This paragraph is left in place, corrected
in-line, rather than silently deleted, so the error and its correction are
both part of the record.

## Still open, disclosed honestly

- The boundary sweep used one real v7 proof (X8's `04_call_to_unknown_safe.sol`
  case) across all three roles' boundaries, not a distinct proof per role;
  this is honest reuse of one real cryptographic artifact for a policy-side
  question (does role/amount classification behave correctly at the
  boundary), not a proof-diversity claim.
- The original 24-case corpus's exact v5 evidence can no longer be
  *resubmitted on-chain* against the current, hardened chaincode (by
  design, `allowLegacyV5=false`); it remains valid as a historical real
  result from when it was genuinely submitted, and -- per the correction
  above -- its cryptographic evidence remains independently re-verifiable
  offline even though it can no longer be resubmitted live.
- See `evidence/e7_independent_reverify_2026-09-23.md` for what remains open
  after that deeper investigation (header/light-client commit-seal
  independence was not re-implemented; the 11 substitution cases are
  covered only via the one proof they reuse).
