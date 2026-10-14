#!/usr/bin/env python3
"""Analysis for S3 (metrics as defined in s3_independent_groundtruth_protocol_2026-09-28.md).
usage: analyze_s3.py cases.json sut.json oracle.json summary.json
Definitions: unsafe accept = SUT EXACT/SAFE while the oracle does not permit unconditionally, or SUT CONDITIONAL while the oracle denies even with the second
approval, or SUT returns a class for a replay. Over-conservative = oracle permits but SUT asks for more or denies. Everything else that differs = class mismatch.
Only in-domain cases enter the primary metrics; out-of-domain and int64-overflow cases are counted separately."""
import json, sys
from collections import Counter, defaultdict

ACCEPT = {"EXACT", "SAFE"}


def sut_class(o, which):
    cls, err = (o["class"], o["err"]) if which == "full" else (o["class_basic"], o["err_basic"])
    if err:
        return "REPLAY_ERROR" if "replay rejected" in err else "ERROR:" + err[:40]
    return cls


def oracle_for(orc, cse, which):
    """Oracle class for the full function, or for the basic one (no registry, no obligation, never EXACT)."""
    if which == "full":
        return orc["oracle_class"]
    if orc["replay"]:
        return "REPLAY_ERROR"
    if not (cse["semBound"] and cse["evAssuranceOK"]):
        return "REJECT"
    if orc["principal_type"] != "DestRole":
        return "REEVALUATE"
    return "SAFE" if orc["B"] else "REJECT"


def verdict(sut, oracle):
    permitted_uncond = oracle in ACCEPT
    permitted_with_ob = oracle in ACCEPT or oracle == "CONDITIONAL"
    if oracle == "REPLAY_ERROR":
        return "match" if sut == "REPLAY_ERROR" else ("unsafe_accept" if sut in ACCEPT | {"CONDITIONAL"} else "class_mismatch")
    if sut in ACCEPT and not permitted_uncond:
        return "unsafe_accept"
    if sut == "CONDITIONAL" and not permitted_with_ob:
        return "unsafe_accept"
    if sut == oracle:
        return "match"
    if permitted_uncond and sut not in ACCEPT:
        return "over_conservative"
    if oracle == "CONDITIONAL" and sut in {"REJECT", "REEVALUATE"}:
        return "over_conservative"
    return "class_mismatch"


def cp_upper(n):
    return None if n == 0 else 1 - 0.05 ** (1 / n)


def main(cases_p, sut_p, orc_p, dst):
    cases = {c["id"]: c for c in json.load(open(cases_p))}
    sut = {o["id"]: o for o in json.load(open(sut_p))}
    orc = {o["id"]: o for o in json.load(open(orc_p))}
    summary = {}
    for which in ("full", "basic"):
        per = defaultdict(Counter)
        overall, outdom, skipped, adv_vs_orc = Counter(), Counter(), 0, Counter()
        examples = defaultdict(list)
        for i, c in cases.items():
            if sut[i].get("skipped"):
                skipped += 1
                continue
            s, o = sut_class(sut[i], which), oracle_for(orc[i], c, which)
            v = verdict(s, o)
            if which == "full" and c.get("expected"):
                adv_vs_orc["agree" if c["expected"] == o else "disagree"] += 1
            if not orc[i]["in_domain"]:
                outdom[v] += 1
                if v != "match" and len(examples["out_of_domain"]) < 10:
                    examples["out_of_domain"].append((i, c["role"], c["amount"], s, o))
                continue
            overall[v] += 1
            per[c["category"]][v] += 1
            if v != "match" and len(examples[v]) < 12:
                examples[v].append((i, c["category"], c["role"], c["amount"], c["semBound"], c["evAssuranceOK"], c["secondApprovalObtained"], s, o))
        n = sum(overall.values())
        summary[which] = {
            "in_domain_cases": n, "overall": dict(overall), "unsafe_accepts": overall["unsafe_accept"],
            "cp95_upper_bound_if_zero": cp_upper(n) if overall["unsafe_accept"] == 0 else None,
            "out_of_domain": dict(outdom), "skipped_int64_overflow": skipped,
            "per_category": {k: dict(v) for k, v in sorted(per.items())}, "examples": {k: v for k, v in examples.items()},
            "adversary_expected_vs_oracle": dict(adv_vs_orc),
        }
    json.dump(summary, open(dst, "w"), indent=1, ensure_ascii=False, default=str)
    for which, s in summary.items():
        print(f"\n== target: {which}  in-domain cases {s['in_domain_cases']}: {s['overall']}  unsafe accepts {s['unsafe_accepts']}"
              + (f"  (95% CP upper bound {100*s['cp95_upper_bound_if_zero']:.3f}%)" if s["cp95_upper_bound_if_zero"] else ""))
        print("   out-of-domain:", s["out_of_domain"], "| int64-overflow skipped:", s["skipped_int64_overflow"])
        if s["adversary_expected_vs_oracle"]:
            print("   adversary's expected class vs Cedar oracle:", s["adversary_expected_vs_oracle"])


if __name__ == "__main__":
    if len(sys.argv) != 5:
        sys.exit(__doc__)
    main(*sys.argv[1:])
