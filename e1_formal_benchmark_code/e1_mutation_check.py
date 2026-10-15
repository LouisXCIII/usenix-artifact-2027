#!/usr/bin/env python3
"""E1 validation step: mutation testing for e1_run_campaign.py's scoring harness.

Not part of the reported benchmark results. Confirms the FAR/FRR/accuracy
scoring pipeline actually has the power to detect a broken classifier, rather
than trivially reporting 0 errors regardless of input -- by deliberately
injecting known bugs into copies of classify() and checking that each mutant
produces a detectably nonzero FAR, FRR, or accuracy drop against the same
generated cases and ground truth used by e1_run_campaign.py.

Run standalone: ./.venv_e1/bin/python e1_mutation_check.py
Not imported by any other file; not part of the reported results pipeline.
"""
from e1_generator import generate_cases
from e1_ground_truth import ground_truth_class
from e1_classifier import ROLE_THRESHOLD, DEST_ACTION_CAP, RESOURCE_DEST_SET, CONTEXT_DEST_ACCEPTS, _claims

UNSAFE_CLASSES = {"CONDITIONAL", "REEVALUATE", "REJECT"}
DIRECT_REUSE_CLASSES = {"EXACT", "SAFE"}


def classify_mutant(case, mutation):
    """Reimplements classify() with one deliberate bug selected by `mutation`."""
    auth_src = bool(case["auth_src_ok"])
    gov_general = bool(case["gov_ok_general"])
    mapping_valid = bool(case["mapping_table_valid"])
    state_rel = bool(case["state_rel_ok"])
    temporal = case["temporal_state"]
    finality = case["finality_state"]
    replay_ok = bool(case["replay_ok"])
    toctou_ok = bool(case["toctou_ok"])
    obligation_discharged = bool(case["obligation_discharged"])

    if mutation == "skip_mapping_validity_check":
        mapping_valid_eff = True  # bug: ignore forged/widened mapping claims
    else:
        mapping_valid_eff = mapping_valid

    fatal = (
        (not auth_src) or (not gov_general) or (not mapping_valid_eff)
        or (temporal == "revoked") or (case["pair_type"] == "contradictory")
    )
    if fatal:
        return "REJECT"

    family = case["family"]
    if family == "role":
        if case["src_role"] == "unknown":
            sem_covered = False
        else:
            sem_covered = case["amount"] <= ROLE_THRESHOLD[case["claimed_dest_role"]]
    elif family == "action":
        sem_covered = 0 <= case["claimed_dest_action"] <= DEST_ACTION_CAP
    elif family == "resource":
        sem_covered = case["claimed_dest_resource"] in RESOURCE_DEST_SET[case["pair_type"]]
    elif family == "context":
        sem_covered = case["claimed_dest_context"] in CONTEXT_DEST_ACCEPTS[case["pair_type"]]
    else:
        raise ValueError(family)

    if mutation == "skip_replay_check":
        replay_ok_eff = True  # bug: ignore replay/reuse of consumed decisions
    else:
        replay_ok_eff = replay_ok

    if mutation == "skip_finality_check":
        final_ok_eff = True  # bug: accept regardless of foreign-finality threshold
    else:
        final_ok_eff = finality in ("at", "above")

    if mutation == "skip_assurance_check":
        assure_ok_eff = True  # bug: PQ requirement satisfied by any evidence tier (violates non-amplification)
    else:
        assure_ok_eff = case["dest_assurance_requirement"] in _claims(case["decision_sig_tier"])

    obl_covered = obligation_discharged
    fresh = temporal == "valid"

    portable = (
        auth_src and gov_general and mapping_valid_eff and state_rel
        and sem_covered and obl_covered and fresh and final_ok_eff and assure_ok_eff
        and replay_ok_eff and toctou_ok
    )
    if portable:
        birefine = case["pair_type"] == "equivalent" and case["mapping_type"] in ("identity", "renaming")
        return "EXACT" if birefine else "SAFE"

    repairable_gaps = set()
    if not obl_covered:
        repairable_gaps.add("obligation")
    if temporal == "stale":
        repairable_gaps.add("stale")
    if finality == "below":
        repairable_gaps.add("finality_below")
    non_repairable_failure = (
        (not sem_covered) or (not fresh and temporal != "stale")
        or (not final_ok_eff and finality != "below") or (not assure_ok_eff)
        or (not replay_ok_eff) or (not toctou_ok) or (not state_rel)
    )
    if repairable_gaps and not non_repairable_failure:
        return "CONDITIONAL"
    if bool(case["destination_local_authority"]):
        return "REEVALUATE"
    return "REJECT"


def score(cases, mutation):
    far_num = far_den = frr_num = frr_den = n_correct = 0
    for case in cases:
        gt = ground_truth_class(case)
        pred = classify_mutant(case, mutation)
        if gt == pred:
            n_correct += 1
        if gt in UNSAFE_CLASSES:
            far_den += 1
            if pred in DIRECT_REUSE_CLASSES:
                far_num += 1
        if gt in DIRECT_REUSE_CLASSES:
            frr_den += 1
            if pred == "REJECT":
                frr_num += 1
    far = far_num / far_den if far_den else 0.0
    frr = frr_num / frr_den if frr_den else 0.0
    acc = n_correct / len(cases)
    return far, frr, acc, far_num, frr_num


if __name__ == "__main__":
    cases = generate_cases(n_per_combo=40, seed=20260923)
    mutations = [
        "none",
        "skip_mapping_validity_check",
        "skip_replay_check",
        "skip_finality_check",
        "skip_assurance_check",
    ]
    print(f"{'mutation':32s} {'FAR':>10s} {'FRR':>10s} {'accuracy':>10s}  detected?")
    for mut in mutations:
        far, frr, acc, far_num, frr_num = score(cases, None if mut == "none" else mut)
        detected = "n/a (baseline)" if mut == "none" else ("YES" if far_num > 0 or acc < 1.0 else "NOT DETECTED")
        print(f"{mut:32s} {far:10.4f} {frr:10.4f} {acc:10.4f}  {detected}")
