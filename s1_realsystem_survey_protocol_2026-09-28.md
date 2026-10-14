# S1 -- Real-system survey: does the destination check its own policy? (PROTOCOL, frozen 2026-09-28)

Written BEFORE any system was inspected, so that the rubric cannot be tuned to the data.
Any change after this point must be appended under "Amendments" with a date and reason,
never edited in place.

## Research question
For deployed cross-domain receivers, when an *authentic* inbound message triggers a state-changing
effect, does the destination (a) check its OWN policy over the concrete effect/parameters, or
(b) rely only on the source's authorization? This is the empirical face of Eq. (1),
Authentic_s(D_s)=1 =/=> Admissible_d(D_s)=1.

## Unit of analysis
A *receiver path*: destination-side code that, on an authenticated inbound message, causes a state change.
Two levels are coded separately, because messaging layers deliver and the application decides:
- L-proto: what the protocol/reference receiver enforces by default.
- L-app: what the first-party or most-cited reference application (e.g. the project's own governance
  receiver) enforces.

## Dimensions (each coded Yes / Partial / No / Unknown; "Unknown" is preferred to a guess)
- D1 Authenticity: the protocol verifies message origin/integrity (expected Yes for all).
- D2 Source binding: the receiver restricts WHICH remote sender/chain is accepted (enforced by
  protocol = Yes; must be written by integrator = Partial; absent = No).
- D3 Effect scope: the destination restricts WHICH actions/targets a message may trigger
  (type/selector/target allowlist). Arbitrary call or delegatecall of attacker/source-chosen
  payload = No.
- D4 Parameter policy: the destination evaluates ITS OWN policy over concrete parameters
  (caps, bounds, rate limits, thresholds) at execution time, independently of the source's authorization.
- D5 Destination delay/veto: a destination-side delay, guardian, or pause that can stop an
  authentic message before it takes effect.
- D6 Current-state evaluation: the check in D4 uses destination state at execution time.

## Tier (derived mechanically from D2-D4; not judged)
- T0  D3=No and D4=No and D2=No  (authenticity-only, unrestricted effect)
- T1  D3=No and D4=No and D2 in {Yes,Partial}  (sender-bound, unrestricted effect)
- T2  D3 in {Yes,Partial} and D4=No  (effect-scoped, no parameter policy)
- T3  D4 in {Yes,Partial}  (destination evaluates its own parameter policy)
"Exposed to the Eq.(1) gap" := tier in {T0,T1,T2}.
Recorded separately, NOT counted as a defect: `by_design_delegation` = Yes when the design
intentionally gives the source domain full authority over the destination (e.g. governance relays).
A T0/T1 design with by_design_delegation=Yes is a deliberate trust decision; the paper must say
so and must not call it a vulnerability.

## Evidence standard
Every coded cell cites a primary source: repository path + commit/tag (or spec section + version).
Quotes <= 15 words. Source snapshots are saved under evidence/s1_sources/ with the commit hash.
Web-page summaries are not primary evidence for code claims; raw source is.

## Inclusion (purposive, NOT random -- so no prevalence estimate is claimed)
Public source or spec; production deployment or production-grade documentation; a cross-domain
receiver that changes destination state. Stratified for diversity over:
(a) general messaging layers, (b) L1<->L2 native messengers, (c) DAO cross-chain governance
executors, (d) IBC/Cosmos and Polkadot XCM, (e) permissioned interoperability (Cacti, Weaver).
Target 12-20 systems. Systems that could not be examined are listed as NOT REACHED, never dropped.

## Secondary corpus (incidents)
Public post-mortems for cross-chain exploits, each coded: authenticity failure
(forged/unauthorised message accepted) vs admissibility gap (authentic message, effect the
destination would not have permitted). Only admissibility-gap cases support the paper's claim;
authenticity failures are reported separately and must not be counted toward it.

## Safety / ethics
Read-only study of public documentation and source. No probing of live systems, no transactions to
mainnet contracts. Any reproduction is on a local fork only and only if a widening case is found.
If a live exploitable case is found: coordinated disclosure before any detail is published.

## Amendments
- A7 (2026-09-28, verification phase): the "read-only public documentation and source" rule is extended to
  cover retrieving the VERIFIED SOURCE of already-deployed contracts from public block-explorer / Sourcify APIs and
  read-only public JSON-RPC calls (`eth_getCode`, `eth_getStorageAt`, and `eth_call` restricted to view functions
  with no state change). Reason: to check whether the code coded at repository HEAD is the code actually deployed.
  Still forbidden: any transaction, any simulation of an exploit, any call to a state-changing function, any
  fork-and-attack of a live system. The purpose is comparison of published code, not testing a live system.
- A8 (2026-09-28, verification phase): a second, independent coder (a different model, given only the protocol, the
  rubric clarifications and the pinned sources, and not the first coder's results) codes the same rows; agreement is
  reported per dimension (Cohen's kappa and raw agreement). This is inter-coder agreement between two AI coders,
  not human agreement, and is reported as such.
- A9 (2026-09-28, after inter-coder disagreement on D2): D2 = Y only if the receiver itself enforces an allowlist or
  enrolled set of remotes; P if the base exposes the remote sender and leaves the check to the application; N if no
  source restriction exists, including designs where authority is isolated in a sender-derived account (recorded as a
  note). Applied to every row; changed R2, R3, R6, R8, R17.
- A10 (2026-09-28): D3 = Y for a fixed-function receiver whose only reachable effect is one action confined by
  construction (token release/mint: R7, R10).
- A11 (2026-09-28): D5 = Y when the destination has a pause, curse or guardian that can stop an authentic message before
  it takes effect; P when there is only a delay without a destination-side veto, or the pause exists on one side only.
- A12 (2026-09-28): R21 (Cacti SATP stage 1, negotiation only) is excluded from the tier counts because it does not
  change destination asset state; both coders doubted it met the inclusion rule. Result after A9-A12 and adjudication:
  see `s1_verification_2026-09-28.md` section 3 (18 tiered paths).
- A13 (2026-09-28, second pass): four systems not reached in the first pass were added (R22 Circle CCTP v1, R23 Scroll messenger,
  R24 Gnosis AMB, R25 Aave bridge executors) and coded blind by a second model; production configurations (ICS-27 params on 9
  Cosmos chains, Polkadot/Asset Hub XCM runtimes, Aave PayloadsController views) were read under A7. A human-coder kit was prepared
  (`s1_verification/human_coding_kit/`); the human coding itself has NOT been done. Current totals: `s1_verification_2026-09-28.md` section 8.5.
