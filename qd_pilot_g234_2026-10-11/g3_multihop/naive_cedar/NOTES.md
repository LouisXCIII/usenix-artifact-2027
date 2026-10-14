# NOTES — Multi-Hop Context Propagation Rule (Cedar)

## Representation choice

`hops` is modeled as a single `context` attribute named `hops`, typed as a
Cedar **Set of Bool** (`Set<Bool>`), not as an ordered list and not as five
separate optional booleans.

Why a Set is sufficient here even though the spec describes `hops` as an
*ordered* list: the GRANT rule is "logical AND over every element," which is
both order-independent and duplicate-independent — `[true, false, true]` and
`[false, true]` reduce to the identical verdict (DENY, because a `false` is
present). A Cedar `Set<Bool>` can only ever contain at most two distinct
values (`{}`, `{true}`, `{false}`, or `{true, false}`), and those four cases
exactly cover every outcome the rule can produce:

- `{}` (empty) → DENY (empty-list case)
- `{true}` → GRANT (no false present, non-empty)
- `{false}` or `{true, false}` → DENY (a false is present somewhere)

So collapsing duplicates and discarding order loses no information that the
decision rule actually uses. This was the simplest of the representations
the task explicitly allowed ("a Cedar Set ... of booleans").

Policy logic:

- `permit` fires when `context.hops != []` AND `!context.hops.contains(false)`.
- A mirroring `forbid` fires when `context.hops == []` OR `context.hops.contains(false)`,
  added defensively/explicitly so the DENY outcome is an explicit `forbid`
  match rather than relying only on Cedar's implicit "deny if nothing
  permits" default — both paths agree, so the net decision is identical
  either way.
- `principal`, `action`, and `resource` are left fully unconstrained (no
  `==`/`in` clauses) since the spec defines no entity/action schema; the
  decision depends only on `context.hops`.

## Empty-list handling

`hops` of length 0 is encoded as the empty Cedar set `[]`. The policy checks
`context.hops == []` explicitly and denies in that case, matching the spec's
"If `hops` is empty, DENY" rule.

## Exact input format the caller must use

When constructing a Cedar authorization request (e.g. via the `cedar-policy`
JSON request format, the Rust/Java/Go SDKs, or `cedar authorize --context`),
the caller must supply a `context` JSON object containing a key `"hops"`
whose value is a **JSON array of JSON booleans**, length 1 to 5 (per spec;
the policy itself does not enforce the 1-5 bound — see caveat below), e.g.:

```json
{
  "hops": [true, true, false, true]
}
```

Cedar's JSON encoding represents a `Set<Bool>` as a JSON array literal of
booleans — the same JSON shape a list would use — so no special wrapper or
type tag is needed; passing `[true, true, false, true]` as the value of
`hops` is exactly how `context.hops` becomes a Cedar `Set<Bool>` containing
`{true, false}` at evaluation time.

Important caveats for the calling program:

1. **Order and duplicates are not preserved.** Cedar collapses the JSON
   array into a true set. `[true, false]` and `[false, true, true]` produce
   the identical Cedar value (`{true, false}`) and therefore the identical
   decision. This is intentional and harmless for this rule (see above), but
   the caller must not expect the policy to echo back position-specific
   information — it only ever answers GRANT/DENY.
2. **Length bound (1–5) is not enforced by this policy file.** The spec's
   "1 to 5 elements" constraint is a property of the caller's input domain,
   not something expressible/necessary in this Cedar policy, since the
   decision rule (AND-of-all-elements, empty→DENY) is already total and
   correct for a set of any size, including sizes outside 1–5. If strict
   input validation of the length is required, it must be done by the
   calling program before constructing the request (e.g. in an optional
   Cedar schema with a validated context shape), not by this policy.
3. **Empty list** must be passed as `"hops": []` (an empty JSON array), which
   the policy denies.
