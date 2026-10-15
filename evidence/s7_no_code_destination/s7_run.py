#!/usr/bin/env python3
"""S7: a Fabric destination that never receives the code or the feature vector x (real networks, authcc deployed chaincode).
Protocol: evidence/s7_no_code_destination_protocol_2026-09-28.md (frozen before the first transaction).
  python3 s7_run.py setup      SetClock(now+300) as the Org1 admin, then fetch one fresh Besu header + proofs (node s7_fetch.js)
  python3 s7_run.py pilot      one QUERY per negative category on one file, to check that no input crashes the chaincode
  python3 s7_run.py run        M1 (22 treatment invokes), positive control (B1Store with x), M4 negatives, M5 B0 contrasts; records Fabric heights
  python3 s7_run.py blocks     fetch the Fabric blocks of the run and search their raw bytes for x and W (M2b)
  python3 s7_run.py analyze    write s7_summary.json and print the results
"""
import json, math, os, subprocess, sys, time

HERE = os.path.dirname(os.path.abspath(__file__))
S4 = os.path.join(HERE, "..", "s4_fair_b1")
TN = "~/authcc-network/fabric-samples/test-network"
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
SAT = ["03_dos_safe.sol", "03_dos_vuln.sol", "04_call_to_unknown_safe.sol", "06_unused_return_safe.sol", "08_timestamp_vuln.sol", "09_mishandled_exception_vuln.sol"]
ACCEPT = {"SAFE", "EXACT", "CONDITIONAL"}
STAMP = str(int(time.time()))


def dumps(o):
    return json.dumps(o, separators=(",", ":"))


fx = {f["name"]: f for f in json.load(open(os.path.join(S4, "fixtures.json")))["files"]}
WB = json.load(open(os.path.join(S4, "go_src", "b1_model_weights.json")))
S4_COMMIT = {r["name"]: r["committedClass"] for r in json.load(open(os.path.join(S4, "e2e_commit.json"))) if r["variant"] == "b4-groth16"}


def state():
    return json.load(open(os.path.join(HERE, "besu_state.json")))


def proof_pub(name):
    f = fx[name]
    return (f["strict_proof"], f["strict_public"]) if f["strict_circuit_satisfiable"] else (f["proof"], f["public"])


def v7_args(did, name, *, code=None, proof_from=None, ml_from=None, feat_from=None, amount=AMOUNT, tamper=False, pub_feature_from=None):
    st = state()
    e = st["entries"]
    code = code or name
    pf, pub = proof_pub(proof_from or name)
    pub = list(pub)
    if pub_feature_from:
        pub[6] = proof_pub(pub_feature_from)[1][6]
    if tamper:
        pf = json.loads(json.dumps(pf))
        s = pf["pi_a"][0]
        pf["pi_a"][0] = s[:-1] + str((int(s[-1]) + 1) % 10)
    return [did, "besu-mainnet", ROLE, str(amount), str(LIMIT), e[code]["codeHash"], dumps(pf), dumps(pub), dumps(st["header"]),
            dumps(e[ml_from or code]["mlProof"]), dumps(e[feat_from or code]["featProof"])]


def b0_args(did, name, *, code=None, amount=AMOUNT):
    st = state(); e = st["entries"]; code = code or name
    pf, pub = proof_pub(name)
    return [did, "besu-mainnet", ROLE, str(amount), str(LIMIT), e[code]["codeHash"], dumps(pf), dumps(pub), dumps(st["header"]), dumps(e[code]["mlProof"])]


def b1_args(did, name, x=True):
    st = state(); e = st["entries"]
    xs = dumps(fx[name]["x_q"]) if x else "[]"
    return [did, "besu-mainnet", ROLE, str(AMOUNT), str(LIMIT), e[name]["codeHash"], xs, dumps(st["header"]), dumps(e[name]["mlProof"]), dumps(e[name]["featProof"])]


