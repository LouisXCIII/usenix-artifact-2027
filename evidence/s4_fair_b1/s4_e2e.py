#!/usr/bin/env python3
"""S4 end-to-end harness on the real Fabric network (authcc v2.27 seq 39) with real Besu proofs.
  python3 s4_e2e.py correctness            one query per (file, variant); checks class parity
  python3 s4_e2e.py timing [REPS]          REPS in-chaincode endorsement timings per (file, variant), variant order rotated per rep
  python3 s4_e2e.py commit                 one real invoke (endorse + order + commit) per (file, variant); client wall time
Queries run the full entry point (including PutState into the simulated rwset) and do not commit, so no ledger growth;
invokes are the real submission path. Times inside the chaincode are time.Now() deltas (no CLI/gRPC noise); client wall times
include the same `peer` CLI start-up for every variant. Protocol: evidence/s4_fair_b1_protocol_2026-09-28.md (amendments A4+)."""
import json, os, subprocess, sys, time

HERE = os.path.dirname(os.path.abspath(__file__))
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

fx = {f["name"]: f for f in json.load(open(os.path.join(HERE, "fixtures.json")))["files"]}
state = json.load(open(os.path.join(HERE, "besu_state.json")))
WB = open("~/paper_backups/submit_paper2/evidence/s4_fair_b1/go_src/b1_model_weights.json").read()
VARIANTS = ["b4-groth16", "b1-store", "b1-nostore", "b1-sha"]
FN = {"b4-groth16": "SubmitDecisionTrustlessV7Timed", "b1-store": "SubmitDecisionB1StoreTimed",
      "b1-nostore": "SubmitDecisionB1NoStoreTimed", "b1-sha": "SubmitDecisionB1ShaTimed"}
# commit path: B4 is the deployed, unmodified SubmitDecisionTrustlessV7; the B1 entry points return only the class (deterministic), so both
# organizations' endorsements match. (The *Timed variants return timing JSON and can only be used with `query`.)
CFN = {"b4-groth16": "SubmitDecisionTrustlessV7", "b1-store": "SubmitDecisionB1Store",
       "b1-nostore": "SubmitDecisionB1NoStore", "b1-sha": "SubmitDecisionB1Sha"}
ROLE, AMOUNT, LIMIT = "manager", 100000, 1000000


def dumps(o):
    return json.dumps(o, separators=(",", ":"))


def args_for(variant, name, did):
    f, e = fx[name], state["entries"][name]
    header, ml = dumps(state["header"]), dumps(e["mlProof"])
    x = dumps(f["x_q"])
    base = [did, "besu-mainnet", ROLE, str(AMOUNT), str(LIMIT), e["codeHash"]]
    if variant == "b4-groth16":
        pf, pub = (f["strict_proof"], f["strict_public"]) if f["strict_circuit_satisfiable"] else (f["proof"], f["public"])
        return base + [dumps(pf), dumps(pub), header, ml, dumps(e["featProof"])]
    if variant == "b1-store":
        return base + [x, header, ml, dumps(e["featProof"])]
    if variant == "b1-nostore":
        return base + [x, dumps(json.loads(WB)), header, ml, dumps(e["featProof"])]
    return base + [x, header, ml, dumps(e["shaProof"])]


def query(variant, name, did):
    cmd = ["peer", "chaincode", "query", "-C", "authchannel", "-n", "authcc", "-c",
           json.dumps({"function": FN[variant], "Args": args_for(variant, name, did)})]
    t0 = time.perf_counter()
    r = subprocess.run(cmd, env=env, capture_output=True, text=True)
    wall = (time.perf_counter() - t0) * 1000
    try:
        return json.loads(r.stdout.strip()), wall
    except Exception:
        return {"error": (r.stdout + r.stderr).strip()[-300:]}, wall


