#!/usr/bin/env python3
"""S5-C: how much does the feature vector x reveal about the source contract beyond the verdict ADP already publishes?
Protocol: evidence/s5_deep_dive_protocol_2026-09-28.md (incl. amendment A1). Two stages:
  python3 s5_disclosure.py features    run the real scanner on every corpus contract, capture x, verdict and attributes -> s5_corpus_features.json
  python3 s5_disclosure.py analyse     grouped 5-fold CV probes (x vs verdict-only vs prior) -> s5_disclosure_results.json
Attributes are fixed in ATTRS below before any result was seen; none is dropped afterwards."""
import glob, hashlib, json, os, re, sys, random
import numpy as np

HERE = os.path.dirname(os.path.abspath(__file__))
ROOT = "~/pqc-hardhat"
EV = "~/paper_backups/submit_paper2/evidence"
SOURCES = [
    (f"{ROOT}/external_corpus/smartbugs-curated/dataset", "smartbugs", None),
    (f"{EV}/s1_verification/deployed_sources", "deployed", "dir"),
    (f"{EV}/s1_sources", "s1_sources", "dir"),
    (f"{EV}/e7_big_push_2026-09-24/new_contracts_source", "e7", None),
    ("~/Desktop/projectAI/buildVocab/SourceCodesCleaned", "weakaccess", None),
    (f"{ROOT}/test_contracts", "test22", None),
]


def strip_comments(s):
    s = re.sub(r"/\*.*?\*/", " ", s, flags=re.S)
    return re.sub(r"//[^\n]*", " ", s)


ATTRS = {
    "delegatecall": lambda c, raw: bool(re.search(r"\bdelegatecall\b", c)),
    "selfdestruct": lambda c, raw: bool(re.search(r"\b(selfdestruct|suicide)\b", c)),
    "tx_origin": lambda c, raw: "tx.origin" in c,
    "assembly": lambda c, raw: bool(re.search(r"\bassembly\b", c)),
    "low_level_call": lambda c, raw: bool(re.search(r"\.call\s*(\.value\s*\(|\{|\()", c)),
    "block_timestamp": lambda c, raw: bool(re.search(r"block\.timestamp|\bnow\b", c)),
    "payable": lambda c, raw: bool(re.search(r"\bpayable\b", c)),
    "loop": lambda c, raw: bool(re.search(r"\b(for|while)\s*\(", c)),
    "ecrecover": lambda c, raw: "ecrecover" in c,
    "erc20_like": lambda c, raw: bool(re.search(r"\bfunction\s+transfer\s*\(", c)) and bool(re.search(r"\bbalanceOf\b", c)),
    "inherits": lambda c, raw: bool(re.search(r"\bcontract\s+\w+\s+is\s+\w+", c)),
    "send_or_transfer": lambda c, raw: bool(re.search(r"\.(send|transfer)\s*\(", c)),
    "pragma_04": lambda c, raw: bool(re.search(r"pragma\s+solidity\s*[\^>=<~ ]*0\.4", raw)),
}
# size attributes are thresholded at the corpus median in analyse()


def gather():
    seen, out = {}, []
    for base, tag, grp in SOURCES:
        for p in sorted(glob.glob(base + "/**/*.sol", recursive=True)):
            try:
                raw = open(p, errors="ignore").read()
            except Exception:
                continue
            norm = re.sub(r"\s+", " ", strip_comments(raw)).strip()
            if len(norm) < 80:
                continue
            h = hashlib.sha256(norm.encode()).hexdigest()
            if h in seen:
                continue
            seen[h] = p
            rel = os.path.relpath(p, base)
            proj = f"{tag}:{rel.split('/')[0]}" if grp == "dir" else None
            out.append({"path": p, "corpus": tag, "project": proj, "raw": raw, "clean": strip_comments(raw), "norm": norm})
    return out


