#!/usr/bin/env python3
"""
Closes the gap found 2026-10-13 while redesigning the independent-verification artifact
(see plan file "Ke hoach moi (2026-10-13)"): S5 (obligation stripped) and S6 (obligation
weakened) and S7 (obligation laundered) already exist as REAL on-chain transactions, but
only on the weaker, non-proof-bound SubmitDecisionObligationV2 (e3_s5s7_s15s16_onchain_2026-
09-24) -- never on the real Trustless/V7 family that requires a genuine ZK proof + genuine
Besu header. This script runs all 3 for real on SubmitDecisionTrustlessV7ObligationVerifiedDischarge,
following the exact same pattern as evidence/s11_behavior_safety/run.py's case C (real
SubmitDecisionFull CONDITIONAL -> real DischargeObligation -> patched entry point referencing
that real discharge), varying ONLY the obligationProfile/mappedMechanism/enforcementWitnessVerified
fields so each case isolates exactly one failure mode:
  S5: profile "combined" (required=4), mechanism ""     (stripped -- mapped to nothing)
  S6: profile "combined" (required=4), mechanism "notification" (weakened -- too weak)
  S7: profile "audit"    (required=2), mechanism "audit" (sufficient), witness=False (laundered)
Each case's own discharge record is genuinely created+discharged first (rules 3a/3b/3c in
dac_ta_quy_tac_nghia_vu.md all pass), isolating the REJECT to the specific rule under test.

Usage: python3 run_s5s6s7_real.py
Depends on evidence/s8_population/s8_run.py (fixtures, real header+proofs, peer_invoke helpers)
-- run `python3 ../s8_population/s8_run.py setup` first if besu_state_s8.json is stale.
"""
import importlib.util, json, os, time

HERE = os.path.dirname(os.path.abspath(__file__))
S8 = os.path.join(HERE, "..", "s8_population")
spec = importlib.util.spec_from_file_location("s8_run", os.path.join(S8, "s8_run.py"))
m = importlib.util.module_from_spec(spec)
spec.loader.exec_module(m)

STAMP = str(int(time.time()))
OUT = os.path.join(HERE, "s5s6s7_result.json")

N0 = m.SAT[0]


def v7obl_verified_args(did, n, profile, mechanism, ref, witness, **kw):
    return m.v7_args(did, n, **kw) + [profile, mechanism, ref, str(bool(witness)).lower()]


def make_case(key, amount, profile, mechanism, witness):
    did_full = f"qd3-{key}-full-{STAMP}"
    r_full = m.do((f"{key}_submitdecisionfull_conditional", "SubmitDecisionFull",
                   [did_full, "besu-mainnet", "manager", str(amount), str(m.LIMIT), "true", "true", "false"], {}))
    r_discharge = m.do((f"{key}_real_discharge", "DischargeObligation", [did_full], {}))
    did_verified = f"qd3-{key}-verified-{STAMP}"
    r_verified = m.do((f"{key}_v7oblvd_{profile}_{mechanism or 'empty'}_w{witness}",
                       "SubmitDecisionTrustlessV7ObligationVerifiedDischarge",
                       v7obl_verified_args(did_verified, N0, profile, mechanism, did_full, witness, amount=amount), {}))
    decision_final = m.read_decision(did_verified)
    return {
        "amount": amount, "obligationProfile": profile, "mappedMechanism": mechanism,
        "enforcementWitnessVerified": witness,
        "conditional": r_full, "real_discharge": r_discharge, "verified_submit": r_verified,
        "decisionId_full": did_full, "decisionId_verified": did_verified,
        "readDecision_verified": decision_final,
        "conditional_ok": r_full.get("committed") == "CONDITIONAL",
        "discharge_ok": r_discharge.get("committed") in ("EXACT", "SAFE"),
    }


def run():
    log = {"stamp": STAMP, "fixture": N0, "heightBefore": m.height()}

    log["S5_stripped"] = make_case("s5", 350000, "combined", "", True)
    log["S6_weakened"] = make_case("s6", 360000, "combined", "notification", True)
    log["S7_laundered"] = make_case("s7", 370000, "audit", "audit", False)

    time.sleep(3)
    log["heightAfter"] = m.height()
    json.dump(log, open(OUT, "w"), indent=2)
    print(json.dumps(log, indent=2))


if __name__ == "__main__":
    run()
