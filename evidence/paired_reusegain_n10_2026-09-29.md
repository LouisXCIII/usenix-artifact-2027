# Paired ReuseGain, N=10-per-file re-measurement of B1perf (fixes a repetition-count asymmetry), 2026-09-29

Reviewer point: the paired ReuseGain comparison (`paired_reusegain_2026-09-29.md`,
`paired_reusegain_result.json`) joined B4's per-file median of 60 pooled in-chaincode reps
against a SINGLE B1perf measurement per file -- a genuine repetition-count asymmetry between
the two sides of the pair, since a median of many reps and one lone sample are not directly
comparable measures of central tendency.

## Fix

Re-measured B1perf as 10 fresh reps per file (median taken per file), reusing the exact same
model/classify/role/amount setup as the original `b1_warm_rescan.py`
(`~/pqc-hardhat/experiments/portability/`), loading the model once (post-warm-up convention).
`MLScanner` caches results by content hash (`gateway.py`): to genuinely re-run inference 10
times per file rather than hit the cache on reps 2-10, `scanner._cache.clear()` is called
before every single `scan()` call, and every result's `cached` field is asserted `False`. New
script: `b1_warm_rescan_n10.py` (additive, does not modify the original).

## Result

22 files, N=10 reps each (220 genuine model-inference calls total, no cache hits):
per-file medians range roughly 2.7-10.0ms (see
`e9perf_b1_warm_rescan_n10_2026-09-29.json` for full per-file reps).

Joined by filename against the same B4 per-file medians already collected
(`paired_reusegain_result.json`'s `b4_median_ms` column, unchanged -- no new chaincode
transactions were run, only the B1perf side was re-measured):

| Metric | Old (B1perf single measurement) | New (B1perf median of 10) |
|---|---|---|
| Median ReuseGain | 49.3% | **54.2%** |
| Mean ReuseGain | 52.1% | 55.7% |
| Range | 25.6%--78.3% | 29.8%--79.4% |
| Files favoring B4 | 22/22 | 22/22 |
| Wilcoxon signed-rank (two-sided) | p=4.8e-7 | p=4.8e-7 (unchanged: still the minimum attainable p at n=22 with all differences the same sign) |

The correction moved the reported figure up (49.3%->54.2%), not down -- the direction of the
fix is not predictable a priori from the asymmetry alone (it depends on the specific run's
noise, not a systematic bias in a single direction), but the corrected, symmetric-repetition
design is the methodologically defensible one regardless of which way it moved the number.

## Reproduction
```
cd ~/pqc-hardhat/experiments/portability
python3 b1_warm_rescan_n10.py
# then join with paired_reusegain_result.json's b4_median_ms by filename, Wilcoxon signed-rank
```
Raw: `e9perf_b1_warm_rescan_n10_2026-09-29.json` (per-file 10-rep arrays + medians),
`paired_reusegain_n10_result_2026-09-29.json` (joined pairs + test statistics).
