#!/usr/bin/env python3
"""
qD2: a second, genuinely independent administrative domain (AWS) co-signs the
obligation-discharge coverage check that S11b (same day) closed for ONE domain only.
See chaincode/q_d_domain_attestation.go's header comment for the full design rationale.

This script runs REAL transactions against the real authchannel (peer chaincode invoke,
waitForEvent, mined) plus a REAL SSH round-trip to a separate AWS EC2 instance holding its own
secp256k1 key (generated there, never transmitted) to show:
  (a) Genuine agreement: real CONDITIONAL -> real DischargeObligation -> AWS independently
      re-derives coverage=true from the real decision content and signs -> SAFE.
  (b) Forged AWS signature: the SAME genuine request, but signed by a freshly generated
      "attacker" key that is NOT the registered AWS domain authority -> REJECT (signer mismatch).
      This is the core q_D security property: controlling the main chain's own state is not
      enough to forge the second domain's agreement.
  (c) Independent-reimplementation divergence probe: a boundary case (amount exactly at the
      role's threshold) checked against BOTH the chaincode's own logic and AWS's independent
      re-implementation, to see whether two independently-written implementations agree at the
      edge. Reported honestly whichever way it comes out.

Usage: python3 run.py
Depends on evidence/s8_population/s8_run.py (fixtures, real header+proofs, peer_invoke helpers)
and evidence/s11_behavior_safety/run.py's v7obl_verified_args-style argument building.
Run s8_run.py's setup() (SetClock + fresh header fetch) before this if besu_state_s8.json is
more than ~a few hours old.
"""
import importlib.util, json, os, subprocess, time

HERE = os.path.dirname(os.path.abspath(__file__))
S8 = os.path.join(HERE, "..", "s8_population")
spec = importlib.util.spec_from_file_location("s8_run", os.path.join(S8, "s8_run.py"))
m = importlib.util.module_from_spec(spec)
spec.loader.exec_module(m)

STAMP = str(int(time.time()))
OUT = os.path.join(HERE, "qd2_result.json")
AWS_HOST = "ec2-user@<redacted-ip>"
AWS_KEY = os.path.expanduser("~/.ssh/authcc-domain2-aws-key.pem")
AWS_DOMAIN_AUTHORITY = "0x6ad6b3ee6b4db60c28eb9de1b73945371ffcc063"

N0 = m.SAT[0]
LEGIT_AMOUNT = 300000  # manager threshold 500000, requiresObligation needs amount > 250000
THRESHOLD_AMOUNT = 500000  # exactly at the manager role's threshold -- boundary probe for case (c)


def v7_domain_args(did, n, profile, mechanism, ref, witness, aws_passed, aws_sig, **kw):
    return m.v7_args(did, n, **kw) + [profile, mechanism, ref, str(bool(witness)).lower(), str(bool(aws_passed)).lower(), aws_sig]


def aws_verify_sign(decision_id, obligation_discharge_ref, source_domain, role, amount,
                     obligation_profile, mapped_mechanism, enforcement_witness_verified,
                     referenced_record, forged_key_hex=None):
    """Real SSH round-trip to the AWS box: scp the request, run qd2aws verify-sign there
    (independent re-implementation, never calls this project's Go code), capture its signed
    JSON result. If forged_key_hex is given, instead signs locally with a non-AWS key to
    produce a deliberately forged attestation for the adversarial case (b)."""
    req = {
        "decisionId": decision_id, "obligationDischargeRef": obligation_discharge_ref,
        "sourceDomain": source_domain, "role": role, "amount": amount,
        "obligationProfile": obligation_profile, "mappedMechanism": mapped_mechanism,
        "enforcementWitnessVerified": enforcement_witness_verified,
        "referencedRecord": referenced_record,
    }
    req_path = f"/tmp/qd2_req_{decision_id}.json"
    with open(req_path, "w") as f:
        json.dump(req, f)
    remote_path = f"/tmp/qd2_req_{decision_id}.json"
    subprocess.run(["scp", "-i", AWS_KEY, req_path, f"{AWS_HOST}:{remote_path}"],
                    capture_output=True, text=True, check=True)
    r = subprocess.run(
        ["ssh", "-i", AWS_KEY, AWS_HOST,
         f"cd ~/qd2_aws && ./qd2aws verify-sign < {remote_path}"],
        capture_output=True, text=True, timeout=30)
    out = r.stdout.strip()
    result = json.loads(out)
    os.remove(req_path)
    return result


