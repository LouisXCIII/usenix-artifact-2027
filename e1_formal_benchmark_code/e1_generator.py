#!/usr/bin/env python3
"""E1: generated formal benchmark for ADP (Manuscriptsp_m2_r1.tex subsec:formal-benchmark).

Generates concrete decision-transfer instances stratified over the 50
(pair_type x mapping_type) combinations from e1_oracle.py, plus independently
drawn obligation/temporal/finality/assurance/replay/TOCTOU/governance/state
dimensions. Each case carries BOTH:
  - the observable fields a real verifier/classifier would see (src/claimed-dest
    values, obligation status, temporal/finality state, assurance evidence tiers,
    replay/TOCTOU/state-relation flags), and
  - a ground-truth-relevant `sem_covered_instance`/`mapping_table_valid` computed
    by DIRECT EVALUATION of the same deterministic destination guards/accepted-
    mapping tables (ground truth for a deterministic function is just evaluating
    it correctly; this is admission criterion (ii) in the paper's methodology).
    The class-level SMT soundness of each (pair_type, mapping_type) combination
    -- admission criterion (iii) -- is computed separately, once per combination,
    in e1_oracle.py.

Called by e1_run_campaign.py. Not imported anywhere else.
"""
import json
import random

from e1_oracle import PAIR_TYPES, MAPPING_TYPES, ROLE_THRESHOLD

ROLE_FAMILY = {"identity", "renaming", "role_refinement", "role_widening"}
ACTION_FAMILY = {"action_restriction", "action_expansion"}
RESOURCE_FAMILY = {"resource_substitution"}
CONTEXT_FAMILY = {"context_preservation", "context_loss", "context_strengthening"}

DEST_ACTION_CAP = 2

RESOURCE_DEST_SET = {
    "equivalent": [2], "source_stricter": [2], "destination_stricter": [2, 3, 4],
    "incomparable": [3, 4], "contradictory": [],
}
CONTEXT_DEST_ACCEPTS = {
    "equivalent": [True], "source_stricter": [True, False], "destination_stricter": [True],
    "incomparable": [False], "contradictory": [],
}


def thr(role):
    return ROLE_THRESHOLD[role]


