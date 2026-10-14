#!/usr/bin/env python3
"""Power check of the S3 test suite: inject known defects into a SCRATCH COPY of the Go decision code and see whether the suite notices.
usage: mutation_check.py <authcc_copy_dir> <original_authchaincode.go> <oracle_cedar.py> <python_with_cedarpy> <out.json> <name>=<cases.json> [...]
Never touches the deployed tree: it edits <authcc_copy_dir>/chaincode/authchaincode.go and restores the original afterwards."""
import json, os, shutil, subprocess, sys, tempfile
from collections import Counter

MUTATIONS = [
    ("M01 cap check `>` -> `>=` (rejects at exactly the cap)", "if amount > threshold {", "if amount >= threshold {", "over-conservative"),
    ("M02 cap check allows cap+1", "if amount > threshold {", "if amount > threshold+1 {", "unsafe"),
    ("M03 obligation only above the full cap (skips the upper half)", "requiresObligation := amount > threshold/2", "requiresObligation := amount > threshold", "unsafe"),
    ("M04 obligation from exactly half (`>=`)", "requiresObligation := amount > threshold/2", "requiresObligation := amount >= threshold/2", "over-conservative"),
    ("M05 obligation threshold uses a third instead of a half", "requiresObligation := amount > threshold/2", "requiresObligation := amount > threshold/3", "over-conservative"),
    ("M06 second-approval flag ignored", "if requiresObligation && !secondApprovalObtained {", "if requiresObligation && false {", "unsafe"),
    ("M07 semBound attestation not checked", "if !semBound {", "if false {", "unsafe"),
    ("M08 evAssuranceOK attestation not checked", "if !evAssuranceOK {", "if false {", "unsafe"),
    ("M09 replay check removed", "if exists {", "if exists && false {", "unsafe"),
    ("M10 registry corrupted: gerente -> director", '"gerente":        "manager"', '"gerente":        "director"', "unsafe"),
    ("M11 policy data corrupted: clerk cap 100000", '"clerk":    10000', '"clerk":    100000', "unsafe"),
]


def run_suite(workdir, oracle_py, py, tag, cases_path, tmp):
    sut_out, orc_out, sm = (os.path.join(tmp, f"{tag}_{x}.json") for x in ("sut", "orc", "sum"))
    env = dict(os.environ, S3_CASES=cases_path, S3_OUT=sut_out)
    subprocess.run(["go", "test", "-mod=vendor", "./chaincode", "-run", "TestS3Independent"], cwd=workdir, env=env, check=True,
                   stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, timeout=900)
    subprocess.run([py, oracle_py, cases_path, orc_out], check=True, stdout=subprocess.DEVNULL)
    here = os.path.dirname(os.path.abspath(__file__))
    subprocess.run([sys.executable, os.path.join(here, "analyze_s3.py"), cases_path, sut_out, orc_out, sm], check=True, stdout=subprocess.DEVNULL)
    return json.load(open(sm))["full"]["overall"]


def main():
    workdir, original, oracle_py, py, out = sys.argv[1:6]
    suites = dict(a.split("=", 1) for a in sys.argv[6:])
    target = os.path.join(workdir, "chaincode", "authchaincode.go")
    src = open(original).read()
    tmp = tempfile.mkdtemp()
    results = []
    try:
        for name, old, new, kind in [("M00 unmodified (control)", None, None, "control")] + MUTATIONS:
            code = src
            if old is not None:
                if old not in src:
                    results.append({"mutation": name, "error": "target string not found"})
                    continue
                code = src.replace(old, new)
            open(target, "w").write(code)
            row = {"mutation": name, "kind": kind}
            for tag, cp in suites.items():
                try:
                    ov = run_suite(workdir, oracle_py, py, tag, cp, tmp)
                    row[tag] = dict(ov)
                    row[tag + "_detected"] = any(k != "match" for k in ov)
                except subprocess.CalledProcessError as e:
                    row[tag] = "build/run failed"
                    row[tag + "_detected"] = None
            results.append(row)
            print(name, {t: row.get(t + "_detected") for t in suites})
    finally:
        open(target, "w").write(src)
    json.dump(results, open(out, "w"), indent=1)


if __name__ == "__main__":
    if len(sys.argv) < 7:
        sys.exit(__doc__)
    main()
