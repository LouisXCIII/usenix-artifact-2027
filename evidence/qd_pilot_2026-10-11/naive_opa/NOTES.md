# Destination Authorization Policy — Testing Notes

## Package / rule names

- Package: `destination_authz`
- Boolean decision rule: `allow` (default `false`, `true` exactly when the
  transfer should be GRANTed)
- String decision rule: `decision` (default `"DENY"`, becomes `"GRANT"`
  exactly when `allow` is true)

Both rules encode the identical decision; `decision` is provided only for
callers that prefer a string over a boolean.

## Query paths

Query one of:

- `data.destination_authz.allow` → `true` / `false`
- `data.destination_authz.decision` → `"GRANT"` / `"DENY"`

## Expected `input` JSON shape

```json
{
  "role": "manager",
  "amount": 50000,
  "tier": 2
}
```

- `role` (string): one of `clerk`, `manager`, `director`, `gerente`,
  `gerente_senior`. Any other value is undefined in the `caps` lookup and
  will cause `allow` to be `false` (DENY).
- `amount` (number): requested transfer amount in USD, a positive integer.
- `tier` (number): achieved risk tier, 0-3 inclusive. `0` always denies;
  any value outside `0-3` also denies.

## Example CLI invocations

Evaluate with `opa eval` against a local input file:

```bash
opa eval -i input.json -d policy.rego 'data.destination_authz.allow'
opa eval -i input.json -d policy.rego 'data.destination_authz.decision'
```

Or inline:

```bash
echo '{"role":"manager","amount":50000,"tier":2}' | \
  opa eval -I -d policy.rego 'data.destination_authz.allow'
```

## Worked examples (for sanity-checking a test harness)

Role caps: clerk=$10,000; manager=$500,000; director=$2,000,000;
gerente=$500,000; gerente_senior=$2,000,000.

| role     | amount  | tier | cap × fraction      | expected `allow` |
|----------|---------|------|----------------------|-------------------|
| clerk    | 1000    | 1    | 10,000 × 0.10 = 1000 | true (==, boundary) |
| clerk    | 1001    | 1    | 1000                 | false |
| manager  | 250000  | 2    | 500,000 × 0.50 = 250,000 | true (boundary) |
| manager  | 250001  | 2    | 250,000              | false |
| director | 2000000 | 3    | 2,000,000 × 1.0 = 2,000,000 | true (boundary) |
| director | 1       | 0    | n/a — tier 0 always denies | false |
| gerente_senior | 2000000 | 3 | 2,000,000 | true |
| unknown_role | 1  | 3    | role not in `caps`   | false |

Fractions are computed with exact integer arithmetic
(`amount * denominator <= cap * numerator`) rather than floating point, so
boundary amounts (e.g. exactly 10%/50%/100% of a cap) are handled without
rounding error.