def run():
    log = {"stamp": STAMP, "chaincodeVersion": "authcc_2.37/seq49", "fixture": N0,
           "heightBefore": m.height(), "awsDomainAuthority": AWS_DOMAIN_AUTHORITY}

    # --- (a) Genuine cross-domain agreement ---
    did_cond = f"qd2-a-cond-{STAMP}"
    r_cond = m.do(("A1_submitdecisionfull_conditional", "SubmitDecisionFull",
                   [did_cond, "besu-mainnet", "manager", str(LEGIT_AMOUNT), str(m.LIMIT), "true", "true", "false"], {}))
    r_discharge = m.do(("A2_real_discharge", "DischargeObligation", [did_cond], {}))

    did_a = f"qd2-a-domainverified-{STAMP}"
    aws_result_a = aws_verify_sign(
        did_a, did_cond, "besu-mainnet", "manager", LEGIT_AMOUNT, "audit", "audit", True,
        {"decisionId": did_cond, "obligationDischarged": True, "sourceDomain": "besu-mainnet",
         "role": "manager", "amount": LEGIT_AMOUNT})
    r_a = m.do(("A3_both_domains_agree", "SubmitDecisionTrustlessV7ObligationDomainVerified",
               v7_domain_args(did_a, N0, "audit", "audit", did_cond, True,
                               aws_result_a["passed"], aws_result_a["compactSig"], amount=LEGIT_AMOUNT), {}))
    log["a_genuine_agreement"] = {
        "conditional": r_cond, "real_discharge": r_discharge,
        "aws_independent_result": aws_result_a, "submit": r_a,
        "both_domains_safe": r_a.get("committed") == "SAFE",
    }
    json.dump(log, open(OUT, "w"), indent=2)

    # --- (b) Forged AWS signature (wrong key, not the registered AWS domain authority) ---
    did_b = f"qd2-b-forged-{STAMP}"
    forge_priv_hex = subprocess.run(["openssl", "rand", "-hex", "32"], capture_output=True, text=True).stdout.strip()
    forge_r = subprocess.run(
        ["ssh", "-i", AWS_KEY, AWS_HOST,
         f"rm -rf /tmp/forge_home && mkdir -p /tmp/forge_home && "
         f"printf '%s' '{forge_priv_hex}' > /tmp/forge_home/.qd2_domain_key && "
         f"chmod 600 /tmp/forge_home/.qd2_domain_key"],
        capture_output=True, text=True)
    if forge_r.returncode != 0:
        raise RuntimeError(f"forged-key setup failed: {forge_r.stdout} {forge_r.stderr}")
    req_b = {
        "decisionId": did_b, "obligationDischargeRef": did_cond, "sourceDomain": "besu-mainnet",
        "role": "manager", "amount": LEGIT_AMOUNT, "obligationProfile": "audit", "mappedMechanism": "audit",
        "enforcementWitnessVerified": True,
        "referencedRecord": {"decisionId": did_cond, "obligationDischarged": True,
                              "sourceDomain": "besu-mainnet", "role": "manager", "amount": LEGIT_AMOUNT},
    }
    req_b_path = f"/tmp/qd2_req_{did_b}.json"
    with open(req_b_path, "w") as f:
        json.dump(req_b, f)
    subprocess.run(["scp", "-i", AWS_KEY, req_b_path, f"{AWS_HOST}:/tmp/qd2_req_{did_b}.json"], check=True, capture_output=True)
    forge_sign = subprocess.run(
        ["ssh", "-i", AWS_KEY, AWS_HOST,
         f"cd ~/qd2_aws && HOME=/tmp/forge_home ./qd2aws verify-sign < /tmp/qd2_req_{did_b}.json"],
        capture_output=True, text=True)
    aws_result_b = json.loads(forge_sign.stdout.strip())
    os.remove(req_b_path)

    r_b = m.do(("B_forged_signature_rejected", "SubmitDecisionTrustlessV7ObligationDomainVerified",
               v7_domain_args(did_b, N0, "audit", "audit", did_cond, True,
                               True, aws_result_b["compactSig"], amount=LEGIT_AMOUNT), {}))
    log["b_forged_signature"] = {
        "forged_signer": aws_result_b["awsSigner"], "real_aws_authority": AWS_DOMAIN_AUTHORITY,
        "submit": r_b, "attack_rejected": r_b.get("committed") == "REJECT",
    }
    json.dump(log, open(OUT, "w"), indent=2)

    # --- (c) Boundary probe: amount exactly at threshold, checked both sides ---
    did_c_cond = f"qd2-c-cond-{STAMP}"
    r_c_cond = m.do(("C1_boundary_conditional", "SubmitDecisionFull",
                     [did_c_cond, "besu-mainnet", "manager", str(THRESHOLD_AMOUNT), str(m.LIMIT), "true", "true", "false"], {}))
    r_c_discharge = m.do(("C2_boundary_discharge", "DischargeObligation", [did_c_cond], {}))
    did_c = f"qd2-c-domainverified-{STAMP}"
    aws_result_c = aws_verify_sign(
        did_c, did_c_cond, "besu-mainnet", "manager", THRESHOLD_AMOUNT, "audit", "audit", True,
        {"decisionId": did_c_cond, "obligationDischarged": True, "sourceDomain": "besu-mainnet",
         "role": "manager", "amount": THRESHOLD_AMOUNT})
    r_c = m.do(("C3_boundary_both_domains", "SubmitDecisionTrustlessV7ObligationDomainVerified",
               v7_domain_args(did_c, N0, "audit", "audit", did_c_cond, True,
                               aws_result_c["passed"], aws_result_c["compactSig"], amount=THRESHOLD_AMOUNT), {}))
    log["c_boundary_probe"] = {
        "amount": THRESHOLD_AMOUNT, "conditional": r_c_cond, "real_discharge": r_c_discharge,
        "aws_independent_passed": aws_result_c["passed"], "chain_committed": r_c.get("committed"),
        "divergence_found": (r_c.get("committed") == "SAFE") != aws_result_c["passed"],
    }

    time.sleep(3)
    log["heightAfter"] = m.height()
    json.dump(log, open(OUT, "w"), indent=2)
    print(json.dumps(log, indent=2))


if __name__ == "__main__":
    run()
