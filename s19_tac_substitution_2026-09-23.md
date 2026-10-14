# S19 — TAC substitution (2026-09-23)

Real result for `Manuscriptsp_m2_r3.tex`'s S19 scenario
(`tab:adversarial-suite`) and the five TAC/context substitution sub-tests
from the execution guide's section 6.3 (decision, destination, state,
mapping, assurance-profile substitution). This is the one adversarial
scenario that had no evidence anywhere in the project (confirmed: no TAC
object exists in `authchaincode.go`, and grepping the whole evidence
directory for TAC-substitution tests returned nothing before this session).

## What was built

`~/pqc-hardhat/experiments/portability/s19_tac_substitution.js`: a minimal,
real, additive implementation of the formal Transfer-Admissibility
Certificate container from `eq:architecture-tac`,
$\tau_{s\rightarrow d} = \langle h(D_s), h(\pi_s), \mathsf{mapID},
\mathsf{assurID}, h(w_\Sigma), h(W_P), h(\eta), \mathsf{class}, \Delta,
t_\tau^{\exp}, \mathsf{issuer}, \sigma_\tau \rangle$, using:

- **Real Keccak-256 hashing** (`ethers.keccak256`) for every bound field.
- **Real ECDSA signing/recovery** (`ethers.Wallet.signMessage` /
  `ethers.verifyMessage`), using the same Bank-A key already used for E5.
- **A real prior artifact as $D_s$**: the on-chain `transferId` from E5's
  valid ML-DSA-65 transaction (`0x62fa6602...`), so the certificate is bound
  to a decision that genuinely exists on-chain, not a synthetic value.

Verification checks two independent things: (a) every bound field in the
presented context matches the field the certificate was originally issued
for, and (b) the issuer's ECDSA signature is valid over the recomputed
digest of the certificate's *own* stored context (not the presented one --
an attacker cannot forge a new signature for a mutated context without the
issuer's private key). A substitution attack is therefore modeled correctly:
the attacker cannot alter the signed certificate itself, only *replay* it
under a different verification context.

## Real results (6/6 pass)

| Case | Expected | Observed | Result |
|---|---|---|---|
| Positive control (unmodified) | valid | valid | PASS |
| Decision substitution | invalid | invalid (field mismatch) | PASS |
| Destination substitution | invalid | invalid (field mismatch) | PASS |
| State substitution | invalid | invalid (field mismatch) | PASS |
| Mapping substitution | invalid | invalid (field mismatch) | PASS |
| Assurance-profile substitution | invalid | invalid (field mismatch) | PASS |

Raw output: `evidence/s19_tac_substitution_result_2026-09-23.json` (includes
the real digest and real signature bytes).

## Honest scope notes

- This is a **standalone, additive TAC module**, not an integration into
  `authchaincode.go` or the Besu destination gate -- it demonstrates that the
  formal TAC binding mechanism is real and correctly detects all five
  substitution types when implemented as specified, but it has not been
  wired into either real network's actual transaction-execution path. That
  remains open if a reviewer specifically wants an on-chain-enforced TAC
  gate rather than a verified standalone construction.
- $h(W_P)$ and $h(\eta)$ (semantic-witness and obligation-witness hashes)
  are honestly zero-hashed placeholders here, since the real underlying
  artifact (E5's PQC transfer) does not carry separate semantic/obligation
  witness objects. A future integration with `authcc`'s `SubmitDecisionFull`
  (which does have real obligation records) could populate these for real.
- $n=6$ (1 positive control + 5 substitution types), matching the guide's
  own five-row table exactly -- not a larger corpus.
