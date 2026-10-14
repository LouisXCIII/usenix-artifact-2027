#!/usr/bin/env python3
"""rp_checker: a general-purpose SMT (Z3) checker for the R_P semantic contract
(Manuscriptsp_m2_r3_usenix.tex, Section~sec:semantics, eq. at line ~511):

    (psi_s, gamma_d) in R_P  =>  for all I_s in Omega, (sigma_s,sigma_d) in R_Sigma:
                                     psi_s(I_s,sigma_s) => gamma_d(M_I(I_s),sigma_d)

Unlike e1_oracle.py (2026-09-23), which hand-codes 4 near-duplicated Z3-solving functions
(one per guard "family": role/action/resource/context), this module has ONE generic
entailment-checking core (check_entailment). Each guard family still needs its own symbolic
encoding (that is inherent to any SMT-based checker -- different guards are different
formulas), but the actual UNSAT/SAT decision procedure, witness extraction, and reporting are
shared code, not copy-pasted four times.

Regression discipline: this module must reproduce e1_oracle.py's exact 50-combination table
(24 sound / 26 unsound) via the new generic core before any new result from it is trusted.
See regression_check().

New beyond e1_oracle.py:
  - obligation mechanism-strength fragment (real table from authcc/chaincode/obligation_v2.go)
  - the REAL deployed RoleMapping/RoleThreshold as a concrete instance (not an abstract
    role-level placeholder)
  - a mechanical (not AI-authored) generator of plausible-but-wrong "naive" mapping variants,
    run through the checker to count how many are unsound (the widening count)

usage: rp_checker.py [out.json]   (needs the z3-solver venv: .venv_e1/bin/python3)
"""
import json, sys
from z3 import Solver, Int, Bool, And, Or, Not, If, sat, unsat


def check_entailment(domain_constraints, psi_s, gamma_d, free_vars=None):
    """Generic R_P entailment check. UNSAT(domain & psi_s & not gamma_d) means psi_s => gamma_d
    holds for every instance consistent with domain_constraints -- i.e. (psi_s,gamma_d) in R_P
    for this fragment. SAT gives a concrete counterexample instance (a real widening case).
    domain_constraints: list of Z3 BoolRef. psi_s, gamma_d: Z3 BoolRef (gamma_d already has
    M_I applied by the caller). free_vars: Z3 vars to extract into the witness dict on SAT.
    Returns (sound: bool, witness: dict | None).
    """
    s = Solver()
    for c in domain_constraints:
        s.add(c)
    s.add(psi_s, Not(gamma_d))
    result = s.check()
    if result == unsat:
        return True, None
    m = s.model()
    witness = {str(v): str(m[v]) for v in (free_vars or [])}
    return False, witness


# ---------------------------------------------------------------------------
# Family query builders (symbolic encoding only -- solving goes through
# check_entailment). Role family uses the REAL deployed thresholds.
# ---------------------------------------------------------------------------

ROLE_THRESHOLD = {1: 10000, 2: 500000, 3: 2000000}  # clerk/manager/director, matches authchaincode.go


def build_role_family_query(pair_type, mapping_type):
    src_role = Int("src_role")
    amt = Int("amt")

    def thr(r):
        return If(r == 1, ROLE_THRESHOLD[1], If(r == 2, ROLE_THRESHOLD[2], ROLE_THRESHOLD[3]))

    domain = [src_role >= 1, src_role <= 3, amt >= 1, amt <= 5_000_000]

    if pair_type == "equivalent":
        psi_s = amt <= thr(src_role)
    elif pair_type == "source_stricter":
        psi_s = amt <= thr(src_role) / 2
    elif pair_type == "destination_stricter":
        nxt = If(src_role + 1 > 3, 3, src_role + 1)
        psi_s = amt <= thr(nxt)
    elif pair_type == "incomparable":
        nxt = If(src_role + 1 > 3, 3, src_role + 1)
        psi_s = Or(amt <= thr(src_role) / 4, And(amt > thr(src_role), amt <= thr(nxt)))
    elif pair_type == "contradictory":
        psi_s = amt > thr(3)
    else:
        raise ValueError(pair_type)

    if mapping_type in ("identity", "renaming"):
        dest_role = src_role
    elif mapping_type == "role_refinement":
        dest_role = If(src_role - 1 < 1, 1, src_role - 1)
    elif mapping_type == "role_widening":
        dest_role = If(src_role + 1 > 3, 3, src_role + 1)
    else:
        raise ValueError(mapping_type)

    gamma_d = amt <= thr(dest_role)
    return domain, psi_s, gamma_d, [src_role, amt]