def features():
    os.environ.setdefault("ZKP_STUB_MODE", "false")
    sys.path.insert(0, f"{ROOT}/gateway")
    from gateway import MLScanner
    from lastlayer_witness_v5 import get_model_weights_hash_v5, run_forward_capture_live, quantize_x, verify_quantized_argmax
    sc = MLScanner()
    sc.load_model()
    _, W_q, b_q = get_model_weights_hash_v5(sc)
    items = gather()
    print("unique contracts:", len(items), flush=True)
    rows, skipped = [], 0
    for i, it in enumerate(items):
        try:
            x_f, _, _ = run_forward_capture_live(sc, it["raw"])
            x_q = quantize_x(x_f)
            pred, _ = verify_quantized_argmax(x_q, W_q, b_q)
        except Exception as e:
            skipped += 1
            continue
        c, raw = it["clean"], it["raw"]
        toks = re.findall(r"\w+|[^\s\w]", it["norm"])
        sh = sorted({hash(tuple(toks[j:j + 5])) & 0xFFFFFFFF for j in range(max(1, len(toks) - 4))})
        rows.append({"path": it["path"], "corpus": it["corpus"], "project": it["project"], "x_q": x_q, "pred": int(pred),
                     "attrs": {k: bool(f(c, raw)) for k, f in ATTRS.items()},
                     "lines": raw.count("\n") + 1,
                     "n_functions": len(re.findall(r"\bfunction\s+\w+\s*\([^)]*\)[^{;]*\b(public|external)\b", c)),
                     "shingles": sh})
        if i % 100 == 0:
            print(i, flush=True)
    json.dump({"skipped": skipped, "rows": rows}, open(os.path.join(HERE, "s5_corpus_features.json"), "w"))
    print("wrote", len(rows), "rows, skipped", skipped)


