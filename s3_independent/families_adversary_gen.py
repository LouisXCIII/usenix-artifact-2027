#!/usr/bin/env python3
import json, itertools, collections, os

I64MAX = 2**63 - 1
I64MIN = -2**63
CAPS = {"manager": 500000, "director": 2000000, "clerk": 10000}
PROF = {"empty": 0, "notification": 1, "audit": 2, "dual_approval": 3, "combined": 4}
MECH = {"none": 0, "notification": 1, "audit": 2, "dual_approval": 3, "combined": 4}
GRANTS = {"classical": {"ClassicalDecisionAuth"},
          "hybrid": {"ClassicalDecisionAuth", "PQDecisionAuth"},
          "pq": {"PQDecisionAuth"},
          "unverifiable": set()}
REQS = ["ClassicalDecisionAuth", "PQDecisionAuth"]

BAD_ROLES = ["Manager", "manager ", " manager", "MANAGER", "admin", "", "Director", "director\t",
             "clerk\n", "CLERK", "managers", "mana ger", "mаnager", "root", "manager,director",
             "director ", "Clerk", "superuser", "manager\u0000", "​director"]

def family(target, a):
    if target == "action":
        return 0 <= a["actionLevel"] <= 2
    if target == "resource":
        return a["resourceID"] == 2
    if target == "context":
        return a["contextOK"] is True
    if target == "obligation_v2":
        p, m = a["obligationProfile"], a["mappedMechanism"]
        if p == "empty":
            return True
        return (m != "none" and MECH[m] >= PROF[p] and a["obligationDischarged"] and a["enforcementWitnessVerified"])
    if target == "assurance_v2":
        return a["destAssuranceRequirement"] in GRANTS[a["verifiedTier"]]
    if target == "assurance_a6":
        return a["destAssuranceRequirement"] in GRANTS[a["decisionSigTier"]]

def oracle(target, role, amount, sem, ev, a):
    ok = sem and ev and role in CAPS and amount <= CAPS[role] and family(target, a)
    return "ACCEPT" if ok else "NOT_ACCEPT"

cases = []
counter = [0]
DOMAINS = ["besu", "fabric", "besu", "besu-mainnet", "fabric"]

def add(target, cat, role, amount, sem, ev, a, rationale, expected=None, sourceLimit=None, sourceDomain=None):
    counter[0] += 1
    n = counter[0]
    if expected is None:
        expected = oracle(target, role, amount, sem, ev, a)
    if sourceLimit is None:
        sourceLimit = amount
    cases.append({
        "id": "B%04d" % n, "target": target, "category": cat,
        "decisionId": "dec-%s-%04d" % (target, n),
        "sourceDomain": sourceDomain if sourceDomain is not None else DOMAINS[n % len(DOMAINS)],
        "role": role, "amount": amount, "sourceLimit": sourceLimit,
        "semBound": sem, "evAssuranceOK": ev, "args": dict(a),
        "expected": expected, "rationale": rationale})

# ---- target-specific pass / fail args ----
def pass_args(t):
    if t == "action": return [{"actionLevel": 0}, {"actionLevel": 1}, {"actionLevel": 2}]
    if t == "resource": return [{"resourceID": 2}]
    if t == "context": return [{"contextOK": True}]
    if t == "obligation_v2":
        return [{"obligationProfile": "empty", "mappedMechanism": "none", "obligationDischarged": False, "enforcementWitnessVerified": False},
                {"obligationProfile": "audit", "mappedMechanism": "audit", "obligationDischarged": True, "enforcementWitnessVerified": True},
                {"obligationProfile": "combined", "mappedMechanism": "combined", "obligationDischarged": True, "enforcementWitnessVerified": True},
                {"obligationProfile": "notification", "mappedMechanism": "dual_approval", "obligationDischarged": True, "enforcementWitnessVerified": True}]
    if t == "assurance_v2":
        return [{"claimedTier": "classical", "verifiedTier": "hybrid", "destAssuranceRequirement": "PQDecisionAuth"},
                {"claimedTier": "pq", "verifiedTier": "pq", "destAssuranceRequirement": "PQDecisionAuth"},
                {"claimedTier": "pq", "verifiedTier": "classical", "destAssuranceRequirement": "ClassicalDecisionAuth"}]
    if t == "assurance_a6":
        return [{"decisionSigTier": "hybrid", "destAssuranceRequirement": "PQDecisionAuth"},
                {"decisionSigTier": "classical", "destAssuranceRequirement": "ClassicalDecisionAuth"},
                {"decisionSigTier": "pq", "destAssuranceRequirement": "PQDecisionAuth"},
                {"decisionSigTier": "hybrid", "destAssuranceRequirement": "ClassicalDecisionAuth"}]

