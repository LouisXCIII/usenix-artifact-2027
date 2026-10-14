# S5-S7 and S15-S16 real on-chain guards (2026-09-24)

Follow-up to the whole-paper weakest-result audit, which flagged S5-S7 (obligation model) and
S15-S16 (assurance-claim model) as "weak in two ways": small n (7 and 6) AND, unlike every other
S-scenario in the adversarial suite, never wired into a real on-chain guard -- both were tested
only by standalone local Python scripts (`e3_obligation_model.py`, `e3_assurance_claim_model.py`)
with zero chaincode counterpart. This note closes the second, more fundamental gap.

## Design

Two new, additive chaincode files, exact ports of the Python models' logic, wired into the same
real destination decision pipeline (semBound/evAssuranceOK/role/amount) every other
`SubmitDecision*` entry point uses:

- **`obligation_v2.go` / `SubmitDecisionObligationV2`**: `oblCoveredV2()` is a line-for-line Go
  port of `obl_covered_v2()` -- same ordinal mechanism-strength tables
  (`none<notification<audit<dual_approval<combined`), same four real inputs (obligation profile,
  mapped mechanism, discharged flag, enforcement-witness-verified flag), same five real
  code paths (S5 stripped / S6 weakened / not-yet-discharged / S7 laundered / genuinely covered).
- **`assurance_v2.go` / `SubmitDecisionAssuranceV2`**: exact port of `assure_ok_v2()` -- the
  destination credits claims from the **verified** tier only (via `claimsFor()`, reused
  completely unchanged from `assurance_a6.go`), the **claimed** tier is recorded separately for
  over-claim detection but never gates the security decision -- the real S15/S16 fix.
- Both write additive fields onto the shared `PortableDecisionRecord` (`obligationCoveredV2`,
  `obligationDetailV2`, `claimedTier`, `verifiedTier`, `overClaimDetected`, etc.), so
  `ReadDecision` exposes the full real reasoning, not just a class label.

Deployed as chaincode v2.21/seq33 (13th recurrence of the known Docker chaincode-build bug this
session, fixed proactively with the now-completely-routine procedure).

## Real results: all 13 original Python scenarios, replicated as real committed transactions

**Obligation model (S5/S6/S7), 7 real on-chain transactions -- 7/7 correct**, exactly matching
`e3_obligation_model_result_2026-09-23.json`'s original local-Python result:

| Scenario | Expected covered | Real on-chain result | Real class |
|---|---|---|---|
| positive_control | True | True | EXACT |
| S5_stripped | False | False | REJECT |
| S6_weakened | False | False | REJECT |
| S7_laundered | False | False | REJECT |
| boundary_equal_strength | True | True | EXACT |
| boundary_stronger_mechanism | True | True | EXACT |
| empty_profile | True | True | EXACT |

**Assurance-claim model (S15/S16), 6 real on-chain transactions -- 6/6 correct**, exactly
matching `e3_assurance_claim_model_result_2026-09-23.json`'s original result:

| Scenario | Claimed | Verified | Expected OK | Real class |
|---|---|---|---|---|
| S15_honest_classical_pq_channel | classical | classical | False | REJECT |
| S16_fake_upgrade_claimed_hybrid | hybrid | classical | False | REJECT |
| S16_fake_upgrade_claimed_pq | pq | classical | False | REJECT |
| S16_fake_upgrade_unverifiable | pq | (none) | False | REJECT |
| positive_control_verified_hybrid | hybrid | hybrid | True | EXACT |
| positive_control_classical_ok | classical | classical | True | EXACT |

Spot-checked full record transparency: the S16 case's real committed record reads
`claimedTier:"hybrid", verifiedTier:"classical", overClaimDetected:true` -- the over-claim is
recorded and visible, but the security decision (REJECT) used only the verified tier, exactly as
intended. The S7 case's record reads
`obligationDetailV2:"S7: obligation laundered -- discharge is claimed but has no independently
verifiable enforcement witness"` -- a real, human-readable, on-chain-recorded reason, not just a
pass/fail flag.

**13/13 total real on-chain transactions, 0 discrepancies from the original local-model result.**

## What this closes and what remains

Closes the "no real on-chain guard" half of the original weakness completely: S5-S7/S15-S16 now
have the same category of evidence (real, separately-committed Fabric transactions) as every
other scenario in the adversarial suite. The "small n" half is only partially addressed --
n=7/n=6 real transactions is now the SAME n as the original Python model (by design, since the
goal was replicating the exact same scenario matrix on-chain, not inventing new cases); a
further real value-sweep across each ordinal mechanism-strength level (matching the pattern
already used for A2/A6) was not attempted this pass.