def analyse():
    from sklearn.linear_model import LogisticRegression
    from sklearn.metrics import roc_auc_score
    from sklearn.model_selection import GroupKFold
    from sklearn.preprocessing import StandardScaler
    d = json.load(open(os.path.join(HERE, "s5_corpus_features.json")))
    rows = d["rows"]
    sel = sys.argv[2] if len(sys.argv) > 2 else None   # amendment A3: restrict to one corpus (corpus is then constant)
    if sel:
        rows = [r for r in rows if r["corpus"] == sel]
    n = len(rows)
    X = np.array([r["x_q"] for r in rows], dtype=float)
    pred = np.array([r["pred"] for r in rows])
    V = np.eye(12)[pred]
    tags = sorted({r["corpus"] for r in rows})
    Cn = np.array([[1.0 if r["corpus"] == t else 0.0 for t in tags] for r in rows])  # corpus-of-origin one-hot (control, amendment A2)
    VC = np.hstack([V, Cn])
    # groups: near-duplicate components (Jaccard >= 0.6 on 5-token shingles), plus same S1 project
    sets = [set(r["shingles"]) for r in rows]
    parent = list(range(n))

    def find(a):
        while parent[a] != a:
            parent[a] = parent[parent[a]]
            a = parent[a]
        return a

    def union(a, b):
        parent[find(a)] = find(b)

    for i in range(n):
        for j in range(i + 1, n):
            a, b = sets[i], sets[j]
            if not a or not b:
                continue
            inter = len(a & b)
            if inter and inter / (len(a) + len(b) - inter) >= 0.6:
                union(i, j)
    byproj = {}
    for i, r in enumerate(rows):
        if r["project"]:
            byproj.setdefault(r["project"], []).append(i)
    for idx in byproj.values():
        for k in idx[1:]:
            union(idx[0], k)
    groups = np.array([find(i) for i in range(n)])
    sizes = np.bincount(np.unique(groups, return_inverse=True)[1])
    print(f"contracts {n}, groups {len(sizes)}, largest group {sizes.max()} ({100 * sizes.max() / n:.1f}%)")

    lines = np.array([r["lines"] for r in rows])
    nf = np.array([r["n_functions"] for r in rows])
    attrs = {k: np.array([r["attrs"][k] for r in rows], dtype=int) for k in ATTRS}
    attrs["lines_above_median"] = (lines > np.median(lines)).astype(int)
    attrs["public_functions_above_median"] = (nf > np.median(nf)).astype(int)

    gkf = GroupKFold(n_splits=5)
    splits = list(gkf.split(X, groups=groups))
    fold_sizes = [len(te) for _, te in splits]
    rng = np.random.default_rng(20260928)
    ug = np.unique(groups)
    res = {"n_contracts": n, "n_groups": int(len(ug)), "largest_group_fraction": float(sizes.max() / n), "fold_sizes": fold_sizes,
           "verdict_class_counts": {int(k): int((pred == k).sum()) for k in np.unique(pred)}, "attributes": {}}
    for name, y in attrs.items():
        npos, nneg = int(y.sum()), int((1 - y).sum())
        entry = {"n_pos": npos, "n_neg": nneg, "prevalence": npos / n}
        if npos < 10 or nneg < 10:
            entry["scored"] = False
            res["attributes"][name] = entry
            continue
        oof = {"x": np.full(n, np.nan), "verdict": np.full(n, np.nan), "corpus": np.full(n, np.nan), "vc": np.full(n, np.nan)}
        for tr, te in splits:
            if len(set(y[tr])) < 2:
                continue
            sc = StandardScaler().fit(X[tr])
            m = LogisticRegression(C=1.0, max_iter=2000).fit(sc.transform(X[tr]), y[tr])
            oof["x"][te] = m.predict_proba(sc.transform(X[te]))[:, 1]
            mv = LogisticRegression(C=1.0, max_iter=2000).fit(V[tr], y[tr])
            oof["verdict"][te] = mv.predict_proba(V[te])[:, 1]
            oof["corpus"][te] = LogisticRegression(C=1.0, max_iter=2000).fit(Cn[tr], y[tr]).predict_proba(Cn[te])[:, 1]
            oof["vc"][te] = LogisticRegression(C=1.0, max_iter=2000).fit(VC[tr], y[tr]).predict_proba(VC[te])[:, 1]
        ok = ~np.isnan(oof["x"])
        if len(set(y[ok])) < 2:
            entry["scored"] = False
            res["attributes"][name] = entry
            continue
        auc_x, auc_v = roc_auc_score(y[ok], oof["x"][ok]), roc_auc_score(y[ok], oof["verdict"][ok])
        auc_c, auc_vc = roc_auc_score(y[ok], oof["corpus"][ok]), roc_auc_score(y[ok], oof["vc"][ok])
        boots = []
        gid = {g: np.where((groups == g) & ok)[0] for g in ug}
        for _ in range(500):
            pick = rng.choice(ug, size=len(ug), replace=True)
            idx = np.concatenate([gid[g] for g in pick if len(gid[g])])
            if len(set(y[idx])) < 2:
                continue
            boots.append((roc_auc_score(y[idx], oof["x"][idx]), roc_auc_score(y[idx], oof["verdict"][idx]),
                          roc_auc_score(y[idx], oof["corpus"][idx]), roc_auc_score(y[idx], oof["vc"][idx])))
        b = np.array(boots)
        lo_x, hi_x = np.percentile(b[:, 0], [2.5, 97.5])
        lo_v, hi_v = np.percentile(b[:, 1], [2.5, 97.5])
        lo_d, hi_d = np.percentile(b[:, 0] - b[:, 1], [2.5, 97.5])
        cons = b[:, 0] - np.maximum(0.5, b[:, 3])   # conservative leakage: x versus the best of prior and verdict+corpus
        lo_k, hi_k = np.percentile(cons, [2.5, 97.5])
        lo_x5, hi_x5 = np.percentile(b[:, 0] - 0.5, [2.5, 97.5])
        entry.update({"scored": True, "auc_x": auc_x, "auc_x_ci95": [lo_x, hi_x], "auc_verdict": auc_v, "auc_verdict_ci95": [lo_v, hi_v],
                      "auc_prior": 0.5, "leakage_auc_x_minus_verdict": auc_x - auc_v, "leakage_ci95": [lo_d, hi_d], "n_boot": len(b),
                      "auc_corpus_only": auc_c, "auc_verdict_plus_corpus": auc_vc,
                      "leakage_conservative": auc_x - max(0.5, auc_vc), "leakage_conservative_ci95": [lo_k, hi_k],
                      "auc_x_minus_prior_ci95": [lo_x5, hi_x5]})
        res["attributes"][name] = entry
        print(f"{name:30s} prev {entry['prevalence']:.2f} npos {npos:3d} | x {auc_x:.3f} [{lo_x:.3f},{hi_x:.3f}] | verdict {auc_v:.3f} corpus {auc_c:.3f} verdict+corpus {auc_vc:.3f} | conservative leakage {auc_x - max(0.5, auc_vc):+.3f} [{lo_k:+.3f},{hi_k:+.3f}]")
    json.dump(res, open(os.path.join(HERE, "s5_disclosure_results.json" if not sel else f"s5_disclosure_results_within_{sel}.json"), "w"), indent=1)


if __name__ == "__main__":
    {"features": features, "analyse": analyse}[sys.argv[1]]()
