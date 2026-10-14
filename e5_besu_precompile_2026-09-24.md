# E5: real Besu-level EVM precompile for ML-DSA-65, closing the off-chain-attestation limitation (2026-09-24)

Direct follow-up to the disclosed E5 limitation: "Chưa có precompile EVM xác minh chữ ký
ML-DSA-65 (PQC) trực tiếp trên smart contract; việc xác minh vẫn chạy off-chain và báo cáo qua
chứng thực đã ký (2 triển khai độc lập -- Liboqs + CIRCL -- phải đồng thuận, nhưng chưa loại bỏ
hoàn toàn biên giới tin cậy off-chain)." User explicitly chose the higher-risk, higher-fidelity
path ("Besu precompile thật") over a pure-Solidity verifier after being warned of the tradeoffs.

## What was built

A genuine, additive Besu-level EVM precompile -- not a Solidity approximation -- forked from the
**official Besu `25.12.0` release tag** (`~/pqc-hardhat/besu-mldsa-precompile/`, a fresh clone,
kept fully separate from the unrelated PQC-consensus paper's own Besu fork at
`~/pqc-research/besu-pqc/`, which was read only for structural reference and never modified):

- `MLDSA65VerifyPrecompiledContract.java` (new): implements `PrecompiledContract`, modeled
  directly on the real `P256VerifyPrecompiledContract.java` (EIP-7951). Fixed 5,315-byte input
  (`messageHash(32B) || publicKeyX509(1974B) || signature(3309B)`), output `Bytes32(1)` (valid)
  or empty (invalid/malformed), fail-closed on every error path. Uses BouncyCastle 1.80's real
  `Signature.getInstance("ML-DSA-65","BC")` -- the exact version already platform-pinned in
  official Besu 25.12.0's own `platform/build.gradle`, no version bump needed.
- `Address.java`: new constant `MLDSA65_VERIFY = Address.precompiled(0x0101)` (next unused
  address after the real `P256_VERIFY` at `0x0100`).
