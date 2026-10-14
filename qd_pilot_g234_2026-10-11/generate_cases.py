#!/usr/bin/env python3
"""Mechanical case generators for the Group 2/3/4 pilot (obligation, multi-hop,
cross-vocabulary mapping). Ground truth is computed independently here, directly from
spec_v1.md's prose for each group -- NOT by calling into any naive/hardened policy file.
No AI-adversarial content; cases are boundary-value (G2, G4) or exhaustive (G3, since its
domain -- booleans, length 1-5 -- is small enough to enumerate completely)."""
import itertools
import json
import os

OUT = os.path.dirname(os.path.abspath(__file__))

# ---------------------------------------------------------------------------
# Group 2: obligation / evidence-strength
# ---------------------------------------------------------------------------
CAP = {"clerk": 10000, "manager": 500000, "director": 2000000,
       "gerente": 500000, "gerente_senior": 2000000}
HALF = {"clerk": 5000, "manager": 250000, "director": 1000000,
        "gerente": 250000, "gerente_senior": 1000000}
STRENGTH = {"none": 0, "notification": 1, "audit": 2, "dual_approval": 3, "combined": 4}
EVIDENCE_TYPES = ["none", "notification", "audit", "dual_approval", "combined", "unknown_evidence_xyz"]


def real_grant_g2(role, amount, evidence_type):
    if amount <= 0:
        return False
    if role not in CAP:
        return False
    if amount > CAP[role]:
        return False
    if amount <= HALF[role]:
        return True
    s = STRENGTH.get(evidence_type, 0)
    return s >= 3


def gen_g2():
    cases = []
    cid = 0
    for role in CAP:
        cap, half = CAP[role], HALF[role]
        amounts = sorted(set([1, half - 1, half, half + 1, cap - 1, cap, cap + 1, 2 * cap, 0, -1]))
        for amount in amounts:
            for ev in EVIDENCE_TYPES:
                cases.append({"case_id": cid, "role": role, "amount": amount,
                              "evidence_type": ev, "real_grant": real_grant_g2(role, amount, ev)})
                cid += 1
    return cases


# ---------------------------------------------------------------------------
# Group 3: multi-hop AND-over-all
# ---------------------------------------------------------------------------
def real_grant_g3(hops):
    if len(hops) == 0:
        return False
    return all(hops)


def gen_g3():
    cases = []
    cid = 0
    cases.append({"case_id": cid, "hops": [], "real_grant": real_grant_g3([])})
    cid += 1
    for n in range(1, 6):
        for combo in itertools.product([True, False], repeat=n):
            hops = list(combo)
            cases.append({"case_id": cid, "hops": hops, "real_grant": real_grant_g3(hops)})
            cid += 1
    return cases


# ---------------------------------------------------------------------------
# Group 4: cross-vocabulary role mapping
# ---------------------------------------------------------------------------
VOCAB_CAP = {"gerente": 500000, "gerente_senior": 2000000, "funcionario": 10000,
             "diretor": 2000000, "supervisor": 500000}
UNMAPPED_ROLES = ["estagiario", "manager", ""]  # "manager" = destination-side name trap


def real_grant_g4(source_role, amount):
    if amount <= 0:
        return False
    if source_role not in VOCAB_CAP:
        return False
    return amount <= VOCAB_CAP[source_role]


def gen_g4():
    cases = []
    cid = 0
    for role in list(VOCAB_CAP.keys()) + UNMAPPED_ROLES:
        if role in VOCAB_CAP:
            cap = VOCAB_CAP[role]
            amounts = sorted(set([1, cap - 1, cap, cap + 1, 2 * cap, 0, -1]))
        else:
            amounts = [1, 10000, 500000, 2000000, 4000000, 0, -1]
        for amount in amounts:
            cases.append({"case_id": cid, "source_role": role, "amount": amount,
                          "real_grant": real_grant_g4(role, amount)})
            cid += 1
    return cases


if __name__ == "__main__":
    g2 = gen_g2()
    g3 = gen_g3()
    g4 = gen_g4()
    json.dump(g2, open(os.path.join(OUT, "cases2.json"), "w"), indent=2)
    json.dump(g3, open(os.path.join(OUT, "cases3.json"), "w"), indent=2)
    json.dump(g4, open(os.path.join(OUT, "cases4.json"), "w"), indent=2)
    print(f"G2 (obligation): {len(g2)} cases, {sum(c['real_grant'] for c in g2)} GRANT")
    print(f"G3 (multi-hop):  {len(g3)} cases, {sum(c['real_grant'] for c in g3)} GRANT")
    print(f"G4 (vocab):      {len(g4)} cases, {sum(c['real_grant'] for c in g4)} GRANT")
