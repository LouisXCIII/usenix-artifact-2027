# E4 N=1000 replay + E10 N=200/500 concurrency -- closing two guide-flagged gaps (2026-09-24)

Direct follow-up to a full cross-check of `ADP_r3_Experimental_Execution_Guide.docx` against the
manuscript, which found two real, honestly-disclosed gaps versus the guide's explicit requirements:
E4's replay depth sweep only reached N=100 (guide asks 1/10/100/1{,}000), and E10's request-volume
scaling only reached N=100 (guide asks 10^2/10^3/10^4/10^5).

## E4: real N=1000 replay depth (closes "1{,}000 was not run and remains open")

Reused the exact real methodology already validated at N=100
(`evidence/e4_replay_toctou_limitations_2026-09-23.md`): one real decision submitted, then
resubmitted under the identical `decisionID` 1,000 times sequentially against the live network,
via real `peer chaincode invoke` calls (not simulated).

- **991/1000 explicitly rejected at endorsement** (`"already recorded (replay rejected)"`).
- The remaining **9 (attempts #1-9)** were, as with the N=100 case, not assumed lost -- traced via
  real block decoding (`qscc GetBlockByNumber` via `peer chaincode query`, decoded with the
  existing from-scratch protobuf walker, since `peer channel fetch` only returns the orderer's
  un-annotated copy with an empty `TRANSACTIONS_FILTER`). All 10 real transactions for this
  decisionID (the 1 original + 9 earliest replays) landed in the **same real block (1122)**:
  index 0 = `VALID` (the original), indices 1-9 = **`MVCC_READ_CONFLICT`**, confirmed directly.
- **True end-to-end result: 1000/1000 correctly rejected, 0 bypasses**, via the same two real
  layers of defense already established at N=100 (endorsement-level replay check +
  ledger-commit-level MVCC read-set validation) -- the "raced ahead" window is bounded to the
  handful of attempts fired before the very first commit lands, not something that scales with N
  (exactly 9 stray attempts at both N=100 and N=1,000).
- Final ledger read-back confirms the record is unchanged from its original real values (no
  overwrite across all 1,000 attempts). A fresh-nonce positive control committed normally.
- 95% upper bound on the true miss rate, 0 observed bypasses in 1,000 real trials:
  $1-0.05^{1/1000}=0.30\%$ (down from 2.95% at $n=100$).

## E10: real concurrency at N=200 and N=500 (partially closes the 10^2-10^5 gap)

Reused the exact corrected N=100 methodology
(`evidence/e10_artifact_sizes_latency_concurrency_2026-09-24.md`'s v2 script: real, separately
committed `SubmitDecision` transactions submitted concurrently via parallel `subprocess.Popen`,
wall-clock measured from submission until every single one is confirmed **committed** via
`ReadDecision` polling, not just endorsed). Attempted cautiously given real, disclosed host memory
pressure (swap fully exhausted, 8.0/8.0 GiB, from concurrent QEMU VMs unrelated to this paper) --
each step checked host memory before proceeding.

| N | Endorsed | Committed | Wall-clock | Mean ms/tx |
|---|---|---|---|---|
| 100 (2026-09-23 baseline) | 100/100 | 100/100 | 5.790s | 57.9 |
| 200 | 200/200 | 200/200 | 11.357s | 56.8 |
| 500 | 500/500 | 500/500 | 27.720s | 55.4 |

**Real finding: per-transaction cost stays flat (55-58ms/tx) from N=100 through N=500** -- linear
total-time scaling, no degradation, confirming the earlier N=100 point was not an isolated
measurement but part of a genuinely stable trend across a 5x larger real range.

## Honest scope notes

- E4: this closes the guide's literal N=1,000 requirement in full -- no further replay-depth gap
  remains versus the guide text.
- E10: N=500 is still $200\times$ short of the guide's $10^5$ upper target. Real host memory
  pressure (swap exhausted by unrelated concurrent VM workloads on this shared OpenStack host) was
  monitored at each step; N=500 completed cleanly with memory stable throughout, so the ceiling
  observed here is not yet a confirmed hard limit -- but pushing toward $10^3$-$10^5$ was not
  attempted this pass, consistent with the already-disclosed risk of exhausting local
  process/file-descriptor limits at that scale. This narrows, but does not close, the guide's
  volume-scaling gap.

## Files

- `e4_replay_depth_1000_result_2026-09-24.json` -- full real N=1,000 replay result.
- `e10_concurrency_n200_n500_2026-09-24.json` -- N=100/200/500 real concurrency results.
