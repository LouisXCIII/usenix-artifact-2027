#!/usr/bin/env python3
"""rp_checker_zk: the first genuinely non-trivial real R_P instance for this project's
checker (Tru 4 of the top-tier review's roadmap, 2026-10-11).

Background (why this file exists). rp_checker.py's appendix disclosure (evidence/
rp_checker_2026-10-10.md, and Manuscriptsp_m2_r3_usenix.tex around line 1592) states: "the
deployed system currently has no destination guard whose safety genuinely depends on an
unverified source-trust relation... a genuinely non-trivial real instance requires a guard
the destination cannot evaluate from its own state alone -- e.g. a zero-knowledge-proven
private source predicate the destination must accept via a verified mapping rather than
re-derive -- which the current workload does not yet have."

That statement is WRONG, found by re-reading authcc/chaincode/zkverify.go and featurebind.go
instead of only the RoleThreshold/obligation_v2.go fragments rp_checker.py already covered.
sec:semantics's own Phi_s = {mlScanPassed, approvalCount>=k} (line ~531) is established at the
source by running a private ML model over private features -- the destination never has those
features or weights. SubmitDecisionTrustlessV7 (the production entry point the paper's main
security claims rest on) never re-runs the scan; it only ever sees a Groth16 proof + public
signals, plus a Merkle-Patricia proof that one public signal (featureHash) matches an
independently-committed on-chain value, and accepts mlScanPassed's stand-in (semBound) only if
BOTH verify. That is precisely "a guard the destination cannot evaluate from its own state
alone... accepted via a verified mapping rather than re-derived."

Two sub-fragments, both real deployed production code, both formalized below:

  (A) SemBound proof-policy (zkverify.go, verifySemBoundProof): scanPassed=1 AND
      maxVulnClass=0 AND minMargin>=requiredMinMargin AND modelWeightsHash=pinned. R_Sigma
      here is the Groth16 relation itself (binds public signals to the hidden witness x,W,b);
      treated as a trusted oracle in this Z3 encoding exactly as the rest of this project
      treats the Coq-mechanization-vs-chaincode gap -- this module does not re-verify the
      pairing (zkverify_test.go already does that, in Go, against real proof/key files); it
      checks the POLICY layer applied to whatever public signals a verified proof carries.
      M_I is identity (the public signals ARE the intent payload).

  (B) Feature-binding (featurebind.go, verifyFeatureBinding, used by
      SubmitDecisionTrustlessV7): featureHash is proven, by Merkle-Patricia proof under the
      SAME verified Besu header SubmitDecisionTrustless already establishes, to equal
      FeatureCommitRegistry.featureHash[codeHash] -- an on-chain value ONE pinned scanner
      authority committed at scan time, independent of the prover. Without this check, a
      forged-feature proof still pairing-verifies (the circuit only enforces internal
      consistency between its own x and its own featureHash output, not that x is genuinely
      the honest scan of THIS contract) -- which is exactly what two already-existing real
      experiments measured, reused here rather than re-run:
        X7 (v6, no feature-binding at all; evidence/x7_forged_features_corpus_scale_2026-09-21.json):
          11/11 corpus-scale forged-feature proofs classify "SAFE" under the real v6 policy,
          9 of them for contracts whose HONEST classification is "REJECT" -- real, measured,
          corpus-scale widening.
        X9 (v7's own public featureHash signal checked for INTERNAL self-consistency only, not
          against the registry; evidence/x9_summary.json): all 11 forged proofs still
          `forged_proof_valid=true` -- the circuit-internal check alone does not catch the
          forgery; only the separate registry comparison below does.

Because real_gamma_d is, by construction, exactly what psi_s demands, check_entailment's UNSAT
result on the real policy is close to a restatement, not new information. The new information
is in the NAIVE VARIANTS -- mechanically ablating or mis-encoding one real conjunct at a time
(the same class of implementation mistake a programmer porting this check could realistically
make) and asking whether the resulting policy still recognizes exactly psi_s and nothing more.
Five variants (NV2-NV6, fragment A) are checked symbolically in Z3. Two more (NV1, NV_FB) are
NOT Z3 exercises at all -- they are real, already-measured attack data (X1 for fragment A's
model-weights-binding dimension; X7/X9 for fragment B's feature-binding dimension), reused
here as the empirical half of a formal+empirical pairing the project has not had before for
any R_P-relevant guard.

Regression discipline note: this module is independent of rp_checker.py's existing 50-case
regression table (a different fragment entirely) and does not modify rp_checker.py; it imports
check_entailment/find_divergence from it (shared core, not re-implemented).

usage: rp_checker_zk.py [out.json]   (needs the z3-solver venv: .venv_e1/bin/python3)
"""
import json, os, sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from rp_checker import check_entailment, find_divergence
from z3 import Int, Bool, And, Not

