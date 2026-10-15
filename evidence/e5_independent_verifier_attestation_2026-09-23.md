# E5 limitations, resolved where resolvable (2026-09-23)

Continuing the "why + fix or explain" pass (E1, E3, E4) applied to E5's
disclosed limitations (`evidence/e5_pq_profiles_2026-09-23.md`, "Honest scope
notes").

## (1) `offChainVerified` is caller-supplied and trusted -- FIXED (real independent on-chain attestation, not an on-chain lattice precompile)

**Why it existed**: `PQCInterBankTransferV2.submitTransfer` takes
`offChainVerified` as a plain `bool` argument from `msg.sender` -- the same
account submitting the transfer self-reports whether its own PQC signature
verified, and the contract trusts it outright. There is no cheap on-chain
lattice-signature precompile (RIP-7212 P256Verify was already confirmed
absent on this Besu build for classical P-256; an equivalent for ML-DSA/
Dilithium does not exist at all in production EVM systems -- building one is
a substantially harder, currently-open research problem this session did not
attempt).

**Fix (real, not simulated)**: two new, additive contracts -- the existing,
deployed `PQCInterBankTransferV2` is untouched, its n=2 evidence stands
unchanged.

- `contracts/PQCVerifierOracle.sol`: a designated, SEPARATE identity (a
  third real funded wallet, `PRIVATE_KEY_C`, distinct from both banks'
  `PRIVATE_KEY`/`PRIVATE_KEY_B`) is the only address allowed to call
  `attest(pubKeyHash, msgHash, sigHash, verifiedResult)`. Each attestation is
  its own real, separately signed on-chain transaction from that identity.
- `contracts/PQCTransferV3.sol`: `submitTransfer` requires a pre-existing
  oracle attestation for the exact `(pubKeyHash, msgHash, sigHash)` tuple and
  sets `VERIFIED`/`REJECTED` from **that** attestation, not from the caller's
  `claimedVerified` argument. `claimedVerified` is still recorded, purely to
  detect a mismatch against the independent result (mirrors the
  claimed-tier-vs-verified-tier split already built for E3's S15/S16 formal
  model, `evidence/e3_limitations_resolved_2026-09-23.md`).

This converts single-party self-attestation into two-party, cryptographically
checked attestation: a dishonest submitter's claim can no longer override
what an independent party actually verified.

## (2) n=2, not a corpus -- FIXED (n=2 -> n=7, five new real diverse scenarios)

**Why it existed**: the original E5 evidence was one generator run producing
one valid case and one tampered case.

**Fix**: `experiments/portability/e5_independent_verifier_attestation.js`
uses a new, standalone `KeyOps.java` CLI (new file, reuses the existing
`DilithiumNative` JNI binding unchanged, no existing project file modified)
to generate two real, independent ML-DSA-65 keypairs and sign multiple
chosen messages with each -- something the original single-shot generator
could not do. Five real on-chain scenarios were run against the live Besu
network (chainId 1337):

| Scenario | Real construction | Independent re-verification | On-chain result |
|---|---|---|---|
| S1 valid, honest claim | genuine signature, genuine message | `true` | `VERIFIED`, no mismatch |
| S2 tampered, honest claim | 1-byte-flipped signature, bank correctly reports failure | `false` | `REJECTED`, no mismatch |
| S3 tampered, **dishonest** claim | same tuple as S2, bank now claims `true` | `false` (attestation already on-chain, reused) | `REJECTED` -- the lie does not override the independent attestation; `selfReportMismatch=true` |
| S4 message substitution | a genuinely valid signature over message A, submitted as evidence for message B it was never taken over | `false` (fails cryptographically against B) | `REJECTED`, mismatch=true |
| S5 pubkey substitution | a genuinely valid signature produced under Bank-A's key, submitted by Bank-B as its own evidence | `false` (fails against Bank-B's actually-registered key) | `REJECTED`, mismatch=true |

**5/5 real outcomes matched expectation**, including the two new negative
controls (S3, S5) that specifically target the two things a purely n=2 demo
could not show: a lying submitter, and a signature bound to the wrong
identity. Combined with the original n=2 (V2) result, this project now has
**n=7 real on-chain PQC-gated transactions** across two independent contract
designs (self-attested V2, independently-attested V3).

Real addresses/tx hashes: oracle `0x65c2711D0fD586368f33b962cFa243E6A431E78B`,
`PQCTransferV3` `0x033fa0158d3Eb580C4bdcce565704E11F197542C`, both deployed by
Bank-A on the live network. Raw output:
`evidence/e5_independent_verifier_attestation_result_2026-09-23.json`
(per-scenario attest/submit tx hashes, transfer ids, and on-chain field
values).

## Still open, disclosed honestly

- An on-chain lattice-signature (ML-DSA/Dilithium) verifying precompile does
  not exist on this or any production EVM network; this fix works around
  that gap with an independent-attestor pattern rather than solving on-chain
  native verification, which remains open.
- The independent verifier's attestation still ultimately runs the same
  native liboqs binary the rest of the project uses (a different process
  invocation and a different signing identity, but the same underlying
  verifier implementation) -- it is not verification by an independently
  implemented, diverse codebase.
- P0 (classical decision, classical transport) still has no real on-chain
  instance on this testbed, for the same architectural reason as before
  (transport is mandatorily PQ everywhere); unchanged, not claimed as fixed.
