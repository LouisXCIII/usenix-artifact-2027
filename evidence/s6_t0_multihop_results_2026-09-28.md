# S6 results: origin-timestamp (t0) composition across real on-chain hops (2026-09-28)

Protocol: `s6_t0_multihop_protocol_2026-09-28.md` (frozen before the run; amendment A1 also before the run). Files: `s6/` (`multihop_t0.go`, `s6_t0_multihop.py`, `s6_results.json`, `s6_run.log`,
`s6_results_smoke_neg.json` = a first negative-control-only smoke run).
Chaincode: v2.29 / sequence 41, additive file `multihop_t0.go` only (`authchaincode.go` sha256 e50ed4b1... unchanged; no existing entry point changed).

## What was built
`SubmitHopDecisionT0` (composed): OriginT0 is recorded at hop 0 from the LEDGER clock (`ledgerNow`, advanced only by the pinned clock authority; not `GetTxTimestamp`, which the submitter
chooses) and COPIED by every later hop from the previous hop's ledger record; there is no timestamp argument. A hop is SAFE iff the previous composed flag AND the hop's context AND
(ledgerNow - OriginT0 <= DeltaDecision = 120 s); a REJECT propagates to all later hops. `SubmitHopDecisionT0Local` is the ablation: same context propagation and state-binding
continuity, but freshness is per hop (gap since the previous hop <= 120 s), i.e. the pairwise-local behaviour. Both return only the class, so both endorsing peers agree.

## Results (real Fabric transactions; 42 path records, 184 hop transactions, 0 transaction errors)
- Every path's ledger classes equal the model prediction computed from the ages recorded on the ledger (42/42), and OriginT0 is identical in every hop record of every path (42/42).
- Fresh paths (back to back; h = 1, 2, 3, 4, 5, 10; 2 replicas per variant, 1 for h = 10): both variants SAFE at every hop (ages up to 56 s at h = 10, well inside 120 s).
- Context loss at hop 2 of a 4-hop path: REJECT from hop 2 on in both variants (existing behaviour preserved), 4/4 records.
- Stale paths (one hop every 50 s on a shared tick schedule; per-hop gap 50 s, cumulative age 0, 50, 100, 150, ...): h = 3 (control, ages up to 100 s): both variants SAFE at every hop.
  For h = 4, 5, 10 (6 paths per variant): composed-t0 returns SAFE at hops 0-2 and REJECT from hop 3 (age 150 s) on, so the end-to-end decision is REJECT in **6/6**; the pairwise-local variant
  returns SAFE at every hop, so it accepts **6/6** paths whose total age exceeds the destination's bound (composed accepted 0/6; one-sided 95% Clopper-Pearson upper bound 39.3%, small n).
- Negative controls (both variants): skipping a hop and replaying a hop are both refused (2/2 each), with the expected error messages.

## Reading
The disclosed gap is closed on the real network: end-to-end freshness composition with t0 is implemented and exercised, and the model's claim that per-hop freshness is insufficient is confirmed by a
constructed counterexample on real transactions. Freshly submitted 10-hop paths reached ages up to 56 s (with one SetClock transaction per tick), so back-to-back hops do accumulate tens of seconds,
consistent with the manuscript's statement (about 2.2 s per hop for the hop transaction alone).

## Limits (must be stated in the paper)
- Time is the ledger clock, set by the clock authority (Org1 admin) to the real wall clock before each tick. A wrong clock defeats every clock-based check in this system; the authority's trust is
  assumed, not tested. Time sources that the submitter chooses (GetTxTimestamp) are deliberately not used.
- The stale paths are constructed (a tick every 50 s) to exceed the bound while every per-hop gap stays inside it; Delta = 120 s and the 50 s spacing are illustrative, not realistic cross-chain figures.
- One constant Delta is used for every hop; per-hop bounds Delta^(j) that differ between hops are not tested.
- Hops are separate transactions on one Fabric channel carrying the context flag and timestamps only (as in the earlier real hop experiments); no Besu light client or ZK evidence per hop.
- Replicas are deterministic repeats, not independent samples; 6 stale paths per variant support a mechanism statement, not a rate.
