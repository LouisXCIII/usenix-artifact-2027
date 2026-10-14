#!/usr/bin/env python3
"""S8: S7 at population scale on the real networks (protocol: evidence/s8_population_protocol_2026-09-28.md). The destination is the DEPLOYED, UNMODIFIED SubmitDecisionTrustlessV7
and never receives x, weights or code.
  python3 s8_run.py setup      SetClock(now+300) as the Org1 admin, then fetch one fresh Besu header + proofs for all registered contracts (node s8_fetch.js)
  python3 s8_run.py run        L1 (173 treatment invokes), positive controls, L2 negatives on the 113 satisfiable contracts, L3 B0 contrasts; four invokes at a time
  python3 s8_run.py blocks     fetch the Fabric blocks of the run and search their raw bytes for x and W (L4)
  python3 s8_run.py analyze    write s8_summary.json"""
import json, os, subprocess, sys, time
from concurrent.futures import ThreadPoolExecutor

HERE = os.path.dirname(os.path.abspath(__file__))
S4 = os.path.join(HERE, "..", "s4_fair_b1")
TN = "/home/deployer/authcc-network/fabric-samples/test-network"
env = dict(os.environ)
env["PATH"] = TN + "/../bin:" + env["PATH"]
env["FABRIC_CFG_PATH"] = TN + "/../config/"
env["CORE_PEER_TLS_ENABLED"] = "true"
env["CORE_PEER_LOCALMSPID"] = "Org1MSP"
P1 = TN + "/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt"
P2 = TN + "/organizations/peerOrganizations/org2.example.com/peers/peer0.org2.example.com/tls/ca.crt"
ORD = TN + "/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem"
env["CORE_PEER_TLS_ROOTCERT_FILE"] = P1
env["CORE_PEER_MSPCONFIGPATH"] = TN + "/organizations/peerOrganizations/org1.example.com/users/Admin@org1.example.com/msp"
env["CORE_PEER_ADDRESS"] = "localhost:17051"

ROLE, AMOUNT, LIMIT, OVER = "manager", 100000, 1000000, 600000
ACCEPT = {"SAFE", "EXACT", "CONDITIONAL"}
STAMP = str(int(time.time()))
WORKERS = 4


def dumps(o):
    return json.dumps(o, separators=(",", ":"))


# fixtures: S8 (167) + the 22 S4 files
F = {f["name"]: f for f in json.load(open(os.path.join(HERE, "fixtures_s8.json")))["files"]}
for f in json.load(open(os.path.join(S4, "fixtures.json")))["files"]:
    F[f["name"]] = f
S8_NAMES = sorted(n for n in F if n.startswith("c") and n[1:].isdigit())
S4_SAT = sorted(n for n in F if n.endswith(".sol") and F[n]["strict_circuit_satisfiable"])
SAT = sorted([n for n in S8_NAMES if F[n]["strict_circuit_satisfiable"]] + S4_SAT)
M1_NAMES = S8_NAMES + S4_SAT
WB = json.load(open(os.path.join(S4, "go_src", "b1_model_weights.json")))


def expected(n):
    return "SAFE" if F[n]["strict_circuit_satisfiable"] else "REJECT"


def state():
    global _ST
    try:
        return _ST
    except NameError:
        _ST = json.load(open(os.path.join(HERE, "besu_state_s8.json")))
        return _ST


def proof_pub(n):
    f = F[n]
    return (f["strict_proof"], f["strict_public"]) if f["strict_circuit_satisfiable"] else (f["proof"], f["public"])


def v7_args(did, n, *, code=None, proof_from=None, ml_from=None, feat_from=None, amount=AMOUNT, tamper=False, pub_feature_from=None):
    st = state(); e = st["entries"]
    code = code or n
    pf, pub = proof_pub(proof_from or n)
    pub = list(pub)
    if pub_feature_from:
        pub[6] = proof_pub(pub_feature_from)[1][6]
    if tamper:
        pf = json.loads(json.dumps(pf)); s = pf["pi_a"][0]; pf["pi_a"][0] = s[:-1] + str((int(s[-1]) + 1) % 10)
    return [did, "besu-mainnet", ROLE, str(amount), str(LIMIT), e[code]["codeHash"], dumps(pf), dumps(pub), dumps(st["header"]),
            dumps(e[ml_from or code]["mlProof"]), dumps(e[feat_from or code]["featProof"])]


