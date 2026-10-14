import json, os
from collections import Counter

CAPS = {"manager": 500000, "director": 2000000, "clerk": 10000}
REG = {"gerente": "manager", "gerente_senior": "director"}
I64MAX = 2**63 - 1
I64MIN = -2**63

seen = set()
cases = []


def oracle(did, role, amount, sem, ev, sec):
    if did in seen:
        return "REPLAY_ERROR"
    seen.add(did)
    if not (sem and ev):
        return "REJECT"
    if role in CAPS:
        mapped, ident = role, True
    elif role in REG:
        mapped, ident = REG[role], False
    else:
        return "REEVALUATE"
    cap = CAPS[mapped]
    if amount > cap:
        return "REJECT"
    # "strictly greater than half of the cap": exact rational comparison
    if 2 * amount > cap and not sec:
        return "CONDITIONAL"
    return "EXACT" if ident else "SAFE"


n = [0]


def add(cat, role, amount, rationale, sem=True, ev=True, sec=False, sl=None,
        dom="src-bank-a", did=None, expected=None):
    n[0] += 1
    cid = "A%03d" % n[0]
    if did is None:
        did = "dec-%04d" % n[0]
    if sl is None:
        sl = amount
    exp = oracle(did, role, amount, sem, ev, sec)
    if expected is not None:
        # override (UNSPECIFIED) but oracle still records the id
        exp = expected
    cases.append({"id": cid, "category": cat, "decisionId": did, "sourceDomain": dom,
                  "role": role, "amount": amount, "sourceLimit": sl, "semBound": sem,
                  "evAssuranceOK": ev, "secondApprovalObtained": sec,
                  "expected": exp, "rationale": rationale})
    return did


# 1. cap boundaries for every role (identity + registry), with and without approval
for role, mapped in [("manager", "manager"), ("director", "director"), ("clerk", "clerk"),
                     ("gerente", "manager"), ("gerente_senior", "director")]:
    cap = CAPS[mapped]
    for d in (-1, 0, 1):
        for sec in (False, True):
            add("cap_boundary", role, cap + d,
                f"{role}->{mapped} amount cap{d:+d}; approval={sec}; over cap REJECT else obligation", sec=sec)
    add("cap_boundary", role, cap + 2, f"{role} cap+2 over cap", sec=True)
    add("cap_boundary", role, 2 * cap, f"{role} double cap (catches cap check vs 2*cap)", sec=True)

# 2. half-cap boundary
for role, mapped in [("manager", "manager"), ("director", "director"), ("clerk", "clerk"),
                     ("gerente", "manager"), ("gerente_senior", "director")]:
    half = CAPS[mapped] // 2
    for d in (-1, 0, 1, 2):
        for sec in (False, True):
            add("half_cap_boundary", role, half + d,
                f"{role} half{d:+d}; strictly greater than half needs approval; approval={sec}", sec=sec)

# 3. cross-role caps: amounts valid for one role but not another
cross = [("clerk", 10001), ("clerk", 250000), ("clerk", 500000), ("manager", 500001), ("manager", 1000000),
         ("manager", 2000000), ("gerente", 2000000), ("gerente", 1000001), ("gerente", 10001),
         ("gerente_senior", 500001), ("gerente_senior", 1500000), ("gerente_senior", 250001),
         ("director", 500001), ("director", 10001), ("clerk", 5001), ("gerente", 5001),
         ("gerente_senior", 2000001), ("director", 1999999)]
for role, amt in cross:
    for sec in (False, True):
        add("cross_role_cap", role, amt,
            f"{role} amount {amt}: cap of mapped role decides (not another role's cap); approval={sec}", sec=sec)

# 4. attestation precedence
combos = [(False, False), (False, True), (True, False)]
scen = [("manager", 100, "valid small"), ("manager", 400000, "needs approval"),
        ("gerente", 100, "registry valid"), ("clerk", 10001, "over cap"),
        ("unknown_role", 100, "unrecognised"), ("Manager", 100, "case variant unrecognised"),
        ("director", I64MAX, "int64 max"), ("gerente_senior", 1500000, "registry needs approval"),
        ("", 100, "empty role")]
for role, amt, desc in scen:
    for sem, ev in combos:
        for sec in ((False, True) if amt == 400000 or amt == 1500000 else (False,)):
            cat = "out_of_domain_empty_role" if role == "" else "attestation_precedence"
            add(cat, role, amt, f"{desc}; semBound={sem} ev={ev}: attestation checked first -> REJECT",
                sem=sem, ev=ev, sec=sec)
