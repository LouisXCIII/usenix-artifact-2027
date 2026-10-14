#!/usr/bin/env python3
"""S8 re-run on the cleaned chaincode (authcc v2.30/seq42, b1fair.go/poseidon_opt.go/s4_b1_entrypoints.go/b1_model_weights.json
removed), mirroring s7_rerun_v230.py. Protocol amendment (s7_no_code_destination_protocol_2026-09-28.md A1 and the same
finding applies to S8, which ran on the same compromised package): re-runs L1 (173 treatment invokes), L2 negatives on the
113 satisfiable contracts (5 categories x113 + 113 replay), L3 B0 contrasts (226), on the real cleaned package. Does not
repeat the SubmitDecisionB1Store positive control (entry point removed); the block-search methodology's validity is
carried over from the original run's positive controls (block 3510, s8_run.json).
  python3 s8_rerun_v230.py setup    SetClock(now+300) then fetch a fresh Besu header + proofs for all 189 registered contracts
  python3 s8_rerun_v230.py run      L1 (173), L2 negatives (565) + N6 replay (113), L3 B0 (226); four invokes at a time
  python3 s8_rerun_v230.py blocks   fetch this run's Fabric blocks and search for x/W (no positive control repeated)
  python3 s8_rerun_v230.py analyze  compare against the original s8_run.json and write s8_rerun_v230_summary.json
"""
import importlib.util, json, os, sys, time
from concurrent.futures import ThreadPoolExecutor

HERE = os.path.dirname(os.path.abspath(__file__))
spec = importlib.util.spec_from_file_location("s8_run", os.path.join(HERE, "s8_run.py"))
m = importlib.util.module_from_spec(spec)
spec.loader.exec_module(m)

STAMP = str(int(time.time()))
OUT = os.path.join(HERE, "s8_rerun_v230.json")


def setup():
    m.setup()


def run():
    log = {"stamp": STAMP, "chaincodeVersion": "authcc_2.30/seq42", "heightBefore": m.height(), "sat": m.SAT, "m1_names": m.M1_NAMES}
    print("height before", log["heightBefore"], "sat", len(m.SAT), "m1", len(m.M1_NAMES), flush=True)

    def save():
        json.dump(log, open(OUT, "w"))

    def batch(jobs, key):
        with ThreadPoolExecutor(m.WORKERS) as ex:
            res = list(ex.map(m.do, jobs))
        log[key] = res
        print(key, len(res), "ok", sum(r["ok"] for r in res), flush=True)
        save()
        return res

    jobs = []
    for n in m.M1_NAMES:
        a = m.v7_args(f"s8v230-t-{n}-{STAMP}", n)
        jobs.append(("L1", "SubmitDecisionTrustlessV7", a, {"name": n, "expected": m.expected(n)}))
    batch(jobs, "l1")

    cats = {
        "N1_proof_substitution": lambda i, n, o: m.v7_args(f"s8v230-n1-{n}-{STAMP}", o, proof_from=n),
        "N2_tampered_proof": lambda i, n, o: m.v7_args(f"s8v230-n2-{n}-{STAMP}", n, tamper=True),
        "N3_forged_featureHash": lambda i, n, o: m.v7_args(f"s8v230-n3-{n}-{STAMP}", n, pub_feature_from=o),
        "N4_wrong_feature_proof": lambda i, n, o: m.v7_args(f"s8v230-n4-{n}-{STAMP}", n, feat_from=o),
        "N5_amount_over_threshold": lambda i, n, o: m.v7_args(f"s8v230-n5-{n}-{STAMP}", n, amount=m.OVER),
    }
    jobs = []
    for cat, mk in cats.items():
        for i, n in enumerate(m.SAT):
            o = m.SAT[(i + 1) % len(m.SAT)]
            jobs.append((cat, "SubmitDecisionTrustlessV7", mk(i, n, o), {"name": n, "other": o}))
    batch(jobs, "negatives")

    l1 = {r["name"]: r for r in log["l1"]}
    jobs = [("N6_replay", "SubmitDecisionTrustlessV7", m.v7_args(l1[n]["decisionId"], n), {"name": n}) for n in m.SAT]
    before = {n: m.read_decision(l1[n]["decisionId"]) for n in m.SAT}
    res = batch(jobs, "replays")
    for r in res:
        r["recordUnchanged"] = before[r["name"]] == m.read_decision(r["decisionId"])
    save()

    jobs = []
    for i, n in enumerate(m.SAT):
        o = m.SAT[(i + 1) % len(m.SAT)]
        jobs.append(("B0_N1_proof_substitution", "SubmitDecisionB0", m.b0_args(f"s8v230-b0n1-{n}-{STAMP}", n, code=o), {"name": n}))
        jobs.append(("B0_N5_amount_over_threshold", "SubmitDecisionB0", m.b0_args(f"s8v230-b0n5-{n}-{STAMP}", n, amount=m.OVER), {"name": n}))
    batch(jobs, "b0")

    time.sleep(4)
    log["heightAfter"] = m.height()
    print("height after", log["heightAfter"], flush=True)
    save()


