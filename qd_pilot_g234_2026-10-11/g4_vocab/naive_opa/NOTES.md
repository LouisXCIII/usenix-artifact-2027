# Notes

- **Package**: `authz.rolemap`. Chosen arbitrarily; the spec did not require a
  specific package name.

- **"Not in the table" check**: implemented via a plain Rego object lookup,
  `role_caps[input.source_role]`. `role_caps` is a fixed object keyed by the
  five source-side role strings from the spec's table, each value being that
  role's destination cap (the destination *role name* itself, e.g.
  `"manager"`/`"director"`/`"clerk"`, is not needed for the decision logic
  and so is left out of the data structure — only the cap matters for
  GRANT/DENY). When `input.source_role` is not one of those five exact
  strings, the lookup is undefined in Rego (not an error), which makes the
  `allow` rule body fail to complete, which in turn falls through to
  `default allow := false`. This means an unrecognized role is denied by
  construction rather than by an explicit `else`/negation branch — there is
  no code path that could accidentally substitute a smallest/largest/any
  particular cap for an unmapped role.

- **`amount <= 0`**: handled by the explicit `input.amount > 0` guard in the
  `allow` body, independent of whether the role is recognized. Combined with
  the lookup above, DENY is the result for every amount <= 0 regardless of
  `source_role`.

- **Duplicate destination mapping**: `gerente_senior` and `diretor` are
  simply two separate keys in `role_caps`, both with value `2000000`. This
  reproduces the spec's intentional many-to-one mapping without any special
  casing.

- **`decision` mirror**: implemented with Rego's `else` construct
  (`decision := "GRANT" { allow } else := "DENY" { true }`), which is
  equivalent to `allow`'s true/false but rendered as the strings
  `"GRANT"`/`"DENY"` as requested by the task description. It adds no
  additional logic beyond what `allow` already computes.

- **Exact-match semantics**: the lookup is a literal string-equality match
  against the five table keys; no trimming, case-folding, or partial
  matching is performed, since the spec gives no indication that
  `source_role` values should be normalized before comparison (e.g.
  `"Gerente"` or `"gerente "` would not match `"gerente"` and would be
  DENY).
