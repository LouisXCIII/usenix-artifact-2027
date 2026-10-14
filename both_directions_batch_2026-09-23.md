# Both E6 directions scaled to multiple real transactions (2026-09-23)

User asked to test more transactions in both directions (the single-case
"proof of mechanism" limitation explicitly disclosed in
`evidence/f2b_real_tx_verify_2026-09-23.md`). Both directions now have a
real multi-transaction corpus, not n=1.

## Fabric -> Besu: scaled from n=1 to n=10

Submitted 10 real, varied `SubmitDecision` transactions in one batch
(role/amount chosen to span all three real outcome classes, not
cherry-picked to all succeed):

| # | role | amount | threshold | real class |
|---|---|---:|---:|---|
| 0 | manager | 200,000 | 500,000 | SAFE |
| 1 | manager | 600,000 | 500,000 | REJECT |
| 2 | director | 1,500,000 | 2,000,000 | SAFE |
| 3 | clerk | 5,000 | 10,000 | SAFE |
| 4 | clerk | 50,000 | 10,000 | REJECT |
| 5 | auditor | 100,000 | 500,000 | REEVALUATE |
| 6 | manager | 300,000 | 500,000 | REJECT (semBound=false) |
| 7 | director | 300,000 | 2,000,000 | REJECT (evAssuranceOk=false) |
| 8 | manager | 499,999 | 500,000 | SAFE |
| 9 | clerk | 9,999 | 10,000 | SAFE |

All 10 committed together in real block 324 (the orderer's own real
batching behaviour -- the same mechanism E4 already documented, not
engineered). Ran `fabric_extract_real_tx.js` on each of the 10 envelopes
(all 10 endorsement signatures independently verified off-chain first),
then `f2b_batch_verify.js`: deployed one `FabricRealTxVerifier`, verified
the real orderer block-commit signature once (shared by all 10, since
they're in the same block), then verified each of the 10 real endorsement
signature pairs individually on-chain.

**Result: 10/10 real transactions fully verified on-chain**
(`quorumMet=true`, `fullyVerified=true` for every one), plus the shared
block commitment verified true. 10 distinct real transaction hashes.
Contract: `0xf854f5B3E2B62f7e59D16246239687E8CD35D472`. Raw data:
`evidence/f2b_batch_verify_result_2026-09-23.json`.

Note: the *destination transfer class* (SAFE/REJECT/REEVALUATE, computed by
the Fabric-side `classify()`) is not re-derived by the Besu verifier -- what
Besu verifies is that the real Org1+Org2 endorsers genuinely signed this
exact transaction and it is genuinely committed, regardless of which class
the decision itself received. Spanning all three real classes here shows
the mechanism is agnostic to the decision outcome, as it should be: it
authenticates the transaction, not its content's classification.

## Besu -> Fabric: existing real N=15 corpus, reconfirmed live

Not new this turn -- already reconfirmed live in the immediately preceding
session step (`evidence/both_directions_retest_2026-09-23.md`) after fixing
the ledger-clock staleness: the real `SubmitDecisionTrustlessV7` corpus (X8:
2 cases, X9: 13 cases spanning honest/forged variants) ran fresh, live,
right now: **honest 3/3 SAFE, forged 0/12 SAFE (all correctly REJECTED)** --
15 real transactions total, already a real multi-transaction corpus on this
side (unlike the Fabric->Besu direction, which had only 1-2 before this
turn).

## Updated honest scope

- Fabric->Besu is now n=10 (was n=1), still smaller than the forward
  direction's dozens of transactions, but no longer a single-case proof of
  mechanism.
- Besu->Fabric remains n=15, confirmed live and fresh.
- Both directions are exercised against real transactions the live network
  itself produced (Fabric->Besu) or genuinely fresh evidence (Besu->Fabric);
  no side used self-generated/synthetic signed content in either batch.