def fail_args(t):
    if t == "action": return [{"actionLevel": v} for v in [3, 4, -1, 100, I64MAX, I64MIN, 2**31, -2**31, 2**32 + 1, 2**32 + 2, 256 + 2, 65536 + 1]]
    if t == "resource": return [{"resourceID": v} for v in [0, 1, 3, 99, -1, -2, I64MAX, I64MIN, 2**32 + 2, 2**31 + 2, 258, 65538, 20, 12, 22]]
    if t == "context": return [{"contextOK": False}]
    if t == "obligation_v2":
        return [{"obligationProfile": "audit", "mappedMechanism": "none", "obligationDischarged": True, "enforcementWitnessVerified": True},
                {"obligationProfile": "dual_approval", "mappedMechanism": "audit", "obligationDischarged": True, "enforcementWitnessVerified": True},
                {"obligationProfile": "combined", "mappedMechanism": "combined", "obligationDischarged": True, "enforcementWitnessVerified": False},
                {"obligationProfile": "notification", "mappedMechanism": "notification", "obligationDischarged": False, "enforcementWitnessVerified": True},
                {"obligationProfile": "combined", "mappedMechanism": "dual_approval", "obligationDischarged": True, "enforcementWitnessVerified": True},
                {"obligationProfile": "notification", "mappedMechanism": "none", "obligationDischarged": True, "enforcementWitnessVerified": True}]
    if t == "assurance_v2":
        return [{"claimedTier": "hybrid", "verifiedTier": "classical", "destAssuranceRequirement": "PQDecisionAuth"},
                {"claimedTier": "pq", "verifiedTier": "unverifiable", "destAssuranceRequirement": "PQDecisionAuth"},
                {"claimedTier": "classical", "verifiedTier": "pq", "destAssuranceRequirement": "ClassicalDecisionAuth"},
                {"claimedTier": "hybrid", "verifiedTier": "unverifiable", "destAssuranceRequirement": "ClassicalDecisionAuth"},
                {"claimedTier": "pq", "verifiedTier": "classical", "destAssuranceRequirement": "PQDecisionAuth"}]
    if t == "assurance_a6":
        return [{"decisionSigTier": "classical", "destAssuranceRequirement": "PQDecisionAuth"},
                {"decisionSigTier": "pq", "destAssuranceRequirement": "ClassicalDecisionAuth"}]

TARGETS = ["action", "resource", "context", "obligation_v2", "assurance_v2", "assurance_a6"]

def cyc(lst, i): return lst[i % len(lst)]