def blocks():
    log = json.load(open(OUT))
    lo, hi = log["heightBefore"], log["heightAfter"]
    names = sorted(set(log["m1_names"]))
    keys = {n: ",".join(map(str, m.F[n]["x_q"][:16])).encode() for n in names}
    wkey = ",".join(map(str, m.WB["W_q"][0][:8])).encode()
    os.makedirs(os.path.join(HERE, "blocks_tmp_v230"), exist_ok=True)
    rows = []
    import subprocess
    for b in range(lo, hi):
        f = os.path.join(HERE, "blocks_tmp_v230", f"b{b}.block")
        subprocess.run(["peer", "channel", "fetch", str(b), f, "-o", "localhost:17050", "--ordererTLSHostnameOverride", "orderer.example.com",
                        "-c", "authchannel", "--tls", "--cafile", m.ORD], env=m.env, capture_output=True, text=True)
        data = open(f, "rb").read() if os.path.exists(f) else b""
        rows.append({"block": b, "bytes": len(data), "xHits": sorted(n for n, k in keys.items() if k in data), "wHit": wkey in data})
        if b % 50 == 0:
            print("block", b, flush=True)
    json.dump({"range": [lo, hi], "rows": rows}, open(os.path.join(HERE, "s8_rerun_v230_blocks.json"), "w"), indent=1)
    print("blocks searched", len(rows), "x hits (expect 0):", sum(len(r["xHits"]) for r in rows), "W hits (expect 0):", sum(r["wHit"] for r in rows))


def analyze():
    log = json.load(open(OUT))
    orig = json.load(open(os.path.join(HERE, "s8_run.json")))
    orig_l1 = {r["name"]: r["committed"] for r in orig["l1"]}
    l1 = log["l1"]
    out = {
        "chaincodeVersion": "authcc_2.30/seq42 (b1fair.go/poseidon_opt.go/s4_b1_entrypoints.go/b1_model_weights.json absent)",
        "L1": {"n": len(l1), "committed_match_expected": sum(1 for r in l1 if r["committed"] == r["expected"]),
               "committed_match_original_run": sum(1 for r in l1 if r["committed"] == orig_l1.get(r["name"])),
               "safe": sum(1 for r in l1 if r["committed"] == "SAFE"), "reject": sum(1 for r in l1 if r["committed"] == "REJECT")},
    }
    neg = {}
    for r in log["negatives"] + log["replays"]:
        c = neg.setdefault(r["kind"], {"n": 0, "accepted": 0, "errors": 0})
        c["n"] += 1
        cls = r.get("committed") or r.get("returned")
        if r["kind"] == "N6_replay":
            c["accepted"] += 0 if (r.get("recordUnchanged") and not r["ok"]) else 1
        elif not r["ok"]:
            c["errors"] += 1
        elif cls in m.ACCEPT:
            c["accepted"] += 1
    out["L2"] = neg
    b0 = {}
    for r in log["b0"]:
        c = b0.setdefault(r["kind"], {"n": 0, "accepted": 0})
        c["n"] += 1
        if (r.get("committed") or r.get("returned")) in m.ACCEPT:
            c["accepted"] += 1
    out["L3_B0"] = b0
    p = os.path.join(HERE, "s8_rerun_v230_blocks.json")
    if os.path.exists(p):
        blk = json.load(open(p))
        out["L4"] = {"blocks": len(blk["rows"]), "x_hits": sum(len(r["xHits"]) for r in blk["rows"]), "w_hits": sum(r["wHit"] for r in blk["rows"])}
    json.dump(out, open(os.path.join(HERE, "s8_rerun_v230_summary.json"), "w"), indent=1)
    print(json.dumps(out, indent=1))


if __name__ == "__main__":
    {"setup": setup, "run": run, "blocks": blocks, "analyze": analyze}[sys.argv[1]]()
