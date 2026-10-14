// Multi-Hop Context Propagation Rule
//
// GRANT iff every element of context.hops is `true` (logical AND over the
// whole list). DENY if any element is `false`, or if the list is empty.
//
// Representation choice (see NOTES.md for full detail):
//   context.hops is a Cedar Set<Bool> built from the caller's ordered
//   hops[0..n-1] list, n in 1..5. Because the decision rule only needs to
//   know "is there any `false` anywhere in the list" and "is the list
//   empty", a Set is sufficient: order and duplicate counts do not affect
//   the outcome, only the presence/absence of `false` and of any element
//   at all.

permit (
    principal,
    action,
    resource
)
when {
    context.hops != [] &&
    !context.hops.contains(false)
};

forbid (
    principal,
    action,
    resource
)
when {
    context.hops == [] ||
    context.hops.contains(false)
};
