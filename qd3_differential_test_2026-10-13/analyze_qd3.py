#!/usr/bin/env python3
"""Compare independent_oblcovered.py output vs the real-function Go harness output.
Usage: analyze_qd3.py cases.json independent_output.json real_output.json summary.json
Reports on ALL cases -- no category is excluded from the match/mismatch count."""
import json
import sys


def main():
    cases_path, indep_path, real_path, out_path = sys.argv[1:5]
    cases = {c["decisionId"]: c for c in json.load(open(cases_path))}
    indep = {o["decisionId"]: o["covered"] for o in json.load(open(indep_path))}
    real = json.load(open(real_path))

    n_match = n_mismatch = n_err = 0
    mismatches = []
    for r in real:
        did = r["decisionId"]
        if r.get("err"):
            n_err += 1
            continue
        if did not in indep:
            continue
        if r["covered"] == indep[did]:
            n_match += 1
        else:
            n_mismatch += 1
            mismatches.append({
                "decisionId": did, "real_covered": r["covered"], "real_class": r["class"],
                "independent_covered": indep[did],
                "axes": cases[did].get("_sweepAxes", "n/a (not a sweep case)"),
            })

    summary = {"n_total": len(real), "n_match": n_match, "n_mismatch": n_mismatch,
               "n_err": n_err, "mismatches": mismatches}
    json.dump(summary, open(out_path, "w"), indent=2)
    print(json.dumps({k: v for k, v in summary.items() if k != "mismatches"}, indent=2))
    if mismatches:
        print(f"{len(mismatches)} MISMATCHES -- see {out_path}")


if __name__ == "__main__":
    main()
