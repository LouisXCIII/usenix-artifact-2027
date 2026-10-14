#!/usr/bin/env python3
"""S4 analysis (protocol: evidence/s4_fair_b1_protocol_2026-09-28.md). Reads timings_run{1,2,3}.json, writes analysis_s4.json.
Per run and variant: pooled median/P95/P99 (ms), median of per-file medians, ratio to B4 (Groth16 verify) per file,
bootstrap 95% CI (10,000 resamples of the 22 files, seed 20260928) on the median per-file ratio, and a paired Wilcoxon
signed-rank test on the 22 per-file medians. Disclosure counts come from the serialised argument sizes."""
import json, os, random, statistics as st
from scipy.stats import wilcoxon

HERE = os.path.dirname(os.path.abspath(__file__))
VARIANTS = {"b1_store_ns": "B1-store (Poseidon x, stored model)", "b1_nostore_ns": "B1-nostore (Poseidon x + Poseidon W,b)",
            "b1_sha_ns": "B1-sha (SHA-256 commitment to x, stored model) [amendment A1]"}


def q(a, p):
    a = sorted(a)
    k = (len(a) - 1) * p
    f = int(k)
    c = min(f + 1, len(a) - 1)
    return a[f] + (a[c] - a[f]) * (k - f)


def boot_ci(vals, seed=20260928, n=10000):
    rnd = random.Random(seed)
    meds = sorted(st.median(rnd.choices(vals, k=len(vals))) for _ in range(n))
    return meds[int(0.025 * n)], meds[int(0.975 * n)]


def analyse(run):
    d = json.load(open(os.path.join(HERE, f"timings_run{run}.json")))
    out = {"reps": d["reps"], "files": len(d["files"]), "variants": {}}
    b4 = [st.median(f["b4_groth16_ns"]) for f in d["files"]]
    pool4 = [v for f in d["files"] for v in f["b4_groth16_ns"]]
    out["b4_groth16_verify_ms"] = {"pooled_median": q(pool4, .5) / 1e6, "p95": q(pool4, .95) / 1e6, "p99": q(pool4, .99) / 1e6,
                                   "median_of_file_medians": st.median(b4) / 1e6}
    for key, label in VARIANTS.items():
        if not d["files"][0].get(key):
            continue
        b1 = [st.median(f[key]) for f in d["files"]]
        pool = [v for f in d["files"] for v in f[key]]
        ratio = [x / y for x, y in zip(b1, b4)]  # B1 time / B4 time, per file
        w = wilcoxon(b1, b4)
        lo, hi = boot_ci(ratio)
        out["variants"][key] = {"label": label, "pooled_median_ms": q(pool, .5) / 1e6, "p95_ms": q(pool, .95) / 1e6,
                                "p99_ms": q(pool, .99) / 1e6, "median_of_file_medians_ms": st.median(b1) / 1e6,
                                "median_ratio_B1_over_B4": st.median(ratio), "ratio_ci95": [lo, hi],
                                "wilcoxon_p": w.pvalue, "files_where_b1_slower": sum(1 for r in ratio if r > 1)}
    return out


def main():
    res = {"runs": {}, "disclosure": {}}
    for r in (1, 2, 3):
        p = os.path.join(HERE, f"timings_run{r}.json")
        if os.path.exists(p):
            res["runs"][r] = analyse(r)
    d = json.load(open(os.path.join(HERE, "timings_run2.json")))
    f0 = d["files"][0]
    res["disclosure"] = {
        "b1_must_receive_x": {"fields": 256, "json_bytes": f0["x_json_bytes"], "source_private": True},
        "b1_nostore_also_receives_W_b": {"fields": 3084, "json_bytes": f0["wb_json_bytes"],
                                         "source_private": "only if the model is proprietary (here it is pinned by a public hash)"},
        "adp_receives": {"proof_json_bytes": f0["proof_json_bytes"], "public_signals_json_bytes": f0["public_json_bytes"],
                         "public_signals": 7, "raw_feature_fields_disclosed": 0,
                         "note": "featureHash and commitmentHash are digests, not the features"},
    }
    comp = os.path.join(HERE, "components_run1.json")
    if os.path.exists(comp):
        res["components_ms"] = json.load(open(comp))
    json.dump(res, open(os.path.join(HERE, "analysis_s4.json"), "w"), indent=1)
    for r, a in res["runs"].items():
        print(f"run {r}: files={a['files']} reps={a['reps']}  B4 Groth16 verify: median {a['b4_groth16_verify_ms']['pooled_median']:.3f} ms "
              f"(P95 {a['b4_groth16_verify_ms']['p95']:.3f}, P99 {a['b4_groth16_verify_ms']['p99']:.3f})")
        for k, v in a["variants"].items():
            print(f"   {v['label']}: median {v['pooled_median_ms']:.3f} ms (P95 {v['p95_ms']:.3f}, P99 {v['p99_ms']:.3f}); "
                  f"B1/B4 ratio {v['median_ratio_B1_over_B4']:.3f} CI95 [{v['ratio_ci95'][0]:.3f},{v['ratio_ci95'][1]:.3f}]; "
                  f"Wilcoxon p={v['wilcoxon_p']:.2g}; B1 slower on {v['files_where_b1_slower']}/{a['files']} files")
    print(json.dumps(res["disclosure"], indent=1))


if __name__ == "__main__":
    main()
