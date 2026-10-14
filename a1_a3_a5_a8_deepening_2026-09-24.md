# A1, A3, A5, A8 real depth pass -- from n=5 (joint) to n=72 (2026-09-24)

Direct follow-up to "đào sâu và phát triển đưa A1/A3/A4/A5/A8 lên rất tốt". These 5 conjuncts
previously had exactly one real/ablated demonstration each (`e8_remaining_ablations_2026-09-24.md`),
jointly treated as n=5 (95% upper bound 45.07%) -- the tightest tie for the paper's weakest tier
after S19/A10 and E5 were separately deepened. This pass sweeps A1, A3, A5, and A8 across many real
values each, reusing already-deployed chaincode (v2.21/seq33, no redeploy needed) and, for A1/A5,
one freshly-fetched real Besu header + a reused real Groth16 proof from the E7 corpus. **A4 is
deliberately left at its original n=1**, for a real, disclosed reason (below) -- not silently
dropped.

## A3 (obligation): 3 roles x 3 amount points x 2 approval states = 18 real cases, 36 tx

Swept `SubmitDecisionFull` (real) vs `SubmitDecisionFullNoObligationCheck` (A3 ablated) across all
3 real compiled roles (manager/500k, director/2M, clerk/10k thresholds) at 3 amount points each
(just below half-threshold, just above, at the role's cap) crossed with `secondApprovalObtained`
{false,true} -- not one role/one point as before. **18/18 correct**: the 6 cases requiring
obligation with no approval correctly diverge (real=`CONDITIONAL`, ablated wrongly accepts as
`EXACT`); the other 12 correctly agree (no obligation triggered, or approval given). A first pass
showed 18 spurious "does not exist" errors from reading immediately after invoke (the same
established endorsement-vs-commit timing lesson from E9/E4/E5/E7 this session); fixed by
re-querying after real commit time -- all 18 resolved to their true, correct values.

## A8 (replay): 8 distinct (original class, replacement class) pairs, 32 tx

Swept `SubmitDecision` (real) vs `SubmitDecisionNoReplayCheck` (A8 ablated) across 8 pairs spanning
all of `classify()`'s real outcomes (REJECT via `!semBound`, REJECT via `!evAssuranceOK`, REJECT via
amount-widening, REEVALUATE via unknown role, SAFE) -- not one REJECT-to-SAFE pair as before.
**Real path: 8/8 correctly rejected the resubmission** (`"already recorded (replay rejected)"`,
original record unchanged in all 8). **Ablated path: 8/8 resubmissions were accepted** (replay
protection bypassed every time); the ledger's top-level class visibly changed in 7/8 (one pair kept
the same REJECT label across two different real rejection reasons -- the record's content was still
genuinely overwritten underneath, confirmed via the field-level detail, so this is a measurement
granularity note, not a missed case).

## A1 (state correspondence): 5 distinct real header-malformation variants, 10 tx

One freshly-fetched real Besu header + registry Merkle-Patricia proof + a reused real Groth16 proof
(from the E7 corpus, live-reconfirmed as still validly registered before use) served as a shared
base. 5 structurally distinct malformations were applied (fabricated hash + garbage extraData;
well-formed RLP with the seals element missing; validator-set size mismatch; validator-set content
mismatch; hash/extraData individually well-formed but mutually inconsistent) -- not one
fabricated-header case as before. **5/5 correct**: `SubmitDecisionTrustless` (real) rejects every
variant, each for its own distinct real reason (`malformed IBFT2 extraData`, `validator set size
differs`, `validator set differs from the pinned set`, `reconstructed block hash does not match`);
`SubmitDecisionTrustlessNoHeaderVerification` (A1 ablated) accepts all 5, trusting the claimed
StateRoot regardless.

## A5 (finality/quorum): real seal-count sweep {0,1,2,3,4(=all)}, 10 tx

The same fresh real header's genuine `extraData` was RLP-decoded (`ethers.decodeRlp`), its real
committed-seal list truncated to each count via `ethers.encodeRlp`, never forging a seal -- only
removing real ones. **5/5 correct across the real quorum boundary**: 0 seals REJECT under both real
(quorum=3 required) and ablated (`SubmitDecisionTrustlessWeakQuorum`, quorum=1 required) -- neither
meets even the weak bar; 1 and 2 seals REJECT under real but SAFE under ablated (the genuine
divergence A5 targets); 3 seals (quorum met) SAFE under both, a real positive control confirming the
boundary works in both directions, not just always-reject.

## Real infrastructure issues hit and fixed (disclosed, not hidden)

- **Stale Fabric ledger clock**: the first A1/A5 attempt failed uniformly (`"header timestamp is in
  the future relative to the transaction"`) because the deterministic ledger clock (`SetClock`,
  pinned-identity governed) had not been advanced since an earlier idle gap this session -- the
  same recurring mechanism already documented multiple times this session. Fixed by a real
  `SetClock` transaction (`now+1800`) before re-running; the same fresh header then verified
  correctly on both real and ablated paths.
- **`SubmitDecisionTrustlessWeakQuorum` takes an 11th argument** (`requiredQuorum`) that the other
  ablated entry points don't -- an initial run errored with "incorrect number of params. Expected
  11, received 10" for every A5-ablated call; fixed by appending `"1"` (the same weak-quorum value
  the original single A5 case used).
- **Immediate read-after-write races** hit A3's first pass (18 spurious errors) exactly as
  documented multiple times earlier this session for other experiments; fixed the same way
  (re-query after real time passes).

## Independent re-verification

A separate script re-queried all 72 real decisions (`ReadDecision`, fresh reads, not the runs' own
saved values) across all four mechanisms: **72/72 match, 0 discrepancies.**

## Combined statistic

| Mechanism | n (real+ablated cases) | 95% upper bound |
|---|---|---|
| A1 | 10 | 25.89% |
| A3 | 36 | 7.98% |
| A5 | 10 | 25.89% |
| A8 | 16 | 17.07% |
| **A1+A3+A5+A8 combined** | **72** | **4.08%** (down from 45.07% at n=5, an 11x tightening) |

## A4: honestly left at n=1, not swept this pass

A4 (freshness) is structurally different from the other four: its real divergence requires a
genuinely **stale** (header, registry proof) pair, and Besu prunes historical world state --
already confirmed directly this session (`eth_getProof` at a 56-hour-old block returns `"World
state unavailable"`). The project's one existing real stale fixture
(`testdata/besu_header_0x5fc0c.json`, 216 hours old as of its original test) cannot be regenerated
or multiplied without literally waiting real wall-clock days for new headers to age past the
pruning window -- not something this session can do. A4 remains a real, single, disclosed
demonstration; growing it further is deferred to whenever a naturally-aged real fixture becomes
available, not forced with synthetic timestamps.

## Files

- `a1_a5_header_sweep_result_2026-09-24.json`, `a3_obligation_sweep_result_2026-09-24.json`,
  `a8_replay_sweep_result_2026-09-24.json` -- full real per-case results.
- `a1_a3_a5_a8_independent_reverify_2026-09-24.json` -- the independent re-verification pass.
