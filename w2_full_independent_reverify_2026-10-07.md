# Independent re-verification of Ab1/Ab3/Ab5/Ab8 decisions, current scale (2026-10-07)

Supersedes the 2026-09-24 check (72/72 match, 88 transactions), which covered
Ab1/Ab5 at their pre-upgrade scale (n=5 each at the time). Ab1 and Ab5 were
later upgraded to n=60 (2026-10-06); this check re-derives the correct totals
at the current scale and re-runs the independent re-verification for real
against the live Fabric ledger, rather than editing the old numbers by hand.

## Scale

| Mechanism | n (real-path trials) | Transactions | Distinct decision IDs |
|---|---|---|---|
| Ab1 (state correspondence) | 60 | 120 (60 real + 60 ablated) | 120 |
| Ab5 (finality) | 60 | 120 (60 real + 60 ablated) | 120 |
| Ab3 (obligation) | 18 | 36 (18 real + 18 ablated) | 36 |
| Ab8 (replay) | 8 pairs | 32 (original submit + resubmission attempt, both paths) | 16 (resubmission reuses the original decision ID; it does not mint a new one) |
| **Total** | | **308** | **292** |

## Method

Script `experiments/portability/w2_full_independent_reverify_2026-10-07.js`:
for each of the 292 distinct decision IDs recorded by the four mechanisms'
own saved result files (`w2_ab1_ab5_n60_result.json`,
`a3_obligation_sweep_result_2026-09-24_verified.json`,
`a8_replay_sweep_result_2026-09-24.json`), issue a fresh
`peer chaincode query ... ReadDecision` call against the live `authcc`
chaincode (v2.35/sequence 47, restarted from its Docker volume for this
check -- `authcc_peer0.org1.example.com`/`authcc_peer0.org2.example.com`,
confirmed via `docker inspect`, not the empty `compose_*` volume this
project has hit before) and compare the on-chain fields
(`sourceDomain`, `role`, `amount`, `sourceLimit`, `semBound`,
`evAssuranceOk`, `transferClass`, `destThreshold`) against the value each
run's own script recorded at submission time. For Ab8, the expected value
is the pair's own recorded *final* class (`finalReal`/`finalAbl`), since a
resubmission attempt can overwrite the ablated-path record.

## Result

**292/292 match, 0 mismatch, 0 query error.**

This is a data-integrity check, not a statistical error-rate claim: it
confirms the chain's own stored state matches what each run's script
recorded immediately after submission, catching any silent corruption,
wrong-identity overwrite, or transcription error between submission and
the numbers quoted in the paper -- it does not re-derive or re-run the
original FAR/Clopper-Pearson bounds, which are unaffected by this check.

Raw: `experiments/portability/w2_full_independent_reverify_2026-10-07_result.json`.
