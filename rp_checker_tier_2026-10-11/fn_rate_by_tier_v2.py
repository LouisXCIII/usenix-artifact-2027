#!/usr/bin/env python3
"""FN-rate-by-tier calibration, v2 (2026-10-11), fixing a real bug in v1 and adding the
benign-corpus likelihood-ratio comparison, pi-sensitivity, and a disclosed-budget check
that two further rounds of external review asked for.

Bug fixed from v1 (disclosed in evidence/rp_checker_tier_2026-10-11.md): v1's
pred_margin() returns (top_predicted_class, margin), but v1's tier-membership test only
checked `margin >= floor`, NOT `top_predicted_class == 0` (the scanner's own SAFE
verdict). That let a contract whose top class was a VULNERABILITY class, confidently
predicted, count as "clearing the tier" -- inflating the apparent false-negative rate.
Fixed here to require BOTH conditions, exactly matching s8_prepare.py's own
`strict_sat = (pred==0 and margin>=MM)` definition and chaincode/risk_tier.go's real
on-chain tier logic. Sanity check: at the original single deployed floor, this script's
corrected count must equal the paper's already-published 15/253 figure exactly.

Still structurally limited, disclosed, not fixed in this pass: the known-vulnerable
corpus (smartbugs-curated, weakaccess) and the benign-proxy corpus (production contracts
from the S1 survey) are the SAME corpora already used elsewhere in this project; no
held-out split was performed (calibration and reporting use the same data), and at least
one of the two known-vulnerable corpora (WeakAccessMod) appears by name inside this
project's separate ML-training pipeline's dataset files, a real, unresolved risk that
these numbers are optimistic through training/evaluation overlap.
"""
import json, os
from scipy.stats import beta

EV = "/home/deployer/paper_backups/submit_paper2/evidence"
rows = json.load(open(os.path.join(EV, "s5", "s5_corpus_features.json")))["rows"]
W = json.load(open(os.path.join(EV, "s4_fair_b1", "go_src", "b1_model_weights.json")))
Wq, bq = W["W_q"], W["b_q"]
MM = json.load(open(os.path.join(EV, "s4_fair_b1", "fixtures.json")))["min_margin"]


def pred_margin(x):
    lg = [sum(w * v for w, v in zip(row, x)) + b for row, b in zip(Wq, bq)]
    top = sorted(range(12), key=lambda i: -lg[i])
    return top[0], lg[top[0]] - lg[top[1]]


def cp(k, n, a=0.05):
    lo = 0.0 if k == 0 else beta.ppf(a / 2, k, n - k + 1)
    hi = 1.0 if k == n else beta.ppf(1 - a / 2, k + 1, n - k)
    return lo, hi


TIER_FLOOR = {1: 54445422, 2: 67108864, 3: 83886080}
TIER_FRACTION = {1: 0.10, 2: 0.50, 3: 1.00}

vuln = [r for r in rows if r["corpus"] in ("smartbugs", "weakaccess")]
assert len(vuln) == 253, len(vuln)
benign = [r for r in rows if r["corpus"] == "deployed"]
assert len(benign) == 263, len(benign)

# Sanity check against the paper's already-published, independently-computed figure.
k_sanity = sum(1 for r in vuln if pred_margin(r["x_q"])[0] == 0 and pred_margin(r["x_q"])[1] >= MM)
assert k_sanity == 15, f"sanity check failed: {k_sanity}/253, expected 15/253"
print(f"Sanity check passed: pred==0 and margin>=MM({MM}) gives {k_sanity}/253 (matches paper's published 15/253).\n")


def tier_counts(items):
    out = {}
    for t, floor in TIER_FLOOR.items():
        k = sum(1 for r in items if pred_margin(r["x_q"])[0] == 0 and pred_margin(r["x_q"])[1] >= floor)
        out[t] = k
    return out


cum_vuln = tier_counts(vuln)   # P[achieved tier >= t | vulnerable]
cum_benign = tier_counts(benign)  # P[achieved tier >= t | benign-proxy]
n_v, n_b = len(vuln), len(benign)

