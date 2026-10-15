#!/usr/bin/env python3
"""S5-B: prover cost against circuit size, on the REAL circuits built earlier in this project (protocol: evidence/s5_deep_dive_protocol_2026-09-28.md).
For each circuit: constraint count and proving-key size, witness-generation time, `snarkjs groth16 prove` wall time and peak resident memory (3 runs), and a
snarkjs verification of the produced proof. Inputs are the circuit's own `input_A_*.json` (v3/v4/v5) or the S4 fixture input (v7)."""
import json, os, re, subprocess, sys, time

ROOT = "/home/deployer/pqc-hardhat"
HERE = os.path.dirname(os.path.abspath(__file__))
TMP = "/tmp/s5_prover"
os.makedirs(TMP, exist_ok=True)
CIRCUITS = [
    ("v3_last_layer_light", "circuits_v3", "ml_scan_proof_v3", "input_A_v3.json"),
    ("v5_last_layer_weights_digest", "circuits_v5", "ml_scan_proof_v5", "input_A_v5.json"),
    ("v7_last_layer_weights_and_feature_digest", "circuits_v7", "ml_scan_proof_v7", "/tmp/s4_fixtures/03_dos_safe.sol_strict_in.json"),
    ("v4_full_head_3_linear_layers", "circuits_v4", "ml_scan_proof_v4", "input_A_v4.json"),
]


def run(cmd, timed=False):
    if timed:
        cmd = ["/usr/bin/time", "-f", "TIME %e %M"] + cmd
    t0 = time.time()
    r = subprocess.run(cmd, capture_output=True, text=True)
    el = time.time() - t0
    rss = None
    m = re.search(r"TIME ([\d.]+) (\d+)", r.stderr)
    if m:
        el, rss = float(m.group(1)), int(m.group(2))
    return r, el, rss


def main():
    only = sys.argv[1:] or None
    out_path = os.path.join(HERE, "s5_prover_scaling.json")
    res = json.load(open(out_path)) if os.path.exists(out_path) else {}
    for label, d, base, inp in CIRCUITS:
        if only and label.split("_")[0] not in only:
            continue
        cd = f"{ROOT}/{d}"
        inp = inp if inp.startswith("/") else f"{cd}/{inp}"
        r, _, _ = run(["snarkjs", "r1cs", "info", f"{cd}/{base}.r1cs"])
        m = re.search(r"# of Constraints: (\d+)", re.sub(r"\x1b\[[0-9;]*m", "", r.stdout + r.stderr))
        entry = {"circuit": label, "constraints": int(m.group(1)) if m else None,
                 "zkey_bytes": os.path.getsize(f"{cd}/{base}_final.zkey"), "witness_s": [], "prove_s": [], "prove_peak_rss_kb": []}
        wtns = f"{TMP}/{label}.wtns"
        for i in range(3):
            r, el, _ = run(["node", f"{cd}/{base}_js/generate_witness.js", f"{cd}/{base}_js/{base}.wasm", inp, wtns])
            if r.returncode != 0:
                entry["error"] = "witness generation failed: " + (r.stderr or r.stdout)[-200:]
                break
            entry["witness_s"].append(el)
            proof, pub = f"{TMP}/{label}_proof.json", f"{TMP}/{label}_pub.json"
            r, el, rss = run(["snarkjs", "groth16", "prove", f"{cd}/{base}_final.zkey", wtns, proof, pub], timed=True)
            if r.returncode != 0:
                entry["error"] = "prove failed: " + r.stderr[-200:]
                break
            entry["prove_s"].append(el)
            entry["prove_peak_rss_kb"].append(rss)
            print(label, f"run {i + 1}: witness {entry['witness_s'][-1]:.2f}s prove {el:.1f}s rss {rss / 1e6:.2f} GB", flush=True)
        if "error" not in entry:
            vk = [f for f in os.listdir(cd) if f.startswith("verification_key") and f.endswith(".json")]
            vkp = f"{cd}/verification_key_{base.replace('ml_scan_proof_', '')}.json"
            if not os.path.exists(vkp) and vk:
                vkp = f"{cd}/{vk[0]}"
            v, _, _ = run(["snarkjs", "groth16", "verify", vkp, pub, proof])
            entry["verify_ok"] = "OK" in v.stdout
            entry["public_signals"] = len(json.load(open(pub)))
        res[label] = entry
        json.dump(res, open(out_path, "w"), indent=1)
        print("->", {k: v for k, v in entry.items() if k not in ("witness_s",)}, flush=True)


if __name__ == "__main__":
    main()
