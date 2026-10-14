# E10 — disclosure-bytes metric, real (2026-09-23)

Real, narrow result for `Manuscriptsp_m2_r3.tex`'s E10 disclosure/
recomputation metrics (execution guide section 12.3). This is the one part
of that section answerable from artifacts already produced this session
without new infrastructure; the rest is disclosed as open below rather than
forced.

## What was computed

- **TAC raw binary size** (the material a $B_4$ direct-reuse destination
  must be shown): 32-byte Keccak-256 digest + 65-byte ECDSA signature
  (r,s,v) = **97 bytes**, from the real certificate issued in
  `evidence/s19_tac_substitution_result_2026-09-23.json`.
- **Full source-decision-context size** (the material a $B_1$
  full-destination-re-evaluation would need to see, since it cannot rely on
  a compact portability witness): the real E1 generated case objects
  (N=2{,}000, same corpus as `evidence/e1_formal_benchmark_2026-09-23.md`),
  JSON-serialized: **560–625 bytes, mean 588.9 bytes**.
- **Disclosure reduction ratio**: 588.9 / 97 ≈ **6.07×**.

Raw computation: `evidence/e10_disclosure_bytes_2026-09-23.json`.

## Honest scope notes

- The full-context figure is a JSON serialization of a generated benchmark
  case, not a wire-optimized encoding of an actual source policy object; a
  production encoding could be smaller than 589 bytes, so 6.07× should be
  read as illustrative of the witness-vs-full-disclosure gap, not a tight
  wire-format-to-wire-format comparison.
- **Recomputation count** (guide: "number of destination policy
  predicates/facts reevaluated under B1 vs checks reused under B4") is
  **not reported**: this project's real B4 implementation
  (`SubmitDecisionTrustless`) does not "reuse a cached value" in the sense
  the metric implies -- it independently re-verifies `semBound`/`evOK` via a
  real ZK proof and light-client check, which is a *different and stronger*
  verification path than B1's trust in a caller-supplied flag, not a
  "skipped recomputation." Forcing a recomputation-count number here would
  either misrepresent this real system or require redesigning it around the
  guide's assumed cost model; neither was done, so this metric is left open.
- The remaining E10 items (full $p/m/o/N$ independent complexity sweeps,
  $10^2$–$10^5$ request-volume scaling, multi-hop latency, TAC/proof/PQC
  artifact-size sweep beyond the two real points already measured) are not
  covered by this note and remain open, as already disclosed in
  `subsec:perf-volume`/`subsec:perf-multihop`.
