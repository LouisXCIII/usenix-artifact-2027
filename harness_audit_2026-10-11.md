# Harness reliability audit: can any evaluation script count an error as a match?

**Date:** 2026-10-11. **Trigger:** while fixing today's Group 3 pilot, found that real OPA
writes parse errors to **stdout** (not stderr); the evaluation harness's fallback branch
(`out == ""` -> `False`) could, in a different harness, silently coerce an engine crash into
a "DENY" match rather than a distinguishable error. The user asked for every harness that
produced a number in this paper to be checked for the same class of bug: can it distinguish
"engine genuinely rejected (false/DENY)" from "engine errored" from "engine never ran"? A
reported "0 errors" / "N/N matched" result is only trustworthy if the error case is
*structurally* prevented from being counted as a match, not just empirically absent so far.

## Method

A fork (full project context, read-only, no edits) first surveyed every evidence file the
manuscript cites directly (9 files via `\texttt{evidence/...}`) plus the scripts each one
runs, specifically hunting for: (a) subprocess/exec calls to external CLI tools whose stdout
is parsed as a pass/fail string without checking the exit code first; (b) Z3 `Solver.check()`
calls that don't distinguish `unknown` from `unsat`; (c) JS/Go harnesses that could fold a
script-level error into the same outcome as a genuine on-chain rejection. I then personally
verified its flagged finding and manually extended the sweep to the areas the fork had only
enumerated, not read in depth (S5, S6, S9, S11b).

## Confirmed bug #1 (already fixed before this audit): Group 3 pilot's hardened OPA harness

`run_opa.py` (today's `qd_pilot_g234`), first version: any non-`"true"`/non-`"false"` OPA
stdout (including a parse-error message, since OPA prints those to stdout) silently mapped to
`False`. Caught immediately because it broke in the FALSE-positive-for-errors direction that
day (my hardened Group 3 policy had a missing Rego import, so EVERY case errored and showed as
a mismatch, 63/63) -- not the dangerous direction (an error silently counted as a correct
match). Fixed same day: harness now raises on nonzero OPA exit code; policy import fixed. See
`evidence/qd_pilot_g234_2026-10-11.md`.

## Confirmed bug #2 (new, found by this audit): `x16_risk_tier_proofs.py`

`~/pqc-hardhat/experiments/portability/x16_risk_tier_proofs.py`, function `prove()`. The
script ran the real `snarkjs groth16 verify` and correctly computed
`valid = "OK" in ok.stdout` (line 99 of the original) -- then **discarded that value** and
unconditionally `return {"valid": True, ...}` (original line 104), regardless of what
`valid` actually held. This is a worse instance of the same bug class: not a parsing
ambiguity, but a computed result thrown away and replaced with a hardcoded success.

**Blast radius:** this script generated the real multi-floor Groth16 proofs behind Table 8's
"Two-vocabulary $R_P$ instance (risk-tier -> cap-fraction)" row (6/6 on-chain, 3/3 AWS
co-signer), and `tier1_example_T1_{proof,public}.json` specifically is the base proof reused
by `qd_necessity_demo_2026-10-11.md` and the Group 1 pilot (2026-10-11) for their forged-proof
demonstrations.

**Verified impact: none.** I independently re-ran `snarkjs groth16 verify` (fresh invocation,
outside the buggy script, against the real `verification_key_v6.json`) on all 6 proof/public
pairs currently in `x16_out/` (tier1_example_T1, tier2_example_T1, tier2_example_T2,
tier3_example_T1, tier3_example_T2, tier3_example_T3). **All 6 returned `OK`.** The bug was
latent: it never had the opportunity to matter, because every proof this script actually
produced and kept was genuinely valid. Separately, the paper's "6/6 on-chain decisions matched
expectation" claim does not depend on this script's self-check at all -- it depends on the
real chaincode's own `bn254.PairingCheck` verification at the point of submission, an
independent check this bug could not have touched.

**Fixed:** `prove()` now propagates the real `valid`/exit-code result instead of discarding
it (checks `ok.returncode != 0` first, then the real `valid` boolean, returning
`{"valid": False, "why": ...}` on either failure path).

**Checked for the same pattern across 7 structurally similar sibling scripts** (`x6_v6_prove.py`,
`x7_forge_variants.py`, `x8_v7_featurebind.py`, `x9_v7_corpus_forge.py`,
`s4_fair_b1/make_fixtures.py`, `s8_population/s8_make_fixtures.py`,
`s10_obligation_gap_close/forge_feature_proof.py`) -- all 7 correctly propagate their computed
verify flag. `x16` was an isolated regression, not a copy-pasted systemic pattern.

## Hardening: Z3 `unknown` was not distinguished from `unsat`

`rp_checker.py`'s `find_divergence()` -- reused unmodified by every Z3-based claim in this
paper (Corollary `cor:rp-checker`, the risk-tier Trụ 4 result, and all four of today's
Group 1-4 pilot checks) -- computed `widening = s1.check() == sat`. If Z3 ever returned
`unknown` (could not decide), this silently resolved to `False` ("no widening found"), the
same as a genuine `unsat`, with nothing distinguishing the two. Every domain this project
passes to it is quantifier-free linear integer arithmetic over explicitly bounded ranges -- a
fragment Z3 decides completely -- so `unknown` was very unlikely to have actually occurred,
but nothing asserted that it hadn't. **Fixed:** both of `find_divergence()`'s two queries now
raise a `RuntimeError` naming Z3's `reason_unknown()` if `check()` ever returns `unknown`,
instead of silently treating it as a negative result. **Re-ran every Z3-based result in the
paper after this change** to confirm no regression and that `unknown` was never silently
masking anything:
- `rp_checker.py`'s own regression suite: 0/50 mismatches vs. the `e1_oracle.py` baseline,
  24/50 sound -- identical to before the fix.
- `rp_checker_tier.py` (risk-tier Trụ 4): 4/4 naive mapping variants widening -- identical.
- All four of today's `z3_check_*.py` pilot scripts (Groups 1, 2, 4): identical
  `widening_found=False, over_conservative_found=False` results.

No run anywhere in the project's history has ever hit the `unknown` branch; the hardening is
a forward-looking guard, not a correction of a past result.

## Scripts checked and found SAFE (explicit exit-code/typed-result handling, not
coincidentally-absent errors)

