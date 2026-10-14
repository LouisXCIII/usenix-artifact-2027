# §9 Performance and Scalability — closing the large quantitative gaps (real, 2026-09-24)

Follow-up to `e10_artifact_sizes_latency_concurrency_2026-09-24.md`. That note closed E10's
disclosure/latency/concurrency items; this note tackles the larger, still-open §9 quantitative
gaps flagged in the same-day manuscript review: `tab:perf-components` (component latency
decomposition), the PQC headline numbers (`\PerfPQVerify`/`\PerfPQSize`), `tab:perf-platform`
(platform-specific execution cost), and `\PerfCPU`/`\PerfMemory`. `tab:perf-baseline` (B0-B4
paired latency) and the `p/m/o` policy/mapping/obligation complexity sweep are addressed with an
honest scope explanation rather than forced numbers -- see "Honest scope notes".

## 1. Component latency decomposition (`tab:perf-components`)

**First attempt (discarded, wrong methodology)**: measured latency deltas between
ablated-vs-unablated function pairs (A1/A2/A3/A4/A6) via separate `peer chaincode query`
invocations. Result: deltas were noise (some negative), because per-invocation CLI/gRPC
process-spawn overhead (~45-56ms median per call) completely swamped the real in-chaincode
computation. Raw: `e9perf_component_latency_result.json` (kept for the record, not used).

**Corrected approach**: added `SubmitDecisionTrustlessTimed` (new file
`chaincode/perf_instrumented.go`, chaincode v2.17/seq29 -- 9th recurrence of the known
Docker chaincode-build bug this session, fixed proactively with the established procedure), an
additive, non-security-relevant copy of the real `SubmitDecisionTrustless` logic that wraps each
internal step with `time.Now()` and returns per-step nanosecond timings in a single chaincode
invocation -- eliminating CLI overhead from the measurement entirely. N=30 real `peer chaincode
query` calls, live network:

| Component | Median | P95 | P99 |
|---|---|---|---|
| Foreign-proof (ZK) verification | 2.145 ms | 3.095 ms | 3.452 ms |
| Mapping/state relation (light-client header + MPT proof) | 0.848 ms | 1.687 ms | 1.788 ms |
| `decisionExists` read | 0.222 ms | 0.668 ms | 0.698 ms |
| Ledger-clock read (freshness input) | 0.229 ms | 0.483 ms | 0.554 ms |
| Classification (semantic/policy decision) | 0.0005 ms | 0.0008 ms | 0.0008 ms |
| `PutState` write | 0.227 ms | 0.361 ms | 0.602 ms |
| **Total (full B4 path)** | **3.621 ms** | **6.013 ms** | **6.840 ms** |

Real, useful finding: the ZK proof verification (real gnark-crypto BN254 pairing check) and the
light-client header + Merkle-Patricia proof check together account for ~83% of the total real
in-chaincode cost (2.145+0.848 = 2.993 of 3.621 ms median); every other step (existence check,
clock read, classification, state write) is sub-millisecond. The classification step itself
(0.0005 ms) directly confirms -- by real measurement, not assumption -- that the semantic/
threshold check is a simple map-lookup-plus-comparison with no measurable cost.

**Follow-up, per explicit request ("đo trực tiếp T_obligation/T_assurance thay vì suy luận")**:
the obligation and assurance checks were *initially* reported by code inspection only (same
structural pattern as the measured classification step). Directly instrumented and measured
instead: added `SubmitDecisionFullTimed` and `SubmitDecisionWithAssuranceTimed` to
`perf_instrumented.go` (same additive, non-security-relevant pattern as
`SubmitDecisionTrustlessTimed`), deployed as chaincode v2.18/seq30 (10th recurrence of the
known Docker chaincode-build bug this session, fixed proactively with the established
procedure). Real N=30 results:

| Component | Median | P95 | P99 |
|---|---|---|---|
| Obligation check (`requiresObligation := amount > threshold/2`) | 0.0000 ms | 0.0001 ms | 0.0001 ms |
| Assurance check (`claimsFor(decisionSigTier)[destAssuranceRequirement]`) | 0.0005 ms | 0.0008 ms | 0.0011 ms |
| Role-mapping/threshold lookup (both functions, for reference) | ~0.0005 ms | ~0.0007 ms | ~0.0008 ms |

Confirms the earlier code-inspection-based inference with real, directly-measured evidence
(genuinely negligible, not merely assumed): both are sub-microsecond, consistent with the
measured classification step and with each other (same underlying pattern: map/switch lookup +
numeric comparison, no I/O, no cryptography). Raw:
`e9perf_obligation_assurance_result.json`.

