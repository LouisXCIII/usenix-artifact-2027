# S0/S2/S3 real reverse direction (Fabric -> EVM) -- closing the guide's bidirectionality gap for H1/H2 (2026-09-24)

Direct follow-up to the `ADP_r3_Experimental_Execution_Guide.docx` cross-check, which found the
S0-S19 adversarial suite was almost entirely tested in one direction (Besu/EVM as source, Fabric as
destination). The guide's own E3 step 5 only asks to repeat scenarios "where the scenario has a
meaningful counterpart" -- rather than attempt all 20 scenarios in reverse (many, like S9
revoked-credential or S15/S16 PQC-specific, are not meaningfully different by direction, or already
have real reverse-direction evidence at the light-client/crypto layer via
`FabricNestedInclusionVerifier.sol`/`FabricRealTxVerifier.sol`), this closes the reverse direction
for the 3 scenarios that most directly test the paper's central H1/H2 non-widening claim: S0
(positive control), S2 (role widening), S3 (threshold widening).

## What was built

`EVMDestinationGate.sol` (new, additive): a faithful, real port of `authchaincode.go`'s
`classify()`/`RoleThreshold`/`RoleMapping` -- identical real role thresholds (manager/500k,
director/2M, clerk/10k) and identical real mapping table (`gerente`->manager, `gerente_senior`
->director) -- run in the OPPOSITE role from the rest of this paper's evidence: here EVM/Besu is
the DESTINATION deciding on a source claim, not the source. This is a genuine "meaningful
counterpart" (same policy semantics, mirrored platform), not an unrelated new test.

This contract does not itself re-verify Fabric-side cryptographic state -- that is the
already-separately-established light-client/MPT layer (`FabricNestedInclusionVerifier.sol`,
`FabricRealTxVerifier.sol`, both already real and reported in `subsec:results-multihop`). Its job,
matching this pass's scope, is the destination POLICY decision only.

## Real test: 3 on-chain EVM transactions, each bound to a REAL Fabric decisionID

Rather than synthetic source references, each case's `sourceRef` is a real Fabric decisionID/certId
already produced earlier this session by genuinely separate real experiments (A3's obligation
sweep, A8's replay sweep, S19's TAC certificate) -- so the "source" side of this reverse-direction
test is anchored to real, already-verified Fabric artifacts, not a placeholder string.

| Scenario | sourceRef (real Fabric artifact) | role | amount | Expected | Observed |
|---|---|---|---|---|---|
| S0 positive control | `a3-real-manager-below_half-...` (real A3 decision) | manager | 200,000 | EXACT | **EXACT** |
| S2 role widening | `a8-real-reject_semBound_to_safe-...` (real A8 decision) | gerente_senior (mapped -> director) | 1,800,000 | SAFE | **SAFE** |
| S3 threshold widening | `0xeee239c5...` (real S19 TAC certId) | manager | 600,000 (exceeds 500k) | REJECT | **REJECT** |

**3/3 matched expectation.** S2 confirms the structural non-widening property: a mapped
(non-identity) role can be legitimately admitted, but only ever as `SAFE`, never `EXACT` --
`EXACT` is reserved for identity-preserving reuse, so a widened role is, by construction, never
treated as a direct reuse even when legitimately admissible. S3 confirms an amount genuinely
exceeding the real destination threshold is rejected regardless of role validity.

## Independent re-verification

A separate script re-read all 3 decisions fresh from chain (`getDecision`, not the run's own saved
values): **3/3 re-confirmed**, identical classes.

## Honest scope notes

- This closes the reverse direction only for S0/S2/S3 (the core H1/H2 non-widening scenarios), not
  the full S0-S19 suite. The remaining scenarios were judged, per the guide's own "meaningful
  counterpart" qualifier, to either not differ meaningfully by direction (e.g. S8 stale state, S11
  TOCTOU -- these are platform/chaincode properties, already demonstrated symmetric) or to already
  have real reverse-direction evidence at the cryptographic layer (S18's two-hop counterexample
  already uses Fabric independently re-verifying Besu state).
- This is n=3, not a swept corpus -- it demonstrates the reverse-direction mechanism is real and
  correctly wired for the central claim, not a statistically powered campaign matching this
  session's other depth passes (A1/A3/A5/A8, S19/A10, E5).
- `EVMDestinationGate` was deployed fresh for this test (not previously used in the paper's other
  Besu evidence) -- it is a new, additive contract, no existing contract was modified.

## Files

- `EVMDestinationGate_2026-09-24.sol` -- the new contract (copy for reproducibility; canonical
  source at `~/pqc-hardhat/contracts/EVMDestinationGate.sol`).
- `evm_reverse_direction_result_2026-09-24.json` -- the 3 real on-chain results.
