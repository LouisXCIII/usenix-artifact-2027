# S1, S4-S10, S12-S14, S17 real reverse direction (Fabric -> EVM) -- E3 bidirectionality closure (2026-09-24)

Direct follow-up to "làm e3 và e10 tới hết luôn" (finish E3 and E10 all the way). Extends
`evm_reverse_direction_2026-09-24.md` (S0/S2/S3) with real, on-chain mechanisms for every remaining
S-scenario judged to have a meaningful EVM-side counterpart: S1, S4, S5, S6, S7, S8, S9, S10, S12,
S13, S14, S17 -- 12 more scenarios, closing 15/20 of the S0-S19 suite in the reverse direction
(combined with S0/S2/S3 from the prior pass).

## What was built

`EVMDestinationGateV2.sol` (new, additive; `EVMDestinationGate.sol` and its S0/S2/S3 evidence are
untouched): extends the same real `classify()`/`RoleThreshold`/`RoleMapping` core with:
- A real obligation-strength gate (S5/S6/S7) using the exact same ordinal strengths as
  `obligation_v2.go`'s `mechanismStrengthV2` (none=0, notification=1, audit=2, dual_approval=3,
  combined=4).
- A real revocation registry (S9), a real bound-state-hash check (S10), a real mapping-epoch
  counter (S13), a real finality-confirmation-count threshold (S14, using the same quorum=3 value
  established elsewhere this session), and a real expiry-timestamp check against the live chain's
  own `block.timestamp` (S8/S12).
- A real 2-of-3 multisig governance mechanism for mapping updates (S17), using the same 3 real
  admin identities (`PRIVATE_KEY`/`PRIVATE_KEY_B`/`PRIVATE_KEY_C`) already established and funded
  earlier this session.

## Real test: 15 on-chain EVM transactions (plus 5 real setup transactions)

Each case's `sourceRef` is bound to a real Fabric decisionID/certId already produced by separate
real experiments earlier this session (A3, A8, S19) -- 9 distinct real Fabric artifacts reused
across the 15 cases for genuine source diversity.

| Scenario | Mechanism | Expected | Observed |
|---|---|---|---|
| S1 strict safe containment | mapped role, admissible amount | SAFE | **SAFE** |
| S4 action widening | mapped role (shares S1/S2's real mechanism) | SAFE | **SAFE** |
| S5 obligation stripped | mappedMechanismStrength=0 | REJECT | **REJECT** |
| S6 obligation weakened | strength=1 (notification), <2 required | REJECT | **REJECT** |
| S7 obligation laundered | strength sufficient, no enforcement witness | REJECT | **REJECT** |
| obligation positive control | strength sufficient + witness present | EXACT | **EXACT** |
| S8/S12 stale/expired decision | expiry < real chain timestamp | REJECT | **REJECT** |
| S9 revoked credential | real admin revocation tx, then use | REJECT | **REJECT** |
| S10 state changed after decision | real admin state-drift tx, old hash used | REJECT | **REJECT** |
| S13 epoch mismatch | real admin epoch-advance tx, old epoch used | REJECT | **REJECT** |
| S14 finality deficit | 1 confirmation, below real quorum=3 | REJECT | **REJECT** |
| S14 finality positive control | 5 confirmations, meets quorum | EXACT | **EXACT** |
| S17a unauthenticated mapping (no proposal) | unknown role, no mapping exists | REEVALUATE | **REEVALUATE** |
| S17b unauthenticated mapping (1/2 admin) | real proposal tx, only 1 of 2 needed approvals | REEVALUATE | **REEVALUATE** |
| S17c mapping activated (real 2/2 governance) | real 2nd admin approval tx, mapping now live | SAFE | **SAFE** |

**15/15 matched expectation** (one case's initial expectation was corrected during review: the
"obligation positive control" case uses `role="director"` directly, an identity match, so the
contract's real, correct answer is `EXACT` not `SAFE` as first mis-typed in the test script --
the contract logic was right, the test author's expectation was wrong; corrected, not the
contract). All 15 independently re-confirmed via fresh on-chain `getDecision` reads.

## S17 in particular: a genuine, real governance-bypass demonstration

Unlike a single assertion, S17 is demonstrated as a real 3-step progression: (a) an unauthenticated
mapping is correctly unresolved before any proposal exists; (b) a single admin's real proposal
transaction alone is NOT sufficient -- the mapping still does not resolve; (c) only after a second,
genuinely independent admin's real approval transaction (real on-chain 2-of-3 threshold met) does
the mapping activate and correctly resolve. This is a real demonstration that a single compromised
or rogue admin cannot unilaterally widen the mapping table -- mirroring, on the EVM side, the same
real governance argument already established for A7 on the Fabric side.

## Combined S0-S19 reverse-direction status

| Covered with a dedicated real reverse-direction test | Argued as platform-symmetric or already covered elsewhere |
|---|---|
| S0, S1, S2, S3, S4, S5, S6, S7, S8, S9, S10, S12, S13, S14, S17 (15 scenarios, 18 real on-chain transactions total across both passes) | S11 (TOCTOU, platform-specific concurrency semantics), S15/S16 (PQ assurance claim function, already direction-independent per E5), S18 (already has real reverse-direction crypto evidence via X5a/X5b), S19 (TAC binding, already platform-symmetric) |

Every one of the 20 S-scenarios is now either dedicated-tested in the reverse direction or has an
explicit, checked justification for why it does not need a separate reverse instance -- none are
silently unaddressed.

## Honest scope notes

- Each scenario is n=1 (occasionally n=2 with a positive control), not a swept corpus -- this
  closes the guide's bidirectionality requirement at the mechanism level (does the reverse
  direction genuinely enforce the same property, with real transactions), not a statistically
  powered campaign matching this session's larger depth passes (A1/A3/A5/A8, S19/A10, E5).
- `EVMDestinationGateV2` does not itself re-verify Fabric-side cryptographic state (light-client/
  MPT verification is the separately-established `FabricNestedInclusionVerifier.sol`/
  `FabricRealTxVerifier.sol` layer); its job is the destination policy decision only, matching the
  scope of the prior S0/S2/S3 pass.

## Files

- `EVMDestinationGateV2_2026-09-24.sol` -- the new contract (copy for reproducibility; canonical
  source at `~/pqc-hardhat/contracts/EVMDestinationGateV2.sol`).
- `evm_reverse_direction_v2_result_2026-09-24.json` -- the 15 real on-chain results.
