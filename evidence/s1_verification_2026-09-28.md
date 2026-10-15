# S1 -- Verification phase (2026-09-28): second coder, deployed code, post-mortems, target layer

Follows `s1_realsystem_survey_results_2026-09-28.md`. Raw material is in `s1_verification/`:
`coder1_codes_before_adjudication.json`, `coder2_blind_codes.json`, `adjudicated_codes.json`,
`deployed_sources/` (verified sources fetched from public explorers), `needles_*.json` and `needles_result_*.json`
(the statements tested against deployed code), `fetch_deployed.py`, `check_needles.py`. Amendments A7-A12 are in the protocol file.

## 1. Second coder (independent, blind)
Coder 2 was a different model (Opus) that received only the protocol, rubric clarifications and the pinned
sources, in a directory that did not contain coder 1's results. It coded 20 rows. It reported opening nothing
outside its directory. This is agreement between two AI coders, not human agreement.

Agreement BEFORE adjudication (19 rows coded by both; D2-D5 are the tier inputs):

| Dimension | Raw agreement | Cohen's kappa |
|---|---|---|
| D2 source binding | 11/19 = 0.58 | 0.34 |
| D3 effect scope | 16/19 = 0.84 | 0.68 |
| D4 parameter policy | 18/19 = 0.95 | 0.87 |
| D5 destination delay/veto | 14/19 = 0.74 | 0.55 |
| D2-D5 pooled | 59/76 = 0.78 | 0.63 |
| Tier (strict) | 14/19 = 0.74 | 0.53 |
| Tier (lenient) | 14/19 = 0.74 | 0.60 |

Reading: D3/D4 (the dimensions that decide T2/T3) agree well; D2 does not. Kappa on 19 items with skewed
categories is unstable; treat these as indicative, not precise.

## 2. Why they disagreed, and what was decided (each cell re-checked against the source)
Most disagreements were MY inconsistencies or under-examined cells, not coder 2 errors.

| Cell | Coder 1 -> adjudicated | Reason (verified) |
|---|---|---|
| R3 D2, R8 D2 | N -> P | Same situation as R14/R15/R16 (base exposes the remote sender, app must check), which I had coded P. Applied one rule (A9). |
| R6 D2 | N -> P | `Any2EVMMessage` has a `sender` field (ccip Client.sol:14) but CCIPReceiver never checks it. Coder 2 said U because Client.sol was not in its files; I added the file. |
| R2 D2, R17 D2 | Y -> N | Per-sender-account designs accept any remote sender; authority is isolated in a sender-derived account. A literal reading of D2 gives N (A9). The isolation is kept as a note. |
| R7 D2 | U -> Y | `isRemotePool` check, TokenPool.sol:483-484. |
| R7 D5 | U -> Y | RMN `isCursed` blocks release, TokenPool.sol:479. |
| R7, R10 D3 | N/U -> Y | Fixed-function receivers (release/mint only) confine the effect by construction (A10). |
| R10 D2 | U -> Y | `_verifyPeer`, NttManager.sol:193,209. Pause via `whenNotPaused` (:168,181,192,208). |
| R15 D4, D5 | N -> U | L2 execution code was not examined; I had written N where U was correct. |
| R17 D5 | P -> Y | `HostEnabled` off-switch in OnRecvPacket, ibc_module.go:118 = a destination pause (A11). |
| R19 D5 | P -> N | No pause/delay/guardian found in the pinned executor files; `Barrier` is a filter, already counted in D3/D4. |
| R14 D5 | P (kept) | Coder 2 said N. `paused()` is a no-op on L2 and real on L1, so the answer depends on message direction; recorded as a rubric ambiguity, not tier-relevant. |
| R21 | excluded | Stage 1 is negotiation only; both coders doubted it "changes destination state" (A12). Not tiered. |

