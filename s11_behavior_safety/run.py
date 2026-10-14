#!/usr/bin/env python3
"""
S11: real on-chain demonstration + fix of the "behavior-safety beyond policy-safety" gap
found 2026-10-08 (added to the paper's Table 8 the same day).

Finding (direct source reading of ~/fabric-network/authcc/chaincode/obligation_v2.go and
v7_obligation.go): SubmitDecisionObligationV2 and SubmitDecisionTrustlessV7Obligation both take
obligationDischarged / enforcementWitnessVerified as RAW CALLER-SUPPLIED booleans, fed straight
into oblCoveredV2 with no cross-check against any real on-chain discharge event. The only real
discharge mechanism, DischargeObligation (authchaincode.go:480), flips ONLY its own record's
ObligationDischarged field -- obligation_v2.go/v7_obligation.go never read that field back. A
caller can therefore claim obligationDischarged=true for a transfer whose obligation was never
discharged anywhere on-chain, and get classified SAFE on real semBound/evOK/role/amount alone:
policy-safe (every existing check passes) but not behavior-safe.

This script runs REAL transactions (peer chaincode invoke, waitForEvent, mined) against the real
authchannel to show:
  A. Exploit, reproduced live on the DEPLOYED, UNMODIFIED SubmitDecisionTrustlessV7Obligation
     (v2.35/seq47): genuine proof + genuine obligation-profile/mechanism, but
     obligationDischarged=true is a bare, unbacked claim (no real discharge event for this
     decision anywhere on-chain) -> SAFE. The vulnerability, live.
  B. Same scenario (same genuine proof, same obligationProfile/mechanism, same bare claim
     replaced by an EMPTY obligationDischargeRef) submitted to the NEW, patched
     SubmitDecisionTrustlessV7ObligationVerifiedDischarge -> REJECT. The fix closes it.
  C. Legitimate case: a real CONDITIONAL decision is created (SubmitDecisionFull) and genuinely
     discharged (DischargeObligation, a real state transition). A NEW decision for the SAME
     sourceDomain/role/amount references that real discharge via obligationDischargeRef on the
     patched entry point -> SAFE. The fix does not break the honest path.
  D. Confused-deputy / cross-wiring: the SAME real discharged record from C is referenced by a
     decision for a DIFFERENT amount -> REJECT (binding mismatch caught).
  E. Replay sanity on the new entry point.

Usage: python3 run.py
Depends on evidence/s8_population/s8_run.py (fixtures, real header+proofs, peer_invoke helpers).
Run `python3 ../s8_population/s8_run.py setup`-equivalent (SetClock + fresh header fetch) BEFORE
this script if the cached besu_state_s8.json is more than ~a few hours old.
"""
import importlib.util, json, os, time

HERE = os.path.dirname(os.path.abspath(__file__))
S8 = os.path.join(HERE, "..", "s8_population")
spec = importlib.util.spec_from_file_location("s8_run", os.path.join(S8, "s8_run.py"))
m = importlib.util.module_from_spec(spec)
spec.loader.exec_module(m)

STAMP = str(int(time.time()))
OUT = os.path.join(HERE, "s11_result.json")

N0 = m.SAT[0]  # one real, distinct satisfiable fixture contract (same choice S10 made)
LEGIT_AMOUNT = 300000  # manager threshold 500000, requiresObligation needs amount > 250000
WRONG_AMOUNT = 310000  # different amount -> binding mismatch for case D


def v7obl_verified_args(did, n, profile, mechanism, ref, witness, **kw):
    return m.v7_args(did, n, **kw) + [profile, mechanism, ref, str(bool(witness)).lower()]


def run():
    log = {"stamp": STAMP, "chaincodeVersion_before": "authcc_2.35/seq47", "fixture": N0, "heightBefore": m.height()}

    # A. Exploit reproduced live on the deployed, unmodified entry point.
    did_a = f"s11-a-v7obl-exploit-{STAMP}"
    r_a = m.do(("A_v7obl_trusts_bare_discharge_claim", "SubmitDecisionTrustlessV7Obligation",
               m.v7_args(did_a, N0) + ["audit", "audit", "true", "true"], {}))
    log["A_exploit_on_deployed_v7obligation"] = {
        "result": r_a,
        "gap_demonstrated": r_a.get("committed") in m.ACCEPT,
    }

    # B. Same scenario, patched entry point, no real discharge reference -> must REJECT.
    did_b = f"s11-b-v7oblvd-norebound-{STAMP}"
    r_b = m.do(("B_patched_rejects_unbacked_claim", "SubmitDecisionTrustlessV7ObligationVerifiedDischarge",
               v7obl_verified_args(did_b, N0, "audit", "audit", "", True), {}))
    log["B_patched_rejects_unverified"] = {
        "result": r_b,
        "fix_confirmed": r_b.get("committed") == "REJECT",
    }

    # C. Legitimate case: real CONDITIONAL -> real DischargeObligation -> patched entry point
    #    referencing it for the SAME transfer.
    did_c_full = f"s11-c-full-conditional-{STAMP}"
    r_c_full = m.do(("C1_submitdecisionfull_conditional", "SubmitDecisionFull",
                     [did_c_full, "besu-mainnet", "manager", str(LEGIT_AMOUNT), str(m.LIMIT), "true", "true", "false"], {}))
    r_c_discharge = m.do(("C2_real_discharge", "DischargeObligation", [did_c_full], {}))
    did_c_verified = f"s11-c-v7oblvd-legit-{STAMP}"
    r_c_verified = m.do(("C3_patched_accepts_real_discharge", "SubmitDecisionTrustlessV7ObligationVerifiedDischarge",
                        v7obl_verified_args(did_c_verified, N0, "audit", "audit", did_c_full, True, amount=LEGIT_AMOUNT), {}))
    log["C_legitimate_discharge_honored"] = {
        "conditional": r_c_full, "real_discharge": r_c_discharge, "verified_submit": r_c_verified,
        "honest_path_preserved": r_c_full.get("committed") == "CONDITIONAL"
        and r_c_discharge.get("committed") in ("EXACT", "SAFE")
        and r_c_verified.get("committed") == "SAFE",
    }

    # D. Confused-deputy: same real discharged record, but a DIFFERENT amount claimed here.
    did_d = f"s11-d-v7oblvd-crosswire-{STAMP}"
    r_d = m.do(("D_patched_rejects_crosswired_discharge", "SubmitDecisionTrustlessV7ObligationVerifiedDischarge",
               v7obl_verified_args(did_d, N0, "audit", "audit", did_c_full, True, amount=WRONG_AMOUNT), {}))
    log["D_crosswire_rejected"] = {
        "result": r_d,
        "binding_check_confirmed": r_d.get("committed") == "REJECT",
    }

    # E. Replay sanity on the new entry point.
    before = m.read_decision(did_b)
    r_e = m.do(("E_replay_rejected", "SubmitDecisionTrustlessV7ObligationVerifiedDischarge",
               v7obl_verified_args(did_b, N0, "audit", "audit", "", True), {}))
    after = m.read_decision(did_b)
    log["E_replay"] = {"attempt": r_e, "recordUnchanged": before == after}

    time.sleep(3)
    log["heightAfter"] = m.height()
    json.dump(log, open(OUT, "w"), indent=2)
    print(json.dumps(log, indent=2))


if __name__ == "__main__":
    run()
