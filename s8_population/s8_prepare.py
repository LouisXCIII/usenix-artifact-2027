#!/usr/bin/env python3
"""S8 preparation (protocol: evidence/s8_population_protocol_2026-09-28.md).
Offline: (1) strict satisfiability of every corpus contract from x_q and the pinned weights (replication checked against the S4 circuit flags),
(2) Measure P: scanner pass-through on corpora known to be vulnerable, with Clopper-Pearson intervals,
(3) the fixed sets for the chain runs: all strictly satisfiable contracts, and 60 non-satisfiable ones drawn with seed 20260928.
Outputs: s8_sets.json, s8_measure_p.json."""
import json, os, random
from collections import Counter, defaultdict
from scipy.stats import beta

HERE = os.path.dirname(os.path.abspath(__file__))
EV = os.path.join(HERE, "..")
fx = json.load(open(os.path.join(EV, "s4_fair_b1", "fixtures.json")))
MM = fx["min_margin"]
W = json.load(open(os.path.join(EV, "s4_fair_b1", "go_src", "b1_model_weights.json")))
Wq, bq = W["W_q"], W["b_q"]
rows = json.load(open(os.path.join(EV, "s5", "s5_corpus_features.json")))["rows"]
SEED = 20260928


def pred_margin(x):
    lg = [sum(w * v for w, v in zip(row, x)) + b for row, b in zip(Wq, bq)]
    top = sorted(range(12), key=lambda i: -lg[i])
    return top[0], lg[top[0]] - lg[top[1]]


def cp(k, n, a=0.05):
    lo = 0.0 if k == 0 else beta.ppf(a / 2, k, n - k + 1)
    hi = 1.0 if k == n else beta.ppf(1 - a / 2, k + 1, n - k)
    return lo, hi


# (1) replication check on the S4 files
ok = 0
for f in fx["files"]:
    p, m = pred_margin(f["x_q"])
    ok += (p == 0 and m >= MM) == f["strict_circuit_satisfiable"] and p == f["pred"] and m == f["margin"]
assert ok == 22, ok

items = []
for i, r in enumerate(rows):
    p, m = pred_margin(r["x_q"])
    items.append({"id": f"c{i:03d}", "path": r["path"], "corpus": r["corpus"], "pred": p, "margin": m, "strict_sat": bool(p == 0 and m >= MM),
                  "exists": os.path.exists(r["path"])})
missing = [it["path"] for it in items if not it["exists"]]
sat = [it for it in items if it["strict_sat"]]
s4_names = {os.path.basename(f["name"]) for f in fx["files"]}
sat_s4 = [it for it in sat if it["corpus"] == "test22"]
sat_new = [it for it in sat if it["corpus"] != "test22"]
unsat_pool = [it for it in items if not it["strict_sat"] and it["corpus"] != "test22"]
rng = random.Random(SEED)
unsat_sample = sorted(rng.sample(unsat_pool, 60), key=lambda it: it["id"])

# (2) Measure P
by = defaultdict(lambda: [0, 0])
for it in items:
    by[it["corpus"]][1] += 1
    by[it["corpus"]][0] += it["strict_sat"]
P = {"floor_min_margin": MM, "per_corpus": {}}
for c, (k, n) in sorted(by.items()):
    lo, hi = cp(k, n)
    P["per_corpus"][c] = {"passed": k, "n": n, "rate": k / n, "cp95": [lo, hi]}
known_vuln = ["smartbugs", "weakaccess"]
k = sum(by[c][0] for c in known_vuln); n = sum(by[c][1] for c in known_vuln)
P["known_vulnerable_pooled"] = {"corpora": known_vuln, "passed": k, "n": n, "rate": k / n, "cp95": list(cp(k, n))}
t22 = [it for it in items if it["corpus"] == "test22"]
P["test22_by_name"] = {"vuln_named_passed": sum(1 for it in t22 if "_vuln" in it["path"] and it["strict_sat"]), "vuln_named_n": sum(1 for it in t22 if "_vuln" in it["path"]),
                       "safe_named_passed": sum(1 for it in t22 if "_safe" in it["path"] and it["strict_sat"]), "safe_named_n": sum(1 for it in t22 if "_safe" in it["path"])}
P["missing_source_files"] = missing
json.dump(P, open(os.path.join(HERE, "s8_measure_p.json"), "w"), indent=1)
json.dump({"seed": SEED, "sat_new": sat_new, "sat_s4": sat_s4, "unsat_sample": unsat_sample, "counts": {"sat_all": len(sat), "sat_new": len(sat_new), "sat_s4": len(sat_s4), "unsat_sample": len(unsat_sample)}},
          open(os.path.join(HERE, "s8_sets.json"), "w"), indent=1)
print(json.dumps(P, indent=1))
print("sets:", len(sat), "satisfiable;", len(sat_new), "new;", len(sat_s4), "from S4;", len(unsat_sample), "unsat sample; missing sources:", len(missing))
