// hardened_cedar/policy.cedar -- the "hardened" arm (full-knowledge author = me), implementing
// spec_v1.md's rule exactly for all 5 roles (role cap * tier fraction, tier>=1 required).
// This is the upper-bound baseline the pilot asks for: deliberately NOT given to a fresh
// context-free author, written with the real table already known.
permit(principal, action == DestinationAuth::Action::"SubmitTransfer", resource)
when {
    context.tier >= 1 &&
    ((context.role == "clerk"          && context.tier == 1 && context.amount <= 1000) ||
     (context.role == "clerk"          && context.tier == 2 && context.amount <= 5000) ||
     (context.role == "clerk"          && context.tier == 3 && context.amount <= 10000) ||
     (context.role == "manager"        && context.tier == 1 && context.amount <= 50000) ||
     (context.role == "manager"        && context.tier == 2 && context.amount <= 250000) ||
     (context.role == "manager"        && context.tier == 3 && context.amount <= 500000) ||
     (context.role == "director"       && context.tier == 1 && context.amount <= 200000) ||
     (context.role == "director"       && context.tier == 2 && context.amount <= 1000000) ||
     (context.role == "director"       && context.tier == 3 && context.amount <= 2000000) ||
     (context.role == "gerente"        && context.tier == 1 && context.amount <= 50000) ||
     (context.role == "gerente"        && context.tier == 2 && context.amount <= 250000) ||
     (context.role == "gerente"        && context.tier == 3 && context.amount <= 500000) ||
     (context.role == "gerente_senior" && context.tier == 1 && context.amount <= 200000) ||
     (context.role == "gerente_senior" && context.tier == 2 && context.amount <= 1000000) ||
     (context.role == "gerente_senior" && context.tier == 3 && context.amount <= 2000000))
};
