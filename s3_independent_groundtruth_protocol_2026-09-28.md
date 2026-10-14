# S3 -- Independent ground truth for the deployed classifier (PROTOCOL, frozen 2026-09-28)

Roadmap item #3. Reviewer question: "Is FAR = 0 an artifact of a circular ground truth?" Written BEFORE any adversarial case
was generated or run. Later changes go under "Amendments" with date and reason.

## Where the circularity is (verified by reading the code today)
- `e1_ground_truth.py` (labeler) and `e1_classifier.py` (system under test) implement the SAME conjunction and the SAME class-selection
  logic; the labeler reads a precomputed `sem_covered_instance` field. A zero-FAR result therefore shows the two reproduce one formula.
- The E1 benchmark runs a Python re-implementation, not the deployed Go code. The real on-chain corpus (n=190) runs the Go code but its
  ground truth is computed by the same formula.

## System under test (SUT)
The deployed Go decision function `SubmitDecisionFull` (and, as a second target, `SubmitDecision`/`classify`) from
`~/fabric-network/authcc/chaincode/authchaincode.go`, executed OFFLINE with an in-memory stub replacing the ledger (only `GetState`/`PutState`).
The decision code is unmodified: the harness copies the chaincode tree into a scratch directory and adds one test file; the SHA-256 of
`authchaincode.go` is recorded in the results. No Fabric network is touched.

## Independent oracle
1. Semantics of the destination policy P_d are written as CEDAR policies (engine: `cedarpy`, an independent, formally verified implementation),
   from the specification below, NOT from the Go code.
2. Oracle class = assembled from four Cedar decisions per case (A: second approval absent; B: second approval present; C: role recognised, tested with
   a small positive amount and true flags; D: attestations) plus the spec's ordering rule. The assembly is spec-level logic in the harness; the role,
   cap and obligation semantics are Cedar's.
3. A second, independent reading: the adversary's own expected class per case (below). Disagreements between the adversary and the oracle are
   reported as SPEC AMBIGUITIES, not as SUT defects.

## Specification of P_d (given verbatim to the adversary; contains no code)
- Destination roles and caps (integer amounts, same currency unit): `manager` 500000, `director` 2000000, `clerk` 10000.
- Role names are compared as exact, case-sensitive strings; no trimming or normalisation.
- A source role name that is already a destination role name is IDENTITY-mapped. The mapping registry translates exactly two other source names:
  `gerente` -> `manager`, `gerente_senior` -> `director`. Any other role string is unrecognised.
- Two attestations must both be true, `semBound` and `evAssuranceOK`; if either is false the class is REJECT, whatever else holds (checked first).
- Then: unrecognised role -> REEVALUATE (the destination has no policy for it; defer to local evaluation).
- Then: amount greater than the cap of the mapped role -> REJECT (would widen authority).
- Then: obligation. If amount is strictly greater than half of the cap, a second approval is required; `secondApprovalObtained=false` -> CONDITIONAL.
- Otherwise permitted: EXACT when the submitted role was identity-mapped, SAFE when it was translated through the registry.
- `sourceLimit` is informational and never changes the class.
- A decision ID that already exists is rejected as a replay (an error is returned, nothing is overwritten).
- In-domain inputs: amount is a positive integer, role is a non-empty string. Inputs outside this domain are reported separately (see below).

## Adversary
A fresh agent of a different model, given ONLY the specification above, the function signature
`SubmitDecisionFull(decisionID, sourceDomain, role, amount, sourceLimit, semBound, evAssuranceOK, secondApprovalObtained) -> class` and the class
definitions. It is not given source code, tests, earlier results or the oracle. Task: produce at least 300 cases designed to make a plausible
implementation either accept something the specification forbids or reject something it permits, covering at least: cap boundaries and off-by-one,
half-cap boundary, mapping abuse (foreign roles that resemble destination roles; case, whitespace, unicode, prefix/suffix variants), integer extremes,
flag combinations and their precedence, obligation with over-cap amounts, sourceLimit inconsistent with amount, replay of a decision ID. For each case it
writes its own expected class and a one-line reason. Case format: JSON list of
`{id, category, decisionId, sourceDomain, role, amount, sourceLimit, semBound, evAssuranceOK, secondApprovalObtained, expected, rationale}`.
Volume complement: a property-based sampler (written by me, so NOT independent of my knowledge of the code) adds several thousand cases around the same
boundaries; it is reported separately from the adversary's cases.

## Metrics (defined now)
- UNSAFE ACCEPT: SUT returns EXACT or SAFE while the oracle says not unconditionally permitted, or SUT returns CONDITIONAL while the oracle says the case is
  denied even with the second approval.
