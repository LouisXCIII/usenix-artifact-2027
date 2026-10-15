#!/usr/bin/env python3
"""S3b: independent oracle (Cedar) and analysis for the six other decision families (protocol amendment A3).
  families.py oracle  cases.json oracle.json
  families.py analyze cases.json sut.json oracle.json summary.json
The oracle is written from the specification in adversary_SPEC2.md (which comes from the paper's formal model), not from the Go code.
Primary property: SAFETY OF ACCEPTANCE. accepted := class in {EXACT, SAFE}.
unsafe accept = SUT accepts, oracle does not. over-conservative = oracle accepts, SUT does not. Out-of-domain cases are reported separately."""
import json, sys
from collections import Counter, defaultdict

CAPS = {"manager": 500000, "director": 2000000, "clerk": 10000}
ACCEPT = {"EXACT", "SAFE"}
PROFILE = {"empty": 0, "notification": 1, "audit": 2, "dual_approval": 3, "combined": 4}
MECH = {"none": 0, "notification": 1, "audit": 2, "dual_approval": 3, "combined": 4}
TIER = {"classical": (True, False), "hybrid": (True, True), "pq": (False, True), "unverifiable": (False, False)}
REQ = {"ClassicalDecisionAuth", "PQDecisionAuth"}


def _policies():
    base = "context.amount <= {cap}"
    fam = {
        "action": "context.actionLevel >= 0 && context.actionLevel <= 2",
        "resource": "context.resourceId == 2",
        "context": "context.contextOK",
        "obligation_v2": "(context.profile.required == 0 || (context.mechanism.strength > 0 && context.mechanism.strength >= context.profile.required && context.discharged && context.witness))",
        "assurance": "((context.requirement == \"ClassicalDecisionAuth\" && context.verified.claimsClassical) || (context.requirement == \"PQDecisionAuth\" && context.verified.claimsPQ))",
    }
    out = {}
    for name, cond in fam.items():
        out[name] = "\n".join(
            f'permit(principal in DestRole::"{r}", action == Action::"approve", resource) when {{ {base.format(cap=c)} && {cond} }};'
            for r, c in CAPS.items())
    return out


def _entities():
    e = [{"uid": {"type": "DestRole", "id": r}, "attrs": {}, "parents": []} for r in CAPS]
    e += [{"uid": {"type": "Profile", "id": k}, "attrs": {"required": v}, "parents": []} for k, v in PROFILE.items()]
    e += [{"uid": {"type": "Mech", "id": k}, "attrs": {"strength": v}, "parents": []} for k, v in MECH.items()]
    e += [{"uid": {"type": "Tier", "id": k}, "attrs": {"claimsClassical": a, "claimsPQ": b}, "parents": []} for k, (a, b) in TIER.items()]
    return e


def _ref(t, i):
    return {"__entity": {"type": t, "id": i}}


def oracle(cases_p, out_p):
    import cedarpy
    pol, ents = _policies(), _entities()
    out = []
    for c in cases_p:
        t, a = c["target"], c.get("args", {})
        amt = c["amount"]
        in_dom = isinstance(amt, int) and amt > 0 and isinstance(c["role"], str) and c["role"] != "" and not (-(2**63) > amt or amt > 2**63 - 1)
        ctx = {"amount": amt if isinstance(amt, int) and -(2**63) <= amt <= 2**63 - 1 else 0}
        pkey = t
        try:
            if t == "action":
                lv = a["actionLevel"]
                if not (-(2**63) <= lv <= 2**63 - 1):
                    raise KeyError
                ctx["actionLevel"] = lv
                in_dom = in_dom and lv >= 0
            elif t == "resource":
                ctx["resourceId"] = a["resourceID"]
            elif t == "context":
                ctx["contextOK"] = bool(a["contextOK"])
            elif t == "obligation_v2":
                if a["obligationProfile"] not in PROFILE or a["mappedMechanism"] not in MECH:
                    raise KeyError
                ctx.update(profile=_ref("Profile", a["obligationProfile"]), mechanism=_ref("Mech", a["mappedMechanism"]),
                           discharged=bool(a["obligationDischarged"]), witness=bool(a["enforcementWitnessVerified"]))
            elif t == "assurance_v2":
                if a["verifiedTier"] not in TIER or a["claimedTier"] not in ("classical", "hybrid", "pq") or a["destAssuranceRequirement"] not in REQ:
                    raise KeyError
                pkey = "assurance"
                ctx.update(verified=_ref("Tier", a["verifiedTier"]), requirement=a["destAssuranceRequirement"])
            elif t == "assurance_a6":
                if a["decisionSigTier"] not in ("classical", "hybrid", "pq") or a["destAssuranceRequirement"] not in REQ:
                    raise KeyError
                pkey = "assurance"
                ctx.update(verified=_ref("Tier", a["decisionSigTier"]), requirement=a["destAssuranceRequirement"])
        except (KeyError, TypeError):
            in_dom = False
            out.append({"id": c["id"], "oracle_accept": None, "in_domain": False, "note": "argument outside the specified domain"})
            continue
        role = c["role"]
        principal = {"type": "DestRole", "id": role} if role in CAPS else {"type": "Unknown", "id": role}
        extra = [] if role in CAPS else [{"uid": principal, "attrs": {}, "parents": []}]
        req = {"principal": principal, "action": {"type": "Action", "id": "approve"}, "resource": {"type": "Res", "id": "r"}, "context": ctx}
        allowed = cedarpy.is_authorized(req, pol[pkey], ents + extra).decision == cedarpy.Decision.Allow
        accept = bool(c["semBound"] and c["evAssuranceOK"] and allowed)
        out.append({"id": c["id"], "oracle_accept": accept, "in_domain": bool(in_dom)})
    json.dump(out, open(out_p, "w"), indent=1)
    print(f"oracle: {len(out)} cases -> {out_p}")


