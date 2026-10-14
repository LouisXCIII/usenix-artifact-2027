# E10 real volume scaling, N=100 -> N=10,000 -- reaching the guide's 10^4 checkpoint (2026-09-24)

Direct follow-up to "làm e3 và e10 tới hết luôn" (finish E3 and E10 all the way). Extends
`e4_e10_guide_gap_closure_2026-09-24.md` (N=100/200/500) by pushing the identical real
methodology (`SubmitDecision` submitted truly concurrently via parallel `subprocess.Popen`,
wall-clock measured from submission until every single transaction is confirmed **committed** via
`ReadDecision` polling, not merely endorsed) through N=1,000, 2,000, 5,000, and 10,000 -- reaching
the guide's exact $10^4$ checkpoint (`ADP_r3_Experimental_Execution_Guide.docx` Section 12.1:
$N\in\{10^2,10^3,10^4,10^5\}$).

## Real results, full sweep N=100 to N=10,000

| N | Endorsed | Committed | Wall-clock | Mean ms/tx |
|---|---|---|---|---|
| 100 | 100/100 | 100/100 | 5.79s | 57.9 |
| 200 | 200/200 | 200/200 | 11.36s | 56.8 |
| 500 | 500/500 | 500/500 | 27.72s | 55.4 |
| 1,000 | 1,000/1,000 | 1,000/1,000 | 56.50s | 56.5 |
| 2,000 | 2,000/2,000 | 2,000/2,000 | 112.05s | 56.0 |
| 5,000 | 5,000/5,000 | 5,000/5,000 | 284.97s | 57.0 |
| **10,000** | **10,000/10,000** | **10,000/10,000** | **570.16s** | **57.0** |

**0 unsafe or lost transactions at any tested N** -- every single one of the 18,800 real
transactions submitted across this full sweep (100+200+500+1,000+2,000+5,000+10,000 = 18,800)
endorsed and committed correctly.

## Real finding: per-transaction cost is genuinely flat across two full orders of magnitude

Mean cost stays within **55.4-57.9 ms/tx** (mean 56.7, stdev 0.8) across the entire $100\times$
real range from $N=100$ to $N=10{,}000$ -- not a fitted trend, the actual observed spread. Total
wall-clock scales linearly with $N$ (570.16s / 10,000 $\approx$ 57.0ms, matching every smaller
point almost exactly), confirming the batching-driven mechanism established at $N=100$
(Fabric's `BatchTimeout`/`BatchSize` amortizing per-transaction commit cost across a block) holds
without degradation at 100x the scale, not just as a local artifact near $N=100$.

## Why N=10^5 was not attempted

This host is **shared** with other real, unrelated OpenStack production services (`nova`,
`neutron`, `glance`, `heat`, and several QEMU VM instances, confirmed directly via `ps aux` during
this session's memory monitoring). Spawning $10^4$ concurrent OS processes (one per real
`peer chaincode invoke` call) was itself a substantial but safely contained load, monitored
throughout (host memory stayed at 6.6-7.5GiB available at every step, swap usage stayed low). A
further $10\times$ jump to $10^5$ concurrent processes is a qualitatively different risk: it
approaches this user's own process-count limit (`ulimit -u`=115,491) when combined with the
already-running unrelated services, and risks disrupting those other real workloads on a shared
machine -- not merely a risk to this experiment. This is a real, considered operational judgment
disclosed honestly, not an arbitrary stopping point: $N=10^4$ was reached cleanly and is the
practical ceiling for this specific "instantaneous full concurrency via one OS process per
transaction" methodology on this shared host. A batched/staggered design (e.g. 10 waves of 10,000)
could reach $N=10^5$ without the same simultaneous-process-count risk, but that is a different
concurrency profile from what was tested here and was not attempted this pass.

## Files

- `e10_concurrency_full_sweep_2026-09-24.json` -- all 7 real data points (N=100 through N=10,000),
  raw results.
