# General-purpose SMT (Z3) checker for the R_P semantic contract (2026-10-10)

Answers the top-tier review's priority-#1 recommendation ("Biến $R_P$ từ giả thiết thành thứ
được kiểm chứng... Xây một mapping checker bằng Z3... chạy checker trên các ánh xạ thực tế và
trên ánh xạ bị đột biến. Báo cáo bao nhiêu ánh xạ 'ngây thơ' gây widening").

## What already existed, found before writing anything new

`~/pqc-hardhat/experiments/portability/e1_oracle.py` (2026-09-23) already uses Z3 to check
UNSAT of `(psi_s AND NOT gamma_d)` for 50 (pair_type x mapping_type) combinations, using the
real deployed thresholds (`clerk=10000, manager=500000, director=2000000`). Result: 24/50
sound, 26/50 unsound, already in the manuscript's "Formal benchmark" paragraph. But this is 4
separate, near-duplicated Python functions (one hand-coded Z3 query per guard family), and its
50-combination result was never explicitly connected back to Theorem 1's $R_P$ premise for any
concrete real-system instance -- the actual gap the review names ("$R_P$ là hypothesis, không
được suy ra hay kiểm chứng").

**Key finding from reading the manuscript's formal model (Section~sec:semantics, line
508-533) before designing anything**: in the real deployed system, the `RoleThreshold`/amount
guard is explicitly a **destination-local fact**, not $R_P$-mediated at all -- the source
domain's own recorded predicates are only `mlScanPassed` and `approvalCount>=k`; it never
evaluates a role/amount threshold. So $R_P$, strictly, has no real-system instance for the
RoleThreshold fragment to discharge in the first place; the honest framing is that this guard
is *vacuously* outside $R_P$'s scope by design (safer than relying on $R_P$, not weaker).

## What is new: `rp_checker.py`

One generic entailment-checking core, `check_entailment(domain, psi_s, gamma_d, free_vars)`,
replacing the 4 repeated Solver-building blocks in `e1_oracle.py` with shared code. Guard
families still need their own symbolic encoding (inherent to SMT checking -- different guards
are different formulas), but the actual UNSAT/SAT decision, witness extraction, and reporting
are unified.

**Regression discipline (mandatory before trusting anything new)**: reran all 50 of
`e1_oracle.py`'s combinations through the new generic core. **0 mismatches, 24/50 sound,
exact reproduction of the 2026-09-23 baseline.** Only after this passed were new results
trusted.

**New beyond e1_oracle.py**:

1. **Obligation mechanism-strength fragment** (real table from
   `~/fabric-network/authcc/chaincode/obligation_v2.go`, `oblCoveredV2`): checked whether the
   real system's identity-name obligation mapping (profile "audit" maps to mechanism "audit",
   etc. -- the only mapping policy the deployed system actually uses) is $R_P$-sound.
   **First attempt was methodologically wrong and caught before trusting it**: left the
   destination's chosen mechanism as an unconstrained free variable, so Z3 "found" the
   deliberate obligation-stripping case (profile=notification, mechanism=stripped) as a
   counterexample -- but that is exactly the case the real guard exists to REJECT, not a
   genuine $R_P$ violation. Root cause: $\psi_s$ must describe what the source establishes;
   the destination's mechanism choice is determined by a concrete mapping $M_I$, not a free
   variable. Fixed by encoding the real identity mapping explicitly. Result: **sound** --
   checker-verified, 0 counterexamples.
