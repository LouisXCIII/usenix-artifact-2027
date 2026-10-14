# Task: red-team six destination-side decision functions (specification only)

You play an ADVERSARY. Below are the specifications of six destination-side decision functions that decide whether an authorization decision made in a source domain may be accepted at the destination. Somebody
has implemented them. You have not seen the implementations and must not look for them. Write test cases that would expose an implementation that is subtly wrong: one that ACCEPTS something the specification forbids
(the dangerous direction), or refuses something it permits.

## Common part (all six functions)
Each function takes the common arguments `decisionId, sourceDomain, role (string), amount (int64), sourceLimit (int64), semBound (bool), evAssuranceOK (bool)` plus target-specific arguments (below), and returns a class:
EXACT, SAFE, CONDITIONAL, REEVALUATE or REJECT (or an error for a repeated decisionId, which you do not need to test). For this exercise only ONE distinction matters: is the decision ACCEPTED (class EXACT or SAFE) or NOT ACCEPTED
(any other class or an error)? A decision is ACCEPTED if and only if ALL of the following hold, otherwise it is NOT ACCEPTED:
1. `semBound` is true AND `evAssuranceOK` is true.
2. `role` is exactly one of the destination roles (case-sensitive, no trimming): `manager` (cap 500000), `director` (cap 2000000), `clerk` (cap 10000). Any other string, including the source-side names `gerente` and `gerente_senior`,
   is unrecognised: do not use those two names in your cases.
3. `amount` is at most the cap of the role (amounts are positive integers; `amount == cap` is allowed; `sourceLimit` is informational and never matters).
4. The target-specific check below holds.
These functions have no "second approval" rule; only the target-specific check is added to conditions 1-3. When several conditions fail at once the result is simply NOT ACCEPTED (you do not need to predict which class).

## The six targets and their target-specific check
- `action` — extra argument `actionLevel` (int64). The check holds iff `0 <= actionLevel <= 2` (destination cap on the claimed action level is 2; levels 0, 1, 2 are permitted; 3 and above are not; negative levels are outside the domain).
- `resource` — extra argument `resourceID` (int64). The check holds iff `resourceID == 2` (the single pinned, accepted equivalence entry; every other id, e.g. 0, 1, 3, 99, is not accepted).
- `context` — extra argument `contextOK` (bool). The check holds iff `contextOK` is true.
- `obligation_v2` — extra arguments `obligationProfile` (string), `mappedMechanism` (string), `obligationDischarged` (bool), `enforcementWitnessVerified` (bool).
  Strength ordinals: profile (what the source obligation demands): `empty`=0, `notification`=1, `audit`=2, `dual_approval`=3, `combined`=4. Mechanism (what the destination actually maps the obligation to): `none`=0 (the obligation was
  dropped), `notification`=1, `audit`=2, `dual_approval`=3, `combined`=4. The check holds iff the profile is `empty`, OR (the mechanism is not `none` AND mechanism strength >= profile strength AND `obligationDischarged` is true AND
  `enforcementWitnessVerified` is true). (Dropped obligation = stripping; weaker mechanism = weakening; claimed-discharged without a verifiable witness = laundering.)
- `assurance_v2` — extra arguments `claimedTier`, `verifiedTier`, `destAssuranceRequirement` (strings). `claimedTier` in {classical, hybrid, pq}; `verifiedTier` in {classical, hybrid, pq, unverifiable}; requirement in
  {ClassicalDecisionAuth, PQDecisionAuth}. Only what was independently VERIFIED counts; the claimed tier is irrelevant. Claims granted by a verified tier: classical -> {ClassicalDecisionAuth}; hybrid -> {ClassicalDecisionAuth, PQDecisionAuth};
  pq -> {PQDecisionAuth}; unverifiable -> {} (nothing). The check holds iff the requirement is among the claims granted by `verifiedTier`.
- `assurance_a6` — extra arguments `decisionSigTier` (in {classical, hybrid, pq}) and `destAssuranceRequirement` (in {ClassicalDecisionAuth, PQDecisionAuth}). Claims granted by the tier are as above; the check holds iff the requirement
  is among the claims granted by `decisionSigTier`.
Strings outside the listed sets (other tiers, other mechanism names, wrong case, empty string) are outside the domain: put them in categories starting `out_of_domain_` and give your best reading, or `UNSPECIFIED`.

## What to produce
One JSON file `output/cases.json`: a list of objects
`{"id":"B001","target":"action","category":"...","decisionId":"...","sourceDomain":"besu","role":"manager","amount":1000,"sourceLimit":1000,"semBound":true,"evAssuranceOK":true,"args":{"actionLevel":2},"expected":"ACCEPT","rationale":"one line"}`
- `expected` is `ACCEPT` or `NOT_ACCEPT` (or `UNSPECIFIED`), your reading of the specification. `args` holds exactly the target-specific arguments named above for that target (use the exact key names).
- `id` and `decisionId` unique across the whole file.
- At least 150 cases PER TARGET (900+ in total). Include, for every target: the family check passing while exactly one of conditions 1-3 fails (each of them in turn: semBound false, evAssuranceOK false, unrecognised role such as `Manager`,
  `manager ` or `admin`, amount = cap+1); the family check failing while conditions 1-3 all hold; boundary values (cap, cap+1, cap-1; action levels -1, 0, 2, 3, 4, and huge values; resource ids 0, 1, 2, 3, 99, -1, int64 limits); and both attestations
  together with a failing family check. For `obligation_v2` include ALL 100 combinations of profile x mechanism x discharged x witness with conditions 1-3 holding (category `exhaustive_obligation`), plus at least 50 more. For `assurance_v2` include
  all 24 combinations of claimed x verified x requirement with conditions 1-3 holding (`exhaustive_assurance_v2`) plus at least 126 more. For `assurance_a6` include all 6 combinations (`exhaustive_assurance_a6`) plus at least 144 more that vary the base conditions.
- Prefer diversity over near-duplicates in the non-exhaustive part; aim for cases where two reasonable implementations would disagree.

## Rules
Work only inside this directory. Do not read or search any other file on the machine. Do not use the web. Write `output/cases.json` (a small generator script is fine), check that it parses, print the number of cases per target and per category,
and reply in at most 6 lines: totals per target, and anything in the specification you found ambiguous.
