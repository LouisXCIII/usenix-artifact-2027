#!/usr/bin/env python3
"""S7 re-run on the cleaned chaincode (authcc v2.30/seq42, b1fair.go/poseidon_opt.go/s4_b1_entrypoints.go/b1_model_weights.json removed).
Protocol amendment (evidence/s7_no_code_destination_protocol_2026-09-28.md, A1, 2026-09-29): the original run's premise
("the destination is given no code, no features and no model weights") was violated at the packaging level -- the installed
chaincode package at the time contained the raw model weights via b1fair.go's go:embed (found by S9 Measure C, evidence/
s9_dictionary_attack_results_2026-09-29.md). This script re-runs M1 (22 treatment invokes), M4 (30 negative controls + 6
replay attempts), and M5 (12 B0 contrasts) on the real, now-cleaned package, and re-runs the block-content search (M2b)
on this run's own blocks. It does NOT repeat the B1Store positive control (M2's validation control), because
SubmitDecisionB1Store no longer exists in this package; the block-search methodology's validity is carried over from the
original run's positive control (block 3422, s7_run.json), which tests the search script's own correctness, not the
chaincode version.
  python3 s7_rerun_v230.py setup   SetClock(now+300) then fetch a fresh Besu header + proofs (node s7_fetch.js)
  python3 s7_rerun_v230.py run     M1 (22), M4 negatives (30) + N6 replay (6), M5 B0 contrasts (12)
  python3 s7_rerun_v230.py blocks  fetch this run's Fabric blocks and search for x/W (no positive control available)
  python3 s7_rerun_v230.py analyze compare against the original s7_run.json and write s7_rerun_v230_summary.json
"""
import importlib.util, json, os, sys, time

HERE = os.path.dirname(os.path.abspath(__file__))
spec = importlib.util.spec_from_file_location("s7_run", os.path.join(HERE, "s7_run.py"))
m = importlib.util.module_from_spec(spec)
spec.loader.exec_module(m)

STAMP = str(int(time.time()))
OUT = os.path.join(HERE, "s7_rerun_v230.json")


def setup():
    m.setup()


def run():
    names = sorted(m.fx)
    log = {"stamp": STAMP, "chaincodeVersion": "authcc_2.30/seq42", "heightBefore": m.height(), "m1": [], "negatives": [], "b0": []}
    print("height before", log["heightBefore"], flush=True)

    def save():
        json.dump(log, open(OUT, "w"), indent=1)

    for n in names:
        did = f"s7v230-t-{n}-{STAMP}"
        a = m.v7_args(did, n)
        r = m.peer_invoke("SubmitDecisionTrustlessV7", a)
        rec = m.read_decision(did) if r["ok"] else None
        log["m1"].append({"name": n, "decisionId": did, "ok": r["ok"], "returned": r["class"], "committed": (rec or {}).get("transferClass"),
                          "expected": m.S4_COMMIT[n], "wallMs": r["wallMs"], "tail": r["tail"]})
        print("M1", n, r["ok"], r["class"], (rec or {}).get("transferClass"), "expected", m.S4_COMMIT[n], flush=True)
        save()

    cats = {
        "N1_proof_substitution": lambda i, n, o: m.v7_args(f"s7v230-n1-{n}-{STAMP}", o, proof_from=n),
        "N2_tampered_proof": lambda i, n, o: m.v7_args(f"s7v230-n2-{n}-{STAMP}", n, tamper=True),
        "N3_forged_featureHash": lambda i, n, o: m.v7_args(f"s7v230-n3-{n}-{STAMP}", n, pub_feature_from=o),
        "N4_wrong_feature_proof": lambda i, n, o: m.v7_args(f"s7v230-n4-{n}-{STAMP}", n, feat_from=o),
        "N5_amount_over_threshold": lambda i, n, o: m.v7_args(f"s7v230-n5-{n}-{STAMP}", n, amount=m.OVER),
    }
    for cat, mk in cats.items():
        for i, n in enumerate(m.SAT):
            o = m.SAT[(i + 1) % len(m.SAT)]
            a = mk(i, n, o)
            r = m.peer_invoke("SubmitDecisionTrustlessV7", a)
            did = a[0]
            rec = m.read_decision(did) if r["ok"] else None
            log["negatives"].append({"category": cat, "name": n, "decisionId": did, "ok": r["ok"], "returned": r["class"],
                                     "committed": (rec or {}).get("transferClass"), "tail": r["tail"]})
            print(cat, n, r["ok"], r["class"], (rec or {}).get("transferClass"), flush=True)
            save()

    for n in m.SAT:
        did0 = f"s7v230-t-{n}-{STAMP}"
        before = m.read_decision(did0)
        r = m.peer_invoke("SubmitDecisionTrustlessV7", m.v7_args(did0, n))
        after = m.read_decision(did0)
        log["negatives"].append({"category": "N6_replay", "name": n, "decisionId": did0, "ok": r["ok"], "returned": r["class"],
                                 "committed": (after or {}).get("transferClass"), "recordUnchanged": before == after, "tail": r["tail"]})
        print("N6_replay", n, r["ok"], r["class"], "unchanged" if before == after else "CHANGED", flush=True)
        save()

    for i, n in enumerate(m.SAT):
        o = m.SAT[(i + 1) % len(m.SAT)]
        for cat, a in (("B0_N1_proof_substitution", m.b0_args(f"s7v230-b0n1-{n}-{STAMP}", n, code=o)),
                       ("B0_N5_amount_over_threshold", m.b0_args(f"s7v230-b0n5-{n}-{STAMP}", n, amount=m.OVER))):
            r = m.peer_invoke("SubmitDecisionB0", a)
            rec = m.read_decision(a[0]) if r["ok"] else None
            log["b0"].append({"category": cat, "name": n, "decisionId": a[0], "ok": r["ok"], "returned": r["class"], "committed": (rec or {}).get("transferClass"), "tail": r["tail"]})
            print(cat, n, r["ok"], r["class"], (rec or {}).get("transferClass"), flush=True)
            save()

    time.sleep(4)
    log["heightAfter"] = m.height()
    print("height after", log["heightAfter"], flush=True)
    save()


