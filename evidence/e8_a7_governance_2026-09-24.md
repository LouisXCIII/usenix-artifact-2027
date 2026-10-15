# E8, A7 built deeper: a real per-transaction ablated twin via Fabric's own endorsement policy (2026-09-24)

`evidence/e8_remaining_ablations_2026-09-24.md` explained A7 (mapping/
governance authentication) as not admitting a per-transaction ablated twin
via chaincode logic, since the mapping table is compiled into the source
and its governance happens at deployment time. The user asked to go
deeper. There IS a real, per-transaction-comparable mechanism after all:
Fabric's chaincode **transaction endorsement policy** (distinct from the
*lifecycle* approval policy governing deployment itself, which genuinely
cannot be weakened without violating the channel's own policy). The
endorsement policy governs whether a *decision's submission* carries
genuine multi-party authorization -- exactly what "gov_ok_general" means
-- and it is configurable per chaincode definition via
`--signature-policy`, entirely through Fabric's own real mechanism, no new
Go code required.

## Design

- **Real** (chaincode v2.11/seq23): the channel's actual default policy,
  `MAJORITY Endorsement` (`configtx/configtx.yaml`) -- for 2 orgs, majority
  requires both.
- **Ablated** (chaincode v2.12/seq24, same source, only the commit-time
  `--signature-policy` differs): `OR('Org1MSP.member')` -- only Org1's
  endorsement is required. Deploying this itself still required genuine
  2-of-2 lifecycle approval (Org1MSP+Org2MSP), since *that* governance
  layer is real and was not touched -- only the resulting chaincode's own
  transaction-endorsement requirement was deliberately weakened, as a
  deployed configuration choice, to build the comparison.
- Both used the identical `SubmitDecision` source code; only the number of
  organizations required to endorse a call to it differs.

## Real results

| | Endorsed by | CLI response | Actual committed outcome |
|---|---|---|---|
| Real (v2.11, MAJORITY) | Org1 only | `status 200, SAFE` (endorsement/simulation succeeded) | **`ENDORSEMENT_POLICY_FAILURE`** (validation code 10, decoded from the real committed block) -- record does not exist |
| Ablated (v2.12, OR(Org1)) | Org1 only | `status 200, SAFE` | **Genuinely committed**: `amount=200000, class=SAFE` |

The real case's CLI response looked identical to the ablated case's
(`status 200, SAFE`) -- this is expected and consistent with everything
already established this session (E4, A8): `peer chaincode invoke`'s
response reflects only the endorsement/simulation step, not final
validation. The real difference only shows up in the actually committed
ledger state, confirmed here by decoding the real block's
`TRANSACTIONS_FILTER` metadata directly: validation code **10,
`ENDORSEMENT_POLICY_FAILURE`**, an exact, named real code, not an inferred
one.

## Restoration (mandatory, not optional)

Since this deliberately weakened the *live* chaincode's real transaction
governance, it was restored immediately after the comparison: chaincode
v2.13/seq25, same source, default policy (no `--signature-policy`
override). Verified for real:
- Single-org submission under v2.13 -> record does not exist (rejected
  again, governance restored).
- Normal dual-org submission under v2.13 -> record exists, `SAFE` (normal
  operation unaffected).

## Result

**4/4 real transactions/checks matched expectation** (real-rejects,
ablated-accepts, restored-rejects-again, restored-normal-operation-works).
A7 now has a genuine, real, per-transaction-comparable on-chain
demonstration after all -- built from Fabric's own real endorsement-policy
mechanism, not fabricated chaincode logic. Raw output:
`evidence/e8_a7_governance_result_2026-09-24.json`.

## What remains correctly unablated

The *lifecycle* governance layer itself (who may approve deploying a new
chaincode DEFINITION, i.e. who may change `RoleMapping`/`RoleThreshold`'s
source) was never weakened and was not the subject of this test --
deploying even the weak-transaction-policy v2.12 still required genuine
2-of-2 lifecycle approval throughout. That deeper layer remains what this
session's earlier explanation correctly described: enforced by the
channel's own configuration, not overridable by any chaincode-level action.
