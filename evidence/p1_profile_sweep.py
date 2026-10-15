#!/usr/bin/env python3
"""Real on-chain E5 PQ-migration profile sweep (P0-P3) against the Guide's own table, using the
ALREADY-DEPLOYED SubmitDecisionWithAssurance entry point (chaincode v2.42/seq54, no new code).
No new chaincode: claimsFor(decisionSigTier) already exists exactly as needed.
Each profile submits one real, mined Fabric transaction; results are read back via ReadDecision."""
import json
import os
import subprocess
import time

TN = "~/authcc-network/fabric-samples/test-network"
P1CERT = TN + "/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt"
P2CERT = TN + "/organizations/peerOrganizations/org2.example.com/peers/peer0.org2.example.com/tls/ca.crt"
ORD = TN + "/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem"

env = dict(os.environ)
env["PATH"] = TN + "/../bin:/usr/local/go/bin:" + env["PATH"]
env["FABRIC_CFG_PATH"] = TN + "/../config/"
env["CORE_PEER_TLS_ENABLED"] = "true"
env["CORE_PEER_LOCALMSPID"] = "Org1MSP"
env["CORE_PEER_TLS_ROOTCERT_FILE"] = P1CERT
env["CORE_PEER_MSPCONFIGPATH"] = TN + "/organizations/peerOrganizations/org1.example.com/users/Admin@org1.example.com/msp"
env["CORE_PEER_ADDRESS"] = "localhost:17051"

STAMP = str(int(time.time()))

# Guide table (ADP_r3_Experimental_Execution_Guide.docx, section 7):
#   P0: classical sig, classical transport, ClassicalDecisionAuth required -> accepted (positive control)
#   P1: classical sig, PQ/hybrid transport, PQDecisionAuth required -> REJECT (critical negative control:
#       "PQ-secure transport is not PQ-authenticated authorization")
#   P2: hybrid (classical+ML-DSA) evidence, PQ/hybrid transport, hybrid/dual required -> accepted
#   P3: pq sig, PQ transport, PQDecisionAuth required -> accepted (PQ positive control)
# P4 (classical sig, HYBRID transport, ClassicalDecisionAuth required -> accepted, "separates transport
# migration from policy requirement") is NOT run: SubmitDecisionWithAssurance has no transport-channel
# parameter at all (only decisionSigTier), so it cannot distinguish P4 from P0 -- disclosed, not faked.
PROFILES = [
    {"profile": "P0", "decisionSigTier": "classical", "destAssuranceRequirement": "ClassicalDecisionAuth",
     "expected": "EXACT", "note": "positive control"},
    {"profile": "P1", "decisionSigTier": "classical", "destAssuranceRequirement": "PQDecisionAuth",
     "expected": "REJECT", "note": "critical negative control (Guide 5.2): must not be Safe solely because channel is PQ"},
    {"profile": "P2", "decisionSigTier": "hybrid", "destAssuranceRequirement": "PQDecisionAuth",
     "expected": "EXACT", "note": "hybrid grants both claims"},
    {"profile": "P3", "decisionSigTier": "pq", "destAssuranceRequirement": "PQDecisionAuth",
     "expected": "EXACT", "note": "PQ positive control -- not previously run (2026-09-24 run only tested classical/hybrid)"},
]

ROLE = "manager"
AMOUNT = "200000"  # well under RoleThreshold["manager"]=500000
SOURCE_LIMIT = "200000"


def invoke(decision_id, role, amount, source_limit, sem_bound, ev_assurance_ok, sig_tier, dest_req):
    args = [decision_id, "src-domain-p1sweep", role, amount, source_limit,
            "true" if sem_bound else "false", "true" if ev_assurance_ok else "false",
            sig_tier, dest_req]
    cmd = ["peer", "chaincode", "invoke", "-o", "localhost:17050",
           "--ordererTLSHostnameOverride", "orderer.example.com", "--tls", "--cafile", ORD,
           "-C", "authchannel", "-n", "authcc",
           "--peerAddresses", "localhost:17051", "--tlsRootCertFiles", P1CERT,
           "--peerAddresses", "localhost:19051", "--tlsRootCertFiles", P2CERT,
           "--waitForEvent", "-c", json.dumps({"function": "SubmitDecisionWithAssurance", "Args": args})]
    r = subprocess.run(cmd, env=env, capture_output=True, text=True, timeout=60)
    out = r.stdout + r.stderr
    ok = "Chaincode invoke successful" in out
    payload = None
    if ok and 'payload:"' in out:
        payload = out.split('payload:"')[1].split('"')[0]
    return ok, payload, out, args


def read_decision(decision_id):
    cmd = ["peer", "chaincode", "query", "-C", "authchannel", "-n", "authcc",
           "-c", json.dumps({"function": "ReadDecision", "Args": [decision_id]})]
    r = subprocess.run(cmd, env=env, capture_output=True, text=True, timeout=30)
    try:
        return json.loads(r.stdout)
    except Exception:
        return {"raw": r.stdout, "err": r.stderr}


def main():
    results = []
    for p in PROFILES:
        decision_id = f"p1sweep-{p['profile']}-{STAMP}"
        ok, payload, raw, args = invoke(decision_id, ROLE, AMOUNT, SOURCE_LIMIT,
                                         True, True, p["decisionSigTier"], p["destAssuranceRequirement"])
        time.sleep(2)
        rec = read_decision(decision_id)
        got_class = rec.get("transferClass") if isinstance(rec, dict) else None
        results.append({
            "profile": p["profile"], "decisionId": decision_id,
            "decisionSigTier": p["decisionSigTier"], "destAssuranceRequirement": p["destAssuranceRequirement"],
            "note": p["note"], "invoke_ok": ok, "invoke_payload": payload,
            "expected_class": p["expected"], "got_class": got_class,
            "correct": got_class == p["expected"], "full_record": rec,
        })
        print(f"{p['profile']}: tier={p['decisionSigTier']} req={p['destAssuranceRequirement']} "
              f"expected={p['expected']} got={got_class} invoke_ok={ok}")
        if not ok:
            print("  INVOKE OUTPUT:", raw[:500])

    out_path = os.path.join(os.path.dirname(os.path.abspath(__file__)),
                             "p1_profile_sweep_result_2026-10-11.json")
    json.dump(results, open(out_path, "w"), indent=2)
    n_correct = sum(1 for r in results if r["correct"])
    print(f"\n{n_correct}/{len(results)} correct. Written to {out_path}")


if __name__ == "__main__":
    main()
