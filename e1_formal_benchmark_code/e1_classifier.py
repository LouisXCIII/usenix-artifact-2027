#!/usr/bin/env python3
"""E1: classifier under test -- a standalone reimplementation of
Manuscriptsp_m2_r1.tex eq:portable-final / eq:five-class-classifier, applied to
the cases produced by e1_generator.py. This is a separate code path from
e1_generator.py's ground-truth computation and from e1_oracle.py's Z3
class-level soundness audit (independence per the paper's 3-part ground-truth
admission criterion). Its role/amount-threshold fragment mirrors the real
SubmitDecisionFull chaincode logic (RoleThreshold={clerk:10000,manager:500000,
director:2000000}) already validated on-chain in
evidence/all_five_transfer_classes_2026-09-21.md; the temporal/finality/
assurance/replay/TOCTOU/obligation conjuncts are direct translations of the
formal definitions in Section "Authorization-Decision Portability Semantics"
(eq:freshness-predicate, eq:finality-predicate, eq:assurance-sufficient,
eq:replay-safe, eq:toctou-safe, eq:obligation-covered).

Called by e1_run_campaign.py. Not imported anywhere else.
"""

ROLE_THRESHOLD = {1: 10000, 2: 500000, 3: 2000000}
DEST_ACTION_CAP = 2
RESOURCE_DEST_SET = {
    "equivalent": [2], "source_stricter": [2], "destination_stricter": [2, 3, 4],
    "incomparable": [3, 4], "contradictory": [],
}
CONTEXT_DEST_ACCEPTS = {
    "equivalent": [True], "source_stricter": [True, False], "destination_stricter": [True],
    "incomparable": [False], "contradictory": [],
}


def _claims(decision_sig_tier):
    if decision_sig_tier == "classical":
        return {"ClassicalDecisionAuth"}
    if decision_sig_tier == "hybrid":
        return {"ClassicalDecisionAuth", "PQDecisionAuth"}
    if decision_sig_tier == "pq":
        return {"PQDecisionAuth"}
    raise ValueError(decision_sig_tier)


def classify(case):
    """Returns (predicted_class, diagnostics dict)."""
    diag = {}

    auth_src = bool(case["auth_src_ok"])
    gov_general = bool(case["gov_ok_general"])
    mapping_valid = bool(case["mapping_table_valid"])
    state_rel = bool(case["state_rel_ok"])
    temporal = case["temporal_state"]
    finality = case["finality_state"]
    replay_ok = bool(case["replay_ok"])
    toctou_ok = bool(case["toctou_ok"])
    obligation_discharged = bool(case["obligation_discharged"])

    fatal = (
        (not auth_src)
        or (not gov_general)
        or (not mapping_valid)
        or (temporal == "revoked")
        or (case["pair_type"] == "contradictory")
    )
    diag["fatal"] = fatal
    if fatal:
        return "REJECT", diag

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
    diag["sem_covered"] = sem_covered

    obl_covered = obligation_discharged
    fresh = temporal == "valid"
    final_ok = finality in ("at", "above")
    assure_ok = case["dest_assurance_requirement"] in _claims(case["decision_sig_tier"])
    diag.update(obl_covered=obl_covered, fresh=fresh, final_ok=final_ok, assure_ok=assure_ok)

    portable = (
        auth_src and gov_general and mapping_valid and state_rel
        and sem_covered and obl_covered and fresh and final_ok and assure_ok
        and replay_ok and toctou_ok
    )
    diag["portable"] = portable

    if portable:
        birefine = (
            case["pair_type"] == "equivalent"
            and case["mapping_type"] in ("identity", "renaming")
        )
        return ("EXACT" if birefine else "SAFE"), diag

    # Exactly-one-repairable-gap test for CONDITIONAL (Satisfiable(Delta)).
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
    satisfiable_delta = bool(repairable_gaps) and not non_repairable_failure
    diag["satisfiable_delta"] = satisfiable_delta
    if satisfiable_delta:
        return "CONDITIONAL", diag

    if bool(case["destination_local_authority"]):
        return "REEVALUATE", diag

    return "REJECT", diag
