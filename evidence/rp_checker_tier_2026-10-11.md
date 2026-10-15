# Tru 4 proper: a two-vocabulary R_P instance (2026-10-11)

> **Corrected same day, after a second round of external review of this same file.**
> The original version of this file used self-praising language ("genuine, real",
> "Critically") the result does not need, pooled a prover-authenticity variant (NV3)
> into the same "4/4 mapping errors widening" count as three real mapping-error
> variants, called the AWS co-signer an "independently-governed administrative
> domain" that "independently re-derives... from scratch" (language that contradicts
> this same project's own disclosure elsewhere that the AWS account and the OpenStack
> deployment are operated by the same authors, not a second independently governed
> organization), and did not disclose the per-tier multi-proof cost or this result's
> small illustrative scale. All of these are fixed below; the underlying real
> measurements are unchanged. The manuscript's Table 8 row, appendix paragraph, and
> Open Science index entry are corrected to match.

Closes the gap confirmed open same-day by two independent external reviews
(evidence/rp_checker_zk_2026-10-11.md's correction banner): the SemBound/feature-binding
fragment is NOT a non-trivial $R_P$ instance, because $\psi_s$ and $\gamma_d$ were stated over
the identical public-signal vocabulary -- $M_I$ was the identity map, $\gamma_d\equiv\psi_s$ by
construction. Both reviews independently recommended the same fix: a source predicate on one
scale, a destination predicate on a genuinely different scale, connected by a real $M_I$ mapping
a checker can catch misconfigurations in. This file is that fix, built real, not simulated.

## Why this differs structurally from the corrected SemBound mistake

- **Source vocabulary ($\Psi_s$)**: a risk TIER (0-3), expressed purely on the ZK circuit's own
  quantized-logit margin scale. The v6/v7 circuit relation does not output the achieved margin
  as a continuous public value -- `minMargin` is a PARAMETER supplied at proof-generation time,
  and witness generation genuinely fails if the real margin falls short (confirmed empirically,
  not assumed). A destination that requests real proofs at several different floors for the same
  contract gets a real, measured tier: which floors a genuine prover can satisfy. Caveat
  disclosed, not papered over: *which floors are queried at all* is a destination-side choice
  (the destination decides how many tiers to request proofs at) -- a governance parameter, not
  something the source volunteers unprompted. *Whether a given contract's real margin clears a
  given floor*, once that floor is fixed, is the genuine source-side fact proof soundness
  prevents the source from faking. The two are disclosed separately, not conflated.
- **Destination vocabulary ($\Psi_d$)**: dollar amounts against `RoleThreshold`'s real role caps
  (clerk/manager/director), completely unrelated to the margin scale.
- **$M_I$**: a real governance table, tier -> fraction-of-role-cap-allowed (basis points),
  translating one scale into the other -- not an identity pass-through. The checker below
  verifies only this table's internal transcription consistency; it does not calibrate the
  table against the scanner's own measured 15/253 (5.9%) false-negative rate by tier, which is
  not measured in this pass (a natural, not-yet-done extension, same as the AWS-table-divergence
  extension noted below).
- **$\gamma_d$ is NOT definitionally equal to $\psi_s$**: $\gamma_d$ (amount <= cap(role) x
  fraction(tier)) depends on role and amount, which $\psi_s$ (achieved tier) says nothing about
  at all. This is the structural difference from the corrected SemBound fragment.

## Part 1: real multi-floor Groth16 proofs build a genuine tier signal

`x16_risk_tier_proofs.py` (new): no circuit change, no new trusted setup -- reuses the existing
v6 circuit and verification key exactly. Three floors (T1=3.2452 logits, the unchanged real
deployed floor; T2=4.0; T3=5.0). Four real, already-scanned corpus contracts (real margins from
the existing x6 corpus, not re-picked to fit) land in four different tiers:

| Contract | Real margin (logits) | Achieved tier |
|---|---|---|
| `04_call_to_unknown_safe.sol` | 5.5318 | **3** (all 3 floors succeed) |
| `08_timestamp_vuln.sol` | 4.5947 | **2** (T1,T2 succeed, T3 fails) |
| `09_mishandled_exception_vuln.sol` | 3.2478 | **1** (T1 succeeds, barely; T2 fails) |
| `SafeBank.sol` | 1.2359 | **0** (no valid proof at any floor) |

Raw: `x16_summary.json` (per-floor witness-generation success/failure for every case).

## Part 2: Z3 formalization, 3/3 mapping errors plus 1 authenticity ablation, all widening

`rp_checker_tier.py` (new, imports `find_divergence` from `rp_checker.py`, does not modify it).
Real policy: `amount*10000 <= cap(role) * fraction(tier)`, tier>=1 required. Four mechanically
constructed naive variants (not AI-generated, same discipline as every other naive-variant
generator in this project); on inspection, three are genuine mapping mistakes and one (NV3) is a
prover-authenticity mistake (the category already covered correctly in `rp_checker_zk.py`), so it
is reported separately rather than pooled into the same count:

| Variant | Mistake | Category | Widening |
|---|---|---|---|
| NV1 | governance table transcribed in reverse order | mapping error | **YES** |
| NV2 | tiers 2 and 3's fractions swapped (realistic single-entry transcription error) | mapping error | **YES** |
| NV4 | achieved tier mis-indexed one band too high before the fraction lookup | mapping error | **YES** |
| NV3 | trusts a caller-declared tier instead of the one real multi-floor proofs establish | prover-authenticity (not a mapping error) | **YES** |

**3 of 3** genuine cross-vocabulary mapping errors cause real widening; the separately-reported
authenticity variant (NV3) is also widening. Each has a concrete Z3 counterexample witness
(`rp_checker_tier_results_2026-10-11.json`). Note on the standard used: "widening" here means
divergence from this project's own reference table -- a transcription self-check. Part 2b below
replaces that self-reference with a real, externally-grounded exposure requirement for the
headline comparison.

## Part 2b: FN-rate-by-tier calibration against an externally-annotated corpus (2026-10-11, corrected)

**Closes the gap this file previously disclosed as "not attempted in this pass" -- corrected
twice after two further rounds of external review, both real catches, disclosed in full below.**

**A real bug in the first version of this section, found and fixed.** The first version's
script (`fn_rate_by_tier.py`, kept for the record) computed `pred_margin(x)` -- which returns
`(top_predicted_class, margin)` -- but only tested `margin >= floor`, NOT
`top_predicted_class == 0` (the scanner's own SAFE verdict). A contract whose top predicted
class was a *vulnerability* class, confidently predicted, could still have a large top-two-logit
margin and so incorrectly counted as "clearing the tier" -- inflating the apparent false-negative
rate. `fn_rate_by_tier_v2.py` fixes this to require both conditions, exactly matching
`s8_population/s8_prepare.py`'s own `strict_sat = (pred==0 and margin>=MM)` definition and
`chaincode/risk_tier.go`'s real on-chain tier logic, and self-checks against the paper's
already-published `15/253` figure at the original single floor (passes exactly: `15/253`, not
close -- exact). The first version's published numbers (FN rate 22.1%/10.3%/3.2%, "a real
miscalibration" at tier 2) were therefore simply wrong, not just methodologically thin; corrected
below.

**A real methodological gap, found and fixed.** A further review pointed out that the 253-contract
corpus is *entirely* externally-annotated-vulnerable: what v1 computed, even after the bug fix,
is $\Pr[\text{tier}\ge t \mid \text{vulnerable}]$, not $\Pr[\text{vulnerable}\mid\text{tier}=t]$
-- and that cumulative "clears floor $t$" counts are monotonically non-increasing in $t$ *by
construction*, for any scanner, so "the false-negative rate decreases with tier" alone is not
evidence the assumption holds. Fixed by adding a genuine benign comparison: the `deployed`
corpus already in this project (263 real production bridge/DeFi contract sources from the S1
survey; a *proxy* for benign traffic -- not independently audited vulnerability-free, same
caveat disclosed, not resolved, below), giving $\Pr[\text{tier}\ge t\mid\text{benign}]$
independently, which is not guaranteed to track the vulnerable corpus's curve at all.

`fn_rate_by_tier_v2.py` (new; same `pred_margin()`/Clopper-Pearson machinery as v1, same
feature source `s5/s5_corpus_features.json`, no new scanning):

| Tier | $\Pr[{\ge}t\mid\mathrm{vuln}]$ ($n{=}253$) | CP95 | $\Pr[{\ge}t\mid\mathrm{benign}]$ ($n{=}263$) | CP95 | $\mathrm{LR}(t)$ |
|---|---|---|---|---|---|
| 1 | 5.93% | 3.36-9.59% | 15.21% | 11.09-20.13% | **0.390** |
| 2 | 3.95% | 1.91-7.15% | 11.79% | 8.15-16.31% | **0.335** |
| 3 | 1.19% | 0.25-3.43% | 6.84% | 4.11-10.60% | **0.173** |

**The likelihood ratio $\mathrm{LR}(t)=\Pr[t\mid\mathrm{vuln}]/\Pr[t\mid\mathrm{benign}]$
genuinely decreases with tier (0.390 -> 0.335 -> 0.173).** This is real evidence -- not
structurally guaranteed, since the benign corpus's curve could have moved any direction -- that
clearing a higher floor is associated with lower relative vulnerability likelihood. This is the
closest thing in this pass to validating the "higher margin means lower risk" assumption; it is
weaker than a clean separation (the two CP95 ranges above are not disjoint at every tier pair)
but the direction is consistent and corpus-independent.

**Bayes posterior $\Pr[\mathrm{vuln}\mid\mathrm{tier}=t]$, swept over an unknown real-traffic
prior $\pi$ (discrete, exactly-achieved tiers, not cumulative)**, since $\pi$ is not known and a
single assumed value should not be hidden:

| Tier | $\pi{=}1\%$ | $\pi{=}5\%$ | $\pi{=}20\%$ |
|---|---|---|---|
| 1 | 0.58% | 2.95% | 12.62% |
| 2 | 0.56% | 2.86% | 12.28% |
| 3 | 0.17% | 0.90% | 4.15% |

At every $\pi$ tested, tier 3 has a meaningfully lower posterior than tiers 1 and 2; tiers 1 and
2 are close to each other (not cleanly separated) at every $\pi$. Point-estimate expected
exposure $\Pr[\mathrm{vuln}\mid t]\times\mathrm{fraction}(t)$ at $\pi{=}5\%$: tier 1 $0.295\%$,
tier 2 $1.431\%$, tier 3 $0.904\%$ of cap -- **tier 2 is still the point-estimate worst point**
of the real deployed table (10% cap at tier 1 vs.\ 50% at tier 2 outweighs tier 2's modestly
lower per-decision risk), consistent with v1's finding even after the bug fix and the Bayes
correction.

**This point-estimate separation is not robust once small-sample uncertainty is accounted for.**
Tier 3 has only 3-8 contracts behind its rate in either corpus; its CP95 interval is wide. Using
the conservative worst case over the confidence interval (CP95 upper bound on the vulnerable
rate, CP95 lower bound on the benign rate, cumulative $\ge t$ framing, $\pi{=}5\%$) instead of
the point estimate:

| Table | UB-EEF(1) | UB-EEF(2) | UB-EEF(3) | max UB-EEF |
|---|---|---|---|---|
| Real (deployed) | 0.435% | 2.206% | **4.206%** | **4.206%** |
| NV1 (table reversed) | 4.352% | 2.206% | 0.421% | **4.352%** |
| NV2 (tiers 2/3 swapped) | 0.435% | 4.412% | 2.103% | **4.412%** |
| NV4 (off-by-one) | 2.176% | 4.412% | 4.206% | **4.412%** |

Under this conservative accounting, (a) the worst tier for the real table shifts from tier 2 to
tier 3 -- the point estimate's "tier 2 is worst" finding is itself not robust to the small tier-3
sample -- and (b) the real table's worst-case bound (4.206%) is close to, not dramatically
better than, the three naive variants' (4.352%/4.412%/4.412%), unlike the clean separation the
uncorrected point-estimate comparison in the earlier version of this section suggested. **The
real table is still better than every naive variant at this conservative bound, but only
modestly, and the headline "real miscalibration at tier 2" framing from the earlier version of
this section overstated a finding the small corpus does not support as cleanly as it first
looked.** Reported honestly rather than kept at the more favorable point estimate.

**Data-leakage investigation (2026-10-11), the originally-flagged concern does not hold up.**
The first version of this section flagged, as an unresolved risk, that the string
`WeakAccessMod` appears inside `~/Desktop/projectAI/buildVocab/trainning/dataset/results*/
solidity_vulnerabilities.json`. Traced further: that file belongs to a separate dataset-prep
effort (`buildVocab/trainning/dataset/dedup_and_group_split.py`, a `GroupShuffleSplit`-based
pipeline that does reference `WeakAccessMod` by name for stratification), but its own output
files (`outpt_v2_final/`, `outpt_v3_final/`, the `WeakAccessMod` source directory itself) are
all dated August 2026. The pinned model (`fold0_epoch14_f10.9661.pt`) is dated 2026-02-05 --
six months earlier -- so this specific pipeline cannot be its training source. The REAL training
run was located and read directly: `[unrelated path on disk]/lstm/logs/enhanced_lstm_20260203_225232.log`
records this exact checkpoint being saved at 2026-02-05 13:28:02, loading data from
`[unrelated path on disk]/finalSlicing/test/outputs/processed_data.pt` (952,126 samples, a
Windows-side path not present on this machine). Across all 11 training-run logs found for this
pipeline, none mentions `smartbugs` or `weakaccess` (checked with `grep -il`, zero matches).
**Conclusion: the specific leakage path originally flagged does not hold up** -- the
`WeakAccessMod`-referencing pipeline is a different, later, apparently unconnected effort, not
the one that produced the pinned model. This is not a complete leakage proof: the real training
corpus (`finalSlicing/test/outputs/processed_data.pt`) lived on a separate machine never
transferred here, so a direct file-hash comparison against the 253-contract evaluation corpus
was not possible, and the possibility of unrelated overlap (e.g.\ both pipelines independently
drawing from a common public Solidity-vulnerability dataset) is not ruled out. Reported as
"investigated, specific concern not supported by the evidence found" rather than either "fixed"
or "unresolved". **No train/test split or cross-validation was performed** on the evaluation
side either: calibration and reporting use the same corpus, a real limitation, not corrected
here given the scope of this pass.

**What this still does not establish**: no exposure *budget* is asserted or defended (what
counts as "safe enough" exposure is a destination governance choice left open); the benign-proxy
corpus is real production code, not independently audited vulnerability-free; this is a finite,
enumerable comparison over 3 tiers (computed directly, not via Z3, the right tool for a
symbolic universally-quantified amount, not a fixed per-tier summary statistic); possible
training-data overlap is flagged, not resolved. Raw: `fn_rate_by_tier_2026-10-11.json` (v1,
superseded, kept for the disclosed-bug record), `fn_rate_by_tier_v2_2026-10-11.json` (v2,
current).

## Part 3: real on-chain deployment, Fabric only (no Besu/OpenStack dependency)

`chaincode/risk_tier.go` (new, additive): `SubmitDecisionRiskTierVerified` reuses
`verifyGroth16`/`groth16Proof`/`groth16VKey`/the embedded v6 verification key from `zkverify.go`
unchanged; derives the achieved tier from the submitted proof's own declared floor (public[4])
rather than one fixed constant. Deployed live: v2.40, sequence 52 (later v2.42/sequence 54, see
Part 4), both organizations approved.

**6 real on-chain transactions** (real proofs from Part 1, real `RoleThreshold` caps, role=
director throughout for a clean same-role comparison):

| # | Tier | Amount | Adjusted cap | Expected | Actual |
|---|---|---|---|---|---|
| 1 | 3 (100%) | 1,800,000 | 2,000,000 | SAFE | **SAFE** |
| 2 | 3 (100%) | 2,500,000 | 2,000,000 | REJECT | **REJECT** |
| 3 | **2 (50%)** | **1,800,000** (same amount as #1) | **1,000,000** | **REJECT** | **REJECT** |
| 4 | 2 (50%) | 900,000 | 1,000,000 | SAFE | **SAFE** |
| 5 | 1 (10%) | 150,000 | 200,000 | SAFE | **SAFE** |
| 6 | 1 (10%) | 250,000 | 200,000 | REJECT | **REJECT** |

**6/6 matched expectation**, on an illustrative, small scale (four contracts, six on-chain
decisions) -- a mechanism demonstration, not a statistical claim. Cases #1 vs #3 are the crux of
the whole result: the identical role (director) and identical amount (1,800,000) are SAFE under
one real ZK-proven risk tier and REJECT under another -- a real, on-chain effect that the old
destination-local `RoleThreshold` check alone could never produce. Raw:
`risktier_onchain_results_2026-10-11.json`.

**Cost, disclosed and not amortized away:** determining a contract's tier this way costs one
real Groth16 proof per floor queried (three floors here), each measured at 8-15s and several GB
of memory (Part 1). Establishing a tier is therefore several times more expensive on the source
side than a single-floor proof -- a real cost of this design, not folded into the existing
single-proof cost figures used elsewhere in this project.

**Infrastructure note (answers "does this run on your real OpenStack/AWS systems"):** this part
needs only the Fabric network (Docker, this host) -- the OpenStack-hosted Besu main chain was
NOT required and was not brought up for Part 3, because the risk-tier signal is established
entirely through the off-chain Groth16 proof pipeline and verified on-chain in Fabric; it does
not depend on `SubmitDecisionTrustless*`'s Besu light-client path at all.

## Part 4: a second, separately-hosted co-signer (AWS, same operator) over M_I itself

Per the user's explicit request to check real OpenStack/AWS deployability: the existing AWS
account (`q_d_domain_attestation.go`'s account, a distinct new key) was confirmed reachable and
used for a real, live second check over the tier-fraction table specifically -- not just
obligation-discharge coverage (q_d_domain_attestation.go's existing scope). As disclosed
elsewhere in this project (the second-key discharge co-signature result), the AWS account and
the OpenStack deployment used everywhere else in this paper are operated by the same authors --
this is a second, physically- and key-separated signer under the same operator, not a second,
independently governed organization.

- `risktier_aws/main.go` (new, standalone, on AWS only): holds its own from-scratch second
  implementation (not a call into the Fabric Go module -- re-typed from the governance table's
  spec, same "dual independent implementation" methodology used for the liboqs/circl oracles
  elsewhere in this project) of the SAME agreed tier->fraction mapping, and signs an attestation
  with a key generated on, and never transmitted off, that AWS account. Because both
  implementations encode the same agreed table rather than independently discovering it, what
  this tests is whether the table and decision inputs survive transmission to a
  separately-keyed second check intact -- not independent discovery or governance of the
  table's values. Address: `0x731c7cef6ba0b375f6ae6df77f29058d73d6bd97`.
- `SubmitDecisionRiskTierDomainVerified` (chaincode/risk_tier.go): requires BOTH Fabric's own
  local computation AND a genuine AWS co-signature agreeing before SAFE is possible.

**A real bug found and fixed during this part, disclosed in full**: the first on-chain attempt
produced REJECT with a genuine-but-wrong recovered AWS signer address. Root-caused (not
guessed): the digest function in `risk_tier.go` called `word()` -- which, in the SAME Go
package, resolves to `commit.go`'s existing 32-byte big-endian encoder, not an 8-byte one
matching `risktier_aws`'s own encoding, because of an accidental name collision (both files are
in `package chaincode`). Fixed by adding a distinctly-named local `word8()` helper matching
`risktier_aws`'s encoding exactly (the same defensive pattern `q_d_domain_attestation.go` already
used for the identical reason). A second, unrelated test-harness mistake (reusing an AWS
signature signed over one `decisionId` for an on-chain submission under a different
`decisionId`, after a replay-rejection forced a rename) was also caught and fixed by re-signing
with the exact ID used on-chain -- the chaincode's rejection in that case was the replay/digest
binding working correctly, not a bug.

**3 real on-chain transactions**, redeployed after the fix (v2.42, sequence 54):

| # | Scenario | Fabric alone would say | Expected | Actual |
|---|---|---|---|---|
| A | genuine AWS co-signature, both domains agree (tier 2, amount 900,000) | SAFE | SAFE | **SAFE** |
| B | genuine AWS co-signature, both domains agree it's over cap (tier 2, amount 1,800,000) | REJECT | REJECT | **REJECT** |
| C | AWS signature corrupted (1 hex char flipped) | SAFE | **REJECT** | **REJECT** |

Case C is the real security property: even though Fabric's own local computation alone would
say SAFE, a forged/corrupted second-domain signature is caught and fails closed. Raw:
`risktier_aws_onchain_results_2026-10-11.json`.

## What this does and does not establish

- Real $R_\Sigma$/$M_I$ encoding: $R_\Sigma$ is the Groth16 relation (trusted cryptographic
  oracle, same boundary used throughout this project); $M_I$ is the tier->fraction table,
  genuinely independent of both vocabularies it connects.
- Does not claim the specific fraction values (0%/10%/50%/100%) are independently derived
  security parameters -- they are a destination governance choice for this experiment,
  illustrative, same category as other threshold choices already disclosed elsewhere in this
  project (e.g. the `minConfidence`/margin floors).
- `SemBound`/`EvAssuranceOK` (the full light-client/obligation pipeline) are not part of this
  entry point -- it isolates the tier-mapping guard specifically, matching the isolation level
  used by every other single-fragment ablation/checker in this project.
- Part 4's AWS co-signer holds a second implementation of the SAME agreed governance table (by
  design, to test transmission integrity and forgery-resistance, not independent governance or
  independent discovery of the table); it was not given a deliberately different table to probe
  real divergence detection in this pass -- that is a natural extension, not attempted here.
- The "4/4 widening" claim in an earlier version of this file pooled a prover-authenticity
  variant (NV3) into the same count as three real mapping-error variants; corrected above to
  3/3 mapping errors plus 1 separately-reported authenticity variant.
- "Widening" throughout Part 2 means divergence from this project's own reference table. Part 2b
  supplies the independent, externally-grounded exposure comparison a reviewer asked for instead
  (real false-negative rate by tier on an external corpus); no exposure *budget* is asserted or
  checked there, only a comparison of the real table against the naive variants under the same
  real rates -- see Part 2b's own "What this still does not establish".
- Not yet wired into the measured $B_4$ entry points (`SubmitDecisionTrustlessV7`/
  `V7Obligation`) -- a separate, isolated entry point, same disclosure already made for the
  MapRec governance ablation (evidence/maprec_ablation_2026-10-11.md).

## Files

- `rp_checker_tier_2026-10-11/x16_risk_tier_proofs.py`, `x16_summary.json` (real multi-floor
  proof generation and results; canonical copies also at
  `~/pqc-hardhat/experiments/portability/`)
- `rp_checker_tier_2026-10-11/rp_checker_tier.py`, `rp_checker_tier_results_2026-10-11.json` (Z3
  checker and results)
- `rp_checker_tier_2026-10-11/fn_rate_by_tier.py`, `fn_rate_by_tier_2026-10-11.json` (Part 2b
  v1, superseded -- kept for the disclosed `pred==0` bug record, not used for any claim)
- `rp_checker_tier_2026-10-11/fn_rate_by_tier_v2.py`, `fn_rate_by_tier_v2_2026-10-11.json`
  (Part 2b v2, current: corrected false-negative rate by tier, benign-corpus likelihood ratio,
  Bayes $\pi$-sensitivity, and the conservative CP95-bound real-vs-naive-variant comparison;
  canonical copies of both JSONs also at `~/paper_backups/submit_paper2/evidence/`)
- `rp_checker_tier_2026-10-11/leakage_investigation_2026-10-11.md` (the data-leakage
  investigation's full step-by-step trace, summarized in Part 2b above)
- `rp_checker_tier_2026-10-11/risk_tier.go` (chaincode; canonical copy also at
  `~/fabric-network/authcc/chaincode/risk_tier.go`)
- `rp_checker_tier_2026-10-11/risktier_aws_main.go` (AWS-side independent re-implementation;
  canonical copy on the AWS instance only, never transmitted back with its key)
- `rp_checker_tier_2026-10-11/risktier_onchain_results_2026-10-11.json`,
  `risktier_aws_onchain_results_2026-10-11.json` (raw `ReadDecision` records, all 9 real
  transactions)
