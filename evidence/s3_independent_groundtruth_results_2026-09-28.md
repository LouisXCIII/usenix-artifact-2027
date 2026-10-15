# S3 -- Independent ground truth for the deployed classifier: RESULTS (2026-09-28)

Protocol: `s3_independent_groundtruth_protocol_2026-09-28.md` (frozen before any adversarial case existed; amendments A1, A2 are dated and explained there).
Code and raw outputs: `s3_independent/`. System under test: the UNMODIFIED deployed Go functions `SubmitDecisionFull` and `SubmitDecision` from
`~/fabric-network/authcc/chaincode/authchaincode.go` (SHA-256 e50ed4b1c4b837b561ac879c9bfad20f79ba6139707e78c3763575f6af4e51b9, recorded before and after every run), run
OFFLINE with an in-memory stub replacing the ledger. Independent oracle: Cedar policies (`cedarpy`) written from the specification, not from the code. Adversary: a
fresh Opus agent given only `s3_independent/adversary_SPEC.md` (no code, tests or results).

## Result
| Suite | Cases | In-domain (primary) | Match | Unsafe accepts | Over-conservative | Class mismatch | 95% upper bound (0 unsafe) |
|---|---|---|---|---|---|---|---|
| Adversary (independent), full function | 522 | 480 | 480 | 0 | 0 | 0 | 0.622% of these cases |
| Sampler (mine, NOT independent), full function | 6000 | 5741 | 5741 | 0 | 0 | 0 | 0.052% of these cases |
| Adversary, `SubmitDecision` (simpler policy, A1) | 522 | 480 | 480 | 0 | 0 | 0 | 0.622% |
| Sampler, `SubmitDecision` | 6000 | 5741 | 5741 | 0 | 0 | 0 | 0.052% |
The bound is over the cases tried, which are adversarial and not a random sample; it is NOT a population rate.
Not in the primary metrics: 7 adversary and 49 sampler cases outside the 64-bit range (skipped); 35 adversary and 210 sampler out-of-domain cases (see below).
The adversary produced 19 categories (role-string tricks 126, sourceLimit informational 72, int64 limits 48, cap boundary 40, half-cap boundary 40, cross-role cap 36, attestation
precedence 33, replay 18 plus 7 replay set-ups, registry-mapped caps 18, unneeded approval 15, decisionId look-alikes 9, and others). Its own expected class agreed with the Cedar oracle in all 480 in-domain
cases (500 of 515 evaluated overall; the 15 differences are the out-of-domain artifact below), so the specification was read the same way by two independent readings (the adversary's and Cedar's) in-domain,
and the deployed code agreed with both. All 25 cases in the replay categories matched, as did the 9 decisionId look-alikes.

## Power of the test: injected defects
11 single-line defects injected into a scratch copy of the Go code (`s3_independent/mutation_check.py`, `mutation_result.json`): cap `>`->`>=`, cap+1 allowed, obligation skipped for the upper half,
obligation from exactly half, obligation at a third, second-approval flag ignored, semBound not checked, evAssuranceOK not checked, replay check removed, registry entry corrupted
(gerente -> director), policy datum corrupted (clerk cap 100000). Control (no defect): clean. Detected: 11 of 11 by the adversary suite and 11 of 11 by the sampler. Unsafe-type defects
produced unsafe accepts (for example 34-38 adversary cases for cap+1 / skipped obligation); over-conservative defects produced over-conservative verdicts. The deployed source hash was unchanged afterwards.
This shows the suite can see this kind of defect; it does not show it can see every kind.

## What went wrong on the way (kept, not hidden)
The first sampler run reported 87 "unsafe accepts". All 87 were defects in MY ORACLE (it ignored the case's actual second-approval flag) plus one bookkeeping error for skipped cases. They were fixed from the
specification text before the adversary's cases were run (protocol A2; the first summary is kept as `run1_sampler_summary_ORACLE_DEFECTIVE.json`). A mismatch is a finding only after the oracle itself has
been checked against the specification.
Separately, 15 adversary out-of-domain cases with extreme negative amounts were flagged because Cedar's 64-bit `amount * 2` overflows and denies; the SUT and the adversary agree on EXACT/SAFE there. That is an
artifact of my Cedar encoding, not a finding about the classifier.

## Observation outside the specification (not counted as a defect)
The deployed `SubmitDecisionFull` accepts non-positive amounts. In the adversary's probes, 26 of 26 cases with amount 0 (5) or negative (21), a recognised role and both attestations true were returned EXACT or SAFE (16 EXACT, 10 SAFE),
because `amount > threshold` and `amount > threshold/2` are both false for them. The specification is defined for positive amounts, so this is an input-validation gap, not a violation. Whether it matters depends on
whether a source can bind a non-positive amount into a decision that the destination then executes. A one-line hardening (reject amount <= 0) is suggested; it has NOT been applied and would need a chaincode upgrade.

## What this does and does not show
- It shows that, for the role / amount / obligation / attestation-precedence / replay fragment, the deployed Go classifier agrees with an independent semantics on 480 adversarial and 5741 sampled in-domain cases, and that
  the labels no longer come from the classifier's own formula. This partly closes the "circular ground truth" threat for THIS fragment, and it removes the objection that the E1 benchmark runs a re-implementation instead
  of the deployed code.
- It does NOT repair the E1 benchmark itself (its labeler still mirrors its classifier); the paper's wording that E1 is a formal regression test remains correct.
- It does NOT test: the light-client / zero-knowledge proof paths, the action / resource / context families, multi-hop, mapping-registry governance (a malicious registry entry is out of scope by assumption A7), or anything the
  specification does not state.
- The policy tested is small (three roles, two registry names), so a clean result was expected (stated in the protocol); the informative parts are the independent labels, the power check and the observation above.
- The adversary is one AI model given only the specification; its cases are diverse (522) but not exhaustive.
- Expressiveness is a separate question: this says nothing about whether the classifier can decide real-world policies (roadmap item #2).

## Suggested paper change (not yet made)
Add a short subsection "Independent validation of the deployed classifier" stating the setup, the table above, the 11/11 power check and the negative-amount observation, and cite it in the internal-validity paragraph as partial mitigation of the
shared-logic threat, limited to the tested fragment.