def peer_invoke(fn, args, wait=True):
    cmd = ["peer", "chaincode", "invoke", "-o", "localhost:17050", "--ordererTLSHostnameOverride", "orderer.example.com", "--tls", "--cafile", ORD,
           "-C", "authchannel", "-n", "authcc", "--peerAddresses", "localhost:17051", "--tlsRootCertFiles", P1,
           "--peerAddresses", "localhost:19051", "--tlsRootCertFiles", P2] + (["--waitForEvent"] if wait else []) + ["-c", json.dumps({"function": fn, "Args": args})]
    t0 = time.perf_counter()
    r = subprocess.run(cmd, env=env, capture_output=True, text=True)
    wall = (time.perf_counter() - t0) * 1000
    out = r.stdout + r.stderr
    ok = "Chaincode invoke successful" in out
    cls = out.split('payload:"')[1].split('"')[0] if ok and 'payload:"' in out else None
    return {"ok": ok, "class": cls, "wallMs": wall, "tail": "" if ok else out.strip()[-260:]}


def peer_query(fn, args):
    r = subprocess.run(["peer", "chaincode", "query", "-C", "authchannel", "-n", "authcc", "-c", json.dumps({"function": fn, "Args": args})],
                       env=env, capture_output=True, text=True)
    return {"stdout": r.stdout.strip()[:300], "stderr": r.stderr.strip()[-300:]}


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
    print(subprocess.run(["node", os.path.join(HERE, "s7_fetch.js")], capture_output=True, text=True).stdout)


def pilot():
    n = SAT[0]; o = SAT[1]
    cases = {
        "treatment": v7_args("s7-pilot-t-" + STAMP, n),
        "N1_proof_substitution": v7_args("s7-pilot-n1-" + STAMP, o, proof_from=n),
        "N2_tampered_proof": v7_args("s7-pilot-n2-" + STAMP, n, tamper=True),
        "N3_forged_featureHash": v7_args("s7-pilot-n3-" + STAMP, n, pub_feature_from=o),
        "N4_wrong_feature_proof": v7_args("s7-pilot-n4-" + STAMP, n, feat_from=o),
        "N5_amount_over_threshold": v7_args("s7-pilot-n5-" + STAMP, n, amount=OVER),
    }
    out = {}
    for k, a in cases.items():
        out[k] = peer_query("SubmitDecisionTrustlessV7", a)
        print(k, out[k], flush=True)
    out["C3_B1Store_without_x"] = peer_query("SubmitDecisionB1Store", b1_args("s7-pilot-c3-" + STAMP, n, x=False))
    print("C3", out["C3_B1Store_without_x"])
    json.dump(out, open(os.path.join(HERE, "s7_pilot.json"), "w"), indent=1)


