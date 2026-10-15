# Independent policy oracle: mechanical boundary-value sweep (2026-10-10)

Supersedes the 2026-09-28 S3 result in `evidence/s3_independent/` for manuscript use. The
2026-09-28 run was removed from the paper by the author's own decision ("session 3 very good
but has AI participation") because its test-case generation (`adversary_gen.py`,
`families_adversary_gen.py`) used an AI-authored "blind adversary". This run keeps the same
oracle and harness, written from the specification, but replaces case generation with a fully
deterministic sweep over every combination of a defined set of boundary points — exhaustive
over that defined grid, not exhaustion of the (unbounded) amount domain itself — no
AI-authored adversarial content anywhere in this run.

## What is reused unchanged from 2026-09-28

- `evidence/s3_independent/oracle_cedar.py` — independent oracle, Cedar policies written from
  the specification (not from the Go source).
- `evidence/s3_independent/s3_independent_test.go` — Go harness running the real, unmodified
  `SubmitDecisionFull`/`SubmitDecision` entry points offline against an in-memory ledger stub.
- `evidence/s3_independent/analyze_s3.py` — verdict classifier (match / unsafe_accept /
  over_conservative / class_mismatch) and Clopper-Pearson 95% upper bound.
- `evidence/s3_independent/mutation_check.py` — 11 hand-authored code mutations (M01-M11,
  mechanical, not adversarial reasoning; same methodology class as every other mutation-testing
  pass in this project).

## What is new

- `evidence/s3_mechanical_sweep/cases_mechanical_sweep.py` — the only new file. A deterministic
  generator: every case comes from a fixed formula over a Cartesian product of axes (role,
  amount boundary points derived by formula from each role's cap, the 2x2x2 semBound/
  evAssuranceOK/secondApprovalObtained space, plus a mechanical replay axis). No hand-picked
  "adversarial" case, no narrative rationale field, no randomness. Re-running the generator
  produces byte-identical output.

## SUT verification

