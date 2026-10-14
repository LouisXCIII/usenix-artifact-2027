# S11, S15, S16, S18, S19 real reverse direction -- E3 full 20/20 closure (2026-09-24)

Direct follow-up to "làm nốt 5 kịch bản S còn thiếu" (finish the remaining 5 S-scenarios). The
previous two passes (`evm_reverse_direction_2026-09-24.md`, `evm_reverse_direction_v2_2026-09-24.md`)
covered 15/20 scenarios with dedicated real tests and argued the remaining 5 (S11, S15, S16, S18,
S19) did not need one. This pass closes all 5 with real, dedicated on-chain mechanisms instead,
bringing the S0-S19 reverse-direction (Fabric -> EVM) coverage to a genuine **20/20**.

## S11 (TOCTOU): real two-phase propose/execute, real race

New contract `EVMDestinationGateV3.sol` adds a genuine two-transaction flow (`proposeExecution`
then `executeDecision`) -- V1/V2's `submitDecision` is atomic (analysis and execution are the same
transaction), which structurally has no TOCTOU window at all; V3 adds the window deliberately so
S11 has something real to test.

- **Case A (unauthorized race)**: a real second identity (`PRIVATE_KEY_B`) attempts a real
  concurrent `setBoundStateHash` call fired alongside the real `executeDecision` call. Real result:
  the attacker's call correctly reverts (not the contract owner), so the race cannot occur at all
  -- `executeDecision` proceeds and correctly returns `SAFE`.
- **Case B (genuine authorized state drift)**: the real contract owner performs a real state change
  BEFORE `executeDecision` is called. Real result: **`REJECT`**, `"TOCTOU blocked -- state changed
  between analysis and execution"`.

This demonstrates two real, independent defenses (matching the "two layers of defense in depth"
pattern already established for A9/E4 this session): access control prevents an unauthorized race
from occurring at all, and the state-hash recheck catches a genuine, authorized state change that
lands between analysis and execution.

## S15/S16 (assurance-tier claim vs. verified): grounded in the real E5 result

A real `claimsFor()`/`submitAssuranceDecision()` pair, using the SAME real E5 dual-implementation
artifacts already produced this session (`e5_dual_implementation_v4_result_2026-09-24.json`):
the real valid-signature transaction (both Liboqs and CIRCL independently verified `true`) and the
real tampered-signature transaction (both verified `false`) serve as the real `verifiedTier`
ground truth, not synthetic values.

- **S15**: classical decision + PQ channel, honestly claimed classical, PQ required ->
  **`REJECT`** (a classical decision does not itself satisfy a PQ requirement).
- **S16**: the SAME real tampered-signature artifact, but `claimedTier="pq"` (a fake upgrade
  claim) -> **`REJECT`** (the real verified tier, not the claim, gates the decision).
- **Positive control**: the real valid-signature artifact, genuinely `pq`-verified by both real
  implementations -> **`SAFE`**.

## S18 (multi-hop stripping): fed a real Fabric hop-chain result

Rather than re-implement Fabric's hop-chaining mechanism on EVM, `submitMultiHopFinal()` takes the
REAL accumulated `composedContextOk` value, read moments earlier via `ReadHop` from a genuine,
freshly-committed 2-hop Fabric chain (`multihop_real.go`'s already-established
`SubmitHopDecision`/`SubmitHopDecisionPairwiseLocal`):

- A real composed chain with context stripped at hop 1 (`contextAtHop=false`) produced real
  `composedContextOk=false` on Fabric -> fed into the EVM final hop -> **`REJECT`**.
- The real pairwise-local twin of the same injected-loss chain produced `class=SAFE` on Fabric
  (the ablated design's real vulnerability) -> passed into EVM as `pairwiseLocalOk=true`,
  confirming the EVM final hop's composed check catches what the pairwise-local design would have
  missed.
- A real, unstripped 2-hop chain (`composedContextOk=true`) -> **`SAFE`**.

## S19 (TAC substitution): the existing TACVerifier.sol, now Fabric-bound

Reuses `TACVerifier.sol` (unchanged, already validated for 50/50 real substitution attempts
earlier this session) but issues a new real certificate with `h(D_s)` bound to a genuinely real
FABRIC decisionID (`a3-real-manager-at_cap-...`) rather than the Besu-sourced artifacts used in
the original S19/A10 pass -- making this instance a genuine Fabric -> EVM test. All 5 real
substitution types plus the positive control were run against it.

**6/6 matched expectation** (1 positive control + 5 substitution types, all correctly rejected
except the unmodified positive control).

## Combined real results

The results files record **7 primary S11/S15/S16/S18 on-chain outcomes**
(`evm_reverse_direction_v3_result_2026-09-24.json`) plus **6 primary S19 outcomes**
(`evm_s19_fabric_bound_result_2026-09-24.json`) -- **13/13 matched expectation**. Each primary
outcome's transaction is a real, separately mined Besu transaction; several scenarios also
required real setup transactions not separately itemized as their own outcome (S11's real
`proposeExecution`/`setBoundStateHash` calls; S19's real certificate `issue()` call), all also
real and separately mined. All 7 S11/S15/S16/S18 outcomes independently re-confirmed via fresh
on-chain `getDecision` reads; the S19 certificate's persisted storage independently re-confirmed
via a fresh `certs` read.

## Final S0-S19 reverse-direction status: 20/20

Combined with the two prior passes (S0/S2/S3, then S1/S4-S10/S12-S14/S17), every one of the 20
S-scenarios now has a dedicated, real, on-chain reverse-direction (Fabric -> EVM) test. None
argued away, none silently skipped.

## Honest scope notes

- Each of these final 5 scenarios is tested with n=1-2 real cases (not a swept corpus), matching
  the scope of the S1-S17 pass -- this closes the guide's bidirectionality requirement at the
  mechanism level, not as a statistically powered campaign.
- S18's EVM contract takes the composed/pairwise-local flags as real, externally-supplied values
  (read from a real, separately-verified Fabric chain) rather than re-deriving them on-chain --
  consistent with this pass's scope of testing the destination POLICY consequence of a real
  composed value, not re-implementing Fabric's hop-chaining mechanism a second time on EVM.
- S11's contract adds a genuine two-phase flow specifically to give this property something real
  to test; V1/V2's atomic single-transaction design remains the paper's primary real mechanism and
  is not modified.

## Files

- `EVMDestinationGateV3_2026-09-24.sol` -- the new contract for S11/S15/S16/S18 (copy for
  reproducibility; canonical source at `~/pqc-hardhat/contracts/EVMDestinationGateV3.sol`).
- `evm_reverse_direction_v3_result_2026-09-24.json` -- the 7 real S11/S15/S16/S18 on-chain results.
- `evm_s19_fabric_bound_result_2026-09-24.json` -- the 6 real S19 on-chain results.