def blocks():
    log = json.load(open(OUT))
    lo, hi = log["heightBefore"], log["heightAfter"]
    names = sorted(m.fx)
    keys = {n: ",".join(map(str, m.fx[n]["x_q"][:16])).encode() for n in names}
    os.makedirs(os.path.join(HERE, "blocks_tmp_v230"), exist_ok=True)
    rows = []
    for b in range(lo, hi):
        f = os.path.join(HERE, "blocks_tmp_v230", f"b{b}.block")
        import subprocess
        subprocess.run(["peer", "channel", "fetch", str(b), f, "-o", "localhost:17050", "--ordererTLSHostnameOverride", "orderer.example.com",
                        "-c", "authchannel", "--tls", "--cafile", m.ORD], env=m.env, capture_output=True, text=True)
        data = open(f, "rb").read() if os.path.exists(f) else b""
        rows.append({"block": b, "bytes": len(data), "xHits": sorted(n for n, k in keys.items() if k in data)})
    json.dump({"range": [lo, hi], "rows": rows}, open(os.path.join(HERE, "s7_rerun_v230_blocks.json"), "w"), indent=1)
    treat_hits = sum(len(r["xHits"]) for r in rows)
    print("blocks searched", len(rows), "x hits (expect 0):", treat_hits)


def analyze():
    log = json.load(open(OUT))
    orig = json.load(open(os.path.join(HERE, "s7_run.json")))
    orig_m1 = {r["name"]: r["committed"] for r in orig["m1"]}
    m1 = log["m1"]
    match_orig = sum(1 for r in m1 if r["committed"] == orig_m1.get(r["name"]))
    out = {
        "chaincodeVersion": "authcc_2.30/seq42 (b1fair.go/poseidon_opt.go/s4_b1_entrypoints.go/b1_model_weights.json absent)",
        "M1": {"n": len(m1), "committed_match_expected": sum(1 for r in m1 if r["committed"] == r["expected"]),
               "committed_match_original_run": match_orig},
    }
    neg = {}
    for r in log["negatives"]:
        c = neg.setdefault(r["category"], {"n": 0, "accepted": 0, "errors": 0})
        c["n"] += 1
        cls = r.get("committed") or r.get("returned")
        if r["category"] == "N6_replay":
            c["accepted"] += 0 if (r.get("recordUnchanged") and not r["ok"]) else 1
        elif not r["ok"]:
            c["errors"] += 1
        elif cls in m.ACCEPT:
            c["accepted"] += 1
    out["negatives"] = neg
    b0 = {}
    for r in log["b0"]:
        c = b0.setdefault(r["category"], {"n": 0, "accepted": 0})
        c["n"] += 1
        if (r.get("committed") or r.get("returned")) in m.ACCEPT:
            c["accepted"] += 1
    out["b0"] = b0
    p = os.path.join(HERE, "s7_rerun_v230_blocks.json")
    if os.path.exists(p):
        blk = json.load(open(p))
        out["blocks"] = {"n": len(blk["rows"]), "x_hits": sum(len(r["xHits"]) for r in blk["rows"])}
    json.dump(out, open(os.path.join(HERE, "s7_rerun_v230_summary.json"), "w"), indent=1)
    print(json.dumps(out, indent=1))


if __name__ == "__main__":
    {"setup": setup, "run": run, "blocks": blocks, "analyze": analyze}[sys.argv[1]]()
