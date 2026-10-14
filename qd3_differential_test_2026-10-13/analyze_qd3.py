#!/usr/bin/env python3
"""Compare independent_oblcovered.py output vs the real-function Go harness output.
Usage: analyze_qd3.py cases.json independent_output.json real_output.json summary.json
Reports on ALL cases -- no category is excluded from the match/mismatch count.

2026-10-14 fix (self-audit finding): a case present in real_output.json but absent
from independent_output.json used to be silently skipped (not counted in n_match,
n_mismatch, or n_err, while n_total still counted it) -- a future run with any
real/independent ID drift would have under-reported its true mismatch rate without
any visible signal. Verified on the actual 2026-10-13 run data that this never
affected the published 707/707 result (both files' ID sets matched exactly), but the
script itself had no structural guarantee of that. Fixed: any such case is now
counted explicitly as n_missing_from_indep and listed, and the script exits non-zero
with a loud warning if this count is ever nonzero, rather than passing silently."""
import json
import sys


def main():
    cases_path, indep_path, real_path, out_path = sys.argv[1:5]
    cases = {c["decisionId"]: c for c in json.load(open(cases_path))}
    indep = {o["decisionId"]: o["covered"] for o in json.load(open(indep_path))}
    real = json.load(open(real_path))

    n_match = n_mismatch = n_err = 0
    mismatches = []
    missing_from_indep = []
    for r in real:
        did = r["decisionId"]
        if r.get("err"):
            n_err += 1
            continue
        if did not in indep:
            missing_from_indep.append(did)
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
               "n_err": n_err, "n_missing_from_indep": len(missing_from_indep),
               "missing_from_indep": missing_from_indep, "mismatches": mismatches}
    json.dump(summary, open(out_path, "w"), indent=2)
    print(json.dumps({k: v for k, v in summary.items() if k != "mismatches"}, indent=2))
    if mismatches:
        print(f"{len(mismatches)} MISMATCHES -- see {out_path}")
    if missing_from_indep:
        print(f"WARNING: {len(missing_from_indep)} decisionId(s) in real_output.json have "
              f"no matching entry in independent_output.json -- these were NOT compared "
              f"and are counted separately, not silently dropped. See {out_path}.")
        sys.exit(1)


if __name__ == "__main__":
    main()
