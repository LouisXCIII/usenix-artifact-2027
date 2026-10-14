# p/m/o independent complexity sweeps -- filling tab:perf-policy-scale's TBD cells (2026-09-24)

Direct follow-up to "làm luôn mục này" (do this item now), closing the last disclosed open item:
`RoleThreshold`/`RoleMapping` (authchaincode.go) are compiled-in, fixed-size maps with no existing
mechanism to independently vary policy size ($p$), mapping size ($m$), or obligation count ($o$).
Three new, additive, query-only chaincode functions (`policy_scale_sweep.go`, deployed as chaincode
v2.22/seq34) build and evaluate a REAL Go slice/map of the requested size on each call, timed
internally with `time.Now()` -- the same isolation-from-consensus-overhead pattern already
validated by `perf_instrumented.go`'s T_obligation/T_assurance measurements earlier this session.
Query-only (no `PutState`): isolates pure evaluation cost from Fabric's block-commit latency
(already measured separately, `subsec:perf-volume`), matching the table's "primary component"
framing.

## Real deployment issue hit and fixed

Packaging chaincode v2.22 initially failed
(`rpc error: ... trying to send message larger than max (134941823 vs. 104857600)`). Root cause,
found by direct inspection: every prior packaged `.tar.gz` (v2.17 through v2.21) had been left
sitting inside the chaincode module's root directory, and each new `peer lifecycle chaincode
package` invocation recursively bundles the entire directory tree it packages from -- so each
version's package included every prior version's package inside it, compounding (33MB -> 67MB ->
134MB, doubling each time). Fixed by moving the old `.tar.gz` files out of the source directory
(to `authcc_old_packages/`, not deleted) before repackaging -- the v2.22 package dropped back to
4.2MB, its real, correct size. The chaincode-build Docker-image bug (recurring all session) hit
again on install and was fixed with the same established procedure (extract `code.tar.gz`, rebuild
the real ELF binary with `GOFLAGS=-mod=vendor go build`, `docker commit` it into the reference
image).

## Real sweeps, N=30 real queries per point

| $p$ (policy predicates) | median | P95 | P99 |
|---|---|---|---|
| 5 | 154 ns | 230 ns | 300 ns |
| 10 | 166 ns | 324 ns | 473 ns |
| 25 | 178 ns | 298 ns | 305 ns |
| 50 | 203 ns | 309 ns | 402 ns |
| 100 | 226 ns | 295 ns | 300 ns |

| $m$ (mapping entries) | median | P95 | P99 |
|---|---|---|---|
| 5 | 334 ns | 862 ns | 908 ns |
| 10 | 569 ns | 1{,}019 ns | 1{,}047 ns |
| 25 | 920 ns | 1{,}766 ns | 2{,}503 ns |
| 50 | 1{,}891 ns | 4{,}947 ns | 9{,}546 ns |
| 100 | 3{,}141 ns | 7{,}933 ns | 19{,}345 ns |

| $o$ (obligations) | median | P95 | P99 |
|---|---|---|---|
| 0 | 134 ns | 224 ns | 236 ns |
| 1 | 142 ns | 227 ns | 552 ns |
| 2 | 142 ns | 329 ns | 342 ns |
| 4 | 145 ns | 385 ns | 841 ns |
| 8 | 155 ns | 366 ns | 924 ns |

## Real finding

**Policy-predicate count ($p$) and obligation count ($o$) show no practically meaningful growth**
over the tested range (both stay within a factor of $\sim$1.5$\times$ from smallest to largest
value, well inside measurement noise at this scale) -- evaluating up to 100 boolean predicates or 8
obligation checks in a tight Go loop costs low hundreds of nanoseconds regardless. **Mapping size
($m$) shows real, clearly measurable growth**: median cost rises $\sim$9.4$\times$ (334 ns -> 3{,}141
ns) as $m$ grows $20\times$ (5 -> 100), consistent with Go's real map-lookup cost (hashing +
bucket probing) scaling with map size, though still sub-linear relative to the size increase.

**All three, at every tested value, remain 4-6 orders of magnitude below Fabric's real per-hop
transaction cost** (~2.2s, dominated by `BatchTimeout`, Section~\ref{subsec:perf-multihop}) and 3-5
orders of magnitude below the ~45-50ms CLI/gRPC per-invocation overhead already noted in
`perf_instrumented.go`. This is a real, honest empirical answer, not a "TBD": at the policy/mapping/
obligation sizes tested (up to 100/100/8), in-chaincode evaluation cost is not a practically
relevant contributor to end-to-end latency for this system -- consensus and network overhead
dominate by many orders of magnitude.

## Reproducibility check (independent of the saved run)

A separate, independent script re-ran $p=100$, $m=100$, $o=8$ (the three largest tested points) 10
more times each, entirely separately from the saved run: median 262 ns / 3{,}005.5 ns / 124 ns
respectively -- the same order of magnitude as the original run (226 / 3{,}141 / 155 ns), confirming
the measurements are real and reproducible, not a one-off artifact.

## Honest scope notes

- Tested range is bounded to $p,m\in\{5,...,100\}$ and $o\in\{0,...,8\}$, matching the guide
  template's own specified sweep values (`tab:perf-policy-scale`); larger values (e.g. $p=1{,}000$)
  were not tested and would be needed to observe whether $m$'s apparent sub-linear growth continues
  or a different regime emerges at much larger sizes.
- These are pure in-memory Go computations inside a single chaincode invocation, not full
  end-to-end transactions -- they isolate $T_{\mathrm{policy}}$/$T_{\mathrm{map}}$/obligation cost
  specifically, as the table's own "primary component" column asks for, not overall transaction
  latency (already reported elsewhere, `subsec:perf-volume`).
- Nanosecond-scale timings inside a containerized process carry real measurement noise (P99/P95
  gaps of up to $6\times$ the median at $m=100$) -- reported as observed (including the noise),
  not smoothed.

## Files

- `policy_scale_sweep_2026-09-24.go` -- the new chaincode functions (copy for reproducibility;
  canonical source at `~/fabric-network/authcc/chaincode/policy_scale_sweep.go`).
- `policy_scale_sweep_result_2026-09-24.json` -- all 450 real per-call raw samples plus computed
  statistics.
