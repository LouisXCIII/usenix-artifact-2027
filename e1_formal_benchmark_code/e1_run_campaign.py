#!/usr/bin/env python3
"""E1: runs the full formal-benchmark campaign for Manuscriptsp_m2_r1.tex
Section "Generated Formal Benchmark and Ground Truth" / Section "Security Results".

Pipeline:
  1. e1_oracle.build_oracle_table()      -- Z3 class-level soundness audit of all
                                             50 (pair_type x mapping_type) combos.
  2. e1_generator.generate_cases()       -- N concrete instances stratified over
                                             those 50 combos plus independent
                                             obligation/temporal/finality/assurance/
                                             replay/TOCTOU/governance dimensions.
  3. e1_ground_truth.ground_truth_class() -- true label per case (direct evaluation).
  4. e1_classifier.classify()             -- predicted label per case (system under
                                              test, separate code path).
  5. Aggregate FAR/FRR (eq:far, eq:frr), five-class accuracy, full confusion
     matrix, and per-(pair_type,mapping_type) breakdown.

Writes e1_formal_benchmark_results_2026-09-23.json. Prints a summary used to
fill Manuscriptsp_m2_r1.tex's \\Res* macros and Tables tab:formal-security-results
/ tab:baseline-security-results (B4 row) by hand (no auto-edit of the .tex file
from this script; the .tex is edited separately after reviewing this output).
"""
import json
from collections import Counter, defaultdict

from e1_oracle import build_oracle_table, PAIR_TYPES, MAPPING_TYPES
from e1_generator import generate_cases
from e1_ground_truth import ground_truth_class
from e1_classifier import classify

UNSAFE_CLASSES = {"CONDITIONAL", "REEVALUATE", "REJECT"}  # i.e. NOT {EXACT, SAFE}
DIRECT_REUSE_CLASSES = {"EXACT", "SAFE"}


def main():
    oracle_table = build_oracle_table()

    cases = generate_cases(n_per_combo=40, seed=20260923)
    n_total = len(cases)

    confusion = defaultdict(Counter)  # confusion[gt_class][pred_class] += 1
    far_num = 0  # unsafe (gt) -> {Exact,Safe} (pred)
    far_den = 0  # unsafe (gt) count
    frr_num = 0  # safe (gt) -> Reject (pred)
    frr_den = 0  # safe (gt) count
    per_combo = defaultdict(lambda: {"n": 0, "gt": Counter(), "pred": Counter(), "far_num": 0, "far_den": 0})

    mismatches = []

    for case in cases:
        gt = ground_truth_class(case)
        pred, diag = classify(case)
        confusion[gt][pred] += 1

        combo_key = f"{case['pair_type']}|{case['mapping_type']}"
        pc = per_combo[combo_key]
        pc["n"] += 1
        pc["gt"][gt] += 1
        pc["pred"][pred] += 1

        gt_unsafe = gt in UNSAFE_CLASSES
        gt_safe_direct = gt in DIRECT_REUSE_CLASSES

        if gt_unsafe:
            far_den += 1
            pc["far_den"] += 1
            if pred in DIRECT_REUSE_CLASSES:
                far_num += 1
                pc["far_num"] += 1
        if gt_safe_direct:
            frr_den += 1
            if pred == "REJECT":
                frr_num += 1

        if gt != pred:
            mismatches.append({"id": case["id"], "combo": combo_key, "gt": gt, "pred": pred})

    far = far_num / far_den if far_den else 0.0
    frr = frr_num / frr_den if frr_den else 0.0
    n_correct = sum(confusion[c][c] for c in confusion)
    accuracy = n_correct / n_total

    gt_totals = Counter()
    for gt, preds in confusion.items():
        gt_totals[gt] += sum(preds.values())

    results = {
        "n_total": n_total,
        "n_pair_types": len(PAIR_TYPES),
        "n_mapping_types": len(MAPPING_TYPES),
        "n_combinations": len(PAIR_TYPES) * len(MAPPING_TYPES),
        "n_per_combo": 40,
        "seed": 20260923,
        "far": far,
        "far_num_unsafe_accepted": far_num,
        "far_den_unsafe_total": far_den,
        "frr": frr,
        "frr_num_safe_rejected": frr_num,
        "frr_den_safe_total": frr_den,
        "five_class_accuracy": accuracy,
        "n_correct": n_correct,
        "ground_truth_class_totals": dict(gt_totals),
        "confusion_matrix": {gt: dict(preds) for gt, preds in confusion.items()},
        "n_mismatches": len(mismatches),
        "mismatches_sample": mismatches[:25],
        "oracle_class_level_soundness": {
            "n_combinations": len(oracle_table),
            "n_sound": sum(1 for v in oracle_table.values() if v["sem_covered"]),
            "n_unsound": sum(1 for v in oracle_table.values() if not v["sem_covered"]),
            "table": oracle_table,
        },
        "per_combo_breakdown": {
            k: {
                "n": v["n"],
                "ground_truth": dict(v["gt"]),
                "predicted": dict(v["pred"]),
                "far_num": v["far_num"],
                "far_den": v["far_den"],
            }
            for k, v in per_combo.items()
        },
    }

    with open("e1_formal_benchmark_results_2026-09-23.json", "w") as f:
        json.dump(results, f, indent=2)

    print(f"N cases: {n_total} across {results['n_combinations']} (pair_type x mapping_type) combinations")
    print(f"Ground-truth class totals: {dict(gt_totals)}")
    print(f"FAR (unsafe -> Exact/Safe) = {far_num}/{far_den} = {far:.6f}")
    print(f"FRR (safe -> Reject)       = {frr_num}/{frr_den} = {frr:.6f}")
    print(f"Five-class accuracy        = {n_correct}/{n_total} = {accuracy:.6f}")
    print(f"Mismatches: {len(mismatches)}")
    if mismatches:
        for m in mismatches[:10]:
            print(f"  {m}")
    print(f"Oracle: {results['oracle_class_level_soundness']['n_sound']}/{results['oracle_class_level_soundness']['n_combinations']} combinations universally sound (Z3-verified)")


if __name__ == "__main__":
    main()