result = {
    "n_vulnerable": n_v, "n_benign_proxy": n_b,
    "vulnerable_corpora": ["smartbugs", "weakaccess"],
    "benign_proxy_corpus": "deployed (S1 real production bridge/DeFi sources)",
    "sanity_check_15_of_253_reproduced": True,
    "cumulative": {}, "discrete": {}, "likelihood_ratio": {},
    "bayes_pi_sensitivity": {},
}

print(f"{'tier':>4} {'P[>=t|vuln]':>12} {'CP95':>18} {'P[>=t|benign]':>14} {'CP95':>18} {'LR(t)=P(v)/P(b)':>16}")
for t in (1, 2, 3):
    kv, kb = cum_vuln[t], cum_benign[t]
    pv, pb = kv / n_v, kb / n_b
    lv, hv = cp(kv, n_v)
    lb, hb = cp(kb, n_b)
    lr = pv / pb if pb > 0 else float("inf")
    result["cumulative"][t] = {
        "vuln": {"k": kv, "n": n_v, "rate": pv, "cp95": [lv, hv]},
        "benign": {"k": kb, "n": n_b, "rate": pb, "cp95": [lb, hb]},
        "likelihood_ratio": lr,
    }
    print(f"{t:>4} {pv*100:>11.2f}% [{lv*100:5.2f},{hv*100:5.2f}]% {pb*100:>13.2f}% [{lb*100:5.2f},{hb*100:5.2f}]% {lr:>16.3f}")

# Discrete (exactly-achieved) tier probabilities, floor at 0.
def discrete(cum, n):
    d = {}
    d[1] = cum[1] - cum[2]
    d[2] = cum[2] - cum[3]
    d[3] = cum[3]
    return d


disc_vuln = discrete(cum_vuln, n_v)
disc_benign = discrete(cum_benign, n_b)
print(f"\n{'tier':>4} {'P[=t|vuln]':>11} {'P[=t|benign]':>13}")
for t in (1, 2, 3):
    pv, pb = disc_vuln[t] / n_v, disc_benign[t] / n_b
    result["discrete"][t] = {"vuln_rate": pv, "benign_rate": pb}
    print(f"{t:>4} {pv*100:>10.2f}% {pb*100:>12.2f}%")

# Bayes: Pr[vuln | tier = t] for a sweep of priors pi (base rate of vulnerable in real
# traffic), using discrete per-tier rates.
print(f"\n{'tier':>4} " + " ".join(f"pi={p*100:g}%".rjust(10) for p in (0.01, 0.05, 0.20)))
for t in (1, 2, 3):
    pv, pb = disc_vuln[t] / n_v, disc_benign[t] / n_b
    row = {}
    vals = []
    for pi in (0.01, 0.05, 0.20):
        num = pi * pv
        den = pi * pv + (1 - pi) * pb
        post = num / den if den > 0 else float("nan")
        row[f"pi_{pi}"] = post
        vals.append(post)
    result["bayes_pi_sensitivity"][t] = row
    print(f"{t:>4} " + " ".join(f"{v*100:9.2f}%" for v in vals))

# Expected-exposure-fraction-of-cap at pi=5%, discrete tiers, real table vs naive variants.
PI_MID = 0.05
exposure_pi5 = {}
for t in (1, 2, 3):
    pv, pb = disc_vuln[t] / n_v, disc_benign[t] / n_b
    post = (PI_MID * pv) / (PI_MID * pv + (1 - PI_MID) * pb)
    exposure_pi5[t] = post * TIER_FRACTION[t]
result["expected_exposure_fraction_of_cap_pi5pct"] = exposure_pi5
print(f"\nExpected exposure (Bayes, pi=5%) fraction of cap: "
      + ", ".join(f"t{t}={exposure_pi5[t]*100:.3f}%" for t in (1, 2, 3)))
print(f"Worst tier: {max(exposure_pi5, key=exposure_pi5.get)}")

out_path = os.path.join(EV, "fn_rate_by_tier_v2_2026-10-11.json")
json.dump(result, open(out_path, "w"), indent=2)
print(f"\nWritten: {out_path}")
