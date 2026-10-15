#!/usr/bin/env python3
"""Run the real OPA v0.70.0 engine (same binary as the Group 1 pilot) on every case for
both the naive and hardened Rego policy in each of groups 2/3/4, via subprocess `opa
eval` -- not re-implemented/simulated. Records mismatches against real_grant."""
import json
import os
import subprocess

HERE = os.path.dirname(os.path.abspath(__file__))
OPA = "/tmp/claude-1001/-home-deployer-Desktop/a3382147-c260-49ca-a416-2758524dd365/scratchpad/qd_pilot/opa"

GROUPS = {
    "g2_obligation": {"cases": "cases2.json"},
    "g3_multihop": {"cases": "cases3.json"},
    "g4_vocab": {"cases": "cases4.json"},
}


def detect_package(path):
    for line in open(path):
        line = line.strip()
        if line.startswith("package "):
            return line.split()[1]
    raise RuntimeError(f"no package decl in {path}")


def eval_case(policy_path, pkg, input_obj, query_field="allow"):
    inp = json.dumps(input_obj)
    q = f"data.{pkg}.{query_field}"
    r = subprocess.run([OPA, "eval", "-d", policy_path, "-I", q, "--format", "raw"],
                        input=inp, capture_output=True, text=True, timeout=10)
    out = r.stdout.strip()
    if r.returncode != 0:
        raise RuntimeError(f"opa eval failed (rc={r.returncode}) on {policy_path}: {out[:300]}")
    if out == "true":
        return True
    if out == "false" or out == "":
        return False
    return None  # unexpected


def case_to_input(group, case):
    if group == "g2_obligation":
        return {"role": case["role"], "amount": case["amount"], "evidence_type": case["evidence_type"]}
    if group == "g3_multihop":
        return {"hops": case["hops"]}
    if group == "g4_vocab":
        return {"source_role": case["source_role"], "amount": case["amount"]}
    raise ValueError(group)


def main():
    all_results = {}
    for group, cfg in GROUPS.items():
        cases = json.load(open(os.path.join(HERE, cfg["cases"])))
        naive_path = os.path.join(HERE, group, "naive_opa", "policy.rego")
        hardened_path = os.path.join(HERE, group, "hardened_opa", "policy.rego")
        naive_pkg = detect_package(naive_path)
        hardened_pkg = detect_package(hardened_path)
        results = {"naive_pkg": naive_pkg, "hardened_pkg": hardened_pkg,
                   "naive_mismatches": [], "hardened_mismatches": [],
                   "n_cases": len(cases)}
        for case in cases:
            inp = case_to_input(group, case)
            real = case["real_grant"]
            naive_decision = eval_case(naive_path, naive_pkg, inp)
            hardened_decision = eval_case(hardened_path, hardened_pkg, inp)
            if naive_decision != real:
                results["naive_mismatches"].append({"case_id": case["case_id"], "input": inp,
                                                      "real": real, "got": naive_decision})
            if hardened_decision != real:
                results["hardened_mismatches"].append({"case_id": case["case_id"], "input": inp,
                                                          "real": real, "got": hardened_decision})
        results["naive_mismatch_count"] = len(results["naive_mismatches"])
        results["hardened_mismatch_count"] = len(results["hardened_mismatches"])
        all_results[group] = results
        print(f"{group}: n={len(cases)} naive_mismatches={results['naive_mismatch_count']} "
              f"hardened_mismatches={results['hardened_mismatch_count']}")
    json.dump(all_results, open(os.path.join(HERE, "opa_results.json"), "w"), indent=2)


if __name__ == "__main__":
    main()
