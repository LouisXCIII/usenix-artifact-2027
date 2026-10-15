# E3 limitations, resolved where resolvable (2026-09-23)

Continuing the same "why + fix or explain" pass from E1, applied to E3's
three disclosed limitations
(`evidence/e3_s0s18_crosswalk_2026-09-23.md`).

## (1) Sample sizes come from post-hoc slicing, not purpose-built cases -- FIXED

**Why it existed**: S2-S17's evidence was a post-hoc filter of E1's
incidentally-stratified N=2,000 corpus (built for E1's own purposes), not a
set of cases deliberately constructed to stress each scenario.

**Fix**: `e3_targeted_scenarios.py` -- 30 purpose-built, boundary-focused
cases across 13 scenarios (S0,S1,S2,S3,S4,S8,S9,S10,S12,S13,S14,S15,S17),
each constructed directly (not filtered from a shared pool), scored with
the same real, already-validated logic (`e1_ground_truth.py`/
`e1_classifier.py` -- no new classification code). Includes exact-boundary
cases random sampling would rarely hit precisely, e.g. S3's amount swept to
exactly the destination threshold (499{,}999 / 500{,}000 / 500{,}001 ->
SAFE / SAFE / REJECT) and S14's finality state swept across
below/at/above. **Result: 30/30 correct, 0 unsafe accepted.** Raw output:
`evidence/e3_targeted_scenarios_result_2026-09-23.json`.

## (2) S4/S6/S7/S16 share evidence with an adjacent scenario

**S2 vs S4 -- FIXED, and it turned out to be a reporting gap, not a data
gap**: re-querying the existing E1 corpus by `mapping_type` shows S2
(`role_widening`) and S4 (`action_expansion`) were always separately
distinguishable in the underlying data (200 cases each, 0 unsafe accepted
in either) -- they were only ever *reported* as one combined number.
Corrected to report them separately.

**S5/S6/S7 (obligation stripping/weakening/laundering) and S15/S16
(honest vs. fake assurance-tier claim) -- FIXED FOR REAL, by extending the
formal model (2026-09-23, later same session)**: the user asked for the
model to actually be extended rather than left as an explained gap. Two
new, additive modules were built (neither modifies `e1_classifier.py` /
`e1_ground_truth.py`, so the already-reported N=2,000 campaign is
untouched):

- `e3_obligation_model.py`: replaces the single "discharged" boolean with
  four independent facts -- the source's *required* mechanism strength
  (ordinal: none < notification < audit < dual\_approval < combined), the
  destination's *mapped* mechanism (or `None` if $M_O$ drops it entirely --
  S5), that mechanism's strength (S6 fails when it is weaker than
  required), and whether the discharge claim carries an independently
  verifiable enforcement witness (S7 fails when it does not, i.e. it is
  metadata-only). **7/7 cases correct**, including two boundary controls
  (a destination mechanism exactly as strong as required, and one
  strictly stronger) that a coarser model could not have distinguished
  from a "weakened" case. Raw output:
  `evidence/e3_obligation_model_result_2026-09-23.json`.

- `e3_assurance_claim_model.py`: separates the *claimed* assurance tier
  from the *independently verified* tier (mirroring the real E5 result,
  `evidence/e5_pq_profiles_2026-09-23.md`, where a tampered signature
  contributed no claim regardless of what was asserted); claims are
  computed from the verified tier only, so an over-claim (S16: adapter
  claims hybrid/pq, verification only confirms classical) is rejected
  exactly like S15's honest classical report, while a genuinely verified
  upgrade is correctly accepted. **6/6 cases correct**, including an
  "unverifiable claim" case (verified tier `None`) contributing zero
  claims. Raw output:
  `evidence/e3_assurance_claim_model_result_2026-09-23.json`.

Both models are new, standalone formal extensions -- not yet folded into
the core N=2{,}000 E1 campaign or an on-chain guard (that would be the
natural next step, following E1's context/action/resource pattern, but
was not done this pass).

## (3) S18 has only n=1 real evidence -- STRENGTHENED (real cross-reference, no new work needed)

**Why it existed**: S18 (multi-hop stripping) had only X5a's single real
on-chain counterexample.

**Resolution**: E9's already-completed work
(`evidence/e9_multihop_extension_2026-09-23.md`) is, structurally, exactly
this scenario generalized: context dropped at an intermediate hop, tested
at every possible injection position across h in {1,2,3,4,5,10} -- **25
real injected-loss cases, 0/25 accepted end-to-end** under the composed
check. This was already real and complete; it had simply not been
cross-referenced from S18's crosswalk entry. No new experiment was needed,
only the citation.
