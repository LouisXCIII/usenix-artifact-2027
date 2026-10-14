# S4 protocol: fair B1 baseline and disclosure measurement (roadmap item #4)

Written 2026-09-28, BEFORE any B1-fair measurement was run. Amendments are appended at the bottom with dates.

## Why
The existing B1 (`b1_warm_rescan.py`, `subsec:perf-reevaluation`) is a PyTorch scan, off-chain, in Python, over `.sol` files. B4 is a Go
chaincode entry point. Different language, runtime and trust boundary, so `ReuseGain = (T_B1 - T_B4)/T_B1` compares two different things.
Roadmap #4 asks for (a) B1 with the same policy engine, language and runtime as B4, and (b) a scenario where the destination has no
access to the source's private data: measure how much must be disclosed under B1 versus ADP. Report endorsement time and end-to-end separately.

## What the proof actually attests (read from `circuits_v7/ml_scan_proof_v7.circom`, not assumed)
The Groth16 relation covers ONLY the last layer of the source classifier: private x[256], W[12][256], b[12], salt; public signals
solidityCodeHash, commitmentHash, maxVulnClass, minMargin, modelWeightsHash, featureHash (+ output scanPassed). The circuit computes
logits = W x + b (quantised, scale 4096), argmax, margin >= minMargin, Poseidon digest of W,b (206 chunks of 15) equal to modelWeightsHash,
Poseidon digest of x (18 chunks of 15) equal to featureHash. The upstream feature extraction (tokeniser + VulnLSTM up to classifier[8])
is NOT in the circuit. So a B1 that re-executes exactly this relation gives the same assurance as the proof; a B1 that re-runs the whole
scanner (the old B1) gives a stronger, different assurance.

## Definitions (frozen)
- B4 = `SubmitDecisionTrustlessV7`: existence check, Groth16 verify (7 public signals), source header + registry MPT proofs
  (`verifySourceState`), feature-commitment MPT proof (`verifyFeatureBinding`), `classify`.
- B1-same-relation (PRIMARY, fair): identical to B4 except the Groth16 check is replaced by re-executing the relation in Go, in the same
  entry-point shape and runtime:
  (i) `verifySourceState` and `verifyFeatureBinding` unchanged (they authenticate x through the pinned on-chain featureHash commitment);
  (ii) recompute the Poseidon feature digest of the supplied x and require it to equal the proven on-chain featureHash;
  (iii) recompute logits = W x + b in big-int as the circuit does, argmax, class <= requiredMaxVulnClass, margin >= requiredMinMargin;
  (iv) require W,b to be the pinned model: two variants:
       - B1-store: the destination stores the pinned (W,b) at deployment (weights are pinned by hash), no per-call digest;
       - B1-nostore: the destination recomputes the 206-chunk Poseidon weights digest on every call and compares it with the pinned hash.
  (v) `classify` unchanged.
  B1-store is the primary comparison (it favours B1; if ADP wins against B1-store the claim is conservative).
- B1-full-pipeline: the existing Python warm rescan (`e9perf_b1_warm_rescan_2026-09-25.json`), kept as a separate row. Different assurance.
- Poseidon must be the SAME function as circomlib's optimised Poseidon (t=17), verified against circomlibjs test vectors before use,
  and implemented with the optimised partial rounds (not the slow reference form) so B1 is not made artificially slow.

## What is measured
1. Endorsement-time proxy (in-function, same process, same host, interleaved B4 / B1-store / B1-nostore per fixture, N >= 30 per row,
   CPU governor `performance`): median, P95, P99, bootstrap 95% CI, paired Wilcoxon on per-fixture pairs. In-function time excludes peer
   gRPC/endorsement overhead; that overhead is the same for both and is measured end-to-end on Fabric in a later step (not part of this offline run).
2. Component breakdown: source-state verify, feature binding, Groth16, Poseidon(x), logits, Poseidon(W,b).
3. Disclosure (per decision, exact counts):
   - B1 must receive: x (256 fields), and W,b (3084 fields) unless stored; bytes as serialised in the entry-point argument.
   - ADP must receive: proof + 7 public signals (+ header/MPT proofs, which B1 also needs).
   - Report fields and bytes for each, and the subset that is source-private (x; W,b if the model is proprietary).
   Not claimed: that x can or cannot be inverted to source code. The claim is only "B1 requires the destination to be given x; ADP does not".
4. Feasibility statement for the scenario where the source will not release x: B1 is not executable (0 of N), ADP is.

## What is NOT measured here (declared up front)
- End-to-end on Fabric (peer/orderer/commit); needs the Fabric network.
- Any relation upstream of the last layer.
- Whether real source domains would disclose x.

## Pre-declared reading of outcomes
The result is reported whichever way it falls. If B1-store is faster than B4 in endorsement time, the paper must say so and rest the
value of ADP on disclosure and on not provisioning/maintaining the source model at the destination, not on latency. No threshold or
parameter is changed after seeing timings.

## Amendments
(none yet)

- A1 (2026-09-28, AFTER the first timing run, dated after seeing data): added variant B1-sha (SHA-256 commitment to x, stored model), because Poseidon(x)
  accounted for nearly all of B1-store's cost and a reviewer could say B1 need not use a circuit-friendly hash. It favours B1, is reported as an extra row,
  and the primary B1-store row and the B4 row are unchanged. Runs 2 and 3 include it; run 1 does not.
- A2 (2026-09-28): the offline harness times the components that differ between B4 and B1 (Groth16 verification vs re-execution). The shared components
  (header, MPT proofs, classify) are identical in both and need the Besu source chain to build fresh fixtures, so entry-point-level timing is left for the network run.
- A3 (2026-09-28): proofs used for timing were generated under a relaxed policy (maxVulnClass=11, minMargin=0) so that all 22 files have one; Groth16 verification
  runs the pairing before any policy check and cost does not depend on public values (the harness asserts the pairing passed). Correctness parity used the strict policy.
- A4 (2026-09-28, after the offline results): the end-to-end run was done on the real networks. Besu main chain (chainId 1337) and Fabric `authcc` were restarted per the runbook.
  22 fixtures were registered on Besu (ML registry 0x4BDF...: `recordMLScanPass` only where the bit was not already set by earlier corpus work, approvals by admins A and B;
  FeatureCommitRegistry #1 (Poseidon commitments, scanner authority C) and a NEW FeatureCommitRegistry #2 at 0xc425C824... holding SHA-256(x) commitments). NOTE: several
  `mlScanPassed` bits were already true from the earlier corpus (real g1 verdicts), so they reflect the scanner's verdict, not the strict-margin criterion used for B1 parity;
  class parity does not depend on it (all variants reject when the strict relation fails).
- A5: chaincode v2.27/seq39 added `*Timed` entry points (B4 copy, B1 store/nostore/sha) that return timing JSON. Those are non-deterministic across endorsers, so they can only be
  used with `peer chaincode query` (endorsement only). For real submissions v2.28/seq40 added deterministic B1 entry points (return the class only); the B4 submission path is the
  deployed, UNMODIFIED `SubmitDecisionTrustlessV7`. Existing files and `authchaincode.go` (sha256 e50ed4b1...) are unchanged.
- A6: the Fabric ledger clock was set once to now+300 s and one Besu header fetched afterwards; that single header (age ~5 min on the ledger clock, bound 24 h) is used for all runs.
  Run 1 timings ran on v2.27, run 2 on v2.28 (timed code identical). 30 repetitions per file and variant, order rotated per repetition.
