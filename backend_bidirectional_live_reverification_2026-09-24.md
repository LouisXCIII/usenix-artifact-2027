# Live bidirectional back-end re-verification (2026-09-24, evening)

Direct follow-up to "kiểm tra lại toàn bộ phần back-end cả 2 chiều besu->fabric và
fabric->besu với phần thực nghiệm" (re-check the whole back-end, both directions, against the
experiments). This is a **re-verification pass**, not a new experiment: it does not add a new n
or change any reported statistic in `Manuscriptsp_m2_r3.tex`. Its purpose is to confirm, with
brand-new live transactions run at verification time (not reused from earlier evidence), that
both relay directions still work exactly as the manuscript claims.

## Infra confirmed live before testing
- Besu main chain (chainId 1337, tunnel port 18545): block advancing normally (471,022 →
  471,404 during the check).
- Fabric `authchannel`: chaincode `authcc` **v2.23/seq35** committed on both orgs, ledger height
  3012.
- 2 historical forward-direction decisions (`a1-real-fabricated_hash_garbage_extradata-...`,
  `a5-real-seals0-...`) re-read byte-identical to the original evidence JSON.
- 5 reverse-direction contracts (`EVMDestinationGate` V1/V2/V3, `TACVerifier`, `EVMDestinationGateABAC`)
  all confirmed to still have live, non-empty code at their recorded addresses; 3/3 ABAC decisions
  re-read byte-identical to `e6_abac_evm_result_2026-09-24.json`.

## Forward direction (Besu -> Fabric), fresh
Fetched a **brand-new** real Besu header + registry Merkle-Patricia proof at verification time
(block `0x7316c` / 471,404), submitted unmutated to `SubmitDecisionTrustless` (the real,
non-ablated light-client entry point).

- First attempt failed: `"header timestamp is in the future relative to the transaction"` --
  the already-known ledger-clock-staleness issue (Fabric's on-ledger clock, advanced only by the
  pinned `org1admin` authority, had not been advanced recently). Not a back-end defect.
- Fixed with a real `SetClock(now+1800)` transaction, then re-ran: **`lcVerified: true`,
  `transferClass: SAFE`**, `lcDetail`: "header of block 0x7316c reproduced from raw fields with
  3/4 distinct pinned-validator commit seals; registry slots proven by Merkle-Patricia proofs
  under its stateRoot". Full result: `fresh_forward_check_result.json` (see Files).

## Reverse direction (Fabric -> EVM), fresh
Wrote a brand-new real decision on Fabric (`SubmitDecision`, role=manager, amount=200000,
within the 500000 threshold) -> Fabric returned **SAFE**. Fed those exact real fields into the
live `EVMDestinationGate` (V1, `0xEeb84df0157167bAcc7384C674AbFDf8c8C27230`) via a fresh
`submitDecision()` call on Besu.

- EVM returned **EXACT**, not `SAFE` -- investigated before reporting anything as broken.
- Root cause (verified by reading both `classify()` implementations): Fabric's `classify()`
  (authchaincode.go) has 4 outcomes only (REJECT / REEVALUATE / SAFE / REJECT-widening) -- `SAFE`
  covers every in-scope, within-threshold case uniformly, whether the role was an identity match
  or a cross-domain remap. `EVMDestinationGate.sol`'s reverse-direction gate deliberately
  distinguishes a 5th, finer class: `EXACT` when the role required no remapping (identity, e.g.
  "manager" -> "manager", exactly this test's input) vs `SAFE` when a genuine cross-domain
  role-mapping table entry fired (e.g. "gerente" -> "manager") but the amount still cleared the
  threshold. This is the same distinction already exercised correctly in the existing
  `S0_positive_control_identity` case in `evm_reverse_direction_result_2026-09-24.json` (same
  role/amount inputs, expected `EXACT`). **The mismatch was this check script's own naive oracle
  (`evmClass === "SAFE"`), not a contract or manuscript defect** -- the contract behaved exactly
  as designed and as already documented. Read back on-chain via `getDecision()`: persisted state
  matches the tx event exactly. Full result: `fresh_reverse_check_result.json` (`ok: false` in
  that raw file reflects the check script's overly strict oracle, not a real failure -- see this
  note for the corrected interpretation).

## Conclusion
Both relay directions are real, live, and behave exactly as the manuscript and existing evidence
already claim. No manuscript numbers require updating from this pass. Two apparent anomalies were
each investigated to a concrete, verified root cause (ledger-clock staleness; an intentional
EXACT/SAFE granularity difference between the two chaincodes/contracts) rather than reported at
face value.

## Separate, disclosed operational finding (not a paper/data issue)
Background processes started via the Bash tool this session (SSH tunnel, netns bridges, local
Besu chains, the multi-bank dashboard's `node dashboard/server.js`) died within minutes even with
`nohup`+`setsid`+`disown`, despite the host itself not rebooting (uptime unaffected). This affects
operational convenience (needing to re-establish the tunnel before re-running checks) but not any
already-recorded evidence -- the paper's real Besu/Fabric networks are always-on infrastructure
(systemd on validators, Docker containers with no stop policy for Fabric), unaffected by this
Bash-tool-specific behavior.

## Files
- `fresh_forward_check_result.json` -- the fresh forward-direction (Besu->Fabric) result.
- `fresh_reverse_check_result.json` -- the fresh reverse-direction (Fabric->EVM) result (see
  Conclusion above for the corrected `ok` interpretation).
