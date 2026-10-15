# E6 real expansion + trust-root verification (2026-09-24)

Two real, additive pieces of work, both following up on the earlier trust-root check
(`e7_corpus_expansion_2026-09-24/README.md` Part 4) at the user's explicit request to pursue
both directions: (1) extend E6's weakest arm (`FabricNestedInclusionVerifier`, previously n=2),
and (2) check another real trust anchor (the Fabric identity public keys registered in
`FabricRealTxVerifier`).

## Part 1 -- FabricNestedInclusionVerifier real corpus: n=2 -> n=12

`fabric_nested_inclusion_verify.js`'s original 2 real cases (block 240 single-envelope, block 20
envelope 2-of-5) had a 95% upper confidence bound of 77.64% on the true failure rate -- far too
wide to be a meaningful security claim on its own. Extended with 10 new real cases, reusing the
exact same real extraction logic (`fabric_block_ref_verify.js`'s protobuf walker, unmodified)
against 6 new real, already-committed Fabric blocks not previously used for this check
(1000, 1005, 1008, 1010, 1012, 1013 -- spanning both single-envelope and 10-envelope blocks, at
varied envelope positions 0/2/5/7/9 for further position-diversity beyond the original's single
position-2-of-5 case):

| Case | Block | Envelope position | Result |
|---|---|---|---|
| block_1008_env0 | 1008 | 0 of 1 | `fullyProvenTx=true` |
| block_1010_env0 | 1010 | 0 of 1 | `fullyProvenTx=true` |
| block_1012_env0 | 1012 | 0 of 1 | `fullyProvenTx=true` |
| block_1000_env0 | 1000 | 0 of 10 | `fullyProvenTx=true` |
| block_1000_env5 | 1000 | 5 of 10 | `fullyProvenTx=true` |
| block_1000_env9 | 1000 | 9 of 10 | `fullyProvenTx=true` |
| block_1005_env2 | 1005 | 2 of 10 | `fullyProvenTx=true` |
| block_1005_env7 | 1005 | 7 of 10 | `fullyProvenTx=true` |
| block_1013_env0 | 1013 | 0 of 10 | `fullyProvenTx=true` |
| block_1013_env9 | 1013 | 9 of 10 | `fullyProvenTx=true` |

**10/10 real new cases proven on-chain.** A second real negative control (block_1005_env7's real
bytes at a deliberately wrong offset) correctly reverted: `"envelope not found at offset in
block data"`.

**Real corpus totals: n=12 (up from 2), 2 negative controls (up from 1).** 95% upper bound on
the true failure rate: **22.09%** (down from 77.64%) -- more than 3.5x tighter. Deployed to a
fresh contract instance at `0xd6665A50E8f2cCa8526E260c3B97c6e3191D933E` (Besu, chainId 1337).

Raw: `nested_inclusion_extract_v2.json` (extracted byte fields), `e6_nested_inclusion_expand_result.json`
(real tx hashes and outcomes).

## Part 2 -- Fabric identity trust-root verification

`FabricRealTxVerifier`'s security depends entirely on the P-256 public keys registered via its
`registerIdentity()` -- if those were stale (e.g., from a since-regenerated Fabric network), all
of E6's real verifications would be meaningless despite reporting success. Checked directly:
extracted the real Org1MSP/Org2MSP/OrdererMSP public key coordinates fresh from block 1014 (a
real, current block from this session's own recent work, fetched live via `peer channel fetch`,
not reused from an earlier extraction), and compared byte-for-byte against
`identityKeys()`'s on-chain values at the deployed `FabricRealTxVerifier`
(`0xB3FF28BFA00e43a4A2A079d6FaE47dC11b2b2CCD`):

| Identity | On-chain qx/qy | Freshly extracted (block 1014) | Match |
|---|---|---|---|
| Org1MSP | `0x92beac10...`/`0x958d9708...` | identical | Yes |
| Org2MSP | `0x6d2c6a7e...`/`0xcad2fee3...` | identical | Yes |
| OrdererMSP | `0x7b66c389...`/`0xd39a5c90...` | identical | Yes |

**All three identical.** Confirms the registered trust anchor for the Fabric->EVM direction is
still the real, current network's actual identity, not a stale copy -- the counterpart check to
the EVM->Fabric direction's validator-set/verification-key confirmation already done. Raw:
`fabric_identity_trust_root.json`.
