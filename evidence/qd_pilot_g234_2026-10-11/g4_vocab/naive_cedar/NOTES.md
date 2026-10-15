# Notes on implementation choices

- **"Not in the table" check**: Rather than writing an explicit `forbid` rule for
  unmapped roles, I relied on Cedar's default-deny semantics. The single `permit`
  policy's `when` clause is a disjunction of exactly the five `(source_role, cap)`
  pairs from the table. Any `source_role` string that is not one of the five exact
  matches (whether it's genuinely unrecognized like `"estagiario"`, or a
  destination-side name like `"manager"` that only superficially resembles a valid
  value) fails every disjunct, so no `permit` policy matches, and Cedar's
  implicit default denies the request. This avoids needing to enumerate or pattern-
  match "any other string" explicitly.

- **`amount <= 0` always deny**: each disjunct requires `context.amount > 0` in
  addition to the upper-bound cap check, so non-positive amounts are denied
  regardless of role, independent of the table lookup.

- **No `forbid` statements**: since Cedar denies by default, a `forbid` is only
  needed to override a `permit` that would otherwise fire. Because the single
  `permit`'s condition already encodes the full grant rule exactly ("grant iff
  role is known AND amount is in (0, cap]"), no `forbid` is necessary or added.

- **`principal`, `action`, `resource`**: the spec only talks about `source_role`
  and `amount`, both modeled as `context` attributes. The policy scope uses
  unconstrained `principal`, `action`, `resource` (i.e., it does not restrict by
  who is asking, what action, or what resource) since the spec gives no entity
  types or hierarchy to scope against. All authorization logic lives in the
  `when` clause over `context`.

- **Types**: `source_role` is treated as a Cedar `String`; `amount` as a Cedar
  `Long` (integer). Numeric literals for the caps (500000, 2000000, 10000) are
  written without comma separators, as Cedar does not support them.

- Per the spec, the duplicate mapping of `"gerente_senior"` and `"diretor"` to
  the same destination role/cap is intentional and preserved as two separate
  disjuncts with the same bound (2000000), not merged or deduplicated in any
  way that would change behavior.
