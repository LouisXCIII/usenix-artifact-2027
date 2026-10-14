# QD3: independent second implementation of obligation-coverage, differential-tested (2026-10-13)

## Method

1. A fresh `general-purpose` agent, with NO access to any chaincode source and NO web search,
   was given only prose quoted verbatim from this paper's Discussion/Limitations sections
   (the S11b and Second-key-discharge paragraphs, plus the sentence naming the five obligation
   categories) and asked to write `independent_oblcovered.py` from that text alone.
2. The file was frozen (SHA-256 `0b34d92652053dc7497f18a97be3e5184a1269d7bfa62f316b094364bc3fe3a6`,
   `2026-10-13T04:22:58Z`) **before** running it against any case data — see `freeze_record.md`.
3. A 700-case mechanical sweep (`cases_mechanical_sweep_qd3.py`) covers the full Cartesian
   product of `obligationProfile`(5, incl. `"empty"`) × `mappedMechanism`(5, incl. `""`) ×
   `enforcementWitnessVerified`(2) × discharge-ref state(7) × `alreadyCitedByAnotherDecision`(2).
   **No combination is excluded** from the sweep or from the reported match rate, including the
   `required==0`/`"empty"` boundary and the S7 (witness-lying) axis.
4. The REAL, unmodified `SubmitDecisionTrustlessV7ObligationVerifiedDischarge` (no refactor) was
   called directly, in-process, via an in-memory ledger stub (`qd3_differential_test.go`),
   reusing one real cached ZK proof + Besu light-client header fixture (the obligation fields are
   independent of the evidence chain, so one fixture is valid for every case).
5. Layer 1: the same comparison was additionally run on the 7 real on-chain decisions from
   `qd_block_verification_2026-10-13/` (6 scenarios + the post-fix reuse pair), with inputs taken
   from this session's own logged run parameters (not retyped by hand) and checked three ways at
   once: on-chain ground truth (via `qdblockverify recover`) = offline-harness output = frozen
   independent implementation's output. This also serves as the equivalence check requested for
   not refactoring: it shows the offline harness reproduces real on-chain behavior exactly, not
   just an isolated self-consistent copy of it.

## Result

- **Layer 1 (7 real on-chain decisions): 7/7 three-way agreement** — on-chain, offline harness,
  and the independent implementation all agree on every case.
- **Layer 2 (700-case mechanical sweep): 700/700 match, 0 mismatches, 0 exclusions.**
- Combined: **707/707**.
- The independent implementation's own inferred mechanism-strength ordering
  (`empty<notification<audit<dual_approval<=combined`, chosen without seeing the code, explicitly
  disclosed as a judgment call in `freeze_record.md`) **happened to coincide exactly** with the
  real deployed ordinal values. This is itself informative but not proof of independent
  correctness of that ordering — see framing below.

## How to read this result (framing, verbatim as agreed)

> A second implementation written only from the paper's prose description of the obligation and
> discharge rules agrees with the deployed logic on all 707 cases; this establishes consistency
> between the stated semantics and the implementation, not independent correctness of those
> rules.

The Discussion/Limitations prose this spec was written from was itself composed after the code
existed, to describe it — so the independence this test achieves is modest: a faithfulness check
of the paper's own description against its implementation, not a check against an
implementation-blind formal specification. The second implementation is LLM-authored (same model
family as the rest of this project's tooling), disclosed explicitly — not two independent human
experts, and the two sides' errors could in principle be correlated. A stronger alternative —
formalizing the mechanism-strength ordering and discharge-matching conditions as actual
definitions in the paper's Semantics appendix, and deriving a spec from that instead — was
identified as valuable (it would also give a formal definition to "enforceable," which
Theorem~2 currently relies on without one) but is noted as future work, not attempted here.

## Files

`independent_oblcovered.py`, `freeze_record.md`, `cases_mechanical_sweep_qd3.py`,
`sweep_cases.json`, `layer1_cases.json`, `independent_output.json`/`layer1_independent_output.json`,
`real_output.json`/`layer1_real_output.json`, `analyze_qd3.py`. The Go harness
(`qd3_differential_test.go`) lives in a disposable `/tmp` scratch copy of the chaincode package
(not committed to the live `~/fabric-network/authcc` checkout, since it changes nothing there) —
its full source is reproduced in this directory as `qd3_differential_test.go.txt` for the record.
