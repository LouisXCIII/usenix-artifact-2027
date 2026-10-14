# Real on-chain ablation of Ab7 as formally defined (2026-10-11)

Closes the gap `evidence/e8_a7_governance_2026-09-24.md` and
`evidence/e8_a9_a7_expansion_2026-09-24/README.md` left open: "An on-chain ablation of Ab7 as
formally defined -- accepting a `MapRec` whose own governance signature is absent or invalid --
remains open work." Those two files built a real but *different* mechanism (Fabric's
transaction endorsement policy, `gov_ok_general`), disclosed honestly as a related analog, not
Ab7 itself. This file is the real thing.

## What "Ab7 as formally defined" means and why it had no twin before

Section~sec:model's formal `MapRec = <id, scope, M_I, M_T, R_Sigma, R_P, M_O, ...>` is a
governance-approved record establishing the destination's role-mapping policy. In the real
deployed system, the mapping (`RoleMapping`/`RoleThreshold`, `authchaincode.go`) is compiled
directly into the chaincode source -- there was no separately-deployed, independently signed
`MapRec` object for an ablation to target.

## What was built

- `chaincode/maprec_governance.go` (new, additive, does not modify any existing entry point):
  a `MapRec` is now a real caller-supplied structure (`id`, `scope`, role->cap JSON table),
  authenticated by a real secp256k1 compact signature, recovered with
  `secpecdsa.RecoverCompact` -- the exact pattern already used three times in this codebase
  (`lightclient.go`'s `verifyIbft2Header`, `ablation_a5_quorum.go`'s
  `verifyIbft2HeaderCustomQuorum`, `q_d_domain_attestation.go`'s `verifyAwsDomainAttestation`)
  -- against a single pinned "mapping-governance authority" address,
  `0xf94d8b83004c9adaeefd8696a70cf864ca79053e`.
  - `SubmitDecisionMapRecVerified` (REAL): rejects fail-closed if the MapRec's signature does
    not recover to the pinned authority; only then uses the MapRec's claimed cap for the
    decision's role.
  - `SubmitDecisionMapRecUnverified` (ABLATED, deliberate twin): identical, except it never
    calls the signature check at all -- the signature is not even a parameter. This is Ab7 as
    formally defined.
- `tools/maprec_sign/main.go` (new, standalone, excluded from the deployed chaincode package):
  generates the signing key and produces compact signatures over the exact digest
  `maprec_governance.go`'s `mapRecDigest` computes
  (`keccak256(id || scope || role_1 || cap_1(32B BE) || ...)`, roles sorted for determinism).
  The private key never leaves this tool's local key file; only the derived address is embedded
  in the chaincode.
- Deployed to the real, live `authcc` chaincode on `authchannel`: v2.39, sequence 51, both
  Org1MSP and Org2MSP approved. Pre-existing data (e.g. `decision-001`) confirmed unchanged
  after the upgrade.

## Real design: 2 role cycles x 4 checks = 8 real on-chain transactions

Same n=8, two-cycle structure as the existing Ab7 endorsement-policy analog, but diversity is
by **role** (clerk/director) rather than by organization, since this is a role-mapping
mechanism, not an org-endorsement one.

| # | Decision ID | Entry point | MapRec | Amount vs. real cap | Expected | Actual |
|---|---|---|---|---|---|---|
| 1 | `maprec-c1-real-safe` | `*Verified` | genuine (clerk=10000) | 5,000 ≤ 10,000 | SAFE | **SAFE** |
| 2 | `maprec-c1-real-reject` | `*Verified` | genuine | 15,000 > 10,000 | REJECT | **REJECT** |
| 3 | `maprec-c1-forged-rejected` | `*Verified` | forged (clerk=2,000,000, wrong signer) | 15,000 | REJECT (sig caught) | **REJECT** |
| 4 | `maprec-c1-forged-unsafe-accept` | `*Unverified` (ablated) | same forged MapRec | 15,000 | **SAFE (unsafe accept)** | **SAFE** |
| 5 | `maprec-c2-real-safe` | `*Verified` | genuine (director=2,000,000) | 1,500,000 ≤ 2,000,000 | SAFE | **SAFE** |
| 6 | `maprec-c2-real-reject` | `*Verified` | genuine | 2,500,000 > 2,000,000 | REJECT | **REJECT** |
| 7 | `maprec-c2-forged-rejected` | `*Verified` | forged (director=50,000,000, wrong signer) | 2,500,000 | REJECT (sig caught) | **REJECT** |
| 8 | `maprec-c2-forged-unsafe-accept` | `*Unverified` (ablated) | same forged MapRec | 2,500,000 | **SAFE (unsafe accept)** | **SAFE** |

**8/8 real on-chain transactions matched expectation**, including the two genuine unsafe-accept
demonstrations (#4, #8) -- the first real, on-chain, per-transaction-comparable instance of Ab7
as formally defined, not an analog. `ReadDecision` on each (raw:
`maprec_results_2026-10-11.json`) confirms the mechanism, not just the verdict: checks 1/2/5/6
show `mapRecVerified=true`, `mapRecSigner` matching the pinned authority; checks 3/7 show
`mapRecVerified` omitted (false), `mapRecSigner` recovering to the forger's address, and the
cap table explicitly marked "NOT trusted" (`destThreshold=0`, no fallback value used); checks
4/8 show the ablated path trusting the forged cap directly (`destThreshold=2000000` and
`50000000` respectively -- the attacker's claimed values, not the real 10,000/2,000,000).

## No confidence bound reported

Same reasoning as the corrected Ab7 endorsement-policy paragraph, above: these 8 checks are
heterogeneous, individually deterministic verifications of a signature-verification mechanism
(real-accepts-within-cap, real-rejects-over-cap, real-rejects-forged-signature,
ablated-accepts-forged-unsafely), not repeated draws of one stochastic quantity. Plain counts
are reported, no Clopper-Pearson claim.

## What this does and does not establish

- Establishes, for real, on-chain: a MapRec whose governance signature does not recover to the
  pinned authority is never trusted by the real entry point (fail-closed), and that the
  consequence of skipping this check (the ablated twin) is a genuine unsafe accept at the
  amount/role decision level, not merely "a bad config value existed somewhere."
- Does not re-verify `SubmitDecisionMapRecVerified` against the full ZK/light-client pipeline
  (`SemBound`/`EvAssuranceOK` are caller-supplied booleans here) -- this experiment isolates the
  MapRec-authentication guard specifically, matching the isolation level the existing Ab7
  endorsement-policy analog already used; both guards exist in this codebase but have not
  (yet) been combined into one entry point.
- A single pinned authority key, not a multi-party governance quorum over MapRec updates
  (e.g. 2-of-3 signers) -- the formal model's `MapRec` does not itself specify a particular
  governance arity, and a single-key design is the minimal real instance of "governance
  signature" needed to demonstrate the ablation; a quorum variant is a natural extension, not
  attempted here.
- `role->cap` is a flat JSON map, not independently varying `M_I`/`R_P` per the full formal
  `MapRec` tuple -- this targets exactly the signature-authentication half of Ab7, not every
  field the formal `MapRec` could carry.

## Files

- `maprec_ablation_2026-10-11/maprec_governance.go` (chaincode; canonical copy also at
  `~/fabric-network/authcc/chaincode/maprec_governance.go`)
- `maprec_ablation_2026-10-11/maprec_sign_main.go` (keygen/signing tool; canonical copy at
  `~/fabric-network/authcc/tools/maprec_sign/main.go`, excluded from the deployed package)
- `maprec_ablation_2026-10-11/maprec_results_2026-10-11.json` (all 8 `ReadDecision` records,
  raw, as queried from the live chain)