add("attestation_precedence", "manager", 100, "both attestations true, approval true, small -> EXACT", sec=True)
add("attestation_precedence", "gerente", 100, "both true, registry -> SAFE")
add("attestation_precedence", "clerk", 1, "both true -> EXACT, minimal amount")

# 5. role-string tricks (all unrecognised -> REEVALUATE), with small amount
tricks = [
    "Manager", "MANAGER", "manageR", "Director", "DIRECTOR", "Clerk", "CLERK", "Gerente", "GERENTE",
    "Gerente_Senior", "GERENTE_SENIOR", "gerente_Senior",
    " manager", "manager ", " manager ", "\tmanager", "manager\t", "manager\n", "\nmanager", "manager\r\n",
    "gerente ", " gerente", "gerente\t", "gerente_senior ", " gerente_senior", "clerk ", " clerk", "director\n",
    "manager2", "manager_", "managers", "xmanager", "manage", "manag", "m", "mgr", "directors", "direct",
    "clerks", "cler", "clerk1", "_clerk",
    "gerente_senior_x", "gerente_seniorx", "gerente_senio", "gerente_", "gerent", "gerentes", "xgerente",
    "gerente-senior", "gerente senior", "gerenteSenior", "gerente__senior", "senior_gerente", "gerente_senior_",
    "_gerente", "gerente.senior", "gerente/senior",
    "mаnager",          # Cyrillic a
    "манагер",  # full cyrillic menedzher-ish
    "managеr",          # Cyrillic e
    "dirеctor",         # Cyrillic e
    "сlerk",            # Cyrillic es
    "gеrente",          # Cyrillic e
    "gerеnte_senior",   # Cyrillic e
    "manager​",         # zero width space
    "​manager",
    "man‍ager",         # ZWJ
    "﻿manager",         # BOM
    "clerk‌",           # ZWNJ
    "gerente⁠",         # word joiner
    "manager ",         # NBSP
    " gerente",
    "gerente_senior　",  # ideographic space
    "ｍａｎａｇｅｒ",  # fullwidth manager
    "ｃｌｅｒｋ",  # fullwidth clerk
    "dırector",         # dotless i
    "DİRECTOR".lower(),      # i with combining dot after lower()
    "ᴍanager",          # small capital M
    "manageŕ",         # combining acute
    "gerénte",
    "geʀente",          # small capital R
    "gerente＿senior",   # fullwidth low line
    "gerente‐senior",   # hyphen
    "manager\u0000",         # NUL suffix (C-string truncation)
    "clerk\u0000x",
    "gerente\u0000_senior",
    "‮manager",         # RTL override
    "man­ager",         # soft hyphen
    "manaɡer",          # latin script g
    "Ⅳ",               # roman numeral (irrelevant)
    "manager😀".encode("utf-16", "surrogatepass").decode("utf-16"),
    "manager,director", "manager|clerk", "[\"manager\"]", "manager;", "role=manager", "'manager'", "\"manager\"",
    "manager\\", "gerente\\_senior", "gerente%5Fsenior", "gerente_senior%00", "Z2VyZW50ZQ==",
    "admin", "root", "superuser", "*", ".*", "manager.*", "null", "undefined", "None", "true", "0",
    "__proto__", "constructor", "toString", "hasOwnProperty", "valueOf", "prototype",
    "director ", "clerk\u000b", "manager\u000c", "manager\u0085", "manager ",
]
for i, r in enumerate(tricks):
    amt = [100, 5000, 250000, 400000, 10001, 1000000][i % 6]
    sec = (i % 3 == 0)
    add("role_string_trick", r, amt,
        f"role {json.dumps(r)} is not an exact destination or registry name -> REEVALUATE (no trimming/normalisation)", sec=sec)

# over-cap amounts with unrecognised roles: REEVALUATE precedes cap check
for r, amt in [("Manager", 500001), ("gerente ", 500001), ("gerente_senior_x", 2000001), ("clerk​", 10001),
               ("admin", I64MAX), ("CLERK", 10001), ("mаnager", 99999999)]:
    add("precedence_unrecognised_over_cap", r, amt,
        "unrecognised role checked before cap -> REEVALUATE even though amount exceeds every/any cap")

# 6. registry-name look-alikes around mapped caps
for r in ["gerente", "gerente_senior"]:
    for amt in [1, 250000, 250001, 500000, 500001, 1000000, 1000001, 2000000, 2000001]:
        add("registry_mapped_cap", r, amt,
            f"{r}->{REG[r]}; amount {amt}; no approval", sec=False)

