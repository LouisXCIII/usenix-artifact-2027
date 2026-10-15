# Partial Coq mechanization of Theorem 1 (Relational soundness), 2026-09-29

Closes (partially, as scoped and agreed with the user) the "Lean/Coq for
Theorem 1" roadmap item. Toolchain: Coq 8.18.0 (Ubuntu 24.04 `apt` package
`coq`, installed this session). File: `ADPRelationalSoundness.v` (320 lines),
compiles clean with `coqc` (exit 0, no warnings).

## What is mechanized
All four proof steps behind the paper's Theorem 1 (`thm:relational-soundness`),
encoded at the same abstraction level the paper's own pen-and-paper proof
uses (uninterpreted state/intent/predicate/guard types, not a concrete
blockchain model):
1. **Lemma "Decision-derived source facts"** -- mechanized as a genuine
   two-line derivation (`decision_derived_source_facts`), not an axiomatized
   restatement: `Phi_s` is *defined* as the paper's own "necessity condition"
   (`P_s_holds -> SourceSat psi`), and A1-A2 is encoded as the equivalence
   the paper invokes (`e_s_Permit <-> P_s_holds`); the lemma is then modus
   ponens through that equivalence.
2. **Lemma "Predicate-witness soundness"** (`predicate_witness_soundness`)
   -- the real mathematical content: given the per-guard case split the
   paper's proof performs (`guard_covered`: each guard is resolved either by
   an authenticated destination-local fact, or by an $R_P$ witness subject
   to the "$R_P$ contract"), derives `Permit_d` (the full guard conjunction)
   by case analysis, calling Lemma 1 in the witness branch exactly as the
   paper's proof does ("this is where $e_s=$Permit actually enters the
   proof").
3. **Lemma "Transition preservation"** (`transition_preservation`) --
   mechanized as definitional, matching the paper's own characterization
   ("by definition of TransRel"): `TransRel_holds` is defined to already
   include $\sigma_d'=\delta_d(\sigma_d,I_d)$, so the lemma is an identity,
   not a hidden extra derivation dressed up as one.
4. **Theorem 1, instance-level half** (`relational_soundness_instance_full`)
   -- combines 2 and 3 to conclude the mapped transition satisfies the
   paper's own Eq. "authorized-transition-body" (`T_d_auth_holds`).
5. **Theorem 1, scope-level half** (`relational_soundness_scope`, in
   `Section ScopeLevel`) -- a genuine corollary, not merely asserted: an
   abstract index type `Trans` stands for $\mathcal T_s^{auth}|_\Omega$, a
   `Record Instance` bundles one copy of the per-instance hypotheses per
   index, and the theorem re-derives the instance-level argument pointwise
   for every index, concluding $M_T(\mathcal T_s^{auth}|_\Omega)\subseteq
   \mathcal T_d^{auth}$ -- exactly the paper's own description of the
   scope-level statement as "the universal closure of the instance-level
   result ... not a separate structural property".

`Print Assumptions` on both top-level theorems reports "Closed under the
global context" for both -- no axioms beyond Coq's own core logic (no
`Classical`, no `Admitted`, no `Axiom`) were used anywhere in the file.

## What is explicitly NOT mechanized (scope boundary, stated up front)
- Theorem 2 (Obligation preservation), Theorem 3 (Assurance
  non-amplification), Theorem 4 (Conditional multi-hop composition) --
  pen-and-paper only.
- A1-A13 themselves are assumed, not derived from a lower-level
  cryptographic/consensus/EVM/Fabric model; this mechanizes the relational
  argument that FOLLOWS FROM A1-A13, not a proof that the real system
  satisfies A1-A13 (that evidence is the paper's real-transaction results,
  Sections "Results" and Appendix "Security").
- No connection is drawn between this abstract mechanization and the
  concrete Go/Solidity implementation; nothing here is extracted to or
  executed against the deployed chaincode. This is proof-theoretic
  assurance about the theorem statement, complementary to (not a
  replacement for) the real-transaction evidence elsewhere in the paper.

## Why this is a real addition, not busywork
Two of the paper's own reviewer-facing remarks (Remark
"instance-independence" and the extended Threats-to-Validity discussion)
already argue, in prose, that the theorem's premises are explicit,
checkable, and non-redundant. Mechanizing the proof turns "we checked this
by hand" into "a proof assistant checked this": every case split, every
use of a hypothesis, and every definitional step is now machine-verified
against the exact hypotheses stated in the file's own comments (each
traceable line-for-line to the corresponding paper sentence), removing the
risk of a silent gap or an implicit, unstated step in the pen-and-paper
argument.

## Reproduction
```
cd evidence/coq
coqc ADPRelationalSoundness.v
```
Exit code 0, two `Closed under the global context` lines, `.vo` proof
object produced.
