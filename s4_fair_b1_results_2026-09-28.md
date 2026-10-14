# S4 results: fair B1 baseline and disclosure (2026-09-28)

Protocol: `s4_fair_b1_protocol_2026-09-28.md` (frozen before measuring; amendment A1 below was added after the first timing run).
Files: `s4_fair_b1/` (make_fixtures.py, fixtures.json, timings_run{1,2,3}.json, components_run1.json, analyze_s4.py, analysis_s4.json).
Code: built in a scratch copy of the chaincode (deployed `authchaincode.go` sha256 e50ed4b1... untouched). Copies of the four new Go files
(`poseidon_opt.go`, `b1fair.go`, `poseidon_opt_test.go`, `b1fair_test.go`) plus their test data are kept in `s4_fair_b1/go_src/`.

## What was built and how it was validated
- circomlib-compatible optimised Poseidon (t=17) in Go. Validated: 5 raw hashes and 3 chained digests (with negative inputs) equal circomlibjs;
  the 206-chunk digest of the real quantised weights equals the pinned `modelWeightsHash` (10494038872765858080...).
- B1 same-relation re-execution in Go (feature digest, model check, logits, argmax, margin, class), mirroring the circuit including the
  |logit| < 2^40 domain and first-maximum tie rule.
- 22 real .sol files (the same 22 distinct files as the corpus): real x from the real scanner up to classifier[8], real Groth16 v7 proofs
  (snarkjs, all 22 verify; all 22 also pass the pairing check inside the Go verifier).
- Correctness parity: B1 (store, nostore, sha) accepts exactly when the STRICT circuit relation is satisfiable (6 accepted, 16 rejected of 22),
  computed independently by running the circuit's own witness generator. Recomputed feature hashes equal the fixtures' circomlibjs values.
  Tamper checks (one feature changed, one weight changed, margin raised, truncated vector) are all refused.

## Endorsement-time proxy (in-function, Go, same process and host, order rotated per repetition, CPU governor `performance`)
Three independent replications (a fresh process each), 22 files x 100 repetitions each. Times in ms, pooled median (P95, P99):

| Variant | run 1 | run 2 | run 3 |
|---|---|---|---|
| B4: Groth16 verification (incl. argument and key decoding) | 0.851 (1.252, 1.946) | 0.849 (1.588, 1.954) | 0.844 (1.346, 1.913) |
| B1-store: Poseidon(x) + stored model | 1.637 (2.188, 3.463) | 1.623 (2.849, 3.485) | 1.614 (2.176, 3.459) |
| B1-nostore: + Poseidon digest of W,b every call | 19.830 (22.639, 24.945) | 19.867 (22.325, 25.572) | 19.795 (21.724, 24.656) |
| B1-sha (amendment A1): SHA-256 commitment to x, stored model | (not in run 1) | 0.027 (0.040, 0.048) | 0.027 (0.040, 0.048) |

Ratios B1/B4 (median over 22 files; bootstrap 95% CI over files): store 1.91-1.92 [1.89, 1.93]; nostore 23.3-23.4 [22.9, 23.5]; sha 0.032 [0.031, 0.032].
Paired Wilcoxon on the 22 per-file medians: p = 4.8e-7 for every comparison (all 22 files go the same way). Per-file spread is tiny because the work is data-independent.

Component medians (ms, first fixture, 300 reps): decode x 0.018; Poseidon(x) 1.690; SHA-256 path total 0.022; relation (3,072 MACs) 0.001;
decode W,b 0.283; Poseidon(W,b) 17.684; decode proof+signals 0.006; decode verification key 0.018; Groth16 pairing core 0.824; verifySemBoundProof total 0.859.
(The components come from a separate process, so they sum to within a few percent of, not exactly, the pooled medians above.)

## Reading (pre-declared: reported as it fell)
1. Against the same-runtime, same-relation B1, ADP's destination-side verification is NOT slower: Groth16 (0.85 ms) beats B1-store (1.6 ms) and B1-nostore (19.8 ms).
2. It is much slower than the best possible B1 (B1-sha, 0.027 ms, about 31x cheaper). B1-store's cost is almost entirely the circuit-friendly Poseidon commitment; if the scanner
   authority also commits SHA-256(x) (one more registry slot) the destination binds x almost for free. Re-executing a 12x256 linear layer costs about 1 microsecond.
3. Therefore the paper's earlier ReuseGain (43-49%) does not measure "reuse vs re-evaluation of the same thing". It compares B4 with B1^perf, an off-chain Python re-run of the
   WHOLE scanner (tokenizer + VulnLSTM), which is a stronger check than the proof (the circuit covers only the last layer; upstream features are trusted through the scanner's
   featureHash commitment). Different language, runtime, trust boundary, and assurance.
4. Cost shift, previously unmeasured: proof generation at the source takes about 8.0 s per proof (snarkjs, 5 runs, 7.9-8.1 s), 5.1-5.8 GB peak memory, plus 0.56 s witness generation;
   proving key 199 MB. This is off the destination's critical path but is a real cost of ADP that the destination-side numbers hide.
5. Derived, not measured end to end: replacing Groth16 (0.85 ms) by B1-sha (0.03 ms) inside the 2.1 ms B4 path would give about 1.3 ms, i.e. a negative ReuseGain of roughly -60% for this
   relation. That number assumes the shared components (header, MPT proofs) are unchanged; it needs the Besu + Fabric run to confirm.

## Disclosure (exact serialised sizes)
- B1 needs the destination to receive x: 256 fields, 866 bytes of JSON per decision, all derived from the source contract's code. Without a provisioned model it also needs
  W,b: 3,084 fields, 17,254 bytes (public here: the model is pinned by a hash, so it is only source-private if a real deployment keeps the model proprietary).