def cp_upper(n):
    return None if n == 0 else 1 - 0.05 ** (1 / n)


def analyze(cases_p, sut_p, orc_p, dst):
    cases = {c["id"]: c for c in json.load(open(cases_p))}
    sut = {o["id"]: o for o in json.load(open(sut_p))}
    orc = {o["id"]: o for o in json.load(open(orc_p))}
    res = {}
    for target in sorted({c["target"] for c in cases.values()}):
        prim, ood, skipped = Counter(), Counter(), 0
        per, examples, adv = defaultdict(Counter), [], Counter()
        for i, c in cases.items():
            if c["target"] != target:
                continue
            s, o = sut[i], orc[i]
            if s.get("skipped") or o["oracle_accept"] is None:
                skipped += 1
                continue
            sut_acc = (s["class"] in ACCEPT) and not s["err"]
            if sut_acc == o["oracle_accept"]:
                v = "match"
            else:
                v = "unsafe_accept" if sut_acc else "over_conservative"
            exp = c.get("expected")
            if exp in ("ACCEPT", "NOT_ACCEPT"):
                adv["agree" if (exp == "ACCEPT") == o["oracle_accept"] else "disagree"] += 1
            if not o["in_domain"]:
                ood[v] += 1
                continue
            prim[v] += 1
            per[c["category"]][v] += 1
            if v != "match" and len(examples) < 15:
                examples.append((i, c["category"], c["role"], c["amount"], c["semBound"], c["evAssuranceOK"], c["args"], s["class"], o["oracle_accept"]))
        n = sum(prim.values())
        res[target] = {"in_domain_cases": n, "overall": dict(prim), "unsafe_accepts": prim["unsafe_accept"],
                       "cp95_upper_bound_if_zero": cp_upper(n) if prim["unsafe_accept"] == 0 else None,
                       "out_of_domain": dict(ood), "skipped": skipped, "adversary_expected_vs_oracle": dict(adv),
                       "per_category": {k: dict(v) for k, v in sorted(per.items())}, "examples": examples}
    json.dump(res, open(dst, "w"), indent=1, default=str)
    tot_n = sum(r["in_domain_cases"] for r in res.values())
    tot_u = sum(r["unsafe_accepts"] for r in res.values())
    for t, r in res.items():
        b = f"{100*r['cp95_upper_bound_if_zero']:.3f}%" if r["cp95_upper_bound_if_zero"] else "n/a"
        print(f"{t.ljust(14)} in-domain {r['in_domain_cases']:4d}  {r['overall']}  unsafe {r['unsafe_accepts']}  bound {b} | ood {r['out_of_domain']} skipped {r['skipped']} | adv-vs-oracle {r['adversary_expected_vs_oracle']}")
    print(f"TOTAL in-domain {tot_n}, unsafe accepts {tot_u}")


if __name__ == "__main__":
    if len(sys.argv) >= 4 and sys.argv[1] == "oracle":
        oracle(json.load(open(sys.argv[2])), sys.argv[3])
    elif len(sys.argv) == 6 and sys.argv[1] == "analyze":
        analyze(*sys.argv[2:])
    else:
        sys.exit(__doc__)
