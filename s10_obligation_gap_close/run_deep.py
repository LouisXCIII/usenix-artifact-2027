#!/usr/bin/env python3
"""
S10 deep dive: scale case B and case C of run.py from n=1 to n=30 real, satisfiable, committed
fixtures (SAT[:30], deterministic subset of the 113-fixture S8 population), and cover all three
formal obligation-defect modes (S5 stripped / S6 weakened / S7 laundered) instead of only S5, plus
all three established proof-chain negative-control categories (N1 substitution / N2 tamper /
N3 forged featureHash) instead of only the tamper one. This is run AFTER run.py (n=1 sanity pass)
to get a real, reportable Clopper-Pearson-style upper bound instead of a single anecdote, matching
the n=30 convention already used elsewhere in this project (e.g. the B0/B2/B3 real-latency runs).

  B_obligation_sweep: for each of 30 fixtures, 1 genuine SubmitDecisionTrustlessV7 call (baseline,
    always expected SAFE, confirming V7 is blind to obligation regardless of what the obligation
    sweep below would say) + 4 SubmitDecisionTrustlessV7Obligation calls with the SAME genuine
    proof/header: healthy (expect SAFE), S5 stripped (expect REJECT), S6 weakened (expect REJECT),
    S7 laundered (expect REJECT). 30*(1+4) = 150 real transactions.

  C_proof_robustness_sweep: for each of 30 fixtures, 3 SubmitDecisionTrustlessV7Obligation calls,
    each with GENUINELY GOOD obligation parameters (audit/audit/discharged/witnessed) but an
    attacked proof side -- N1 (another fixture's real valid proof substituted onto this codeHash),
    N2 (this fixture's own proof with one digit tampered), N3 (this fixture's proof but another
    fixture's featureHash substituted into the public signal) -- all expected REJECT regardless of
    how good the obligation story is. 30*3 = 90 real transactions.

  C_claim_trust_role_sweep: the n=1 structural finding from run.py (SubmitDecisionObligationV2 is
    fooled by a bare semBound=true claim with zero proof attached, because the function has no
    proof parameter at all) repeated across all 3 real compiled roles at their own threshold caps,
    to show the claim-trust gap is a structural property of the function signature, not a
    role-specific coincidence. 3 real transactions.

Usage: python3 run_deep.py   (run AFTER run.py; reuses its fresh header/proof state, does not
re-fetch; if LedgerClock has gone stale again since run.py, re-run `s8_run.py setup` first)
"""
import importlib.util, json, os, time
from concurrent.futures import ThreadPoolExecutor

HERE = os.path.dirname(os.path.abspath(__file__))
S8 = os.path.join(HERE, "..", "s8_population")
spec = importlib.util.spec_from_file_location("s8_run", os.path.join(S8, "s8_run.py"))
m = importlib.util.module_from_spec(spec)
spec.loader.exec_module(m)

STAMP = str(int(time.time()))
OUT = os.path.join(HERE, "s10_deep_result.json")
FIXTURES = m.SAT[:30]
WORKERS = 4


def v7obl_args(did, n, profile, mechanism, discharged, witness, **kw):
    return m.v7_args(did, n, **kw) + [profile, mechanism, str(bool(discharged)).lower(), str(bool(witness)).lower()]


OBL_COND = {
    "healthy":      dict(profile="audit", mechanism="audit", discharged=True, witness=True),       # expect SAFE
    "S5_stripped":  dict(profile="dual_approval", mechanism="", discharged=False, witness=False),    # expect REJECT
    "S6_weakened":  dict(profile="combined", mechanism="notification", discharged=True, witness=True),  # expect REJECT
    "S7_laundered": dict(profile="audit", mechanism="audit", discharged=True, witness=False),        # expect REJECT
}
EXPECT_B = {"healthy": "SAFE", "S5_stripped": "REJECT", "S6_weakened": "REJECT", "S7_laundered": "REJECT"}
ROLE_THRESHOLD = {"manager": 500000, "director": 2000000, "clerk": 10000}


def batch(jobs, workers=WORKERS):
    with ThreadPoolExecutor(workers) as ex:
        return list(ex.map(m.do, jobs))


def cp_upper(n):
    return 1 - 0.05 ** (1.0 / n) if n else None