def b0_args(did, n, *, code=None, amount=AMOUNT):
    st = state(); e = st["entries"]; code = code or n
    pf, pub = proof_pub(n)
    return [did, "besu-mainnet", ROLE, str(amount), str(LIMIT), e[code]["codeHash"], dumps(pf), dumps(pub), dumps(st["header"]), dumps(e[code]["mlProof"])]


def b1_args(did, n):
    st = state(); e = st["entries"]
    return [did, "besu-mainnet", ROLE, str(AMOUNT), str(LIMIT), e[n]["codeHash"], dumps(F[n]["x_q"]), dumps(st["header"]), dumps(e[n]["mlProof"]), dumps(e[n]["featProof"])]


def peer_invoke(fn, args):
    cmd = ["peer", "chaincode", "invoke", "-o", "localhost:17050", "--ordererTLSHostnameOverride", "orderer.example.com", "--tls", "--cafile", ORD,
           "-C", "authchannel", "-n", "authcc", "--peerAddresses", "localhost:17051", "--tlsRootCertFiles", P1,
           "--peerAddresses", "localhost:19051", "--tlsRootCertFiles", P2, "--waitForEvent", "-c", json.dumps({"function": fn, "Args": args})]
    t0 = time.perf_counter()
    r = subprocess.run(cmd, env=env, capture_output=True, text=True)
    wall = (time.perf_counter() - t0) * 1000
    out = r.stdout + r.stderr
    ok = "Chaincode invoke successful" in out
    cls = out.split('payload:"')[1].split('"')[0] if ok and 'payload:"' in out else None
    return {"ok": ok, "class": cls, "wallMs": wall, "tail": "" if ok else out.strip()[-200:]}


def read_decision(did):
    for _ in range(8):
        r = subprocess.run(["peer", "chaincode", "query", "-C", "authchannel", "-n", "authcc", "-c", json.dumps({"function": "ReadDecision", "Args": [did]})],
                           env=env, capture_output=True, text=True)
        try:
            return json.loads(r.stdout)
        except Exception:
            time.sleep(1)
    return None


def height():
    r = subprocess.run(["peer", "channel", "getinfo", "-c", "authchannel"], env=env, capture_output=True, text=True)
    return json.loads((r.stdout + r.stderr).split("Blockchain info: ")[1].split("\n")[0])["height"]


def cp_upper(n):
    return 1 - 0.05 ** (1.0 / n) if n else None


def setup():
    t = int(time.time()) + 300
    res = peer_invoke("SetClock", [str(t)])
    print("SetClock", t, res["ok"], res["tail"])
    if not res["ok"]:
        sys.exit(1)
    time.sleep(3)
    r = subprocess.run(["node", os.path.join(HERE, "s8_fetch.js")], capture_output=True, text=True)
    print(r.stdout, r.stderr[-300:])


def do(job):
    kind, fn, args, meta = job
    r = peer_invoke(fn, args)
    rec = read_decision(args[0]) if r["ok"] else None
    return {**meta, "kind": kind, "decisionId": args[0], "ok": r["ok"], "returned": r["class"], "committed": (rec or {}).get("transferClass"), "wallMs": r["wallMs"], "tail": r["tail"]}


