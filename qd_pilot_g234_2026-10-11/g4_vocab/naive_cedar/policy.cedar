// Cross-Vocabulary Role Mapping Rule
//
// A request is identified only by two context attributes:
//   - context.source_role : string, the role name in the SOURCE system's vocabulary
//   - context.amount      : long,   a dollar amount
//
// Translation table (source_role -> destination_role / cap):
//   "gerente"         -> "manager"  / 500000
//   "gerente_senior"  -> "director" / 2000000
//   "funcionario"     -> "clerk"    / 10000
//   "diretor"         -> "director" / 2000000
//   "supervisor"      -> "manager"  / 500000
//
// Cedar's authorization semantics are default-deny: a request is only
// authorized if some `permit` policy matches it (and no `forbid` policy
// matches it). There is exactly one `permit` policy below, and it matches
// a request if and only if source_role is one of the five known strings
// AND amount is strictly positive AND amount does not exceed that role's
// destination cap. Any source_role not listed (including a destination-side
// name like "manager") never satisfies any clause, so it falls through to
// the implicit default-deny. Likewise amount <= 0 never satisfies any
// clause, so it is always denied regardless of role.

permit (
    principal,
    action,
    resource
)
when {
    (context.source_role == "gerente"        && context.amount > 0 && context.amount <= 500000)   ||
    (context.source_role == "gerente_senior" && context.amount > 0 && context.amount <= 2000000)  ||
    (context.source_role == "funcionario"    && context.amount > 0 && context.amount <= 10000)    ||
    (context.source_role == "diretor"        && context.amount > 0 && context.amount <= 2000000)  ||
    (context.source_role == "supervisor"     && context.amount > 0 && context.amount <= 500000)
};
