#!/usr/bin/env python3
"""E1: independent ground-truth labeler for the cases produced by e1_generator.py.

Computes the TRUE five-class label per case by direct evaluation of the same
deterministic destination guards already recorded on the case (sem_covered_instance,
mapping_table_valid -- ground truth for a deterministic function is evaluating it
correctly; admission criterion (ii) in the paper's methodology). This module is
intentionally a separate code path from e1_classifier.py (the system under test):
it is called once by e1_run_campaign.py to produce the reference label that
e1_classifier.py's prediction is scored against, and is never given the
classifier's diagnostics.

Called by e1_run_campaign.py. Not imported anywhere else.
"""


def _claims(decision_sig_tier):
    if decision_sig_tier == "classical":
        return {"ClassicalDecisionAuth"}
    if decision_sig_tier == "hybrid":
        return {"ClassicalDecisionAuth", "PQDecisionAuth"}
    if decision_sig_tier == "pq":
        return {"PQDecisionAuth"}
    raise ValueError(decision_sig_tier)


def ground_truth_class(case):
    auth_src = bool(case["auth_src_ok"])
    gov_general = bool(case["gov_ok_general"])
    mapping_valid = bool(case["mapping_table_valid"])
    state_rel = bool(case["state_rel_ok"])
    temporal = case["temporal_state"]
    finality = case["finality_state"]
    replay_ok = bool(case["replay_ok"])
    toctou_ok = bool(case["toctou_ok"])
    obligation_discharged = bool(case["obligation_discharged"])
    sem_covered = bool(case["sem_covered_instance"])

    fatal = (
        (not auth_src)
        or (not gov_general)
        or (not mapping_valid)
        or (temporal == "revoked")
        or (case["pair_type"] == "contradictory")
    )
    if fatal:
        return "REJECT"

    obl_covered = obligation_discharged
    fresh = temporal == "valid"
    final_ok = finality in ("at", "above")
    assure_ok = case["dest_assurance_requirement"] in _claims(case["decision_sig_tier"])

    portable = (
        auth_src and gov_general and mapping_valid and state_rel
        and sem_covered and obl_covered and fresh and final_ok and assure_ok
        and replay_ok and toctou_ok
    )
    if portable:
        birefine = (
            case["pair_type"] == "equivalent"
            and case["mapping_type"] in ("identity", "renaming")
        )
        return "EXACT" if birefine else "SAFE"

    repairable_gaps = set()
    if not obl_covered:
        repairable_gaps.add("obligation")
    if temporal == "stale":
        repairable_gaps.add("stale")
    if finality == "below":
        repairable_gaps.add("finality_below")

    non_repairable_failure = (
        (not sem_covered)
        or (not fresh and temporal != "stale")
        or (not final_ok and finality != "below")
        or (not assure_ok)
        or (not replay_ok)
        or (not toctou_ok)
        or (not state_rel)
    )
    if repairable_gaps and not non_repairable_failure:
        return "CONDITIONAL"

    if bool(case["destination_local_authority"]):
        return "REEVALUATE"

    return "REJECT"