2. **The real deployed `RoleMapping` as a concrete instance** (`gerente`->`manager`,
   `gerente_senior`->`director`, real caps), checked against the one scenario where this guard
   *would* need to go through $R_P$: a hypothetical design where the destination trusted a
   source-claimed cap unchecked instead of re-evaluating its own (the exact naive design the
   Introduction's widening example warns against). **Result: unsound for both real mappings**
   (source cap can range above the real destination cap) -- this is the formal confirmation of
   *why* the real system's destination-local design is the correct one, not a flaw in the
   deployed system itself (which does not do this).
3. **Mechanical "naive mapping" variants** (hand-enumerated, not AI-generated), each checked in
   **both directions** against the real correct guard -- widening (naive guard accepts
   something the real guard would reject: an actual safety violation) versus over-conservative
   (naive guard rejects something the real guard would accept: an availability bug, not a
   safety one). This distinction matters: lumping every mismatch together as "widening" would
   overstate the finding.

## Results: naive mapping variants (6 checked)

| Variant | Result | Witness / note |
|---|---|---|
| V1 trust source cap unchecked (gerente) | **WIDENING** | amt=500001, src_cap=500001 (exceeds real manager cap 500000) |
| V1 trust source cap unchecked (gerente_senior) | **WIDENING** | amt=2000001, src_cap=2000001 (exceeds real director cap 2000000) |
| V2 reversed threshold comparison | **WIDENING** | role=1 (clerk), amt=10001 (exceeds real clerk cap 10000, reversed check would accept) |
| V3 off-by-one role-hierarchy shift applied to an identity case | over-conservative only | shifting down only makes the guard stricter, never unsafe -- correctly NOT counted as widening |
| V4 ignores the `required==0` (empty profile) edge case | over-conservative only | always demands >=1 even when the source required nothing -- rejects a legitimate case, not unsafe |
| V5 trust caller claim unchecked (S11b pattern) | over-conservative only (see note) | the ordinal `mapped>=required` check is a tautology under the real identity mapping, so no real-reject point exists to diverge from in this numeric fragment -- S11b's actual vulnerability (binary discharge-event authenticity) is outside what this ordinal fragment expresses; disclosed honestly rather than claiming a false positive |

These 6 variants are hand-picked worked examples demonstrating the checker can tell widening
apart from over-conservative behavior -- **not a random or representative sample**, so "3 of 6"
is not a frequency/rate claim about naive implementations in general (a review caught an
earlier draft of the manuscript text implying this; fixed there and stated explicitly here).
**3 show genuine widening** (V1 x2, V2); **3 show only over-conservative behavior**, correctly
distinguished rather than inflated into "widening."

## Corollary this discharges for Theorem 1

Does **not** modify Theorem 1 itself (the general relational result, mechanized in Coq at the
model's abstraction level). Adds a new corollary, with the entailment direction corrected
after an external review caught it stated backwards in an earlier draft: the semantic contract
is $(\psi_s,\gamma_d)\in R_P \Rightarrow$ entailment, a one-way implication, so a checker
cannot "derive" membership from it by converse. What a checker-verified UNSAT genuinely
licenses is the other, valid direction: if `rp_checker`'s `check_entailment` verifies UNSAT of
$\psi_s(I,\hat\sigma_s)\land R_\Sigma(\hat\sigma_s,\hat\sigma_d)\land\lnot\gamma_d(M_I(I),\hat\sigma_d)$
over $\Omega$ for concrete $\psi_s,\gamma_d,M_I,R_\Sigma$, then **admitting** $(\psi_s,\gamma_d)$
into $R_P$ satisfies the contract (inclusion cannot violate an implication whose antecedent
this instance is shown to make the consequent hold for) -- a constructive/introduction move,
not a backward derivation from the contract. Theorem 1's $R_P$ hypothesis is discharged for
that instance under this reading.

**$R_\Sigma$/$M_I$ encoding, stated explicitly (a review-caught gap in the first draft)**:
source and destination variables are identified directly in each family's Z3 encoding, i.e.
$R_\Sigma$ is encoded as the identity relation on the shared comparable quantity (amount,
action level, resource id, context flag); $M_I$ is the explicit mapping function applied
inside $\gamma_d$. A more general, independently-varying $R_\Sigma$ is not encoded. Each
family's correspondence between its Z3 encoding and the real Go guard is itself an unverified
assumption, the same category of gap as the Coq-mechanization-vs-chaincode correspondence
disclosed elsewhere in this project -- not something this checker establishes.

**The real system has no non-trivial $R_P$ instance yet (the most important finding of this
review round, not a weakness to hide)**: `RoleThreshold`/amount is destination-local (no real
instance for $R_P$ to discharge at all); the real obligation mapping is an identity mapping
under which `mapped>=required` is a tautology (also not non-trivial). What this checker
verifies today is that the real, trivial fragments are sound, and that hypothetical
alternative designs needing a genuine $R_P$ would be unsound -- confirming why the real
destination-local design is correct, not exhibiting a real non-trivial $R_P$ case in
production. A genuinely non-trivial instance needs a guard the destination cannot evaluate
from its own state alone (e.g. a zero-knowledge-proven private source predicate accepted via
a verified mapping rather than re-derived) -- which the current workload does not have. This
directly validates the point made by the review that caught this: Trụ 1 (this checker) and
Trụ 4 (a privacy-sensitive ZK-proven workload) are linked -- without Trụ 4, this checker has
no real non-trivial case to verify yet; it is a general-purpose tool proven correct on real
trivial fragments and clear on hypothetical ones, ready to apply the moment a real non-trivial
$R_P$ instance exists.

Scope: only the quantifier-free linear-arithmetic / enumerated-type fragment the checker can
express (bounded integers, finite enums, boolean) -- the same honest fragment restriction the
paper already applies to the formal benchmark; no claim of checking more than this decidable
fragment reaches. No Coq soundness proof of the checker itself was attempted (out of scope per
the review's own guidance: "Z3 cộng kiểm thử là đủ").

## Scope and what this does not show

- Does not cover every guard in the deployed system (time/freshness windows, action
  allowlists reuse `e1_oracle.py`'s existing untouched families; not re-verified here).
- The obligation fragment's "sound" result is specific to the real identity-name mapping
  policy; it does not claim soundness for an arbitrary renaming policy between obligation
  vocabularies (the real system does not have one, unlike roles).
- V5's non-result is a genuine limit of the ordinal fragment, not evidence that trusting an
  unchecked caller claim is safe in general -- it is not, as S11b itself already demonstrated
  in a different part of the real system.

## Files

- `evidence/rp_checker/rp_checker.py` (the checker, canonical copy also at
  `~/pqc-hardhat/experiments/portability/rp_checker.py`)
- `evidence/rp_checker/rp_checker_results_2026-10-10.json` (full output: regression table,
  obligation fragment, real registry mapping, naive variants)
- `evidence/rp_checker/e1_oracle_table_2026-09-23.json` (baseline table used for the
  regression check)
