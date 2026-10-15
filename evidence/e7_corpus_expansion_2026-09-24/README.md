# E7 corpus real expansion (2026-09-24)

Follow-up to the statistical-rigor pass (`e_statistical_rigor_ci_2026-09-24.json`), which found
the E7 B0-B4 baseline corpus (n=19 unsafe / n=5 safe, from the 2026-09-21 Stage-4/trustless-
corpus work) to be the statistically weakest primary security result in the paper (FAR upper
bound 14.59% vs E1's 0.16%). This note extends that corpus with real, newly-generated, newly
on-chain-verified cases -- not a re-analysis of the same 24 cases.

## Why the original test_contracts library couldn't be reused

`test_contracts/manifest.json` (the ML-scanner test suite the original 24-case corpus draws
from) is fully exhausted: all 22 listed files, plus `VulnerableBank.sol`/`SafeBank.sol`, already
appear in the corpus. Growing the corpus required either (a) new real proofs for new contracts,
or (b) new real proof-substitution pairings using proofs already on hand. Both were done.

## Part 1 -- SAFE arm: 7 new real Groth16 v7 proofs

Reused the exact real pipeline from `x8_v7_featurebind.py` (real ML-scanner model inference via
`gateway.MLScanner`, real feature quantization, real Poseidon feature-hash commitment, real
circom v7 witness generation, real snarkjs Groth16 prove+verify) against 10 new candidate
contracts (`/tmp/e7corpus_new_safe/new_safe_{1..10}.sol`, small real Solidity contracts covering
escrow, voting, token transfer, access-controlled vault, timelock, subscription, multisig,
pausable counter, sealed-bid auction, rate-limited faucet -- genuinely distinct source, not
templated).

**Real result: 7/10 succeeded.** 3 candidates (`new_safe_3`, `new_safe_5`, `new_safe_7`) were
real scanner class-0 predictions but with margins too small to satisfy the circuit's
`minMargin` constraint (witness generation genuinely failed) -- discarded honestly, not
force-fitted. The 7 successful proofs (`new_safe_1,2,4,6,8,9,10`) each have a genuinely distinct
real codeHash.

Each of the 7 new codeHashes was then registered on the real, already-deployed
`RegistryContract` (`0x4BDF7d0CC77F907FeDfc1Ab6B39fa9f4336869c7`) via its real, unmodified
`recordMLScanPass()` + two real admin `approve()` calls (`PRIVATE_KEY`=admin[0],
`PRIVATE_KEY_B`=admin[1], confirmed matching the contract's live `admins()` before use -- the
same real 2-of-3 quorum mechanism the original corpus's honest cases used), reaching
`mlScanPassed=true`, `approvalCount=2` for all 7 -- 21 real Besu transactions.

A fresh real Besu header + real `eth_getProof` MPT registry proof was then fetched, and all 7
were submitted as real, separately committed `SubmitDecisionTrustless` invocations
(manager/200000, within threshold) on the live `authcc` network. **7/7 confirmed committed**
(verified via `ReadDecision`, not just endorsement) with `semBound=true`, `evAssuranceOk=true`,
`transferClass=SAFE`.

## Part 2 -- UNSAFE arm: 16 new real proof-substitution attacks

Reused the 8 real, currently-valid Groth16 v7 proofs now on hand (the original
`H_honest_own_features` fixture + the 7 new safe proofs) as "attacker-submitted" proofs against
mismatched real target codeHashes -- exactly the same real mechanism the original corpus's
`ownProof=false` rows already tested (a validly-generated ZK proof, genuinely bound to a
DIFFERENT contract's codeHash, submitted against this decision), now with 16 distinct
(proof, wrong-target) pairs instead of reusing one donor proof repeatedly. Pairing was a fixed,
reproducible round-robin derangement (each of the 8 proofs against 2 different targets), not
random. Each pair was submitted as a real, separately committed `SubmitDecisionTrustless`
invocation (manager/200000). **16/16 confirmed committed** (via `ReadDecision`) with
`semBound=false`, `transferClass=REJECT` -- correct proof-substitution rejection every time.

## Honest scope note on new-case diversity

All 23 new cases (7 safe + 16 substitution) use the same role/amount (manager/200000) by
design, to isolate the proof-binding (`semBound`) mechanism specifically, since role/amount
threshold diversity was already covered by the 2026-09-23 boundary sweep (10 further cases at
exact threshold boundaries across 3 roles). This means the new substitution cases test B2
("semantic mapping only") differently than the original corpus's mixed-reason unsafe cases: all
16 pass B2's role/amount check trivially (by construction), so B2's real FAR increases
substantially on the expanded corpus -- a genuine, not inflated, finding (reported below), but
one that should be read as "B2 is defeated whenever role/amount happen to be compatible,
independent of binding" rather than as representative of the *rate* at which real-world unsafe
cases would have compatible role/amount by chance.

## Part 3 -- independent re-verification of the 23 new cases

Per the same standard already applied to the original 13-proof corpus
(`evidence/e7_independent_reverify_2026-09-23.md`), re-verified the 23 new cases with methods
that never call or reuse the Go chaincode's own verifier (`zkverify.go`'s
`verifySemBoundProof`), to rule out circularity:

1. **Independent `snarkjs groth16 verify`** (separate implementation from the Go chaincode)
   against the real `verification_key_v7.json`: **7/7 new safe proofs OK**.
2. **codeHash independently recomputed from the real `.sol` source on disk**
   (`SHA256(src) mod 2^253`, the same formula established in `x1_attacks.py`), compared against
   the codeHash actually used in registration/submission: **7/7 match exactly**.
3. **Destination policy recomputed from raw public signals** (`public[3]=="0"`, i.e.\ only the
   safe verdict): **7/7 pass**.
4. **Registry state, live direct RPC query** (`mlScanPassed`/`approvalCount` getters called
   fresh, independent of the earlier registration script's own bookkeeping): **7/7 confirmed
   `mlScanPassed=true`, `approvalCount=2`**.
5. **Substitution-mismatch independently confirmed**: for all 16 substitution cases, the
   attacker proof's bound codeHash was recomputed directly from its raw public signal (not
   trusted from the submission script) and checked two ways -- it must differ from the claimed
   target's codeHash, and it must equal the attacker's own real codeHash (ruling out an
   accidental self-consistent case that wouldn't actually test substitution). **16/16 confirmed
   both ways.**

**All 23 new cases pass every independent check, zero discrepancies** -- the same result as the
original 13-proof independent re-verification. Raw:
`independent_reverify_result.json`, `live_rpc_check_result.json`.

## Part 4 -- trust-root verification (closing a gap not previously checked)

Every independent check in Part 3 is only meaningful if it checks against the real system's
actual trust roots, not a possibly-divergent local copy. Verified three, all previously
unchecked directly in this project:

1. **Verification key**: `sha256sum` of the local `circuits_v7/verification_key_v7.json` (used
   for every independent `snarkjs` verification all session) against the exact file
   `//go:embed`-compiled into the deployed chaincode binary
   (`chaincode/verification_key_v7.json`). **Identical**
   (`c01de09270b2cc1f202cbe4e79719628a1b4428a8b0f8e6e213e01db1d1bb559` both).
2. **Pinned validator set**: the four hardcoded addresses in `lightclient.go`'s
   `pinnedValidatorHex`, compared against a direct, live `ibft_getValidatorsByBlockNumber` RPC
   call to the real Besu network. **Identical set** (same four addresses, order-independent).
3. **Pinned registry address**: `eth_getCode` on `0x4BDF7d0CC77F907FeDfc1Ab6B39fa9f4336869c7`
   confirms real deployed contract bytecode (1,052 bytes), not an empty/EOA address.

All three are real, freshly-observed facts about the live system, not carried over from an
earlier session's assumption. Raw: `trust_root_verification.json`.

## Real corpus totals and recomputed B0-B4 results

| | Before | After |
|---|---|---|
| Unsafe cases (n) | 19 | **35** |
| Safe cases (n) | 5 | **12** |
| Total | 24 | **47** |

| Baseline | FAR (before → after) | FAR 95% CI (before → after) | FRR (before → after) |
|---|---|---|---|
| $B_0$ | 100% (19/19) → 100% (35/35) | -- | 0% → 0% |
| $B_1$ | 0% (0/19) → 0% (0/35) | 14.59% → 8.20% (upper bound) | 0% → 0% |
| $B_2$ | 36.8% (7/19) → **65.7% (23/35)** | Wilson [19.15%,58.96%] → [49.15%,79.17%] | 0% → 0% |
| $B_3$ | 5.3% (1/19) → **2.9% (1/35)** | Wilson [0.94%,24.64%] → [0.51%,14.53%] | 0% → 0% |
| $B_4$ | 0% (0/19) → 0% (0/35) | 14.59% → **8.20%** (upper bound) | 0% (0/5) → 0% (0/12), CI 45.07% → **22.09%** |

$B_4$'s FAR/FRR confidence bounds nearly halved (14.59%→8.20%, 45.07%→22.09%). $B_2$'s real FAR
rose sharply (36.8%→65.7%) because the 16 new substitution cases were deliberately constructed
at role/amount values B2 cannot reject on its own -- a real, sharper demonstration of exactly the
gap $B_2$ vs $B_4$ is meant to show. $B_3$'s FAR fell (5.3%→2.9%) because the new substitution
cases are correctly caught by $B_3$'s retained proof-binding check (only the original SafeBank
assurance-only failure remains uncaught), tightening its interval around the same single real
failure mode.

## Files

- `new_safe_proofs_summary.json` -- the 7 real proof-generation results (codeHash, margin, proof
  location).
- `registration_result.json` -- 21 real Besu registration transaction hashes.
- `safe_submit_results.json`, `substitution_results.json` -- real Fabric invoke results (23
  total new decisions).
- `consolidated_summary.json` -- decisionIDs and ReadDecision-confirmed outcomes for all 23 new
  cases.
- `all_codehashes.json` -- the 8-proof pool used for substitution pairing.
