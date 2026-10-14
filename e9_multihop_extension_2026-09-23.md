# E9 — Multi-hop preservation, extended to h ∈ {1,2,3,4,5,10} (2026-09-23)

Real result for `Manuscriptsp_m2_r3.tex`'s `subsec:results-multihop`
(Table `tab:multihop-security-results`) and the execution guide's section 11
run matrix. Previously the project's only multi-hop evidence was X5a/X5b at
n=2 hops on live Besu (a real light-client-verified counterexample where
naive pairwise-only acceptance says SAFE but composed/trustless
verification correctly says REJECT). The guide's own section 11.1
explicitly permits extending beyond real physical chains for this
experiment: "a 10-hop path may use containerized logical domains while
preserving independent governance/configuration semantics." This session
generalizes X5a's mechanism (composed vs. pairwise-local acceptance) to
h ∈ {1,2,3,4,5,10}, using the same real Keccak-256 hash-chaining primitive
(`ethers.keccak256`) already used for S19/A10, rather than repeating X5a's
literal light-client/ZK-proof machinery at every hop (which would require
h separate real blockchains, outside this session's scope).

## What was built

`~/pqc-hardhat/experiments/portability/e9_multihop_extension.js`. For each
h, a **preservation-control** path (every hop honestly forwards
context/obligation/assurance) and an **injected-loss** path at **every**
possible hop position 1..h (not just early/middle/late -- a full sweep, 31
real cases total) that silently drops the context property at exactly one
hop. Two verification strategies are computed for every case:

- **Pairwise-local**: each hop's own check only verifies state-binding
  continuity with its immediate predecessor and a minimum assurance floor
  -- it never examines context, matching the real X5a `boolean_path`'s
  behavior of accepting based on a narrow, hop-local guard.
- **Composed**: context/obligation/assurance are threaded through a real
  Keccak-256 hash chain at every hop; the destination's guard recomputes
  the chain and requires context=true, matching the real X5a
  `trustless_path`'s behavior of re-verifying the full accumulated state.

## Real results (31 cases: 6 preservation-control + 25 injected-loss)

| h | Preservation control (composed) | Injected-loss cases | Unsafe accepted (composed) | Unsafe accepted (pairwise-local) |
|---|---|---|---|---|
| 1 | SAFE | 1 | 0/1 | 1/1 |
| 2 | SAFE | 2 | 0/2 | 2/2 |
| 3 | SAFE | 3 | 0/3 | 3/3 |
| 4 | SAFE | 4 | 0/4 | 4/4 |
| 5 | SAFE | 5 | 0/5 | 5/5 |
| 10 | SAFE | 10 | 0/10 | 10/10 |

**Totals**: 0/6 preservation-control paths incorrectly rejected; **0/25**
injected-loss paths accepted end-to-end under the composed check (matches
`\ResMultiHopUnsafeAccept{0}`); **25/25** injected-loss paths incorrectly
accepted under the pairwise-local check, at every tested hop count and
every tested injection position. Raw output:
`evidence/e9_multihop_extension_result_2026-09-23.json`.

## Interpretation

- Hop count alone never causes rejection (preservation-control paths remain
  valid at every tested h, including h=10) -- the composed mechanism's cost
  is not "more hops means more false rejections."
- The composed check's detection is **position-invariant**: an injected
  loss at hop 1 of a 10-hop path is caught exactly as reliably as one at
  hop 10 -- generalizing X5a's single real data point (injection at hop 2
  of 2) to a full sweep.
- The pairwise-local failure is 100% at every h tested, including h=1. This
  is expected by construction (the pairwise-local guard never examines
  context, by design, matching the real X5a `boolean_path`'s narrow check),
  not a surprising or hop-count-dependent finding; it establishes that the
  gap pairwise-local composition misses is present even in the degenerate
  single-hop case, and multi-hop composition does not introduce or remove
  it -- what changes with h is only that the composed mechanism's guarantee
  is shown to hold uniformly across all tested path lengths and injection
  positions.

## Honest scope notes

- This is **not** a repeat of X5a's real on-chain light-client/ZK-proof
  verification at every hop -- it is a generalization of the same
  composed-vs-pairwise mechanism using real cryptographic hash-chaining, at
  a different (logical, not physical-blockchain) level, exactly as the
  execution guide's section 11.1 permits for this specific experiment.
  X5a/X5b remain the only fully-on-chain multi-hop evidence, at n=2.
  Extending the FULL X5a-style real on-chain verification (light client +
  ZK proof re-verification at every intermediate hop) to h>2 remains open.
  Evidence size per hop is small and real: each hop's commitment is exactly
  32 bytes (one Keccak-256 digest); cumulative chain size is O(1) at the
  destination (only the final commitment need be verified against the
  known origin), so evidence growth with h is negligible in this
  construction -- a real, if modest, input to `tab:perf-multihop`.
- Only the context property was tested as the injected-loss type (matching
  X5a's own real finding, which was a `semBound`/context-style loss); the
  guide's other loss types (obligation strip, assurance-claim loss, stale
  epoch, insufficient finality) were not separately swept for every h.
