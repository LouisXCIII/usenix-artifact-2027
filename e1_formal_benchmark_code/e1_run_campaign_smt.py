#!/usr/bin/env python3
"""E1 limitation (1) fix, verification run: reruns the full N=2000 E1 campaign using
e1_ground_truth_smt.py (Z3-based) as ground truth instead of e1_ground_truth.py (plain Python),
scored against e1_classifier.py. Also cross-checks the Z3-based ground truth against the original
plain-Python ground truth on every case, to confirm the two independently-written encodings of
the same formal definition agree (a sanity check on the Z3 encoding itself, not part of the
reported classifier-accuracy result).
"""
import json

from e1_generator import generate_cases
from e1_ground_truth import ground_truth_class
from e1_ground_truth_smt import ground_truth_class_smt
from e1_classifier import classify

UNSAFE_CLASSES = {"CONDITIONAL", "REEVALUATE", "REJECT"}
DIRECT_REUSE_CLASSES = {"EXACT", "SAFE"}


def main():
    cases = generate_cases(n_per_combo=40, seed=20260923)
    n = len(cases)

    far_num = far_den = frr_num = frr_den = n_correct = 0
    gt_vs_gt_smt_mismatches = []

    for case in cases:
        gt_plain = ground_truth_class(case)
        gt_smt = ground_truth_class_smt(case)
        if gt_plain != gt_smt:
            gt_vs_gt_smt_mismatches.append({"id": case["id"], "plain": gt_plain, "smt": gt_smt})

        pred, _ = classify(case)
        if gt_smt == pred:
            n_correct += 1
        if gt_smt in UNSAFE_CLASSES:
            far_den += 1
            if pred in DIRECT_REUSE_CLASSES:
                far_num += 1
        if gt_smt in DIRECT_REUSE_CLASSES:
            frr_den += 1
            if pred == "REJECT":
                frr_num += 1

    far = far_num / far_den if far_den else 0.0
    frr = frr_num / frr_den if frr_den else 0.0
    accuracy = n_correct / n

    print(f"N cases: {n}")
    print(f"Plain-Python ground truth vs Z3 ground truth mismatches: {len(gt_vs_gt_smt_mismatches)} (encoding sanity check)")
    for m in gt_vs_gt_smt_mismatches[:10]:
        print("  ", m)
    print(f"Classifier vs Z3 ground truth: FAR={far_num}/{far_den}={far:.6f}  FRR={frr_num}/{frr_den}={frr:.6f}  accuracy={n_correct}/{n}={accuracy:.6f}")

    with open("e1_run_campaign_smt_result_2026-09-23.json", "w") as f:
        json.dump({
            "n": n,
            "gt_encoding_mismatches": len(gt_vs_gt_smt_mismatches),
            "gt_encoding_mismatch_sample": gt_vs_gt_smt_mismatches[:25],
            "far": far, "far_num": far_num, "far_den": far_den,
            "frr": frr, "frr_num": frr_num, "frr_den": frr_den,
            "accuracy": accuracy, "n_correct": n_correct,
        }, f, indent=2)
    print("Wrote e1_run_campaign_smt_result_2026-09-23.json")


if __name__ == "__main__":
    main()
