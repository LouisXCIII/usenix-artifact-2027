// hardened_cedar/policy.cedar -- full-knowledge upper-bound baseline (author = me) for
// spec_v1.md's multi-hop AND-over-all-hops rule. Written INDEPENDENTLY of the naive
// agent's Set<Bool> representation (which this harness's context ALSO carries, under
// the key `hops`, purely for the naive policy's own consumption) -- this hardened
// policy instead uses five explicit per-index attributes (hop1..hop5) plus hop_count,
// an enumeration-style encoding chosen before the naive agent's output was read.
permit(principal, action, resource)
when {
    context.hop_count >= 1 &&
    context.hop1 &&
    (context.hop_count < 2 || context.hop2) &&
    (context.hop_count < 3 || context.hop3) &&
    (context.hop_count < 4 || context.hop4) &&
    (context.hop_count < 5 || context.hop5)
};
