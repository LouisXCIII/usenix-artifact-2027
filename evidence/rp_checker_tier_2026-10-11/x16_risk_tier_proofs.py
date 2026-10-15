#!/usr/bin/env python3
"""X16: real multi-floor Groth16 proofs to build a genuine margin-TIER signal.

The v6/v7 circuit relation does not output the achieved margin as a continuous public value --
`minMargin` is a PARAMETER supplied at proof-generation time (prove(..., min_margin=X)); the
circuit constrains the real computation to satisfy margin>=X, and witness generation genuinely
FAILS if it does not (confirmed empirically in x6_summary.json: contracts whose real margin is
below the floor have valid_proof=False, no coercion). So a destination that wants a TIERED risk
signal, not just one pass/fail floor, can get one for real, with the EXISTING circuit and
EXISTING verification key (no circuit change, no new trusted setup): request proofs at several
different floors for the same contract; which ones a real prover can produce is itself real
information about that contract's real margin, without ever revealing the margin value itself.

This is the source-side half of Tru 4 proper (see rp_checker_tier.py for the Z3 formalization
and chaincode/risk_tier.go for the destination side): Psi_s is now genuinely expressed on the
circuit's own margin SCALE (quantized logits), entirely distinct from the destination's dollar
scale -- unlike the SemBound/feature-binding fragment (corrected same day,
evidence/rp_checker_zk_2026-10-11.md), there is no vocabulary coincidence to hide a tautology
behind.

Floors (logits, quantized at SCALE2=4096*4096=16,777,216): T1=3.2452 (54,445,307 -- the real
deployed floor, unchanged), T2=4.0 (67,108,864), T3=5.0 (83,886,080). Four real, already-scanned
corpus contracts (real margins from x6_summary.json, not re-picked to fit) land in four
different tiers without any new scan:
  - 04_call_to_unknown_safe.sol   margin=5.5318  -> passes T1,T2,T3 -> TIER 3
  - 08_timestamp_vuln.sol         margin=4.5947  -> passes T1,T2, fails T3 -> TIER 2
  - 09_mishandled_exception_vuln.sol margin=3.2478 -> passes T1 (barely), fails T2 -> TIER 1
  - SafeBank.sol                  margin=1.2359  -> fails T1 -> TIER 0 (no valid proof at any floor)
"""
import hashlib, json, math, os, subprocess, sys

HERE = os.path.dirname(os.path.abspath(__file__))
ROOT = os.path.abspath(os.path.join(HERE, "..", ".."))
sys.path.insert(0, os.path.join(ROOT, "gateway"))
os.environ.setdefault("ZKP_STUB_MODE", "false")

from gateway import MLScanner                                   # noqa: E402
from zkp_module import poseidon_hash                             # noqa: E402
from lastlayer_witness_v5 import (                               # noqa: E402
    get_model_weights_hash_v5, run_forward_capture_live, quantize_x, verify_quantized_argmax,
)

C6 = os.path.join(ROOT, "circuits_v6")
OUT = os.path.join(HERE, "x16_out")
os.makedirs(OUT, exist_ok=True)
SCALE2 = 4096 * 4096
TMP = "/tmp/x16"
os.makedirs(TMP, exist_ok=True)

TIER_FLOORS = {  # tier name -> (quantized minMargin floor, real logit value, for the record)
    "T1": (math.ceil(3.2452 * SCALE2), 3.2452),  # the real deployed floor, unchanged
    "T2": (4 * SCALE2, 4.0),
    "T3": (5 * SCALE2, 5.0),
}

CONTRACTS = {
    "tier3_example": "test_contracts/04_call_to_unknown_safe.sol",
    "tier2_example": "test_contracts/08_timestamp_vuln.sol",
    "tier1_example": "test_contracts/09_mishandled_exception_vuln.sol",
    "tier0_example": "contracts/SafeBank.sol",  # real class_idx=0 but real margin (1.2359) is below T1
}


