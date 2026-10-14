#!/usr/bin/env python3
"""
Companion to run_s5s6s7_real.py (same 2026-10-13 plan). Produces the remaining 3 of the 6
real on-chain obligation-coverage scenarios needed for the Mu'c 1/Mu'c 3 independent-
verification artifact, all from this SAME fresh batch (avoids hunting block numbers across
older S11b/qd2 runs from different dates):
  clean:          genuine discharge, sufficient mechanism, witness=true -> SAFE
  confused_deputy: genuine discharge for amount X, but a DIFFERENT decision (amount Y)
                   references it -> REJECT (binding mismatch, S11b case D's pattern, redone fresh)
  reuse:          genuine discharge cited once (SAFE), cited a SECOND time by a different
                   decisionId for the SAME sourceDomain/role/amount -> second citation REJECT

Usage: python3 run_clean_cd_reuse_real.py
"""
import importlib.util, json, os, time

HERE = os.path.dirname(os.path.abspath(__file__))
S8 = os.path.join(HERE, "..", "s8_population")
spec = importlib.util.spec_from_file_location("s8_run", os.path.join(S8, "s8_run.py"))
m = importlib.util.module_from_spec(spec)
spec.loader.exec_module(m)

STAMP = str(int(time.time()))
OUT = os.path.join(HERE, "clean_cd_reuse_result.json")
N0 = m.SAT[0]


def v7obl_verified_args(did, n, profile, mechanism, ref, witness, **kw):
    return m.v7_args(did, n, **kw) + [profile, mechanism, ref, str(bool(witness)).lower()]


def run():
    log = {"stamp": STAMP, "fixture": N0, "heightBefore": m.height()}

    # 1. Clean pass.
    amt_clean = 380000
    did_clean_full = f"qd3-clean-full-{STAMP}"
    r1 = m.do(("clean_conditional", "SubmitDecisionFull",
              [did_clean_full, "besu-mainnet", "manager", str(amt_clean), str(m.LIMIT), "true", "true", "false"], {}))
    r2 = m.do(("clean_discharge", "DischargeObligation", [did_clean_full], {}))
    did_clean_verified = f"qd3-clean-verified-{STAMP}"
    r3 = m.do(("clean_v7oblvd", "SubmitDecisionTrustlessV7ObligationVerifiedDischarge",
              v7obl_verified_args(did_clean_verified, N0, "audit", "audit", did_clean_full, True, amount=amt_clean), {}))
    log["clean"] = {
        "amount": amt_clean, "conditional": r1, "real_discharge": r2, "verified_submit": r3,
        "decisionId_full": did_clean_full, "decisionId_verified": did_clean_verified,
        "readDecision_verified": m.read_decision(did_clean_verified),
    }

    # 2. Confused-deputy: discharge genuinely created for amt_cd_real, but a DIFFERENT
    #    decision (amt_cd_wrong) cites it.
    amt_cd_real, amt_cd_wrong = 390000, 395000
    did_cd_full = f"qd3-cd-full-{STAMP}"
    r4 = m.do(("cd_conditional", "SubmitDecisionFull",
              [did_cd_full, "besu-mainnet", "manager", str(amt_cd_real), str(m.LIMIT), "true", "true", "false"], {}))
    r5 = m.do(("cd_discharge", "DischargeObligation", [did_cd_full], {}))
    did_cd_verified = f"qd3-cd-verified-{STAMP}"
    r6 = m.do(("cd_v7oblvd_crosswire", "SubmitDecisionTrustlessV7ObligationVerifiedDischarge",
              v7obl_verified_args(did_cd_verified, N0, "audit", "audit", did_cd_full, True, amount=amt_cd_wrong), {}))
    log["confused_deputy"] = {
        "amount_real_discharge": amt_cd_real, "amount_claimed": amt_cd_wrong,
        "conditional": r4, "real_discharge": r5, "verified_submit": r6,
        "decisionId_full": did_cd_full, "decisionId_verified": did_cd_verified,
        "readDecision_verified": m.read_decision(did_cd_verified),
    }

    # 3. Reuse: genuine discharge cited once (expect SAFE), then a second, DIFFERENT
    #    decisionId cites the SAME discharge for the SAME sourceDomain/role/amount.
    amt_reuse = 400000
    did_reuse_full = f"qd3-reuse-full-{STAMP}"
    r7 = m.do(("reuse_conditional", "SubmitDecisionFull",
              [did_reuse_full, "besu-mainnet", "manager", str(amt_reuse), str(m.LIMIT), "true", "true", "false"], {}))
    r8 = m.do(("reuse_discharge", "DischargeObligation", [did_reuse_full], {}))
    did_reuse_first = f"qd3-reuse-first-{STAMP}"
    r9 = m.do(("reuse_first_citation", "SubmitDecisionTrustlessV7ObligationVerifiedDischarge",
              v7obl_verified_args(did_reuse_first, N0, "audit", "audit", did_reuse_full, True, amount=amt_reuse), {}))
    did_reuse_second = f"qd3-reuse-second-{STAMP}"
    r10 = m.do(("reuse_second_citation_should_reject", "SubmitDecisionTrustlessV7ObligationVerifiedDischarge",
               v7obl_verified_args(did_reuse_second, N0, "audit", "audit", did_reuse_full, True, amount=amt_reuse), {}))
    log["reuse"] = {
        "amount": amt_reuse, "conditional": r7, "real_discharge": r8,
        "first_citation": r9, "second_citation": r10,
        "decisionId_full": did_reuse_full,
        "decisionId_first": did_reuse_first, "decisionId_second": did_reuse_second,
        "readDecision_first": m.read_decision(did_reuse_first),
        "readDecision_second": m.read_decision(did_reuse_second),
    }

    time.sleep(3)
    log["heightAfter"] = m.height()
    json.dump(log, open(OUT, "w"), indent=2)
    print(json.dumps(log, indent=2))


if __name__ == "__main__":
    run()
