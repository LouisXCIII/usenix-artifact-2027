#!/usr/bin/env python3
"""
independent_oblcovered.py

(a) Strength ordering assumed for the five obligation categories, from
weakest to strongest: empty (0) < notification (1) < audit (2) <
dual_approval (3) <= combined (4). Rationale: "empty" requires nothing by
definition, so it is the floor. "notification" is purely informational
(no gate on the action), so it is the lightest real control. "audit" adds
a persistent after-the-fact record, which is a stronger guarantee than a
notice but still does not block anything before it happens. "dual_approval"
gates the action itself on a second party's consent before it proceeds,
which this implementation treats as a strictly stronger control than an
audit trail alone (preventive beats detective). "combined" is modeled as
at least as strong as the strongest single mechanism (dual_approval), so
it sits at the top.

(b) Discharge rules implemented: whenever obligationProfile != "empty", a
genuine, on-chain discharge must be shown for THIS transfer: the case must
carry a non-empty obligationDischargeRef, the referenced record must exist
(not null), have ObligationDischarged==True, and match the case's own
(sourceDomain, role, amount) triple exactly (otherwise it is a confused-deputy
borrow of an unrelated transfer's discharge). The record must also not be
flagged alreadyCitedByAnotherDecision (single-use binding: a real discharge
is consumed on its first citation and cannot cover a second, distinct
decision, even for the identical triple).

(c) Ambiguities resolved: (1) The prose never pins down whether an empty
obligationProfile still needs enforcementWitnessVerified/discharge checks;
this implementation treats "empty" as requiring nothing at all (vacuously
covered), since there is no obligation to discharge or witness. (2) The
prose describes discharge via "a referenced decision record" but does not
say what happens when obligationProfile is non-empty yet no discharge is
even claimed (obligationDischargeRef == ""); this implementation treats an
absent claim the same as a never-discharged obligation -> not covered,
consistent with the paper's framing that an undischarged-but-claimed-safe
transfer is exactly the bug being fixed. (3) enforcementWitnessVerified is
explicitly "caller-supplied" with "no further logic" cross-checking it (the
open S7 laundering gap); this implementation still requires it to be True
for genuine (ground-truth) coverage when a mechanism is required, since the
task asks for "genuinely, fully covered," and a non-independently-witnessed
enforcement is not fully covered even though the real system cannot detect
a lie here.
"""

import json
import sys

STRENGTH = {
    "empty": 0,
    "notification": 1,
    "audit": 2,
    "dual_approval": 3,
    "combined": 4,
}


def mechanism_strength(value):
    """Map a mappedMechanism string (possibly "") to its ordinal strength."""
    if value == "" or value is None:
        return STRENGTH["empty"]
    return STRENGTH[value]


def is_discharge_genuine(case):
    """
    Returns True iff the case's obligationDischargeRef points to a real,
    matching, not-already-consumed discharge record.
    """
    ref = case.get("obligationDischargeRef", "")
    if not ref:
        # No discharge claimed at all -> obligation was never shown discharged.
        return False

    record = case.get("referencedDischargeRecord")
    if record is None:
        # Ref points to nothing that exists on-chain -> claim was never true.
        return False

    if not record.get("obligationDischarged", False):
        return False

    # Must be bound to the SAME transfer (sourceDomain, role, amount) --
    # otherwise this is a confused-deputy reuse of an unrelated discharge.
    if (
        record.get("sourceDomain") != case.get("sourceDomain")
        or record.get("role") != case.get("role")
        or record.get("amount") != case.get("amount")
    ):
        return False

    # Single-use binding: a real discharge already consumed by another
    # decision cannot cover this one too.
    if record.get("alreadyCitedByAnotherDecision", False):
        return False

    return True


def compute_covered(case):
    required = STRENGTH[case["obligationProfile"]]

    # Nothing is required at all -> vacuously, fully covered.
    if required == STRENGTH["empty"]:
        return True

    # Stripping (mapped to nothing) / weakening (mapped to something too
    # weak) check: the destination's mechanism must be at least as strong
    # as what the source domain's obligation profile requires.
    mapped = mechanism_strength(case.get("mappedMechanism", ""))
    if mapped < required:
        return False

    # Discharge must be genuinely shown for this exact transfer, and not
    # already consumed elsewhere.
    if not is_discharge_genuine(case):
        return False

    # Laundering check: the enforcement must be (claimed to be)
    # independently witnessed. This flag is trusted at face value, per the
    # spec, but is still required for genuine coverage.
    if not case.get("enforcementWitnessVerified", False):
        return False

    return True


def main():
    if len(sys.argv) != 3:
        sys.stderr.write(
            "usage: independent_oblcovered.py <input.json> <output.json>\n"
        )
        sys.exit(1)

    input_path = sys.argv[1]
    output_path = sys.argv[2]

    with open(input_path, "r") as f:
        cases = json.load(f)

    results = []
    for case in cases:
        covered = compute_covered(case)
        results.append({"decisionId": case["decisionId"], "covered": covered})

    with open(output_path, "w") as f:
        json.dump(results, f, indent=2)


if __name__ == "__main__":
    main()
