#!/usr/bin/env python3
"""
Mechanical, deterministic sweep generator for the QD3 differential test (obligation-coverage
logic, SubmitDecisionTrustlessV7ObligationVerifiedDischarge). Full Cartesian product, NO
combination excluded (per explicit review instruction 2026-10-13: excluding `required==0`/
"empty" or excluding "known-bug" categories like S7 both read as result-selection).

Axes:
  obligationProfile          5 values, including "empty"
  mappedMechanism            5 values, including "" (nothing mapped)
  enforcementWitnessVerified 2 values
  discharge-ref state        7 values (no_ref, match, domain_mismatch, role_mismatch,
                              amount_mismatch, not_exist, not_discharged)
  alreadyCitedByAnotherDecision  2 values (kept even where not operative for a given
                              discharge-ref state -- redundant in some slices, intentionally
                              not filtered out)

Total: 5*5*2*7*2 = 700 cases.
"""
import json

PROFILES = ["empty", "notification", "audit", "dual_approval", "combined"]
MECHANISMS = ["", "notification", "audit", "dual_approval", "combined"]
WITNESS = [True, False]
ALREADY_CITED = [True, False]

FIXED_SOURCE_DOMAIN = "besu-mainnet"
FIXED_ROLE = "manager"
FIXED_AMOUNT = 300000

DISCHARGE_STATES = [
    "no_ref", "match", "domain_mismatch", "role_mismatch",
    "amount_mismatch", "not_exist", "not_discharged",
]


def discharge_fields(state, already_cited):
    if state == "no_ref":
        return "", None
    ref = "qd3-sweep-ref"
    if state == "not_exist":
        return ref + "-missing", None
    rec = {
        "obligationDischarged": state != "not_discharged",
        "sourceDomain": FIXED_SOURCE_DOMAIN,
        "role": FIXED_ROLE,
        "amount": FIXED_AMOUNT,
        "alreadyCitedByAnotherDecision": already_cited,
    }
    if state == "domain_mismatch":
        rec["sourceDomain"] = "other-domain"
    elif state == "role_mismatch":
        rec["role"] = "director"
    elif state == "amount_mismatch":
        rec["amount"] = 999999
    return ref, rec


def generate():
    cases = []
    i = 0
    for profile in PROFILES:
        for mechanism in MECHANISMS:
            for witness in WITNESS:
                for dstate in DISCHARGE_STATES:
                    for already_cited in ALREADY_CITED:
                        i += 1
                        ref, rec = discharge_fields(dstate, already_cited)
                        cases.append({
                            "decisionId": f"qd3-sweep-{i:04d}",
                            "sourceDomain": FIXED_SOURCE_DOMAIN,
                            "role": FIXED_ROLE,
                            "amount": FIXED_AMOUNT,
                            "obligationProfile": profile,
                            "mappedMechanism": mechanism,
                            "obligationDischargeRef": ref,
                            "referencedDischargeRecord": rec,
                            "enforcementWitnessVerified": witness,
                            "_sweepAxes": {
                                "obligationProfile": profile, "mappedMechanism": mechanism,
                                "enforcementWitnessVerified": witness,
                                "dischargeState": dstate, "alreadyCitedByAnotherDecision": already_cited,
                            },
                        })
    return cases


if __name__ == "__main__":
    import sys
    out = sys.argv[1] if len(sys.argv) > 1 else "sweep_cases.json"
    cases = generate()
    json.dump(cases, open(out, "w"), indent=1)
    print(f"generated {len(cases)} cases -> {out}")
