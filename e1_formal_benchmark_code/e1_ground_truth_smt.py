#!/usr/bin/env python3
"""E1 limitation (1) fix: an independent ground-truth labeler for the SAME formal definition
(eq:portable-final / eq:five-class-classifier) as e1_ground_truth.py and e1_classifier.py, but
evaluated through Z3's constraint solver instead of Python if/elif branching.

Mathematically this decides the same thing as e1_ground_truth.py for any concrete case (there is
only one correct answer to "does this case satisfy eq:portable-final"), so agreement between this
module and e1_classifier.py is still not proof of an independently-discovered fact the way the
class-level Z3 audit in e1_oracle.py is. What it DOES add: a genuinely different execution engine
(Z3's SAT/SMT solver, not the Python interpreter evaluating nested if/elif) reduces the chance
that a control-flow bug present in e1_classifier.py's hand-written branching (e.g. wrong operator
precedence, an inverted condition, a misplaced early return) is silently reproduced by a human
re-typing the same formula by hand into e1_ground_truth.py -- the two are no longer "the same code
written twice," they are two different computational paradigms applied to the same specification.

For each case, every input (auth_src_ok, sem_covered_instance, temporal_state, etc.) is a fixed,
concrete value -- there is no free variable to search over, so Z3 is used here as a constraint
CHECKER (build the exact boolean formula for each candidate class from eq:five-class-classifier,
ask Z3 whether it is satisfiable under the case's fixed values) rather than as a solver discovering
something new. This is disclosed explicitly, matching E1's own "why FAR=FRR=0 is expected, not
vacuous" framing.

Called by e1_run_campaign_smt.py. Not imported anywhere else.
"""
from z3 import Solver, Bool, BoolVal, And, Or, Not, sat


def _claims_z3(decision_sig_tier):
    # Returns (has_classical, has_pq) as Python bools -- the claim SET is finite/small so this is
    # still a direct lookup, not something requiring symbolic search; the Z3 formulas below use it.
    if decision_sig_tier == "classical":
        return True, False
    if decision_sig_tier == "hybrid":
        return True, True
    if decision_sig_tier == "pq":
        return False, True
    raise ValueError(decision_sig_tier)


def ground_truth_class_smt(case):
    s = Solver()

    auth_src = BoolVal(bool(case["auth_src_ok"]))
    gov_general = BoolVal(bool(case["gov_ok_general"]))
    mapping_valid = BoolVal(bool(case["mapping_table_valid"]))
    state_rel = BoolVal(bool(case["state_rel_ok"]))
    sem_covered = BoolVal(bool(case["sem_covered_instance"]))
    obl_covered = BoolVal(bool(case["obligation_discharged"]))
    replay_ok = BoolVal(bool(case["replay_ok"]))
    toctou_ok = BoolVal(bool(case["toctou_ok"]))

    temporal = case["temporal_state"]
    finality = case["finality_state"]
    fresh = BoolVal(temporal == "valid")
    final_ok = BoolVal(finality in ("at", "above"))
    revoked = BoolVal(temporal == "revoked")
    stale = BoolVal(temporal == "stale")
    finality_below = BoolVal(finality == "below")
    contradictory = BoolVal(case["pair_type"] == "contradictory")

    has_classical, has_pq = _claims_z3(case["decision_sig_tier"])
    req = case["dest_assurance_requirement"]
    assure_ok_val = (req == "ClassicalDecisionAuth" and has_classical) or (req == "PQDecisionAuth" and has_pq)
    assure_ok = BoolVal(assure_ok_val)

    equivalent_identity = BoolVal(
        case["pair_type"] == "equivalent" and case["mapping_type"] in ("identity", "renaming")
    )
    destination_local_authority = BoolVal(bool(case["destination_local_authority"]))

    fatal_formula = Or(Not(auth_src), Not(gov_general), Not(mapping_valid), revoked, contradictory)

    def sat_under(extra):
        s.push()
        s.add(extra)
        r = s.check()
        s.pop()
        return r == sat

    if sat_under(fatal_formula):
        return "REJECT"

    portable_formula = And(
        auth_src, gov_general, mapping_valid, state_rel, sem_covered,
        obl_covered, fresh, final_ok, assure_ok, replay_ok, toctou_ok,
    )
    if sat_under(portable_formula):
        if sat_under(equivalent_identity):
            return "EXACT"
        return "SAFE"

    repairable = Or(Not(obl_covered), stale, finality_below)
    non_repairable_failure = Or(
        Not(sem_covered),
        And(Not(fresh), Not(stale)),
        And(Not(final_ok), Not(finality_below)),
        Not(assure_ok), Not(replay_ok), Not(toctou_ok), Not(state_rel),
    )
    satisfiable_delta = And(repairable, Not(non_repairable_failure))
    if sat_under(satisfiable_delta):
        return "CONDITIONAL"

    if sat_under(destination_local_authority):
        return "REEVALUATE"

    return "REJECT"
