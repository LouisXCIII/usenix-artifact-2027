# S1 -- Real-system survey: RESULTS, first pass (2026-09-28)

> SUPERSEDED IN PART. This is the FIRST-PASS coding by a single coder. A second-coder check, deployed-code check and
> adjudication followed the same day (`s1_verification_2026-09-28.md`, amendments A7-A12 in the protocol). After
> them the counts below (17/19 strict, 15/19 lenient) became 16/18 strict and 14/18 lenient, R2/R3/R6/R7/R8/R10/R15/R17
> cells changed, and R21 was excluded. A second pass later the same day (section 8 of the verification file) added four
> systems (R22-R25) and production-configuration checks: CURRENT figures are 20/22 strict and 18/22 lenient paths exposed,
> 22 paths in 17 projects. This file is kept unedited below as the audit trail of the first pass.

Protocol: `s1_realsystem_survey_protocol_2026-09-28.md` (frozen before any system was read; deviations
listed under "Amendments" below, all made AFTER seeing data and therefore disclosed, not hidden).
Sources: raw source files pinned by commit in `s1_sources/MANIFEST.json` (33 files, 19 repositories).
Coder: a single coder (the assistant). No inter-rater reliability has been measured. Every cell cites a
file:line in `s1_sources/` so that any reader can re-check it. A second coder on a subset is required
before this is claimed in a paper (see "What this does NOT show").

## Coding table (D2 source binding, D3 effect scope, D4 parameter policy, D5 destination delay/veto)
Y=Yes, P=Partial, N=No (for L-proto rows N means "not provided by the protocol/base class"), U=Unknown.
E = destination-side policy over the EVIDENCE it accepts (verifier set, quorum, finality) -- descriptive only,
not part of the tier. deleg = by_design_delegation. Tier strict counts a Partial in D3/D4 as No;
tier lenient counts it as Yes (Amendment A1).

