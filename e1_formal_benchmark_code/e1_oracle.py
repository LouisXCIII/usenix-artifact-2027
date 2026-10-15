#!/usr/bin/env python3
"""E1: independent SMT oracle for the ADP formal benchmark (Manuscriptsp_m2_r1.tex,
Section "Generated Formal Benchmark and Ground Truth" / eq:five-class-classifier).

For each of the 50 (pair_type x mapping_type) combinations used by the generator
(e1_generator.py), this module uses Z3 to decide, independently of the classifier
under test (e1_classifier.py), whether the semantic-coverage relation SemCovered
(the predicate-refinement relation R_P of eq:predicate-relation / eq:semantic-coverage)
holds for ALL concrete instances consistent with that combination's declared source/
destination predicate classes -- i.e. it checks UNSAT of (source predicate AND NOT
mapped destination guard). SAT means a counterexample instance exists (SemCovered=0
for that combination); UNSAT means the mapping/pair-type combination is behaviorally
sound (SemCovered=1).

This deliberately does NOT reuse the classifier's own (table-lookup / direct-evaluation)
logic, matching the paper's ground-truth admission criterion: (i) construction
template, (ii) direct destination evaluation, (iii) independent SMT agreement.
"""
import json
from z3 import Solver, Int, Bool, And, Or, Not, If, sat, unsat

PAIR_TYPES = ["equivalent", "source_stricter", "destination_stricter", "incomparable", "contradictory"]
MAPPING_TYPES = [
    "identity", "renaming", "role_refinement", "role_widening",
    "action_restriction", "action_expansion",
    "resource_substitution",
    "context_preservation", "context_loss", "context_strengthening",
]

ROLE_THRESHOLD = {1: 10000, 2: 500000, 3: 2000000}  # clerk/manager/director -- matches authchaincode.go RoleThreshold


def check_role_family(pair_type, mapping_type):
    """Stressed field = (source role level 1..3, amount). Returns (sem_covered, note)."""
    src_role = Int("src_role")
    amt = Int("amt")

    def thr(r):
        return If(r == 1, ROLE_THRESHOLD[1], If(r == 2, ROLE_THRESHOLD[2], ROLE_THRESHOLD[3]))

    role_domain = And(src_role >= 1, src_role <= 3)

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

    s = Solver()
    s.add(role_domain, amt >= 1, amt <= 5_000_000)
    s.add(psi_s, Not(gamma_d))
    result = s.check()
    if result == unsat:
        return True, "UNSAT(counterexample search) -> sound for all instances"
    m = s.model()
    return False, f"SAT counterexample: src_role={m[src_role]}, amt={m[amt]}"


def check_action_family(pair_type, mapping_type):
    """Stressed field = action_level (0=view,1=discharge,2=approve,3=transfer). DEST_CAP=2 (approve)."""
    src_action = Int("src_action")
    DEST_CAP = 2

    if pair_type == "equivalent":
        psi_s = src_action <= DEST_CAP
    elif pair_type == "source_stricter":
        psi_s = src_action <= DEST_CAP - 1
    elif pair_type == "destination_stricter":
        psi_s = src_action <= DEST_CAP + 1
    elif pair_type == "incomparable":
        psi_s = Or(src_action == 0, src_action == 3)
    elif pair_type == "contradictory":
        psi_s = src_action > 3  # unsatisfiable vs any real action level -> impossible source
    else:
        raise ValueError(pair_type)

    if mapping_type == "action_restriction":
        claimed = src_action
    elif mapping_type == "action_expansion":
        claimed = src_action + 1
    else:
        raise ValueError(mapping_type)

    gamma_d = And(claimed >= 0, claimed <= DEST_CAP)

    s = Solver()
    s.add(src_action >= 0, src_action <= 3)
    s.add(psi_s, Not(gamma_d))
    result = s.check()
    if result == unsat:
        return True, "UNSAT -> sound"
    m = s.model()
    return False, f"SAT counterexample: src_action={m[src_action]}"


def check_resource_family(pair_type):
    """Stressed field = resource_id in {1..5}. Source always asserts resource==2 (fixed decision).
    Destination-accepted set varies by pair_type; mapping_type=resource_substitution forwards
    the id via an accepted-equivalence table (honest forward of the literal value)."""
    res = Int("res")
    src_val = 2
    dest_set = {
        "equivalent": [2],
        "source_stricter": [2],
        "destination_stricter": [2, 3, 4],
        "incomparable": [3, 4],
        "contradictory": [],
    }[pair_type]

    psi_s = res == src_val
    claimed = res
    gamma_d = Or(*[claimed == v for v in dest_set]) if dest_set else False

    s = Solver()
    s.add(res >= 1, res <= 5)
    s.add(psi_s, Not(gamma_d))
    result = s.check()
    if result == unsat:
        return True, "UNSAT -> sound"
    return False, "SAT counterexample: resource id not in destination-accepted equivalence set"


def check_context_family(pair_type, mapping_type):
    """Stressed field = context flag (bool). Source asserts context==True (its actual, true value)."""
    ctx = Bool("ctx")
    src_val = True
    dest_accepts = {
        "equivalent": [True],
        "source_stricter": [True, False],
        "destination_stricter": [True],
        "incomparable": [False],
        "contradictory": [],
    }[pair_type]

    psi_s = ctx == src_val
    if mapping_type == "context_preservation":
        claimed = ctx
    elif mapping_type == "context_loss":
        claimed = Bool("claimed_dropped")
    elif mapping_type == "context_strengthening":
        claimed = ctx
    else:
        raise ValueError(mapping_type)

    s = Solver()
    if mapping_type == "context_loss":
        s.add(claimed == False)
    gamma_d = Or(*[claimed == v for v in dest_accepts]) if dest_accepts else False
    s.add(psi_s, Not(gamma_d))
    result = s.check()
    if result == unsat:
        return True, "UNSAT -> sound"
    return False, "SAT counterexample: context flag lost/mismatched relative to destination requirement"


def build_oracle_table():
    table = {}
    for pt in PAIR_TYPES:
        for mt in MAPPING_TYPES:
            if mt in ("identity", "renaming", "role_refinement", "role_widening"):
                sound, note = check_role_family(pt, mt)
            elif mt in ("action_restriction", "action_expansion"):
                sound, note = check_action_family(pt, mt)
            elif mt == "resource_substitution":
                sound, note = check_resource_family(pt)
            elif mt in ("context_preservation", "context_loss", "context_strengthening"):
                sound, note = check_context_family(pt, mt)
            else:
                raise ValueError(mt)
            table[f"{pt}|{mt}"] = {"pair_type": pt, "mapping_type": mt, "sem_covered": sound, "z3_note": note}
    return table


if __name__ == "__main__":
    table = build_oracle_table()
    with open("e1_oracle_table_2026-09-23.json", "w") as f:
        json.dump(table, f, indent=2)
    n_sound = sum(1 for v in table.values() if v["sem_covered"])
    print(f"Computed {len(table)} (pair_type x mapping_type) combinations via Z3; {n_sound} sound, {len(table) - n_sound} unsound.")
    for k, v in table.items():
        print(f"  {k:45s} sem_covered={v['sem_covered']}")
