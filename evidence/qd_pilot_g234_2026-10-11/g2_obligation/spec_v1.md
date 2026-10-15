# Authorization Policy Specification — Obligation / Evidence-Strength Rule

You are asked to write an authorization policy from this specification alone. Do not
search for, read, or assume the existence of any other code, repository, or prior
conversation. Treat this page as the complete and only source of truth.

## Scenario

A destination system must decide whether to GRANT or DENY a money-transfer request,
given three inputs:

- `role`: one of `"clerk"`, `"manager"`, `"director"`, `"gerente"`, `"gerente_senior"`.
- `amount`: a positive integer, in dollars.
- `evidence_type`: a string describing what supporting evidence was supplied with the
  request. It is one of `"none"`, `"notification"`, `"audit"`, `"dual_approval"`,
  `"combined"`, or possibly some other string not in this list.

## Role dollar caps (hard upper limit regardless of evidence)

| role            | cap       |
|-----------------|-----------|
| clerk           | 10,000    |
| manager         | 500,000   |
| director        | 2,000,000 |
| gerente         | 500,000   |
| gerente_senior  | 2,000,000 |

If `amount` exceeds the role's cap, the request is always DENY, no matter what evidence
is supplied.

## Evidence strength levels

Each evidence type has a strength level:

| evidence_type   | strength |
|-----------------|----------|
| (unrecognized / anything not listed below) | 0 |
| "none"          | 0 |
| "notification"  | 1 |
| "audit"         | 2 |
| "dual_approval" | 3 |
| "combined"      | 4 |

## The rule to implement

For amounts at or below HALF of the role's cap, no extra evidence is required: GRANT
(subject still to the role cap above).

For amounts strictly above half of the role's cap (but still at or below the full cap),
the request additionally requires evidence of strength **3 or higher** (i.e.
`"dual_approval"` or `"combined"`). If the supplied evidence's strength is below 3,
DENY even though the amount is within the role's cap.

`amount <= 0` is always DENY (invalid request), independent of role/evidence.

Half-of-cap thresholds, computed once for convenience (integer, round down if odd):

| role            | half-cap  |
|-----------------|-----------|
| clerk           | 5,000     |
| manager         | 250,000   |
| director        | 1,000,000 |
| gerente         | 250,000   |
| gerente_senior  | 1,000,000 |

## What to produce

Write ONE authorization policy file implementing exactly this rule, in the policy
language you are asked to use (Cedar or Rego/OPA — you will be told which). The policy
should take `role`, `amount`, and `evidence_type` as input and produce a GRANT/DENY (or
allow=true/false) decision matching the table above exactly, for every combination of
role, amount, and evidence_type, not just the boundary values shown in the tables.

Also write a short NOTES.md (a few sentences) describing any choices you made that the
spec did not fully pin down (for example, how you handled an `evidence_type` string that
does not appear in the strength table).

Do not add any additional restrictions, exceptions, or logic beyond what is written
above. Do not consult any other document, codebase, or context.
