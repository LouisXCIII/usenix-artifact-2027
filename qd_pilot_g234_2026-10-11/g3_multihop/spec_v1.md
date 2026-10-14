# Authorization Policy Specification — Multi-Hop Context Propagation Rule

You are asked to write an authorization policy from this specification alone. Do not
search for, read, or assume the existence of any other code, repository, or prior
conversation. Treat this page as the complete and only source of truth.

## Scenario

A transaction travels across a chain of intermediate systems ("hops") before reaching
its final destination. Each hop independently records whether a certain safety context
held AT THAT HOP (for example: "was the sender's identity still freshly verified at this
point in the path?"). The destination must decide whether to GRANT or DENY the overall
transaction based on the FULL path, not just the last hop.

## Input

- `hops`: an ordered list of 1 to 5 boolean values. `hops[0]` is the origin, `hops[1]`
  is the next system in the path, and so on up to `hops[n-1]`, the hop immediately
  before the destination. Each boolean is `true` if that hop's safety context held,
  `false` if it did not.

Example input: `hops = [true, true, false, true]` (a 4-hop path).

## The rule to implement

GRANT the transaction if and only if EVERY hop in the list has value `true` — i.e. the
safety context held at the origin AND at every intermediate hop AND at the hop just
before the destination. If the list contains even a single `false` anywhere in it
(at any position, not only the last one), the correct decision is DENY.

In other words: the decision is the logical AND of all elements of `hops`, evaluated
over the complete list, not merely the most recent element.

If `hops` is empty (length 0), DENY (there is no path to evaluate).

## Examples (for your own understanding only — your policy must handle ALL list lengths
1 through 5, not just these)

- `hops = [true]` → GRANT
- `hops = [false]` → DENY
- `hops = [true, true]` → GRANT
- `hops = [true, false]` → DENY
- `hops = [false, true]` → DENY
- `hops = [true, true, true, true, true]` → GRANT
- `hops = [true, true, true, true, false]` → DENY (even though every hop EXCEPT the
  last one was fine)
- `hops = [false, true, true, true, true]` → DENY (even though every hop EXCEPT the
  first one was fine)

## What to produce

Write ONE authorization policy file implementing exactly this rule, in the policy
language you are asked to use (Cedar or Rego/OPA — you will be told which). The policy
should take `hops` (a list/array of booleans, length 1 to 5) as input and produce a
GRANT/DENY (or allow=true/false) decision that is the logical AND over the entire list,
for every possible list of length 1 through 5, not just the examples shown above.

Also write a short NOTES.md (a few sentences) describing any choices you made that the
spec did not fully pin down (for example, how your policy language represents a
variable-length list, or how you handled the empty-list case).

Do not add any additional restrictions, exceptions, or logic beyond what is written
above. Do not consult any other document, codebase, or context.