# Real deployed constants, copied verbatim from authcc/chaincode/zkverify.go.
REQUIRED_MIN_MARGIN = 54445307
REQUIRED_MAX_VULN_CLASS = 0

HERE = os.path.dirname(os.path.abspath(__file__))
X1_SUMMARY_PATH = os.path.join(HERE, "x1_out", "x1_summary.json")
EVIDENCE_DIR = os.path.abspath(os.path.join(HERE, "..", "..", "..",
                                             "paper_backups", "submit_paper2", "evidence"))
X7_PATH = os.path.join(EVIDENCE_DIR, "x7_forged_features_corpus_scale_2026-09-21.json")
X9_PATH = os.path.join(EVIDENCE_DIR, "x9_summary.json")


# ---------------------------------------------------------------------------
# Fragment A: SemBound proof-policy (zkverify.go)
# ---------------------------------------------------------------------------

def _vars_a():
    scan_passed = Bool("scanPassed")
    max_vuln_class = Int("maxVulnClass")
    margin = Int("minMargin")
    model_hash_matches_pinned = Bool("modelHashMatchesPinned")  # True iff public[5]==pinnedModelWeightsHash
    return scan_passed, max_vuln_class, margin, model_hash_matches_pinned


def real_gamma_d_a(sp, mvc, m, mh):
    """verifySemBoundProof, v6/v7 branch, lines 236-263 of zkverify.go, ported faithfully."""
    return And(sp, mvc == REQUIRED_MAX_VULN_CLASS, mh, m >= REQUIRED_MIN_MARGIN)


def psi_s_a(sp, mvc, m, mh):
    """What an honest run of the source's own Phi_s={mlScanPassed,...} establishes, in the
    circuit's public-signal vocabulary. Identical in form to real_gamma_d_a by design -- Phi_s
    IS what the policy is supposed to recognize; the naive variants ask whether a plausible
    mis-port still recognizes exactly this and nothing more."""
    return real_gamma_d_a(sp, mvc, m, mh)


def domain_a(sp, mvc, m, mh):
    # maxVulnClass bounded by the classifier's 12-class output (matching the margin-derivation
    # comment in zkverify.go: "softmax confidence over 12 classes"). margin bounded by a
    # generous symmetric range. sp/mh left otherwise free: the Groth16 circuit's own arithmetic
    # constraints bind these signals to WHATEVER private witness the prover chose, not to the
    # genuine pinned model or genuine features -- an honest abstraction, and exactly what X1
    # (below) measured directly for the weights dimension.
    return [mvc >= 0, mvc <= 11, m >= -(10**9), m <= 10**9]


def regression_check_fragment_a():
    """Sanity check only: confirm the real policy, ported into Z3, is self-consistent (UNSAT by
    construction, since psi_s_a == real_gamma_d_a). A mismatch would mean a transcription bug."""
    sp, mvc, m, mh = _vars_a()
    sound, witness = check_entailment(domain_a(sp, mvc, m, mh), psi_s_a(sp, mvc, m, mh),
                                       real_gamma_d_a(sp, mvc, m, mh), free_vars=[sp, mvc, m, mh])
    return {"sound": sound, "witness": witness}


def naive_variants_fragment_a():
    """NV2-NV6: mechanical single-conjunct ablations/mis-encodings of fragment A's real policy
    (not AI-generated adversarial content -- same discipline as rp_checker.py's own variants)."""
    variants = []
    sp, mvc, m, mh = _vars_a()
    domain = domain_a(sp, mvc, m, mh)
    real = real_gamma_d_a(sp, mvc, m, mh)

    naive2 = And(sp, mvc == REQUIRED_MAX_VULN_CLASS, m >= REQUIRED_MIN_MARGIN)  # no modelHash check
    variants.append(("NV2_omit_model_weights_hash_check", domain, real, naive2, [sp, mvc, m, mh],
                      "destination never checks modelWeightsHash==pinned -- formal analogue of "
                      "X1's real A3 attack (substituted weights)"))

    naive3 = And(sp, mvc == REQUIRED_MAX_VULN_CLASS, mh)  # no margin check
    variants.append(("NV3_omit_margin_check", domain, real, naive3, [sp, mvc, m, mh],
                      "destination never checks minMargin -- a borderline verdict would be "
                      "accepted as if it met the destination's confidence floor"))

    naive4 = And(sp, mvc <= REQUIRED_MAX_VULN_CLASS + 1, mh, m >= REQUIRED_MIN_MARGIN)  # off-by-one
    variants.append(("NV4_maxVulnClass_off_by_one", domain, real, naive4, [sp, mvc, m, mh],
                      "comparison relaxed to <=1 instead of ==0, silently accepting a class-1 verdict"))

    naive5 = And(sp, mvc == REQUIRED_MAX_VULN_CLASS, mh, m <= REQUIRED_MIN_MARGIN)  # reversed
    variants.append(("NV5_margin_comparison_reversed", domain, real, naive5, [sp, mvc, m, mh],
                      "comparison operator flipped on the margin floor (classic sign/direction bug)"))

    naive6 = And(mvc == REQUIRED_MAX_VULN_CLASS, mh, m >= REQUIRED_MIN_MARGIN)  # no scanPassed check
    variants.append(("NV6_omit_scanPassed_check", domain, real, naive6, [sp, mvc, m, mh],
                      "destination never checks the circuit's own scanPassed output signal"))

    results = {}
    for name, dom, real_g, naive_g, fv, note in variants:
        results[name] = {**find_divergence(dom, real_g, naive_g, fv), "note": note}
    return results


