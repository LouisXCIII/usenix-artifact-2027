#!/usr/bin/env python3
"""Mechanical, exhaustive case generator for the independent-policy-oracle check
(replaces the 2026-09-28 S3 AI-adversary generator: adversary_gen.py / families_adversary_gen.py).

Every case is produced by a fixed formula over a Cartesian product of axes below -- there is
no hand-authored "adversarial" case, no narrative rationale, and no randomness (contrast with
the removed adversary_gen.py, which had AI-authored per-case rationale strings, and with
sampler.py, which used a random seed). This generator is deterministic: re-running it produces
byte-identical output.

Axes:
  role        5 known roles (3 destination-identity + 2 source-registry-mapped) + 2 malformed
              role strings (empty, unrecognised) to exercise the REEVALUATE/unknown-role path.
  amount      10 points per role, derived by formula from that role's cap (see AMOUNTS_FOR):
              1, half-1, half, half+1, cap-1, cap, cap+1, 2*cap, 0, -1.
  semBound, evAssuranceOK, secondApprovalObtained
              full 2x2x2 = 8 combinations.
  decisionId  unique per combination ("mech-NNNNN"), except REPLAY_COUNT of them are forced to
              reuse an earlier decisionId verbatim (first REPLAY_COUNT generated ids, taken in
              generation order) to mechanically exercise the REPLAY_ERROR path.
  sourceLimit fixed = amount (the Cedar policy fragment under test does not model sourceLimit;
              same scope restriction as the original S3, documented in the results file).

usage: cases_mechanical_sweep.py out.json
"""
import json, sys

CAPS = {"manager": 500000, "director": 2000000, "clerk": 10000}
REGISTRY = {"gerente": "manager", "gerente_senior": "director"}
MALFORMED_ROLES = ["", "not_a_role"]
REPLAY_COUNT = 10


def amounts_for(cap):
    half = cap // 2
    # fixed formula, same 10 points for every role, derived only from that role's cap
    return [1, half - 1, half, half + 1, cap - 1, cap, cap + 1, 2 * cap, 0, -1]


def axis_tag(amount, cap):
    half = cap // 2
    if amount <= 0:
        return "non_positive_amount"
    if amount in (cap - 1, cap, cap + 1, 2 * cap):
        return "cap_boundary"
    if amount in (half - 1, half, half + 1):
        return "half_cap_boundary"
    return "low_amount"


def main(dst):
    cases = []
    n = 0
    all_roles = list(CAPS) + list(REGISTRY) + MALFORMED_ROLES
    for role in all_roles:
        cap = CAPS.get(role) or CAPS.get(REGISTRY.get(role), 1)  # malformed roles: cap unused, fixed base for amount formula
        for amount in amounts_for(cap):
            for sem in (True, False):
                for ev in (True, False):
                    for sec in (True, False):
                        n += 1
                        cid = f"M{n:05d}"
                        did = f"mech-{n:05d}"
                        cases.append({
                            "id": cid,
                            "category": axis_tag(amount, cap) if role not in MALFORMED_ROLES else "malformed_role",
                            "decisionId": did,
                            "sourceDomain": "besu-mainnet",
                            "role": role,
                            "amount": amount,
                            "sourceLimit": amount,
                            "semBound": sem,
                            "evAssuranceOK": ev,
                            "secondApprovalObtained": sec,
                        })

    # mechanical replay axis: force REPLAY_COUNT cases to reuse the decisionId of an EARLIER,
    # amount>0/known-role case (so the replay-check's effect is observable independent of the
    # amount<=0 hardening guard -- both the source and the reused-id case must be "in-domain
    # valid" or a disabled replay-check would be masked by an unrelated early return). Source
    # = the first REPLAY_COUNT in-domain-valid cases generated; target = the LAST REPLAY_COUNT
    # in-domain-valid cases generated (both deterministic, by generation order).
    valid_idx = [i for i, c in enumerate(cases) if c["amount"] > 0 and c["role"] not in MALFORMED_ROLES]
    replay_targets = [cases[i]["decisionId"] for i in valid_idx[:REPLAY_COUNT]]
    for i, did in enumerate(replay_targets):
        idx = valid_idx[-REPLAY_COUNT + i]
        cases[idx] = dict(cases[idx])
        cases[idx]["decisionId"] = did
        cases[idx]["category"] = "replay"

    json.dump(cases, open(dst, "w"), indent=1)
    by_cat = {}
    for c in cases:
        by_cat[c["category"]] = by_cat.get(c["category"], 0) + 1
    print(f"generated {len(cases)} cases -> {dst}")
    for cat, cnt in sorted(by_cat.items()):
        print(f"  {cat}: {cnt}")


if __name__ == "__main__":
    if len(sys.argv) != 2:
        sys.exit(__doc__)
    main(sys.argv[1])