def draw_role_instance(rng, pair_type):
    """Returns (src_role or 'unknown', amt)."""
    if rng.random() < 0.12:
        return "unknown", rng.randint(1000, 3_000_000)
    src_role = rng.choice([1, 2, 3])
    if pair_type == "equivalent":
        amt = rng.randint(1, thr(src_role))
    elif pair_type == "source_stricter":
        amt = rng.randint(1, max(1, thr(src_role) // 2))
    elif pair_type == "destination_stricter":
        nxt = min(src_role + 1, 3)
        amt = rng.randint(1, thr(nxt))
    elif pair_type == "incomparable":
        nxt = min(src_role + 1, 3)
        if rng.random() < 0.5:
            amt = rng.randint(1, max(1, thr(src_role) // 4))
        else:
            amt = rng.randint(thr(src_role) + 1, max(thr(src_role) + 1, thr(nxt)))
    elif pair_type == "contradictory":
        amt = rng.randint(thr(3) + 1, thr(3) + 2_000_000)
    else:
        raise ValueError(pair_type)
    return src_role, amt


def mapped_role(mapping_type, src_role):
    if src_role == "unknown":
        return None
    if mapping_type in ("identity", "renaming"):
        return src_role
    if mapping_type == "role_refinement":
        return max(1, src_role - 1)
    if mapping_type == "role_widening":
        return min(3, src_role + 1)
    raise ValueError(mapping_type)


def role_mapping_table_valid(mapping_type, src_role, claimed_dest):
    if src_role == "unknown":
        return False
    if mapping_type == "role_widening":
        return False  # no accepted table entry for a widened claim
    return claimed_dest == mapped_role(mapping_type, src_role)


def draw_action_instance(rng, pair_type):
    if pair_type == "equivalent":
        src = rng.randint(0, DEST_ACTION_CAP)
    elif pair_type == "source_stricter":
        src = rng.randint(0, max(0, DEST_ACTION_CAP - 1))
    elif pair_type == "destination_stricter":
        src = rng.randint(0, DEST_ACTION_CAP + 1)
    elif pair_type == "incomparable":
        src = rng.choice([0, 3])
    elif pair_type == "contradictory":
        src = 99
    else:
        raise ValueError(pair_type)
    return src


def draw_resource_instance():
    return 2  # source always asserts its true resource id = 2


def draw_context_instance():
    return True  # source's true context flag


def generate_cases(n_per_combo=40, seed=20260923):
    rng = random.Random(seed)
    cases = []
    cid = 0
    for pair_type in PAIR_TYPES:
        for mapping_type in MAPPING_TYPES:
            for _ in range(n_per_combo):
                cid += 1
                case = {
                    "id": f"e1-{cid:05d}",
                    "pair_type": pair_type,
                    "mapping_type": mapping_type,
                }

                if mapping_type in ROLE_FAMILY:
                    src_role, amt = draw_role_instance(rng, pair_type)
                    claimed_dest = mapped_role(mapping_type, src_role) if src_role != "unknown" else rng.choice([1, 2, 3])
                    case["family"] = "role"
                    case["src_role"] = src_role
                    case["amount"] = amt
                    case["claimed_dest_role"] = claimed_dest
                    case["mapping_table_valid"] = role_mapping_table_valid(mapping_type, src_role, claimed_dest)
                    if src_role == "unknown":
                        case["sem_covered_instance"] = False
                    else:
                        case["sem_covered_instance"] = amt <= thr(claimed_dest)

                elif mapping_type in ACTION_FAMILY:
                    src_action = draw_action_instance(rng, pair_type)
                    claimed = src_action + (1 if mapping_type == "action_expansion" else 0)
                    case["family"] = "action"
                    case["src_action"] = src_action
                    case["claimed_dest_action"] = claimed
                    case["mapping_table_valid"] = (mapping_type == "action_restriction")
                    case["sem_covered_instance"] = 0 <= claimed <= DEST_ACTION_CAP

                elif mapping_type in RESOURCE_FAMILY:
                    src_res = draw_resource_instance()
                    claimed = src_res
                    case["family"] = "resource"
                    case["src_resource"] = src_res
                    case["claimed_dest_resource"] = claimed
                    case["mapping_table_valid"] = True
                    case["sem_covered_instance"] = claimed in RESOURCE_DEST_SET[pair_type]

                elif mapping_type in CONTEXT_FAMILY:
                    src_ctx = draw_context_instance()
                    claimed = False if mapping_type == "context_loss" else src_ctx
                    case["family"] = "context"
                    case["src_context"] = src_ctx
                    case["claimed_dest_context"] = claimed
                    case["mapping_table_valid"] = True
                    case["sem_covered_instance"] = claimed in CONTEXT_DEST_ACCEPTS[pair_type]
                else:
                    raise ValueError(mapping_type)

                # Independent dimensions (orthogonal to the semantic-mapping family)
                case["obligation_profile"] = rng.choices(
                    ["empty", "audit", "dual_approval", "notification", "combined"],
                    weights=[40, 15, 15, 15, 15])[0]
                case["obligation_discharged"] = (
                    True if case["obligation_profile"] == "empty" else rng.random() < 0.6)

                case["temporal_state"] = rng.choices(
                    ["valid", "not_yet_valid", "expired", "stale", "epoch_mismatched", "revoked"],
                    weights=[70, 6, 6, 6, 6, 6])[0]

                case["finality_state"] = rng.choices(["above", "at", "below"], weights=[65, 20, 15])[0]

                case["decision_sig_tier"] = rng.choices(["classical", "hybrid", "pq"], weights=[40, 30, 30])[0]
                case["transport_tier"] = rng.choices(["classical", "hybrid", "pq"], weights=[30, 35, 35])[0]
                case["dest_assurance_requirement"] = rng.choices(
                    ["ClassicalDecisionAuth", "PQDecisionAuth"], weights=[55, 45])[0]

                case["gov_ok_general"] = rng.random() < 0.92
                case["auth_src_ok"] = rng.random() < 0.95
                case["state_rel_ok"] = rng.random() < 0.90
                case["replay_ok"] = rng.random() < 0.93
                case["toctou_ok"] = rng.random() < 0.93
                case["destination_local_authority"] = (case.get("src_role") == "unknown") or rng.random() < 0.12

                cases.append(case)
    return cases


if __name__ == "__main__":
    cases = generate_cases()
    with open("e1_generated_cases_2026-09-23.json", "w") as f:
        json.dump(cases, f, indent=2)
    print(f"Generated {len(cases)} cases across {len(PAIR_TYPES) * len(MAPPING_TYPES)} (pair_type x mapping_type) combinations.")
