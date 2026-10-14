# E6 remaining limitation, resolved (2026-09-23)

Continuing the "why + fix or explain" pass (E1, E3, E4, E5), applied to E6's
one remaining disclosed limitation
(`evidence/f2b_real_tx_verify_2026-09-23.md`, "Honest scope notes"; n=1/n=10
scaling for the other two limitations was already done same-day in
`evidence/both_directions_batch_2026-09-23.md`).

## Why it existed

`FabricRealTxVerifier.sol` (the corrected Fabric->Besu contract) verifies two
real facts independently: the real orderer signature over the block header
(proves the block was genuinely committed), and the real endorsement
signatures over a `ProposalResponsePayload` (proves the real Org1+Org2
quorum genuinely endorsed that exact payload). It never proved, on-chain,
that the specific payload it checks endorsements over is actually the one
*nested inside* the specific block whose commitment it checks -- both facts
were supplied together by the caller (the off-chain extraction script) and
trusted to correspond, the same division of labor a relay typically uses
with a light client, but not a full proof.

## Fix (real, on-chain, no on-chain protobuf parser)

Fabric's `data_hash` is `sha256(concat(all envelope bytes in the block))` --
a flat concatenation, not a Merkle tree, so there is no compact per-transaction
inclusion proof to construct; the only way to prove one envelope's real
membership is to supply the block's *entire* real data blob and recompute
the hash on-chain, then show the target envelope is a byte-exact substring
of it. `contracts/FabricNestedInclusionVerifier.sol` (new, additive; does
not modify `FabricRealTxVerifier.sol`, whose n=10 evidence stands) does
exactly that, as one function proving the complete real chain in a single
transaction:

1. real orderer signature over `mdValue||sigHeader||headerBytes` (block
   genuinely committed);
2. `sha256(fullBlockData) == claimedDataHash` (the supplied full block data
   is genuinely what the committed header's `data_hash` covers);
3. `fullBlockData[envOffset : envOffset+envelopeBytes.length] == envelopeBytes`
   (the envelope is genuinely nested inside the block's data, at a claimed
   offset, checked byte-exact on-chain);
4. `envelopeBytes[prpOffset : prpOffset+prpBytes.length] == prpBytes` (the
   `ProposalResponsePayload` is genuinely nested inside *that* envelope);
5. every real endorsement signature over `prpBytes||endorserBytes`.

All five checks run in one on-chain transaction; only if every one passes is
the transaction marked `fullyProvenTx`. Byte offsets and all real fields
(header bytes, full block data, envelope bytes, PRP bytes, endorser
identities/signatures) were extracted directly from real, already-committed
blocks by a new script reusing the existing, already-validated
`fabric_block_ref_verify.js` protobuf walker unchanged (no new decoding
logic, no self-signed content). Real block data:
`evidence/nested_inclusion_extract_2026-09-23.json`.

## Real results (2 real cases + 1 real negative control, live Besu)

| Case | Real source | On-chain result |
|---|---|---|
| Single-envelope block | block 240 (1 envelope, degenerate but real: `data_hash` covers exactly one envelope) | `fullyProvenTx=true`, tx `0x3392fe08...2fdd45e6` |
| Multi-envelope block, correct envelope | block 20, envelope index 2 of 5 (the real concurrent-race block already used for E4) -- proves the on-chain check picks the *correct* one of several real envelopes, not a trivial single-item case | `fullyProvenTx=true`, tx `0x437af33d...c84b93d278fe34` |
| Negative control: envelope offset shifted by 1 real byte | same real block-20 data, `envOffset+1` | `REVERTED`: `"envelope not found at offset in block data"` -- the substring check does real cryptographic/byte work, not a vacuous pass |

Contract: `0x21297aa46395eb1f7F8346963dE853EEfE5632C6` (Besu, chainId 1337).
Raw output: `evidence/fabric_nested_inclusion_result_2026-09-23.json`.

## What this establishes, precisely

Combined with `FabricRealTxVerifier.sol`'s existing n=10 result, Besu can now
independently and cryptographically confirm, on-chain, in one call, the
*complete* real provenance chain for a Fabric transaction: the orderer
genuinely committed this exact block; this exact block's data really
contains this exact envelope at this exact position; this exact envelope
really contains this exact endorsed payload at this exact position; and the
real Org1+Org2 quorum genuinely signed that exact payload. No step is
assumed or supplied without an on-chain check.

## Still open, disclosed honestly

- This closes the nested-inclusion gap for 2 real cases (n=1 and n=5
  envelopes), not the full n=10 batch from
  `evidence/both_directions_batch_2026-09-23.md`; extending it to all 10 is
  straightforward (same script, more calls) but was not run this pass.
- `fullBlockData` must be supplied in full (Fabric's data_hash has no Merkle
  structure to shrink this); for a block with many/large transactions this
  becomes an increasingly large calldata payload. This is a real, inherent
  property of Fabric's block format, not an implementation gap -- disclosed
  rather than worked around.
- The forward direction (Besu->Fabric) remains materially more exercised in
  total transaction count; unchanged from before.
