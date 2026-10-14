# Notes on implementation choices

- **Entity types**: the spec does not constrain the Cedar entity model, so I
  picked `principal : User`, `action == Action::"Transfer"`, and
  `resource : Account`. None of these are referenced by the rule logic itself
  (only `context` is), so any reasonable choice would work identically.

- **Inputs as context**: `role`, `amount`, and `evidence_type` are modeled as
  `context.role` (String), `context.amount` (Long), and `context.evidence_type`
  (String), exactly as instructed.

- **Default-deny instead of explicit `forbid`**: Cedar denies a request
  unless some `permit` matches (and no `forbid` matches). I used only
  `permit` statements for the exact GRANT conditions in the spec (amount in
  (0, half-cap] for any evidence; amount in (half-cap, cap] with evidence
  strength >= 3). Every DENY case in the spec — `amount <= 0`, `amount` over
  the role's cap, an unrecognized `role`, and insufficient evidence above
  half-cap — simply has no matching `permit`, so it falls through to the
  implicit DENY. This avoids writing any extra `forbid` logic that isn't in
  the spec.

- **Evidence strength check**: rather than building a numeric
  strength-lookup table in Cedar (which has no map/dictionary literal
  suitable for this), the "strength >= 3" condition is expressed directly as
  `evidence_type == "dual_approval" || evidence_type == "combined"`, since
  those are the only two evidence types whose strength reaches 3 per the
  spec's table. Any other string (including `"none"`, `"notification"`,
  `"audit"`, or any unrecognized value) is correctly treated as insufficient
  (strength < 3) by simply not matching this condition.

- **Unrecognized `role`**: the spec only defines caps/half-caps for the five
  listed roles. A request with any other `role` value matches no `permit`
  rule and is therefore DENY, which seemed the only sensible behavior given
  no cap is defined for it.

- **Caps and half-caps as literals**: the per-role cap and half-cap values
  are hard-coded directly from the spec's own precomputed tables (e.g.
  clerk: 5,000 / 10,000) rather than computed at evaluation time (e.g.
  `amount <= cap / 2`), to avoid any ambiguity around integer-division
  rounding and to keep the policy's conditions simple boundary comparisons.

- **No additional restrictions**: no extra conditions (time windows,
  resource identity, principal attributes, etc.) were added beyond what the
  spec describes.