## 3. Adjudicated result (18 tiered paths; supersedes 17/19 and 15/19 in the first-pass file)
| Tier | strict | lenient |
|---|---|---|
| T0 | 1 | 1 |
| T1 | 13 | 11 |
| T2 | 2 | 2 |
| T3 | 2 | 4 |
| Exposed to the Eq.(1) gap (T0-T2) | 16 / 18 | 14 / 18 |
| ... of which by-design delegation (R2,R4,R9,R12,R13) | 5 | 5 |
| Exposed, not by-design | 11 | 9 |
The qualitative conclusion is unchanged. The adjudicated codes were produced by the same person who coded first,
after seeing coder 2's answers; they are a consensus product, not a third independent coding. R15's tier rests on
D4=U (treated as T1).

## 4. Deployed code versus repository HEAD (A7; read-only public data)
Method: for each deployed contract, fetch the verified source from a public explorer/Sourcify, following EIP-1967
proxies to the implementation, then test whether the statements that support my coded cells appear (comments and
whitespace stripped). Addresses come from each project's own registry (roots.json, address book, chainlog,
hyperlane-registry, axelar chains config, Uniswap README); Maker's L2 relay was read from the L1 relay's public
`l2GovernanceRelay()` view. 13 deployed contracts on 5 chains (OP Mainnet, Arbitrum, Polygon, Ethereum, BNB),
covering 7 of the 18 tiered paths, 50 statements in total: 45 match literally.

| Path | Deployed contract(s) | Statements | Result |
|---|---|---|---|
| R14 OP messenger | L2CrossDomainMessenger impl `0x250AF3f4...` behind `0x4200...0007` (OP Mainnet) | 4 | 4/4 |
| R13 Compound | Bridge receivers on Arbitrum `0x42480C37...`, Polygon `0x18281dfC...`, Optimism `0xC3a73A70...` | 12 | 12/12; receivers expose no cancel |
| R12 Maker | L2GovernanceRelay `0x10E6593C...` (Arbitrum) | 2 | 2/2 (`delegatecall` of a source-chosen target) |
| R9 Uniswap | Wormhole receiver `0x341c1511...` (BNB), Sourcify `exact_match` | 3 | 3/3 |
| R11 Aave | PayloadsController on Ethereum `0xdAbad81a...`, Polygon `0x401B5D02...`, Arbitrum `0x89644CA1...`; CrossChainController `0xEd42a7D8...` | 17 | 14/17 literal; the 3 misses are one textual difference (below) |
| R4 Axelar | AxelarServiceGovernance `0x7Acbae6C...` (Ethereum; same address on at least 7 chains) | 4 | 4/4 |
| R2 Hyperlane | InterchainAccountRouter Ethereum `0xC00b94c1...`, Arbitrum `0xF90A3d40...` | 8 | 6/8; delegatecall path absent (below) |

Differences found (the useful part of this check):
- Aave `cancelPayload`: deployed is `external onlyGuardian`; HEAD is `external virtual onlyGuardian`. Same guardian-only
  cancel; the literal test missed only the `virtual` keyword on all three chains. The coded cell holds.
- Hyperlane ICA: the deployed `CallLib.call` is a plain `functionCallWithValue(to, data, value)`. The DELEGATECALL
  sentinel I cited (HEAD Call.sol:37) is NOT in the deployed code; it was added later. D3=N still holds (arbitrary
  target/data), but the delegatecall evidence in the first-pass file describes HEAD only.
- Axelar: the deployed contract is AxelarServiceGovernance, which adds an operator that can execute a proposal early
  once SOURCE governance approves it (`ApproveOperatorProposal`). That weakens the delay; it is not a destination veto
  (cancel is also a source command). D5=P stands.
Not deployed-checked (11 of 18): R1, R3, R5, R6, R7, R8, R10, R15, R16 (base classes/libraries with many deployments,
or L2 engines), R17, R19 (chain-specific runtime configuration). These remain repository-level codings.

