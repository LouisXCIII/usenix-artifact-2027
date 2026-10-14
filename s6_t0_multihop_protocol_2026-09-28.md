# S6 protocol: origin-timestamp (t0) composition across real on-chain hops

Written 2026-09-28 BEFORE any S6 run. Amendments are appended below with dates.

## Why
The manuscript (Section on multi-hop, Appendix E.3) requires that freshness at hop j be evaluated against the ORIGIN timestamp t0 carried in the path state, so that
B_n's bound Delta_decision is checked against the total path age, and it discloses that the real deployed multi-hop chaincode (`SubmitHopDecision`) propagates only a boolean
context flag and no t0. S6 closes that gap on the real Fabric network and tests the claim that per-hop (pairwise-local) freshness is NOT sufficient for end-to-end freshness.

## What is built (additive, new file, no existing entry point changed)
- `SubmitHopDecisionT0(chainID, hopIndex, contextAtHop)`: composed with t0. Time source is the ledger clock (`ledgerNow`, advanced only by the pinned clock authority;
  NOT `GetTxTimestamp`, which the submitter chooses, see clock.go/X4). Hop 0 records OriginT0 = ledgerNow. Hop k>0 reads the previous hop record from the ledger, COPIES its
  OriginT0 (there is no caller-supplied timestamp argument), and computes age = ledgerNow - OriginT0. Class = SAFE iff previous composed flag AND contextAtHop AND age <= DeltaDecision;
  otherwise REJECT, and the REJECT is propagated (composed flag false) to all later hops.
- `SubmitHopDecisionT0Local`: identical (same context propagation, same state-binding continuity) EXCEPT freshness is measured per hop, against the previous hop's ledger time
  (gap = ledgerNow - prev.HopTime <= DeltaDecision), which is the "pairwise-local" behaviour the model says is insufficient.
- Both return only the class string (deterministic across the two endorsing peers). `ReadHopT0` returns a stored record.
- DeltaDecision is a chaincode constant of the destination (120 s), not a caller argument.

## Experiment (real Fabric transactions, one channel, separate transaction per hop, as in the earlier real multi-hop runs)
Before every tick the clock authority (Org1 admin) sets the ledger clock to the real wall clock; then every replica of both variants submits its hop at that ledger time.
- Fresh paths (hops submitted back to back): h in {1, 2, 3, 4, 5, 10}, 2 replicas per variant. Expected: both variants SAFE at every hop (total real age well below 120 s).
- Stale paths (one hop every 50 s of real time, so each per-hop gap 50 s < 120 s but cumulative age 0, 50, 100, 150, ... s): h in {3, 4, 5, 10}, 3 replicas per variant.
  Expected: h = 3 is a control (ages 0, 50, 100 s): both variants SAFE. For h in {4, 5, 10}: composed-t0 returns SAFE at hops 0-2 and REJECT from hop 3 on (age 150 s > 120 s);
  the local variant returns SAFE at every hop. The end-to-end decision is the class of the last hop.
- Checks on every path: all hop records carry the same OriginT0 as hop 0; recorded ages equal ledger-clock differences.
- Negative controls (both variants): skipping a hop and replaying a hop are refused.
- Injected context loss (contextAtHop=false at one hop) still yields REJECT in both variants that propagate context (regression check of the existing behaviour).

## What is reported / not claimed
Reported: per (variant, scenario) the class at every hop, the end-to-end class, the number of stale-beyond-Delta paths accepted end to end by each variant, with the Clopper-Pearson bound
if zero. Not claimed: that Delta = 120 s or the 50 s spacing is realistic; anything about per-hop bounds Delta^(j) that differ between hops (a single constant is used); anything about
the Besu light client or ZK evidence per hop (hops carry the context flag and the timestamps only, like the earlier real hop experiments); trust in the ledger clock authority is
assumed, not tested (an authority that sets a false clock defeats it, as for every clock-based check in this system). Real per-hop latency is not the subject.

## Amendments
(none yet)

- A1 (2026-09-28, BEFORE any S6 run): to keep every fresh path well inside Delta while several replicas share a tick, fresh paths use 2 replicas per variant except h = 10 (1 replica);
  stale paths use 2 replicas per variant (not 3) because all stale scenarios run on one shared 50 s tick schedule (h = 3, 4, 5, 10 together, about 34 s of transactions per tick).
  The expectations are evaluated from the ages recorded on the ledger (not from wall-clock assumptions): composed-t0 SAFE iff context preserved and age since origin <= 120 s at every
  hop so far; local SAFE iff context preserved and each gap <= 120 s. Mismatches between the ledger's class and this model prediction are reported.
- Deployed for S6: chaincode v2.29 / sequence 41 (adds multihop_t0.go only; authchaincode.go sha256 e50ed4b1... unchanged).
