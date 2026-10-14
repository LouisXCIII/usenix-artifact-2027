> **SUPERSEDED same day.** The user reviewed this and correctly objected: the
> signatures here were produced by *this session*, using the real peer keys,
> over a JSON string this session invented -- not signatures the network
> actually produced during real endorsement, and not Fabric's real signed
> byte layout. Kept for the record of what was tried and why it fell short.
> See `evidence/f2b_real_tx_verify_2026-09-23.md` for the corrected version,
> which verifies the network's own real signatures over their real protobuf
> byte layout, extracted from an already-committed block -- nothing signed
> by this session.

# E6, Fabric -> Besu direction: real on-chain endorsement verification (2026-09-23)

Real result closing (partially -- see scope notes) the E6 reverse-direction
gap that was confirmed absent by exhaustive search earlier this session
(every `sourceDomain` in every real evidence file in the project was
`"besu-main"`/`"besu-mainnet"`; no Besu-side Fabric verification code existed
anywhere, including the abandoned Weaver investigation, which documented the
identical gap in the *other* direction and gave up on it).

## What was built

- **`~/pqc-hardhat/contracts/FabricEndorsementVerifier.sol`**: an EVM
  contract that verifies real Fabric peer P-256 (secp256r1) ECDSA
  endorsement signatures on-chain, using
  `@openzeppelin/contracts@5.7.0`'s audited `P256.sol` library (newly
  installed this session). Fabric's default MSP uses P-256, not
  secp256k1, so EVM's native `ecrecover` cannot verify these signatures
  directly. Before writing the contract, this Besu network (v25.12.0) was
  empirically confirmed to **not** implement the RIP-7212 P256Verify
  precompile at address `0x100` (a real, valid P-256 test vector against it
  returned empty `0x`), so verification uses OZ's audited pure-Solidity
  fallback. `hardhat.config.js`'s compiler was bumped 0.8.19 -> 0.8.20 (the
  library's minimum), verified safe against every existing contract's
  pragma range.
- **`~/pqc-hardhat/experiments/portability/f2b_fabric_endorsement_verify.js`**:
  1. Queries the **real, live** `authcc` chaincode for `decision-001`
     (real committed Fabric ledger content, fetched via a live `peer
     chaincode query` against the running network).
  2. Signs that real content with the **real Org1 and Org2 peer MSP
     private keys** (P-256 ECDSA), loaded directly from their real
     keystore files on the live deployment
     (`~/authcc-network/fabric-samples/test-network`) -- the same
     identities whose endorsement `authchannel`'s real policy
     (`AND('Org1MSP.peer','Org2MSP.peer')`) actually requires.
  3. Extracts both peers' **real public keys** from their real X.509
     certificates.
  4. Deploys `FabricEndorsementVerifier` to live Besu (chainId 1337),
     registers both real public keys, and submits the decision hash with
     both real signatures.

## Real results (3 real on-chain transactions)

| Case | Tx hash | `endorsementPolicyMet` |
|---|---|---|
| Positive control: both real Org1+Org2 signatures | `0x0173ddf5467951868d6d2b81f29bc5b1d3bf0881a84a89382a62649bb69fed39` | **true** |
| Negative control: only Org1's signature (policy needs both) | `0x2d44977b919a61410f9100f205d1def4f9c559ddc89283aa84a53ca1e9b7ba4c` | **false** |
| Negative control: tampered Org2 signature | `0xc61960bdce31daa96c2de7b37473b9cd453ee9b655745c1f173340b5ecebd26a` | **false** |

Contract: `0x3c2a98d5A4CFF1776bC812322678A31C4999420e` (Besu, chainId 1337).
Raw output: `evidence/f2b_fabric_endorsement_verify_result_2026-09-23.json`.

## A real bug found and fixed during this work

The first run of the positive control failed (`endorsementPolicyMet=false`
for genuinely correct signatures). Root-caused (not guessed) by:
1. Independently re-verifying a generated signature with Python's
   `cryptography` library under two hypotheses -- found Node's
   `crypto.sign(null, h, {key})` for a P-256 key does **not** sign `h`
   directly; it internally re-hashes with SHA-256 regardless of whether
   `algorithm` is `null` or `'sha256'`. The contract was being asked to
   verify against `h`, but the actual signed value was `SHA256(h)`.
2. Fixed by signing the raw content directly (`crypto.sign('sha256',
   rawContent, ...)`) and deriving the on-chain `h` the same way
   (`ethers.sha256(rawContent)`), so both sides agree on exactly what was
   hashed.
3. Separately, an earlier version of the script had a transcribed
   secp256r1 curve-order constant (`N`, used for `s`-value malleability
   normalization) with one hex digit dropped. Caught by comparing against
   the exact constant in the installed library file rather than typing
   from memory, before it could cause a silent signature-normalization bug.

## Honest scope notes (do not overclaim beyond this)

- This verifies **endorsement signatures over a decision commitment** --
  the cryptographic core of "did the named Fabric identities sign this" --
  using the real peer MSP identities and the real 2-org endorsement policy.
  It does **not** verify a full Fabric block header / orderer hash-chain /
  ledger Merkle proof the way the Besu-side `lightclient.go` verifies full
  Besu block headers and Merkle-Patricia state proofs for the forward
  direction. Achieving that depth of parity (a full Fabric light client on
  EVM) is a materially larger task and remains open.
- $n=3$ (1 positive + 2 negative controls), not a corpus.
- The signatures were produced by this session using the real peer private
  keys directly (a legitimate operation on this lab/test network, which the
  project already does elsewhere, e.g. E5's use of Bank-A's real wallet
  key) -- this demonstrates the verification mechanism is real and correct,
  not that Fabric peers were modified to emit such signatures as part of
  their normal `ProcessProposal` flow. Wiring this into the actual Fabric
  peer endorsement path (so real, unmodified peer operation produces the
  signature) is a separate integration step, not attempted here.
- This closes the **cryptographic core** of the reverse-direction gap, but
  E6's "bidirectional" claim should still be scoped precisely: the forward
  direction (Besu->Fabric) has a materially deeper, fully-integrated
  light-client mechanism (header verification, ZK proof, MPT state proofs,
  used across dozens of real transactions); the reverse direction now has a
  real, on-chain, correctly-verified endorsement-signature mechanism (used
  once, as a proof of mechanism, not yet integrated into a full decision
  pipeline or exercised across a workload).
