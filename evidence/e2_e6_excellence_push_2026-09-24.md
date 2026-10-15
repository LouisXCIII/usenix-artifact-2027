# E2 unit tests + E6 real ABAC workload (both directions) -- closing the two remaining "rất tốt" items (2026-09-24)

Direct follow-up to "E2 và E6 làm sao lên xuất sắc luôn" (how to push E2 and E6 to excellent),
after a per-E1-E10 grading pass rated both "rất tốt" (very good) rather than "xuất sắc"
(excellent) because neither had a dedicated, real depth pass of its own this session (E2 was only
validated indirectly via E1's Z3 regression; E6 was foundational infrastructure without an
explicit real ABAC demonstration, one of the guide's 6 named workload families).

## E2: 5 real unit tests closing the guide's remaining sub-requirements

Reuses the existing, unmodified `SubmitDecisionFull` entry point -- no new chaincode. Closes the
guide's E2 unit-test sequence items not yet explicitly, separately demonstrated:

| Test | Real inputs | Expected | Observed |
|---|---|---|---|
| Prohibited repair: widening | manager/\$600,000 (exceeds \$500k), approved=true | REJECT | **REJECT** |
| Prohibited repair: Fatal override | semBound=false, manager/\$200,000, approved=true | REJECT | **REJECT** |
| Fatal vs. Reevaluate precedence | semBound=false AND unknown role together | REJECT | **REJECT** |
| Genuine repair, positive control (a) | director/\$1,500,000 (>half), approved=false | CONDITIONAL | **CONDITIONAL** |
| Genuine repair, positive control (b) -- determinism | identical logical input, fresh decisionID | CONDITIONAL | **CONDITIONAL** |

**5/5 matched expectation**, independently re-confirmed via fresh `ReadDecision` reads. The first
three confirm `Repairable(Delta)` genuinely cannot be used to override a widening violation or a
Fatal (unbound-semantics) failure -- Fatal always takes precedence over both Repairable and
Reevaluate, exactly matching the guide's fail-closed precedence
(`Fatal > Portable/BiRefine > Repairable(Delta) > CanEval_d > Reject`). The last two confirm
Conditional is not blanket-disabled (a genuinely repairable case reaches it correctly) and that
classification is deterministic under a frozen registry/state (identical input, identical
output).

## E6: real ABAC workload, both directions

The guide's Section 8.3 lists ABAC (organization/attribute mapping with context) as a workload
family DISTINCT from RBAC (role alone). Every other real result in this paper uses role-only
(RBAC) authorization; a real, dedicated ABAC test had not been built. (This is different from the
2026-09-21 `rbac_vs_abac_onchain_2026-09-21.md` comparison, which tested attribute RICHNESS --
role-only vs. role+semBound+evAssuranceOK -- not organizational CONTEXT; the two are
complementary, not overlapping.)

### Fabric side (new): `abac.go` / `SubmitDecisionABAC`

New, additive chaincode (deployed as v2.23/seq35, the known Docker-build-binary bug recurred a
15th time this session and was fixed with the same routine procedure -- also recurred the
tarball-leftover packaging bug found earlier today, fixed the same way). Requires role AND the
caller's REAL organizational identity together
(`ctx.GetClientIdentity().GetMSPID()`, read from the actual cryptographic transaction identity,
not a caller-supplied argument):

| Case | Submitted by (real MSP) | Role | Amount | Expected | Observed |
|---|---|---|---|---|---|
| Org1, within Org1's real cap | Org1MSP | director | \$1,800,000 | SAFE | **SAFE** |
| Org2, SAME role+amount, exceeds Org2's real cap | Org2MSP | director | \$1,800,000 | REJECT | **REJECT** |
| Org2, within Org2's own real cap | Org2MSP | director | \$150,000 | SAFE | **SAFE** |

**3/3 matched expectation**, independently re-confirmed. The middle case is the genuine ABAC
demonstration: identical role and amount, but a real, different organizational identity produces
a real, different outcome -- not explainable by role alone.

### EVM side (new): `EVMDestinationGateABAC.sol` / `submitDecisionABAC`

Mirrors the Fabric-side design exactly (same role-threshold-per-organization structure), using
`msg.sender`'s real address (mapped to a real organization label at deployment from two already-
established real identities, `PRIVATE_KEY`/`PRIVATE_KEY_B`) as the real organizational attribute:

| Case | Submitted by | Role | Amount | Expected | Observed |
|---|---|---|---|---|---|
| Org1, within cap | Org1 identity | director | \$1,800,000 | SAFE | **SAFE** |
| Org2, same role+amount, exceeds cap | Org2 identity | director | \$1,800,000 | REJECT | **REJECT** |
| Org2, within own cap | Org2 identity | director | \$150,000 | SAFE | **SAFE** |

**3/3 matched expectation**, independently re-confirmed via fresh `getDecision` reads.

## Combined: E6's ABAC workload now closed in both directions

6/6 real ABAC transactions total (3 Fabric-side + 3 EVM-side), 6/6 correct, all independently
re-verified. Combined with the already-real RBAC, Threshold, State-dependent, Temporal, and
Obligation-bearing workload families (all demonstrated bidirectionally via the E3 reverse-
direction passes and their forward-direction counterparts), **all 6 of the guide's named E6
workload families now have real, bidirectional evidence**.

## Honest scope notes

- Both ABAC tests are n=3 each (n=6 combined), not a swept corpus -- matching the scope of the
  other single-mechanism demonstrations this session (S11/S15/S16/S18/S19's n=1-2), not a
  statistically powered campaign.
- The EVM-side "organization" attribute is a real address-to-label mapping fixed at deployment,
  not a live cross-chain-verified organizational credential -- a simpler real mechanism than
  Fabric's native MSP identity system, disclosed as such rather than overstated as equivalent.
- E2's determinism test used only 2 repetitions (not a larger repeated-trials campaign); this is
  sufficient to demonstrate the property qualitatively but not to bound a failure rate the way
  the session's larger CI-driven results do.

## Files

- `e2_unit_tests_result_2026-09-24.json` -- the 5 real E2 unit-test results.
- `abac_2026-09-24.go` -- the new Fabric ABAC chaincode (copy for reproducibility; canonical
  source at `~/fabric-network/authcc/chaincode/abac.go`).
- `e6_abac_result_2026-09-24.json` -- the 3 real Fabric-side ABAC results.
- `EVMDestinationGateABAC_2026-09-24.sol` -- the new EVM ABAC contract (copy for reproducibility).
- `e6_abac_evm_result_2026-09-24.json` -- the 3 real EVM-side ABAC results.
