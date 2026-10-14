# Full-chain nested-inclusion proof, extended to a real batch of 10 (2026-09-25)

Closes the disclosed limitation: the header->data_hash->envelope->payload full-chain proof
(`FabricNestedInclusionVerifier.verifyFullChain`) "has not yet been extended to the full
10-transaction batch" that the block-commit+endorsement check (`FabricRealTxVerifier`) already
covered in `evidence/both_directions_batch_2026-09-23.md`.

## Method

10 real, varied `SubmitDecision` transactions were submitted concurrently (fired together via
`peer chaincode invoke`, no waiting between them) so the real orderer's own batching would land
them in one block, matching the forward direction's existing batch design: role/amount/semBound/
evAssuranceOK combinations spanning all three real outcome classes (`SAFE`, `REJECT`,
`REEVALUATE`), not selected to all succeed. All 10 landed in real block 3031
(`peer channel getinfo` height 3032 immediately after), confirmed via `qscc GetBlockByNumber`
fetch + the existing, unmodified `fabric_block_ref_verify.js` walker: 10 real envelopes, real
`data_hash` match, real orderer signature valid.

For each of the 10 envelopes, `extract_nested_batch10.js` (reusing `extract_nested.js`'s exact
extraction logic, parameterized over `envIdx`) pulled the real header bytes, full raw block data,
envelope bytes + offset, endorsed-payload bytes + offset, and both organizations' real endorsement
signatures -- the same real byte-exact substring extraction already validated for the earlier 12
cases, no new decoding logic.

`nested_batch10_verify.js` deployed one `FabricNestedInclusionVerifier` and called
`verifyFullChain` once per real transaction (10 separate real on-chain transactions), each proving
independently: the real orderer signature over the block header; `sha256` of the complete real
block data equalling the header's committed `data_hash`; the envelope as a byte-exact substring of
the block data at its real offset; the endorsed payload as a byte-exact substring of the envelope
at its real offset; and both organizations' real endorsement signatures over that payload.

## Results

**10/10 additionally proven on-chain** (`fullyProvenTx=true` for every one), all from a single
real block, each independently verified rather than relying on a shared shortcut. Combined with
the pre-existing 12/12 real cases (single-envelope and multi-envelope blocks, varied envelope
positions), the full-chain proof now stands at **22/22 real cases fully proven on-chain, 2/2
negative controls correctly rejected**. 95% Clopper-Pearson upper bound on the true failure rate:
$1-0.05^{1/22} = 12.73\%$ (down from $22.09\%$ at n=12, and $77.64\%$ at the original n=3).

## Interpretation

The full-chain nested-inclusion proof is no longer limited to the smaller, position-diversity
corpus -- it now also covers a real, coherent batch of 10 simultaneously-submitted transactions
sharing one block, matching the forward direction's batch scale exactly. The only remaining,
disclosed gap for this direction is the proof-depth asymmetry versus the forward direction's
Merkle-Patricia state proofs -- an inherent property of Fabric's flat-concatenation `data_hash`
format (no Merkle structure exists to shorten the proof), not a gap that can be closed by adding
more cases.

## Source

- `/tmp/.../scratchpad/extract_nested_batch10.js`, `nested_batch10_verify.js`
- `evidence/e6_nested_inclusion_batch10_2026-09-25.json` (raw on-chain result, 10 real tx hashes)