def invoke(variant, name, did):
    cmd = ["peer", "chaincode", "invoke", "-o", "localhost:17050", "--ordererTLSHostnameOverride", "orderer.example.com",
           "--tls", "--cafile", ORD, "-C", "authchannel", "-n", "authcc",
           "--peerAddresses", "localhost:17051", "--tlsRootCertFiles", P1,
           "--peerAddresses", "localhost:19051", "--tlsRootCertFiles", P2,
           "--waitForEvent", "-c", json.dumps({"function": CFN[variant], "Args": args_for(variant, name, did)})]
    t0 = time.perf_counter()
    r = subprocess.run(cmd, env=env, capture_output=True, text=True)
    wall = (time.perf_counter() - t0) * 1000
    out = r.stdout + r.stderr
    ok = "Chaincode invoke successful" in out
    payload = None
    if 'payload:"' in out:
        payload = {"class": out.split('payload:"')[1].split('"')[0]}
    return ok, payload, wall, out.strip()[-200:]


def read_decision(did):
    for _ in range(10):
        r = subprocess.run(["peer", "chaincode", "query", "-C", "authchannel", "-n", "authcc", "-c",
                            json.dumps({"function": "ReadDecision", "Args": [did]})], env=env, capture_output=True, text=True)
        try:
            return json.loads(r.stdout)
        except Exception:
            time.sleep(1)
    return None


def main():
    mode = sys.argv[1]
    names = sorted(fx)
    if mode == "correctness":
        res = {}
        for n in names:
            res[n] = {}
            for v in VARIANTS:
                out, wall = query(v, n, f"s4-q-{v}-{n}")
                res[n][v] = {"class": out.get("class"), "verifyOk": out.get("verifyOk"), "detail": out.get("detail"), "error": out.get("error"),
                             "totalNs": out.get("totalNs")}
            cls = {v: res[n][v]["class"] for v in VARIANTS}
            print(n, cls, "strict_sat", fx[n]["strict_circuit_satisfiable"], flush=True)
        json.dump(res, open(os.path.join(HERE, "e2e_correctness.json"), "w"), indent=1)
        agree = sum(1 for n in names if len({res[n][v]["class"] for v in VARIANTS}) == 1 and res[n][VARIANTS[0]]["class"])
        print(f"class parity across the 4 variants: {agree}/{len(names)} files")
    elif mode == "timing":
        reps = int(sys.argv[2]) if len(sys.argv) > 2 else 30
        rows = []
        for n in names:
            per = {v: [] for v in VARIANTS}
            for i in range(reps):
                for k in range(len(VARIANTS)):
                    v = VARIANTS[(i + k) % len(VARIANTS)]
                    out, wall = query(v, n, f"s4-t-{v}-{n}-{i}")
                    if "error" in out:
                        print("ERR", n, v, out["error"], flush=True)
                        continue
                    per[v].append({k2: out[k2] for k2 in ("existsNs", "verifyNs", "ledgerNowNs", "sourceStateNs", "featureBindNs", "classifyNs", "putStateNs", "totalNs")} | {"class": out["class"], "wallMs": wall})
            rows.append({"name": n, "reps": reps, "samples": per})
            print(n, {v: (sorted(s["totalNs"] for s in per[v])[len(per[v]) // 2] / 1e6 if per[v] else None) for v in VARIANTS}, flush=True)
            json.dump({"reps": reps, "rows": rows, "clock_note": "ledger clock set to now+300s before header fetch"}, open(os.path.join(HERE, "e2e_timing.json"), "w"))
    elif mode == "commit":
        rows = []
        stamp = str(int(time.time()))
        for n in names:
            for v in VARIANTS:
                did = f"s4-c-{v}-{n}-{stamp}"
                ok, payload, wall, tail = invoke(v, n, did)
                rec = read_decision(did) if ok else None
                rows.append({"name": n, "variant": v, "ok": ok, "wallMs": wall, "class": (payload or {}).get("class"),
                             "committedClass": (rec or {}).get("transferClass"), "tail": tail if not ok else ""})
                print(n, v, ok, f"{wall:.0f} ms", rows[-1]["class"], rows[-1]["committedClass"], flush=True)
                json.dump(rows, open(os.path.join(HERE, "e2e_commit.json"), "w"), indent=1)


if __name__ == "__main__":
    main()
