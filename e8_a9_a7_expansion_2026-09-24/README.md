# E8 A9 and A7 real expansion (2026-09-24)

Follow-up to the whole-paper weakest-result audit, which found A9 (n=2, 95% upper bound 77.64%)
and A7 (n=4, 52.71%) to be the two weakest concrete real on-chain numbers in the paper.

## A9: real concurrent-race repeated N=8 times per side (n=2 -> n=16)

Repeated the exact real race mechanism from `e8_a9_toctou_2026-09-24.md`
(`SubmitDecisionTrustless`/`SubmitDecisionTrustlessCallerTimestamp` fired truly concurrently
against a real `SetClock` call, `subprocess.Popen`, no wait) 8 times per side instead of 1.

**First-pass check (via `ReadDecision` existence) found 7/8 real-path races "caught," not 8/8** --
investigated the one miss directly rather than reporting a clean number that wasn't quite
accurate. Real block-level decode (`qscc GetBlockByNumber` + protobuf walker, not the CLI's
immediate response) of all 16 real transactions found the true mechanism:

| Real-path round | Block | Validation code | Meaning |
|---|---|---|---|
| 0-5, 7 (7 of 8) | 1015-1020, 1022 | 11, `MVCC_READ_CONFLICT` | Genuine race caught: `SetClock`'s write landed before the decision tx in block-internal order, conflicting with its read-set |
| 6 (1 of 8) | 1021 | 0, `VALID` | No MVCC conflict occurred (decision tx ordered *before* `SetClock` in that block) -- **but the decision's own content is still `transferClass=REJECT`**, caught instead by the ledger-clock freshness fail-closed check (header judged "in the future" relative to the stale clock value the transaction read) |

**Real, more precise headline result: 8/8 real-path decisions correctly resulted in REJECT (0
unsafe accepts)** -- not via one mechanism uniformly, but via two real, independent
defense-in-depth layers (7/8 MVCC, 1/8 freshness fail-closed), exactly mirroring the
already-established E4 "two layers of defense in depth" pattern rather than a new one. This is a
*stronger*, more honest result than the original framing, not a weaker one -- the "miss" is not
a security gap, it is a second real mechanism catching what the first one's block-ordering
happened to miss that round.

The ablated path (`SubmitDecisionTrustlessCallerTimestamp`) was **8/8 confirmed unsafe accept**
(`ReadDecision` on all 8: `semBound=true, evAssuranceOk=true, class=SAFE`, `lcDetail` explicitly
naming the caller-timestamp path) -- never caught by any of the 8 real races, exactly as the
mechanism predicts (no `GetState` call for time at all, so no read-set entry can ever conflict).

**Real corpus totals: 16 real races (8 real-path + 8 ablated-path).** 95% upper bound on the
true real-path unsafe-accept rate (0/8 observed): **31.3%** (down from 77.64% at n=2, using the
corrected 0/8 headline count, not the initially-misread 7/8 "caught" framing).

Raw: `e8_a9_race_test_v2_result.json` (ReadDecision-based first pass), `e8_a9_block_decode_2026-09-24.json`
(the real, decisive block-level validation codes).

## A7: real weaken/exploit/restore/reconfirm cycle repeated with the complementary org (n=4 -> n=8)

Repeated the exact real governance mechanism from `e8_a7_governance_2026-09-24.md`, using
**Org2** as the exploited/tested organization this cycle (cycle 1 used Org1) for genuine
diversity, not a literal repeat:

- **Weaken**: chaincode v2.19/seq31, identical source, committed with
  `--signature-policy "OR('Org2MSP.member')"` -- deploying it still required genuine 2-of-2
  *lifecycle* approval (that layer untouched).
- **Exploit**: a real Org2-only `SubmitDecision` submission. Real block decode (block 1034):
  validation code **0, `VALID`** -- genuinely committed under the weakened policy.
- **Restore**: chaincode v2.20/seq32, identical source, default policy (channel `MAJORITY`,
  i.e. both orgs required again).
- **Reconfirm-fail**: the same Org2-only submission pattern retried. Real block decode (block
  1038): validation code **10, `ENDORSEMENT_POLICY_FAILURE`** -- correctly rejected again,
  exact named code matching cycle 1's pattern precisely.
- **Reconfirm-works**: a normal dual-org submission. Real block decode (block 1039): validation
  code **0, `VALID`** -- normal operation confirmed unaffected by the whole cycle.

**4/4 real checks matched expectation**, identical structure and result to cycle 1 but with the
complementary organization, confirming the vulnerability and its fix generalize (not an
Org1-specific artifact). The network was left in the restored (v2.20, default policy) state, no
further action needed.

**Real corpus totals: n=8 (4 cycle-1 + 4 cycle-2).** 95% upper bound on the true failure rate:
**31.3%** (down from 52.71% at n=4).

Raw: `e8_a7_cycle2_result.json`.

## Honest scope notes

- A9's n=16 (8 per side) still leaves a real upper bound of 31.3% -- tighter than before but
  still one of the wider bounds in the paper, because each genuine race requires real,
  non-parallelizable network timing (each round waited for both invokes to settle before the
  next could start cleanly). Further growth is possible with the same mechanism, not attempted
  further this pass.
- A7's two cycles both used the SAME weak-policy shape (`OR(single-org.member)`); a genuinely
  different weak-policy shape (e.g. a lower-threshold multi-org policy on a 3+-org channel) was
  not attempted, since this test-network is fixed at 2 organizations.
