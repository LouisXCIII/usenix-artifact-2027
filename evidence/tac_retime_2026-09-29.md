# TAC issue/verify re-measurement, 2026-09-29

Reviewer request: the paper's `T_TAC` (construction+verification) figure was still the stale
pre-CPU-governor-fix N=30 measurement (`0.623+1.573`ms, from
`e9perf_section9_quantitative_gaps_2026-09-24.md`), disclosed with an "order-of-magnitude
indication, not comparable component-for-component" caveat. Since TAC issue/verify is pure
off-chain crypto (Keccak-256 hashing + ECDSA sign/recover via `ethers.Wallet`, no chain/consensus
access at all), it was never actually exposed to the CPU-governor issue that affected in-chaincode
wall-clock timing -- so it can be freshly re-measured in minutes without touching any live
infrastructure, and the caveat sentence can be dropped once it is.

## Method

Reused the exact `tacDigest`/`issueTac`/`verifyTac` logic from
`~/pqc-hardhat/experiments/portability/s19_tac_substitution.js` verbatim (same real Keccak-256 +
real ECDSA signing/recovery, same Bank-A private key already used for E5/S19), in a new,
standalone, read-only timing harness (`tac_retime.js`, not committed into the pqc-hardhat repo --
run once from that directory so Node could resolve `dotenv`/`ethers`, then removed). No network
or chain calls: `issueTac`/`verifyTac` are pure local computation.

N=30 issue + N=30 verify reps, `process.hrtime.bigint()` wall-clock per rep, first 3 reps of each
discarded as JIT warm-up (consistent with the paper's own "post-warm-up" convention used
elsewhere for B4/B1perf), leaving N=27 each.

## Result

| Component | Median | P95 |
|---|---|---|
| TAC issue (construction) | 0.590 ms | 0.783 ms |
| TAC verify | 1.487 ms | 1.546 ms |
| Combined (sum of medians) | 2.077 ms | -- |

Consistent with the old pre-fix figure (0.623+1.573=2.196ms vs new 0.590+1.487=2.077ms, a ~5%
difference well within run-to-run noise for an untouched, ungoverned-by-the-CPU-fix code path) --
confirming the old figure was not actually stale in substance, only in provenance (measured
before the fix, but not by a mechanism the fix could have affected). The paper's disclosure is
updated to report this fresh figure and drop the now-unnecessary "order-of-magnitude, not
comparable" caveat.

Raw: `tac_retime_result_2026-09-29.json` (N=30 raw per-rep arrays before warm-up trim).

## Reproduction
```
cd ~/pqc-hardhat/experiments/portability
# (recreate tac_retime.js locally: identical tacDigest/issueTac/verifyTac from
#  s19_tac_substitution.js, N=30 timing loop, no chain calls)
node tac_retime.js
```
