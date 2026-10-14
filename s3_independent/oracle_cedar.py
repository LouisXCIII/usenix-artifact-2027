#!/usr/bin/env python3
"""Independent oracle for S3 (see s3_independent_groundtruth_protocol_2026-09-28.md).

Semantics of the destination policy P_d are expressed as CEDAR policies, written from the specification (not from the Go code).
The class is assembled from four decisions per case:
  A  Cedar decision with second approval absent     B  Cedar decision with second approval present
  C  role recognised (Cedar, amount=1, approval true) D  the two source attestations (spec-level, checked first)
usage: oracle_cedar.py cases.json out.json      (needs: pip install cedarpy)
A repeated decisionId is a replay: expected outcome "REPLAY_ERROR" (spec: an error is returned, nothing overwritten)."""
import json, sys
import cedarpy

CAPS = {"manager": 500000, "director": 2000000, "clerk": 10000}
REGISTRY = {"gerente": "manager", "gerente_senior": "director"}

POLICIES = "\n".join(
    f'''permit(principal in DestRole::"{role}", action == Action::"approve", resource)
 when {{ context.amount <= {cap} && (context.secondApproval || context.amount * 2 <= {cap}) }};'''
    for role, cap in CAPS.items()
)
ENTITIES = (
    [{"uid": {"type": "DestRole", "id": r}, "attrs": {}, "parents": []} for r in CAPS]
    + [{"uid": {"type": "SourceRole", "id": s}, "attrs": {}, "parents": [{"type": "DestRole", "id": d}]} for s, d in REGISTRY.items()]
)


def principal_of(role):
    """Role string -> Cedar principal (exact, case-sensitive; no normalisation)."""
    if role in CAPS:
        return {"type": "DestRole", "id": role}, True
    if role in REGISTRY:
        return {"type": "SourceRole", "id": role}, False
    return {"type": "Unknown", "id": role}, False


def cedar_allow(principal, amount, second, extra_entities):
    req = {
        "principal": principal,
        "action": {"type": "Action", "id": "approve"},
        "resource": {"type": "Res", "id": "r"},
        "context": {"amount": int(amount), "secondApproval": bool(second)},
    }
    return cedarpy.is_authorized(req, POLICIES, ENTITIES + extra_entities).decision == cedarpy.Decision.Allow


def oracle_class(case):
    principal, identity = principal_of(case["role"])
    extra = [] if principal["type"] != "Unknown" else [{"uid": principal, "attrs": {}, "parents": []}]
    amount = int(case["amount"])
    A = cedar_allow(principal, amount, False, extra)
    B = cedar_allow(principal, amount, True, extra)
    C = cedar_allow(principal, 1, True, extra)
    # Spec: "If amount > half of the cap a second approval is required; if secondApprovalObtained is false -> CONDITIONAL;
    # otherwise permitted." So the decision that applies to THIS case uses the case's actual approval flag.
    actual = B if case["secondApprovalObtained"] else A
    if not (case["semBound"] and case["evAssuranceOK"]):
        cls = "REJECT"
    elif not C:
        cls = "REEVALUATE"
    elif actual:
        cls = "EXACT" if identity else "SAFE"
    elif B:
        cls = "CONDITIONAL"
    else:
        cls = "REJECT"
    return cls, A, B, C, principal["type"]


def fits_int64(x):
    return isinstance(x, int) and -(2**63) <= x <= 2**63 - 1


def main(src, dst):
    cases = json.load(open(src))
    seen, out = set(), []
    for c in cases:
        if not (fits_int64(c["amount"]) and fits_int64(c["sourceLimit"])):
            # cannot be submitted (64-bit function signature; Cedar Long is 64-bit): skipped by the harness, so it never counts as "seen"
            out.append({"id": c["id"], "oracle_class": "SKIPPED_INT64", "class_ignoring_replay": "SKIPPED_INT64", "A": None, "B": None,
                        "C": None, "principal_type": None, "in_domain": False, "replay": False})
            continue
        replay = c["decisionId"] in seen
        seen.add(c["decisionId"])
        cls, A, B, C, ptype = oracle_class(c)
        in_domain = isinstance(c["amount"], int) and c["amount"] > 0 and isinstance(c["role"], str) and c["role"] != ""
        out.append({
            "id": c["id"], "oracle_class": "REPLAY_ERROR" if replay else cls, "class_ignoring_replay": cls,
            "A": A, "B": B, "C": C, "principal_type": ptype, "in_domain": in_domain, "replay": replay,
        })
    json.dump(out, open(dst, "w"), indent=1)
    print(f"oracle: {len(out)} cases -> {dst}")


if __name__ == "__main__":
    if len(sys.argv) != 3:
        sys.exit(__doc__)
    main(sys.argv[1], sys.argv[2])
