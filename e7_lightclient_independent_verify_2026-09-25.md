# Independent re-implementation of IBFT2 header + commit-seal verification (2026-09-25)

Closes the disclosed limitation in `subsec:results-baselines`: "the header/light-client
commit-seal mechanism itself was not independently re-implemented, disclosed as still open."

## What was built

`light_client_independent_verify.js`: a from-scratch re-implementation of
`lightclient.go`'s `verifyIbft2Header`, written without reading or importing any of
`lightclient.go`/`zkverify.go` -- only the underlying scheme (Besu's public
`BftBlockHashing.calculateDataHashForCommittedSeal` IBFT2 header/seal RLP layout) is shared,
since that is Besu's own documented on-chain format, not this project's code. Uses `ethers.js`'s
own RLP encoder, `keccak256`, and secp256k1 ECDSA recovery -- a different language, library, and
codebase from the Go chaincode's `decred`-derived `secpecdsa` package. The validator set is
fetched fresh via the live `ibft_getValidatorsByBlockNumber` RPC call, an independent source from
the chaincode's hardcoded `pinnedValidatorHex` array (not a reuse of it).

## Method

For a real, freshly-fetched header at the current chain tip:
1. Decode the header's real `extraData` (RLP list `[vanity, validators, vote, round, seals]`)
   independently.
2. Cross-check the decoded validator set against the live `ibft_getValidatorsByBlockNumber`
   result (an independent source, not the chaincode's hardcoded pinning).
3. Reconstruct the block hash from the 15 raw header fields with `extraData` substituted by
   `RLP([vanity, validators, vote])` (the block-hash form) and compare to the claimed hash.
4. Independently compute the commit-seal hash with `extraData` substituted by
   `RLP([vanity, validators, vote, round])` (seals excluded, per Besu's scheme).
5. For each 65-byte commit seal, recover the signer's address via ECDSA public-key recovery
   over the seal hash and check membership in the validator set; require
   `>= ceil(2n/3)` distinct valid signers (n=4 here, quorum=3).
6. Negative control: tamper one byte of a real seal and confirm the independent recovery no
   longer yields a valid pinned-validator address.

## Results

Two real, independently-fetched blocks at the live chain tip:

| Block | Validators | Seals | Independent block-hash match | Distinct valid signers | Quorum | Negative control |
|---|---|---|---|---|---|---|
| 494023 | 4 | 3 | MATCH | 3/4 | 3 required -- PASS | correctly rejected (wrong address) |
| 494030 | 4 | 3 | MATCH | 3/4 | 3 required -- PASS | correctly rejected (recovery error) |

Both real cases: independent block-hash reconstruction matched the claimed hash exactly,
independent signer recovery correctly identified the real committing validators (a *different*
3-of-4 subset each time, confirming the match is not coincidental), quorum was correctly
verified, and the negative control correctly failed recovery/membership both times (once by
recovering a non-validator address, once by hitting an invalid-curve-point error during
recovery -- both are correct rejections, just via different failure paths, which is itself
expected since ECDSA recovery on tampered input has no guaranteed failure mode other than "does
not recover a valid signer").

## Interpretation

The header/light-client commit-seal mechanism now has a second, genuinely independent
implementation (different language, different cryptographic library, different validator-set
source) that agrees with the chaincode's real result on two real, live cases, with a working
negative control. This closes the previously disclosed gap; the only remaining independence gap
for the real (non-stub) corpus was already closed in
`evidence/e7_independent_reverify_2026-09-23.md`.

## Source

`/tmp/.../scratchpad/light_client_independent_verify.js` (self-contained, ~120 lines, only
dependency is `ethers` v6).
