# Negative tests — each check must fail for its OWN stated reason, independently

The verifier (`verifier/main.go`) runs every check (dataHash, endorsements, endorsementPolicy,
ordererSig, validationFilter, hashChain) independently and reports each one's own PASS/FAIL —
it never stops at the first failure, so a FAIL can always be attributed to the exact check that
caught it. Generator: `verifier/cmd/makenegatives/`. All tampered copies live only in
`tampered_blocks/` here (and `/tmp` while testing) — the real `../blocks/` package is untouched.

All 6 cases below have a saved `.block` file under `tampered_blocks/` and can be re-checked
by copying the file into its own directory as `b<any-number>.block` (the verifier's directory
scan requires that exact naming) and running `qdblockverify verify <that-dir> ../certs`.

| Case | What was changed | dataHash | endorsements | endorsementPolicy | ordererSig | validationFilter |
|---|---|---|---|---|---|---|
| Generic byte flip (regenerated 2026-10-13; the original Bước 1 one-off test that first found the DataHash gap was run but its tampered file was not saved as a standalone artifact — this is a freshly generated, saved, re-checkable replacement using the same technique: midpoint of a different block's envelope, `blocks/b5085.block`) | 1 byte flipped at the envelope midpoint, header untouched | **FAIL** | PASS | PASS | PASS | PASS |
| A: RWSet byte flip, DataHash left stale | 1 byte flipped at the envelope midpoint of `blocks/b5082.block` (fell inside an unsigned region), header untouched | **FAIL** | PASS | PASS | PASS | PASS |
| B: same flip, DataHash recomputed to match | as A, but DataHash updated to the new (tampered) Data | PASS | PASS | PASS | **FAIL** | PASS |
| C: whole envelope spliced from a different real block | tx 0 replaced by a genuinely-signed-elsewhere envelope, DataHash recomputed | PASS | PASS | PASS | **FAIL** | PASS |
| D: endorser identity replaced with an untrusted self-signed cert | endorsement 0's Endorser PEM swapped for a cert with no relation to any loaded root, DataHash recomputed | PASS | **FAIL** (cert chain to Org2MSP root: unknown authority) | **FAIL** (cascades: only 1 verified org) | **FAIL** | PASS |
| E: `TRANSACTIONS_FILTER` byte changed from VALID to an invalid code | byte 0 of `blocks/b5082.block`'s (qscc-fetched) validation-code slot changed `0x00`$\to$`0x01`, nothing else touched | PASS | PASS | PASS | PASS | **PASS (see below — not a FAIL case)** |

Every FAIL above is attributed to the exact mechanism being tested, never to a different check
than the one the case was designed to isolate:

- **A** confirms the generic byte-flip gap found in Bước 1 is real and reproducible (DataHash
  alone catches tampering outside the signed regions; nothing else does).
- **B and C** confirm the real defense-in-depth story even when an attacker is sophisticated
  enough to recompute DataHash correctly: without the orderer's private key, `ordererSig` still
  catches it every time, because the orderer's signature covers the header (which embeds
  DataHash) — no amount of re-hashing Data alone can forge it.
- **D** confirms the cert-chain-to-root check independently rejects an untrusted CA, with the
  endorsement-policy check correctly cascading (a rejected endorsement no longer counts toward
  the 2-org requirement) — both failing for their own distinct, correctly-worded reasons.

**Case E, full explanation.** Unlike A–D, flipping the `TRANSACTIONS_FILTER` byte does **not**
flip `validationFilter` to FAIL in `verify`'s output — and this is correct, not a bug. Reading
`computeTxValidity` in `verifier/main.go`: the `len(filter) == n` branch (the normal, populated
case, confirmed exercised by all 25 real qscc-fetched blocks) reports `PASS` whenever the slot
has the right *length* for the transaction count — it is a check that the slot is
**structurally well-formed**, not a claim that every transaction it describes is VALID. The
per-transaction validity itself is recorded in the returned `TxValidity` array, which is used
downstream by `cmdRecover`, not by `verify`'s block-level summary. Confirmed directly: running
`qdblockverify recover` against this tampered block for the decision it contains
(`qd3-s5-verified-1791859143`) returns `NOT FOUND (or not VALID)` — `recover` correctly refuses
to trust the now-invalid-coded transaction, exactly as designed. So Case E demonstrates a real,
two-layer design, not a single PASS/FAIL check: `verify`'s `validationFilter` check validates
*slot format* (and correctly still reports PASS here), while `recover`'s per-transaction
filtering is what actually enforces *content* trust (and correctly excludes this transaction).
Reporting this as a simple "FAIL" would misstate what the check promises; reporting it as a
silent "PASS, nothing to see" would hide that `recover`'s real protection was exercised and
held. Both halves are disclosed here instead.