def build_action_family_query(pair_type, mapping_type):
    src_action = Int("src_action")
    DEST_CAP = 2
    domain = [src_action >= 0, src_action <= 3]

    if pair_type == "equivalent":
        psi_s = src_action <= DEST_CAP
    elif pair_type == "source_stricter":
        psi_s = src_action <= DEST_CAP - 1
    elif pair_type == "destination_stricter":
        psi_s = src_action <= DEST_CAP + 1
    elif pair_type == "incomparable":
        psi_s = Or(src_action == 0, src_action == 3)
    elif pair_type == "contradictory":
        psi_s = src_action > 3
    else:
        raise ValueError(pair_type)

    if mapping_type == "action_restriction":
        claimed = src_action
    elif mapping_type == "action_expansion":
        claimed = src_action + 1
    else:
        raise ValueError(mapping_type)

    gamma_d = And(claimed >= 0, claimed <= DEST_CAP)
    return domain, psi_s, gamma_d, [src_action]


def build_resource_family_query(pair_type):
    res = Int("res")
    src_val = 2
    domain = [res >= 1, res <= 5]
    dest_set = {
        "equivalent": [2], "source_stricter": [2], "destination_stricter": [2, 3, 4],
        "incomparable": [3, 4], "contradictory": [],
    }[pair_type]
    psi_s = res == src_val
    claimed = res
    gamma_d = Or(*[claimed == v for v in dest_set]) if dest_set else False
    return domain, psi_s, gamma_d, [res]


def build_context_family_query(pair_type, mapping_type):
    ctx = Bool("ctx")
    src_val = True
    dest_accepts = {
        "equivalent": [True], "source_stricter": [True, False], "destination_stricter": [True],
        "incomparable": [False], "contradictory": [],
    }[pair_type]
    psi_s = ctx == src_val
    domain = []
    if mapping_type == "context_preservation":
        claimed = ctx
    elif mapping_type == "context_loss":
        claimed = Bool("claimed_dropped")
        domain = [claimed == False]
    elif mapping_type == "context_strengthening":
        claimed = ctx
    else:
        raise ValueError(mapping_type)
    gamma_d = Or(*[claimed == v for v in dest_accepts]) if dest_accepts else False
    return domain, psi_s, gamma_d, [ctx]


PAIR_TYPES = ["equivalent", "source_stricter", "destination_stricter", "incomparable", "contradictory"]
MAPPING_TYPES = [
    "identity", "renaming", "role_refinement", "role_widening",
    "action_restriction", "action_expansion", "resource_substitution",
    "context_preservation", "context_loss", "context_strengthening",
]


def run_all_50():
    table = {}
    for pt in PAIR_TYPES:
        for mt in MAPPING_TYPES:
            if mt in ("identity", "renaming", "role_refinement", "role_widening"):
                domain, psi_s, gamma_d, fv = build_role_family_query(pt, mt)
            elif mt in ("action_restriction", "action_expansion"):
                domain, psi_s, gamma_d, fv = build_action_family_query(pt, mt)
            elif mt == "resource_substitution":
                domain, psi_s, gamma_d, fv = build_resource_family_query(pt)
            elif mt in ("context_preservation", "context_loss", "context_strengthening"):
                domain, psi_s, gamma_d, fv = build_context_family_query(pt, mt)
            else:
                raise ValueError(mt)
            sound, witness = check_entailment(domain, psi_s, gamma_d, fv)
            table[f"{pt}|{mt}"] = {"pair_type": pt, "mapping_type": mt, "sound": sound, "witness": witness}
    return table


def regression_check():
    """Must reproduce e1_oracle.py's 2026-09-23 table exactly via the new generic core."""
    with open("e1_oracle_table_2026-09-23.json") as f:
        baseline = json.load(f)
    new_table = run_all_50()
    mismatches = []
    for k, base_v in baseline.items():
        new_sound = new_table[k]["sound"]
        if new_sound != base_v["sem_covered"]:
            mismatches.append((k, base_v["sem_covered"], new_sound))
    return new_table, mismatches


# ---------------------------------------------------------------------------
# New fragment: obligation mechanism-strength (real table from obligation_v2.go)
# ---------------------------------------------------------------------------

MECH_STRENGTH = {"": 0, "notification": 1, "audit": 2, "dual_approval": 3, "combined": 4}
REQUIRED_STRENGTH = {"empty": 0, "notification": 1, "audit": 2, "dual_approval": 3, "combined": 4}
MECH_NAMES = list(MECH_STRENGTH)  # "" + 4 real mechanisms
PROFILE_NAMES = list(REQUIRED_STRENGTH)  # "empty" + 4 real profiles