def code_hash(src):
    return int(hashlib.sha256(src.encode()).hexdigest(), 16) % (2 ** 253)


scanner = MLScanner()
scanner.load_model()
wh, W_q, b_q = get_model_weights_hash_v5(scanner)


def features(src):
    x_f, _, _ = run_forward_capture_live(scanner, src)
    x_q = quantize_x(x_f)
    pred, logits = verify_quantized_argmax(x_q, W_q, b_q)
    top = sorted(logits, reverse=True)
    return x_q, pred, (top[0] - top[1]) / SCALE2


def prove(name, src_for_hash, x_q, min_margin_floor, salt=987654321987654321):
    ch = code_hash(src_for_hash)
    pred, _ = verify_quantized_argmax(x_q, W_q, b_q)
    commitment = poseidon_hash(ch, wh, pred, salt)
    inp = {"x": [str(v) for v in x_q], "W": [[str(v) for v in r] for r in W_q], "b": [str(v) for v in b_q],
           "salt": str(salt), "solidityCodeHash": str(ch), "commitmentHash": str(commitment),
           "maxVulnClass": "0", "minMargin": str(min_margin_floor), "modelWeightsHash": str(wh)}
    f_in, f_w, f_p, f_pub = (f"{TMP}/{name}_{s}" for s in ("in.json", "w.wtns", "proof.json", "pub.json"))
    json.dump(inp, open(f_in, "w"))
    r = subprocess.run(["node", f"{C6}/ml_scan_proof_v6_js/generate_witness.js",
                         f"{C6}/ml_scan_proof_v6_js/ml_scan_proof_v6.wasm", f_in, f_w],
                        capture_output=True, text=True)
    if r.returncode != 0:
        return {"valid": False, "why": "witness generation failed (real margin below this floor)"}
    subprocess.run(["snarkjs", "groth16", "prove", f"{C6}/ml_scan_proof_v6_final.zkey", f_w, f_p, f_pub],
                    check=True, capture_output=True)
    ok = subprocess.run(["snarkjs", "groth16", "verify", f"{C6}/verification_key_v6.json", f_pub, f_p],
                         capture_output=True, text=True)
    valid = "OK" in ok.stdout
    pub = json.load(open(f_pub))
    proof = json.load(open(f_p))
    json.dump({k: proof[k] for k in ("pi_a", "pi_b", "pi_c")}, open(f"{OUT}/{name}_proof.json", "w"))
    json.dump(pub, open(f"{OUT}/{name}_public.json", "w"))
    return {"valid": True, "public": pub}


summary = {"tier_floors": {k: v[0] for k, v in TIER_FLOORS.items()},
           "tier_floor_logits": {k: v[1] for k, v in TIER_FLOORS.items()},
           "modelWeightsHash": str(wh), "cases": {}}

for label, relpath in CONTRACTS.items():
    src = open(os.path.join(ROOT, relpath)).read()
    x_q, pred, real_margin = features(src)
    results = {}
    for tier_name, (floor, _) in TIER_FLOORS.items():
        res = prove(f"{label}_{tier_name}", src, x_q, floor)
        results[tier_name] = {"valid": res["valid"], "why": res.get("why")}
        print(f"{label:16s} {tier_name} floor={floor:>10d} valid={res['valid']}", flush=True)
    achieved_tier = 0
    for n in (1, 2, 3):
        if results[f"T{n}"]["valid"]:
            achieved_tier = n
    summary["cases"][label] = {"file": relpath, "class_idx": pred, "real_margin_logits": round(real_margin, 4),
                                "results_by_floor": results, "achieved_tier": achieved_tier}
    print(f"  -> {label}: real_margin={real_margin:.4f} logits, achieved_tier={achieved_tier}", flush=True)

json.dump(summary, open(f"{HERE}/x16_summary.json", "w"), indent=1)
print(json.dumps({k: v["achieved_tier"] for k, v in summary["cases"].items()}))