- `MainnetPrecompiledContracts.java`: registered in `populateForIstanbul` (confirmed by reading
  `MainnetProtocolSpecs.java` that Berlin/Muir Glacier inherit Istanbul's precompile registry
  unchanged, matching this network's real genesis `berlinBlock:0`), so the precompile is active
  from block 0.

Real, measured (not assumed) sizes: a standalone BouncyCastle 1.80 smoke test generated 8 fresh
ML-DSA-65 keypairs and confirmed the JCA-encoded (X.509) public key is a stable 1,974 bytes
(22 bytes more than the raw 1,952-byte key reported elsewhere in this paper's evidence, because
`PublicKey.getEncoded()` returns the full DER wrapper) and the signature a stable 3,309 bytes,
across all 8 samples.

## Build

Real environmental issues hit and resolved, none related to the precompile code itself:
- Gradle dependency-verification checksum mismatch (BouncyCastle republished a `1.80.2` patch
  after this Besu release's checksums were pinned) -- fixed via the documented
  `--write-verification-metadata sha256` mechanism (not by disabling verification).
- Disk pressure (host at 96% full mid-build) -- traced to an 8GB Ethereum reference-test-fixture
  submodule Gradle auto-fetched but this task never needed; deleted (`git submodule deinit` +
  removing the auto-fetched `build/execution-spec-tests` directory), freeing the host back to a
  safe margin before continuing.

`:evm:compileJava` (the module containing the new file) compiled clean:
`BUILD SUCCESSFUL in 19m 18s`. Full runnable distribution (`installDist`): `BUILD SUCCESSFUL in
5m 25s`, 92 tasks (87 executed, 5 up-to-date).

## Isolated smoke test (before touching the live network)

A temporary, throwaway single-node `--network=DEV` instance (not the real network, no shared
state) was started with the new binary. Real `eth_call` against address `0x101`:

| Case | Input | Result |
|---|---|---|
| Valid ML-DSA-65 signature | real keypair, real sign | `0x0...01` (VALID) |
| Tampered signature (1 bit flipped) | same keypair, corrupted sig | `0x` (INVALID) |
| Malformed input (4 bytes) | garbage | `0x` (INVALID, no crash) |

**3/3 correct.** Only after this passed was the live-network rollout considered.

## Deployment to all 4 live validators

Identical binary (SHA-256 `0f3ea8a1...6d4a9`, verified matching on every host after transfer)
deployed one validator at a time, each verified healthy before moving to the next -- following
the pre-approved staged plan, never all 4 at once:

| Validator | IP | Result |
|---|---|---|
| 1 | 10.10.20.14 | binary swapped, restarted, peerCount 3→0→3 within ~10s, blocks resumed ~2s/block |
| 2 | 10.10.20.141 | same, verified via running jar classpath (`besu-app-26.9-develop-05b6c1a.jar`) |
| 3 | 10.20.20.114 | same |
| 4 | 10.20.20.45 | same |

Old `/opt/besu` backed up to `/opt/besu.bak-20260924` on every validator before replacement
(rollback available). After all 4: `net_peerCount=3` on every check, `ibft_getValidatorsByBlockNumber`
unchanged (same 4 addresses), block height advancing steadily throughout the whole rollout
(0x74315 → 0x74649 over the course of this pass) -- no fork, no quorum loss, no stall.

Real `eth_call` against `0x101` on the **live network** (not the isolated smoke node) reproduced
the same valid→VALID / tampered→INVALID result.

## Real on-chain corpus: `PQCTransferV5.sol`

New, additive contract (`PQCTransferV5.sol`) replaces `PQCTransferV4.sol`'s 2-oracle
off-chain-attestation model with a direct `staticcall` to the real precompile at `0x101` --
no oracle contract, no off-chain verifier identity to trust. Public keys are stored in the
precompile's real X.509 encoding (1,974 bytes), disclosed as a different, non-interchangeable
format from V4's 1,976-byte raw Liboqs encoding.

12 fresh real ML-DSA-65 keypairs (BouncyCastle 1.80, the same library the live precompile
itself uses) were generated, each producing a genuine valid signature and an independently
tampered copy (a different byte position flipped per pair) -- 24 real `submitTransfer`
transactions submitted to the live `PQCTransferV5` contract on the real, now-fully-upgraded
4-validator network.

**24/24 correct** (12/12 valid → `VERIFIED`, 12/12 tampered → `REJECTED`). 95% upper bound on
the true bypass rate given 0 observed errors in n=24: `1-0.05^(1/24) = 11.73%`.

## What this closes and what remains open

Closed: the off-chain-attestation trust boundary that both `PQCTransferV3` (self-attestation)
and `PQCTransferV4` (2-oracle attestation) still had. ML-DSA-65 verification for this contract
now happens genuinely inside EVM execution, consensus-critical across all 4 validators, not
reported by any off-chain party.

Still open, disclosed rather than overclaimed:
- Gas cost (`FIXED_GAS_COST = 200,000` in the precompile) is a conservative placeholder, not
  yet derived from a real profiled execution-time measurement of this specific precompile.
- This precompile exists only on this private research network's custom Besu build; it is not
  a proposal to any real Ethereum client or EIP.
- n=24 is a real but modest first corpus for the on-chain path specifically (distinct from the
  n=31 the broader E5 assurance-continuity result already has via the 2-oracle path) -- growing
  it further would follow the same real generation/submission pipeline used here.

## Files

- `~/pqc-hardhat/besu-mldsa-precompile/` -- the custom Besu 25.12.0 checkout with the new
  precompile (not committed to any shared repo; local build artifact).
- `~/pqc-hardhat/contracts/PQCTransferV5.sol` -- the new on-chain-verifying contract.
- `~/pqc-hardhat/experiments/portability/e5_precompile_v5_test.js` -- the real on-chain test.
- `e5_precompile_v5_result_2026-09-24.json` -- the 24 real transaction results.