# 7. int64 limits (in domain: positive)
for role in ["manager", "director", "clerk", "gerente", "gerente_senior"]:
    for amt in [I64MAX, I64MAX - 1, 2**62, 2**62 + 1, 2**32, 2**31, 2**31 - 1, 2**53 + 1, 4611686018427637905]:
        add("int64_limits", role, amt, f"huge positive amount {amt} > cap -> REJECT (watch 2*amount overflow)", sec=False)
add("int64_limits", "manager", 250000, "sourceLimit int64 max, amount at half, no approval", sl=I64MAX)
add("int64_limits", "clerk", 5001, "sourceLimit int64 min informational", sl=I64MIN)
add("int64_limits", "director", 2000000, "sourceLimit int64 max, at cap with approval", sec=True, sl=I64MAX)

# 8. sourceLimit informational
for role, amt, sec in [("manager", 100000, False), ("manager", 400000, False), ("manager", 400000, True),
                        ("gerente", 200000, False), ("clerk", 10000, True), ("director", 2000001, True),
                        ("gerente_senior", 1500000, True), ("clerk", 10001, True), ("Manager", 100, False)]:
    for sl, d in [(0, "zero"), (amt - 1, "smaller"), (amt, "equal"), (amt + 1, "larger"), (-1, "negative"),
                  (1, "one"), (10**12, "much larger"), (-10**12, "very negative")]:
        add("source_limit_informational", role, amt,
            f"sourceLimit {d} than amount ({sl}); informational only", sec=sec, sl=sl)

