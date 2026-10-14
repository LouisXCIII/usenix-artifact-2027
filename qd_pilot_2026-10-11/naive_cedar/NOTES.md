# Testing `policy.cedar`

## Entity / action naming chosen

- Principal entity type: `DestinationAuth::User`
- Resource entity type: `DestinationAuth::Transfer`
- Action: `DestinationAuth::Action::"SubmitTransfer"` (the single action this
  policy scopes to — the policy's `action ==` clause requires exactly this
  action id; any request for a different action is denied by Cedar's
  default-deny, not by this policy)

The policy never reads attributes off the principal or resource entities —
only off `context`. So any concrete entity UIDs of the right type work; they
just need to exist (with no particular attributes) if your evaluator
validates entity references. A minimal `entities.json` with two bare
entities (no `attrs`) is enough:

```json
[
  { "uid": { "type": "DestinationAuth::User", "id": "requester-1" }, "attrs": {}, "parents": [] },
  { "uid": { "type": "DestinationAuth::Transfer", "id": "txn-1" }, "attrs": {}, "parents": [] }
]
```

## Request shape

```json
{
  "principal": { "type": "DestinationAuth::User", "id": "requester-1" },
  "action": { "type": "DestinationAuth::Action", "id": "SubmitTransfer" },
  "resource": { "type": "DestinationAuth::Transfer", "id": "txn-1" },
  "context": {
    "role": "manager",
    "amount": 100000,
    "tier": 2
  }
}
```

(If your tooling wants Cedar's entity-UID string form instead of the
`{type, id}` object form, use `DestinationAuth::User::"requester-1"`,
`DestinationAuth::Action::"SubmitTransfer"`, and
`DestinationAuth::Transfer::"txn-1"` respectively — same request.)

`context.role` must be a Cedar `String`, one of: `"clerk"`, `"manager"`,
`"director"`, `"gerente"`, `"gerente_senior"`. `context.amount` and
`context.tier` must be Cedar `Long` (integers, not strings) — JSON numbers
with no decimal point, as in the example above.

## Expected decisions (a few worked examples)

- `role=manager, amount=50000, tier=1` -> cap $500,000 x 10% = $50,000 ->
  `amount <= 50000` -> **Allow**
- `role=manager, amount=50001, tier=1` -> **Deny** (one dollar over the
  tier-1 threshold)
- `role=director, amount=2000000, tier=3` -> cap $2,000,000 x 100% ->
  **Allow**
- `role=clerk, amount=1, tier=0` -> tier 0 always denies regardless of
  amount -> **Deny**
- `role=gerente_senior, amount=1000000, tier=2` -> cap $2,000,000 x 50% =
  $1,000,000 -> `amount <= 1000000` -> **Allow**
- `role="auditor", amount=1, tier=3` -> unrecognized role resolves to a
  $0 allowed threshold -> **Deny**
- `role=manager, amount=1, tier=4` -> tier outside 0-3 resolves to a $0
  allowed threshold (and isn't tier 0, so the `tier >= 1` guard doesn't
  catch it either) -> still **Deny**, via the threshold-is-0 path

## How to run it (e.g. with the `cedar` CLI)

```
cedar authorize \
  --policies policy.cedar \
  --entities entities.json \
  --principal 'DestinationAuth::User::"requester-1"' \
  --action 'DestinationAuth::Action::"SubmitTransfer"' \
  --resource 'DestinationAuth::Transfer::"txn-1"' \
  --context context.json
```

where `context.json` holds just the `{"role": ..., "amount": ..., "tier": ...}`
object shown above.