Current live source `~/fabric-network/authcc/chaincode/authchaincode.go` (sha256
`17434774a081fdc2bdd61f26ae16d737ad709e5f91ae8db0ea87b60ac2d55369`) differs from the
2026-09-28 snapshot (sha256 `e50ed4b1...`, recorded in `s3_independent/SUT_authchaincode.go.sha256`)
because of unrelated additive work since then (obligation discharge identity check, q_D domain
attestation, etc.). `SubmitDecisionFull`/`SubmitDecision`/`classify()` — the fragment this
oracle models — are structurally unchanged, with one addition: both entry points now reject
`amount <= 0` with an error (hardening that S3's 2026-09-28 note recorded as "documented, NOT
applied" — it is applied now). The oracle does not model this guard, so `amount <= 0` cases
fall outside `in_domain` by the oracle's own definition and are reported separately below, not
folded into the primary claim.

## Generator

560 cases total:

| category | count | axis |
|---|---|---|
| `cap_boundary` | 150 | amount ∈ {cap-1, cap, cap+1, 2×cap} per known role |
| `half_cap_boundary` | 120 | amount ∈ {half-1, half, half+1} per known role |
| `low_amount` | 40 | amount = 1 per known role |
| `non_positive_amount` | 80 | amount ∈ {0, -1} per known role (out-of-domain by design) |
| `malformed_role` | 160 | 2 unrecognised role strings × 10 amount points |
| `replay` | 10 | valid (amount>0, known role) cases forced to reuse an earlier valid case's `decisionId` |

Roles: `manager`, `director`, `clerk` (destination identities, caps 500000/2000000/10000),
`gerente`→`manager`, `gerente_senior`→`director` (source-registry-mapped, non-identity), plus
`""` and `"not_a_role"` (malformed). All flag combinations (semBound × evAssuranceOK ×
secondApprovalObtained) are covered for every role/amount pair: full 2×2×2.

**Replay-axis bug found and fixed during this run, disclosed in full**: the first version of
the generator reassigned `decisionId` on the *last* 10 cases in generation order without
checking their own `amount`/`role` fields. Those cases happened to have `amount<=0` (tail of
the `not_a_role` block), so the `amount<=0` hardening guard returned an error before the
replay-check branch was ever reached — the mutation-testing pass below (M09, replay-check
removed) came back **not detected** (false negative in the *test suite's power*, not evidence
of a SUT defect: the SUT's own replay behavior was never exercised by those particular cases).
Root-caused by hand-mutating a scratch copy and tracing the two replayed cases' actual
`class`/`err` output before accepting the result. Fixed by restricting the replay axis to
cases with `amount>0` and a known role (so a disabled replay-check is observable independent
of the unrelated amount guard); re-run confirmed M09 detected. Kept here as the honest record
of a real test-harness defect found and fixed, same disclosure standard used for the S9/S10/S11b
self-corrections elsewhere in this project.

## Results

**In-domain (`amount>0`, role non-empty), both `SubmitDecisionFull` ("full") and
`SubmitDecision` ("basic")**:

- 360/360 match, **0 unsafe accepts**. No confidence bound is reported: this is a fixed,
  deterministic boundary-value enumeration, not a random sample from a larger population, so a
  Clopper-Pearson (or any) statistical bound would not have a valid interpretation here (see
  "Clopper-Pearson bound removed" below).
- Per-category breakdown (full): `cap_boundary` 150/150, `half_cap_boundary` 120/120,
  `low_amount` 40/40, in-domain `malformed_role` (the `not_a_role` cases with `amount>0`)
  40/40, `replay` 10/10 — all match, 0 exceptions in any category.

**Out-of-domain under the base policy (200 cases: `amount<=0` and/or empty role)** — UPGRADED
from a spot-check to a full, explicit verification (see "Full-domain extension" below):
originally only 10 of these 200 were manually inspected before concluding no unsafe accept was
hidden; `oracle_cedar_full_domain.py` now gives all 200 a real expected-vs-actual verdict by
adding one explicit Cedar `forbid` rule (`context.amount <= 0`) to the same policy set:

- `over_conservative` 20 (full) / 12 (basic): SUT rejects via the hardened `amount<=0` guard
  where the *base* oracle (without the extension), which does not model that guard, would
  otherwise permit — i.e. the SUT is *more* restrictive than the base oracle here, never less.
- `class_mismatch` 140 (full) / 148 (basic): under the base oracle these show as mismatches,
  but under the extended oracle (next section) every one of them is a correct agreement.
- `match` 40 (full) / 40 (basic): empty-role cases where both SUT and the base oracle already
  agree (REEVALUATE-equivalent).

**Full-domain extension (`oracle_cedar_full_domain.py`, new, not modifying `oracle_cedar.py`)**:
adds exactly one Cedar rule, `forbid(principal, action, resource) when { context.amount <= 0 };`,
which in Cedar's semantics overrides any `permit` unconditionally. Re-evaluated all 200
previously-out-of-domain cases against this extended policy and compared to the real SUT
output (a case "denies" if the SUT returned an error OR a non-accepting class — REJECT or
REEVALUATE — not only a hard error, which the first pass of this check conflated and had to be
corrected): **200/200 agreement for `SubmitDecisionFull`, 200/200 for `SubmitDecision`, 0
dangerous mismatches (oracle denies, SUT accepts) in either target.** Combined with the
360/360 base-policy result: **560/560 cases match, 0 unsafe accepts, across the full case set,
with an explicitly defined expected outcome for every single case** (no case excluded from
verification).

**Clopper-Pearson bound removed from all reporting of this sweep**: the 560 cases are a fixed,
deterministic enumeration (a boundary-value partition of the role/amount/flag space), not a
random sample from a larger population, so a CP95 upper bound has no valid statistical
interpretation here (an external review caught this — the original version of this evidence
file and the manuscript both reported one; both have been corrected to report plain match
counts with no confidence-bound claim).

**Hardening timeline, fully answered**: `amount<=0` rejection was added to 6 production entry
points (`SubmitDecision`, `SubmitDecisionFull`, `SubmitDecisionObligationV2`,
`SubmitDecisionAssuranceV2`, `SubmitDecisionTrustlessV7`, `SubmitDecisionTrustlessV7Obligation`)
on **2026-10-06** (chaincode seq44, `evidence/s11_chaincode_seq44_amount_guard_v3_2026-10-06.md`).
The original 2026-09-28 S3 run predates this and explicitly documented `SubmitDecisionFull`
accepting `amount<=0` as EXACT/SAFE (26/26 probes) at that time. This 2026-10-10 run is
entirely post-hardening (today's live chaincode). Ablation/baseline variants
(`SubmitDecisionTrustlessWeakQuorum`, `SubmitDecisionFullNoObligationCheck`, B0/B2/B3) were
deliberately left unguarded to keep earlier measurements comparable, per that same file —
not evaluated by this oracle (out of its stated scope).

**Scope qualifier (previously implicit, now stated explicitly everywhere this result is
cited)**: this oracle covers only `SubmitDecisionFull`/`SubmitDecision`. It does **not** cover
`SubmitDecisionTrustlessV7`, its obligation-verified variants, or any of the six
obligation/assurance/context extensions — the entry points the paper's main on-chain security
claims actually rest on. Independent evidence for those comes from the cross-implementation
checks (Groth16 re-verification, dual liboqs/circl PQ oracles, light-client re-implementation)
described in the Discussion section, not from this check. This oracle also tests a single
fixed destination policy (5 roles, fixed caps) — it is not yet a check of the relation $R_P$
between two independently varying policies (that remains a separate, larger piece of future
work: grammar-based random policy/mapping generation, not done here).

## Mutation testing (power check, same 560-case suite)

11/11 mutants detected (M01-M11), control (M00, unmodified code) correctly shows no detection
(no false alarm):

| Mutation | Detected |
|---|---|
| M01 cap check `>`→`>=` | Yes |
| M02 cap check allows cap+1 | Yes |
| M03 obligation skips upper half | Yes |
| M04 obligation from exactly half | Yes |
| M05 obligation threshold uses a third | Yes |
| M06 second-approval flag ignored | Yes |
| M07 semBound not checked | Yes |
| M08 evAssuranceOK not checked | Yes |
| M09 replay check removed | Yes (after the generator fix above) |
| M10 registry corrupted (gerente→director) | Yes |
| M11 policy data corrupted (clerk cap) | Yes |

## Scope and what this does not show

Same scope restriction as the 2026-09-28 S3 run: only the `SubmitDecisionFull`/
`SubmitDecision` fragment (role/amount/threshold/obligation/replay) is covered. The six-family
extension (`ObligationV2`/`AssuranceV2`/`WithAction`/`WithResource`/`WithContext`/
`WithAssurance`, "S3b" in the old evidence) is NOT redone here — it also used AI-adversary
generation in 2026-09-28 and is out of scope for this pass (see plan file, "Phạm vi KHÔNG làm
trong lần này"). `sourceLimit` is fixed to `amount` and not independently swept, same
restriction as before (the Cedar policy fragment under test does not model it). This oracle
tests agreement on a fixed, hand-transcribed specification — it is independent of the Go
implementation, but it is not independent of the spec itself; a spec-level error would not be
caught by this check (disclosed identically to the removed 2026-09-28 version).

## Files

- `evidence/s3_mechanical_sweep/cases_mechanical_sweep.py` (generator)
- `evidence/s3_mechanical_sweep/cases.json` (560 cases)
- `evidence/s3_mechanical_sweep/sut.json` (Go harness output)
- `evidence/s3_mechanical_sweep/oracle.json` (base Cedar oracle output, 360 in-domain cases)
- `evidence/s3_mechanical_sweep/summary.json` (analyze_s3.py output, base oracle)
- `evidence/s3_mechanical_sweep/mutation_result.json` (mutation_check.py output)
- `evidence/s3_mechanical_sweep/oracle_cedar_full_domain.py` (extended oracle, adds the
  explicit `amount<=0` denial rule; does not modify `oracle_cedar.py`)
- `evidence/s3_mechanical_sweep/full_domain_check.json` (all 200 previously out-of-domain
  cases, explicit expected-vs-actual verdict, both targets)
