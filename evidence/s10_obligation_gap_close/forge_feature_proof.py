#!/usr/bin/env python3
"""
Constructs the ONE test this project has never run anywhere (S7/S8/S9/S10): a genuinely valid
Groth16 proof (passes its own pairing check, internally self-consistent) whose public featureHash
does NOT match what the pinned scanner authority committed on-chain for the claimed codeHash --
"attack A2" exactly as described in featurebind.go's own docstring, NOT the N2/N3-style "corrupt a
public signal and let the pairing fail" shortcut (addendum_n3_characterization_2026-10-04.md).

Method (no new ML inference, no circuit changes, reuses only already-computed, cached real
values -- the real pinned model weights are read from go_src/b1_model_weights.json, a leftover
copy from before the S9 packaging fix, NOT from re-running the ML scanner):
  - Target codeHash = fixture B ("04_call_to_unknown_safe.sol"), already registered on Besu with
    its OWN honest featureHash committed in FeatureCommitRegistry.
  - "Forged" feature vector = fixture A's ("03_dos_safe.sol") own real, strict-satisfiable x_q --
    a genuinely different, real feature vector, not a corrupted signal.
  - Build a witness binding fixture A's x to fixture B's codeHash (solidityCodeHash=ch_B,
    featureHash=featureHash_A, commitmentHash=Poseidon(ch_B, weightsHash, pred_A, SALT)) using the
    REAL pinned model weights (W_q/b_q) so every in-circuit constraint is genuinely satisfied
    (margin_A=66,113,634 > requiredMinMargin=54,445,307, so maxVulnClass=0 is honestly reachable).
  - Run the real circuit witness generator + `snarkjs groth16 prove` (circuits_v7) -- a real proof,
    not a modified one. It WILL pass Groth16 verification on its own terms (unlike N2/N3's
    corrupted-public-signal proofs), because every public signal is internally consistent with a
    real witness.
  - Submit it to SubmitDecisionTrustlessV7Obligation claiming codeHash=ch_B, with genuinely good
    obligation parameters (so the obligation chain cannot be the thing that rejects it). Expected:
    zkOK=true (pairing passes), src.Verified=true (ch_B's header/registry proofs are real), but
    featureOK=false via verifyFeatureBinding ("...forged-feature attack rejected"), because
    ch_B's REAL on-chain commitment is Poseidon(x_B), not Poseidon(x_A) -- the first real,
    on-chain exercise of that specific code branch in this project's history.

Usage: python3 forge_feature_proof.py build   (witness+proof, writes forged_proof.json/forged_public.json)
       python3 forge_feature_proof.py submit  (peer chaincode invoke against the live network)
"""
import json, os, subprocess, sys, time

HERE = os.path.dirname(os.path.abspath(__file__))
ROOT = "~/pqc-hardhat"
S4 = "~/paper_backups/submit_paper2/evidence/s4_fair_b1"
S8 = "~/paper_backups/submit_paper2/evidence/s8_population"
C7 = os.path.join(ROOT, "circuits_v7")
TMP = "/tmp/s10_forge"
os.makedirs(TMP, exist_ok=True)
SALT = 192837465192837465
TARGET = "04_call_to_unknown_safe.sol"   # fixture B: the codeHash we CLAIM
DONOR = "03_dos_safe.sol"                # fixture A: whose REAL x we actually use

sys.path.insert(0, os.path.join(ROOT, "gateway"))
os.environ.setdefault("ZKP_STUB_MODE", "false")
from zkp_module import poseidon_hash  # noqa: E402  (real circomlibjs Poseidon, self-tested on import)


def load_fixtures():
    d = json.load(open(os.path.join(S4, "fixtures.json")))
    files = {f["name"]: f for f in d["files"]}
    return d["weights_hash"], d["min_margin"], files


