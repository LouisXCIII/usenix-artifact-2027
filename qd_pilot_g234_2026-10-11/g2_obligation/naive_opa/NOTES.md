# Notes on implementation choices

- **Unrecognized `role`**: the spec only defines caps for five roles
  (`clerk`, `manager`, `director`, `gerente`, `gerente_senior`) and does not
  say what happens for any other role string. In this policy, `cap_map` and
  `half_cap_map` are only defined for those five roles, so a lookup for any
  other role is undefined and both `allow` rule bodies fail. Combined with
  `default allow = false`, this means an unrecognized role always results in
  DENY. This was not explicitly required by the spec but seemed the safest
  default ("fail closed") and does not add any extra restriction beyond what
  is written, since no amount/evidence combination for such a role is
  specified to GRANT.

- **Unrecognized `evidence_type`**: the spec explicitly says any
  unrecognized evidence_type string has strength 0 (same as `"none"`). This
  is implemented via the `strength(e)` function's `else = 0` fallback, which
  applies to any string (including `"none"` itself, and anything not in
  `strength_map`) that isn't one of `"notification"`, `"audit"`,
  `"dual_approval"`, or `"combined"`.

- **Half-of-cap values**: the spec gives a precomputed half-cap table and
  says "round down if odd." All five supplied caps are even, so the given
  half-cap table values are exact; I used the table values directly
  (`half_cap_map`) rather than recomputing `cap / 2` at evaluation time, to
  match the spec's numbers exactly and avoid any floating-point division
  behavior in Rego.

- **`amount <= 0`**: handled directly as a guard (`input.amount > 0`) in
  both `allow` rule bodies, so it is always DENY regardless of role or
  evidence, independent of whether the role/evidence are otherwise valid.

- **Boundary semantics**: "at or below half cap" uses `<=`, and "strictly
  above half of the cap" uses `>`, matching the spec's wording exactly. The
  overall cap check also uses `<=` ("if amount exceeds the role's cap" is
  DENY, so equal to the cap is still allowed, subject to the evidence rule
  for the upper half).

- No additional restrictions, exceptions, or logic were added beyond what is
  written in the specification.