def run():
    log = {"stamp": STAMP, "heightBefore": height(), "sat": SAT, "m1_names": M1_NAMES}
    print("height before", log["heightBefore"], "sat", len(SAT), "m1", len(M1_NAMES), flush=True)

    def save():
        json.dump(log, open(os.path.join(HERE, "s8_run.json"), "w"))

    def batch(jobs, key):
        with ThreadPoolExecutor(WORKERS) as ex:
            res = list(ex.map(do, jobs))
        log[key] = res
        print(key, len(res), "ok", sum(r["ok"] for r in res), flush=True)
        save()
        return res

    # L1 treatment
    jobs = []
    for n in M1_NAMES:
        a = v7_args(f"s8-t-{n}-{STAMP}", n)
        assert not any(",".join(map(str, F[n]["x_q"][:16])) in s for s in a), "x present in treatment args"
        jobs.append(("L1", "SubmitDecisionTrustlessV7", a, {"name": n, "expected": expected(n)}))
    batch(jobs, "l1")
    # positive controls: B1Store WITH x on three satisfiable contracts
    ctl = [SAT[0], SAT[len(SAT) // 2], SAT[-1]]
    batch([("CTL", "SubmitDecisionB1Store", b1_args(f"s8-ctl-{n}-{STAMP}", n), {"name": n}) for n in ctl], "controls")
    # L2 negatives on all satisfiable contracts
    cats = {
        "N1_proof_substitution": lambda i, n, o: v7_args(f"s8-n1-{n}-{STAMP}", o, proof_from=n),
        "N2_tampered_proof": lambda i, n, o: v7_args(f"s8-n2-{n}-{STAMP}", n, tamper=True),
        "N3_forged_featureHash": lambda i, n, o: v7_args(f"s8-n3-{n}-{STAMP}", n, pub_feature_from=o),
        "N4_wrong_feature_proof": lambda i, n, o: v7_args(f"s8-n4-{n}-{STAMP}", n, feat_from=o),
        "N5_amount_over_threshold": lambda i, n, o: v7_args(f"s8-n5-{n}-{STAMP}", n, amount=OVER),
    }
    jobs = []
    for cat, mk in cats.items():
        for i, n in enumerate(SAT):
            o = SAT[(i + 1) % len(SAT)]
            jobs.append((cat, "SubmitDecisionTrustlessV7", mk(i, n, o), {"name": n, "other": o}))
    batch(jobs, "negatives")
    # N6 replay of the accepted M1 decisions
    l1 = {r["name"]: r for r in log["l1"]}
    jobs = [("N6_replay", "SubmitDecisionTrustlessV7", v7_args(l1[n]["decisionId"], n), {"name": n}) for n in SAT]
    before = {n: read_decision(l1[n]["decisionId"]) for n in SAT}
    res = batch(jobs, "replays")
    for r in res:
        r["recordUnchanged"] = before[r["name"]] == read_decision(r["decisionId"])
    save()
    # L3 B0 contrasts
    jobs = []
    for i, n in enumerate(SAT):
        o = SAT[(i + 1) % len(SAT)]
        jobs.append(("B0_N1_proof_substitution", "SubmitDecisionB0", b0_args(f"s8-b0n1-{n}-{STAMP}", n, code=o), {"name": n}))
        jobs.append(("B0_N5_amount_over_threshold", "SubmitDecisionB0", b0_args(f"s8-b0n5-{n}-{STAMP}", n, amount=OVER), {"name": n}))
    batch(jobs, "b0")
    time.sleep(4)
    log["heightAfter"] = height()
    print("height after", log["heightAfter"], flush=True)
    save()


def blocks():
    log = json.load(open(os.path.join(HERE, "s8_run.json")))
    lo, hi = log["heightBefore"], log["heightAfter"]
    ctl_ids = {r["decisionId"] for r in log["controls"]}
    ctl_names = {r["name"] for r in log["controls"]}
    names = sorted(set(log["m1_names"]))
    keys = {n: ",".join(map(str, F[n]["x_q"][:16])).encode() for n in names}
    wkey = ",".join(map(str, WB["W_q"][0][:8])).encode()
    os.makedirs(os.path.join(HERE, "blocks_tmp"), exist_ok=True)
    rows = []
    for b in range(lo, hi):
        f = os.path.join(HERE, "blocks_tmp", f"b{b}.block")
        subprocess.run(["peer", "channel", "fetch", str(b), f, "-o", "localhost:17050", "--ordererTLSHostnameOverride", "orderer.example.com",
                        "-c", "authchannel", "--tls", "--cafile", ORD], env=env, capture_output=True, text=True)
        data = open(f, "rb").read() if os.path.exists(f) else b""
        rows.append({"block": b, "bytes": len(data), "hasControlId": any(i.encode() in data for i in ctl_ids),
                     "xHits": sorted(n for n, k in keys.items() if k in data), "wHit": wkey in data})
        if b % 50 == 0:
            print("block", b, flush=True)
    json.dump({"range": [lo, hi], "rows": rows, "controls": sorted(ctl_names)}, open(os.path.join(HERE, "s8_blocks.json"), "w"))
    ctl_rows = [r for r in rows if r["hasControlId"]]
    treat = [r for r in rows if not r["hasControlId"]]
    print("blocks", len(rows), "control blocks", [r["block"] for r in ctl_rows], "control hits", [r["xHits"] for r in ctl_rows])
    print("treatment-block x hits:", sum(len(r["xHits"]) for r in treat), "W hits:", sum(r["wHit"] for r in treat))


def analyze():
    log = json.load(open(os.path.join(HERE, "s8_run.json")))
    out = {}
    l1 = log["l1"]
    ok_rows = [r for r in l1 if r["ok"]]
    out["L1"] = {"n": len(l1), "invoked_ok": len(ok_rows), "committed_match": sum(1 for r in ok_rows if r["committed"] == r["expected"]),
                 "safe": sum(1 for r in ok_rows if r["committed"] == "SAFE"), "reject": sum(1 for r in ok_rows if r["committed"] == "REJECT"),
                 "mismatches": [(r["name"], r["expected"], r["committed"]) for r in ok_rows if r["committed"] != r["expected"]][:10],
                 "median_wall_ms": sorted(r["wallMs"] for r in l1)[len(l1) // 2]}
    neg = {}
    for r in log["negatives"] + log["replays"]:
        c = neg.setdefault(r["kind"], {"n": 0, "accepted": 0, "errors": 0, "reject_or_reeval": 0})
        c["n"] += 1
        cls = r["committed"] or r["returned"]
        if r["kind"] == "N6_replay":
            c["accepted"] += 0 if (r["recordUnchanged"] and not r["ok"]) else 1
            c["errors"] += 0 if r["ok"] else 1
        elif not r["ok"]:
            c["errors"] += 1
        elif cls in ACCEPT:
            c["accepted"] += 1
        else:
            c["reject_or_reeval"] += 1
    for c in neg.values():
        c["cp95_upper"] = cp_upper(c["n"]) if c["accepted"] == 0 else None
    out["L2"] = neg
    b0 = {}
    for r in log["b0"]:
        c = b0.setdefault(r["kind"], {"n": 0, "accepted": 0, "errors": 0})
        c["n"] += 1
        if not r["ok"]:
            c["errors"] += 1
        elif (r["committed"] or r["returned"]) in ACCEPT:
            c["accepted"] += 1
    out["L3_B0"] = b0
    out["controls"] = [{"name": r["name"], "ok": r["ok"], "class": r["returned"]} for r in log["controls"]]
    p = os.path.join(HERE, "s8_blocks.json")
    if os.path.exists(p):
        blk = json.load(open(p))
        ctl_rows = [r for r in blk["rows"] if r["hasControlId"]]
        treat = [r for r in blk["rows"] if not r["hasControlId"]]
        found = sorted({n for r in ctl_rows for n in r["xHits"]})
        out["L4"] = {"blocks": len(blk["rows"]), "control_blocks": [r["block"] for r in ctl_rows], "controls_expected": blk["controls"], "control_x_found_for": found,
                     "treatment_blocks": len(treat), "treatment_x_hits": sum(len(r["xHits"]) for r in treat), "treatment_W_hits": sum(r["wHit"] for r in treat)}
    json.dump(out, open(os.path.join(HERE, "s8_summary.json"), "w"), indent=1)
    print(json.dumps(out, indent=1))


if __name__ == "__main__":
    {"setup": setup, "run": run, "blocks": blocks, "analyze": analyze}[sys.argv[1]]()