- OVER-CONSERVATIVE: SUT denies or defers a case the oracle unconditionally permits.
- CLASS MISMATCH: any other difference from the oracle class.
- Reported per category and overall, for the adversary's cases and the sampler's cases separately, with the exact one-sided 95% Clopper-Pearson bound for
  zero events. The bound is over the cases tried, which are adversarial and not a random sample; it is NOT a population rate.
- Out-of-domain inputs (amount <= 0, empty role, absurd lengths) are run and reported as observations; they do not enter the primary metrics.

## Scope (declared before running)
Only the role / amount / obligation / T0-precedence / replay fragment of `SubmitDecisionFull` and `SubmitDecision`. NOT tested here: the light-client and
zero-knowledge proof paths, the action/resource/context families (`action.go`, `resource.go`, `context.go`), multi-hop, mapping-registry governance (a
malicious registry entry is out of scope by assumption A7), and any property beyond the specification above.

## Interpretation, fixed in advance
- Any unsafe accept on an in-domain case is a defect in the SUT or a spec ambiguity to be resolved openly; either outcome is reported.
- Zero unsafe accepts across the adversary's cases is evidence that, for THIS FRAGMENT, the earlier zero-FAR result does not depend on a circular label. It is
  not a proof, it says nothing about fragments not tested, and it does not show the classifier is expressive enough for real policies.
- Because the fragment is small, a clean result is expected; the value of the test is that the labels no longer come from the code's own formula.

## Amendments
- A1 (2026-09-28, made before the adversary's cases existed): the second target `SubmitDecision` (`classify`) implements a SIMPLER policy than the specification given to the adversary:
  no mapping registry, no obligation, and it never returns EXACT. Its oracle is derived from the same Cedar decisions with those three differences (registry names are
  unrecognised; only identity roles are recognised; permitted means Cedar's decision with the second approval present, class SAFE). Results for it are reported separately and
  are read as a check of the simpler function, not of the specification.
- A2 (2026-09-28, after the first sampler run, before the adversary's cases were run): two defects in MY ORACLE were found and fixed. (a) The oracle assembled its class from the
  hypothetical "approval absent" decision and ignored the case's actual `secondApprovalObtained`, producing 87 false "unsafe accepts" (all with the approval present, amount above half
  the cap, oracle CONDITIONAL, SUT EXACT/SAFE). The fix follows the specification sentence directly (with the approval obtained the decision is permitted). (b) A decisionId first seen in a case that the
  harness skipped (amount or sourceLimit outside int64) was counted as "seen" by the oracle only; skipped cases no longer count. The first run's summary is kept as
  `run1_sampler_summary_ORACLE_DEFECTIVE.json`. Lesson recorded: a mismatch is a finding only after the oracle itself has been checked against the specification; both fixes were made from the
  specification text, not by looking at which change would make the SUT pass.
- A3 (2026-09-28, scope extension, written before any case for it exists): the scope is extended to the six other pure decision entry points of the deployed chaincode: `SubmitDecisionWithAction`,
  `SubmitDecisionWithResource`, `SubmitDecisionWithContext`, `SubmitDecisionObligationV2`, `SubmitDecisionAssuranceV2`, `SubmitDecisionWithAssurance`. Their specifications are written from the paper's
  formal model (the docstrings of `e3_obligation_model.py` and `e3_assurance_claim_model.py`, the paper's description of the four mapping families), NOT from the Go bodies, which have not been read.
  The primary property is SAFETY OF ACCEPTANCE only: a case is "accepted" when the class is EXACT or SAFE; the oracle says accept iff the attestations hold AND the role is recognised AND amount <= cap AND the
  family check holds. Precedence among simultaneous failures (which of REJECT / REEVALUATE / CONDITIONAL is returned) is NOT part of the primary metric. Registry role names are excluded from these targets
  (their treatment of the registry is not stated by the paper). One blind adversary (fresh Opus agent, specification only) writes at least 150 cases per target. Same metrics, out-of-domain handling and injected-defect
  power check as above. Independent of the earlier `SubmitDecisionFull` result; reported as a second table.
- A4 (2026-09-28, planned): the finding that `SubmitDecisionFull` accepts non-positive amounts is checked against a one-line hardening (reject amount <= 0) applied to the SCRATCH COPY only; the same suites are re-run to show the
  observation disappears and nothing else changes. The deployed chaincode is not modified.
- A5 (2026-09-28, planned): coverage of real policies. The public Cedar example policy sets are classified by whether each policy is expressible in the classifier's role / amount-threshold fragment; the result is a
  coverage number, not a safety claim.
