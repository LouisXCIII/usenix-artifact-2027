#!/usr/bin/env python3
"""S5-A oracle for the v4 full-head relation. For every circuit input (input_*_v4.json) it records
  (a) the logits of the independent Python fixed-point reference (gateway/extract_fullhead_witness.py: forward_head_quantized), and
  (b) the logits the CIRCUIT's own witness computes (main.lin8.out[j], read from the exported witness through the .sym file), when the
      witness can be generated (i.e. the circuit's constraints are satisfiable for that input).
The Go re-execution must agree with both. Output: s5_oracle_v4.json."""
import glob, json, os, re, subprocess, sys

sys.path.insert(0, "/home/deployer/pqc-hardhat/gateway")
from extract_fullhead_witness import forward_head_quantized  # noqa: E402

HERE = os.path.dirname(os.path.abspath(__file__))
C4 = "/home/deployer/pqc-hardhat/circuits_v4"
TMP = "/tmp/s5_v4_inputs"
P = 21888242871839275222246405745257275088548364400416034343698204186575808495617
os.makedirs(TMP, exist_ok=True)


def ints(v):
    return [int(x) for x in v]


def signed(v):
    return v if v < P // 2 else v - P


def main():
    sym = {}
    for line in open(f"{C4}/ml_scan_proof_v4.sym"):
        p = line.strip().split(",")
        if len(p) == 4 and p[3].startswith("main.curMax[") and int(p[1]) >= 0:   # lin8.out is optimised out of the witness; curMax[j] = max(logits[0..j])
            sym[int(re.search(r"\[(\d+)\]", p[3]).group(1))] = int(p[1])
    out = {}
    for path in sorted(glob.glob(f"{C4}/input_*.json")):
        name = os.path.basename(path).replace("input_", "").replace(".json", "")
        d = json.load(open(path))
        logits, _, _ = forward_head_quantized(
            ints(d["x0"]), [ints(r) for r in d["W0"]], ints(d["b0"]), ints(d["gamma1"]), ints(d["beta1"]),
            [ints(r) for r in d["W4"]], ints(d["b4"]), ints(d["gamma5"]), ints(d["beta5"]),
            [ints(r) for r in d["W8"]], ints(d["b8"]))
        py_pred = max(range(12), key=lambda j: logits[j])
        rec = {"file": path, "py_logits": logits, "py_pred": py_pred, "circuit_satisfiable": False, "circuit_running_max": None,
               "py_running_max": [max(logits[: j + 1]) for j in range(12)]}
        wtns, wj = f"{TMP}/{name}.wtns", f"{TMP}/{name}_wtns.json"
        r = subprocess.run(["node", f"{C4}/ml_scan_proof_v4_js/generate_witness.js", f"{C4}/ml_scan_proof_v4_js/ml_scan_proof_v4.wasm", path, wtns],
                           capture_output=True, text=True)
        if r.returncode == 0:
            subprocess.run(["snarkjs", "wtns", "export", "json", wtns, wj], check=True, capture_output=True)
            w = json.load(open(wj))
            rec["circuit_satisfiable"] = True
            rec["circuit_running_max"] = [signed(int(w[sym[j]])) for j in range(12)]
            os.remove(wj)
        out[name] = rec
        print(name, "py_pred", py_pred, "circuit satisfiable", rec["circuit_satisfiable"],
              "running max equal" if rec["circuit_running_max"] == rec["py_running_max"] else ("(no circuit witness)" if rec["circuit_running_max"] is None else "RUNNING-MAX MISMATCH"), flush=True)
    json.dump(out, open(os.path.join(HERE, "s5_oracle_v4.json"), "w"))


if __name__ == "__main__":
    main()
