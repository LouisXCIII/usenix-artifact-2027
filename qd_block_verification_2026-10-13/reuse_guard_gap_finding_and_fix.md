# Finding + fix, 2026-10-13: `SubmitDecisionTrustlessV7ObligationVerifiedDischarge` was missing the reuse/double-citation guard

## Context

While generating a fresh real on-chain test batch (6 obligation-coverage scenarios: clean,
confused-deputy, reuse, S5-stripped, S6-weakened, S7-laundered) for the independent-verification
artifact ("Mức 1"/"Mức 3" redesign, 2026-10-13), the "reuse" scenario was built by citing the same
genuinely-discharged record from two different decisionIds on
`SubmitDecisionTrustlessV7ObligationVerifiedDischarge` (`ablation_behavior_safety.go`), expecting
the second citation to REJECT (same protection as `SubmitDecisionTrustlessV7ObligationDomainVerified`,
fixed 2026-10-08 per `evidence/qd2_domain_attestation_2026-10-08.md`'s red-team finding).

**Live, reproduced result before the fix**: both citations returned `SAFE`.

## Root cause (confirmed by direct source reading, not assumed)

`SubmitDecisionTrustlessV7ObligationVerifiedDischarge`'s discharge-match block checked only:
(a) the referenced record's `ObligationDischarged == true`, and (b) `sourceDomain`/`role`/`amount`
match the citing decision. It never checked or wrote any "already consumed" marker, unlike its
sibling `SubmitDecisionTrustlessV7ObligationDomainVerified` (`q_d_domain_attestation.go`), which
reads/writes a `discharge-used:<ref>` ledger key. The 2026-10-08 fix was applied only to the AWS
co-signed entry point, never backported to this lighter one — an oversight, not an intentional
scope decision (nothing in `ablation_behavior_safety.go`'s own docstring claims reuse protection;
it explicitly lists what it *does* close -- the bare-claim and confused-deputy gaps -- and does not
mention reuse, confirming this was simply missed rather than deferred on purpose).

## Fix (additive within the same function, no new entry point)

Ported the exact `discharge-used:<ref>` check/write pattern from `q_d_domain_attestation.go` into
`SubmitDecisionTrustlessV7ObligationVerifiedDischarge`:
- Added a `case usedData != nil` branch in the discharge-match `switch`, returning the same
  "already consumed... re-citation rejected" detail string used by the DomainVerified path.
- Added the `discharge-used:<ref>` `PutState` write after the decision record itself is committed,
  gated on `obligationDischarged && obligationDischargeRef != ""` (same condition as
  DomainVerified) -- marks a discharge "used" exactly when it was genuinely, validly consumed,
  regardless of the final `TransferClass` (so a discharge consumed by a decision that is REJECTed
  for an unrelated reason, e.g. amount-over-threshold, still correctly cannot be reused).

`go build ./... && go vet ./...` clean on the patched package before deployment. Deployed as
`authcc` v2.43 / sequence 55 (previous: v2.42/seq54), via the standard clean-rsync +
`network.sh deployCC` procedure (`REBOOT_RECOVERY.md` Bước 6-quint).

## Verification (real transactions, not simulated)

1. Confirmed pre-existing data intact after the upgrade: `ReadDecision` on a decision written
   before the upgrade (`qd3-s5-verified-1791859143`) still reads correctly.
2. Re-ran the reuse scenario fresh against the patched chaincode (fixture amount 410000):
   - `SubmitDecisionFull` → `CONDITIONAL` (decisionId `qd3-reusefix-full-1791859886`)
   - `DischargeObligation` → `EXACT` (real discharge)
   - First citation (`qd3-reusefix-first-1791859886`) → **`SAFE`**
   - Second citation (`qd3-reusefix-second-1791859886`), same discharge ref, same
     sourceDomain/role/amount → **`REJECT`**, with
     `obligationDetailV2`: `"...was already consumed by decision \"qd3-reusefix-first-1791859886\"
     (one discharge covers one decision; re-citation rejected)"`.
   Full transcript: `reuse_fix_verify_result.json` in this directory.

## Disposition for the independent-verification artifact (Mức 1 / Mức 3)

The pre-fix "reuse" pair in `clean_cd_reuse_result.json` (`qd3-reuse-first-...`/
`qd3-reuse-second-...`, both `SAFE`) is **not** used as one of the 6 canonical real scenarios --
it documents the bug, not current system behavior, and is kept here only as evidence of this
finding. The canonical "reuse" scenario going forward is the post-fix pair above
(`qd3-reusefix-first-1791859886` / `qd3-reusefix-second-1791859886`), which matches
`dac_ta_quy_tac_nghia_vu.md`'s Rule 3c exactly.