def run():
    names = sorted(fx)
    log = {"stamp": STAMP, "heightBefore": height(), "m1": [], "positiveControlB1": None, "negatives": [], "b0": [], "c3": None, "argBytes": []}
    print("height before", log["heightBefore"], flush=True)

    def save():
        json.dump(log, open(os.path.join(HERE, "s7_run.json"), "w"), indent=1)

    # M1 treatment (22 files)
    for n in names:
        did = f"s7-t-{n}-{STAMP}"
        a = v7_args(did, n)
        assert not any(dumps(fx[n]["x_q"]) in s or ",".join(map(str, fx[n]["x_q"][:16])) in s for s in a), "x present in treatment args"
        r = peer_invoke("SubmitDecisionTrustlessV7", a)
        rec = read_decision(did) if r["ok"] else None
        b1 = b1_args(did, n)
        log["m1"].append({"name": n, "decisionId": did, "ok": r["ok"], "returned": r["class"], "committed": (rec or {}).get("transferClass"),
                          "expected": S4_COMMIT[n], "wallMs": r["wallMs"], "tail": r["tail"]})
        log["argBytes"].append({"name": n, "treatmentBytes": sum(len(s.encode()) for s in a), "b1StoreFormBytes": sum(len(s.encode()) for s in b1),
                                "xBytes": len(dumps(fx[n]["x_q"]).encode())})
        print("M1", n, r["ok"], r["class"], (rec or {}).get("transferClass"), "expected", S4_COMMIT[n], f"{r['wallMs']:.0f} ms", flush=True)
        save()
    # positive control: B1-store invoke WITH x
    ctl = SAT[0]
    did = f"s7-ctl-b1-{ctl}-{STAMP}"
    r = peer_invoke("SubmitDecisionB1Store", b1_args(did, ctl))
    log["positiveControlB1"] = {"name": ctl, "decisionId": did, **r}
    print("POSITIVE CONTROL (B1Store with x)", r["ok"], r["class"], flush=True)
    save()
    # M4 negatives
    cats = {
        "N1_proof_substitution": lambda i, n, o: v7_args(f"s7-n1-{n}-{STAMP}", o, proof_from=n),
        "N2_tampered_proof": lambda i, n, o: v7_args(f"s7-n2-{n}-{STAMP}", n, tamper=True),
        "N3_forged_featureHash": lambda i, n, o: v7_args(f"s7-n3-{n}-{STAMP}", n, pub_feature_from=o),
        "N4_wrong_feature_proof": lambda i, n, o: v7_args(f"s7-n4-{n}-{STAMP}", n, feat_from=o),
        "N5_amount_over_threshold": lambda i, n, o: v7_args(f"s7-n5-{n}-{STAMP}", n, amount=OVER),
    }
    for cat, mk in cats.items():
        for i, n in enumerate(SAT):
            o = SAT[(i + 1) % len(SAT)]
            a = mk(i, n, o)
            r = peer_invoke("SubmitDecisionTrustlessV7", a)
            did = a[0]
            rec = read_decision(did) if r["ok"] else None
            log["negatives"].append({"category": cat, "name": n, "other": o, "decisionId": did, "ok": r["ok"], "returned": r["class"],
                                     "committed": (rec or {}).get("transferClass"), "tail": r["tail"]})
            print(cat, n, r["ok"], r["class"], (rec or {}).get("transferClass"), r["tail"][-90:] if not r["ok"] else "", flush=True)
            save()
    # N6 replay: same decisionID as the accepted M1 treatment transaction
    for n in SAT:
        did0 = f"s7-t-{n}-{STAMP}"
        before = read_decision(did0)
        r = peer_invoke("SubmitDecisionTrustlessV7", v7_args(did0, n))
        after = read_decision(did0)
        log["negatives"].append({"category": "N6_replay", "name": n, "decisionId": did0, "ok": r["ok"], "returned": r["class"],
                                 "committed": (after or {}).get("transferClass"), "recordUnchanged": before == after, "tail": r["tail"]})
        print("N6_replay", n, r["ok"], r["class"], "unchanged" if before == after else "CHANGED", r["tail"][-90:] if not r["ok"] else "", flush=True)
        save()
    # M5 contrasts: B0 with the N1 and N5 inputs
    for i, n in enumerate(SAT):
        o = SAT[(i + 1) % len(SAT)]
        for cat, a in (("B0_N1_proof_substitution", b0_args(f"s7-b0n1-{n}-{STAMP}", n, code=o)),
                       ("B0_N5_amount_over_threshold", b0_args(f"s7-b0n5-{n}-{STAMP}", n, amount=OVER))):
            r = peer_invoke("SubmitDecisionB0", a)
            rec = read_decision(a[0]) if r["ok"] else None
            log["b0"].append({"category": cat, "name": n, "decisionId": a[0], "ok": r["ok"], "returned": r["class"], "committed": (rec or {}).get("transferClass"), "tail": r["tail"]})
            print(cat, n, r["ok"], r["class"], (rec or {}).get("transferClass"), flush=True)
            save()
    log["c3"] = peer_query("SubmitDecisionB1Store", b1_args(f"s7-c3-{STAMP}", SAT[0], x=False))
    time.sleep(4)
    log["heightAfter"] = height()
    print("height after", log["heightAfter"], flush=True)
    save()