# 9. approval supplied when not needed
for role in ["manager", "director", "clerk", "gerente", "gerente_senior"]:
    cap = CAPS[REG.get(role, role)]
    for amt in [1, cap // 4, cap // 2]:
        add("unneeded_approval", role, amt, "approval supplied though not required -> still EXACT/SAFE", sec=True)

# 10. domain / decisionId variation irrelevant
for dom in ["", "dst-bank", "src-bank-a\u0000", "‮domain", "x" * 1000, "manager", "gerente"]:
    add("source_domain_irrelevant", "gerente", 250001, "sourceDomain never affects class", dom=dom, sec=True)

# 11. replay
d1 = add("replay_setup", "manager", 100, "first submission, EXACT", did="rp-001")
add("replay", "manager", 100, "same decisionId, same fields -> REPLAY_ERROR", did="rp-001")
add("replay", "director", 1500000, "same decisionId, different role/amount -> REPLAY_ERROR", did="rp-001", sec=True)
add("replay", "manager", 100, "replay with attestations false: replay error, not REJECT", did="rp-001", sem=False, ev=False)
add("replay", "unknown", 100, "replay with unknown role: replay error, not REEVALUATE", did="rp-001")
add("replay_setup", "clerk", 20000, "REJECT (over cap) submission still counts as submitted", did="rp-002")
add("replay", "clerk", 100, "replay of a previously REJECTed id with now-valid fields -> REPLAY_ERROR", did="rp-002")
add("replay_setup", "manager", 100, "REJECT on attestation; still submitted", did="rp-003", sem=False)
add("replay", "manager", 100, "replay of attestation-REJECTed id, now valid -> REPLAY_ERROR", did="rp-003")
add("replay_setup", "Gerente", 100, "REEVALUATE submission", did="rp-004")
add("replay", "gerente", 100, "replay of REEVALUATE id with fixed role -> REPLAY_ERROR", did="rp-004")
add("replay_setup", "manager", 400000, "CONDITIONAL submission", did="rp-005")
add("replay", "manager", 400000, "replay adding second approval -> REPLAY_ERROR (cannot upgrade)", did="rp-005", sec=True)
add("replay", "manager", 400000, "third submission of same id still REPLAY_ERROR", did="rp-005", sec=True)
add("replay_setup", "gerente_senior", 1000001, "CONDITIONAL registry", did="rp-006")
add("replay", "gerente_senior", 1000001, "replay with different sourceDomain -> REPLAY_ERROR", did="rp-006", dom="other-domain", sec=True)
add("replay", "gerente_senior", 1000001, "replay with different sourceLimit -> REPLAY_ERROR", did="rp-006", sl=-5)
# look-alike ids are NOT replays
add("decisionid_lookalike", "manager", 100, "decisionId differs by case -> new decision", did="RP-001")
add("decisionid_lookalike", "manager", 100, "trailing space -> distinct id", did="rp-001 ")
add("decisionid_lookalike", "manager", 100, "leading space -> distinct id", did=" rp-001")
add("decisionid_lookalike", "manager", 100, "zero-width suffix -> distinct id", did="rp-001​")
add("decisionid_lookalike", "manager", 100, "Cyrillic r -> distinct id", did="рp-001")
add("decisionid_lookalike", "manager", 100, "prefix of existing id -> distinct", did="rp-00")
add("decisionid_lookalike", "manager", 100, "NUL suffix -> distinct id (C-string truncation bug)", did="rp-001\u0000")
add("decisionid_lookalike", "manager", 100, "fullwidth digit -> distinct id", did="rp-00１")
add("replay", "manager", 100, "replay of the case-variant id itself -> REPLAY_ERROR", did="RP-001")
add("replay", "clerk", 1, "replay of the trailing-space id -> REPLAY_ERROR", did="rp-001 ")
# replay of earlier general cases
add("replay", "manager", 1, "replay of case A001 id with different fields", did=cases[0]["decisionId"])
add("replay", "clerk", 1, "replay of an early attestation-reject id", did="dec-%04d" % 200)
add("replay", "gerente", 1, "replay of a role-trick REEVALUATE id", did=cases[260]["decisionId"])
add("replay", "director", 1, "replay of an int64 case id", did=[c for c in cases if c["category"] == "int64_limits"][0]["decisionId"])
add("replay_setup", "clerk", 1, "numeric-looking id", did="12345")
add("decisionid_lookalike", "clerk", 1, "numeric id with leading zero is a different string", did="012345")
add("replay", "clerk", 1, "exact numeric string repeat -> REPLAY_ERROR", did="12345")

# 12. out-of-domain
for role in ["manager", "gerente", "clerk", "director", "gerente_senior"]:
    add("out_of_domain_zero", role, 0, "amount 0 not > half cap, not > cap -> EXACT/SAFE by literal reading")
    add("out_of_domain_negative", role, -1, "negative amount passes both checks literally -> EXACT/SAFE")
    add("out_of_domain_negative", role, I64MIN, "int64 min: literal reading EXACT/SAFE (watch negation/abs overflow)")
    add("out_of_domain_negative", role, -(2**62) - 1,
        "2*amount overflows to large positive in int64; literal reading EXACT/SAFE not CONDITIONAL")
    add("out_of_domain_negative", role, I64MIN + 1, "near int64 min -> EXACT/SAFE")
add("out_of_domain_negative", "manager", -500001, "negative magnitude over cap: still not > cap -> EXACT")
add("out_of_domain_negative", "manager", -1, "negative amount with attestation false -> REJECT", sem=False)
add("out_of_domain_negative", "Manager", -1, "negative amount unrecognised -> REEVALUATE")
add("out_of_domain_zero", "manager", 0, "zero with attestation false -> REJECT", ev=False)
add("out_of_domain_empty_role", "", 100, "empty role is not a known name -> REEVALUATE")
add("out_of_domain_empty_role", "", 10**9, "empty role over any cap: unrecognised first -> REEVALUATE")
add("out_of_domain_empty_role", "", 0, "empty role zero amount -> REEVALUATE")
add("out_of_domain_empty_role", " ", 100, "single space role -> REEVALUATE")
add("out_of_domain_empty_role", "\u0000", 100, "NUL-only role -> REEVALUATE")
add("out_of_domain_empty_decisionid", "manager", 100, "empty decisionId, first use -> EXACT", did="")
add("out_of_domain_empty_decisionid", "manager", 100, "empty decisionId reused -> REPLAY_ERROR", did="")
for role, amt in [("manager", 2**63), ("clerk", 2**64), ("manager", -(2**63) - 1), ("gerente", 2**64 + 100),
                  ("director", 2**63 + 1000000)]:
    add("out_of_domain_overflow", role, amt, "amount outside int64: implementation should refuse input or REJECT",
        expected="UNSPECIFIED")
add("out_of_domain_overflow", "manager", 100, "sourceLimit outside int64 but informational; input may be refused",
    sl=2**64, expected="UNSPECIFIED")
add("out_of_domain_overflow", "clerk", 2**64 + 5000,
    "wraps to 5000 mod 2^64 if truncated -> would be EXACT; must not", expected="UNSPECIFIED")

out = os.path.join(os.path.dirname(os.path.abspath(__file__)), "output", "cases.json")
with open(out, "w", encoding="utf-8") as f:
    json.dump(cases, f, ensure_ascii=True, indent=1)

data = json.load(open(out, encoding="utf-8"))
print("cases", len(data), "distinct ids", len({c["id"] for c in data}))
print(Counter(c["category"] for c in data))
print(Counter(c["expected"] for c in data))