## 5. Primary post-mortems (incident table)
| Incident | Source read | Confirmed | Class |
|---|---|---|---|
| Nomad (2022-08) | Google Cloud/Mandiant analysis | trusted root initialised to 0x00 equals the untrusted value, so messages counted as proven; >$190M | authenticity failure |
| Wormhole (2022-02) | CertiK analysis | deprecated function did not validate the sysvar account, verification bypassed; 120,000 wETH, ~$320M | authenticity failure |
| Harmony Horizon (2022-06) | Halborn analysis | 2 of 5 multisig keys compromised, valid signatures used; ~$100M | authenticity failure |
| Ronin (2022-03) | Halborn analysis | 5 of 9 validator signatures from compromised keys/delegation, no bypass; ~$624M | authenticity failure |
| Poly Network (2021-08) | Kudelski (earlier) | authenticated call, attacker-chosen privileged function | admissibility gap |
These are independent analyst write-ups (Mandiant, CertiK, Halborn), not all victim-authored. The victims' own
post-mortems for Ronin and Harmony could not be retrieved (404 / fetch failure), so those two rest on Halborn.
Classification did not change. Multichain, Orbit, Qubit remain unclassified.

## 6. Target-contract layer (the A3 question), worked case: Compound on Arbitrum
Question: when a receiver forwards an authentic governance message, does the TARGET impose policy the receiver does not?
Chain of authority: source Timelock -> BridgeReceiver `0x42480C37...` -> local Timelock -> Configurator `0xb21b06D7...`
(impl `0x8495AF03...`) -> Comet `0x9c4ec768...` (impl `0x5D3DfD6B...`).
- Configurator setters check only `msg.sender == governor` (e.g. `setSupplyKink`, `setBaseBorrowMin`,
  `updateAssetSupplyCap`, Configurator.sol:133,229,297). They do not validate the value.
- Comet enforces well-formedness invariants when it is (re)built: `borrowCollateralFactor >= liquidateCollateralFactor`
  reverts (`BorrowCFTooLarge`, Comet.sol:276,287), `storeFrontPriceFactor <= FACTOR_SCALE`, decimals and asset-count bounds
  (:137-141).
So the target does hold destination-side parameter checks, but they are static structural invariants: they do not depend
on WHO authorised or HOW strongly, and a governance message may set any value that satisfies them.
MakerDAO is the opposite pole: the L2GovernanceRelay `delegatecall`s a source-chosen spell, and Maker's own README
names the residual risk ("granting minting rights by executing malicious spell with `L2GovernanceRelay`") and relies on
a 2-day source-side delay (maker-arb README lines 56, 68-72).
Limit: one worked case and one documented case, not a survey of target contracts. It shows that "policy lives in the
target" is true only in the weak, static sense for Compound, and not at all for Maker's relay.

## 7. What is now established, and what is still not
Established: (a) the coded evidence holds in 7 deployed systems (with one HEAD-only feature identified); (b) two AI coders
agree well on D3/D4 and poorly on D2, and the D2 rule was made explicit; (c) 4 incidents are authenticity failures per
independent analyses; (d) at the target layer, Compound has static invariants but no relation to source authority.
Still NOT established: no human second coder; 11 of 18 paths not deployed-checked; only two governance targets examined;
no exploitable widening exists or was sought; the sample remains purposive, so no prevalence claim.
(Sections 8-9 below are the same-day second pass on these gaps and change some of these numbers.)

## 8. Second pass (2026-09-28): the items that had been left undone
### 8.1 Production configurations (A7, read-only public data)
- ICS-27 (R17), 9 Cosmos chains via the public REST proxy rest.cosmos.directory, queried 2026-09-28T02:29Z
  (`s1_verification/prod_configs/ics27_*.json`): allow-all `["*"]` on cosmoshub, neutron, juno, noble (4/9); type
  allowlists on osmosis (40 types), injective (34), celestia (12), stride (12), dydx (10) (5/9); akash returned
  501 (no host endpoint). Every allowlist is a list of message-type URLs; none carries a parameter condition.