| # | System / receiver path | Stratum | Lvl | D2 | D3 | D4 | D5 | E | Tier strict / lenient | deleg | Evidence (s1_sources file:line) |
|---|---|---|---|---|---|---|---|---|---|---|---|
| R1 | Hyperlane `Router` base | a | proto | Y | N | N | N | - | T1 / T1 | - | hyperlane/..client__Router.sol:103 (enrolled router == sender) |
| R2 | Hyperlane Interchain Accounts | a | app | Y | N | N | N | - | T1 / T1 | Y | ..InterchainAccountRouter.sol:163 `ica.multicall`; ..Call.sol:37 delegatecall path; ..OwnableMulticall.sol:25 |
| R3 | Axelar `AxelarExecutable` | a | proto | N | N | N | N | - | T0 / T0 | - | axelar-gmp/..AxelarExecutable.sol:46 (gateway approval only; `_execute` left to app) |
| R4 | Axelar `InterchainGovernance` | c | app | Y | N | N | P | - | T1 / T1 | Y | ..InterchainGovernance.sol:61 (NotGovernance), :109 `_call`, :139 (only target != 0); ..TimeLock.sol:46 |
| R5 | LayerZero v2 `OAppReceiver` | a | proto | Y | N | N | N | - | T1 / T1 | - | lz-v2/..OAppReceiver.sol:106 (OnlyPeer), :109 `_lzReceive` abstract |
| R6 | Chainlink CCIP `CCIPReceiver` | a | proto | N | N | N | N | Y | T0 / T0 | - | ccip/..CCIPReceiver.sol:44 (router-only); :62 `getCCVsAndFinalityConfig` (receiver-declared verifiers/finality) |
| R7 | Chainlink CCIP `TokenPool` | a | app | U | N | Y | U | - | T3 / T3 | - | ccip/..TokenPool.sol:489 `_consumeInboundRateLimit`; :625 chain allowlist |
| R8 | Wormhole core `parseAndVerifyVM` | a | proto | N | N | N | N | - | T0 / T0 | - | wormhole/..Messages.sol:183 ("xDapps rely on the hash ... for replay protection") |
| R9 | Uniswap Wormhole receiver | c | app | Y | N | N | N | - | T1 / T1 | Y | uni-wh/..Receiver.sol:102 (emitter), :147 arbitrary `.call`, :74 2-day validity window |
| R10 | Wormhole NTT `NttManager` | a | app | U | U | Y | Y | - | T3 / T3 | - | ntt/..NttManager.sol:259,292 inbound rate limit; :317 queued-transfer completion (D5 = delay applies only to over-limit transfers) |
| R11 | Aave CrossChainReceiver + PayloadsControllerCore | c | app | Y | P | P | Y | Y | T1 / T3 | - | aave-deliv/..CrossChainReceiver.sol:40 (allowed bridge adapters),:203 (required confirmations); aave-gov3/..PayloadsController.sol:58-60 (origin == MESSAGE_ORIGINATOR, chain == ORIGIN_CHAIN_ID); ..PayloadsControllerCore.sol:99-105 (each action needs a configured executor level),:292 (vote level >= payload's required level),:142 (delay),:166-168 (`cancelPayload` onlyGuardian) |
| R12 | MakerDAO `L2GovernanceRelay` (Arbitrum) | c | app | Y | N | N | N | - | T1 / T1 | Y | maker-arb/..L2GovernanceRelay.sol:34,36 (`target.delegatecall(targetData)`) |
| R13 | Compound `BaseBridgeReceiver` | c | app | Y | N | N | P | - | T1 / T1 | Y | comet/..BaseBridgeReceiver.sol:81 (govTimelock), :97 local delay; no cancel function (only initialize/processMessage/executeProposal/state) |
| R14 | OP Stack `CrossDomainMessenger` | b | proto | P | P | N | P | - | T1 / T2 | - | op/..CrossDomainMessenger.sol:264 (`_isUnsafeTarget`), :235 (`paused()`); ..L2CrossDomainMessenger.sol:60 (denylist = self + MessagePasser) |
| R15 | Arbitrum retryable tickets (L1 side) | b | proto | P | N | N | N | - | T1 / T1 | - | nitro/..Inbox.sol:324 `applyL1ToL2Alias(sender)`; L2 execution engine (nitro Go node) NOT examined |
| R16 | Polygon `FxBaseChildTunnel` | b | proto | P | N | N | N | - | T1 / T1 | - | fx/..FxBaseChildTunnel.sol:24-27 (opt-in `validateSender` modifier), :38 fxChild-only |
| R17 | IBC ICS-27 host | d | proto | Y | Y* | N | P | - | T2 / T2 (default posture T1) | - | ibc-go/..host/keeper/relay.go:107 (`AllowMessages`), :123 (signer == ICA); ..host/types/params.go:28 default `"*"`; ..host/ibc_module.go:118 (`HostEnabled` global off-switch in OnRecvPacket = D5 Partial) |
| R19 | Polkadot XCM executor | d | proto | Y | Y | P | P | - | T2 / T3 | - | polkadot/..config.rs:122 `SafeCallFilter`, :65 `Barrier`, :49,:52 origin/reserve; ..lib.rs:1096,269 (applied) |
| R21 | Hyperledger Cacti SATP (stage 1) | e | proto | P | N | N | N | - | T1 / T1 | - | cacti/..stage1-server-service.ts:460 (DLT-supported check), :437 `//TODO Implement` (CONDITIONAL_REJECTED) |
| -- | Hyperledger Weaver | e | -- | -- | -- | -- | -- | Y | not tiered (A4) | - | weaver/..verification_policy.proto:11 (destination evidence policy); ..access_control.proto:20 (source-side read rule) |

`Y*` = the mechanism exists and is destination-configurable, but the shipped default is allow-all (Amendment A2).

## Distribution (19 tiered receiver paths in 15 systems; Weaver not tiered)
| Tier | strict | lenient |
|---|---|---|
| T0 authenticity-only, unrestricted | 3 | 3 |
| T1 sender-bound, unrestricted effect | 12 | 10 |
| T2 effect-scoped, no parameter policy | 2 | 2 |
| T3 destination evaluates own parameter policy | 2 | 4 |
| Exposed to the Eq.(1) gap (T0-T2) | 17 / 19 | 15 / 19 |
| ... of which by_design_delegation=Y (R2,R4,R9,R12,R13) | 5 | 5 |
| Exposed and NOT by-design delegation | 12 | 10 |

Counting rule: an "exposed" path means the receiver code itself does not evaluate the destination's
policy over the concrete effect. It does NOT mean the path is vulnerable (see Amendment A3).

## What the data support (and what they do not)
1. Every path that changes destination state authenticates the message. All but the bare transport
   bases R3, R6, R8 also bind it to a sender, but only R1, R2, R4, R5, R9, R11, R12, R13, R17, R19 do so
   in the receiver itself (D2=Y); R14, R15, R16, R21 leave the sender check to the application (D2=P).
   None of the 19 evaluates a relation between the
   source's authorization and the destination's own policy over the concrete effect. The nearest is
   Aave (R11): the source vote's strength must be >= an access level the payload declares, mapped to a
   destination executor and delay. That is a partial, single-dimension analogue of ADP's threshold check.
2. The two T3 paths that hold under the strict reading (R7 CCIP token pool, R10 NTT) are token-flow
   rate limits: one numeric bound per time window, hard-coded in the pool. Neither is a general policy.
3. XCM (R19) and ICS-27 (R17) are the only frameworks that put a destination policy slot in the
   PROTOCOL rather than leaving it to integrators (`SafeCallFilter`/`Barrier`; `AllowMessages`). Both
   are type-level. ICS-27's default is allow-all; XCM's slot is runtime-supplied and its concrete
   configuration was not examined.
4. Transport layers (Axelar, CCIP, Wormhole core, LayerZero, Hyperlane) provide no destination policy by
   construction. That is expected for a transport; the point is that the policy is left to each
   integrator and there is no shared way to state or check it.
5. Five of the 13 T0/T1 paths (lenient reading: 3 T0 + 10 T1) deliberately give the source domain full
   authority over the destination: R2 (a remote-owned account) and four governance relays (R4, R9, R12,
   R13). Their only mitigation is a delay (R4, R13 partial; R2, R9, R12 none in the receiver). These are
   trust decisions, not defects, and the paper must say so.
6. Cacti SATP (R21) has a slot for destination-conditional acceptance (`CONDITIONAL_REJECTED`) whose
   body is `//TODO Implement` in the pinned snapshot. This is evidence that the need is recognised,
   not evidence that it is met.

## What this does NOT show
- NO exploitable widening was found. No live system was probed and none was reproduced on a fork.
  The survey shows where destination policy is absent in the RECEIVER; whether an integrator has added
  it in a target contract was not examined (Amendment A3).
- No prevalence estimate: the sample is purposive (15 systems chosen for stratum diversity).
- Static reading of repository HEAD, NOT deployed bytecode. Maker's snapshot is the 2021 commit; the
  others are HEAD on 2026-09-28 and may differ from what is live. On-chain bytecode was not compared.
- Single coder, no inter-rater agreement. Some cells are U (not examined) rather than guessed.
- Arbitrum L2 execution (Go node), Optimism-side Maker variant, XCM concrete barriers, and Chainlink
  RMN were not examined.
- NOT REACHED: Gnosis AMB, Scroll/zkSync/Linea messengers, Polygon AggLayer, Across, Stargate/OFT,
  Circle CCTP, Axelar ITS, IBC ICS-20 core and production ICS-27 host configurations, Lido and Arbitrum
  DAO governance executors.

## Secondary corpus: incident classification (5 incidents; convenience sample, NOT a prevalence estimate)
Rule from the protocol: authenticity failure (unauthorised/forged message accepted) vs admissibility gap
(authentic message, effect the destination would not have permitted). Only the latter supports the paper.
Evidence grade: the four authenticity rows rest on web search summaries of public post-mortems; the primary
post-mortems were located but NOT read in full. The Poly Network row was verified earlier in this project
against the Kudelski Security write-up (bib key `kudelski2021polynetwork`) and SlowMist (`slowmist2021polynetwork`).

| Incident (date, approx. loss) | Class | One-line reason | Pointer |
|---|---|---|---|
| Poly Network (2021-08, ~$610M) | ADMISSIBILITY GAP | Authenticated relay call made `EthCrossChainManager` invoke a privileged function on `EthCrossChainData` chosen by an attacker-controlled method-name string; no destination restriction on which function a message may target | Kudelski; SlowMist (in bib) |
| Wormhole (2022-02, ~$326M) | authenticity failure | `verify_signatures` read a spoofed instructions sysvar, so a mint was approved without guardian signatures | CertiK incident analysis; Halborn |
| Ronin (2022-03, ~$624M) | authenticity failure | 5 of 9 validator keys compromised; withdrawals were validly signed but unauthorised | Ronin post-mortem (roninchain.com blog); Halborn |
| Nomad (2022-08, ~$190M) | authenticity failure | Upgrade initialised the trusted root to 0x00, matching the untrusted value, so any message counted as proven | Coinbase incident analysis; Halborn; Google Cloud (Mandiant) |
| Harmony Horizon (2022-06, ~$100M) | authenticity failure (key compromise) | 2 of 5 multisig keys compromised | Harmony post-mortem (Medium); Halborn |

Result: 1 of 5 is an admissibility gap; 4 of 5 are authenticity failures. Not verified and therefore not
classified: Multichain (2023), Orbit Bridge (2024), Qubit (2022).
Consequence that must be stated in the paper: the losses in this sample are dominated by authenticity failures,
which ADP does not address (it assumes Authentic_s; Section 3). The gap ADP targets is real (Poly Network) but the
sample does not show it is the most frequent cause of loss, and it must not be presented as such.

## Amendments (all made after seeing data; dated 2026-09-28)
- A1 (after R14, OP Stack): D3 "Partial" = a denylist or narrow protocol restriction that does not confine the
  effect to a positive set; "Yes" = a positive allowlist or class mapping. Tiers are reported under both
  readings (strict / lenient) because this decision affects R11, R14, R19.
- A2 (after R17, ICS-27): capability (mechanism exists, destination-configurable) and default posture are
  recorded separately; the tier uses the capability, the default is noted.
- A3 (after R2, R9, R12): a T0/T1 receiver does not imply a vulnerability. Destination policy may live in the
  target contracts the receiver calls. The survey measures the RECEIVER's own enforcement only.
- A4 (after Weaver): Weaver is data transfer with a destination evidence policy and a source read policy; it
  executes no destination effect, so it is excluded from the tier counts (scope mismatch with the inclusion rule).
- A5 (after R1, R3, R5): for L-proto rows, N means "not provided by the protocol/base class", which is what a
  transport can be asked for; it is not a defect of the transport.
- A6: column E (evidence-side policy at the destination) was added; it is descriptive and never enters a tier.

## Consequence for the paper (proposed, not yet written)
The defensible claim is structural, not prevalence: current receivers leave the destination's admissibility
check to integrators or delegate it entirely, and the only shipped protocol-level destination policies are
type-level allowlists or single-parameter rate limits; none relates source authorization to destination
policy. This supports "the gap in Eq.(1) is a design property of deployed interoperability stacks" and motivates
a relational check. It does not support "such systems are commonly exploitable".
