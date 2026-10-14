(* ADPMultiHopComposition.v
   Mechanization of Theorem 4 ("Conditional multi-hop composition",
   \label{thm:multihop-composition}, Manuscriptsp_m2_r3_usenix.tex), at the
   same abstraction level as ADPRelationalSoundness.v's mechanization of
   Theorem 1 ("Relational soundness").

   Self-contained (does not Require Import ADPRelationalSoundness.v): the
   per-hop soundness argument below is structurally identical to that
   file's predicate_witness_soundness/relational_soundness_instance_full
   (same guard_covered case split, same R_P-contract pattern), re-derived
   here per hop rather than imported, so this file compiles and can be
   checked independently, and so a reader can see the per-hop argument is
   exactly Theorem 1's, not a different one.

   Scope, stated honestly (mirroring ADPRelationalSoundness.v's own
   scope note):
   - A path of n hops (domains B_0..B_n) is modeled with ONE shared state
     type `Dom` and ONE shared intent type `Int` threaded across the whole
     path -- justified because "destination state of hop j" and "source
     state of hop j+1" are the same object (domain B_j), not two separate
     types needing separate declaration per hop.
   - Each hop j (1<=j<=n) has its own R_Sigma, M_I, and guard apparatus
     (Guard, DestLocal, Witness, the R_P contract, guard_covered), and its
     own delta_d (that hop's destination's own transition function) --
     all indexed by j via Coq Variables of type `nat -> ...`.
   - Theorem 2 (Obligation preservation) and Theorem 3 (Assurance
     non-amplification) are NOT separately re-mechanized here, exactly as
     the paper's own pen-and-paper Theorem-4 proof does not re-derive
     them -- it invokes them as already-established per-hop facts in the
     induction step ("Theorem~\ref{thm:obligation-preservation} preserves
     obligations at each hop ... Theorem~\ref{thm:assurance-nonamplification}
     prevents assurance laundering at each hop"). Their CONCLUSIONS are
     taken here as per-hop hypotheses (ObligationsPreserved j,
     AssuranceNotAmplified j), exactly as A1-A13 are assumed rather than
     re-derived from a lower-level model in ADPRelationalSoundness.v.
   - Freshness/replay/finality-at-every-hop is likewise an abstract
     per-hop Prop hypothesis (TemporalValid j), not modeled at the
     clock/ledger level (that is the real-transaction evidence elsewhere
     in the paper -- S6/w2_s18/Ab11 -- not this proof-theoretic layer).
   - The genuinely NEW mathematical content mechanized here, matching the
     paper's own description of "the substantive claim" ("these purely
     per-hop, local premises are jointly sufficient for the global n-hop
     conclusion; the conclusion is not assumed as a separate composed
     requirement"), is: (a) each hop's own Theorem-1-style guard
     conjunction (Permit_d at that hop) is re-derived from that hop's own
     guard apparatus, chained hop-to-hop via a continuity hypothesis
     linking hop j's destination state/intent to hop (j+1)'s source
     state/intent (the paper's "mapped output becomes the next hop's
     input"); and (b) the composed intent mapping M_I^{0:j} (chaining
     every hop's own M_I from the origin) is well-defined end to end and
     equals, at every hop, that hop's own locally-required intent --
     proved by induction on hop count, not assumed. This is the
     mechanized form of "pairwise success alone is insufficient ... the
     conclusion is not assumed as a separate composed requirement".
   - The compatible-scope premise ($M_I^{j-1:j}(\Omega_j)\subseteq\Omega_{j+1}$)
     is made genuinely load-bearing, not merely named: only the ORIGIN
     hop's scope membership is assumed (scope_member_origin); membership
     at every later hop is a separate theorem proved by induction
     (multihop_scope_propagation), which does not go through without
     scope_compatible -- removing that hypothesis breaks the induction
     step, not just the paper-fidelity of the premise list.
   - The adjacent-hop field-preservation premise (FieldsPreserved/
     fields_preserved_holds, the premise Ab11 shows empirically necessary
     on-chain) is made genuinely load-bearing, the same way
     scope_compatible was: an EARLIER draft assumed ObligationsPreserved
     directly at every hop, which made FieldsPreserved vacuous (caught by
     the mechanical necessity check, evidence/coq/check_necessity.py, and
     independently by external review). Fixed: only the origin hop's
     obligation-preservation is assumed (obligations_hold_origin);
     propagation to every later hop is a separate theorem
     (multihop_obligations_propagation), proved by induction, consuming
     FieldsPreserved at every step -- removing fields_preserved_holds now
     breaks this proof. The premise Ab11 shows is empirically necessary
     is therefore also formally necessary here: both facts are about the
     SAME premise, not two different things that happen to share a name.
   - A state-side continuity hypothesis (continuity_state) was present in
     an earlier draft; the mechanical necessity check showed it was never
     consumed (the conclusion proved here makes no claim about the final
     hop's state relative to the origin's, only its intent), so it was
     removed rather than kept as a named-but-unexercised premise.
   - Still taken as per-hop hypotheses, not re-derived (same justification
     as A1-A13 in ADPRelationalSoundness.v: real conditions for the
     system, established as empirical evidence elsewhere in the paper --
     S6/w2_s18/Ab11 -- not re-derived at this proof-theoretic layer):
     AssuranceNotAmplified (Theorem 3's conclusion) and TemporalValid
     (freshness/replay/finality; note freshness's own $t_0$ mechanism is
     NOT represented here at all, only an abstract per-hop validity Prop).

   Checked with: coqc, Coq 8.18.0 (same toolchain as ADPRelationalSoundness.v).
   Mechanical hypothesis-necessity check (evidence/coq/check_necessity.py)
   re-run after this fix: see coq_mechanization_multihop_2026-10-11.md for
   the current result.
*)

Require Import Arith.
Require Import Lia.

Section MultiHop.

Variable Dom : Type.   (* one state type, shared across every domain B_0..B_n *)
Variable Int : Type.   (* one intent type, shared across every domain B_0..B_n *)

(* Path length: n hops (B_0,...,B_n). Hop j, for 1<=j<=n, is B_{j-1}->B_j. *)
Variable n : nat.

(* ---- Per-hop data, indexed 1..n (hop j uses index j). ---- *)
Variable I_s : nat -> Int.                    (* hop j's own local source intent *)
Variable sigma_s : nat -> Dom.                (* hop j's own local source state *)
Variable sigma_d : nat -> Dom.                (* hop j's own local destination state *)
Variable I_d : nat -> Int.                    (* hop j's own local destination intent *)
Variable sigma_d' : nat -> Dom.               (* hop j's post-transition destination state *)
Variable delta_d : nat -> Dom -> Int -> Dom.  (* hop j's destination's own transition fn *)
Variable M_I : nat -> Int -> Int.             (* hop j's own intent mapping *)
Variable R_Sigma : nat -> Dom -> Dom -> Prop. (* hop j's own state-correspondence relation *)

Variable SrcPred : Type.
Variable GuardIdx : Type.
Variable SourceSat : nat -> SrcPred -> Prop.     (* hop j's source-predicate satisfaction *)
Variable Guard : nat -> GuardIdx -> Prop.        (* hop j's destination guard conjunct *)
Variable DestLocal : nat -> GuardIdx -> Prop.
Variable Witness : nat -> GuardIdx -> SrcPred -> Prop.
Variable e_s_Permit : nat -> Prop.
Variable P_s_holds : nat -> Prop.

Hypothesis A1_A2 : forall j, e_s_Permit j <-> P_s_holds j.
Definition Phi_s (j : nat) (psi : SrcPred) : Prop := P_s_holds j -> SourceSat j psi.

Hypothesis destlocal_sound :
  forall j k, DestLocal j k -> Guard j k.
Hypothesis RP_contract :
  forall j k psi, Witness j k psi -> R_Sigma j (sigma_s j) (sigma_d j) ->
                  SourceSat j psi -> Guard j k.
Hypothesis guard_covered :
  forall j k, DestLocal j k \/ (exists psi, Witness j k psi /\ Phi_s j psi).
Hypothesis R_ADP_sem : forall j, e_s_Permit j /\ R_Sigma j (sigma_s j) (sigma_d j).
Hypothesis TransRel : forall j,
  R_Sigma j (sigma_s j) (sigma_d j) /\ sigma_d' j = delta_d j (sigma_d j) (I_d j).
Hypothesis I_d_is_MI : forall j, I_d j = M_I j (I_s j).

(* ---- Per-hop Theorem 1 instance, re-derived here (not imported) ----
   Identical argument to ADPRelationalSoundness.v's
   predicate_witness_soundness / relational_soundness_instance_full. *)
Definition Permit_d_at (j : nat) : Prop := forall k : GuardIdx, Guard j k.

Lemma permit_d_at_sound : forall j, Permit_d_at j.
Proof.
  intros j k.
  destruct (guard_covered j k) as [Hloc | [psi [Hwit Hphi]]].
  - exact (destlocal_sound j k Hloc).
  - apply (RP_contract j k psi Hwit).
    + destruct (R_ADP_sem j) as [_ HR]. exact HR.
    + apply Hphi. apply (A1_A2 j). destruct (R_ADP_sem j) as [He _]. exact He.
Qed.

Definition T_d_auth_at (j : nat) : Prop :=
  Permit_d_at j /\ sigma_d' j = delta_d j (sigma_d j) (I_d j).

Lemma t_d_auth_at_sound : forall j, T_d_auth_at j.
Proof.
  intro j. split.
  - exact (permit_d_at_sound j).
  - destruct (TransRel j) as [_ Heq]. exact Heq.
Qed.

(* ---- Continuity: hop j's destination becomes hop (j+1)'s source --
   "each hop's mapped output ... [is] the immediately next hop's" input.
   Only the intent-level continuity is needed by any conclusion proved
   below (the conclusion makes no claim about the final hop's STATE
   relative to the origin's), so no state-level continuity hypothesis is
   declared -- one was present in an earlier draft and the mechanical
   necessity check confirmed it was never used; removed rather than left
   as a named-but-unexercised premise. ---- *)
Hypothesis continuity_intent :
  forall j, 1 <= j -> j < n -> I_s (S j) = I_d j.

(* ---- Scope compatibility: M_I^{j-1:j}(Omega_j) subseteq Omega_{j+1}.
   Omega j is hop j's own certified domain (a predicate on Int). ---- *)
Variable Omega : nat -> Int -> Prop.
(* Only the ORIGIN hop's membership is assumed; membership at every later
   hop is PROVED below (multihop_scope_propagation), not assumed -- this
   is what makes scope_compatible load-bearing rather than decorative. *)
Hypothesis scope_member_origin : Omega 1 (I_s 1).
Hypothesis scope_compatible :
  forall j, 1 <= j -> j < n -> forall i, Omega j i -> Omega (S j) (M_I j i).

(* ---- Field preservation: hop j's mapped output preserves every field
   hop (j+1)'s own certified MapRec declares mandatory -- modeled
   abstractly as a per-adjacent-pair Prop. Made genuinely load-bearing
   below (multihop_obligations_propagation): only the ORIGIN hop's
   obligation-preservation is assumed; propagation to every later hop is
   PROVED, consuming FieldsPreserved at each step -- matching the paper's
   own proof sketch exactly ("the adjacent-hop field-preservation premise
   closes the one gap that per-hop obligation preservation alone does not
   cover"). An earlier draft assumed ObligationsPreserved directly at
   every hop, which made this premise vacuously unused (the mechanical
   necessity check caught it); fixed the same way scope_compatible's
   earlier vacuity was fixed, not by assumption but by induction. ---- *)
Variable FieldsPreserved : nat -> Prop.
Hypothesis fields_preserved_holds : forall j, 1 <= j -> j < n -> FieldsPreserved j.

(* ---- Theorem 2 / Theorem 3 conclusions. Obligation preservation: only
   the origin is assumed; later hops are proved (below), consuming
   fields_preserved_holds. Assurance non-amplification and temporal
   validity remain direct per-hop hypotheses, as in the first draft --
   the user's review asked specifically about the field-preservation/
   obligation pairing (the one Ab11 measures), not these two. ---- *)
Variable ObligationsPreserved : nat -> Prop.
Variable AssuranceNotAmplified : nat -> Prop.
Variable TemporalValid : nat -> Prop.
Hypothesis obligations_hold_origin : ObligationsPreserved 1.
Hypothesis obligations_propagate :
  forall j, 1 <= j -> j < n -> ObligationsPreserved j -> FieldsPreserved j ->
            ObligationsPreserved (S j).
Hypothesis assurance_hold : forall j, 1 <= j -> j <= n -> AssuranceNotAmplified j.
Hypothesis temporal_hold : forall j, 1 <= j -> j <= n -> TemporalValid j.

(* ---- The composed mapping M_I^{0:j}: chain hop mappings 1..j from the
   path's origin intent I_s 1. ---- *)
Fixpoint M_I_composed (j : nat) (i0 : Int) : Int :=
  match j with
  | 0 => i0
  | S j' => M_I (S j') (M_I_composed j' i0)
  end.

(* ---- Main induction: for every hop j (1<=j<=n), that hop's own local
   intent I_d j equals the ORIGIN intent (I_s 1) mapped through the fully
   composed chain M_I^{0:j} -- not merely hop j's own local M_I applied
   to some disconnected input. This is the genuinely new content beyond
   restating per-hop Theorem 1: it is what makes hop n's local conclusion
   a statement ABOUT THE WHOLE PATH from B_0, not an isolated fact about
   hop n in a vacuum. Proved by induction, using continuity to link each
   hop's local input back to the previous hop's output. ---- *)
Theorem multihop_intent_composition :
  forall j, 1 <= j -> j <= n -> I_d j = M_I_composed j (I_s 1).
Proof.
  induction j as [|j' IH].
  - intros H1 _. inversion H1.
  - intros _ Hjn.
    destruct (Nat.eq_dec j' 0) as [Hz | Hnz].
    + subst j'. simpl. apply I_d_is_MI.
    + assert (Hj'1 : 1 <= j') by lia.
      assert (Hj'n : j' < n) by lia.
      assert (Hj'len : j' <= n) by lia.
      assert (IHj' : I_d j' = M_I_composed j' (I_s 1)) by (apply IH; assumption).
      simpl.
      rewrite <- IHj'.
      rewrite <- (continuity_intent j' Hj'1 Hj'n).
      apply I_d_is_MI.
Qed.

(* ---- Scope propagation: the origin's certified-scope membership,
   together with compatible-scope inclusion at every intermediate hop,
   propagates all the way to hop n -- proved by induction (using
   continuity_intent to relate hop j's own intent to hop (j-1)'s mapped
   output, exactly as multihop_intent_composition does), NOT assumed.
   This is what makes the compatible-scope premise
   ($M_I^{j-1:j}(\Omega_j)\subseteq\Omega_{j+1}$) genuinely load-bearing:
   without scope_compatible, this induction does not go through. ---- *)
Theorem multihop_scope_propagation :
  forall j, 1 <= j -> j <= n -> Omega j (I_s j).
Proof.
  induction j as [|j' IH].
  - intros H1 _. inversion H1.
  - intros _ Hjn.
    destruct (Nat.eq_dec j' 0) as [Hz | Hnz].
    + subst j'. simpl. exact scope_member_origin.
    + assert (Hj'1 : 1 <= j') by lia.
      assert (Hj'n : j' < n) by lia.
      assert (Hj'len : j' <= n) by lia.
      assert (IHj' : Omega j' (I_s j')) by (apply IH; assumption).
      rewrite (continuity_intent j' Hj'1 Hj'n).
      rewrite (I_d_is_MI j').
      apply (scope_compatible j' Hj'1 Hj'n).
      exact IHj'.
Qed.

(* ---- Obligation propagation: obligation preservation at the origin,
   together with adjacent-hop field preservation at every intermediate
   hop, propagates all the way to hop n -- proved by induction, NOT
   assumed at every hop directly. This is what makes
   fields_preserved_holds genuinely load-bearing, and it is the
   mechanized counterpart of what Ab11 shows empirically: the premise
   Ab11 demonstrates is necessary on-chain (dropping it lets 22/22
   injected-loss chains through) is the SAME premise this induction
   cannot do without -- removing fields_preserved_holds breaks this
   proof, not just the on-chain counterexample. ---- *)
Theorem multihop_obligations_propagation :
  forall j, 1 <= j -> j <= n -> ObligationsPreserved j.
Proof.
  induction j as [|j' IH].
  - intros H1 _. inversion H1.
  - intros _ Hjn.
    destruct (Nat.eq_dec j' 0) as [Hz | Hnz].
    + subst j'. simpl. exact obligations_hold_origin.
    + assert (Hj'1 : 1 <= j') by lia.
      assert (Hj'n : j' < n) by lia.
      assert (Hj'len : j' <= n) by lia.
      assert (IHj' : ObligationsPreserved j') by (apply IH; assumption).
      apply (obligations_propagate j' Hj'1 Hj'n IHj').
      apply (fields_preserved_holds j' Hj'1 Hj'n).
Qed.

(* ---- Main theorem: the full n-hop path is portable to B_n. T_d_auth
   holds at the final hop (re-derived from that hop's own guard
   apparatus, exactly as Theorem 1 does for a single hop); the final
   hop's intent is exactly the origin intent mapped through the fully
   composed chain; the final hop's intent lies within its own certified
   scope, propagated from the origin through every compatible-scope
   inclusion; obligation preservation likewise propagates from the
   origin through every adjacent-hop field-preservation step (not
   assumed at the final hop directly -- this is the premise Ab11 shows
   empirically necessary, and it is formally necessary here too); and
   assurance/temporal validity hold at the final hop. This is the
   mechanized form of Theorem 4's conclusion "the decision is portable
   to B_n": the per-hop, local premises are jointly sufficient for this
   global, n-hop-wide statement, derived here, not assumed as a separate
   "composed" postulate. ---- *)
Theorem multihop_composition_soundness :
  n >= 1 ->
  T_d_auth_at n /\ I_d n = M_I_composed n (I_s 1) /\ Omega n (I_s n)
  /\ ObligationsPreserved n /\ AssuranceNotAmplified n /\ TemporalValid n.
Proof.
  intro Hn.
  split; [| split; [| split; [| split; [| split]]]].
  - apply t_d_auth_at_sound.
  - apply multihop_intent_composition; lia.
  - apply multihop_scope_propagation; lia.
  - apply multihop_obligations_propagation; lia.
  - apply assurance_hold; lia.
  - apply temporal_hold; lia.
Qed.

End MultiHop.

Print Assumptions multihop_composition_soundness.
Print Assumptions multihop_intent_composition.
Print Assumptions multihop_scope_propagation.
