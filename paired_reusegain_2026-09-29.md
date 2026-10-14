# Paired ReuseGain confirmation (B1perf vs B4, identical 22 files), 2026-09-29

Closes the "still open" item from review-4/5: recompute ReuseGain as a genuinely
paired design (same files on both sides), using data already collected --
no new transactions were run for this.

## Why this is needed
The main RQ4b ReuseGain figure (42.6-48.6%, Mann-Whitney) compares $B_1^{perf}$
(measured on the 22 real `test_contracts/*.sol` files) against $B_0$/$B_3$/$B_4$
(measured on a *different* set of 22 synthetically-named fixtures, e.g.
`CellCounter27` -- confirmed by reading `e9perf_diverse22_b0b3b4_2026-09-25.json`'s
file names directly). Matched in count and realism, not in item identity --
Mann-Whitney (independent samples) is the statistically correct test for that
comparison, and remains so; this file adds a second, complementary check: is the
same conclusion also true item-for-item on the identical 22 files, not just in
aggregate across two different samples?

## Method
- $B_4$ side: `evidence/s4_fair_b1/e2e_timing_run1.json` + `run2.json`, key
  `samples["b4-groth16"]`, `totalNs` (in-chaincode component time, NOT the
  client submit-to-commit wall time). Pooled both runs = 60 reps/file, median
  taken per file, converted ns -> ms.
- $B_1^{perf}$ side: `evidence/e9perf_b1_warm_rescan_2026-09-25.json`,
  `results[].wall_ms` (single post-warm-up measurement per file, same
  convention as Table 6).
- Joined by exact file name (all 22 names matched with zero mismatches,
  verified programmatically, not by eye).
- Per-file ReuseGain = (B1perf_ms - B4_median_ms) / B1perf_ms.
- `scipy.stats.wilcoxon` (two-sided) on the 22 paired (B1perf, B4) values.

## Result
n = 22 pairs. All 22 favor $B_4$ (every signed difference has the same sign).
Wilcoxon statistic = 0.0, p = 4.768e-07 (the minimum attainable two-sided
p-value at n=22 when every difference has the same sign -- matches the other
paired Wilcoxon result already in the paper for the same-relation comparison,
which is also all-same-sign at n=22; not a coincidence to be alarmed by).
Median ReuseGain 49.3%, mean 52.1%, range 25.6%-78.3% across the 22 files.
Consistent with, and slightly above the midpoint of, the independent-samples
42.6%-48.6% estimate.

Full per-file pairs: `paired_reusegain_result.json` (this directory).

## What this does NOT show
- Still one host, one CPU-governor state, one run's worth of B1perf timing
  per file (n=1 sample per file on that side, not a distribution) -- the
  per-file ReuseGain values above should not be read as tight per-file
  confidence intervals, only the aggregate median/Wilcoxon result is the
  robust claim.
- Does not change the headline 43-49% range already reported; it is an
  additional, stronger-design confirmation of it, not a replacement.
