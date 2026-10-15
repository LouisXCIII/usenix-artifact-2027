#!/usr/bin/env python3
"""FN-rate-by-tier calibration (2026-10-11), responding to external review's "widening
is only self-reference" critique on the risk-tier R_P instance
(evidence/rp_checker_tier_2026-10-11.md).

Reuses, does not modify: s8_population/s8_prepare.py's exact pred_margin() method and
model weights (same quantized Wq/bq used throughout this project), and
s5/s5_corpus_features.json's existing x_q feature vectors for the 253-contract known-
vulnerable corpus (smartbugs-curated 142 + weakaccess 111) already used to produce the
"15/253 (5.9%)" figure quoted in the manuscript. No new scanning is run: margins are
recomputed offline from already-extracted features, exactly as s8_prepare.py does.

Real tier floors reused unchanged from chaincode/risk_tier.go / x16_risk_tier_proofs.py
(T1=54445422, T2=67108864, T3=83886080) and the real deployed tier->fraction table
(tierFractionBps: 1->1000bps, 2->5000bps, 3->10000bps).

Question answered: for the 253 contracts externally annotated as vulnerable, what
fraction clear each tier's margin floor (= false-negative rate AT that tier, since every
one of these 253 is a real vulnerable contract by construction)? Does a higher tier
floor actually correspond to a lower false-negative rate (the implicit assumption the
tier->fraction table relies on), and what expected-exposure fraction of a role's cap
(FN(t) x fraction(t)) does the real table imply at each tier?
"""
import json, os
from scipy.stats import beta

EV = "~/paper_backups/submit_paper2/evidence"
rows = json.load(open(os.path.join(EV, "s5", "s5_corpus_features.json")))["rows"]
W = json.load(open(os.path.join(EV, "s4_fair_b1", "go_src", "b1_model_weights.json")))
Wq, bq = W["W_q"], W["b_q"]


def pred_margin(x):
    lg = [sum(w * v for w, v in zip(row, x)) + b for row, b in zip(Wq, bq)]
    top = sorted(range(12), key=lambda i: -lg[i])
    return top[0], lg[top[0]] - lg[top[1]]


def cp(k, n, a=0.05):
    lo = 0.0 if k == 0 else beta.ppf(a / 2, k, n - k + 1)
    hi = 1.0 if k == n else beta.ppf(1 - a / 2, k + 1, n - k)
    return lo, hi


KNOWN_VULN = ("smartbugs", "weakaccess")
items = [r for r in rows if r["corpus"] in KNOWN_VULN]
assert len(items) == 253, len(items)

margins = []
for r in items:
    p, m = pred_margin(r["x_q"])
    margins.append(m)

# Real deployed tier floors (chaincode/risk_tier.go, unchanged).
TIER_FLOOR = {1: 54445422, 2: 67108864, 3: 83886080}
# Real deployed tier -> fraction-of-role-cap table (chaincode/risk_tier.go, unchanged).
TIER_FRACTION = {1: 0.10, 2: 0.50, 3: 1.00}

result = {
    "n_total_known_vulnerable": len(items),
    "corpora": list(KNOWN_VULN),
    "per_tier": {},
}
print(f"n = {len(items)} known-vulnerable contracts (smartbugs 142 + weakaccess 111)\n")
print(f"{'tier':>4} {'floor':>10} {'k clears floor':>15} {'FN rate':>9} {'CP95 lo':>9} {'CP95 hi':>9} {'fraction(t)':>12} {'FN(t)*fraction(t)':>18}")
prev_rate = None
monotone_ok = True
for t in (1, 2, 3):
    floor = TIER_FLOOR[t]
    k = sum(1 for m in margins if m >= floor)
    n = len(items)
    rate = k / n
    lo, hi = cp(k, n)
    frac = TIER_FRACTION[t]
    eef = rate * frac
    result["per_tier"][t] = {
        "floor": floor, "k": k, "n": n, "fn_rate": rate,
        "cp95_lo": lo, "cp95_hi": hi,
        "table_fraction": frac, "expected_exposure_fraction_of_cap": eef,
    }
    print(f"{t:>4} {floor:>10} {k:>15} {rate*100:>8.2f}% {lo*100:>8.2f}% {hi*100:>8.2f}% {frac*100:>11.1f}% {eef*100:>17.3f}%")
    if prev_rate is not None and rate > prev_rate:
        monotone_ok = False
    prev_rate = rate

result["fn_rate_monotonically_nonincreasing_in_tier"] = monotone_ok
eefs = [result["per_tier"][t]["expected_exposure_fraction_of_cap"] for t in (1, 2, 3)]
result["expected_exposure_monotonically_nonincreasing_in_tier"] = all(
    eefs[i] >= eefs[i + 1] for i in range(len(eefs) - 1)
)

print(f"\nFN rate monotonically non-increasing in tier (1->2->3): {monotone_ok}")
print(f"Expected-exposure-fraction-of-cap (FN(t)*fraction(t)) monotonically non-increasing in tier: {result['expected_exposure_monotonically_nonincreasing_in_tier']}")

# Compare the real table's expected-exposure-fraction-of-cap, at every tier, against the
# same quantity implied by the three real mapping-error naive variants (rp_checker_tier.py
# NV1/NV2/NV4) -- all computed from the SAME real, externally-annotated-corpus FN rates
# above. This is what gives "widening" real semantic content: higher expected exposure
# under a real, measured false-negative rate, not merely "differs from our own table".
fn = {t: result["per_tier"][t]["fn_rate"] for t in (1, 2, 3)}
real_table = {1: 0.10, 2: 0.50, 3: 1.00}
naive_tables = {
    "real (deployed)": real_table,
    "NV1_table_reversed": {1: 1.00, 2: 0.50, 3: 0.10},
    "NV2_tier2_tier3_swapped": {1: 0.10, 2: 1.00, 3: 0.50},
    # NV4: achieved tier looked up one band too high; tier 3 has no tier 4 to look up, so
    # the bug leaves it clamped at tier 3's own fraction (the realistic failure mode).
    "NV4_off_by_one": {1: real_table[2], 2: real_table[3], 3: real_table[3]},
}
print(f"\n{'table':24} {'EEF(1)%':>9} {'EEF(2)%':>9} {'EEF(3)%':>9} {'max EEF%':>9}")
exposure_comparison = {}
for name, tbl in naive_tables.items():
    eefs_t = {t: fn[t] * tbl[t] for t in (1, 2, 3)}
    exposure_comparison[name] = {
        "fraction_table": tbl,
        "expected_exposure_fraction_of_cap": eefs_t,
        "max_expected_exposure_fraction_of_cap": max(eefs_t.values()),
    }
    print(f"{name:24} {eefs_t[1]*100:9.3f} {eefs_t[2]*100:9.3f} {eefs_t[3]*100:9.3f} {max(eefs_t.values())*100:9.3f}")
result["exposure_comparison_real_vs_naive_variants"] = exposure_comparison
real_max = exposure_comparison["real (deployed)"]["max_expected_exposure_fraction_of_cap"]
result["real_table_has_lower_max_exposure_than_every_naive_variant"] = all(
    real_max < exposure_comparison[k]["max_expected_exposure_fraction_of_cap"]
    for k in naive_tables if k != "real (deployed)"
)
print(f"\nReal deployed table's max expected exposure is lower than every naive variant's: "
      f"{result['real_table_has_lower_max_exposure_than_every_naive_variant']}")

out_path = os.path.join(EV, "fn_rate_by_tier_2026-10-11.json")
json.dump(result, open(out_path, "w"), indent=2)
print(f"\nWritten: {out_path}")
