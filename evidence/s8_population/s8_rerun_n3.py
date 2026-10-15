#!/usr/bin/env python3
"""S8 amendment A2: re-run the N3 (forged featureHash) controls that were no-ops because the partner contract had the same featureHash as the contract itself.
For each such contract, the partner is the next satisfiable contract (cyclic order) whose featureHash DIFFERS. Same header/proofs (besu_state_s8.json), unique decision IDs.
Output: s8_rerun_n3.json. Original records stay in s8_run.json."""
import importlib.util, json, os, time

HERE = os.path.dirname(os.path.abspath(__file__))
spec = importlib.util.spec_from_file_location("s8_run", os.path.join(HERE, "s8_run.py"))
m = importlib.util.module_from_spec(spec)
spec.loader.exec_module(m)

log = json.load(open(os.path.join(HERE, "s8_run.json")))
SAT = log["sat"]
noop = [r["name"] for r in log["negatives"] if r["kind"] == "N3_forged_featureHash" and m.F[r["name"]]["feature_hash"] == m.F[r["other"]]["feature_hash"]]
print("no-op N3 controls:", noop)
out = []
for n in noop:
    i = SAT.index(n)
    partner = next(SAT[(i + k) % len(SAT)] for k in range(1, len(SAT)) if m.F[SAT[(i + k) % len(SAT)]]["feature_hash"] != m.F[n]["feature_hash"])
    r = m.do(("N3_forged_featureHash_rerun", "SubmitDecisionTrustlessV7", m.v7_args(f"s8-n3r-{n}-{m.STAMP}", n, pub_feature_from=partner), {"name": n, "other": partner}))
    r["featureHashDiffers"] = m.F[n]["feature_hash"] != m.F[partner]["feature_hash"]
    out.append(r)
    print(n, "partner", partner, r["ok"], r["returned"], r["committed"], flush=True)
json.dump(out, open(os.path.join(HERE, "s8_rerun_n3.json"), "w"), indent=1)
