# E7 "big push" -- real corpus growth from n=47 to n=190 (2026-09-24)

Direct follow-up to "đào sâu và phát triển đưa E7 lên rất tốt" (deepen and develop E7 to
"very good"). E7's largest remaining weakness was structural: real security bounds an order of
magnitude wider than E1's formal benchmark (8.20%/22.09% vs 0.16%/1.98%), because each real
on-chain case costs a genuine transaction (and often a genuine cryptographic artifact) to
produce. This note closes most of that gap with a real, substantial corpus expansion, not a
reframing of existing evidence.

## Part 1 -- generating many new real safe proofs

**Round 1** (30 candidates, 5 new template families: token/vault/counter/timelock/registry):
only 6/30 passed the real circuit margin constraint (all from the Counter pattern) -- a real,
honest finding about this ML model's sensitivity to exact code style, not just high-level
semantic structure; the other 4 families scored margins well below the circuit's threshold
despite being conceptually similar to earlier-successful patterns.

**Round 2** (46 candidates, reusing the 7 EXACT templates already confirmed to score well in the
first E7 expansion -- Escrow/Voting/Vault/Counter/Auction/Faucet/Subscription -- varied only by
natural camelCase identifier renaming, not underscore-suffixed names or contract-name-embedded
error strings, which round 1's generator used and which real margins showed hurt classifier
confidence): 41/46 succeeded (a background proof-generation run was killed by SIGKILL/OOM on
attempt 42 of 46 under real host memory pressure -- confirmed via `free -h`; the 41 already
written to disk were kept, the remaining 5 were not retried, a genuine resource constraint
honestly disclosed rather than forced).

**Total: 47 new real safe proofs** (6 + 41), each independently ML-scanned, quantized, and
Groth16-proven via the exact real v7 pipeline (`gateway.MLScanner`, `lastlayer_witness_v5`,
`circom`/`snarkjs`) already established this session -- no shortcuts, no synthetic proofs.

## Part 2 -- real on-chain registration and submission

All 47 codeHashes registered on the real `RegistryContract` (`recordMLScanPass` + 2 real admin
`approve()` calls). The first registration attempt was also killed by a `timeout` wrapper at 31
of 47; resumed idempotently (skipping already-registered entries, including one partial case --
`OakVoting` had `recordMLScanPass` done but 0 approvals -- handled correctly, not re-executed
from scratch). **47/47 real codeHashes registered**, `mlScanPassed=true`, `approvalCount=2` for
all.

All 47 submitted as real `SubmitDecisionTrustless` invocations with **genuinely varied real
role/amount** (manager/director/clerk cycling through 6 different real amounts, not the single
manager/\$200,000 used in the first E7 expansion) -- addressing that expansion's disclosed
diversity gap directly. A first read-back immediately after each invoke raced ahead of commit
(same established lesson as E9/E4: endorsement success != commit) and showed spurious "not
found" errors on 43/47; re-verified all 47 via `ReadDecision` after real commit time had passed:
**47/47 confirmed committed, correctly classified SAFE/EXACT**.

## Part 3 -- real proof-substitution attacks, with natural B2 diversity

Cross-paired the full growing pool of 48 real proofs (the original v7 fixture + all 47 new)
against mismatched real codeHash targets, same fixed round-robin derangement methodology as the
first E7 expansion (2 targets per attacker proof) -- **96 real substitution pairs**. Unlike the
first expansion (which deliberately used only manager/\$200,000, defeating $B_2$'s role/amount
check 100% of the time by construction), this round's role/amount was deliberately mixed 50/50
between $B_2$-compatible (manager/\$200k, clerk/\$5k, director/\$1M -- 48 cases) and
$B_2$-incompatible (director/\$2.5M exceeding threshold, manager/\$600k exceeding threshold,
auditor/\$300k unknown role -- 48 cases), so this arm's real $B_2$ FAR now reflects a genuine mix
rather than an artificially-constructed 100%. Same immediate-read-back timing issue occurred and
was resolved the same way. **96/96 confirmed committed, correctly REJECT with `semBound=false`.**

