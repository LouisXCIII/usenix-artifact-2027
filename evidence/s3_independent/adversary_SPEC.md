# Task: red-team a destination authorization gate (specification only)

You play an ADVERSARY. Below is the specification of a destination-side authorization gate that decides whether an authorization decision made in a
different (source) domain may be reused at the destination. Somebody has implemented this specification. You have not seen the implementation and must not
look for it. Your job is to write test cases that would expose an implementation that is subtly wrong: one that accepts something the specification forbids,
or rejects something it permits. Think like an attacker and like a tester who has seen many off-by-one, parsing and precedence bugs.

## The function under test
`Decide(decisionId, sourceDomain, role, amount, sourceLimit, semBound, evAssuranceOK, secondApprovalObtained) -> class`
Types: decisionId, sourceDomain, role are strings; amount and sourceLimit are 64-bit signed integers; the other three are booleans.

## Specification of the destination policy
- Destination roles and caps (integer amounts, one currency unit): `manager` 500000, `director` 2000000, `clerk` 10000.
- Role names are compared as exact, case-sensitive strings. There is no trimming and no normalisation of any kind.
- A source role name that is already a destination role name is IDENTITY-mapped. A mapping registry translates exactly two other source names:
  `gerente` -> `manager`, `gerente_senior` -> `director`. Any other role string is unrecognised.
- Two attestations must both be true: `semBound` and `evAssuranceOK`. If either is false the class is REJECT, whatever else holds. This is checked first.
- Then: an unrecognised role gives REEVALUATE (the destination has no policy for it and defers to local evaluation).
- Then: if amount is greater than the cap of the (mapped) role the class is REJECT (it would widen authority).
- Then: obligation. If amount is strictly greater than half of the cap, a second approval is required. If `secondApprovalObtained` is false the class is CONDITIONAL.
- Otherwise the decision is permitted: EXACT if the submitted role was already a destination role name (identity mapping), SAFE if it was translated through the registry.
- `sourceLimit` is informational only and never changes the class.
- A `decisionId` that has already been submitted earlier in the same run is a replay: the function returns an error (class value REPLAY_ERROR) and changes nothing.
- The specification is defined for positive integer amounts and non-empty role strings. You may also probe outside that domain (zero, negative, empty role, huge values); label those cases with category
  starting `out_of_domain_` and give your best reading of what a correct implementation should do, or `UNSPECIFIED`.

## Classes
EXACT, SAFE, CONDITIONAL, REEVALUATE, REJECT, REPLAY_ERROR (only for a repeated decisionId).

## What to produce
At least 400 cases in one JSON file `output/cases.json`: a JSON list of objects
`{"id": "A001", "category": "...", "decisionId": "...", "sourceDomain": "...", "role": "...", "amount": 0, "sourceLimit": 0, "semBound": true, "evAssuranceOK": true, "secondApprovalObtained": false, "expected": "REJECT", "rationale": "one line"}`
- `id` unique. `decisionId` unique per case, except when you deliberately test replay: then repeat an earlier case's `decisionId` exactly and set `expected` to REPLAY_ERROR.
- `amount` and `sourceLimit` are JSON integers. Values outside the signed 64-bit range are allowed only in cases with category `out_of_domain_overflow`.
- `expected` is YOUR reading of the specification. Be careful: it will be compared against an independent reading, and disagreements will be studied.
- Cover at least: cap boundaries (cap-1, cap, cap+1) for every role and for translated roles; the half-cap boundary (half, half+1, half-1) with and without the second approval; both attestations
  false / one false / both true, including together with an unrecognised role or an over-cap amount (precedence); role-string tricks (case changes, leading/trailing space, tab, zero-width and other unicode look-alikes,
  prefixes and suffixes of valid names such as `manager2`, `Manager`, `gerente `, `gerente_senior_x`, look-alikes of the registry names); the two registry names at and around their mapped caps; amounts near the
  64-bit limits; `sourceLimit` smaller than, equal to and larger than `amount`, and negative; a second approval supplied when it is not needed; repeated `decisionId` (replay), including a replay whose fields differ.
- Prefer diversity over volume: do not pad with near-duplicates. Aim for cases where two reasonable implementations would disagree.

## Rules
Work only inside this directory. Do not read or search any other file on the machine. Do not use the web. Write `output/cases.json`, check that it parses, and reply in at most 5 lines: number of cases, categories used,
and anything in the specification you found ambiguous.