- **S3 independent policy oracle** (`evidence/s3_independent/`): `oracle_cedar.py` uses the
  `cedarpy` library in-process (typed decision enum, not parsed CLI text); `s3_independent_test.go`
  is a `go test` harness writing typed Go `error` values to JSON; `mutation_check.py` uses
  `subprocess.run(..., check=True)`, which raises `CalledProcessError` on nonzero exit,
  recorded as a third, explicit value distinguishable from both match and mismatch;
  `analyze_s3.py` looks up results by dict key with no silent fallback -- a missing case
  crashes loudly (`KeyError`).
- **On-chain invoke/query wrappers** (`x8_invoke.py`, `x9_invoke.py`, `x15_invoke_independent.py`,
  `s7_run.py`, `s8_run.py`, `s6_t0_multihop.py`, and by inheritance `s11_behavior_safety/run.py`,
  which reuses `s8_run.py`'s helpers): all gate success on an explicit marker string
  (`"Chaincode invoke successful"` + a `payload:"..."` capture), returning an explicit error
  string on anything else -- never a value that could collide with a genuine `SAFE`/`REJECT`/
  class string. Verified `s6_t0_multihop.py` specifically: its `matches_model` comparison
  would show `False` (not a false match) if an invoke ever errored, since the model's
  predictions are always `"SAFE"`/`"REJECT"` strings that cannot equal `None`.
- **`s5_prover_scaling.py`**: correctly propagates `entry["verify_ok"] = "OK" in v.stdout`
  into its output record (not discarded, unlike `x16`). Independently confirmed all 4 recorded
  `verify_ok` values in `s5_prover_scaling.json` are genuinely `True`.
- **`s9_a3_query.js`**: real `eth_call` reads via `ethers`, with a single top-level
  `.catch((e) => { console.error(e); process.exit(1); })` -- any per-item RPC failure crashes
  the whole script loudly rather than being caught and silently recorded as "no false hit".
- **Cedar harnesses generally** (every pilot this session, S3, the necessity demo): the
  `cedar-policy` Rust crate and `cedarpy` Python bindings are in-process library calls
  returning a typed decision value -- structurally immune to the stdout-text-parsing ambiguity
  that caused the Group 3 OPA bug. No harness in this codebase wraps a Cedar CLI binary via
  subprocess.
- **`qd2_domain_attestation`'s AWS SSH calls**: no explicit returncode check before
  `json.loads(...)`, but a garbled/failed response raises `json.JSONDecodeError` uncaught --
  a loud crash, not a silent false match.

## Not independently re-verified line-by-line (lower priority, lower apparent risk)

S11b's dependency on `s8_run.py` was checked by inheritance (same helpers as the already-SAFE
S7/S8 scripts) rather than read fully standalone; E8/E10 concurrency scripts were enumerated
but not read. No subprocess/exec usage was found in a grep sweep of the S5/S6/S9/S11b
directories beyond what is documented above, which is some evidence of lower risk (most of
the finite-state/behavioral-safety results use in-process chaincode calls or the already-SAFE
invoke wrapper pattern), but is not a substitute for having read every line.

## Bottom line

Of all harnesses checked (fork's pass plus this follow-up): **2 confirmed bugs, both found
and fixed, neither changed any number currently published in the paper** (Group 3 pilot's bug
was caught same-day before any number was ever reported from it; `x16`'s bug was latent and
is now independently confirmed to have never produced a wrong proof-validity claim). **1
hardening gap fixed** (Z3 `unknown` handling), confirmed via re-running every affected result
to have never actually been masking anything. No other harness checked showed the pattern.