def x1_naive_v5_empirical():
    """NV1: real historical data, not a Z3 query. The legacy v5 relation (allowLegacyV5=true,
    the non-default, disabled path) has no modelWeightsHash public signal at all, and its
    minConfidence signal is not circuit-bound to the genuine forward pass. x1_attacks.py
    (2026-09-21) measured this directly against the real snarkjs prover and the real
    ml_scan_proof_v5 circuit: A2 (forged features), A3 (substituted weights), A2d (forged
    confidence) all produced valid_proof=true."""
    if not os.path.exists(X1_SUMMARY_PATH):
        return {"available": False, "note": f"not found at {X1_SUMMARY_PATH}"}
    with open(X1_SUMMARY_PATH) as f:
        summary = json.load(f)
    relevant = {k: v for k, v in summary.get("attacks", {}).items() if k.startswith(("A2", "A3"))}
    all_valid = bool(relevant) and all(v.get("valid_proof") for v in relevant.values())
    return {
        "available": True, "source_file": "experiments/portability/x1_out/x1_summary.json",
        "relevant_attacks": relevant, "all_dishonest_attacks_produced_valid_proofs": all_valid,
        "note": ("Under the v5 relation, forged-feature (A2), substituted-weights (A3), and "
                 "forged-confidence (A2d) dishonest inputs all produced a pairing-verifying "
                 "proof -- the real, measured instance of the same failure mode NV2/NV3 above "
                 "demonstrate symbolically for the v6/v7 policy. This is why allowLegacyV5 "
                 "defaults to false in production."),
    }


# ---------------------------------------------------------------------------
# Fragment B: feature-binding (featurebind.go), the real production entry point
# SubmitDecisionTrustlessV7's additional condition beyond fragment A.
# ---------------------------------------------------------------------------

def _vars_b():
    sp, mvc, m, mh = _vars_a()
    feature_hash_matches_registry = Bool("featureHashMatchesRegistry")
    return sp, mvc, m, mh, feature_hash_matches_registry


def real_gamma_d_full(sp, mvc, m, mh, fb):
    """SubmitDecisionTrustlessV7's semBound = zkOK (fragment A) AND featureOK (fragment B,
    verifyFeatureBinding), exactly as coded (featurebind.go, SubmitDecisionTrustlessV7)."""
    return And(real_gamma_d_a(sp, mvc, m, mh), fb)


def psi_s_full(sp, mvc, m, mh, fb):
    return real_gamma_d_full(sp, mvc, m, mh, fb)


def naive_variant_omit_feature_binding():
    """NV_FB (symbolic half): the naive policy checks fragment A only and never compares
    featureHash against FeatureCommitRegistry -- i.e. it is exactly v6/early-v7 usage
    (SubmitDecisionTrustlessV7 without this file's verifyFeatureBinding call). Z3 confirms the
    structural fact (dropping a required AND-conjunct can only widen); the REAL achievability of
    the dropped case is established empirically by X7/X9 below, not by this symbolic check alone."""
    sp, mvc, m, mh, fb = _vars_b()
    domain = domain_a(sp, mvc, m, mh)
    real = real_gamma_d_full(sp, mvc, m, mh, fb)
    naive = real_gamma_d_a(sp, mvc, m, mh)  # feature-binding omitted
    return {**find_divergence(domain, real, naive, [sp, mvc, m, mh, fb]),
            "note": ("destination accepts based on fragment A alone, never comparing the "
                      "proof's featureHash to FeatureCommitRegistry.featureHash[codeHash] -- "
                      "the exact omission X7 (v6) and X9 (v7 self-consistency only) measured "
                      "as real, exploitable forged-feature widening, see x7_x9_naive_feature_binding_empirical below")}


