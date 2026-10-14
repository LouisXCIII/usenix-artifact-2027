# E8, real on-chain analogs for A1, A3-A5, A8; architectural explanation for A7, A9 (2026-09-24)

Per the user's explicit request to continue deepening A1, A3-A5, A7-A9 --
the seven conjuncts that previously had no real on-chain analog at all
(only A2 and A6 did, `evidence/e8_value_sweep_2026-09-24.md`). All new
chaincode deployed as v2.11/seq23. The known chaincode-build infra bug
(`evidence/e8_a2_real_ablation_2026-09-23.md`) recurred a third time and
was fixed proactively, immediately, with the documented procedure, before
any test was attempted.

## A3: obligation preservation -- real analog already existed, now has a real ablated twin

`SubmitDecisionFull` (deployed since an earlier session) already has a real
obligation gate: `amount > threshold/2 && !secondApprovalObtained ->
CONDITIONAL`. New: `SubmitDecisionFullNoObligationCheck`
(`ablation_a3_a8.go`) is identical minus that one check.

| Case | secondApprovalObtained | Real | Ablated |
|---|---|---|---|
| director/\$1,500,000 (>half of \$2,000,000 threshold) | false | **CONDITIONAL** | **EXACT (incorrect)** |
| director/\$1,500,000 | true | EXACT | -- |
| director/\$500,000 (<=half, no obligation) | false | EXACT | -- |

**4/4 correct.**

## A8: replay protection -- real analog already existed (E4, N=100), now has a real ablated twin

New: `SubmitDecisionNoReplayCheck` (`ablation_a3_a8.go`), identical to
`SubmitDecision` minus the `decisionExists` gate. A first rapid-fire test
(no wait between submissions) showed a benign endorsement-time race --
exactly the same mechanism E4 already documented (attempts racing past
`decisionExists` before the first write commits, later resolved by MVCC) --
not a real finding about A8 itself; re-run cleanly with a real wait after
each commit:

| | Original | Resubmission (same id, different amount, after real commit) | Final ledger state |
|---|---|---|---|
| Real (`SubmitDecision`) | REJECT/\$600,000 committed | correctly rejected: `"already recorded (replay rejected)"` | unchanged, \$600,000/REJECT |
| Ablated (`SubmitDecisionNoReplayCheck`) | REJECT/\$600,000 committed | **accepted: SAFE/\$200,000** | **overwritten -- original silently lost** |

## A1: state correspondence -- real analog, real fabricated-header attack

`verifySourceStateNoHeaderVerification` (`ablation_lightclient.go`) skips
`verifyIbft2Header` (the block-hash reconstruction + pinned-validator
commit-seal recovery) entirely, trusting the caller-supplied StateRoot
directly. Real test: a header object with a genuinely real StateRoot (taken
from a verified block) but a fabricated hash/parentHash and garbage
`extraData` -- i.e. a header that could never have been produced by the
real validators.

| | Result | Detail |
|---|---|---|
| Real (`SubmitDecisionTrustless`) | **REJECT** | `malformed IBFT2 extraData` -- rejected before any seal is even checked |
| Ablated (`SubmitDecisionTrustlessNoHeaderVerification`) | **SAFE** | accepted -- the real, separately-obtained registry proof happens to verify under the fabricated header's (real) StateRoot |

## A4: freshness -- real analog, real 216-hour-stale header

`verifySourceStateNoFreshnessCheck` (`ablation_lightclient.go`) skips only
the `checkFreshness` call; header validity and the registry proof are still
fully, genuinely checked. Real test used the project's existing, genuinely
216-hour-old real (header, registry proof) pair
(`testdata/besu_header_0x5fc0c.json` + `registry_proof_honest.json`, from
2026-09-21) -- a fresh equally-old pair could not be fetched for this
session's current codeHash because **Besu prunes historical world state**
(`eth_getProof` at a 56-hour-old block returned `"World state unavailable"`,
confirmed directly), a real, disclosed infrastructure constraint.

| | Class | `lcVerified` | Detail |
|---|---|---|---|
| Real (`SubmitDecisionTrustless`) | REJECT | **false** | `header is 780030s old, exceeds the 86400s freshness bound` |
| Ablated (`SubmitDecisionTrustlessNoFreshnessCheck`) | REJECT (different, unrelated reason) | **true** | `header age NOT checked` -- 3/4 real commit seals, `provenApprovals=2`, `provenMlScanPassed=true` |

