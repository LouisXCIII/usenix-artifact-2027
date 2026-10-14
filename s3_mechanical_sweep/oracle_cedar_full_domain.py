#!/usr/bin/env python3
"""Extends the S3 Cedar oracle to explicitly model amount<=0 as a required denial, so the
200 cases previously reported as "out of domain" (and only spot-checked) get a real
expected-vs-actual verdict instead of being excluded from the primary claim.

Does NOT modify evidence/s3_independent/oracle_cedar.py (kept untouched for the original S3
record). Imports its CAPS/REGISTRY/principal_of and adds one Cedar `forbid` rule
(`context.amount <= 0`), which in Cedar's semantics overrides any `permit` unconditionally.
role=="" (empty) is left exactly as the original oracle already models it (principal type
"Unknown" -> REEVALUATE path), since the original oracle already handles that case correctly
(no extension needed there, confirmed in this file's main()).

usage: oracle_cedar_full_domain.py cases.json sut.json out.json
"""
import importlib.util, json, os, sys
import cedarpy

HERE = os.path.dirname(os.path.abspath(__file__))
spec = importlib.util.spec_from_file_location("oracle_cedar", os.path.join(HERE, "..", "s3_independent", "oracle_cedar.py"))
base = importlib.util.module_from_spec(spec)
sys.modules["oracle_cedar"] = base
spec.loader.exec_module(base)

EXTENDED_POLICIES = base.POLICIES + "\nforbid(principal, action, resource) when { context.amount <= 0 };"


def cedar_allow_extended(principal, amount, second, extra_entities):
    req = {
        "principal": principal,
        "action": {"type": "Action", "id": "approve"},
        "resource": {"type": "Res", "id": "r"},
        "context": {"amount": int(amount), "secondApproval": bool(second)},
    }
    return cedarpy.is_authorized(req, EXTENDED_POLICIES, base.ENTITIES + extra_entities).decision == cedarpy.Decision.Allow


def main(cases_p, sut_p, dst):
    cases = {c["id"]: c for c in json.load(open(cases_p))}
    sut = {s["id"]: s for s in json.load(open(sut_p))}
    out_of_domain_ids = [cid for cid, c in cases.items() if c["amount"] <= 0 or c["role"] == ""]

    rows = []
    mismatches = []
    for cid in out_of_domain_ids:
        c = cases[cid]
        principal, identity = base.principal_of(c["role"])
        extra = [] if principal["type"] != "Unknown" else [{"uid": principal, "attrs": {}, "parents": []}]
        permit_any = cedar_allow_extended(principal, c["amount"], True, extra) or cedar_allow_extended(principal, c["amount"], False, extra)
        oracle_deny = not permit_any
        s = sut[cid]
        ACCEPT = {"EXACT", "SAFE", "CONDITIONAL"}
        sut_denies = bool(s["err"]) or s["class"] not in ACCEPT  # error, REJECT, or REEVALUATE all count as "denied"
        row = {"id": cid, "role": c["role"], "amount": c["amount"], "oracle_deny": oracle_deny,
               "sut_denies": sut_denies, "sut_class": s["class"], "sut_err": s["err"]}
        rows.append(row)
        if oracle_deny and not sut_denies:
            mismatches.append(row)  # the only dangerous direction: oracle says deny, SUT permits

    json.dump({"n": len(rows), "mismatches_oracle_deny_sut_permits": mismatches, "rows": rows}, open(dst, "w"), indent=1)
    print(f"checked {len(rows)} previously out-of-domain cases with an explicit amount<=0 forbid rule")
    print(f"oracle_deny but SUT permits (dangerous direction): {len(mismatches)}")
    agree = sum(1 for r in rows if r["oracle_deny"] == r["sut_denies"])
    print(f"oracle_deny == sut_denies agreement: {agree}/{len(rows)}")


if __name__ == "__main__":
    if len(sys.argv) != 4:
        sys.exit(__doc__)
    main(*sys.argv[1:])