def run():
    log = {"stamp": STAMP, "chaincodeVersion": "authcc_2.31/seq43", "n_fixtures": len(FIXTURES), "heightBefore": m.height()}

    # -------- B: obligation-defect sweep (S5/S6/S7 + healthy control), SAME genuine proof each time --------
    jobs = []
    for n in FIXTURES:
        jobs.append(("v7_alone_baseline", "SubmitDecisionTrustlessV7", m.v7_args(f"s10d-b0-{n}-{STAMP}", n), {"fixture": n}))
        for cond, params in OBL_COND.items():
            did = f"s10d-b-{cond}-{n}-{STAMP}"
            jobs.append((cond, "SubmitDecisionTrustlessV7Obligation", v7obl_args(did, n, **params), {"fixture": n}))
    res = batch(jobs)
    bygroup = {}
    for r in res:
        bygroup.setdefault(r["kind"], []).append(r)
    log["B_obligation_sweep"] = bygroup
    summary_b = {}
    for cond, rows in bygroup.items():
        expect = "SAFE" if cond == "v7_alone_baseline" else EXPECT_B[cond]
        n = len(rows)
        match = sum(1 for r in rows if r["ok"] and r["committed"] == expect)
        wrong = sum(1 for r in rows if r["ok"] and r["committed"] != expect)
        errors = sum(1 for r in rows if not r["ok"])
        summary_b[cond] = {"n": n, "expect": expect, "match": match, "wrong": wrong, "errors": errors,
                            "cp95_upper_if_0_wrong": cp_upper(n) if wrong == 0 else None}
    log["B_summary"] = summary_b
    json.dump(log, open(OUT, "w"), indent=2)
    print("B done:", json.dumps(summary_b, indent=2), flush=True)

    # -------- C: proof-chain robustness sweep (N1/N2/N3), GENUINELY GOOD obligation each time --------
    jobs = []
    for i, n in enumerate(FIXTURES):
        o = FIXTURES[(i + 1) % len(FIXTURES)]
        jobs.append(("N1_proof_substitution", "SubmitDecisionTrustlessV7Obligation",
                     v7obl_args(f"s10d-c-n1-{n}-{STAMP}", o, **OBL_COND["healthy"], proof_from=n), {"fixture": n, "other": o}))
        jobs.append(("N2_tampered_proof", "SubmitDecisionTrustlessV7Obligation",
                     v7obl_args(f"s10d-c-n2-{n}-{STAMP}", n, **OBL_COND["healthy"], tamper=True), {"fixture": n}))
        jobs.append(("N3_forged_featureHash", "SubmitDecisionTrustlessV7Obligation",
                     v7obl_args(f"s10d-c-n3-{n}-{STAMP}", n, **OBL_COND["healthy"], pub_feature_from=o), {"fixture": n, "other": o}))
    res = batch(jobs)
    bygroup_c = {}
    for r in res:
        bygroup_c.setdefault(r["kind"], []).append(r)
    log["C_proof_robustness_sweep"] = bygroup_c
    summary_c = {}
    for cat, rows in bygroup_c.items():
        n = len(rows)
        accepted = sum(1 for r in rows if r["ok"] and r["committed"] in m.ACCEPT)
        rejected = sum(1 for r in rows if r["ok"] and r["committed"] not in m.ACCEPT)
        errors = sum(1 for r in rows if not r["ok"])
        summary_c[cat] = {"n": n, "accepted": accepted, "rejected": rejected, "errors": errors,
                           "cp95_upper_if_0_accepted": cp_upper(n) if accepted == 0 else None}
    log["C_summary"] = summary_c
    json.dump(log, open(OUT, "w"), indent=2)
    print("C done:", json.dumps(summary_c, indent=2), flush=True)

    # -------- C (claim-trust generality): ObligationV2 fooled across all 3 real roles at their own cap --------
    jobs = []
    for role, threshold in ROLE_THRESHOLD.items():
        did = f"s10d-c-roletrust-{role}-{STAMP}"
        jobs.append(("C_claim_trust_role", "SubmitDecisionObligationV2",
                     [did, "besu-mainnet", role, str(threshold), str(threshold * 2),
                      "true", "true", "audit", "audit", "true", "true"], {"role": role, "threshold": threshold}))
    res = batch(jobs, workers=3)
    log["C_claim_trust_role_sweep"] = res
    fooled = sum(1 for r in res if r["ok"] and r["committed"] in m.ACCEPT)
    log["C_claim_trust_role_summary"] = {"n": len(res), "fooled": fooled}

    time.sleep(3)
    log["heightAfter"] = m.height()
    json.dump(log, open(OUT, "w"), indent=2)
    print("claim-trust role sweep:", json.dumps(log["C_claim_trust_role_summary"], indent=2))
    print("height", log["heightBefore"], "->", log["heightAfter"])


if __name__ == "__main__":
    run()
