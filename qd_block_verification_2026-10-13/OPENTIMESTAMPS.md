# OpenTimestamps proof (anonymity-preserving, per review 2026-10-13)

**Superseded same day by a broader manifest** (second external review pointed out the first
pass only covered `genesis.block` + root certs, not the actual artifact being claimed):
`manifest_combined_2026-10-13.sha256` = SHA-256 of every file in this package that matters to
the claims made about it — all 25 `.block` files (qscc-fetched), `genesis.block`, the 3 root
CA certs, `verifier/main.go` + `go.mod`/`go.sum` + `cmd/makenegatives/main.go`, every file under
`negative_tests/`, and, from the QD3 differential-test package
(`evidence/qd3_differential_test_2026-10-13/`), the frozen second-implementation source
(`independent_oblcovered.py`, `freeze_record.md`), the sweep generator and analysis scripts, and
every result file (`sweep_cases.json`, `layer1_cases.json`, the independent/real output JSONs,
`summary_sweep.json`, `summary_layer1.json`, `mismatches.json`). `manifest_combined_2026-10-13.sha256.ots`
is an OpenTimestamps proof for that manifest file's own hash, submitted to 4 public calendar
servers (a/b.pool.opentimestamps.org, a.pool.eternitywall.com, ots.btc.catallaxy.com). **Only
the hash was sent — no file content, no name, no identifying metadata.** This is the
anonymity-preserving alternative chosen instead of a named Git tag (which would deanonymize the
author under USENIX's double-blind review). The original narrower
`fingerprints.sha256`/`fingerprints.sha256.ots` (genesis block + root certs only) is kept,
superseded, not deleted.

**Re-stamped once, same day** (self-caught, not from a review): the first `manifest_combined`
stamp was submitted before the 5th negative test (`caseGeneric_random_byte_flip.block`) existed
and before `negative_tests/README.md` was corrected during a later review pass, so that stamp's
hash no longer matched the package's actual final content. The stale proof is kept, clearly
marked, at `manifest_combined_2026-10-13.sha256.ots.STALE_superseded` (do not cite it). The
manifest was regenerated from the current files (`diff` against the stale version confirmed
exactly two lines changed: the corrected `negative_tests/README.md` hash and the new
`caseGeneric` entry; nothing else in the package had drifted) and re-submitted to the same 4
calendars.

**Update (same day, confirmed complete): `ots upgrade` succeeded** — 2 of the 4 calendars
(`alice.btc.calendar.opentimestamps.org`, `bob.btc.calendar.opentimestamps.org`) returned real
Bitcoin block attestations, embedded as full Merkle paths directly in
`manifest_combined_2026-10-13.sha256.ots` (no further network access needed to check them). The
other 2 calendars (`catallaxy`, `finney`) are still pending; this does not weaken the proof —
one confirmed attestation is already sufficient, two is redundant margin.
`ots verify` itself fails on this machine only because there is no local Bitcoin node
(`~/.bitcoin/.cookie` missing) — not because the proof is invalid. Independently cross-checked
instead against a public, OpenTimestamps-unrelated block explorer (Blockstream): the Merkle
roots embedded in the `.ots` for both attested blocks (971214, 971212) match the real on-chain
values byte-for-byte. The proof is complete and ready to attach to the submission bundle.

**Re-stamped a second time, same day** (self-caught again, same lesson as the first re-stamp):
adding negative-test Case E (`TRANSACTIONS_FILTER` VALID→INVALID, `verifier/cmd/makecaseE/`)
and the corresponding `negative_tests/README.md` update changed 3 files the manifest covers,
so the confirmed proof above went stale again. The 2-calendar-confirmed `.ots` from the update
above is kept, clearly marked, at `manifest_combined_2026-10-13.sha256.ots.STALE_superseded_2`
(do not cite it). A fresh manifest was generated and re-submitted; initially only
`btc.calendar.catallaxy.com` accepted (the other 3 timed out, a transient outage).

**Final update (2026-10-14): confirmed complete.** `ots upgrade` now returns
`BitcoinBlockHeaderAttestation(971293)` for `manifest_combined_2026-10-13.sha256.ots` — a full,
self-contained Merkle path from the file's hash to Bitcoin block **971293**, requiring no further
network access to re-check. Verified three ways, not just trusted from the tool: (1) the hash the
`.ots` proof attests to was recomputed fresh (`sha256sum manifest_combined_2026-10-13.sha256`)
and matches byte-for-byte; (2) the manifest on disk was confirmed to already include the Case E
files (`verifier/cmd/makecaseE/main.go`, `negative_tests/tampered_blocks/caseE_filter_invalid_code.block`),
so this is the final, current package, not a stale one; (3) the attested Bitcoin transaction
(`db36f4b1666dccaac6daff6a105c54efc2c62b61ca158c2f749994e32522d04c`) was looked up independently
on a public block explorer unrelated to OpenTimestamps (Blockstream), which confirmed it is
mined, real, at block height 971293, block hash
`0000000000000000000062386c096d43754b74edf9f063156559f49d8e143957` — matching the `.ots` file's
own claim exactly. `ots verify` itself still errors locally only because this machine has no
local Bitcoin node configured (`~/.bitcoin/.cookie` missing) — a tool-configuration gap, not a
proof defect; the independent explorer check above supersedes it. One calendar path
(`btc.calendar.catallaxy.com`) is still pending, which is immaterial: one confirmed Bitcoin
attestation is already a complete, sufficient proof.

**Still open — needs the user's call**: whether to host this evidence package at a public
anonymous repo (e.g. `anonymous.4open.science`) so the paper can link to it directly, or to
rely solely on attaching the `evidence/` folder to the USENIX artifact-submission bundle itself
(no public link needed). Nothing has been published anywhere *about* this project so far — only
an anonymous hash was submitted to public timestamp servers.
