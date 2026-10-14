#!/usr/bin/env python3
"""S6: origin-timestamp (t0) composition across real on-chain hops on Fabric (chaincode v2.29 seq 41).
Protocol: evidence/s6_t0_multihop_protocol_2026-09-28.md (incl. amendment A1). Before every tick the clock authority (Org1 admin) sets the ledger clock to the
real wall clock, then every replica of both variants submits its hop at that ledger time. Outputs s6_results.json and a printed summary."""
import json, os, subprocess, sys, time

HERE = os.path.dirname(os.path.abspath(__file__))
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
DELTA = 120
FN = {"composed": "SubmitHopDecisionT0", "local": "SubmitHopDecisionT0Local"}
STAMP = str(int(time.time()))


def invoke(fn, args):
    cmd = ["peer", "chaincode", "invoke", "-o", "localhost:17050", "--ordererTLSHostnameOverride", "orderer.example.com", "--tls", "--cafile", ORD,
           "-C", "authchannel", "-n", "authcc", "--peerAddresses", "localhost:17051", "--tlsRootCertFiles", P1,
           "--peerAddresses", "localhost:19051", "--tlsRootCertFiles", P2, "--waitForEvent", "-c", json.dumps({"function": fn, "Args": args})]
    r = subprocess.run(cmd, env=env, capture_output=True, text=True)
    out = r.stdout + r.stderr
    if "Chaincode invoke successful" in out and 'payload:"' in out:
        return out.split('payload:"')[1].split('"')[0], None
    if "Chaincode invoke successful" in out:
        return "", None
    return None, out.strip().splitlines()[-1][:300] if out.strip() else "no output"


def set_clock():
    t = int(time.time())
    res, err = invoke("SetClock", [str(t)])
    if err:
        raise RuntimeError("SetClock failed: " + err)
    return t


def read(chain, mode, idx):
    for _ in range(10):
        r = subprocess.run(["peer", "chaincode", "query", "-C", "authchannel", "-n", "authcc", "-c",
                            json.dumps({"function": "ReadHopT0", "Args": [chain, mode, str(idx)]})], env=env, capture_output=True, text=True)
        try:
            return json.loads(r.stdout)
        except Exception:
            time.sleep(1)
    return None


def predict(records):
    """Model prediction from the ledger's own recorded ages: returns the class expected at each hop for the variant."""
    out, ok = [], True
    for r in records:
        fresh = (r["ageSinceOrigin"] <= DELTA) if r["mode"] == "composed" else (r["gapSincePrev"] <= DELTA)
        ok = ok and r["contextAtHop"] and fresh
        out.append("SAFE" if ok else "REJECT")
    return out


def run_paths(paths, gap):
    """paths: list of dicts {name, h, replicas, ctx_loss_hop}. One shared tick schedule; gap=None means back to back."""
    rows = []
    for p in paths:
        p["chains"] = {(v, r): f"s6-{p['name']}-{v}-{r}-{STAMP}" for v in FN for r in range(p["replicas"])}
        p["classes"] = {k: [] for k in p["chains"]}
        p["errors"] = []
    t_start = time.time()
    for k in range(max(p["h"] for p in paths)):
        if gap:
            time.sleep(max(0, t_start + k * gap - time.time()))
        set_clock()
        for p in paths:
            if k >= p["h"]:
                continue
            for (v, r), chain in p["chains"].items():
                ctx = not (p.get("ctx_loss_hop") == k)
                cls, err = invoke(FN[v], [chain, str(k), "true" if ctx else "false"])
                if err:
                    p["errors"].append((v, r, k, err))
                p["classes"][(v, r)].append(cls)
    for p in paths:
        for (v, r), chain in p["chains"].items():
            recs = [read(chain, v, i) for i in range(p["h"])]
            if any(x is None for x in recs):
                rows.append({"scenario": p["name"], "variant": v, "replica": r, "error": "missing hop records", "errors": p["errors"]})
                continue
            pred = predict(recs)
            got = p["classes"][(v, r)]
            origin_ok = all(x["originT0"] == recs[0]["originT0"] for x in recs)
            rows.append({"scenario": p["name"], "variant": v, "replica": r, "chain": chain, "h": p["h"], "gap_s": gap,
                         "classes": got, "model_prediction": pred, "matches_model": got == pred, "origin_t0_identical_across_hops": origin_ok,
                         "ages": [x["ageSinceOrigin"] for x in recs], "gaps": [x["gapSincePrev"] for x in recs], "end_to_end": got[-1] if got else None,
                         "errors": [e for e in p["errors"] if e[0] == v and e[1] == r]})
    return rows


def negative_controls():
    out = []
    for v in FN:
        chain = f"s6-neg-{v}-{STAMP}"
        set_clock()
        c0, e0 = invoke(FN[v], [chain, "0", "true"])
        skip, e_skip = invoke(FN[v], [chain, "2", "true"])
        replay, e_replay = invoke(FN[v], [chain, "0", "true"])
        out.append({"variant": v, "hop0": c0, "skip_hop_refused": e_skip is not None and skip is None, "skip_error": e_skip,
                    "replay_refused": e_replay is not None and replay is None, "replay_error": e_replay})
    return out


def main():
    which = sys.argv[1] if len(sys.argv) > 1 else "all"
    res = json.load(open(os.path.join(HERE, "s6_results.json"))) if os.path.exists(os.path.join(HERE, "s6_results.json")) else {}
    if which in ("fresh", "all"):
        rows = []
        for h in (1, 2, 3, 4, 5, 10):
            rows += run_paths([{"name": f"fresh-h{h}", "h": h, "replicas": 1 if h == 10 else 2}], None)
            print("fresh h", h, "done", flush=True)
        rows += run_paths([{"name": "ctxloss-h4", "h": 4, "replicas": 2, "ctx_loss_hop": 2}], None)
        res["fresh"] = rows
    if which in ("stale", "all"):
        res["stale"] = run_paths([{"name": f"stale-h{h}", "h": h, "replicas": 2} for h in (3, 4, 5, 10)], 50)
        print("stale done", flush=True)
    if which in ("neg", "all"):
        res["negative_controls"] = negative_controls()
    json.dump(res, open(os.path.join(HERE, "s6_results.json"), "w"), indent=1)
    for part in ("fresh", "stale"):
        for r in res.get(part, []):
            print(part, r.get("scenario"), r.get("variant"), r.get("replica"), "->", r.get("classes"), "model ok" if r.get("matches_model") else "MISMATCH/ERR",
                  "t0 same" if r.get("origin_t0_identical_across_hops") else "T0 DIFFERS", "ages", r.get("ages"))
    print(json.dumps(res.get("negative_controls"), indent=1))


if __name__ == "__main__":
    main()