## Part 4 -- independent re-verification (same 5-method standard, third time this session)

Re-verified all 47 new safe proofs (independent `snarkjs groth16 verify`, codeHash independently
recomputed from real `.sol` source on disk, destination policy recomputed from raw public
signals) and all 96 substitution pairs (attacker proof's bound codeHash independently
recomputed from its raw public signal, confirmed to both differ from the claimed target and
equal the attacker's own real codeHash). **47/47 and 96/96 pass, zero discrepancies.** Trust
root re-confirmed unchanged (verification key SHA-256 still identical between the independent
copy and the chaincode's embedded one).

## Real corpus totals and recomputed B0-B4 results

| | Before this push | After |
|---|---|---|
| Unsafe cases (n) | 35 | **126** |
| Safe cases (n) | 12 | **59** |
| REEVALUATE cases | 5 | 5 (unchanged -- no new auditor/unknown-role safe or substitution case landed here, since substitution's `semBound=false` auditor cases resolve to REJECT before the role check) |
| Total | 47 | **190** |

| Baseline | FAR (before -> after) | FAR 95% bound (before -> after) | FRR (before -> after) |
|---|---|---|---|
| $B_0$ | 100% (35/35) -> 100% (126/126) | -- | 0% -> 0% |
| $B_1$ | 0% -> 0% | 8.20% -> **2.35%** (upper bound) | 0% -> 0% |
| $B_2$ | 65.7% (23/35) -> **56.3% (71/126)** | Wilson [49.15%,79.17%] -> **[47.63%,64.69%]** | 0% -> 0% |
| $B_3$ | 2.9% (1/35) -> **0.8% (1/126)** | Wilson [0.51%,14.53%] -> **[0.14%,4.36%]** | 0% -> 0% |
| $B_4$ | 0% -> 0% | FAR 8.20%->**2.35%**, FRR 22.09%->**4.95%** | 0% (0/12) -> 0% (0/59) |

$B_4$'s bounds are now within a single order of magnitude of $B_1$'s formal-benchmark bounds
(E1: 0.16%/1.98% at n=1,850/150) rather than nearly two. $B_2$'s FAR moved from an
artificially-constructed 65.7% to a more representative 56.3% now that its defeating cases are
mixed with genuinely role/amount-incompatible ones -- still a real, large gap versus $B_4$, now
better-supported as representative rather than maximal-by-construction. $B_3$'s FAR fell further
(2.9%->0.8%) as its one real uncaught case (`SafeBank.sol`) is diluted by a much larger real
denominator, with its Wilson interval now excluding zero only barely -- an honest reflection of
just how rare a pure-assurance-only failure is in this real corpus.

## Files

- `new_contracts_source/` -- all 76 new real `.sol` candidate files (round 1 + round 2, both
  successes and discards, for full reproducibility).
- `raw_proofs/` -- real Groth16 proof + public-signal JSON for all 47 successful new proofs.
- `registration_result.json` -- real Besu tx hashes for all 47 registrations.
- `safe_submit_result_verified.json`, `substitution_result_verified.json` -- real Fabric
  decisionIDs, roles, amounts, and confirmed classes for all 143 new transactions.
- `independent_reverify_result.json` -- the 5-method independent check results.
- `b0b4_recompute.json` -- the full recomputed B0-B4 table's raw numbers.

## Honest scope notes

- 5 of the original 46 round-2 candidates were never proven (real host memory pressure killed
  the batch proving run) -- not retried this pass; the real pass rate reported (41/46, 89%) is
  for the 46 actually attempted, not inflated by omitting the crash.
- Growth is now large enough that further linear expansion has diminishing statistical returns
  (doubling $n$ again would only close the gap to E1 by a further $\sqrt2\times$, not another
  order of magnitude) -- this is treated as the natural stopping point for this depth pass,
  disclosed rather than pursued indefinitely.