def build():
    wh_str, min_margin, files = load_fixtures()
    wh = int(wh_str)
    a, b = files[DONOR], files[TARGET]
    assert a["strict_circuit_satisfiable"] and b["strict_circuit_satisfiable"]
    assert a["feature_hash"] != b["feature_hash"], "donor/target must have genuinely different real featureHash"
    assert a["margin"] >= min_margin

    weights = json.load(open(os.path.join(S4, "go_src", "b1_model_weights.json")))
    assert str(weights["weights_hash"]) == wh_str
    W_q, b_q = weights["W_q"], weights["b_q"]

    ch_target = int(b["code_hash"])
    pred_a = a["pred"]
    commit = poseidon_hash(ch_target, wh, pred_a, SALT)

    inp = {
        "x": [str(v) for v in a["x_q"]],
        "W": [[str(v) for v in row] for row in W_q],
        "b": [str(v) for v in b_q],
        "salt": str(SALT),
        "solidityCodeHash": str(ch_target),
        "commitmentHash": str(commit),
        "maxVulnClass": "0",
        "minMargin": str(min_margin),
        "modelWeightsHash": wh_str,
        "featureHash": a["feature_hash"],
    }
    f_in, f_w = f"{TMP}/forge_in.json", f"{TMP}/forge_w.wtns"
    json.dump(inp, open(f_in, "w"))
    t0 = time.time()
    r = subprocess.run(["node", f"{C7}/ml_scan_proof_v7_js/generate_witness.js",
                        f"{C7}/ml_scan_proof_v7_js/ml_scan_proof_v7.wasm", f_in, f_w],
                       capture_output=True, text=True)
    print("witness:", r.returncode == 0, f"{time.time()-t0:.1f}s", r.stderr[-300:] if r.returncode else "")
    if r.returncode != 0:
        sys.exit(1)

    f_p, f_pub = os.path.join(HERE, "forged_proof.json"), os.path.join(HERE, "forged_public.json")
    t0 = time.time()
    r = subprocess.run(["snarkjs", "groth16", "prove", f"{C7}/ml_scan_proof_v7_final.zkey", f_w, f_p, f_pub],
                       capture_output=True, text=True)
    print("prove:", r.returncode == 0, f"{time.time()-t0:.1f}s", r.stderr[-300:] if r.returncode else "")
    if r.returncode != 0:
        sys.exit(1)

    ok = subprocess.run(["snarkjs", "groth16", "verify", f"{C7}/verification_key_v7.json", f_pub, f_p],
                        capture_output=True, text=True)
    print("snarkjs self-verify (should be OK -- this proof IS genuinely valid on its own terms):", ok.stdout.strip())

    proof = json.load(open(f_p))
    pub = json.load(open(f_pub))
    meta = {"donor": DONOR, "target": TARGET, "target_codeHash_dec": str(ch_target),
            "target_codeHash_hex": hex(ch_target), "donor_featureHash_used": a["feature_hash"],
            "target_real_featureHash_onchain_expected": b["feature_hash"],
            "proof": {k: proof[k] for k in ("pi_a", "pi_b", "pi_c")}, "public": pub,
            "snarkjs_self_verify_ok": "OK!" in ok.stdout}
    json.dump(meta, open(os.path.join(HERE, "forged_feature_meta.json"), "w"), indent=2)
    print(json.dumps({k: v for k, v in meta.items() if k not in ("proof",)}, indent=2))


def submit():
    meta = json.load(open(os.path.join(HERE, "forged_feature_meta.json")))
    st = json.load(open(os.path.join(S8, "besu_state_s8.json")))
    e = st["entries"][TARGET]
    did = f"s10-forged-feature-A2-{int(time.time())}"
    args = [did, "besu-mainnet", "manager", "100000", "1000000",
            meta["target_codeHash_hex"][2:].rjust(64, "0"),
            json.dumps(meta["proof"]), json.dumps(meta["public"]), json.dumps(st["header"]),
            json.dumps(e["mlProof"]), json.dumps(e["featProof"]),
            "audit", "audit", "true", "true"]
    TN = "~/authcc-network/fabric-samples/test-network"
    ORD = TN + "/organizations/ordererOrganizations/example.com/tlsca/tlsca.example.com-cert.pem"
    P1 = TN + "/organizations/peerOrganizations/org1.example.com/tlsca/tlsca.org1.example.com-cert.pem"
    P2 = TN + "/organizations/peerOrganizations/org2.example.com/tlsca/tlsca.org2.example.com-cert.pem"
    cmd = ["peer", "chaincode", "invoke",
           "-o", "localhost:17050", "--ordererTLSHostnameOverride", "orderer.example.com", "--tls",
           "--cafile", ORD, "-C", "authchannel", "-n", "authcc",
           "--peerAddresses", "localhost:17051", "--tlsRootCertFiles", P1,
           "--peerAddresses", "localhost:19051", "--tlsRootCertFiles", P2,
           "--waitForEvent", "-c", json.dumps({"function": "SubmitDecisionTrustlessV7Obligation", "Args": args})]
    env = dict(os.environ)
    env["PATH"] = TN + "/../bin:" + env["PATH"]
    env["FABRIC_CFG_PATH"] = TN + "/../config/"
    env["CORE_PEER_TLS_ENABLED"] = "true"
    env["CORE_PEER_LOCALMSPID"] = "Org1MSP"
    env["CORE_PEER_TLS_ROOTCERT_FILE"] = TN + "/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt"
    env["CORE_PEER_MSPCONFIGPATH"] = TN + "/organizations/peerOrganizations/org1.example.com/users/Admin@org1.example.com/msp"
    env["CORE_PEER_ADDRESS"] = "localhost:17051"
    r = subprocess.run(cmd, env=env, capture_output=True, text=True)
    print("invoke ok:", "Chaincode invoke successful" in (r.stdout + r.stderr))
    print((r.stdout + r.stderr)[-500:])
    rq = subprocess.run(["peer", "chaincode", "query", "-C", "authchannel", "-n", "authcc",
                         "-c", json.dumps({"function": "ReadDecision", "Args": [did]})],
                        env=env, capture_output=True, text=True)
    print("decisionId:", did)
    print(rq.stdout)
    json.dump({"decisionId": did, "args_summary": {"codeHash": meta["target_codeHash_hex"]}, "readDecision": rq.stdout},
              open(os.path.join(HERE, "forged_feature_result.json"), "w"), indent=2)


if __name__ == "__main__":
    {"build": build, "submit": submit}[sys.argv[1]]()