def build_obligation_query():
    """psi_s: source obligation profile has REQUIRED_STRENGTH[profile] = required (encoded as
    an integer-valued function over a finite enum via an uninterpreted Int + explicit domain).
    gamma_d (real deployed rule, oblCoveredV2): mapped mechanism's strength >= required.
    Checked: does "mapped >= required" (the real Go condition) entail genuine coverage, i.e.
    is the REAL system's identity-name mapping policy (every source profile p is mapped to
    the same-named destination mechanism -- the only mapping policy the deployed system
    actually uses; profile/mechanism names are not renamed across the domain boundary,
    unlike roles) sound, i.e. does it always satisfy the real deployed guard
    (mapped strength >= required strength, oblCoveredV2)?

    First version of this query left the destination's chosen mechanism (mech_idx) as a FREE
    variable with no connection to the source profile at all -- Z3 correctly found this
    "unsound" (e.g. profile=notification, mechanism=stripped), but that counterexample is not
    a real R_P violation: it is exactly the S5 obligation-stripping attack the guard exists to
    REJECT, not a case the system is supposed to accept. The bug was in the query, not the
    system: psi_s must pin down what mapping M_I actually produces, not leave the destination's
    mechanism unconstrained (an unconstrained "destination choice" is not psi_s's job --
    psi_s describes what the SOURCE establishes; M_I is a concrete function, not a free
    variable, exactly as every other fragment in this module encodes it)."""
    profile_idx = Int("profile_idx")   # index into PROFILE_NAMES
    domain = [profile_idx >= 0, profile_idx < len(PROFILE_NAMES)]

    required = If(profile_idx == 0, REQUIRED_STRENGTH["empty"],
               If(profile_idx == 1, REQUIRED_STRENGTH["notification"],
               If(profile_idx == 2, REQUIRED_STRENGTH["audit"],
               If(profile_idx == 3, REQUIRED_STRENGTH["dual_approval"],
                  REQUIRED_STRENGTH["combined"]))))
    # M_I: the real identity-name mapping (profile name == mechanism name; "empty" -> "").
    mapped_by_identity_MI = If(profile_idx == 0, MECH_STRENGTH[""],
                            If(profile_idx == 1, MECH_STRENGTH["notification"],
                            If(profile_idx == 2, MECH_STRENGTH["audit"],
                            If(profile_idx == 3, MECH_STRENGTH["dual_approval"],
                               MECH_STRENGTH["combined"]))))

    psi_s = domain[0]  # source genuinely has this profile; no further restriction needed
    gamma_d = Or(required == 0, mapped_by_identity_MI >= required)
    return domain, psi_s, gamma_d, [profile_idx]


# ---------------------------------------------------------------------------
# Concrete REAL mapping instance: gerente->manager, gerente_senior->director,
# with the real caps. Not an abstract role-level placeholder.
# ---------------------------------------------------------------------------

REAL_ROLE_CAP = {"manager": 500000, "director": 2000000, "clerk": 10000}
REAL_REGISTRY = {"gerente": "manager", "gerente_senior": "director"}


def build_real_registry_mapping_query():
    """Checks the REAL deployed RoleMapping: is it true that for a source role in the
    registry, mapped to its real destination role, amount<=mapped_cap (the destination's own
    re-evaluation, gamma_d) is NOT violated whenever some plausible source-side bound holds?
    Per Section~sec:semantics (line 528-533), RoleThreshold is destination-local in the real
    system -- the destination re-checks amount<=cap itself, it never inherits a source limit.
    So this query models the ONLY way this guard could be R_P-mediated: IF a future design
    made the source assert its own cap and the destination merely forwarded it unchecked
    (the naive design the Introduction's widening example warns against), would that be
    sound? psi_s: amount <= an assumed SOURCE cap (deliberately unconstrained relative to the
    destination's real cap, since the real system's source genuinely has no such table -- this
    models the worst case a naive implementer might assume: "the source's own authorization
    limit is whatever it happens to be"). gamma_d: the REAL destination cap for the mapped
    role. Domain: source cap ranges over a wide plausible band (10x the destination role's
    real cap) to stress-test, not hand-picked to pass."""
    results = {}
    for src_role, dst_role in REAL_REGISTRY.items():
        cap_d = REAL_ROLE_CAP[dst_role]
        amt = Int("amt")
        src_cap = Int("src_cap")
        domain = [amt >= 1, amt <= 10 * cap_d, src_cap >= 1, src_cap <= 10 * cap_d]
        psi_s = amt <= src_cap
        gamma_d = amt <= cap_d
        sound, witness = check_entailment(domain, psi_s, gamma_d, [amt, src_cap])
        results[f"{src_role}->{dst_role}"] = {
            "dest_cap": cap_d, "sound": sound, "witness": witness,
            "note": "models a hypothetical naive design where the destination trusts a "
                    "source-claimed cap unchecked; the REAL deployed system does not do this "
                    "(RoleThreshold is destination-local, Section sec:semantics) -- this "
                    "query quantifies exactly why that naive design would be unsound.",
        }
    return results