The overall class is REJECT under both paths, but for different reasons:
the ablated path's overall REJECT comes from the accompanying ZK proof not
matching this particular old registry's codeHash (an artifact of reusing
old 2026-09-21 fixtures with a current v7 proof, unavoidable given the
pruning constraint above) -- **not** from freshness. The correct comparison
is `lcVerified`/`lcDetail`: the real path correctly refuses to treat this
header as verified at all (`lcVerified=false`) once it is too old, while the
ablated path genuinely, cryptographically verifies the SAME header as valid
(`lcVerified=true`, real proven approvals and mlScanPassed) and simply never
asks how old it is. This isolates A4's real effect at the light-client layer.

## A5: finality (commit-seal quorum) -- real analog, real seal truncation

`verifyIbft2HeaderCustomQuorum` (`ablation_a5_quorum.go`) is an exact copy
of the real `verifyIbft2Header` (same RLP decoding, same block-hash
reconstruction, same ECDSA seal recovery -- nothing about *how* a seal is
checked is weakened) with the required signer count parameterized instead
of fixed at `ceil(2n/3)`. Real test: a real header's real `extraData` was
RLP-decoded, its seals list truncated from 3 genuine signatures to 1 (still
a real, cryptographically valid signature -- not forged), and re-encoded;
the block hash is unaffected since seals are not among the hashed fields
(confirmed unchanged after truncation).

| | Result | Detail |
|---|---|---|
| Real (`SubmitDecisionTrustless`, quorum=3 required) | **REJECT** | `only 1 distinct validator commit seals, quorum is 3` |
| Ablated (`SubmitDecisionTrustlessWeakQuorum`, quorum=1) | **SAFE** | accepted with 1/4 real signers |

## A7, A9: explained, not ablated -- both are governed at a different layer than a single transaction

**A7 (mapping/governance authentication)**: this project's real analog is
Fabric's own chaincode-lifecycle governance. `RoleMapping`/`RoleThreshold`
(the mapping table) are compiled into the chaincode itself, so changing
them requires the channel's real 2-of-2 (`Org1MSP`+`Org2MSP`) endorsement
policy for a chaincode commit -- exercised for real **11+ times this
session alone** (every version from v2.1 through v2.11, each requiring
both orgs' genuine approval, directly visible via
`peer lifecycle chaincode querycommitted`). This is not something a single
transaction can "ablate": the mapping table's governance happens at
deployment time, not evaluation time, so there is no per-transaction
ablated twin to build without literally deploying a chaincode that violates
the channel's own endorsement policy (which the ordering service itself
would refuse to commit).

**A9 (TOCTOU/state binding)**: already has real evidence
(`evidence/concurrency_toctou_experiment_2026-09-21.md`,
`evidence/e4_replay_toctou_limitations_2026-09-23.md`'s non-policy-relevant
control) showing Fabric's own optimistic concurrency control (MVCC
read-set validation at commit) genuinely catches concurrent state races.
This mechanism is automatic for every `GetState` call inside a chaincode
transaction's simulation phase -- it cannot be selectively disabled for one
function without disabling Fabric's own commit-time validation entirely, a
channel-level setting no chaincode function controls. Explained rather than
faked, matching this project's standing practice for genuinely
non-ablatable, platform-enforced properties.

## Totals

**5 of the 7 remaining conjuncts (A1, A3, A4, A5, A8) now have real,
on-chain ablated-vs-real demonstrations** -- 15 new real transactions this
pass (4 A3, 4 A8, 2 A1, 2 A4, 2 A5, plus the 3 sanity/setup checks).
Combined with A2 (all 4 families, fully swept) and A6 (full truth table)
from earlier this session, **9 of A1-A9 plus A10-A11 now have real
on-chain evidence; only A7 and A9 remain without a per-transaction ablated
twin, both explained as governed at a different layer (deployment
governance, platform concurrency control) rather than left unexplained.**
Raw output: `evidence/e8_a1_a3_a4_a5_a8_result_2026-09-24.json`.

## Still open, disclosed honestly

- A4's demonstration is confounded by an unrelated ZK-proof mismatch in the
  overall class (explained above); the light-client-layer fields
  (`lcVerified`/`lcDetail`) are the clean signal, not the top-level class.
- A1/A4/A5 each use n=1 representative attack construction, not a swept
  range (unlike A2/A6's later, more thorough sweep pass).
- A7/A9 remain, by design, without a per-transaction ablated twin --
  correctly attributed to a different governance layer rather than forced.
