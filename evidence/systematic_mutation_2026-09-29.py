#!/usr/bin/env python3
"""Systematic mutation testing of e1_classifier.py's classify() function.

Extends the hand-picked 4-defect mutation-testing pass (e1_mutation_check.py)
to a full, mechanical operator-mutation sweep: every comparison operator,
boolean operator (and/or), and unary "not" in classify()'s AST is flipped one
at a time to produce a mutant. Each mutant is scored against the same 2000
generated cases and the same independent ground-truth oracle used by the main
formal benchmark (e1_run_campaign.py) -- a mutant is "killed" if it disagrees
with ground truth on at least one case (nonzero FAR, FRR, or accuracy drop),
exactly the detection convention e1_mutation_check.py already established.

No AI-generated mutants or judgment: mutation operators are the standard,
textbook set (Lt<->LtE, Gt<->GtE, Eq<->NotEq, In<->NotIn, And<->Or, remove
unary Not), applied mechanically via Python's ast module to one node at a
time. Not imported by any other file; standalone, like e1_mutation_check.py.

Run: ~/pqc-hardhat/experiments/portability/.venv_e1/bin/python \
     systematic_mutation.py
"""
import ast
import copy
import importlib.util
import json
import sys
import os

PORT_DIR = "~/pqc-hardhat/experiments/portability"
sys.path.insert(0, PORT_DIR)

from e1_generator import generate_cases
from e1_ground_truth import ground_truth_class

CLASSIFIER_SRC = open(os.path.join(PORT_DIR, "e1_classifier.py")).read()

CMP_FLIP = {
    ast.Lt: ast.GtE, ast.GtE: ast.Lt,
    ast.LtE: ast.Gt, ast.Gt: ast.LtE,
    ast.Eq: ast.NotEq, ast.NotEq: ast.Eq,
    ast.In: ast.NotIn, ast.NotIn: ast.In,
}
BOOL_FLIP = {ast.And: ast.Or, ast.Or: ast.And}


class MutantCollector(ast.NodeVisitor):
    """Finds every mutable node inside the classify() function body and
    records (node_id, description, apply_fn) without mutating the shared
    tree; mutants are produced by deep-copying the tree per mutation."""

    def __init__(self):
        self.sites = []  # list of (path_ids consumed via id(), description)

    def visit_FunctionDef(self, node):
        if node.name != "classify":
            self.generic_visit(node)
            return
        for child in ast.walk(node):
            if isinstance(child, ast.Compare) and len(child.ops) == 1:
                op = child.ops[0]
                if type(op) in CMP_FLIP:
                    self.sites.append(("compare", id(child), f"{ast.dump(op)}"))
            elif isinstance(child, ast.BoolOp):
                if type(child.op) in BOOL_FLIP:
                    self.sites.append(("boolop", id(child), f"{type(child.op).__name__}"))
            elif isinstance(child, ast.UnaryOp) and isinstance(child.op, ast.Not):
                self.sites.append(("not", id(child), "Not"))


def build_mutant(mutation_kind, target_id):
    tree = ast.parse(CLASSIFIER_SRC)
    applied = False
    for node in ast.walk(tree):
        if mutation_kind == "compare" and isinstance(node, ast.Compare) and id(node) == target_id:
            pass  # id() won't match across re-parses; handled by index-based approach below
    return tree


def collect_sites(tree):
    """Returns a flat list of mutable nodes (by object identity within THIS
    tree instance) so each can be targeted for exactly one mutation."""
    sites = []
    for node in ast.walk(tree):
        found = False
        if isinstance(node, ast.FunctionDef) and node.name == "classify":
            for child in ast.walk(node):
                if isinstance(child, ast.Compare) and len(child.ops) == 1 and type(child.ops[0]) in CMP_FLIP:
                    sites.append(("compare", child))
                elif isinstance(child, ast.BoolOp) and type(child.op) in BOOL_FLIP:
                    sites.append(("boolop", child))
                elif isinstance(child, ast.UnaryOp) and isinstance(child.op, ast.Not):
                    sites.append(("not", child))
    return sites


