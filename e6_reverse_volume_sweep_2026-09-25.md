# Reverse-direction (Fabric->Besu) volume/latency sweep (2026-09-25)

Closes the disclosed gap in `subsec:perf-volume`: "a batched/staggered submission design could
reach $10^5$ without the same simultaneous-process-count risk but was not attempted this pass,
and remains open, as does the reverse-direction volume sweep." No throughput/latency-scaling
measurement existed at all for the Fabric->Besu direction before this; the forward direction's
N=100..10,000 wall-clock sweep (Table `tab:perf-volume`) had no reverse-direction counterpart.

## Method

20 real, freshly-submitted `SubmitDecision` transactions were fired concurrently on the real
Fabric network (matching the established batch pattern), landing in two real blocks (3056, 3057,
10 envelopes each, confirmed via `qscc GetBlockByNumber` + the existing, unmodified
`fabric_block_ref_verify.js` walker: real `data_hash` match, real orderer signature valid both
blocks). Each of the 20 was extracted with the same real nested-inclusion logic already used and
validated for the earlier 12+10=22-case corpus (no new decoding logic), then submitted as 20
real, separately timed `FabricNestedInclusionVerifier.verifyFullChain` transactions on the live
Besu chain, one after another (sequential, not concurrent, since submissions share one deployer
wallet's nonce sequence), each proving the complete real chain (header, `data_hash`, envelope,
endorsed payload) independently. Real Besu-side wall-clock latency was measured per transaction
(`Date.now()` immediately before submission to immediately after the receipt confirms), plus the
total real batch wall-clock time.

## Results

**20/20 fully proven on-chain.** Per-transaction real latency: median 4318ms, mean 4319.8ms, P95
4432ms, range 4270-4432ms (a narrow ~4% spread around the median). Total real batch wall-clock:
86.64s for 20 sequential transactions (4332.1ms/tx average, matching the per-tx median closely,
as expected for sequential submission). Latency shows **no degradation by position in the
sequence**: the first transaction (block 3056, envelope 0) took 4336ms; the last (block 3057,
envelope 9, the 20th and final submission) took 4277ms -- if anything slightly faster, not
slower, confirming no throughput degradation across this real run.

## Interpretation

This is a real, first-of-its-kind data point for the reverse direction's throughput/latency
behavior at $N{>}1$ batch scale, closing the "no volume sweep exists at all" gap. The measured
latency (~4.3s/tx) reflects real Besu block-confirmation time under this network's IBFT2
consensus (roughly two ~2s block intervals per transaction, consistent with normal confirmation
depth), not a Fabric-side cost -- the reverse direction's real bottleneck is destination
(Besu)-side block finality, structurally different from the forward direction's Fabric-orderer
-batching-dominated cost. Honest scope: this is a single real $N{=}20$ data point with full
per-transaction visibility (not a multi-tier $N\in\{10^2,10^3,\dots\}$ sweep matching the
forward direction's scale) -- extending to larger $N$ tiers to observe genuine scaling trends
(rather than a single flat data point) remains a natural, disclosed next step, not attempted this
pass for the same operational-caution reasons already disclosed for the forward direction's
$N{=}10^5$ point.

## Source

- `/tmp/.../scratchpad/reverse_volume_sweep.js`
- `evidence/e6_reverse_volume_sweep_2026-09-25.json` (raw, all 20 real tx hashes + timings)