- XCM (R19), polkadot-fellows/runtimes @9778ddb91c (`prod_configs/xcm_*__xcm_config.rs`): Polkadot relay has
  `SafeCallFilter = Everything` (xcm_config.rs:257); its Barrier admits `AllowTopLevelPaidExecutionFrom<Everything>` plus
  an explicit unpaid allowlist of system parachains, Fellows and the Asset Hub plurality (:193-215). Asset Hub Polkadot has
  `SafeCallFilter = AllExceptReapStash` (xcm_config.rs:552, everything except one call) and a Barrier with
  `DenyReserveTransferToRelayChain`. The protocol slots exist but are filled with source/payment filters and near-total
  call allowance; no per-call parameter policy.
- Aave (R11) destination class policy, read with view `eth_call`s (`prod_configs/aave_payloadscontroller_views.json`):
  Ethereum PayloadsController: Level 1 executor `0x5300A1a1...` delay 1 day, Level 2 executor `0x17Dd33Ed...` delay 7 days,
  guardian `0xCe52ab41...` (a contract). Polygon and Arbitrum: only Level 1 (1 day); Level 2 executor is the zero address, so
  a payload needing Level 2 cannot be created there (PayloadsControllerCore.sol:99-105). This is the one deployed
  destination-side policy found that ties the effect class to an executor and a delay.

### 8.2 Arbitrum L2 execution (R15)
nitro Go @a618155919: the `ArbitrumSubmitRetryableTx` hook consults `FilteredTransactions().IsFilteredFree(ticketId)`
(tx_processor.go:246); an `ArbFilteredTransactionsManager` precompile lets authorised filterers, added through ArbOwner, mark
tx hashes as filtered. So there is a per-message destination veto by hash and no parameter policy in the retryable hooks.
Caveats: this is HEAD code (2025-2026 headers); whether it is active on Arbitrum One was not verified. R15 becomes
D4 = N (was U), D5 = Y (was U). The deployed L1 Inbox (`0x4Dbd4fc5...` -> impl `0x7C058ad1...`) matches the coded L1-side statements (3/3).