def make_mutant_source(site_index):
    tree = ast.parse(CLASSIFIER_SRC)
    sites = collect_sites(tree)
    kind, node = sites[site_index]
    desc = ""
    if kind == "compare":
        old_op = node.ops[0]
        new_op_cls = CMP_FLIP[type(old_op)]
        desc = f"{type(old_op).__name__} -> {new_op_cls.__name__} (line {node.lineno})"
        node.ops[0] = new_op_cls()
    elif kind == "boolop":
        old_op = node.op
        new_op_cls = BOOL_FLIP[type(old_op)]
        desc = f"{type(old_op).__name__} -> {new_op_cls.__name__} (line {node.lineno})"
        node.op = new_op_cls()
    elif kind == "not":
        desc = f"remove unary Not (line {node.lineno})"
        # Replace the UnaryOp(Not, operand) node in its parent with the bare operand.
        # ast has no parent pointers by default; do a second pass to find and replace.
        replaced = _replace_not_node(tree, node)
        if not replaced:
            desc += " [FAILED to splice]"
    ast.fix_missing_locations(tree)
    return desc, ast.unparse(tree), kind


def _replace_not_node(tree, target):
    """Walk parent-aware to replace `target` (a UnaryOp Not) with its operand
    wherever it is referenced as a field or list element."""
    for node in ast.walk(tree):
        for field, value in ast.iter_fields(node):
            if value is target:
                setattr(node, field, target.operand)
                return True
            if isinstance(value, list):
                for i, item in enumerate(value):
                    if item is target:
                        value[i] = target.operand
                        return True
    return False


def count_sites():
    tree = ast.parse(CLASSIFIER_SRC)
    return len(collect_sites(tree))


def run_mutant(source_text, cases, gt):
    ns = {}
    try:
        exec(compile(source_text, "<mutant>", "exec"), ns)
    except Exception as e:
        return {"error": f"compile/exec failure: {e}"}
    classify_fn = ns["classify"]
    mism = 0
    far = 0
    frr = 0
    unsafe = {"CONDITIONAL", "REEVALUATE", "REJECT"}
    safe = {"EXACT", "SAFE"}
    for c, g in zip(cases, gt):
        try:
            pred, _ = classify_fn(c)
        except Exception as e:
            pred = f"__ERROR__:{e}"
        if pred != g:
            mism += 1
            if g in unsafe and pred in safe:
                far += 1
            if g in safe and pred in unsafe:
                frr += 1
    return {"mismatches": mism, "far_flips": far, "frr_flips": frr, "accuracy": 1 - mism / len(cases)}


def main():
    cases = generate_cases(n_per_combo=40, seed=20260923)
    gt = [ground_truth_class(c) for c in cases]
    n = count_sites()
    print(f"total mutation sites in classify(): {n}")
    results = []
    for i in range(n):
        desc, src, kind = make_mutant_source(i)
        res = run_mutant(src, cases, gt)
        res.update({"site_index": i, "kind": kind, "description": desc})
        killed = res.get("mismatches", 0) > 0 or "error" in res
        res["killed"] = killed
        results.append(res)
        print(f"[{i:02d}] {kind:8s} {desc:55s} killed={killed} mism={res.get('mismatches','?')}")

    killed_n = sum(1 for r in results if r["killed"])
    print()
    print(f"mutation score: {killed_n}/{n} = {killed_n/n*100:.1f}%")
    survivors = [r for r in results if not r["killed"]]
    print(f"survivors: {len(survivors)}")
    for s in survivors:
        print("  SURVIVOR:", s["kind"], s["description"])

    out = {
        "n_mutants": n,
        "killed": killed_n,
        "mutation_score": killed_n / n,
        "results": results,
    }
    json.dump(out, open("systematic_mutation_result.json", "w"), indent=1)
    print("saved systematic_mutation_result.json")


if __name__ == "__main__":
    main()