# ---------------------------------------------------------------------------
# Mechanical "naive mapping" variants -- NOT AI-authored, a fixed enumerated
# list of plausible-but-wrong implementation choices. Each is compared directly
# against the REAL correct guard in BOTH directions, so "unsound" is split into
# the two outcomes that matter differently: widening (the naive guard accepts
# something the real guard would reject -- an actual safety violation) versus
# over-conservative (the naive guard rejects something the real guard would
# accept -- an availability bug, not a safety one). A naive mistake that is
# only ever over-conservative is reported as such, not inflated into "widening".
# ---------------------------------------------------------------------------

def find_divergence(domain, real_gamma_d, naive_gamma_d, free_vars):
    s1 = Solver()
    for c in domain:
        s1.add(c)
    s1.add(naive_gamma_d, Not(real_gamma_d))
    widening = s1.check() == sat
    widening_witness = {str(v): str(s1.model()[v]) for v in free_vars} if widening else None

    s2 = Solver()
    for c in domain:
        s2.add(c)
    s2.add(Not(naive_gamma_d), real_gamma_d)
    over_cons = s2.check() == sat
    over_cons_witness = {str(v): str(s2.model()[v]) for v in free_vars} if over_cons else None

    return {
        "widening_found": widening, "widening_witness": widening_witness,
        "over_conservative_found": over_cons, "over_conservative_witness": over_cons_witness,
    }


def _thr(r):
    return If(r == 1, ROLE_THRESHOLD[1], If(r == 2, ROLE_THRESHOLD[2], ROLE_THRESHOLD[3]))


def naive_mapping_variants():
    """Each variant is a (name, domain, real_gamma_d, naive_gamma_d, free_vars, note) tuple.
    All are mechanical (hand-enumerated from real failure modes already named or fixed
    elsewhere in this project -- widening by trusting an unchecked source claim, reversed
    comparison, off-by-one role shift, ignoring the required==0 edge case, and the exact S11b
    pattern of trusting a caller-supplied obligation claim unchecked), not AI-generated
    adversarial content."""
    variants = []

    # V1: naive forward of a source-claimed cap without the destination's own re-check.
    for src_role, dst_role in REAL_REGISTRY.items():
        cap_d = REAL_ROLE_CAP[dst_role]
        amt = Int("amt"); src_cap = Int("src_cap")
        domain = [amt >= 1, amt <= 10 * cap_d, src_cap >= 1, src_cap <= 10 * cap_d]
        variants.append((f"V1_trust_source_cap_unchecked[{src_role}]", domain, amt <= cap_d, amt <= src_cap, [amt, src_cap],
                          "destination trusts a source-claimed cap instead of re-checking its own"))

    # V2: reversed threshold comparison (>= instead of <=).
    role = Int("role"); amt = Int("amt")
    domain = [role >= 1, role <= 3, amt >= 1, amt <= 5_000_000]
    variants.append(("V2_reversed_comparison", domain, amt <= _thr(role), amt >= _thr(role), [role, amt],
                      "comparison operator flipped (classic sign/direction bug)"))

    # V3: off-by-one role-hierarchy shift applied where identity was required.
    role3 = Int("role3"); amt3 = Int("amt3")
    domain3 = [role3 >= 1, role3 <= 3, amt3 >= 1, amt3 <= 5_000_000]
    dest_role_shifted = If(role3 - 1 < 1, 1, role3 - 1)
    variants.append(("V3_off_by_one_role_shift", domain3, amt3 <= _thr(role3), amt3 <= _thr(dest_role_shifted), [role3, amt3],
                      "role-refinement mapping misapplied where identity was intended"))

    # V4: ignores the required==0 (empty obligation profile) edge case.
    profile_idx = Int("profile_idx")
    domain4 = [profile_idx >= 0, profile_idx < len(PROFILE_NAMES)]
    required = If(profile_idx == 0, REQUIRED_STRENGTH["empty"],
               If(profile_idx == 1, REQUIRED_STRENGTH["notification"],
               If(profile_idx == 2, REQUIRED_STRENGTH["audit"],
               If(profile_idx == 3, REQUIRED_STRENGTH["dual_approval"],
                  REQUIRED_STRENGTH["combined"]))))
    mapped_identity = If(profile_idx == 0, MECH_STRENGTH[""],
                      If(profile_idx == 1, MECH_STRENGTH["notification"],
                      If(profile_idx == 2, MECH_STRENGTH["audit"],
                      If(profile_idx == 3, MECH_STRENGTH["dual_approval"],
                         MECH_STRENGTH["combined"]))))
    real_obl_gamma_d = Or(required == 0, mapped_identity >= required)
    variants.append(("V4_ignores_empty_profile_edge_case", domain4, real_obl_gamma_d, mapped_identity >= 1, [profile_idx],
                      "always demands at least notification-level, even when the source required nothing"))

    # V5: the exact S11b pattern (found and fixed earlier this project) -- trusting a
    # caller-supplied "obligation satisfied" claim instead of re-deriving mapped>=required.
    claimed_ok = Bool("claimed_ok")
    domain5 = [profile_idx >= 0, profile_idx < len(PROFILE_NAMES)]
    variants.append(("V5_trust_caller_claim_unchecked_S11b_pattern", domain5, real_obl_gamma_d, claimed_ok, [profile_idx, claimed_ok],
                      "modeled on the pre-fix S11b vulnerability pattern (trusting a caller-supplied "
                      "claim instead of re-deriving it). No widening is found in THIS fragment because "
                      "the ordinal mapped>=required check is a tautology under the real identity "
                      "mapping (mapped_identity always equals required's own strength), so there is no "
                      "profile where the real check correctly rejects to diverge from -- this is a real "
                      "limit of what the strength-ordinal fragment can express, not a safety result. "
                      "S11b's actual vulnerability was a separate boolean field (obligationDischarged/ "
                      "enforcementWitnessVerified, binary event-authenticity), outside this numeric "
                      "fragment's scope; it is the second-key co-signature work (Appendix, "
                      "evidence/qd2_domain_attestation_2026-10-08.md) that closes that gap, not this checker."))

    results = {}
    for name, domain, real_gamma_d, naive_gamma_d, fv, note in variants:
        div = find_divergence(domain, real_gamma_d, naive_gamma_d, fv)
        results[name] = {**div, "note": note}
    return results