for t in TARGETS:
    P, F = pass_args(t), fail_args(t)
    # 1. all-pass boundary acceptance per role
    i = 0
    for role, cap in CAPS.items():
        for amt, lbl in [(cap, "cap"), (cap - 1, "cap-1"), (1, "1")]:
            add(t, "accept_boundary", role, amt, True, True, cyc(P, i), f"all conditions hold; amount {lbl} for {role}"); i += 1
    for a in P:
        add(t, "accept_family_pass", "manager", 1000, True, True, a, "all conditions hold, each passing family value")
    # 2. exactly one of conditions 1-3 fails, family passes
    for j, role in enumerate(CAPS):
        add(t, "cond1_semBound_false", role, 100, False, True, cyc(P, j), "semBound false only")
        add(t, "cond1_evAssurance_false", role, 100, True, False, cyc(P, j + 1), "evAssuranceOK false only")
    add(t, "cond1_both_false", "director", 5000, False, False, P[0], "both attestations false")
    for j, r in enumerate(BAD_ROLES):
        add(t, "cond2_bad_role", r, 50, True, True, cyc(P, j), f"unrecognised role {r!r}")
    for j, (role, cap) in enumerate(CAPS.items()):
        add(t, "cond3_amount_cap_plus1", role, cap + 1, True, True, cyc(P, j), f"amount cap+1 for {role}")
        add(t, "cond3_amount_huge", role, I64MAX, True, True, cyc(P, j + 1), "amount int64 max")
    add(t, "cond3_cross_role", "clerk", 10001, True, True, P[0], "clerk above clerk cap but below manager cap")
    add(t, "cond3_cross_role", "manager", 500001, True, True, P[-1], "manager above cap but below director cap")
    add(t, "cond3_cross_role", "clerk", 500000, True, True, P[0], "clerk with manager cap amount")
    add(t, "cond3_cross_role", "manager", 2000000, True, True, P[0], "manager with director cap amount")
    add(t, "cond3_amount_2pow31", "director", 2**31, True, True, P[0], "amount 2^31 over director cap")
    add(t, "cond3_amount_2pow32_wrap", "clerk", 2**32 + 5, True, True, P[0], "amount that wraps to 5 in uint32")
    # sourceLimit irrelevant
    add(t, "sourceLimit_irrelevant", "manager", 400000, True, True, P[0], "sourceLimit below amount, informational", sourceLimit=1)
    add(t, "sourceLimit_irrelevant", "clerk", 10000, True, True, P[-1], "sourceLimit zero", sourceLimit=0)
    add(t, "sourceLimit_irrelevant", "director", 2000000, True, True, P[0], "sourceLimit negative", sourceLimit=-5)
    add(t, "sourceLimit_irrelevant", "manager", 500001, True, True, P[0], "sourceLimit huge does not raise cap", sourceLimit=I64MAX)
    add(t, "sourceLimit_irrelevant", "clerk", 20000, True, True, P[0], "sourceLimit above amount does not raise cap", sourceLimit=20000)
    add(t, "sourceDomain_irrelevant", "manager", 7, True, True, P[0], "sourceDomain not a condition", sourceDomain="")
    add(t, "sourceDomain_irrelevant", "director", 7, True, True, P[-1], "sourceDomain not a condition", sourceDomain="unknown-chain")
    # 3. family fails, conditions 1-3 hold (both attestations true)
    for j, a in enumerate(F):
        role = cyc(list(CAPS), j)
        add(t, "family_fail_base_ok", role, cyc([1, 100, CAPS[role]], j), True, True, a, "family check fails, attestations both true, role/amount ok")
    # 4. multi-fail combos
    for j, a in enumerate(F):
        add(t, "multi_fail", cyc(BAD_ROLES, j + 3), 10, cyc([True, False], j), True, a, "family and another condition fail")
    add(t, "multi_fail", "clerk", 10001, False, False, F[0], "everything fails")
    # both attestations false together with failing family check; and single attestation with failing family
    for j in range(6):
        a = cyc(F, j)
        role = cyc(list(CAPS), j)
        add(t, "attest_both_false_family_fail", role, 10, False, False, a, "both attestations false and family check fails")
        add(t, "attest_one_false_family_fail", role, CAPS[role], cyc([True, False], j), cyc([False, True], j), a, "one attestation false and family check fails")
    # 5. target-specific extras
    if t == "action":
        for lv in [-1, 0, 1, 2, 3, 4, 5, 10, 255, I64MAX, I64MIN, -2, 2**62, -3]:
            for role in CAPS:
                add(t, "boundary_action_level", role, CAPS[role], True, True, {"actionLevel": lv}, f"action level {lv} with amount at cap")
        for lv in [0, 2]:
            add(t, "boundary_action_level_cond_fail", "clerk", 10001, True, True, {"actionLevel": lv}, "valid level, amount cap+1")
            add(t, "boundary_action_level_cond_fail", "Manager", 1, True, True, {"actionLevel": lv}, "valid level, wrong case role")
            add(t, "boundary_action_level_cond_fail", "director", 1, False, True, {"actionLevel": lv}, "valid level, semBound false")
            add(t, "boundary_action_level_cond_fail", "director", 1, True, False, {"actionLevel": lv}, "valid level, ev false")
        for lv in [2**32 + 1, 2**32, 2**16 + 2, 2**8 + 1, -2**32 + 2, 2**63 - 2, 3 + 2**32]:
            add(t, "action_truncation", "manager", 5, True, True, {"actionLevel": lv}, "level that truncates into 0..2 under narrow int cast")
    if t == "resource":
        for rid in [0, 1, 2, 3, 99, -1, I64MAX, I64MIN, I64MIN + 2, -2, 2**32 + 2, 2**8 + 2, 2**16 + 2, 4, 1000, 2**62]:
            for role in CAPS:
                add(t, "boundary_resource_id", role, CAPS[role] - 1, True, True, {"resourceID": rid}, f"resource id {rid}")
        for rid in [2]:
            for sem, ev in [(False, False), (False, True), (True, False)]:
                add(t, "resource_pinned_attest_fail", "manager", 10, sem, ev, {"resourceID": rid}, "pinned id but attestation fails")
            for r in ["Clerk", "director ", "admin", "gerent"]:
                add(t, "resource_pinned_role_fail", r, 10, True, True, {"resourceID": rid}, "pinned id but role unrecognised")
    if t == "context":
        for role in CAPS:
            for amt in [1, CAPS[role] - 1, CAPS[role], CAPS[role] + 1, CAPS[role] * 2]:
                for c in [True, False]:
                    add(t, "boundary_context", role, amt, True, True, {"contextOK": c}, f"context {c} amount {amt}")
        for sem, ev in itertools.product([True, False], repeat=2):
            for c in [True, False]:
                for role in ["clerk", "director", "Director"]:
                    add(t, "context_attest_matrix", role, 42, sem, ev, {"contextOK": c}, "attestation x context matrix")
        for r in BAD_ROLES:
            add(t, "context_ok_bad_role", r, 1, True, True, {"contextOK": True}, "context ok, bad role")
        for role in CAPS:
            for amt in [I64MAX, 2**32 + 1, 2**31 - 1]:
                add(t, "context_ok_amount_extreme", role, amt, True, True, {"contextOK": True}, "context ok, amount extreme")
    if t == "obligation_v2":
        for p, m, d, w in itertools.product(PROF, MECH, [True, False], [True, False]):
            a = {"obligationProfile": p, "mappedMechanism": m, "obligationDischarged": d, "enforcementWitnessVerified": w}
            add(t, "exhaustive_obligation", "manager", 250000, True, True, a, f"profile {p} mech {m} discharged {d} witness {w}")
        # passing obligation with each base condition failing in turn
        for j, a in enumerate(P):
            add(t, "obligation_pass_cond_fail", "clerk", 10001, True, True, a, "obligation ok, amount cap+1")
            add(t, "obligation_pass_cond_fail", "Director", 10, True, True, a, "obligation ok, role wrong case")
        # empty profile with conditions failing: empty must not bypass base conditions
        for sem, ev in [(False, True), (True, False), (False, False)]:
            a = {"obligationProfile": "empty", "mappedMechanism": "combined", "obligationDischarged": True, "enforcementWitnessVerified": True}
            add(t, "obligation_empty_cond_fail", "director", 1, sem, ev, a, "empty profile does not waive attestations")
        # out of domain strings
        ood = [
            ("", "audit", "NOT_ACCEPT", "empty-string profile is not the 'empty' profile; fail closed"),
            ("Empty", "none", "NOT_ACCEPT", "wrong-case 'Empty' is not 'empty'; fail closed"),
            ("EMPTY", "none", "NOT_ACCEPT", "wrong-case profile; fail closed"),
            ("empty ", "none", "NOT_ACCEPT", "trailing space profile; fail closed"),
            ("none", "none", "NOT_ACCEPT", "'none' is a mechanism name not a profile; fail closed"),
            ("Audit", "audit", "NOT_ACCEPT", "wrong-case profile unknown strength; fail closed"),
            ("unknown", "combined", "NOT_ACCEPT", "unknown profile with strongest mechanism; fail closed"),
            ("audit", "Audit", "NOT_ACCEPT", "wrong-case mechanism; fail closed"),
            ("audit", "", "NOT_ACCEPT", "empty mechanism behaves as dropped; fail closed"),
            ("notification", "logging", "NOT_ACCEPT", "unknown mechanism; fail closed"),
            ("dual_approval", "dual-approval", "NOT_ACCEPT", "hyphenated mechanism unknown; fail closed"),
            ("combined", "COMBINED", "NOT_ACCEPT", "uppercase mechanism unknown; fail closed"),
            ("combined", "super_combined", "NOT_ACCEPT", "unknown stronger-sounding mechanism; fail closed"),
            ("audit", "empty", "NOT_ACCEPT", "'empty' is a profile not a mechanism; fail closed"),
            ("empty", "bogus", "UNSPECIFIED", "profile empty holds literally but mechanism out of domain"),
            ("empty", "", "UNSPECIFIED", "profile empty holds literally but mechanism empty string"),
        ]
        for p, m, exp, rat in ood:
            a = {"obligationProfile": p, "mappedMechanism": m, "obligationDischarged": True, "enforcementWitnessVerified": True}
            add(t, "out_of_domain_obligation_string", "manager", 100, True, True, a, rat, expected=exp)
        # laundering/stripping/weakening at various roles/amounts
        for j, (p, m) in enumerate([("combined", "audit"), ("dual_approval", "notification"), ("audit", "notification"), ("notification", "none"), ("combined", "none")]):
            a = {"obligationProfile": p, "mappedMechanism": m, "obligationDischarged": True, "enforcementWitnessVerified": True}
            add(t, "obligation_weaken_strip", cyc(list(CAPS), j), 1, True, True, a, "weakened or stripped obligation")
    if t == "assurance_v2":
        tiers_c = ["classical", "hybrid", "pq"]
        tiers_v = ["classical", "hybrid", "pq", "unverifiable"]
        for c, v, r in itertools.product(tiers_c, tiers_v, REQS):
            a = {"claimedTier": c, "verifiedTier": v, "destAssuranceRequirement": r}
            add(t, "exhaustive_assurance_v2", "director", 1500000, True, True, a, f"claimed {c} verified {v} requires {r}")
        # exhaustive again under varied base conditions
        bases = [("clerk", 10000, True, True, "clerk at cap"), ("manager", 1, True, True, "manager amount 1"),
                 ("clerk", 10001, True, True, "clerk cap+1"), ("director", 3, False, True, "semBound false"),
                 ("manager", 3, True, False, "ev false"), ("manager ", 3, True, True, "role trailing space")]
        k = 0
        for (c, v, r) in itertools.product(tiers_c, tiers_v, REQS):
            for b in range(3):
                role, amt, sem, ev, lbl = bases[(k + b * 2) % len(bases)]
                a = {"claimedTier": c, "verifiedTier": v, "destAssuranceRequirement": r}
                add(t, "assurance_v2_varied_base", role, amt, sem, ev, a, f"{lbl}; claimed {c} verified {v} req {r}")
            k += 1
        ood = [
            ({"claimedTier": "pq", "verifiedTier": "PQ", "destAssuranceRequirement": "PQDecisionAuth"}, "NOT_ACCEPT", "wrong-case verified tier grants nothing"),
            ({"claimedTier": "hybrid", "verifiedTier": "Hybrid", "destAssuranceRequirement": "ClassicalDecisionAuth"}, "NOT_ACCEPT", "wrong-case verified tier grants nothing"),
            ({"claimedTier": "pq", "verifiedTier": "", "destAssuranceRequirement": "PQDecisionAuth"}, "NOT_ACCEPT", "empty verified tier grants nothing"),
            ({"claimedTier": "pq", "verifiedTier": "pq-hybrid", "destAssuranceRequirement": "PQDecisionAuth"}, "NOT_ACCEPT", "unknown verified tier grants nothing"),
            ({"claimedTier": "hybrid", "verifiedTier": "unknown", "destAssuranceRequirement": "ClassicalDecisionAuth"}, "NOT_ACCEPT", "unknown verified tier grants nothing"),
            ({"claimedTier": "pq", "verifiedTier": "pq", "destAssuranceRequirement": "pqdecisionauth"}, "NOT_ACCEPT", "wrong-case requirement is never granted"),
            ({"claimedTier": "hybrid", "verifiedTier": "hybrid", "destAssuranceRequirement": ""}, "NOT_ACCEPT", "empty requirement is never in granted set (fail closed)"),
            ({"claimedTier": "hybrid", "verifiedTier": "hybrid", "destAssuranceRequirement": "HybridDecisionAuth"}, "NOT_ACCEPT", "unknown requirement not granted"),
            ({"claimedTier": "classical", "verifiedTier": "classical", "destAssuranceRequirement": "NoneRequired"}, "NOT_ACCEPT", "unknown requirement not granted"),
            ({"claimedTier": "PQ", "verifiedTier": "pq", "destAssuranceRequirement": "PQDecisionAuth"}, "UNSPECIFIED", "claimed tier irrelevant but out of domain"),
            ({"claimedTier": "", "verifiedTier": "hybrid", "destAssuranceRequirement": "ClassicalDecisionAuth"}, "UNSPECIFIED", "claimed tier irrelevant but empty"),
            ({"claimedTier": "unverifiable", "verifiedTier": "hybrid", "destAssuranceRequirement": "PQDecisionAuth"}, "UNSPECIFIED", "claimed tier 'unverifiable' outside claimed set"),
        ]
        for a, exp, rat in ood:
            add(t, "out_of_domain_assurance_v2", "manager", 100, True, True, a, rat, expected=exp)
    if t == "assurance_a6":
        for tier, r in itertools.product(["classical", "hybrid", "pq"], REQS):
            a = {"decisionSigTier": tier, "destAssuranceRequirement": r}
            add(t, "exhaustive_assurance_a6", "manager", 500000, True, True, a, f"tier {tier} requires {r}")
        bases = [("clerk", 10000, True, True, "clerk at cap"), ("clerk", 10001, True, True, "clerk cap+1"),
                 ("director", 2000000, True, True, "director at cap"), ("director", 2000001, True, True, "director cap+1"),
                 ("manager", 499999, True, True, "manager cap-1"), ("manager", 500001, True, True, "manager cap+1"),
                 ("manager", 1, False, True, "semBound false"), ("manager", 1, True, False, "ev false"),
                 ("director", 1, False, False, "both false"), ("Manager", 1, True, True, "role wrong case"),
                 ("clerk ", 1, True, True, "role trailing space"), ("admin", 1, True, True, "role admin"),
                 ("", 1, True, True, "role empty"), ("director", I64MAX, True, True, "amount int64 max"),
                 ("clerk", 2**32 + 100, True, True, "amount wraps in uint32"), ("manager", 250000, True, True, "manager mid"),
                 ("director", 1, True, True, "director amount 1"), ("clerk", 9999, True, True, "clerk cap-1"),
                 ("DIRECTOR", 1, True, True, "role upper"), ("director\n", 1, True, True, "role newline"),
                 ("mаnager", 1, True, True, "cyrillic homoglyph role"), ("clerk", 10000, True, False, "ev false at cap"),
                 ("director", 2000000, False, True, "semBound false at cap"), ("manager", 2000000, True, True, "manager at director cap")]
        for (role, amt, sem, ev, lbl) in bases:
            for tier, r in itertools.product(["classical", "hybrid", "pq"], REQS):
                a = {"decisionSigTier": tier, "destAssuranceRequirement": r}
                add(t, "assurance_a6_varied_base", role, amt, sem, ev, a, f"{lbl}; tier {tier} req {r}")
        ood = [
            ({"decisionSigTier": "unverifiable", "destAssuranceRequirement": "ClassicalDecisionAuth"}, "NOT_ACCEPT", "unverifiable is outside a6 tier set; grants nothing"),
            ({"decisionSigTier": "unverifiable", "destAssuranceRequirement": "PQDecisionAuth"}, "NOT_ACCEPT", "unverifiable outside a6 tier set; grants nothing"),
            ({"decisionSigTier": "Hybrid", "destAssuranceRequirement": "PQDecisionAuth"}, "NOT_ACCEPT", "wrong-case tier grants nothing"),
            ({"decisionSigTier": "PQ", "destAssuranceRequirement": "PQDecisionAuth"}, "NOT_ACCEPT", "wrong-case tier grants nothing"),
            ({"decisionSigTier": "", "destAssuranceRequirement": "ClassicalDecisionAuth"}, "NOT_ACCEPT", "empty tier grants nothing"),
            ({"decisionSigTier": "hybrid", "destAssuranceRequirement": ""}, "NOT_ACCEPT", "empty requirement never granted"),
            ({"decisionSigTier": "hybrid", "destAssuranceRequirement": "HybridDecisionAuth"}, "NOT_ACCEPT", "unknown requirement never granted"),
            ({"decisionSigTier": "pq", "destAssuranceRequirement": "pqDecisionAuth"}, "NOT_ACCEPT", "wrong-case requirement never granted"),
            ({"decisionSigTier": "classical ", "destAssuranceRequirement": "ClassicalDecisionAuth"}, "NOT_ACCEPT", "trailing-space tier grants nothing"),
            ({"decisionSigTier": "ml-dsa", "destAssuranceRequirement": "PQDecisionAuth"}, "NOT_ACCEPT", "unknown tier grants nothing"),
        ]
        for a, exp, rat in ood:
            add(t, "out_of_domain_assurance_a6", "director", 100, True, True, a, rat, expected=exp)
    # out-of-domain amounts (common) for every target
    for amt in [0, -1, I64MIN]:
        add(t, "out_of_domain_amount", "manager", amt, True, True, P[0],
            "amount non-positive is outside the domain; literal reading amount<=cap holds", expected="UNSPECIFIED")

# ---- checks ----
ids = [c["id"] for c in cases]; dids = [c["decisionId"] for c in cases]
assert len(set(ids)) == len(ids) and len(set(dids)) == len(dids)
EXPECTED_KEYS = {"action": {"actionLevel"}, "resource": {"resourceID"}, "context": {"contextOK"},
                 "obligation_v2": {"obligationProfile", "mappedMechanism", "obligationDischarged", "enforcementWitnessVerified"},
                 "assurance_v2": {"claimedTier", "verifiedTier", "destAssuranceRequirement"},
                 "assurance_a6": {"decisionSigTier", "destAssuranceRequirement"}}
for c in cases:
    assert set(c["args"]) == EXPECTED_KEYS[c["target"]], c
    assert c["role"] not in ("gerente", "gerente_senior")
out = os.path.join(os.path.dirname(os.path.abspath(__file__)), "output", "cases.json")
with open(out, "w") as f:
    json.dump(cases, f, indent=1, ensure_ascii=False)
