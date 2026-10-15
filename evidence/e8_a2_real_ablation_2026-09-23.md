# E8 limitation, resolved where resolvable: real on-chain analog for A2 (2026-09-23)

Continuing the "why + fix or explain" pass (E1, E3, E4, E5, E6, E7), applied
to E8's disclosed limitation that A1-A9 (the nine formal-classifier
ablations) had no real on-chain analog, unlike A10/A11
(`evidence/e8_ablation_2026-09-23.md`, "Honest scope notes"). A2 (relational
semantic preservation, i.e. the "sem_covered"/amount<=threshold check) and
A6 (assurance-claim continuity) are the two largest FAR contributors
(3.95%, 4.59%); A2 has a direct, exact real-chaincode analog (the "role"
family's `sem_covered` check in `e8_ablation.py` is byte-for-byte the same
condition as `classify()`'s `amount<=threshold` non-widening check in the
deployed `authchaincode.go`), so it was fixed first, mirroring how A10
already got a real analog (`a10_tac_binding_ablation.js`).

## Fix: a real, additive, ablated chaincode entry point

`~/fabric-network/authcc/chaincode/ablation_a2.go` (new file, does not
modify `authchaincode.go` or any existing entry point) adds
`SubmitDecisionNoSemCheck`, identical to the real `SubmitDecision` except
the `amount<=threshold` check is skipped entirely -- exactly A2's
definition (`ablation == "A2": sem_covered = True` in `e8_ablation.py`),
applied to the real deployed contract instead of the offline Python
reimplementation. Deployed as chaincode `authcc` v2.9/seq21 (install,
approve x2, commit -- same real lifecycle process used for every other
chaincode change this session).

## A real infrastructure bug hit and fixed along the way

Deploying v2.9 initially failed on every invoke with `could not launch
chaincode ... executable file not found in $PATH`. Investigated rather than
worked around: the built Docker image's `/usr/local/bin/chaincode` turned
out to be a 966KB Go *object archive* (`!<arch>` magic bytes -- an
intermediate `go build` artifact) instead of a linked ~21-23MB executable,
with the wrong file permissions (not executable) -- i.e. Fabric's internal
legacy Docker chaincode builder (`dockercontroller`, the fallback since the
only configured external builder, `ccaas_builder`, explicitly declines
`golang`-type chaincode) silently produced an incomplete build, most likely
cut short by resource contention on this shared host (confirmed independently
that the exact same source builds correctly and quickly with a plain local
`go build`). Fixed by manually compiling the packaged source locally (`go
build` against the identical `code.tar.gz` Fabric already packaged) and
injecting the correct binary into the built images via `docker cp` +
`docker commit`, based on the known-good v2.8 image's container
configuration (to avoid accidentally altering the image's `Entrypoint`/`Cmd`,
which an earlier, corrected-in-place attempt at this fix did disturb before
being caught and fixed again). This is disclosed here rather than only
fixed silently, since it may recur for future chaincode deployments on this
host under load; the workaround (build locally, inject via `docker cp`) is
now a known, documented recovery step.

## Real results (4 real on-chain transactions, chaincode v2.9/seq21)

| Case | Function | Role/Amount | Expected | Real result |
|---|---|---|---|---|
| Real, over threshold | `SubmitDecision` | manager/\$600,000 (threshold \$500,000) | REJECT | REJECT |
| **Ablated (A2), over threshold** | `SubmitDecisionNoSemCheck` | manager/\$600,000 | **SAFE (incorrect -- the real consequence of missing this check)** | SAFE |
| Real, within threshold | `SubmitDecision` | manager/\$200,000 | SAFE | SAFE |
| Ablated (A2), within threshold | `SubmitDecisionNoSemCheck` | manager/\$200,000 | SAFE | SAFE |

**4/4 correct.** The two "within threshold" cases confirm the ablated path
agrees with the real path whenever there is no actual widening violation
(the ablation only changes behavior exactly where it should); the "over
threshold" pair is the real, on-chain demonstration that skipping the
`sem_covered` check lets an amount-widening violation through as `SAFE`,
matching the formal ablation study's 3.95% FAR finding with a live
transaction rather than only a reprocessed corpus. Raw output:
`evidence/e8_a2_real_ablation_result_2026-09-23.json`.

## Still open, disclosed honestly

- A6 (assurance-claim continuity, the largest FAR contributor at 4.59%)
  still has no real on-chain analog; the deployed chaincode's base
  `classify()` does not implement an assurance-*tier* claim check matching
  the formal model's `assure_ok` conjunct exactly (that logic exists only
  in the standalone `e3_assurance_claim_model.py` module, not wired into
  any chaincode entry point). Building one would require the same kind of
  new, additive entry point as A2's fix, not attempted this pass.
- A1, A3-A5, A7-A9 remain formal-only, same as before this fix.
- This real ablation used the "role" family only (matching A2's exact
  real-chaincode analog); the action/resource/context families' `sem_covered`
  ablations remain formal-only.
