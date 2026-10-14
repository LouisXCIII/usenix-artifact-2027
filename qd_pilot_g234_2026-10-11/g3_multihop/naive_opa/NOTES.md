# Notes

- `input.hops` is represented as an ordinary JSON array of booleans (Rego's native
  array type), so the same `allow` rule works unchanged for any length from 1 to 5 —
  there is no per-length branching; `every hop in input.hops { hop == true }`
  iterates over however many elements are present.
- Empty list: `count(input.hops) > 0` is required in the `allow` body, so
  `hops = []` fails that condition and falls through to `default allow := false`
  (DENY), matching the spec's explicit empty-list case.
- Lengths beyond 5 or non-boolean elements are not validated/rejected; the spec
  states input is always 1-5 booleans, so no extra input-shape checking was added
  beyond what's needed to compute the AND.
- `decision` is provided as a simple GRANT/DENY string mirror of `allow`, built
  from two complementary rules (`decision := "GRANT" if allow` /
  `decision := "DENY" if not allow`) rather than string concatenation, to keep it
  a plain total function of `allow`.
- Used `import rego.v1` (and its `if`/`every` keywords) for a modern, readable
  rule style; this requires an OPA version that supports rego.v1 (OPA >= 0.59 or
  any version with `import future.keywords` support), which was not specified but
  is a reasonable default assumption.
