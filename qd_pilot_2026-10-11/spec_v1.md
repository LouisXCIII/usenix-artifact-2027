# Destination Authorization Policy — Specification (v1)

## Context

A financial transfer system lets an external source attest, via a trusted verification layer,
three facts about an incoming transfer request:

1. The requester's **role** at the destination institution — one of: `clerk`, `manager`,
   `director`, `gerente`, `gerente_senior`.
2. The requested transfer **amount** in US dollars (a positive integer).
3. An **achieved risk tier** for this specific transfer — an integer from 0 to 3, established by
   the upstream verification layer and already authenticated. You do not need to verify it
   yourself; treat it as ground truth, an already-trustworthy input attribute.

## Your task

Write an authorization policy that grants or denies the transfer, implementing this business
rule exactly as given by the destination's compliance team:

- Each role has a maximum transfer cap (the role's full approved limit):
  - `clerk`: $10,000
  - `manager`: $500,000
  - `director`: $2,000,000
  - `gerente`: $500,000
  - `gerente_senior`: $2,000,000

- The achieved risk tier determines what **fraction** of the role's cap is allowed for this
  specific transfer:
  - tier 0: the transfer must be **DENIED** outright, regardless of role or amount (no valid
    risk assessment was achieved).
  - tier 1: up to **10%** of the role's cap is allowed.
  - tier 2: up to **50%** of the role's cap is allowed.
  - tier 3: up to **100%** of the role's cap is allowed.

- **GRANT** the transfer if and only if: tier is at least 1, AND the requested amount is less
  than or equal to (the role's cap × the tier's allowed fraction). Otherwise **DENY**.

- Treat role, amount, and achieved tier as given, already-trustworthy input attributes to your
  policy. You do not need to verify any cryptographic proof, inspect any blockchain data, or
  check any signature — a separate verification layer has already done that and hands you these
  three values as authenticated facts.

## Deliverable

Write a single **Cedar** policy file implementing exactly this rule, and a single **OPA/Rego**
policy file implementing exactly this rule. Both should take `role`, `amount`, and `tier` as
input and output a grant/deny decision.

## Important

Author this from the specification above alone. Do not search for, read, or reference any
existing codebase, research paper, prior implementation, or related files on this machine —
treat this as a blind authoring exercise using only the text of this specification. Do not run
`grep`/`find`/`git` or browse any directory other than the one you are asked to write your
output files into.
