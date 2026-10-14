# $q_D=2$: a second, genuinely independent administrative domain co-signs obligation-discharge coverage (2026-10-08)

## What this closes, and what it does not

S11b (same day, `evidence/s11_behavior_safety/results_2026-10-08.md`) found and fixed a real gap:
`obligationDischarged`/`enforcementWitnessVerified` were caller-supplied booleans, never
cross-checked against a real on-chain discharge event. S11b's fix
(`SubmitDecisionTrustlessV7ObligationVerifiedDischarge`) requires a real `DischargeObligation`
record bound to the same transfer — but that record lives on the SAME Fabric network, under the
SAME administrative control (this project's own OpenStack deployment), as the decision itself.
S11b's own "What this does and does not establish" section disclosed this is a single-domain
check.

This experiment adds a second, independently-governed co-signer: a secp256k1 key generated on,
and never transmitted off, a separate AWS account (IP redacted for anonymity), which independently
re-derives (from scratch, not by calling this project's Go code) whether the obligation is
genuinely covered, and signs an attestation only if it agrees. The main chain requires BOTH its
own check (S11b's logic) AND a genuine, agreeing AWS signature before SAFE.

**Closes**: an adversary who fully controls the OpenStack domain (including a dishonest or
compromised chaincode admin able to directly manipulate this chain's own ledger state) still
cannot fabricate AWS's agreement without the AWS-held private key — the property the first,
same-day "deployment portability check" experiment did not test, because it redeployed the same
deterministic code with no second key or domain involved at all.

**Does not close**: the AWS side is given the decision's content by an orchestrator script (it
does not independently query the main chain's ledger over a direct network path itself) and
re-validates that content's internal logic/consistency, rather than independently re-deriving
the underlying evidence from first principles. It defends against (a) forged/absent AWS
agreement and (b) a single buggy/compromised domain's logic error that an independently-written
second implementation would catch — not against an orchestrator that lies to both domains
identically, nor against compromise of both domains at once.

## A live regression found and fixed along the way

Before deploying, the currently-running package (`authcc_2.36/seq48`, today's S11b deployment)
was checked via `peer lifecycle chaincode getinstalledpackage` + `tar tzf`. It **still contained
`b1_model_weights.json`** — the exact raw ML model weights exposure S9 found and fixed on
2026-09-29 (`v2.30/seq42`). The fix had regressed: deployments after v2.30 used the runbook's
standard `rsync --exclude '*.tar.gz' --exclude authcc_old_packages --exclude '*_test.go'`
procedure, which does not exclude `b1fair.go`/`s4_b1_entrypoints.go`/`b1_model_weights.json`, so
every redeploy since silently reintroduced the exposure. Fixed as part of this deployment
(`v2.37/seq49` excludes all three files again); post-deploy verification via the same
`getinstalledpackage` + `tar tzf` check confirms their absence. **Action item, not yet done**:
update the runbook's standard deploy procedure to exclude these three files permanently, so this
does not regress a third time.

## Infrastructure

- Chaincode: new, additive file `~/fabric-network/authcc/chaincode/q_d_domain_attestation.go` —
  `SubmitDecisionTrustlessV7ObligationDomainVerified`, re-running S11b's exact evidence-chain and
  discharge-reference pipeline unchanged, plus a `secpecdsa.RecoverCompact`-verified AWS
  co-signature (same recovery pattern already used twice in this codebase: `lightclient.go`'s
  `verifyIbft2Header`, `ablation_a5_quorum.go`'s `verifyIbft2HeaderCustomQuorum`). `go
  build/vet/test` clean. Deployed as `authcc` v2.37/seq49 on the real `authchannel`, both orgs
  approved and committed.
- AWS domain authority: secp256k1 key generated via `qd2aws keygen` on
  `ec2-user@<redacted-ip>`, never transmitted off that machine. Address
  `0x6ad6b3ee6b4db60c28eb9de1b73945371ffcc063`, hardcoded into the chaincode as
  `awsDomainAuthorityAddress` (same pattern as `pinnedValidatorHex`/`policyAuthorityAddress`).
- `qd2aws` (`~/qd2_aws/` on AWS): a standalone Go program, `go build` clean, independently
  re-implementing the obligation-coverage model (role/threshold map, mechanism-strength map,
  confused-deputy binding check) from the paper's own model description — not by importing or
  calling authcc's Go code. Its signature scheme (`ecdsa.SignCompact`) was verified, before any
  real transaction, to recover correctly via the exact chaincode-side
  `secpecdsa.RecoverCompact` call (`qd2_aws/verify_test.go`, `go test` pass).
- Orchestration: `evidence/qd2_domain_attestation/run.py`, importing
  `evidence/s8_population/s8_run.py` the same way `evidence/s11_behavior_safety/run.py` did,
  reusing real fetched Besu header/proofs for fixture `03_dos_safe.sol` (same fixture S10/S11b
  used). SSH/scp round-trip to the AWS box for every signature, no simulation.

## Real on-chain results (n=7 mined transactions, height 4780→4787)

**(a) Genuine cross-domain agreement.** Real `SubmitDecisionFull` → `CONDITIONAL`; real
`DischargeObligation` → `EXACT`; AWS independently re-derives `passed=true` and signs; submitted
to `SubmitDecisionTrustlessV7ObligationDomainVerified` → **SAFE**. Recorded
`obligationDetailV2`: "...genuine signature from `0x6ad6...cc063`, independently re-derived
covered=true". `awsDomainVerified=true`.

**(b) Forged AWS signature.** The identical genuine claim (same `obligationDischargeRef`, same
amount/role), but signed by a freshly generated key that is NOT the registered AWS domain
authority → **REJECT**. Recorded `obligationDetailV2`: "...signature recovered to
`0xeed91cc2...`, not the pinned AWS domain authority `0x6ad6...cc063`". This is the core
security demonstration: full control of the main chain's own ledger state (the forging party
here submits from the same identity as every other transaction in this experiment) is not
sufficient to fabricate the second domain's agreement.

**(c) Boundary probe.** Amount set exactly to the `manager` role's threshold (500,000, the
boundary itself, not above or below it). Both the chain's own logic and AWS's independent
re-implementation agreed: `passed=true`/`SAFE`. **No divergence found** at this boundary — an
honest negative result, reported as such rather than omitted or replaced with a constructed
positive case. This does not establish the two implementations never diverge anywhere, only that
they agree at this one boundary tested.

Raw log: `evidence/qd2_domain_attestation/qd2_result.json`. Full `ReadDecision` records for cases
(a)/(b) quoted above are byte-for-byte from real `peer chaincode query` output, not summarized.

## Independent red-team pass (same day) found and fixed two real issues

Following the exact S9/S10 methodology — a fresh agent, given ONLY the 5 relevant Go source
files (no design rationale, no test scripts, no knowledge of this evidence file), asked to find
any way to get an unsafe SAFE — found **2 confirmed exploitable issues**, both now fixed and
re-verified live:

**CRITICAL — self-fabricated discharge defeats both domains.** `DischargeObligation`
(`authchaincode.go`) had **no caller/identity check at all** — only that the target record was
`CONDITIONAL` with `RequiresObligation=true`. Those two flags come from `SubmitDecisionFull`,
whose `semBound`/`evAssuranceOK`/`secondApprovalObtained` are raw caller-supplied booleans with
zero cryptographic backing. An attacker could therefore: (1) call `SubmitDecisionFull` claiming
`semBound=true, evAssuranceOK=true` with no real proof behind either claim → `CONDITIONAL`; (2)
call `DischargeObligation` on it (anyone could) → `ObligationDischarged=true`; (3) cite it as
`obligationDischargeRef` in `SubmitDecisionTrustlessV7ObligationDomainVerified`. Both the
main-chain discharge check and the AWS independent check only verify the referenced record's
`ObligationDischarged` flag and the (sourceDomain,role,amount) triple — **neither ever checked
whether the referenced record's own semBound/evAssuranceOK claims were themselves real**. An
honest orchestrator relaying this self-fabricated record's real (on-chain, but meaningless)
content would get a genuine AWS signature on `passed=true` — both "independent" domains
genuinely agree, SAFE is returned, and the obligation was never actually discharged by anyone
with real authority. This defeats the entire $q_D=2$ mechanism for this specific vector without
needing the AWS key at all.

**HIGH — discharge records were not single-use.** One genuine discharge for a given
(sourceDomain,role,amount) triple could be cited by unlimited distinct decisionIDs — nothing
bound a discharge to the specific transfer it was meant to cover.

**Tried and found NOT exploitable** (reported for completeness, not omitted): digest/byte-encoding
mismatch between the chaincode and `qd2aws` (confirmed identical on both sides); forging the AWS
signature without the key (the signed digest bakes in `passedByte`, so any tamper changes the
recovered address); decisionID replay (blocked by `decisionExists`); `amount<=0`/unknown
role/profile/mechanism (all correctly REJECT/REEVALUATE); missing/garbage AWS signature (forces
REJECT); keccak/address derivation (Ethereum-convention-identical on both sides);
hash-concatenation field-boundary ambiguity (exists in principle, not exploitable because the
on-chain confused-deputy check re-reads the real tx fields directly, not a value reconstructed
from the digest).

### Fixes (both additive at the call-site, one touches a pre-existing shared function)

1. `DischargeObligation` (`authchaincode.go`) now requires caller identity
   `Org1MSP`/CN=`org1admin` — the exact pattern already used by `SetClock` (`clock.go`), reusing
   `ctx.GetClientIdentity()`/`GetMSPID()`/`GetX509Certificate()`. Every evidence script in this
   project (S10, S11b, this work) already runs as this identity (confirmed via `s8_run.py`'s
   `env`), so no existing evidence or script needed to change. **Residual, disclosed scope**:
   this is a trusted-operator model (same category as the clock authority already is), not
   genuine separation of duties — it stops an outside attacker from self-discharging a
   fabricated record, not a dishonest `org1admin` from doing so deliberately.
2. `SubmitDecisionTrustlessV7ObligationDomainVerified` (`q_d_domain_attestation.go`) now writes
   a `discharge-used:<ref>` marker on first successful citation and rejects any second citation
   of the same discharge reference ("one discharge covers one decision, ever").

**Live re-verification on the fixed deployment (`authcc` v2.38/seq50, both orgs approved and
committed)**, each a real mined transaction:
- Unauthorized discharge attempt (`Org2MSP`/`org2admin`) on a real CONDITIONAL record →
  **rejected**: `"only Org1MSP/org1admin may discharge an obligation (caller Org2MSP/org2admin)"`.
- The same record discharged by the legitimate identity (`Org1MSP`/`org1admin`) → succeeds
  (`EXACT`), confirming the honest path still works.
- First citation of that real discharge → `SAFE`. Second citation of the *same* discharge by a
  *different* decisionID → **REJECT**:
  `"referenced record ... was already consumed by decision ... (one discharge covers one
  decision; re-citation rejected)"`.

## Deep sweep, n=30 both directions (v2.38, pushing the n=1 pass above to a statistical bound)

Matching this project's own established "xuat sac" bar (S10's precedent: n=30, Clopper–Pearson
95% both directions, independent red-team, found-and-fixed issues re-verified): 30 distinct real
CONDITIONAL+discharge cycles (`role=manager`, amounts spread 250,001–491,666 — the valid
CONDITIONAL range below threshold 500,000 — crossing all 4 obligation profile/mechanism
combinations, some with mapped mechanism strictly stronger than required). *Role note*: `v7_args`
in the shared `s8_run.py` helper hardcodes `role="manager"` into the evidence-chain args (a
module-level constant, not a parameter) — an earlier run of this sweep against v2.37 mixed
`director`/`clerk` configs and got 20/30 spurious REJECTs, which turned out to be the (correctly
working) confused-deputy check catching the harness's own role mismatch, not a chaincode defect;
confirmed by inspecting the REJECT details before concluding anything, then narrowed the sweep to
`role=manager` only. Full file: `evidence/qd2_domain_attestation/qd2_sweep_result.json`
(role-bug run preserved as `qd2_sweep_result_v237_rolebug.json` for the record, not deleted).

Each of the 30 real discharges had exactly one genuine citation and one forged-signature citation
submitted against it (60 citation transactions + 60 conditional/discharge transactions = 120
real mined transactions total for the sweep):

- **Genuine agreement: 30/30 → SAFE.** One-sided Clopper–Pearson 95% upper bound on a false-reject
  rate: **9.5%**.
- **Forged signature: 30/30 → REJECT.** One-sided Clopper–Pearson 95% upper bound on a
  false-accept rate: **9.5%**.

## Threats and scope, stated plainly

- This is one additional entry point alongside, not a replacement for, S11b's single-domain
  fix — both remain available; this is the stronger alternative for destinations that want the
  discharge claim's agreement cryptographically bound to a second, separately-governed party.
- The AWS side's "independent verification" is re-validation of orchestrator-supplied content,
  not independent re-derivation of the underlying evidence from first principles (see "What this
  does not close" above) — this is attestation-based domain separation, not a second full
  light-client/evidence pipeline.
- The discharge-authority access control (fix 1) is a trusted-single-operator model, not genuine
  separation of duties — a dishonest `org1admin` could still self-fabricate and self-discharge a
  record. Closing this fully would need a genuinely separate approver identity/domain for the
  discharge step itself, not just for the final co-signature — disclosed as future work, not
  claimed solved here.
- `enforcementWitnessVerified` remains caller-supplied on every entry point in this codebase,
  including this new one — the S7-laundering gap is unaffected by this work and remains open.
