#!/usr/bin/env python3
"""Mechanical hypothesis-necessity check: for every named Hypothesis in both Coq files,
comment it out, try to recompile with coqc, and record whether the proof still succeeds
(BAD -- hypothesis is redundant/unused) or fails (GOOD -- hypothesis is load-bearing).
User's exact request: remove each hypothesis individually, recompile, every removal should
break the proof; any that doesn't is either dead weight or needs a stated reason to keep."""
import json
import os
import re
import shutil
import subprocess

COQDIR = "~/paper_backups/submit_paper2/evidence/coq"
WORKDIR = os.path.dirname(os.path.abspath(__file__))
FILES = ["ADPRelationalSoundness.v", "ADPMultiHopComposition.v"]


def find_hypothesis_blocks(lines):
    """Return list of (name, start_idx, end_idx_inclusive) for each `Hypothesis <name> : ... .`
    block, where start/end are 0-based line indices into `lines`."""
    blocks = []
    i = 0
    while i < len(lines):
        m = re.match(r'^Hypothesis\s+(\w+)\s*:', lines[i])
        if m:
            name = m.group(1)
            start = i
            j = i
            while j < len(lines) and not lines[j].rstrip().endswith('.'):
                j += 1
            blocks.append((name, start, j))
            i = j + 1
        else:
            i += 1
    return blocks


def try_remove(src_path, start, end, out_path):
    with open(src_path) as f:
        lines = f.readlines()
    for k in range(start, end + 1):
        lines[k] = "(* REMOVED-FOR-NECESSITY-CHECK: " + lines[k].rstrip('\n') + " *)\n"
    with open(out_path, "w") as f:
        f.writelines(lines)


def compile_ok(path):
    workdir = os.path.dirname(path)
    name = os.path.basename(path)
    r = subprocess.run(["coqc", name], cwd=workdir, capture_output=True, text=True, timeout=60)
    return r.returncode == 0, r.stdout + r.stderr


def main():
    results = {}
    for fname in FILES:
        src_path = os.path.join(COQDIR, fname)
        with open(src_path) as f:
            lines = f.readlines()
        blocks = find_hypothesis_blocks(lines)
        file_results = []
        for name, start, end in blocks:
            testdir = os.path.join(WORKDIR, f"test_{fname.replace('.v','')}_{name}")
            os.makedirs(testdir, exist_ok=True)
            out_path = os.path.join(testdir, fname)
            try_remove(src_path, start, end, out_path)
            # Both files are self-contained (ADPMultiHopComposition.v does not Require
            # Import the other), so copying just the one file under test suffices.
            ok, log = compile_ok(out_path)
            verdict = "REDUNDANT (compiled without it -- BAD)" if ok else "LOAD-BEARING (broke proof -- expected)"
            file_results.append({"hypothesis": name, "lines": f"{start+1}-{end+1}",
                                  "compiled_without_it": ok, "verdict": verdict})
            print(f"{fname}: {name:30s} -> {verdict}")
            shutil.rmtree(testdir)
        results[fname] = file_results
    json.dump(results, open(os.path.join(WORKDIR, "necessity_check_results.json"), "w"), indent=2)
    print("\nSummary:")
    for fname, rs in results.items():
        bad = [r["hypothesis"] for r in rs if r["compiled_without_it"]]
        print(f"  {fname}: {len(rs)} hypotheses checked, {len(bad)} redundant: {bad}")


if __name__ == "__main__":
    main()
