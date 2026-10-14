# E9, real on-chain multi-hop verification, h in {1,3,4,5,10} (2026-09-24)

`evidence/e9_multihop_extension_2026-09-23.md`'s "Honest scope notes"
disclosed the project's most significant remaining E9 gap: X5a/X5b (h=2)
remained the only fully-on-chain multi-hop evidence; h in {1,3,4,5,10} had
only been generalized via a JS/Keccak256 hash-chain **simulation**, not
real Fabric transactions. This session closes that gap for real.

## What was built

`~/fabric-network/authcc/chaincode/multihop_real.go` (new, additive, no
existing entry point modified): every hop of a multi-hop path is now its
own genuine, separate Fabric transaction. Each hop reads the **previous
hop's real committed record** via `GetState` before writing its own --
real state-binding, not a simulated hash chain.

- `SubmitHopDecision` (composed/real): the accumulated
  context-preservation flag is the logical AND of every hop's own
  `contextAtHop` value, genuinely propagated hop-by-hop through real
  on-chain reads.
- `SubmitHopDecisionPairwiseLocal` (ablated): each hop still genuinely
  requires the previous hop to exist on-chain (real state-binding
  continuity is not removed), but never examines the accumulated context
  truth -- exactly matching X5a's real `boolean_path` behavior.

Deployed as chaincode v2.15/seq27 (the known chaincode-build infra bug
recurred a 6th time this session and was fixed proactively with the
established procedure before testing).

## A real bug found and fixed during construction

The first run attempt failed broadly: every hop after the first errored
with `"previous hop N not found -- real state-binding continuity broken"`.
Investigated rather than worked around: `peer chaincode invoke` returns
after endorsement/broadcast, **not** after the transaction actually
commits to a block -- submitting hop N+1 immediately after hop N's invoke
call means hop N+1's `GetState` read genuinely does not yet see hop N's
write, because it has not committed yet. This is a real, structural
sequential dependency (hop N+1 requires hop N's committed state to exist),
not a race for MVCC to resolve. Fixed by polling `ReadHop` (a real
on-chain query) after every hop until it is genuinely visible in committed
world state before submitting the next hop.

## Real results (56 real on-chain transactions/checks, h in {1,3,4,5,10})

For each h: 1 preservation-control path (every hop honest) + h
injected-loss paths (one per injection position 0..h-1), each run under
both `SubmitHopDecision` (composed) and `SubmitHopDecisionPairwiseLocal`
(pairwise-local), every hop a real, separate, sequentially-committed
transaction.

| h | Preservation control | Injected-loss positions tested | Composed correctly REJECTs | Pairwise-local incorrectly accepts |
|---|---|---|---|---|
| 1 | SAFE/SAFE | 1 | 1/1 | 1/1 |
| 3 | SAFE/SAFE | 3 | 3/3 | 3/3 |
| 4 | SAFE/SAFE | 4 | 4/4 | 4/4 |
| 5 | SAFE/SAFE | 5 | 5/5 | 5/5 |
| 10 | SAFE/SAFE | 10 | 10/10 | 10/10 |

**56/56 real transactions/checks matched expectation.** Every
preservation-control path is accepted under both designs (hop count alone
never causes a real rejection, up to real h=10); every injected-loss path,
at every real position tested including both endpoints and every interior
position, is correctly rejected by the composed design and incorrectly
accepted by the pairwise-local design -- position-invariant detection,
now demonstrated with genuine on-chain transactions rather than a
hash-chain simulation. Raw output:
`evidence/e9_real_multihop_result_2026-09-24.json`.

## What this closes, precisely

This does not repeat X5a's full real light-client + ZK-proof
re-verification machinery at every hop (that would require h separate
real blockchains, which the execution guide's section 11.1 explicitly
does not require for this experiment). What it does close is the gap
between "simulated in JS" and "real Fabric transactions": every hop above
is now a genuine, separately-committed, sequentially-dependent on-chain
transaction on the live `authcc` network, with real `GetState`-based
state-binding between hops -- not a JS hash-chain standing in for one.

## Still open, disclosed honestly

- The full X5a-style real light-client + ZK-proof re-verification at
  every intermediate hop, for h>2, remains open (a materially larger
  undertaking, as before).
- Only the context property was tested as the injected-loss type here, as
  in the original extension; other loss types (obligation strip,
  assurance-claim loss, stale epoch, insufficient finality) were not
  separately swept for every h.
- Each hop currently requires waiting for real commitment before the next
  can be submitted (a genuine sequential-dependency cost of real
  on-chain multi-hop verification, not an artifact of this test) --
  disclosed as a real, inherent latency property of this design, relevant
  to `tab:perf-multihop`.