- ADP needs: proof 684 bytes + 7 public signals 332 bytes, and discloses 0 raw feature fields (featureHash and commitmentHash are digests).
- Not claimed: that x can or cannot be inverted to source code; that a real source would refuse to release x; anything about data other than this classifier.
- Feasibility: if the source will not release x, B1 is not executable (0 of 22), ADP is. That is a property of the design, not a measurement.

## What this means for the paper (must be reflected, see manuscript edits)
- The value of ADP for this workload is (a) confidentiality of x and (b) not requiring the destination to run or trust the source's pipeline, NOT lower destination-side latency.
- Latency claim to keep: B1^perf (full pipeline re-run) is 4.09 ms vs 2.10 ms; scope it explicitly as a full-pipeline comparison of different assurance.
- Latency claim to add: for the attested last-layer relation alone, re-execution in the destination runtime is cheaper than proof verification.
- Break-even (linear extrapolation of 0.33 ns per MAC, NOT measured beyond 3,072 MACs): re-execution would cost as much as Groth16 verification at roughly 2.6 million MACs.
  Larger relations move the balance toward proofs, which is where the SNARK approach pays off; none is measured here.

## Not done
- End to end on Besu + Fabric (peer/orderer/commit), the entry-point-level B1 on Fabric, and the shared components (header, MPT proofs), which need the networks.
- Any relation upstream of the last layer inside a proof.
- Amendment A1 was added after seeing run 1; it favours B1 and was disclosed as such.

## Amendments
- A1 (2026-09-28, after run 1): added B1-sha, because Poseidon(x) dominated B1-store and a reviewer could object that B1 need not use a circuit-friendly hash. Reported as an extra row; the primary B1-store row is unchanged.

## End-to-end on real Fabric with real Besu proofs (added 2026-09-28, protocol amendments A4-A6)
Setup: Besu main chain (4 validators) and Fabric `authcc` restarted; 22 fixtures registered on Besu (ML registry approvals, Poseidon and SHA-256 feature commitments in two
FeatureCommitRegistry contracts); chaincode v2.27/2.28 with additive entry points; fresh header and Merkle-Patricia proofs fetched at one block. Files: `s4_fair_b1/`
(s4_register_besu.js/.log, s4_fetch_proofs.js, s4_e2e.py, e2e_correctness.json, e2e_timing_run{1,2}.json, e2e_commit.json, analyze_e2e.py, analysis_e2e.json, go_src/s4_b1_entrypoints.go).

Correctness on the real networks: all four variants return the same class on 22/22 files (SAFE on the 6 files whose strict relation is satisfiable, REJECT on the other 16; note
3 of the 6 SAFE are files named "vuln": the source scanner passes them, a source-side scan false negative that ADP ports faithfully). 88/88 real submissions were endorsed by both
organizations, ordered and committed; the class returned equals the committed record on all 88.

In-chaincode endorsement time (ms; `time.Now()` deltas inside the entry point, so no CLI/gRPC noise), medians over 22 files x 30 reps, two runs:
| Variant | run 1 | run 2 |
|---|---|---|
| B4 (Groth16 v7, unchanged logic) | 2.030 (P95 3.17) | 2.016 (3.16) |
| B1-store (Poseidon x, stored model) | 2.817 (4.68) | 2.814 (4.70) |
| B1-nostore (+ model digest each call) | 21.68 (26.9) | 21.32 (26.3) |
| B1-sha (SHA-256 commitment, registry #2) | 1.035 (1.28) | 1.035 (1.27) |
Ratios vs B4: store 1.40-1.41, nostore 10.6-10.7, sha 0.51; all 22 files in the same direction (Wilcoxon p = 4.8e-7), CIs above.
Component medians (ms, run 2): the step that differs is `verify`: B4 0.873 (Groth16), store 1.622, nostore 20.1, sha 0.042. Shared steps are the same in all variants:
source-state verification 0.49-0.51, decision-exists 0.16, ledger clock 0.12-0.23, feature-commitment proof 0.03, PutState 0.15-0.18, classify ~0.

Submit-to-commit (88 real submissions, client wall time including the `peer` CLI, `--waitForEvent`): medians B4 2049 ms, B1-store 2050, B1-nostore 2070, B1-sha 2048.
The ordering service's 2 s batch timeout dominates; the endorsement difference (at most 1 ms between B4 and B1-sha, 0.8 ms to B1-store) is 0.05% of it.

Reading: (1) the offline component result is confirmed on the real path: Groth16 0.87 ms here vs 0.85 ms offline; B1-store 1.62 vs 1.61-1.64; B1-nostore 20.1-20.5 vs 19.8; B1-sha 0.04 vs 0.03.
(2) On the full endorsement path, ADP (B4) is 1.0 ms slower than the best B1 (B1-sha) and 0.8 ms faster than B1-store; the "-60%" derived estimate earlier was too small: the measured
ReuseGain against B1-sha is (1.035-2.016)/1.035 = -95%. (3) At submit-to-commit level all variants are indistinguishable (2.05 s), so none of this is a latency argument for or against ADP;
the argument for ADP here remains non-disclosure of x and independence from the source pipeline. (4) Client wall time includes the CLI start-up (same for all variants).
Limits: one network, one host (Besu VMs, Fabric containers and OpenStack share the machine), 22 files with data-independent cost, a single relation, single header; 2 replications
(one on each of two chaincode versions with identical timed code). The two Fabric peers are on the same host, so no network latency.
