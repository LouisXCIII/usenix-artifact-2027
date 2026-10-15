# E5 — Classical/Hybrid/PQ assurance-migration, wired through real evidence (2026-09-23)

Real result for `Manuscriptsp_m2_r3.tex`'s E5 (`subsec:results-assurance`,
Table `tab:crypto-security-results`). Unlike the other E1-derived experiments,
this one required genuinely new work: the offline E1 formal benchmark already
tests the *rule* (`assure_ok` conjunct, eq:pq-scope-example) with synthetic
tier labels, but nothing in the project had previously fed a **real,
independently-verified ML-DSA-65 decision signature** through it. Two things
that already existed separately were wired together instead of building new
PQC plumbing from scratch:

- The deployed `PQCInterBankTransferV2` contract (chainId 1337, same Besu
  network as the rest of this paper's evidence; address
  `0x94b60671bF38e92F57fbdd5d56909455990F537A`), from the companion PQC
  consensus paper's testbed. It records a Dilithium3/ML-DSA-65 signature over
  a transfer payload, `offChainVerified` (computed by the *caller*, off-chain
  -- there is no cheap on-chain lattice-signature precompile), and rejects
  the transfer if that flag is false.
- The real Liboqs 0.15.0 JNI ML-DSA-65 signer/verifier
  (`~/pqc-liboqs/target/pqc-liboqs-1.0-SNAPSHOT.jar`,
  `libmldsa_jni.so`), already used for `test_pqc_liboqs.js` and for v1's real
  sign/verify/gas cost table (`tab:evidence-cost`).

## What was built this session

- A new, small standalone Java CLI, `VerifyOnly.java` (compiled against the
  existing jar, no changes to any existing file), that re-invokes the same
  native `DilithiumNative.verify()` binding on arbitrary
  (pubkey, message, signature) inputs -- needed to independently confirm that
  a *tampered* signature genuinely fails verification, rather than assuming
  it would.
- `~/pqc-hardhat/experiments/portability/e5_pq_profiles.js`: generates one
  real ML-DSA-65 keypair+signature (valid case), builds a byte-flipped
  tampered copy of that same signature, independently re-verifies both
  (real, native verification calls, not simulated), and submits **two real
  on-chain transactions**.

## Real results (both transactions committed on live Besu, chainId 1337)

| Case | Tx hash | On-chain status | `offChainVerified` | Rejection reason |
|---|---|---|---|---|
| Valid ML-DSA-65 signature | `0x473ddad18410056a1f0d8ea340bde2f4c4c796929f4366bf2a54a9b5686ad033` | VERIFIED (1) | `true` (real) | -- |
| Tampered ML-DSA-65 signature | `0xf2c5ac47aed748858e67a44dd49bbe8c0640fc32099aa44f18f087e62bdba308` | REJECTED (2) | `false` (real) | "Off-chain Dilithium3 signature verification failed" |

Raw output: `evidence/e5_pq_profiles_result_2026-09-23.json`.

## Claim-function interpretation (mapped to r3's P0–P4)

The same claim-function logic already used for E1 (`e1_classifier.py`'s
`_claims()` -- classical tx signature grants `ClassicalDecisionAuth`;
a verified ML-DSA/PQ decision signature independently grants
`PQDecisionAuth`; nothing amplifies from transport alone) was applied to
these two real cases:

| Profile | Real instance | Decision-authentication claims present | Verdict |
|---|---|---|---|
| P0 (classical decision, classical transport) | Not realizable on this testbed -- the live network's P2P layer is *mandatorily* ML-KEM768-tunneled (`pqc_tunnel` mesh, `REBOOT_RECOVERY.md` "Buoc 4"; without it the validators never reach IBFT quorum), so no real "classical transport" instance exists here. Sourced from E1's formal-benchmark slice instead (already real, 0 false upgrades). | -- | Formal only |
| P1 (classical decision, **PQ transport**, PQ required) -- critical negative control | The tampered-signature case: the outer Besu transaction is still validly ECDSA-signed and committed, transport is the real always-on ML-KEM768 tunnel, but the inner PQ signature genuinely fails verification | `ClassicalDecisionAuth` only | Correctly does **not** satisfy a PQ decision-authentication requirement -- real, not just asserted |
| P2 (classical + independent accepted ML-DSA decision evidence, hybrid required) | The valid-signature case | `ClassicalDecisionAuth` + `PQDecisionAuth` | Satisfies hybrid/dual requirement |
| P3 (PQ decision signature, PQ transport, PQ required) | Same valid-signature case | `PQDecisionAuth` present | Satisfies PQ positive control |
| P4 (classical decision, hybrid transport, classical accepted) | The tampered-signature case, under the weaker classical-accepted requirement | `ClassicalDecisionAuth` present | Correctly satisfied |

`N_false-upgrade = 0` across both real cases (no case was ever granted
`PQDecisionAuth` without a genuinely, independently verified PQ decision
signature) -- consistent with E1's 385-case formal-benchmark result on the
same property.

## Honest scope notes

- `offChainVerified` is supplied by the *caller* and trusted by the
  contract -- there is no on-chain lattice-signature precompile to verify it
  independently at the EVM level. This session's contribution is verifying it
  independently in a *separate* process (`VerifyOnly.java`, a second,
  distinct invocation of the same native verifier) before submitting, and
  confirming the contract's own gating matches that independent result. This
  is weaker than an on-chain verifying precompile would be, and that gap is
  not closed by this work.
- This is n=2 (one positive, one negative), not a corpus -- it demonstrates
  the mechanism is real and correctly wired, not a statistically powered
  campaign. The 385-case *formal* slice from E1 remains the source for
  aggregate FAR-style statistics on this property.
- P0 has no real on-chain instance on this specific testbed (transport is
  mandatorily PQ everywhere); this is disclosed rather than worked around.
