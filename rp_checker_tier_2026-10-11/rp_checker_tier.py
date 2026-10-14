#!/usr/bin/env python3
"""rp_checker_tier: Tru 4 proper -- the first genuinely non-trivial real R_P instance for this
project, with a real cross-vocabulary mapping M_I (not a tautology-under-identity like the
SemBound/feature-binding fragment, corrected 2026-10-11, evidence/rp_checker_zk_2026-10-11.md).

Why the SemBound fragment was not a real R_P instance (confirmed by two independent external
reviews, same day): its psi_s and gamma_d were stated over the IDENTICAL public-signal
vocabulary the circuit itself outputs, so M_I was the identity map and gamma_d===psi_s by
construction. What that mechanism actually tested was AUTHENTICITY (is psi_s genuinely true),
not a semantic RELATION between two independently-chosen predicate languages.

This module fixes that by construction. psi_s is now stated purely on the SOURCE's own margin
scale (a risk TIER, 0..3, derived from which of several real Groth16 proofs at different margin
floors a real prover could produce -- see x16_risk_tier_proofs.py; no circuit change, no new
trusted setup, real witness-generation failures below the achieved tier's floor). gamma_d is
stated purely on the DESTINATION's own dollar scale (RoleThreshold's real role caps). M_I is a
real governance table mapping tier -> fraction-of-cap-allowed, entirely independent of the
source's own vocabulary and entirely independent of the destination's own role/amount
vocabulary -- a genuine translation a governance body could get wrong, unlike an identity
pass-through. gamma_d is NOT definitionally equal to psi_s (gamma_d depends on role/amount,
which psi_s says nothing about at all); this is the structural test the reviews' critique
implies for genuine non-triviality.

usage: rp_checker_tier.py [out.json]   (needs the z3-solver venv: .venv_e1/bin/python3)
"""
import json, sys, os

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from rp_checker import find_divergence
from z3 import Int, And, Or, If

# Real deployed role caps (authchaincode.go RoleThreshold), unchanged.
REAL_ROLE_CAP = {"clerk": 10000, "manager": 500000, "director": 2000000}
ROLE_IDX = {1: "clerk", 2: "manager", 3: "director"}

# Real governance table, M_I: tier -> fraction-of-cap allowed (basis points, 0-10000).
# tier 0 = no valid proof at any floor (real margin below the base deployed floor) -- reject
# regardless of role/amount. Tiers 1-3 from x16_risk_tier_proofs.py's real floors
# (T1=3.2452 logits, the unchanged real deployed floor; T2=4.0; T3=5.0).
REAL_FRACTION_BPS = {0: 0, 1: 1000, 2: 5000, 3: 10000}  # 0%, 10%, 50%, 100%


def _cap(role_idx):
    return If(role_idx == 1, REAL_ROLE_CAP["clerk"],
           If(role_idx == 2, REAL_ROLE_CAP["manager"], REAL_ROLE_CAP["director"]))


def _fraction(tier, table):
    return If(tier == 0, table[0], If(tier == 1, table[1], If(tier == 2, table[2], table[3])))


def real_gamma_d(tier, role_idx, amount):
    """The real policy: amount <= cap(role) * fraction(tier) / 10000, tier>=1 required."""
    return And(tier >= 1, amount * 10000 <= _cap(role_idx) * _fraction(tier, REAL_FRACTION_BPS))


def domain_constraints(tier, role_idx, amount):
    return [tier >= 0, tier <= 3, role_idx >= 1, role_idx <= 3, amount >= 1, amount <= 20_000_000]