def blocks():
    log = json.load(open(os.path.join(HERE, "s7_run.json")))
    lo, hi = log["heightBefore"], log["heightAfter"]
    ctl_id = log["positiveControlB1"]["decisionId"]
    keys = {n: ",".join(map(str, fx[n]["x_q"][:16])).encode() for n in fx}
    wkey = ",".join(map(str, WB["W_q"][0][:8])).encode()
    ids = {d["decisionId"] for d in log["m1"]} | {d["decisionId"] for d in log["negatives"]} | {d["decisionId"] for d in log["b0"]}
    rows = []
    os.makedirs(os.path.join(HERE, "blocks_tmp"), exist_ok=True)
    for b in range(lo, hi):
        f = os.path.join(HERE, "blocks_tmp", f"b{b}.block")
        subprocess.run(["peer", "channel", "fetch", str(b), f, "-o", "localhost:17050", "--ordererTLSHostnameOverride", "orderer.example.com",
                        "-c", "authchannel", "--tls", "--cafile", ORD], env=env, capture_output=True, text=True)
        data = open(f, "rb").read() if os.path.exists(f) else b""
        rows.append({"block": b, "bytes": len(data), "hasControlId": ctl_id.encode() in data,
                     "hasTreatmentId": any(i.encode() in data for i in ids if i != ctl_id),
                     "xHits": sorted(n for n, k in keys.items() if k in data), "wHit": wkey in data})
        if b % 20 == 0:
            print("block", b, flush=True)
    json.dump({"range": [lo, hi], "rows": rows, "keyNote": "x_q[0:16] compact JSON per file; W_q[0][0:8]"}, open(os.path.join(HERE, "s7_blocks.json"), "w"), indent=1)
    ctl_rows = [r for r in rows if r["hasControlId"]]
    print("blocks fetched", len(rows), "control block(s)", [r["block"] for r in ctl_rows], "control xHits", [r["xHits"] for r in ctl_rows])
    treat = [r for r in rows if not r["hasControlId"]]
    print("treatment-block x hits:", sum(len(r["xHits"]) for r in treat), "W hits:", sum(r["wHit"] for r in treat))


def analyze():
    log = json.load(open(os.path.join(HERE, "s7_run.json")))
    blk = json.load(open(os.path.join(HERE, "s7_blocks.json"))) if os.path.exists(os.path.join(HERE, "s7_blocks.json")) else None
    out = {}
    m1 = log["m1"]
    out["M1"] = {"n": len(m1), "committed_match": sum(1 for r in m1 if r["ok"] and r["committed"] == r["expected"]),
                 "safe": sum(1 for r in m1 if r["committed"] == "SAFE"), "reject": sum(1 for r in m1 if r["committed"] == "REJECT"),
                 "median_wall_ms": sorted(r["wallMs"] for r in m1)[len(m1) // 2]}
    ab = log["argBytes"]
    out["M3"] = {"treatment_bytes_median": sorted(r["treatmentBytes"] for r in ab)[len(ab) // 2], "b1_form_bytes_median": sorted(r["b1StoreFormBytes"] for r in ab)[len(ab) // 2],
                 "x_bytes_median": sorted(r["xBytes"] for r in ab)[len(ab) // 2]}
    neg = {}
    for r in log["negatives"]:
        c = neg.setdefault(r["category"], {"n": 0, "accepted": 0, "errors": 0, "reject_or_reeval": 0})
        c["n"] += 1
        cls = r["committed"] or r["returned"]
        if r["category"] == "N6_replay":
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
    out["M4"] = neg
    b0 = {}
    for r in log["b0"]:
        c = b0.setdefault(r["category"], {"n": 0, "accepted": 0, "errors": 0})
        c["n"] += 1
        if not r["ok"]:
            c["errors"] += 1
        elif (r["committed"] or r["returned"]) in ACCEPT:
            c["accepted"] += 1
    out["M5_B0"] = b0
    out["C3_B1_without_x"] = log["c3"]
    out["positiveControlB1"] = {k: log["positiveControlB1"][k] for k in ("ok", "class")}
    if blk:
        ctl_rows = [r for r in blk["rows"] if r["hasControlId"]]
        treat = [r for r in blk["rows"] if not r["hasControlId"]]
        out["M2b"] = {"blocks": len(blk["rows"]), "control_blocks": [r["block"] for r in ctl_rows], "control_x_found": any(r["xHits"] for r in ctl_rows),
                      "treatment_blocks": len(treat), "treatment_x_hits": sum(len(r["xHits"]) for r in treat), "treatment_W_hits": sum(r["wHit"] for r in treat)}
    json.dump(out, open(os.path.join(HERE, "s7_summary.json"), "w"), indent=1)
    print(json.dumps(out, indent=1))


if __name__ == "__main__":
    {"setup": setup, "pilot": pilot, "run": run, "blocks": blocks, "analyze": analyze}[sys.argv[1]]()
