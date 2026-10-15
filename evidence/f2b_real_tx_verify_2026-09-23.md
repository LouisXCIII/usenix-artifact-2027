# E6, Fabric -> Besu direction: corrected, protocol-accurate real verification (2026-09-23)

Supersedes `evidence/f2b_fabric_endorsement_verify_2026-09-23.md`. The user
reviewed that first attempt and objected, correctly: it verified signatures
*this session produced itself* (using the real peer private keys, but over a
JSON string this session invented) rather than signatures the live network
actually produced during real endorsement, over Fabric's real signed byte
layout. This version fixes both problems: every signature verified here was
produced by the real network during real prior operation, over exactly the
bytes Fabric's own wire protocol signs -- nothing is signed by this session.

## Investigation

1. Built `fabric_block_ref_verify.js`: an independent, from-scratch protobuf
   walker (no protobuf library, no code shared with the Solidity contract)
   that decodes a real Fabric block fetched with `peer channel fetch` and
   verifies, using only Node's built-in `crypto`:
   - the block header hash chain (`sha256(DER(number, prev_hash, data_hash))`
     matches the next block's `previous_hash`);
   - `data_hash == sha256(concat(envelope bytes))`;
   - the real orderer's signature over
     `metadata_value || signature_header || header_bytes`;
   - each real peer endorsement's signature over
     `proposal_response_payload || endorser_identity_bytes`.
2. Ran it against three real, already-committed blocks (237, 238, 239, part
   of the real X15 corpus-scale forged-feature-attack run) and the real
   block 20 (the exact 5-transaction concurrent race already reported in
   `evidence/concurrency_toctou_experiment_2026-09-21.md` for E4) --
   **every** header-chain link, data hash, orderer signature, and
   endorsement signature verified correctly against real, unmodified
   ledger data, confirming the byte layout and signing convention before
   any Solidity was written.
3. Wrote `fabric_extract_real_tx.js` to pull the exact real byte fields for
   one real transaction (block 239, real committed transaction
   `f3508ba182d75a7f...`) out of the block: the real block header bytes, the
   real orderer `(r,s)` (DER-decoded from the real 71-byte signature), the
   real `ProposalResponsePayload` bytes, and both peers' real
   `SerializedIdentity` bytes and real `(r,s)` endorsement signatures
   (DER-decoded), plus both peers' and the orderer's real P-256 public keys
   -- all read directly out of the certificates embedded in the block
   itself, not from external files.
4. Rewrote the contract as `FabricRealTxVerifier.sol`: verifies (A) the real
   orderer signature over the real block header (block commitment) and (B)
   the real Org1+Org2 endorsement signatures over the real
   `ProposalResponsePayload`, each independently, using OpenZeppelin's
   audited `P256.verify()`.

## Real results (5 real on-chain transactions, block 239's real committed tx)

| Case | Tx hash | Result |
|---|---|---|
| Real orderer block-commitment signature | `0xd6785aec0b6d49f5e87f5708cf30daf55bb3b4688694b05a169289891f328e82` | `committedBlocks[headerHash] = true` |
| Real Org1+Org2 endorsement signatures | `0x9d764c252b5385068cb0a17f86c05c4a7219d010bcc1664222459a986f68d3e6` | `endorsedPayloads[payloadHash] = true`; `isFullyVerified = true` |
| Negative control: tampered Org2 signature (1 bit flipped) | `0xe0d73b64bcfb2a1097e9ff514dd69b8d2bf1b020afdc09fa2aabb26672370261` | `quorumMet = false` |
| Negative control: only Org1 present | `0x9550d3af9ec93208845a743cde4523a8527832613df4deca4633268decbbba31` | `quorumMet = false` |
| Negative control: wrong (garbage) orderer public key | `0xf051aacad604cff59ed142cee676ba80b3b2a92c4f819efa7b5701f3587cfc90` | `committed = false` |

Contract: `0xB3FF28BFA00e43a4A2A079d6FaE47dC11b2b2CCD` (Besu, chainId 1337).
Raw data: `evidence/fabric_extract_real_tx_result_2026-09-23.json` (every
extracted byte field), `evidence/f2b_real_tx_verify_result_2026-09-23.json`
(the on-chain results above).

## What this establishes, precisely

Besu independently and cryptographically confirmed, on-chain, using only
data pulled from an already-committed Fabric block:
- that the real Fabric ordering service (`OrdererMSP`) genuinely produced
  this exact block header, i.e. this transaction is really committed, not
  merely proposed; and
- that the real Org1 and Org2 peers -- the exact quorum `authchannel`'s
  real endorsement policy requires -- genuinely endorsed this exact
  transaction payload.

Both checks are over byte sequences and signatures the network itself
produced during real prior operation; this session supplied none of the
cryptographic material being verified, only the (already correct) real
data pulled from the ledger.

## Honest scope notes (unchanged from the first attempt, still apply)

- This does not itself parse the envelope/RWSet protobuf on-chain to prove
  the given `ProposalResponsePayload` is nested inside the specific
  envelope covered by `data_hash` -- block commitment and endorsement are
  verified as two independently signed facts supplied together by the
  caller (the extraction script), the same division of labor a relay
  typically uses with a light client. A full nested inclusion proof
  (`data_hash -> envelope -> payload`) is not attempted.
- $n=1$ real transaction fully verified, plus 3 real negative controls --
  a proof of mechanism, not a corpus or an integration into a live decision
  pipeline.
- The forward direction (Besu->Fabric) remains materially more exercised
  (dozens of real transactions across many workload families); this
  reverse-direction mechanism has been run once.
