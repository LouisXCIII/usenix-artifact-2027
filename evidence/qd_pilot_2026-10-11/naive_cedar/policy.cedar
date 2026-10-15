// Destination Authorization Policy — v1
//
// Implements the destination-institution compliance rule for incoming
// transfer requests. See NOTES.md in this directory for the exact
// request/context JSON shape a caller should use to exercise this policy.
//
// Entities (chosen for this exercise; any concrete UIDs of the right type
// work — this policy does not read any entity attributes, only `context`):
//
//   principal : DestinationAuth::User       (the requester at the destination institution)
//   action    : DestinationAuth::Action::"SubmitTransfer"
//   resource  : DestinationAuth::Transfer    (the specific transfer being authorized)
//
// Context attributes (all required on every request):
//
//   context.role   : String  — one of "clerk", "manager", "director",
//                              "gerente", "gerente_senior"
//   context.amount : Long    — requested transfer amount in US dollars,
//                              a positive integer
//   context.tier   : Long    — achieved risk tier for this transfer, 0-3,
//                              already authenticated by an upstream
//                              verification layer (treated here as a
//                              trustworthy, given fact — not re-verified)
//
// Business rule:
//   - Each role has a maximum cap:
//       clerk           $10,000
//       manager         $500,000
//       director        $2,000,000
//       gerente         $500,000
//       gerente_senior  $2,000,000
//   - tier 0            -> always DENY
//   - tier 1            -> up to 10% of the role's cap allowed
//   - tier 2            -> up to 50% of the role's cap allowed
//   - tier 3            -> up to 100% of the role's cap allowed
//   - GRANT iff tier >= 1 AND amount <= (role's cap * tier's allowed fraction)
//
// Any role string outside the five known roles, or any tier outside 1-3,
// resolves to an allowed-amount threshold of 0, which denies every
// positive amount — i.e. unknown role/tier values are denied, not granted.
// Cedar's own default (no matching `permit`) is deny, so a single `permit`
// rule with the full grant condition below is sufficient; there is no
// need for a separate `forbid`.

permit (
    principal,
    action == DestinationAuth::Action::"SubmitTransfer",
    resource
)
when {
    context.tier >= 1 &&
    context.amount <= (
        if context.role == "clerk" then
            (if context.tier == 1 then 1000
             else (if context.tier == 2 then 5000
             else (if context.tier == 3 then 10000
             else 0)))
        else if context.role == "manager" then
            (if context.tier == 1 then 50000
             else (if context.tier == 2 then 250000
             else (if context.tier == 3 then 500000
             else 0)))
        else if context.role == "director" then
            (if context.tier == 1 then 200000
             else (if context.tier == 2 then 1000000
             else (if context.tier == 3 then 2000000
             else 0)))
        else if context.role == "gerente" then
            (if context.tier == 1 then 50000
             else (if context.tier == 2 then 250000
             else (if context.tier == 3 then 500000
             else 0)))
        else if context.role == "gerente_senior" then
            (if context.tier == 1 then 200000
             else (if context.tier == 2 then 1000000
             else (if context.tier == 3 then 2000000
             else 0)))
        else 0
    )
};
