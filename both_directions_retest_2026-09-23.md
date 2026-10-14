# Both E6 directions re-tested live (2026-09-23)

User asked to re-test both directions of E6. Both were re-run fresh, live,
against the real running infrastructure -- not by re-reading old results.

## Fabric -> Besu (re-tested with a brand-new transaction)

1. Submitted a genuinely new real Fabric transaction right now:
   `SubmitDecision` for a new decision id
   (`both-directions-test-1790175216`), committed in real block 240.
2. Ran the corrected extraction (`fabric_extract_real_tx.js`) and on-chain
   verifier (`f2b_real_tx_verify.js`, `FabricRealTxVerifier.sol`) against
   this brand-new transaction (not the previously-used block 239 case).
3. **Result: all 5 cases correct again** -- real block-commit verified true
   (tx `0x3613252211d97fd2b2959625225a14a17e0584994e1bb765540bf69bb6f10b95`),
   real endorsement verified true with `isFullyVerified=true` (tx
   `0xc10fb9ac1271eb7e023d51af10ef5ff02ed978ecf8121dbec4cac007316dc835`), and
   all 3 negative controls (tampered signature, missing endorser, wrong
   orderer key) correctly rejected. Confirms the mechanism generalizes to a
   different, freshly created transaction, not just the one used to build it.

## Besu -> Fabric (re-tested live; found and fixed a real staleness issue)

Re-ran the established `SubmitDecisionTrustlessV7` mechanism
(`x15_invoke_independent.py`) fresh. First attempt failed even the honest
case (0/3 SAFE instead of the expected 3/3) with
`lcDetail: "header rejected: header timestamp is in the future relative to
the transaction"`.

**Root-caused, not worked around**: `authchaincode.go`'s `clock.go` (a real,
deliberate X4 mitigation already in the codebase) does not trust
submitter-chosen or wall-clock time for freshness checks -- it uses an
authoritative on-ledger clock (`clock:now`) that only a pinned identity
(`Org1MSP`/CN=`org1admin`) can advance, monotonically. Queried it directly:
`LedgerClock` returned `1790061866`, while real time was `1790175657` --
**the ledger clock was ~31.6 hours stale**, left over from the last session,
so any freshly fetched Besu header legitimately looked "future" relative to
it. This is the mechanism working exactly as designed (fail-closed against
an untrusted clock), not a bug.

Fixed by calling `SetClock` with the real, legitimate authority identity
(the same `Admin@org1.example.com` credentials already used throughout this
project, whose certificate's CN is genuinely `org1admin`) to advance the
ledger clock to real current time (with a small forward buffer, since a
fresh header must be fetched *after* the clock is set). Re-fetched fresh
Besu state (`x14_fetch_combined_proofs.js`, live block 437374) and reran:

**Result: honest 3/3 SAFE, forged 0/12 SAFE (all correctly REJECTED)** --
exactly matching the original real X15 result, now reconfirmed live and
fresh rather than replayed from old data. Raw output:
`evidence/x15_fresh_retest_2026-09-23.json`.

## What this confirms

- Both directions of E6 are real, working mechanisms right now, not results
  that only worked once and were never re-checked.
- A genuine operational gap (the ledger clock needing periodic advancement
  by its pinned authority after any period of inactivity) was found and
  fixed for real, and is worth noting as an operational dependency: any
  future session resuming this infrastructure should advance the ledger
  clock before relying on `SubmitDecisionTrustless*` freshness checks,
  exactly as `~/pqc-hardhat/REBOOT_RECOVERY.md`'s existing "Bước 0-bis" /
  "Bước 6-ter" entries document other post-reboot operational steps.
