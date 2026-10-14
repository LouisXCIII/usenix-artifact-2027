> **Superseded by the 2026-09-29 re-run** (amendment A1 in the protocol): the run described in this file used chaincode
> `authcc_2.29/seq41`, which -- unrelated to this experiment's design -- embedded the raw pinned model weights in the same
> package as every production entry point (found by S9 Measure C). The premise "no code, no features, no model weights"
> did not hold for the peer's installed package during this run, only for the ledger data this file's M2 search covers.
> The re-run on the cleaned package (`authcc_2.30/seq42`) matches every number below exactly (22/22, 30/30, 6/6, 12/12,
> 0 block hits); see `s7_rerun_v230_summary.json` and the paper text for the authoritative, premise-clean numbers. This
> file is kept as the first pass and for its positive control (block 3422), which the re-run reuses rather than repeats.

# S7 results: a destination that never receives the code or x (2026-09-28)

Protocol: `s7_no_code_destination_protocol_2026-09-28.md` (frozen before the first transaction; no amendments). Files: `s7_no_code_destination/`
(`s7_fetch.js`, `s7_run.py`, `s7_run.json`, `s7_run.log`, `s7_pilot.json`, `s7_blocks.json`, `s7_summary.json`, `besu_state.json`, `blocks_tmp/` = the 65 raw Fabric blocks searched).
Destination: the DEPLOYED, UNMODIFIED `SubmitDecisionTrustlessV7` on the real Fabric channel (authcc; endorsed by Org1 and Org2, ordered, committed).
Setup: `SetClock(now+300)`, one Besu header at block 540134 with fresh registry proofs; Fabric height 3400 -> 3465 (65 blocks). Run stamp 1790602218.

## What the destination was given
decisionID, sourceDomain, role, amount, sourceLimit, codeHash, the Groth16 proof and 7 public signals, one Besu header, the ML-registry proofs and the Poseidon
feature-commitment proof. It was never given x, the model weights, or code. (Modeled by not sending them; no real developer, no second deployment.)

## Results
- M1 correctness without x: **22/22** committed classes equal the S4 committed B4 classes (6 SAFE, 16 REJECT). Median client submit-to-commit 2,087 ms (ordering batch timeout).
- M2 non-receipt of x: (a) client side: an assertion that no argument contains x or its first 16 features held for all 22 treatment calls. (b) channel side: the raw bytes of all 65 blocks
  were searched for the first 16 quantised features of each of the 22 files and the first 8 entries of the pinned weights. **Positive control:** the deliberate `SubmitDecisionB1Store`
  invoke (which takes x) for 03_dos_safe.sol is in block 3422 and the search finds that file's x there. **Treatment blocks: 0 hits for x (22 files) and 0 for W in the other 64 blocks.**
- M3 bytes (median over 22 calls): treatment call 14,815 bytes of arguments; the same call in the B1-store form (x instead of proof and signals) 14,659 bytes; x alone 869 bytes.
  So the treatment call is NOT smaller (+156 bytes): the arguments are dominated by the Merkle-Patricia proofs and the header, common to both. The difference is what is disclosed, not the size.
- M4 negative controls, real invokes on the 6 files whose strict relation is satisfiable, no x sent:

| Category | n | accepted | outcome | one-sided 95% upper bound |
|---|---|---|---|---|
| N1 proof substitution (proof of file A, codeHash and registry proofs of file B) | 6 | 0 | 6 committed REJECT | 39.3% |
| N2 tampered proof (last digit of pi_a[0]) | 6 | 0 | 6 committed REJECT | 39.3% |
| N3 forged featureHash public signal | 6 | 0 | 6 committed REJECT | 39.3% |
| N4 feature-commitment proof of another file | 6 | 0 | 6 committed REJECT | 39.3% |
| N5 amount 600,000 > manager threshold 500,000, valid proof | 6 | 0 | 6 committed REJECT | 39.3% |
| N6 replay of an accepted decisionID | 6 | 0 | 6 refused at endorsement ("already recorded (replay rejected)"), stored records unchanged | 39.3% |

  Per-category bounds are wide because n = 6; the categories are different mechanisms and are not pooled. The pilot (queries only) ran first and no input crashed the chaincode.
- M5 contrasts with the deployed proof-only baseline `SubmitDecisionB0` (no codeHash binding, no role/amount check): **12/12 accepted** (6 proof-substitution inputs, 6 amount-widening inputs).
  So in this deployment class the destination-side binding and policy checks still decide the outcome; a destination that only checks proof and finality accepts them.
- C3 the re-evaluation baseline without x: a `SubmitDecisionB1Store` query with x omitted returns an endorsement error ("feature vector must have 256 entries"), i.e. it produces no decision.

## Reading
In a deployment where the destination receives neither the code nor x, the deployed direct-reuse entry point still classified all 22 files as in the with-x runs and rejected all 30 substitution,
tampering and widening controls and 6 replays, while the re-evaluation baseline cannot run and the proof-only baseline accepts the substitution and widening inputs. This is the one deployment class
in which not disclosing x has a benefit, and it is now tested on the real substrate.

## What this does NOT show
- No real developer withheld code, no contract was deployed elsewhere; "code not released" means "not sent". The compliance framing is a description of the workload, not a legal or business claim.
- The destination still trusts the source scanner's verdict (A1-A2); ADP transports its errors (three of the six accepted files are ones whose names mark them vulnerable).
- What the destination and a channel reader can infer from the published digests (codeHash, featureHash, commitment) is not analysed; the block search covers raw x and W only, so it does not show x cannot be inferred.
- Freshness/stale headers were not tested (the ledger clock cannot be moved without side effects); each negative category has n = 6; one host, one channel, one administrative domain.
- Latency is not a finding: the treatment call is the same entry point as B4 in S4; its in-chaincode cost was measured there (about 2.0 ms), and the client wall time here is the ordering timeout.
