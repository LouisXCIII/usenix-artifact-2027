# Administrative-domain diversity ($q_D$) — first measured case, 2026-10-08

## Correction (same day, after external review)

A second review correctly identified that framing this as "administrative-domain
diversity" overclaims: `classify()` is a pure deterministic function, so running it on
two machines with identical inputs only confirms the same code produces the same output
on different infrastructure — it does not test cross-domain evidence verification,
light-client behavior, or independent governance, which is what domain diversity in this
paper's model actually means. The Clopper–Pearson bound originally computed below is
also not meaningful for a deterministic function (there is no probabilistic trial to
bound). The manuscript now calls this a **"deployment portability check"**, reports it
as a build/deployment sanity check rather than a statistical claim, and has removed it
from Contributions/Threats-to-validity, keeping it only as a minor appendix item. The
raw results below (5/5 match) are unchanged and still accurate; only the
interpretation/framing was corrected.

## What this closes (original framing, superseded by the correction above)

Table 8 ("Claim-to-evidence mapping") row "Admin.-domain diversity ($q_D$); behavior- vs.
policy-safety" previously read "Future work — one domain tested" (every earlier `authcc`
deployment, including the Fabric network used for all other results in this paper, runs as
multiple VMs under the *same* OpenStack tenant/operator on this host — not independent
administrative control, as the paper's own Limitations section already disclosed).

This experiment deploys a second, functionally identical `authcc` network under a
**genuinely separate administrative domain**: a different cloud provider (Amazon Web
Services, not the OpenStack installation this project otherwise runs on), a different
account, a different region, provisioned and billed independently of the host's
infrastructure.

## Infrastructure (real, not simulated)

- AWS EC2 instance (instance ID and public IP redacted for anonymity) ("authcc-domain2-aws"), region `ap-southeast-2`
  (Sydney), shape `t3.micro` (2 vCPU burstable, 1GiB RAM, 2GiB swap added), Amazon Linux
  2023. Root EBS volume resized 8GiB→30GiB (gp3) to fit the
  Fabric images and chaincode build.
- Fabric `test-network` sample topology (orderer + 2 CAs + peer0.org1 + peer0.org2),
  Hyperledger Fabric 2.5.16 / Fabric CA 1.5.15 — version-matched to the images already in
  use on the main host's `authcc` network for consistency.
- Channel `authchannel`, created and joined by both orgs on this AWS network independently
  of the main-host channel of the same name (separate genesis, separate MSPs, separate
  ledger — no cross-chain relationship; this is a second, parallel deployment, not a new
  org added to the existing channel).

## Chaincode deployed

Real `authcc` chaincode source (`~/fabric-network/authcc/`), packaged and installed on both
peers, approved by both orgs, committed at sequence 1 — `peer lifecycle chaincode
querycommitted` confirms `Approvals: [Org1MSP: true, Org2MSP: true]`.

**Excluded from this deployment** (deliberate, not an oversight): `chaincode/b1fair.go`,
`chaincode/s4_b1_entrypoints.go`, and `chaincode/b1_model_weights.json`. These implement the
B1-reevaluation ML-fairness ablation and `//go:embed` the trained model weights directly
into the chaincode binary — the exact pattern flagged and fixed as a live secrets-exposure
risk in the S9 red-team pass (see `project_chaincode_seq44_registry_state` memory / prior
evidence). Re-including them here, on a second deployment, would have reintroduced that
same exposure. `SubmitDecisionTrustless`/`SubmitDecision`/`classify()` (the functions this
experiment and the paper's core claims depend on) live in `lightclient.go` /
`authchaincode.go` and have no dependency on the excluded files — confirmed by grepping for
cross-file callers before building; `go build -mod=vendor` succeeded cleanly with the
exclusion.

## Experiment: replay the original 5 decisions on the new domain

Same deterministic function (`SubmitDecision` → `classify()`), same 5 decision IDs and
inputs as the original `authcc_redeployment_2026-09-21.md` on the main host, submitted as 5
independent real transactions on the AWS chain, `role=manager` (`RoleThreshold[manager] =
500000`), `sourceDomain="aws-sydney-domain2"`:

| decisionId | amount | semBound | evAssuranceOk | transferClass (AWS, this run) | transferClass (main host, original) | match |
|---|---|---|---|---|---|---|
| decision-001 | 700000 | true | true | REJECT | REJECT | yes |
| decision-002 | 300000 | true | true | SAFE | SAFE | yes |
| decision-003 | 100000 | false | true | REJECT | REJECT | yes |
| decision-004 | 300000 | true | true | SAFE | SAFE | yes |
| decision-005 | 100000 | false | true | REJECT | REJECT | yes |

5/5 match. Each record read back via `peer chaincode query ... ReadDecision` after a mined
`invoke` (not estimated/simulated) — raw JSON captured per decision, shown above as
summarized fields.

Final channel state: `peer channel getinfo -c authchannel` → `height=11` on the AWS network
(genesis + config updates + chaincode definition commit + 5 decision transactions).

## What this does and does not establish

**Does establish**: the contract's classification logic (`classify()`, Theorem T0's
non-widening behavior) is deterministic and reproducible when redeployed under independent
administrative control, independent infrastructure, and independent network — not an
artifact of one specific OpenStack deployment. This is the first real (not simulated)
multi-administrative-domain data point for $q_D$.

**Does not establish**: $q_D \geq 2$ in the full compositional sense of Section 5 (two
services under independent domains jointly gating one decision, per Proposition 1/2) — this
experiment is one chaincode replayed on two domains, not two independently-operated
services composed under a quorum policy $Q$. That remains open; this closes the narrower,
previously "Future work" claim about administrative-domain diversity for the authorization
logic itself, not the full composition claim.

## Evidence trail

- AWS Console: instance and EBS volume IDs redacted for anonymity.
- Deploy logs on AWS host: `/tmp/network_up.log`, `/tmp/deploy_authcc.log`,
  `/tmp/approve_commit.log`, `/tmp/invoke_decisions.log`.
- Local staging: `/tmp/claude-1001/authcc-aws-package/authcc-src.tar.gz` (source package sent
  to AWS, built with `go build -mod=vendor`, exit 0, no errors).