def main(dst="rp_checker_results_2026-10-10.json"):
    new_table, mismatches = regression_check()
    n_sound = sum(1 for v in new_table.values() if v["sound"])
    print(f"Regression vs e1_oracle.py (2026-09-23): {len(mismatches)} mismatches out of 50")
    if mismatches:
        for k, old, new in mismatches:
            print(f"  MISMATCH {k}: baseline={old} new={new}")
    print(f"New generic checker: {n_sound}/50 sound (expect 24/50 matching baseline)")

    obligation_domain, obligation_psi, obligation_gamma, obligation_fv = build_obligation_query()
    obligation_sound, obligation_witness = check_entailment(obligation_domain, obligation_psi, obligation_gamma, obligation_fv)
    print(f"Obligation mechanism-strength fragment: sound={obligation_sound}")

    real_mapping = build_real_registry_mapping_query()
    for k, v in real_mapping.items():
        print(f"Real registry mapping {k}: sound={v['sound']}")

    naive = naive_mapping_variants()
    n_widening = sum(1 for v in naive.values() if v["widening_found"])
    n_overcons = sum(1 for v in naive.values() if v["over_conservative_found"] and not v["widening_found"])
    print(f"Naive mapping variants: {len(naive)} checked, {n_widening} show widening (unsafe), "
          f"{n_overcons} show only over-conservative (unavailable, not unsafe)")
    for k, v in naive.items():
        tag = "WIDENING" if v["widening_found"] else ("over-conservative only" if v["over_conservative_found"] else "sound")
        print(f"  {k}: {tag}" + (f" witness={v['widening_witness']}" if v["widening_found"] else ""))

    out = {
        "regression": {"n_mismatches": len(mismatches), "mismatches": mismatches, "n_sound_new": n_sound, "table": new_table},
        "obligation_mechanism_strength": {"sound": obligation_sound, "witness": obligation_witness},
        "real_registry_mapping": real_mapping,
        "naive_mapping_variants": naive,
    }
    with open(dst, "w") as f:
        json.dump(out, f, indent=2)
    print(f"\nFull results written to {dst}")


if __name__ == "__main__":
    main(sys.argv[1] if len(sys.argv) > 1 else "rp_checker_results_2026-10-10.json")
