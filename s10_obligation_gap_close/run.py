#!/usr/bin/env python3
"""
S10: real on-chain demonstration that the 2026-09-29 S9 red-team finding is closed.

Finding (build_report.js Muc 8, Bullet 29/9): no deployed entry point verified BOTH the real
evidence chain (ZK proof + light-client header, SubmitDecisionTrustlessV7) AND the real
obligation chain (oblCoveredV2, SubmitDecisionObligationV2) in the same transaction. This script
runs REAL transactions (peer chaincode invoke, waitForEvent, mined) against the deployed authcc
chaincode (v2.31/seq43, which adds SubmitDecisionTrustlessV7Obligation -- additive, does not
change V7 or ObligationV2) to show:

  A. Agreement case: genuine proof + genuine obligation -> V7 alone and V7Obligation both SAFE
     (no regression on the happy path).
  B. S5/S6 divergence (the gap that mattered): SAME genuine proof, but the obligation is stripped
     (mappedMechanism=""). V7 alone has no obligation model at all and is SAFE (the real gap).
     V7Obligation correctly REJECTs. This is the empirical demonstration that combining both
     chains in one transaction catches an attack that passing either check alone would miss.
  C. Reverse divergence (ObligationV2's claim-trust gap): ObligationV2 takes semBound as a
     caller-supplied claim with NO real proof attached, and is fooled into SAFE/EXACT.
     V7Obligation, given a TAMPERED (invalid) proof but otherwise-good obligation parameters,
     correctly REJECTs -- it cannot be fooled by a bare claim, because semBound is re-derived
     from a real proof every time.
  D. Replay sanity: re-submitting the same decisionId to V7Obligation is rejected and the
     original record is unchanged (matches the existing replay-rejection behaviour of every
     other Submit* entry point).

Usage: python3 run.py
Depends on evidence/s8_population/s8_run.py (fixtures, real header+proofs, peer_invoke helpers)
and evidence/s8_population/besu_state_s8.json (produced by `s8_run.py setup`; re-run fresh on
2026-10-04 after the post-reboot ledger-clock resync, so the header/proofs used here are current).
"""
import importlib.util, json, os, time

HERE = os.path.dirname(os.path.abspath(__file__))
S8 = os.path.join(HERE, "..", "s8_population")
spec = importlib.util.spec_from_file_location("s8_run", os.path.join(S8, "s8_run.py"))
m = importlib.util.module_from_spec(spec)
spec.loader.exec_module(m)

STAMP = str(int(time.time()))
OUT = os.path.join(HERE, "s10_result.json")

N0 = m.SAT[0]  # one real, distinct satisfiable fixture contract


def v7obl_args(did, n, profile, mechanism, discharged, witness, **kw):
    return m.v7_args(did, n, **kw) + [profile, mechanism, str(bool(discharged)).lower(), str(bool(witness)).lower()]


def run():
    log = {"stamp": STAMP, "chaincodeVersion": "authcc_2.31/seq43", "fixture": N0, "heightBefore": m.height()}

    # A. agreement: genuine proof + genuine obligation
    did_a1 = f"s10-a1-v7-{STAMP}"
    did_a2 = f"s10-a2-v7obl-{STAMP}"
    r_a1 = m.do(("A1_v7_genuine", "SubmitDecisionTrustlessV7", m.v7_args(did_a1, N0), {}))
    r_a2 = m.do(("A2_v7obl_genuine", "SubmitDecisionTrustlessV7Obligation",
                 v7obl_args(did_a2, N0, "audit", "audit", True, True), {}))
    log["A_agreement"] = {"v7_alone": r_a1, "v7_obligation": r_a2}

    # B. S5/S6 divergence: SAME genuine proof, obligation stripped
    did_b1 = f"s10-b1-v7-{STAMP}"
    did_b2 = f"s10-b2-v7obl-{STAMP}"
    r_b1 = m.do(("B1_v7_alone_blind_to_obligation", "SubmitDecisionTrustlessV7", m.v7_args(did_b1, N0), {}))
    r_b2 = m.do(("B2_v7obl_catches_stripped_obligation", "SubmitDecisionTrustlessV7Obligation",
                 v7obl_args(did_b2, N0, "dual_approval", "", False, False), {}))
    log["B_divergence_obligation_stripped"] = {
        "v7_alone": r_b1, "v7_obligation": r_b2,
        "gap_demonstrated": r_b1.get("committed") == "SAFE" and r_b2.get("committed") == "REJECT",
    }

    # C. reverse divergence: ObligationV2 trusts a bare semBound claim; V7Obligation needs a real proof
    did_c1 = f"s10-c1-oblv2-{STAMP}"
    did_c2 = f"s10-c2-v7obl-{STAMP}"
    r_c1 = m.do(("C1_obligationV2_trusts_fake_claim", "SubmitDecisionObligationV2",
                 [did_c1, "besu-mainnet", m.ROLE, str(m.AMOUNT), str(m.LIMIT),
                  "true", "true", "audit", "audit", "true", "true"], {}))
    r_c2 = m.do(("C2_v7obl_rejects_tampered_proof", "SubmitDecisionTrustlessV7Obligation",
                 v7obl_args(did_c2, N0, "audit", "audit", True, True, tamper=True), {}))
    log["C_divergence_claim_trust"] = {
        "obligationV2_fooled": r_c1, "v7_obligation_not_fooled": r_c2,
        "gap_demonstrated": r_c1.get("committed") in m.ACCEPT and r_c2.get("committed") == "REJECT",
    }

    # D. replay sanity on the new entry point
    before = m.read_decision(did_a2)
    r_d1 = m.do(("D1_replay_rejected", "SubmitDecisionTrustlessV7Obligation",
                 v7obl_args(did_a2, N0, "audit", "audit", True, True), {}))
    after = m.read_decision(did_a2)
    log["D_replay"] = {"attempt": r_d1, "recordUnchanged": before == after}

    time.sleep(3)
    log["heightAfter"] = m.height()
    json.dump(log, open(OUT, "w"), indent=2)
    print(json.dumps(log, indent=2))


if __name__ == "__main__":
    run()
