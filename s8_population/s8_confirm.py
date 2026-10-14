#!/usr/bin/env python3
"""S8 Measure D: digest confirmation surface (protocol: evidence/s8_population_protocol_2026-09-28.md).
100 contracts (seed 20260928) x semantics-preserving edits T1 strip comments, T2 normalise whitespace, T3 rename the contract identifier, T4 append a comment + blank lines,
plus T0 = unedited sources as a determinism control. The REAL scanner is re-run on each variant; we compare the quantised feature vector x with the original x_q (identical x
means identical featureHash) and the SHA-256 source hash. Output: s8_confirm.json."""
import hashlib, json, os, random, re, sys

HERE = os.path.dirname(os.path.abspath(__file__))
EV = os.path.join(HERE, "..")
ROOT = "/home/deployer/pqc-hardhat"
sys.path.insert(0, os.path.join(ROOT, "gateway"))
os.environ.setdefault("ZKP_STUB_MODE", "false")
from gateway import MLScanner  # noqa: E402
from lastlayer_witness_v5 import run_forward_capture_live, quantize_x  # noqa: E402

rows = json.load(open(os.path.join(EV, "s5", "s5_corpus_features.json")))["rows"]
MM = json.load(open(os.path.join(EV, "s4_fair_b1", "fixtures.json")))["min_margin"]
_W = json.load(open(os.path.join(EV, "s4_fair_b1", "go_src", "b1_model_weights.json")))


def pred_margin(x):
    lg = [sum(w * v for w, v in zip(row, x)) + b for row, b in zip(_W["W_q"], _W["b_q"])]
    top = sorted(range(12), key=lambda i: -lg[i])
    return top[0], lg[top[0]] - lg[top[1]]


rng = random.Random(20260928)
sample = sorted(rng.sample(range(len(rows)), 100))


def strip_comments(src):
    out, i, n = [], 0, len(src)
    while i < n:
        c = src[i]
        if c in "\"'":
            q = c
            out.append(c); i += 1
            while i < n:
                out.append(src[i])
                if src[i] == "\\" and i + 1 < n:
                    out.append(src[i + 1]); i += 2; continue
                if src[i] == q:
                    i += 1; break
                i += 1
        elif src.startswith("//", i):
            while i < n and src[i] != "\n":
                i += 1
        elif src.startswith("/*", i):
            j = src.find("*/", i + 2)
            i = n if j < 0 else j + 2
        else:
            out.append(c); i += 1
    return "".join(out)


def norm_ws(src):
    lines = [re.sub(r"[ \t]+", " ", ln.strip()) for ln in src.splitlines()]
    return "\n".join(ln for ln in lines if ln)


def rename_contract(src):
    m = re.search(r"\bcontract\s+([A-Za-z_]\w*)", src)
    if not m:
        return src
    name = m.group(1)
    return re.sub(rf"\b{name}\b", name + "Renamed", src)


def append_comment(src):
    return src + "\n\n// appended comment\n\n\n"


TRANSFORMS = {"T0_identity": lambda s: s, "T1_strip_comments": strip_comments, "T2_normalise_whitespace": norm_ws,
              "T3_rename_contract": rename_contract, "T4_append_comment": append_comment}
scanner = MLScanner()
scanner.load_model()
res = {"seed": 20260928, "n": len(sample), "rows": []}
for k, idx in enumerate(sample):
    r = rows[idx]
    src = open(r["path"]).read()
    h0 = hashlib.sha256(src.encode()).hexdigest()
    rec = {"id": f"c{idx:03d}", "corpus": r["corpus"], "variants": {}}
    for tn, fn in TRANSFORMS.items():
        v = fn(src)
        try:
            x_f, _, _ = run_forward_capture_live(scanner, v)
            x_q = quantize_x(x_f)
            same = x_q == r["x_q"]
            diff = None if same else max(abs(a - b) for a, b in zip(x_q, r["x_q"]))
            nd = 0 if same else sum(1 for a, b in zip(x_q, r["x_q"]) if a != b)
            p2, m2 = pred_margin(x_q)
            rec["variants"][tn] = {"x_identical": same, "max_abs_diff": diff, "n_diff": nd, "code_hash_identical": hashlib.sha256(v.encode()).hexdigest() == h0, "changed_source": v != src,
                                   "pred": p2, "margin": m2, "strict_sat": bool(p2 == 0 and m2 >= MM)}   # amendment A1 (post hoc): strict-satisfiability flip
        except Exception as e:  # noqa: BLE001
            rec["variants"][tn] = {"error": str(e)[:200]}
    res["rows"].append(rec)
    if k % 10 == 0:
        print(k, rec["id"], {t: v.get("x_identical") for t, v in rec["variants"].items()}, flush=True)
        json.dump(res, open(os.path.join(HERE, "s8_confirm.json"), "w"))
json.dump(res, open(os.path.join(HERE, "s8_confirm.json"), "w"), indent=1)
print("done")
