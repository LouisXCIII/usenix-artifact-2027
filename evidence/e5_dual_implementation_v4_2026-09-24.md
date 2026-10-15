# E5 further depth pass -- dual-implementation cross-verification + real corpus growth n=7 -> n=31 (2026-09-24)

Direct follow-up to "đào sâu và phát triển đưa E5 lên rất tốt" (deepen and develop E5 to "very
good"). A prior audit this session had briefly mis-stated E5 as still n=2/77.6% -- re-checked
against the actual current manuscript and found E5 had already been grown to n=7 (95% upper bound
34.82%) via `e5_independent_verifier_attestation_2026-09-23.md`, which closed the self-attestation
gap (`offChainVerified` no longer trusted from the submitter alone) but left one honestly-disclosed
weakness: both the submitter's self-check and the independent oracle attestation still ultimately
relied on a single ML-DSA-65 implementation (Liboqs 0.15.0, C). A bug shared by that one codebase
would not have been caught by either check. This pass closes that gap for real and grows the corpus
further.

## Part 1 -- a genuinely independent second implementation

Searched the host for a second, independently-authored ML-DSA-65 (Dilithium3) implementation
rather than writing one from scratch. Found `github.com/cloudflare/circl v1.6.3`
(`sign/mldsa/mldsa65`), a pure-Go implementation of FIPS 204 ML-DSA-65 already vendored on this
machine (`~/go/pkg/mod/`) -- a different team (Cloudflare vs. Open Quantum Safe), different
language (Go vs. C), different codebase entirely from the project's existing Liboqs/JNI verifier,
with no shared code path.

**Compatibility was not assumed -- checked directly**: `mldsa65.PublicKeySize`/`PrivateKeySize`/
`SignatureSize` (1952/4032/3309 bytes) were confirmed byte-identical to Liboqs's
`DilithiumNative.PUB_KEY_SIZE`/`SEC_KEY_SIZE`/`SIG_SIZE` constants. A small standalone Go CLI,
`mldsa_verify_circl` (new file, no existing file touched), was built and run against a real,
freshly Liboqs-generated keypair+signature: CIRCL independently reported **VALID** on the genuine
signature and **INVALID** on a tampered copy (one byte flipped) -- the two independent
implementations agree on real data before anything further was built on top of this.

## Part 2 -- PQCTransferV4: requiring both implementations to agree

`PQCTransferV4.sol` (new, additive; `PQCTransferV2`/`V3` and their existing evidence are
untouched) requires attestations from **two separate `PQCVerifierOracle` instances**, each fed by
a different real identity running a different implementation:
- the **existing, already-deployed** liboqs-side oracle (reused, not redeployed -- its 5 prior
  S1--S5 attestations stand unchanged), fed by the existing `PRIVATE_KEY_C` identity;
- a **new** circl-side oracle, fed by a **freshly generated, newly funded** 4th real identity
  (funded with a genuine 1.0 ETH on-chain transfer from Bank-A, not a pre-set `.env` key).

`submitTransfer` grants `VERIFIED` only if **both** oracles independently attest `true`; any
disagreement between the two real implementations is recorded on-chain (`implementationsDisagree`)
rather than silently resolved either way.

## Part 3 -- real corpus growth, 12 new keypairs x 2 cases each

12 fresh real Liboqs ML-DSA-65 keypairs were generated. For each: a genuine signature over a
distinct real payload (valid case) and a tampered copy with the flipped-byte position varied per
pair (tampered case) were **both independently verified by Liboqs AND CIRCL** before either oracle
attested, then submitted as real `PQCTransferV4` transactions (Bank-A sender, its registered key
rotated to the fresh keypair each round; Bank-B fixed recipient).

**24/24 new real on-chain cases correct** (12/12 valid -> `VERIFIED`, 12/12 tampered ->
`REJECTED`), **0 cross-implementation disagreements** across all 24 cases (Liboqs and CIRCL agreed
on every single real signature, valid and tampered alike), **0 unsafe accepts**.

## Part 4 -- independent re-verification (bypassing the original run's own event logs)

A separate script re-fetched each of the 24 transaction receipts fresh from the chain, re-parsed
the `TransferSubmitted` event to recover each `transferId`, then called `getTransfer` again --
reading the contract's **persisted state**, not the value returned at mine-time -- and compared
against the original run's recorded result. **24/24 match, 0 discrepancies, 0 unsafe accepts, 0
disagreements confirmed independently.**

## Combined real E5 corpus and updated statistic

| | Before this pass | After |
|---|---|---|
| Real on-chain PQC-gated cases (n) | 7 (2 from V2 + 5 from V3 S1--S5) | **31** |
| Implementations attesting | 1 (Liboqs only, even in the "independent" V3 oracle) | **2, independent** (Liboqs C + CIRCL Go), 0 disagreements observed |
| $N_{\mathrm{false\text{-}upgrade}}$ | 0/7 | **0/31** |
| 95% upper bound on true rate | 34.82% | **9.21%** ($1-0.05^{1/31}$), a further $3.8\times$ tightening |

## Honest scope notes (what this pass does NOT close)

- There is still **no on-chain EVM precompile** that verifies an ML-DSA-65 signature directly at
  the smart-contract level -- both implementations run off-chain and report their result via a
  signed attestation transaction, not via on-chain lattice arithmetic. This was already disclosed
  as a substantially harder, unattempted problem in `e5_independent_verifier_attestation_2026-09-23.md`
  and remains genuinely open; V4 improves trust-anchor diversity (two independent codebases must
  agree) but does not remove the off-chain-computation trust boundary itself.
  Real on-chain gas-cost data for a from-scratch ML-DSA verifying precompile does not exist on this
  network and was not fabricated here.
- The 12 new keypairs are structurally similar to each other (same two-case valid/tampered
  pattern per pair, following the project's established methodology) rather than each exploring a
  distinct attack shape the way S1--S5 did; this trades scenario diversity for statistical n, a
  deliberate choice for this depth pass given S1--S5 already covered the distinct attack shapes
  (dishonest claim, message substitution, pubkey substitution).
- Real cost: 24 new on-chain PQC-gated cases required roughly 24 registerBank/rotate + 48 oracle
  attestations (24 x 2 implementations) + 24 submitTransfer real transactions -- all real, mined
  Besu transactions, no simulation.

## Files

- `PQCTransferV4_2026-09-24.sol` -- the new dual-oracle contract (copy for reproducibility; canonical
  source at `~/pqc-hardhat/contracts/PQCTransferV4.sol`).
- `e5_dual_implementation_v4_result_2026-09-24.json` -- the 12-pair run's full real results (tx
  hashes, per-case liboqs/circl verdicts, on-chain status).
- `e5_v4_independent_reverify_2026-09-24.json` -- the independent re-verification pass's result.