def x7_x9_naive_feature_binding_empirical():
    """NV_FB (empirical half): real corpus-scale measurements, not re-run here."""
    out = {"available": False}
    if os.path.exists(X7_PATH):
        with open(X7_PATH) as f:
            x7 = json.load(f)
        n = len(x7)
        n_forged_safe = sum(1 for v in x7 if v.get("forged") == "SAFE")
        n_honest_reject = sum(1 for v in x7 if v.get("honest") == "REJECT")
        out["x7_v6_no_feature_binding"] = {
            "available": True, "source_file": "evidence/x7_forged_features_corpus_scale_2026-09-21.json",
            "n_corpus_contracts": n, "forged_classified_SAFE": n_forged_safe,
            "honest_classified_REJECT": n_honest_reject,
            "note": (f"{n_forged_safe}/{n} real corpus-scale forged-feature proofs classify SAFE "
                     f"under the real v6 policy (fragment A alone), {n_honest_reject} of them for "
                     "contracts whose honest classification is REJECT -- real widening."),
        }
        out["available"] = True
    if os.path.exists(X9_PATH):
        with open(X9_PATH) as f:
            x9 = json.load(f)
        vs = x9.get("variants", [])
        n_forged_valid = sum(1 for v in vs if v.get("forged_proof_valid"))
        n_honest_invalid = sum(1 for v in vs if v.get("honest_proof_valid") is False)
        out["x9_v7_self_consistency_only"] = {
            "available": True, "source_file": "evidence/x9_summary.json",
            "n_variants": len(vs), "forged_proof_valid_count": n_forged_valid,
            "honest_proof_valid_false_count": n_honest_invalid,
            "note": (f"{n_forged_valid}/{len(vs)} forged v7 proofs are valid under circuit-internal "
                     "self-consistency alone (featureHash matches the forged x, but not the "
                     "registry's honest commitment for the real codeHash) -- confirming the "
                     "registry comparison (fragment B), not the circuit's own signal, is what "
                     "closes the gap."),
        }
        out["available"] = True
    return out


def main(dst="rp_checker_zk_results_2026-10-11.json"):
    reg = regression_check_fragment_a()
    print(f"Fragment A self-check (sound by construction): sound={reg['sound']}")
    if not reg["sound"]:
        print("ERROR: fragment A transcription is not self-consistent; stopping.", file=sys.stderr)
        sys.exit(1)

    nv_a = naive_variants_fragment_a()
    widening_a = sum(1 for v in nv_a.values() if v["widening_found"])
    print(f"Fragment A naive variants (NV2-NV6): {widening_a}/{len(nv_a)} widening")
    for name, v in nv_a.items():
        print(f"  {name}: widening={v['widening_found']}")

    x1 = x1_naive_v5_empirical()
    print(f"NV1 (real X1 v5 empirical): all_dishonest_attacks_produced_valid_proofs="
          f"{x1.get('all_dishonest_attacks_produced_valid_proofs')}")

    nv_fb_symbolic = naive_variant_omit_feature_binding()
    print(f"NV_FB symbolic (omit feature-binding): widening={nv_fb_symbolic['widening_found']}")
    nv_fb_empirical = x7_x9_naive_feature_binding_empirical()
    if nv_fb_empirical.get("x7_v6_no_feature_binding"):
        print("NV_FB empirical (X7):", nv_fb_empirical["x7_v6_no_feature_binding"]["note"])
    if nv_fb_empirical.get("x9_v7_self_consistency_only"):
        print("NV_FB empirical (X9):", nv_fb_empirical["x9_v7_self_consistency_only"]["note"])

    total_widening = widening_a + (1 if x1.get("all_dishonest_attacks_produced_valid_proofs") else 0) \
        + (1 if nv_fb_symbolic["widening_found"] else 0)
    out = {
        "fragment_a": "SemBound proof-policy (zkverify.go / verifySemBoundProof)",
        "fragment_b": "Feature-binding (featurebind.go / verifyFeatureBinding, SubmitDecisionTrustlessV7)",
        "fragment_a_self_check": reg,
        "fragment_a_naive_variants_NV2_NV6": nv_a,
        "naive_variant_NV1_real_empirical_v5": x1,
        "naive_variant_NV_FB_symbolic": nv_fb_symbolic,
        "naive_variant_NV_FB_real_empirical_X7_X9": nv_fb_empirical,
        "widening_count_total": total_widening,
        "total_variants": len(nv_a) + 2,  # NV1 + NV_FB
    }
    with open(dst, "w") as f:
        json.dump(out, f, indent=2)
    print(f"Wrote {dst}")
    return out


if __name__ == "__main__":
    main(sys.argv[1] if len(sys.argv) > 1 else "rp_checker_zk_results_2026-10-11.json")