### 8.3 More deployed-source checks (12 of the original 18 paths now have deployed evidence, 2 more have production config)
- Wormhole core, Ethereum `0x98f3c9e6...` (address from Wormhole's own docs): 2/2 statements. Replay tracking exists only for
  governance actions (State.sol:38, Getters.sol:21); application-VAA replay is left to dapps, as coded.
- Hyperlane USDC collateral warp route on Ethereum `0xE0C2e4F8...` (from hyperlane-registry): base Router statements 2/2; the
  implementation is `HypERC20CollateralPausable` (a pausable variant) and contains no rate limiter.
- Axelar InterchainTokenService `0xB5FB4BE0...` (from Axelar's config): `onlyRemoteService` -> `isTrustedAddress(sourceChain,
  sourceAddress)` (InterchainTokenService.sol:173-174): the application implements the source check the base class leaves to it.
- Compound PolygonBridgeReceiver: `msg.sender != fxChild` is checked, then `rootMessageSender` goes to the application check (R16 pattern).
- NOT located: R5 (LayerZero OApp), R6 (CCIPReceiver), R7 (CCIP TokenPool), R10 (NTT manager). No machine-readable registry of
  deployed instances was reachable with the tools available (Chainlink and Wormhole docs pages did not expose addresses).

### 8.4 Four new systems, coded blind by a second model (batch 2)
Sources pinned (`s1_sources/{cctp,scroll,gnosis-amb,bridge-exec}`): R22 Circle CCTP v1, R23 Scroll L2ScrollMessenger, R24 Gnosis/POA AMB
(foreign side), R25 Aave governance bridge executors. I recorded my own codes BEFORE opening coder 2's file
(`coder1_batch2_codes_before_seeing_coder2b.json`; coder 2 = `coder2b_blind_codes.json`).
Agreement: 17/20 cells; all 12 tier-input cells (D2-D4) agree. Differences: R24 D5 (I wrote N from absence in four files; coder 2 wrote U because
the parent contracts that would hold a pause were not provided; U is right), and two descriptive E cells. Caveat: this batch was
coded after the D2/D3/D5 definitions were made explicit (A9-A11), so its higher agreement is expected and is not independent evidence about the
original rubric; kappa on 4 rows is not meaningful and is not reported.
Adjudicated: R22 D2 Y, D3 Y (fixed-function mint), D4 N (CCTP's burn limit is on the source side), D5 Y (whenNotPaused) -> T2;
R23 P/P/N/Y -> T1 strict, T2 lenient; R24 P/N/N/U -> T1; R25 Y/N/N/Y (delay, grace period, `cancel onlyGuardian`, arbitrary targets and delegatecall) -> T1, by-design delegation.

### 8.5 Updated totals (22 tiered paths in 17 projects; supersedes section 3)
| Tier | strict | lenient |
|---|---|---|
| T0 | 1 | 1 |
| T1 | 16 | 13 |
| T2 | 3 | 4 |
| T3 | 2 | 4 |
| Exposed (T0-T2) | 20 / 22 | 18 / 22 |
| By-design delegation (R2, R4, R9, R12, R13, R25) | 6 | 6 |
| Exposed, not by-design | 14 | 12 |
Evidence depth: 12 paths with deployed-source evidence, 2 with production configuration, 8 repository-only (R5, R6, R7, R10, R22-R25).
Codes: `s1_verification/adjudicated_codes.json`.

### 8.6 Incidents (8 in total)
- Ronin: the original 2022 Sky Mavis announcement, retrieved from the Wayback Machine (`prod_configs/ronin_postmortem_wayback_2022.html`),
  states that the attacker "used hacked private keys in order to forge fake withdrawals": authenticity failure, now confirmed from the victim's own text.
- Added: Multichain (2023-07, ~$126M; MPC-key/operational-control compromise), Orbit Bridge (2023-12-31, ~$81.5M; 7 of 10 multisig keys),
  Qubit (2022-01, ~$80M; a legacy `deposit` function emitted a deposit event without a real deposit, so the destination accepted a false source fact).
  All three were classified from web-search summaries of Neptune Mutual, Chainalysis, Socket, SlowMist and Halborn; the pages were not re-read in full.
  Qubit is a source-fact validity failure, not an admissibility gap: the destination would have permitted the mint for a real deposit.
- Harmony's own post-mortem (Medium) could not be reached from this host; Harmony still rests on Halborn.
Corpus: 1 admissibility gap (Poly Network) versus 7 authenticity / fact-validity failures.

### 8.7 Human second coder
Attempted the same day by one author; result recorded in `s1_human_coding_attempt_2026-09-28.md`: 21 rows, agreement with both AI coders at chance
level (pooled kappa about 0), with a pattern suggesting a different reading of the letters. Closed without a re-run. Consequence: the coding scheme is not yet
shown to be reliable across coders, and the AI-AI agreement of section 1 should not be read as validity. The kit (`s1_verification/human_coding_kit/`) remains
for a colleague-coded round after the rubric is revised.

## 9. What is still not done, and why
- Human agreement: needs a person (kit ready).
- R5, R6, R7, R10 deployed-instance check: no reachable registry of deployed instances (section 8.3).
- Exploitable widening: none was found and none was sought beyond reading; it cannot be produced by more analysis.
- Prevalence: the sample is purposive by design; more systems would not turn it into a random sample.
- Target layer: two governance cases (Compound, Maker) plus Aave's class policy; not a survey of target contracts.
- Not examined at all: Across, Stargate/OFT, Lido and Arbitrum DAO executors, Optimism-side Maker relay, CCIP RMN internals, Cacti beyond stage 1.
- Victims' own post-mortems for Harmony, and for the three new incidents.