def naive_variants():
    """Mechanical, hand-enumerated naive M_I/implementation errors -- not AI-generated
    adversarial content, same discipline as every other naive-variant generator in this
    project. Each is a genuine cross-vocabulary mapping mistake, not a prover-authenticity
    mistake (that category is already covered, correctly labeled, in rp_checker_zk.py)."""
    variants = []

    # NV1: tier-fraction table reversed (governance transcription swapped the ordering --
    # higher confidence gets a SMALLER allowance instead of a larger one).
    tier, role_idx, amount = Int("tier"), Int("role_idx"), Int("amount")
    domain = domain_constraints(tier, role_idx, amount)
    reversed_table = {0: 0, 1: 10000, 2: 5000, 3: 1000}
    naive1 = And(tier >= 1, amount * 10000 <= _cap(role_idx) * _fraction(tier, reversed_table))
    variants.append(("NV1_tier_fraction_table_reversed", domain, real_gamma_d(tier, role_idx, amount),
                      naive1, [tier, role_idx, amount],
                      "governance table transcribed in reverse order (tier 3 gets 10%, tier 1 gets 100%)"))

    # NV2: adjacent-tier transcription swap (tiers 2 and 3 swapped -- a realistic single-entry
    # data-entry error in the governance table, not a full reversal).
    tier2, role_idx2, amount2 = Int("tier2"), Int("role_idx2"), Int("amount2")
    domain2 = domain_constraints(tier2, role_idx2, amount2)
    swapped_table = {0: 0, 1: 1000, 2: 10000, 3: 5000}
    naive2 = And(tier2 >= 1, amount2 * 10000 <= _cap(role_idx2) * _fraction(tier2, swapped_table))
    variants.append(("NV2_tier2_tier3_fraction_swapped", domain2, real_gamma_d(tier2, role_idx2, amount2),
                      naive2, [tier2, role_idx2, amount2],
                      "governance table has tiers 2 and 3's fractions swapped (50%<->100%)"))

    # NV3: trusts a caller-supplied "claimed tier" instead of the achieved tier the real
    # multi-floor proof ladder establishes -- the direct cross-vocabulary analogue of
    # rp_checker.py's V1 ("trust an unchecked source-claimed cap"), but now a genuine M_I
    # mistake (tier and dollars are different scales) rather than identity-trust.
    achieved_tier, claimed_tier, role_idx3, amount3 = Int("achieved_tier"), Int("claimed_tier"), Int("role_idx3"), Int("amount3")
    domain3 = [achieved_tier >= 0, achieved_tier <= 3, claimed_tier >= 0, claimed_tier <= 3,
               role_idx3 >= 1, role_idx3 <= 3, amount3 >= 1, amount3 <= 20_000_000]
    real3 = real_gamma_d(achieved_tier, role_idx3, amount3)
    naive3 = real_gamma_d(claimed_tier, role_idx3, amount3)  # uses claimed_tier, unconstrained vs achieved_tier
    variants.append(("NV3_trust_unchecked_caller_claimed_tier", domain3, real3, naive3,
                      [achieved_tier, claimed_tier, role_idx3, amount3],
                      "destination trusts a caller-declared risk tier instead of deriving it "
                      "from which real multi-floor proofs actually verified"))

    # NV4: off-by-one tier boundary (treats a proof at tier T as if it were tier T+1 -- e.g. a
    # porting bug that uses ">" where the real system uses ">=" when assigning achieved tier
    # from the highest floor that verified).
    tier4, role_idx4, amount4 = Int("tier4"), Int("role_idx4"), Int("amount4")
    domain4 = domain_constraints(tier4, role_idx4, amount4)
    shifted_tier = If(tier4 + 1 > 3, 3, tier4 + 1)
    naive4 = And(tier4 >= 1, amount4 * 10000 <= _cap(role_idx4) * _fraction(shifted_tier, REAL_FRACTION_BPS))
    variants.append(("NV4_tier_off_by_one_shift", domain4, real_gamma_d(tier4, role_idx4, amount4),
                      naive4, [tier4, role_idx4, amount4],
                      "achieved tier mis-indexed one band too high before the fraction lookup"))

    results = {}
    for name, dom, real_g, naive_g, fv, note in variants:
        results[name] = {**find_divergence(dom, real_g, naive_g, fv), "note": note}
    return results


def real_proof_tier_evidence():
    """Loads the real x16_risk_tier_proofs.py output (4 real contracts, real multi-floor
    Groth16 proofs, achieved tiers 3/2/1/0) as the empirical grounding for this fragment --
    not re-run here, reused."""
    path = os.path.join(os.path.dirname(os.path.abspath(__file__)), "x16_summary.json")
    if not os.path.exists(path):
        return {"available": False}
    with open(path) as f:
        d = json.load(f)
    return {"available": True, "source_file": "experiments/portability/x16_summary.json",
            "tier_floors_logits": d["tier_floor_logits"], "cases": d["cases"]}


def main(dst="rp_checker_tier_results_2026-10-11.json"):
    nv = naive_variants()
    widening = sum(1 for v in nv.values() if v["widening_found"])
    print(f"Naive variants (genuine M_I mapping errors): {widening}/{len(nv)} widening")
    for name, v in nv.items():
        print(f"  {name}: widening={v['widening_found']}" + (f" witness={v['widening_witness']}" if v["widening_found"] else ""))

    evidence = real_proof_tier_evidence()
    print(f"Real multi-floor proof evidence available: {evidence['available']}")
    if evidence["available"]:
        for label, c in evidence["cases"].items():
            print(f"  {label}: real_margin={c['real_margin_logits']} achieved_tier={c['achieved_tier']}")

    out = {
        "fragment": "Risk-tier (ZK margin ladder) -> role-cap-fraction mapping M_I, genuine two-vocabulary R_P instance",
        "real_fraction_table_bps": REAL_FRACTION_BPS,
        "naive_variants": nv,
        "widening_count": widening,
        "total_variants": len(nv),
        "real_proof_tier_evidence": evidence,
    }
    with open(dst, "w") as f:
        json.dump(out, f, indent=2)
    print(f"Wrote {dst}")
    return out


if __name__ == "__main__":
    main(sys.argv[1] if len(sys.argv) > 1 else "rp_checker_tier_results_2026-10-11.json")
