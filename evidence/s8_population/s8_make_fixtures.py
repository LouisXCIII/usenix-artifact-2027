#!/usr/bin/env python3
"""S8 proof generation (protocol: evidence/s8_population_protocol_2026-09-28.md).
Usage: python3 s8_make_fixtures.py SHARD NSHARDS
For each contract of the fixed S8 sets: code hash = SHA-256(source) mod 2^253 (as in S4), feature digest = Poseidon of x_q (circomlibjs CLI), commitment = Poseidon(code hash,
weights hash, predicted class, salt); strictly satisfiable contracts get a REAL Groth16 v7 proof under the strict policy (maxVulnClass=0, minMargin=floor); the 60 sampled non-satisfiable
contracts get a real proof under the relaxed policy (maxVulnClass=11, minMargin=0), exactly as the S4 fixtures. x_q comes from the S5 scanner run (no scanner is run here).
Output: fixtures_s8_shard<SHARD>.json (fields as in S4 fixtures.json)."""
import hashlib, json, os, subprocess, sys, time

HERE = os.path.dirname(os.path.abspath(__file__))
EV = os.path.join(HERE, "..")
ROOT = "~/pqc-hardhat"
sys.path.insert(0, os.path.join(ROOT, "gateway"))
os.environ.setdefault("ZKP_STUB_MODE", "false")
from zkp_module import poseidon_hash  # noqa: E402

shard, nshards = int(sys.argv[1]), int(sys.argv[2])
C7 = os.path.join(ROOT, "circuits_v7")
TMP = f"/tmp/s8_fixtures_{shard}"
os.makedirs(TMP, exist_ok=True)
fx4 = json.load(open(os.path.join(EV, "s4_fair_b1", "fixtures.json")))
MIN_MARGIN, WH = fx4["min_margin"], int(fx4["weights_hash"])
W = json.load(open(os.path.join(EV, "s4_fair_b1", "go_src", "b1_model_weights.json")))
W_q, b_q = W["W_q"], W["b_q"]
SALT = 192837465192837465
FEATURE_DIGEST_CLI = os.path.join(ROOT, "gateway", "feature_digest_cli.js")
rows = json.load(open(os.path.join(EV, "s5", "s5_corpus_features.json")))["rows"]
sets = json.load(open(os.path.join(HERE, "s8_sets.json")))
todo = [(it, True) for it in sets["sat_new"]] + [(it, False) for it in sets["unsat_sample"]]
todo = [t for i, t in enumerate(todo) if i % nshards == shard]


def code_hash(src):
    return int(hashlib.sha256(src.encode()).hexdigest(), 16) % (2 ** 253)


def feature_hash(x_q):
    p = f"{TMP}/feat_in.json"
    json.dump({"x_q": x_q}, open(p, "w"))
    r = subprocess.run(["node", FEATURE_DIGEST_CLI, p], capture_output=True, text=True, timeout=60)
    if r.returncode != 0:
        raise RuntimeError(r.stderr)
    return int(r.stdout.strip())


def witness(name, inp):
    f_in, f_w = f"{TMP}/{name}_in.json", f"{TMP}/{name}_w.wtns"
    json.dump(inp, open(f_in, "w"))
    r = subprocess.run(["node", f"{C7}/ml_scan_proof_v7_js/generate_witness.js", f"{C7}/ml_scan_proof_v7_js/ml_scan_proof_v7.wasm", f_in, f_w], capture_output=True, text=True)
    return r.returncode == 0, f_w


out = []
for it, strict in todo:
    t0 = time.time()
    r = rows[int(it["id"][1:])]
    assert r["path"] == it["path"]
    src = open(r["path"]).read()
    x_q, pred = r["x_q"], it["pred"]
    fh, ch = feature_hash(x_q), code_hash(src)
    commit = poseidon_hash(ch, WH, pred, SALT)
    max_cls, min_margin = (0, MIN_MARGIN) if strict else (11, 0)
    inp = {"x": [str(v) for v in x_q], "W": [[str(v) for v in row] for row in W_q], "b": [str(v) for v in b_q], "salt": str(SALT), "solidityCodeHash": str(ch),
           "commitmentHash": str(commit), "maxVulnClass": str(max_cls), "minMargin": str(min_margin), "modelWeightsHash": str(WH), "featureHash": str(fh)}
    ok, f_w = witness(it["id"], inp)
    rec = {"name": it["id"], "path": it["path"], "corpus": it["corpus"], "x_q": x_q, "pred": pred, "margin": it["margin"], "strict_circuit_satisfiable": bool(strict),
           "witness_ok": ok, "feature_hash": str(fh), "code_hash": str(ch)}
    if ok:
        f_p, f_pub = f"{TMP}/{it['id']}_proof.json", f"{TMP}/{it['id']}_pub.json"
        subprocess.run(["snarkjs", "groth16", "prove", f"{C7}/ml_scan_proof_v7_final.zkey", f_w, f_p, f_pub], check=True, capture_output=True)
        v = subprocess.run(["snarkjs", "groth16", "verify", f"{C7}/verification_key_v7.json", f_pub, f_p], capture_output=True, text=True)
        proof, pub = json.load(open(f_p)), json.load(open(f_pub))
        pf = {k: proof[k] for k in ("pi_a", "pi_b", "pi_c")}
        rec["strict_proof" if strict else "proof"] = pf
        rec["strict_public" if strict else "public"] = pub
        rec["snarkjs_verify_ok"] = "OK" in v.stdout
    out.append(rec)
    print(it["id"], it["corpus"], "strict" if strict else "relaxed", "witness", ok, "snarkjs", rec.get("snarkjs_verify_ok"), f"{time.time() - t0:.1f}s", flush=True)
    json.dump({"shard": shard, "files": out}, open(os.path.join(HERE, f"fixtures_s8_shard{shard}.json"), "w"))
print("done", len(out))