**T_TAC (construction + verification)**, measured separately (off-chain, not yet wired into
either network's execution path -- same disclosed limitation as S19): reusing the exact real
`tacDigest`/`issueTac`/`verifyTac` logic from `s19_tac_substitution.js` (Keccak-256 + real ECDSA
sign/recover), N=30: issue (construction) median 0.623 ms (P95 0.852 ms), verify median
1.573 ms (P95 1.956 ms) -- combined lifecycle cost ≈2.20 ms median (sum of the two separately
measured real medians, not a jointly-sampled median-of-sums).

Raw: `e9perf_component_timed_result.json`, `e10_tac_timing_result.json`.

**Note on `T_execute` (destination execution)**: this project's simplest real chaincode
function, `SubmitDecision`, doubles as the practical floor for "destination execution with no
portability machinery" -- its own endorsement-time cost was separately measured (median ~48ms
across N=30 `peer chaincode query` calls in the discarded first-pass run) but that number is
dominated by the same CLI overhead noted above, not a clean in-chaincode figure; the
`SubmitDecisionTrustlessTimed` breakdown above is the reliable, noise-free source for
`T_execute` (`PutState`, 0.227 ms median) and every other pipeline stage except `T_TAC`.

## 2. PQC headline numbers (`\PerfPQVerify`, `\PerfPQSize`)

Real ML-DSA-65 (Dilithium3) sign/verify benchmark, N=100 in-process runs (after 15 discarded
warm-up iterations), using the exact real Liboqs 0.15.0 JNI binding this paper's E5 mechanism
uses (`DilithiumNative`), via a pre-existing but previously-unrun benchmark class
(`BenchCompareMLDSA65.java`, part of the shared PQC tooling from the sibling Besu-QBFT-PQC
project -- reused, not written for this paper):

| Operation | Liboqs JNI (real) | BouncyCastle 1.80 ML-DSA-65 (real, for reference) |
|---|---|---|
| KeyGen | median 0.040 ms | median 0.216 ms |
| Sign | median 0.093 ms | median 0.515 ms |
| **Verify** | **median 0.037 ms** | median 0.187 ms |

