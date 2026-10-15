#!/usr/bin/env python3
"""Analysis of the S4 end-to-end runs on real Fabric (authcc v2.27/2.28) with real Besu proofs.
Reads e2e_timing_run{1,2}.json (in-chaincode endorsement timings from `peer chaincode query`, 22 files x 30 reps x 4 variants,
order rotated per repetition) and e2e_commit.json (88 real submissions through ordering and commit). Writes analysis_e2e.json."""
import json, os, random, statistics as st
from scipy.stats import wilcoxon

HERE = os.path.dirname(os.path.abspath(__file__))
V = ["b4-groth16", "b1-store", "b1-nostore", "b1-sha"]
COMP = ["existsNs", "verifyNs", "ledgerNowNs", "sourceStateNs", "featureBindNs", "classifyNs", "putStateNs", "totalNs"]


def q(a, p):
    a = sorted(a)
    k = (len(a) - 1) * p
    f = int(k)
    c = min(f + 1, len(a) - 1)
    return a[f] + (a[c] - a[f]) * (k - f)


def boot(vals, n=10000, seed=20260928):
    r = random.Random(seed)
    m = sorted(st.median(r.choices(vals, k=len(vals))) for _ in range(n))
    return m[int(.025 * n)], m[int(.975 * n)]


def run(i):
    d = json.load(open(os.path.join(HERE, f"e2e_timing_run{i}.json")))
    out = {"reps": d["reps"], "files": len(d["rows"]), "variants": {}, "components_ms": {}}
    fm = {v: [st.median(s["totalNs"] for s in r["samples"][v]) for r in d["rows"]] for v in V}
    for v in V:
        pool = [s["totalNs"] for r in d["rows"] for s in r["samples"][v]]
        out["variants"][v] = {"n": len(pool), "pooled_median_ms": q(pool, .5) / 1e6, "p95_ms": q(pool, .95) / 1e6,
                              "p99_ms": q(pool, .99) / 1e6, "median_of_file_medians_ms": st.median(fm[v]) / 1e6}
        out["components_ms"][v] = {c: q([s[c] for r in d["rows"] for s in r["samples"][v]], .5) / 1e6 for c in COMP}
        if v != "b4-groth16":
            ratio = [a / b for a, b in zip(fm[v], fm["b4-groth16"])]
            lo, hi = boot(ratio)
            out["variants"][v].update({"median_ratio_over_b4": st.median(ratio), "ratio_ci95": [lo, hi],
                                       "wilcoxon_p": wilcoxon(fm[v], fm["b4-groth16"]).pvalue,
                                       "files_slower_than_b4": sum(1 for x in ratio if x > 1)})
    return out


def main():
    res = {"runs": {i: run(i) for i in (1, 2) if os.path.exists(os.path.join(HERE, f"e2e_timing_run{i}.json"))}}
    c = json.load(open(os.path.join(HERE, "e2e_commit.json")))
    res["commit"] = {"submissions": len(c), "committed_ok": sum(x["ok"] for x in c),
                     "class_equals_committed_record": sum(1 for x in c if x["class"] == x["committedClass"]),
                     "wall_ms": {v: {"median": st.median([x["wallMs"] for x in c if x["variant"] == v]),
                                     "min": min(x["wallMs"] for x in c if x["variant"] == v),
                                     "max": max(x["wallMs"] for x in c if x["variant"] == v)} for v in V}}
    by = {}
    for x in c:
        by.setdefault(x["name"], {})[x["variant"]] = x["committedClass"]
    res["commit"]["files_with_identical_committed_class_across_variants"] = sum(1 for d in by.values() if len(set(d.values())) == 1)
    json.dump(res, open(os.path.join(HERE, "analysis_e2e.json"), "w"), indent=1)
    for i, r in res["runs"].items():
        print(f"run {i}: {r['files']} files x {r['reps']} reps")
        for v in V:
            x = r["variants"][v]
            extra = "" if v == "b4-groth16" else f"  ratio vs B4 {x['median_ratio_over_b4']:.3f} CI95 [{x['ratio_ci95'][0]:.3f},{x['ratio_ci95'][1]:.3f}] Wilcoxon p={x['wilcoxon_p']:.2g}, slower on {x['files_slower_than_b4']}/{r['files']}"
            print(f"  {v:11s} total median {x['pooled_median_ms']:.3f} ms (P95 {x['p95_ms']:.3f}, P99 {x['p99_ms']:.3f}){extra}")
        for v in V:
            print("    components", v, {k[:-2]: round(val, 3) for k, val in r["components_ms"][v].items()})
    print(json.dumps(res["commit"], indent=1))


if __name__ == "__main__":
    main()
