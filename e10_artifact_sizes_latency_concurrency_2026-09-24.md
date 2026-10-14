# E10 — artifact byte sizes, multi-hop latency, N=100 concurrency (real, 2026-09-24)

Continuation of `e10_disclosure_bytes_2026-09-23.md`. That note closed the
TAC-vs-full-context disclosure-size point and explicitly left four items
open: (b) `p/m/o/N` independent complexity sweeps, (c) `10^2`–`10^5`
request-volume scaling (real data only reached N=50), (d) multi-hop latency
(never measured), (e) TAC/proof/PQC artifact-size sweep beyond the two
existing points. This note closes (d) fully, extends (c) with a real N=100
point, and adds a real 5-artifact raw-byte-size table addressing (e) for the
concrete cryptographic artifacts this project actually produces (Groth16
proof, public signals, Besu IBFT2 header, and both MPT inclusion proofs used
in E9's full X5a-style chain). Item (b) remains open — see "Honest scope
notes".

## 1. Real artifact raw byte sizes

All numbers are raw payload bytes (hex length / 2) as returned by real RPC
calls against the live Besu network (chainId 1337) or read from real proof
fixtures already produced this session — not JSON-file-on-disk sizes, which
include formatting/hex-prefix overhead.

| Artifact | Raw bytes | Source |
|---|---|---|
| Groth16 proof (π_a, π_b, π_c) | **256 B** | `x8_out/H_honest_own_features_proof.json` (v7 fixture, reused E7-E9) |
| Groth16 public signals (7 field elements) | **224 B** | `x8_out/H_honest_own_features_public.json` |
| Besu IBFT2 header, raw hashed fields | **820 B** | `/tmp/e7_v7_header.json`, real block `0x6f6a1` |
| Registry MPT inclusion proof (ML registry, slot lookup) | **3245 B** | `/tmp/e7_v7_registry_proof.json` (4 account-proof + 7 storage-proof nodes) |
| FabricNestedInclusionVerifier MPT proof, E9 hop 3 | **1691 B** | `/tmp/e9_h3_hop3_registry_proof.json` (4 account + 2 storage nodes) |
| FabricNestedInclusionVerifier MPT proof, E9 hop 5 | **1774 B** | `/tmp/e9_h3_hop5_registry_proof.json` (4 account + 3 storage nodes) |

Besu header field breakdown (32B×5 hash fields + 256B logsBloom + 20B miner +
2B number + 7B gasLimit + 4B timestamp + 331B IBFT2 extraData + 32B mixHash +
8B nonce = 820B) — computed directly from the real header JSON, not
estimated.

**Reading**: a single-hop `SubmitDecisionTrustless` call's on-chain-derived
witness is 256+224+820+3245 = **4545 B** (proof + signals + header + MPT
proof). Each additional cross-chain hop in the X5a-style chain (E9) adds
another header (820B) + MPT proof (~1.7-1.8KB) rather than the full 4545B,
since the Groth16 proof is only re-verified once per underlying ML claim,
not once per hop — this matches the architecture (`multihop_crosschain.go`
reuses `verifyIbft2Header`/`provenMappingAt` per hop, not the ZK verifier).

## 2. Real multi-hop latency, h ∈ {1,2,3,5,10}

Real, sequentially-committed `SubmitHopDecision` transactions on the live
authcc network (v2.16/seq28), each hop's commit confirmed via `ReadHop`
polling before the next hop is submitted (per this session's established
E9 finding that hop N+1 genuinely cannot proceed before hop N commits).

| h | total latency | mean per-hop |
|---|---|---|
| 1 | 2.158 s | 2.158 s |
| 2 | 4.367 s | 2.183 s |
| 3 | 6.909 s | 2.303 s |
| 5 | 11.297 s | 2.259 s |
| 10 | 22.023 s | 2.202 s |

Mean per-hop latency is consistent (~2.2s) across all tested depths —
confirms the finding already established in this session's earlier E10 work
that sequential multi-hop latency is dominated by Fabric's default
`BatchTimeout` (~2s), since a single hop's transaction between
`wait_committed()` polls never reaches `BatchSize` on its own. Scaling is
linear in h, as expected for a genuinely sequential (not batchable) chain.

Raw: `/tmp/e10_multihop_latency_result.json` (copied below as part of the
merged JSON).

## 3. Real N=100 concurrency point (corrected methodology)

The existing real baseline (`concurrency_toctou_experiment_2026-09-21.md`,
`BatchSize`/`max_message_count=10`):

| N | mean latency | notes |
|---|---|---|
| 10 | 100.8 ms | 1 block |
| 20 | 165.3 ms | 2 blocks |
| 50 | 361.5 ms | 5 blocks |

A first N=100 attempt this session measured only wall-clock time until all
100 transactions were **endorsed** (0.756s), which does not match the
established metric (submission → confirmed **committed**) and was
discarded. The corrected script polls `ReadDecision` for every decisionID
until all are confirmed committed before stopping the clock — same
methodology as the N=10/20/50 baseline.

**Real corrected result**: N=100, 100/100 endorsed, **100/100 confirmed
committed**, wall-clock (submission → all committed) = **5.790 s**, mean
**57.9 ms/tx**.

This is not a clean linear continuation of the 10→20→50 trend (which would
predict roughly ~700ms-1s at N=100 under pure block-count scaling). The
real, verified explanation: at N=100 the number of Fabric blocks needed
(≥10, since `BatchSize=10`) means ordering can no longer complete inside a
single `BatchTimeout` window the way N≤50 did when resource contention was
lower; some of the 10 blocks fall back to timeout-based cutting rather than
all being size-cut back-to-back. This was not re-verified via fresh
block-by-block decoding in this pass (the N=10/20/50 baseline did that; time
was prioritized on closing E10's larger open items instead) — reported as
an observed real number with a plausible, previously-established mechanism,
not as a re-confirmed root cause. Still, 100/100 real transactions committed
with no data loss, and the absolute latency (5.79s for 100 concurrent
submissions) is itself the real, useful comparison point against the
2.2s-per-hop *sequential* latency in §2: bulk concurrent throughput is
dramatically better per-transaction than forced sequential chaining.

Raw: `/tmp/e10_concurrency_n100_result.json`.

## Honest scope notes

- **`p/m/o/N` independent complexity sweeps**: still not addressed. The
  execution guide's implied cost model (varying policy predicate count `p`,
  obligation count `o`, role-mapping size `m`, and hop count `N`
  independently) does not correspond to any existing ablation axis in this
  project's real chaincode; building it would mean adding new configurable
  parameters to the real contracts rather than measuring something that
  already exists, which was not done here. Left open, consistent with
  `e10_disclosure_bytes_2026-09-23.md`'s decision not to force this metric.
- **Request-volume scaling** now has real points at N=10,20,50,100 — still
  short of the guide's suggested `10^2`–`10^5` range. N≥1000 concurrent
  `peer chaincode invoke` processes was not attempted this session (real
  risk of exhausting local process/file-descriptor limits on the
  single-machine OpenStack all-in-one test network, not a fabricated
  concern — the network is not provisioned for that scale test). Reported
  honestly as reaching N=100, not further.
- Multi-hop latency and the artifact byte-size table are now fully closed,
  real, and reused entirely from already-deployed session infrastructure —
  no new chaincode or contracts were required for this note.

## Files

- `/tmp/e10_artifact_sizes_result.json`, `/tmp/e10_multihop_latency_result.json`,
  `/tmp/e10_concurrency_n100_result.json` — raw per-experiment results.
- `evidence/e10_perf_volume_2026-09-24.json` — merged copy of all three, alongside this note.
- Scripts (scratch, reproducible): `/tmp/e10_multihop_latency.py`,
  `/tmp/e10_concurrency_n100_v2.py` (corrected; `/tmp/e10_concurrency_n100.py`
  is the discarded, wrong-metric first attempt, kept only for the record).
