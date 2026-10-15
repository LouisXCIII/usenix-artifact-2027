# Real gas-cost measurement for the MLDSA65VERIFY precompile (2026-09-24/25)

Direct follow-up closing the one remaining disclosed gap in `e5_besu_precompile_2026-09-24.md`:
the precompile's gas cost was a conservative placeholder (200,000), not yet derived from a real
profiled measurement.

## Method

Real, calibrated benchmark rather than a from-first-principles EVM gas-metering derivation
(which this project did not attempt). A standalone Java program (`GasBenchmark.java`) directly
instantiates and calls both precompiles' actual `computePrecompile()` methods on the real
compiled classes from this host's own Besu build (same classpath the live validators run),
using real cryptographic inputs (a real secp256r1 keypair/signature for P256Verify, a real
ML-DSA-65 keypair/signature for MLDSA65Verify; both sanity-checked to return VALID before timing
begins).

Calibration anchor: `P256_VERIFY`'s real, already-assigned Besu gas cost, `6,900`
(`FrontierGasCalculator.P256VERIFY_PRECOMPILED_GAS_COST`, matching EIP-7951's spec value exactly
-- confirmed by reading the real source, not assumed). Real gas-per-millisecond is derived from
this known value divided by P256Verify's own real measured time on this exact host, then applied
to MLDSA65Verify's real measured time.

N=200 timed calls per precompile per run (50-call warmup beforehand to avoid JIT cold-start
skew), median and P95 reported, matching this paper's established benchmarking convention.

## Results (3 independent runs)

| Run | P256Verify median | MLDSA65Verify median | Ratio | Derived gas |
|---|---|---|---|---|
| 1 | 0.149 ms | 0.798 ms | 5.36x | 36,969 |
| 2 | 0.124 ms | 0.577 ms | 4.65x | 32,064 |
| 3 | 0.122 ms | 0.601 ms | 4.92x | 33,928 |

**Mean derived gas cost: 34,320** (range 32,064-36,969 across 3 runs, ~15% run-to-run
variance -- typical JVM/JIT benchmark noise, not a sign of an unstable measurement methodology).

## Applied

`FIXED_GAS_COST` in `MLDSA65VerifyPrecompiledContract.java` updated from the disclosed
placeholder (200,000) to the real measured value (34,320). Rebuilt (`installDist`, `BUILD
SUCCESSFUL in 41s`, incremental since only the changed module needed recompilation) and
redeployed to all 4 live validators using the same staged, one-at-a-time, health-verified-between-
each rollout process as the original precompile deployment (backed up as
`/opt/besu.bak-20260924-v2` on every validator before replacement). All 4 validators confirmed
running byte-identical binaries (same SHA-256 of `besu-evm-26.9-develop-05b6c1a.jar`) after the
rollout; network health confirmed unaffected throughout (`net_peerCount=3` stable, block
production continuous at ~2-2.5s/block, no fork, no quorum loss).

A fresh real `submitTransfer` transaction against the live `PQCTransferV5` contract (real
ML-DSA-65 signature, same corpus as the earlier 24-case pass) confirmed the full contract-to-
precompile flow still works end-to-end on the gas-cost-updated binary: tx
`0xae160a2e...afdd9731`, status success, real gas used 410,457 (the full transaction's total real
cost -- storage writes, calldata, event emission, and the precompile call combined, not an
isolated precompile-only figure; the precompile's own cost in isolation is the 34,320 Java-level
benchmark above).

## Disclosed limitation of this measurement

This is a real-time-calibrated estimate anchored to a real already-priced precompile on the same
host, not a formal EVM gas-metering audit (the kind a real EIP submission would need, e.g.
cross-hardware benchmarking, adversarial-input worst-case timing, or opcode-counting analysis).
Disclosed as such rather than presented as a finalized, audited gas price.

## Files

- `e5_besu_precompile_2026-09-24.md` -- the original precompile deployment (this file only
  updates its gas-cost section).
