# Authorization Policy Specification — Cross-Vocabulary Role Mapping Rule

You are asked to write an authorization policy from this specification alone. Do not
search for, read, or assume the existence of any other code, repository, or prior
conversation. Treat this page as the complete and only source of truth.

## Scenario

A request arrives with a role name written in the SOURCE system's own vocabulary. The
destination system does not know the source vocabulary directly — it only knows its
OWN role names and dollar caps. Before applying any dollar-amount check, the source
role name must first be translated to a destination role name using the table below.

## Input

- `source_role`: a string, the role name as written by the source system. It will be
  one of the five strings in the left column of the table below, OR it may be some
  other string not in the table at all (an unrecognized/unmapped source role).
- `amount`: a positive integer, in dollars.

## Source-to-destination role translation table

| source_role      | destination_role | destination cap |
|------------------|------------------|------------------|
| "gerente"        | "manager"        | 500,000          |
| "gerente_senior"  | "director"       | 2,000,000        |
| "funcionario"    | "clerk"          | 10,000           |
| "diretor"        | "director"       | 2,000,000        |
| "supervisor"     | "manager"        | 500,000          |

Note that TWO different source roles ("gerente_senior" and "diretor") both translate to
the SAME destination role ("director") with the SAME cap — this is intentional, not a
mistake to fix.

## The rule to implement

1. Look up `source_role` in the table above to find its `destination_role` and cap.
2. If `source_role` is NOT in the table (any string not exactly matching one of the
   five listed), the request must be DENY. Do not guess, do not fall back to the
   smallest or largest cap, do not treat an unrecognized role as any particular known
   role — an unmapped source role is always DENY.
3. If `source_role` IS in the table, GRANT if and only if `amount` is less than or
   equal to the destination cap for that role. If `amount` exceeds the cap, DENY.
4. `amount <= 0` is always DENY, independent of role.

## Examples (for your own understanding only — your policy must handle every source_role
string, not just these)

- `source_role = "gerente"`, `amount = 500000` → GRANT (at cap)
- `source_role = "gerente"`, `amount = 500001` → DENY (over cap)
- `source_role = "diretor"`, `amount = 1999999` → GRANT
- `source_role = "estagiario"`, `amount = 1` → DENY (not in the table — unmapped
  source role, regardless of how small the amount is)
- `source_role = "manager"`, `amount = 1` → DENY ("manager" is a DESTINATION-side role
  name, not one of the five SOURCE-side strings in the table — it is not in the
  left column, so it is unmapped and must be DENY)

## What to produce

Write ONE authorization policy file implementing exactly this rule, in the policy
language you are asked to use (Cedar or Rego/OPA — you will be told which). The policy
should take `source_role` and `amount` as input and produce a GRANT/DENY (or
allow=true/false) decision matching the rule above exactly, for every possible
`source_role` string (listed or not) and every `amount`.

Also write a short NOTES.md (a few sentences) describing any choices you made that the
spec did not fully pin down (for example, exactly how you implemented the "not in the
table" check).

Do not add any additional restrictions, exceptions, or logic beyond what is written
above. Do not consult any other document, codebase, or context.
