#!/usr/bin/env python3
"""S4 fixture generator (protocol: evidence/s4_fair_b1_protocol_2026-09-28.md).
For each of the 22 distinct real .sol files: run the REAL scanner up to classifier[8] to capture x, quantise it, compute the
Poseidon feature digest (circomlibjs), decide whether the STRICT relation (maxVulnClass=0, minMargin=MIN_MARGIN) is satisfiable by
running the circuit's own witness generator, and produce a REAL Groth16 v7 proof under a RELAXED policy (maxVulnClass=11, minMargin=0)
so that every file has a proof to time. Verification cost of Groth16 does not depend on the public values, so the relaxed proofs are
valid timing fixtures; correctness parity (B1 accepts iff the strict circuit is satisfiable) uses the strict result instead.
Output: fixtures.json in this directory."""
import glob, hashlib, json, math, os, subprocess, sys, time

HERE = os.path.dirname(os.path.abspath(__file__))
ROOT = "~/pqc-hardhat"
sys.path.insert(0, os.path.join(ROOT, "gateway"))
os.environ.setdefault("ZKP_STUB_MODE", "false")
from gateway import MLScanner                                    # noqa: E402
from zkp_module import poseidon_hash                              # noqa: E402
from lastlayer_witness_v5 import (                                 # noqa: E402
    get_model_weights_hash_v5, run_forward_capture_live, quantize_x, verify_quantized_argmax,
)

C7 = os.path.join(ROOT, "circuits_v7")
TMP = "/tmp/s4_fixtures"
os.makedirs(TMP, exist_ok=True)
SCALE2 = 4096 * 4096
MIN_MARGIN = math.ceil(math.log(0.7 * 11 / 0.3) * SCALE2)   # same derivation as zkverify.go requiredMinMargin
FEATURE_DIGEST_CLI = os.path.join(ROOT, "gateway", "feature_digest_cli.js")
SALT = 192837465192837465


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
    r = subprocess.run(["node", f"{C7}/ml_scan_proof_v7_js/generate_witness.js",
                        f"{C7}/ml_scan_proof_v7_js/ml_scan_proof_v7.wasm", f_in, f_w],
                       capture_output=True, text=True)
    return r.returncode == 0, f_w


def main():
    scanner = MLScanner()
    scanner.load_model()
    wh, W_q, b_q = get_model_weights_hash_v5(scanner)
    out = {"min_margin": MIN_MARGIN, "weights_hash": str(wh), "files": []}
    for path in sorted(glob.glob(f"{ROOT}/test_contracts/*.sol")):
        name = os.path.basename(path)
        src = open(path).read()
        t0 = time.time()
        x_f, _, _ = run_forward_capture_live(scanner, src)
        x_q = quantize_x(x_f)
        pred, logits = verify_quantized_argmax(x_q, W_q, b_q)
        top = sorted(logits, reverse=True)
        margin = top[0] - top[1]
        fh = feature_hash(x_q)
        ch = code_hash(src)
        commit = poseidon_hash(ch, wh, pred, SALT)

        def inputs(max_cls, min_margin):
            return {"x": [str(v) for v in x_q], "W": [[str(v) for v in r] for r in W_q], "b": [str(v) for v in b_q],
                    "salt": str(SALT), "solidityCodeHash": str(ch), "commitmentHash": str(commit),
                    "maxVulnClass": str(max_cls), "minMargin": str(min_margin), "modelWeightsHash": str(wh),
                    "featureHash": str(fh)}

        strict_ok, _ = witness(name + "_strict", inputs(0, MIN_MARGIN))
        relaxed_ok, f_w = witness(name + "_relaxed", inputs(11, 0))
        rec = {"name": name, "src_bytes": len(src.encode()), "x_q": x_q, "pred": pred, "margin": margin,
               "strict_circuit_satisfiable": strict_ok, "relaxed_witness_ok": relaxed_ok,
               "feature_hash": str(fh), "code_hash": str(ch)}
        if relaxed_ok:
            f_p, f_pub = f"{TMP}/{name}_proof.json", f"{TMP}/{name}_pub.json"
            subprocess.run(["snarkjs", "groth16", "prove", f"{C7}/ml_scan_proof_v7_final.zkey", f_w, f_p, f_pub],
                           check=True, capture_output=True)
            ok = subprocess.run(["snarkjs", "groth16", "verify", f"{C7}/verification_key_v7.json", f_pub, f_p],
                                capture_output=True, text=True)
            proof, pub = json.load(open(f_p)), json.load(open(f_pub))
            rec["proof"] = {k: proof[k] for k in ("pi_a", "pi_b", "pi_c")}
            rec["public"] = pub
            rec["snarkjs_verify_ok"] = "OK" in ok.stdout
        out["files"].append(rec)
        print(f"{name}: pred={pred} margin={margin / SCALE2:.3f} strict_sat={strict_ok} relaxed_ok={relaxed_ok} "
              f"snarkjs_ok={rec.get('snarkjs_verify_ok')} {time.time() - t0:.1f}s", flush=True)
    json.dump(out, open(os.path.join(HERE, "fixtures.json"), "w"))
    print("wrote fixtures.json", len(out["files"]))


if __name__ == "__main__":
    main()