**\PerfPQVerify = 0.037 ms** (median, Liboqs JNI, n=100) -- the real number used by this paper's
actual E5 verification path. **\PerfPQSize = 3309 B** (ML-DSA-65 signature; matches
`DilithiumNative.SIG_SIZE`, confirmed identical between the JNI class constant and the live
benchmark's measured signature length) **+ 1952 B raw public key if not already known to the
destination** (5261 B combined). BouncyCastle numbers are reported for context (this is the
library besu-pqc's own consensus-signing path uses, from the sibling PQC-consensus paper) but
are not this paper's own measured path.

Raw: `pqc_bench_2026-09-24.log`.

## 3. Platform-specific execution cost (`tab:perf-platform`)

Real Besu-side gas, fetched via `eth_getTransactionReceipt` against already-mined real
transactions from this session's E5/E6 work (not newly submitted for this table):

| Contract call | Direction relevance | Real gas used |
|---|---|---|
| `FabricRealTxVerifier`: block-commitment check | Fabric→EVM (ADP overhead) | 327,461 |
| `FabricRealTxVerifier`: endorsement check | Fabric→EVM (ADP overhead) | 636,283 |
| `FabricNestedInclusionVerifier`: full chain, single-envelope block | Fabric→EVM (ADP overhead, deeper) | 1,060,906 |
| `FabricNestedInclusionVerifier`: full chain, multi-envelope block | Fabric→EVM (ADP overhead, deeper) | 1,379,887 |
| `PQCInterBankTransferV2`: valid ML-DSA-65 signature | (assurance-tier evidence, not ADP core) | 440,052 |
| `PQCInterBankTransferV2`: tampered signature (still processed) | (assurance-tier evidence, not ADP core) | 500,721 |

Combined with the real Fabric-side numbers already established this session:

| Direction | Local execution | Ledger/finality | ADP-specific overhead | Total (dominant term) |
|---|---|---|---|---|
| EVM→Fabric (Besu source, Fabric destination) | off-chain proof generation + `eth_getProof`/`eth_getBlockByNumber` reads (no gas; not separately re-timed this pass) | Fabric commit: ~2.2 s/tx sequential (`BatchTimeout`-dominated, §9.5), or 57.9 ms/tx amortized at N=100 concurrent (§9.4) | Fabric chaincode: 3.621 ms median (§1 above, light client + ZK proof + record) | ~2.2 s (Fabric commit dominates; ADP chaincode logic itself is a ~0.16% contributor) |
| Fabric→EVM (Fabric source, Besu destination) | Fabric-side commit of the referenced real transaction (already exists, not re-executed at verification time) | Besu block finality: IBFT2, 2 s block period, immediate single-block finality (`ibftConfig.json`) | Besu gas, real: 963,744 (block-commit+endorsement, `FabricRealTxVerifier`) or up to 1,379,887 (full nested-inclusion depth, `FabricNestedInclusionVerifier`) | ~2 s + 1 Besu block (gas has no wall-clock cost in this permissioned deployment beyond the 1 block it is included in) |

Honest note: gas is reported as the execution-unit measure (as the manuscript's own framing for
this subsection already states), not converted to a monetary or wall-clock figure, since this is
a permissioned deployment with no real fee market.

## 4. CPU/memory utilization under real load (`\PerfCPU`, `\PerfMemory`)

Real `docker stats` sampling (14 samples, 0.5 s apart) taken *during* a real N=100 concurrent
`SubmitDecision` burst (the same real burst methodology as
`e10_artifact_sizes_latency_concurrency_2026-09-24.md`'s N=100 concurrency point, resampled here
specifically for CPU/memory rather than latency):

| Container | CPU peak | CPU mean | Mem peak | Mem mean |
|---|---|---|---|---|
| `authcc-orderer.example.com` | 0.14% | 0.10% | 65.8 MiB | 65.5 MiB |
| `authcc-peer0.org1.example.com` | 3.96% | 2.66% | 146.8 MiB | 146.1 MiB |
| `authcc-peer0.org2.example.com` | 4.23% | 2.58% | 146.8 MiB | 146.2 MiB |

**\PerfCPU = 4.23% peak (single peer container, of a 16-thread host) at N=100 concurrent
load.** **\PerfMemory = 146.8 MiB peak (single peer container).** Real, if modest, finding: at
this testbed's load levels (up to N=100 real concurrent transactions), Fabric peer CPU/memory
usage is a small fraction of host capacity -- confirms the already-established finding
(§9.4/§9.5) that latency at this scale is bounded by ordering/`BatchTimeout` behavior, not by
compute-resource saturation. This does **not** establish a saturation point (where throughput
stops scaling with load); that would require pushing load well past N=100, which this session's
disclosed process/FD-limit constraint on N (see `e10_artifact_sizes_latency_concurrency_2026-09-24.md`
§3) also limits here.

Raw: `cpu_mem_n100_2026-09-24.json`.

## Honest scope notes

- **`tab:perf-baseline` (B0-B4 paired latency/throughput)**: investigated directly (read
  `stage6_trustless_corpus.py` and the E7 rerun methodology again). Real finding: in this
  project's actual implementation, B0/B2/B3 are **post-hoc reclassification rules** applied to
  the same already-computed real `SubmitDecisionTrustless` (B4) output (different acceptance
  formulas over the same `semBound`/`evOK`/role/amount fields), not independently executed
  systems -- so they have no real, distinct latency of their own to measure (their own rule
  evaluation is a trivial boolean expression over cached fields, similar in kind to the
  sub-microsecond `classify()` measurement in §1, and would misleadingly suggest "B0/B2/B3 cost
  ~0ms" if reported literally). B1 ("full destination re-evaluation") likewise has no built
  standalone re-verification *system* in this project to time -- only a local Python
  ground-truth formula (`e1_ground_truth.py`), whose own evaluation is similarly trivial and
  does not represent the real cost a genuine independent re-verification engine would have. This
  is the same category of honest gap already disclosed for the "recomputation-count" metric in
  `e10_disclosure_bytes_2026-09-23.md`: forcing numbers here would either misrepresent the real
  system or require building a new, substantial subsystem (a genuine independent destination-side
  re-verification engine) that does not exist. What **is** real and reported instead: B4's own
  absolute cost (§1 above, 3.621 ms chaincode-only / ~2.2 s commit-inclusive) and the disclosure-
  size-based reuse benefit already established (§10, 6.07× compact-witness-vs-full-disclosure).
- **`p/m/o` policy/mapping/obligation complexity scaling sweep**: still not addressed, same
  reason as previously disclosed (`e10_artifact_sizes_latency_concurrency_2026-09-24.md`) -- no
  existing knob in the real chaincode independently varies policy/mapping/obligation size.
- ~~The component-latency table's `T_state`/`T_assurance` cells outside the directly-instrumented
  trustless path are extrapolated by code inspection~~ -- **closed by follow-up**: both are now
  directly measured (see "Follow-up" in §1 above), not extrapolated.
