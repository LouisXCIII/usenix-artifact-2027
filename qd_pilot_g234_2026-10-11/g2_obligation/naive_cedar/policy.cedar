// Authorization policy: Obligation / Evidence-Strength Rule
//
// Entities (chosen for this policy; not constrained by the spec):
//   principal : User
//   action    : Action::"Transfer"
//   resource  : Account
//
// Inputs are modeled as context attributes on the request:
//   context.role           : String  ("clerk" | "manager" | "director" | "gerente" | "gerente_senior")
//   context.amount         : Long    (dollars; positive integer expected)
//   context.evidence_type  : String  ("none" | "notification" | "audit" | "dual_approval" | "combined" | other)
//
// Design: Cedar's authorization semantics default to DENY unless some
// `permit` matches and no `forbid` matches. This policy relies on that
// default-deny behavior to express every DENY case (amount <= 0, amount
// over the role cap, unrecognized role, and insufficient evidence above
// half-cap) simply by NOT writing a permit for it. No `forbid` statements
// are needed or used.
//
// Evidence strength (spec table): none=0, notification=1, audit=2,
// dual_approval=3, combined=4, anything unrecognized=0. Only strength >= 3
// is ever required by this rule, so it is checked directly as
// `evidence_type == "dual_approval" || evidence_type == "combined"`.

// ---- clerk: cap 10,000 / half-cap 5,000 ----

permit (
  principal,
  action == Action::"Transfer",
  resource
)
when {
  context.role == "clerk" &&
  context.amount > 0 &&
  context.amount <= 5000
};

permit (
  principal,
  action == Action::"Transfer",
  resource
)
when {
  context.role == "clerk" &&
  context.amount > 5000 &&
  context.amount <= 10000 &&
  (context.evidence_type == "dual_approval" || context.evidence_type == "combined")
};

// ---- manager: cap 500,000 / half-cap 250,000 ----

permit (
  principal,
  action == Action::"Transfer",
  resource
)
when {
  context.role == "manager" &&
  context.amount > 0 &&
  context.amount <= 250000
};

permit (
  principal,
  action == Action::"Transfer",
  resource
)
when {
  context.role == "manager" &&
  context.amount > 250000 &&
  context.amount <= 500000 &&
  (context.evidence_type == "dual_approval" || context.evidence_type == "combined")
};

// ---- director: cap 2,000,000 / half-cap 1,000,000 ----

permit (
  principal,
  action == Action::"Transfer",
  resource
)
when {
  context.role == "director" &&
  context.amount > 0 &&
  context.amount <= 1000000
};

permit (
  principal,
  action == Action::"Transfer",
  resource
)
when {
  context.role == "director" &&
  context.amount > 1000000 &&
  context.amount <= 2000000 &&
  (context.evidence_type == "dual_approval" || context.evidence_type == "combined")
};

// ---- gerente: cap 500,000 / half-cap 250,000 ----

permit (
  principal,
  action == Action::"Transfer",
  resource
)
when {
  context.role == "gerente" &&
  context.amount > 0 &&
  context.amount <= 250000
};

permit (
  principal,
  action == Action::"Transfer",
  resource
)
when {
  context.role == "gerente" &&
  context.amount > 250000 &&
  context.amount <= 500000 &&
  (context.evidence_type == "dual_approval" || context.evidence_type == "combined")
};

// ---- gerente_senior: cap 2,000,000 / half-cap 1,000,000 ----

permit (
  principal,
  action == Action::"Transfer",
  resource
)
when {
  context.role == "gerente_senior" &&
  context.amount > 0 &&
  context.amount <= 1000000
};

permit (
  principal,
  action == Action::"Transfer",
  resource
)
when {
  context.role == "gerente_senior" &&
  context.amount > 1000000 &&
  context.amount <= 2000000 &&
  (context.evidence_type == "dual_approval" || context.evidence_type == "combined")
};
