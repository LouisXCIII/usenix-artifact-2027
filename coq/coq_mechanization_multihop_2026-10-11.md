# Coq mechanization of Theorem 4 (Conditional multi-hop composition), 2026-10-11

Per the user's explicit priority call ("việc có giá trị nhất là cơ chế hóa Theorem 4
(multi-hop) bằng Coq") after a review identified Theorem 4 as the paper's most
content-rich result (real on-chain evidence, Ab11) yet the only one of the four theorems
not yet mechanized, pen-and-paper only. File: `ADPMultiHopComposition.v` (~300 lines),
`evidence/coq/`, compiles clean with `coqc` (Coq 8.18.0, same toolchain as
`ADPRelationalSoundness.v`'s existing Theorem 1 mechanization). **Revised once after a
second review found the first version's `fields_preserved_holds` premise -- the exact
premise Ab11 shows empirically necessary -- was vacuous in the mechanization; fixed, see
"A second gap, more consequential than the first" below.**

## What is mechanized

Theorem 4 ("Conditional multi-hop composition", `\label{thm:multihop-composition}`), at
the same abstraction level as Theorem 1's existing mechanization: a path of $n$ hops,
one shared state type `Dom` and intent type `Int` threaded across every domain
$B_0,\ldots,B_n$ (since "destination state of hop $j$" and "source state of hop $j{+}1$"
are the same object, not two separate types).

Four theorems, in order:

1. **`multihop_intent_composition`**: for every hop $j$ ($1\le j\le n$), that hop's own
   local intent `I_d j` equals the path's ORIGIN intent (`I_s 1`) mapped through the
   fully composed chain `M_I_composed j`. Proved by induction, using `continuity_intent`
   (hop $j$'s mapped output becomes hop $j{+}1$'s input) to link each hop's local input
   back to the previous hop's output.
2. **`multihop_scope_propagation`**: for every hop $j$, that hop's own intent lies within
   its own certified scope (`Omega j (I_s j)`), propagated from the origin's own scope
   membership through every intermediate hop's compatible-scope inclusion
   ($M_I^{j-1:j}(\Omega_j)\subseteq\Omega_{j+1}$, `scope_compatible`). Proved by
   induction, structurally parallel to (1).
3. **`multihop_obligations_propagation`** (added in the revision): for every hop $j$,
   `ObligationsPreserved j` holds, propagated from the origin's own obligation
   preservation (`obligations_hold_origin`) through every intermediate hop's
   adjacent-hop field preservation (`FieldsPreserved j` / `fields_preserved_holds`,
   consumed via `obligations_propagate`). Proved by induction, structurally parallel to
   (1) and (2) -- this is the theorem that makes `fields_preserved_holds` genuinely
   load-bearing; see below.
4. **`multihop_composition_soundness`** (the main theorem): for $n\ge1$, at the final hop
   $n$: $T_d^{\mathrm{auth}}$ holds (re-derived from hop $n$'s own guard apparatus,
   exactly as Theorem 1 does for a single hop); the final intent equals the
   fully-composed mapping of the origin intent (1); the final intent lies within its own
   certified scope, propagated from the origin (2); `ObligationsPreserved n` holds,
   propagated from the origin through field preservation (3); and
   `AssuranceNotAmplified n` / `TemporalValid n` (Theorem 3's conclusion and
   freshness/replay/finality) hold, still taken directly as per-hop hypotheses. This is
   the mechanized form of Theorem 4's conclusion "the decision is portable to $B_n$."

`Print Assumptions` on the theorems reports "Closed under the global context" -- no
axioms beyond Coq's own core logic (no `Classical`, no `Admitted`, no `Axiom`) anywhere
in the file, same standard as the existing Theorem 1 mechanization.

## A second gap, more consequential than the first, caught by external review

After the first version (3 theorems, `scope_compatible` made load-bearing, mechanical
necessity check run: 17/19 load-bearing, 2 disclosed exceptions), a second review raised
a sharper objection than "two premises are unused": **the specific premise left unused,
`fields_preserved_holds`, is exactly the premise the paper's own on-chain experiment
(Ab11) demonstrates is empirically necessary** -- removing it lets all 22 injected-loss
chains through. A formally-necessary/empirically-necessary premise that diverges like
this can only happen if the mechanized conclusion is weaker than what the paper states:
the review's diagnosis was that `ObligationsPreserved` had been assumed directly at
*every* hop (`obligations_hold : forall j, ... -> ObligationsPreserved j`), so the final
theorem's `ObligationsPreserved n` conjunct was discharged by citing that hypothesis
directly at $j=n$ -- trivially, with no path through `fields_preserved_holds` at all.
Checked and confirmed exactly right: `multihop_composition_soundness`'s original proof
line for that conjunct was `apply obligations_hold; lia`, no induction involved.

**Fixed the same way `scope_compatible`'s gap was fixed** (the review named this
analogy explicitly): `obligations_hold` was narrowed to only the origin hop
(`obligations_hold_origin : ObligationsPreserved 1`); a new hypothesis
`obligations_propagate : forall j, ... -> ObligationsPreserved j -> FieldsPreserved j ->
ObligationsPreserved (S j)` encodes the paper's own proof-sketch language ("the
adjacent-hop field-preservation premise closes the one gap that per-hop obligation
preservation alone does not cover"); and `multihop_obligations_propagation` proves
`ObligationsPreserved n` by induction, consuming `fields_preserved_holds` at every step.
Also removed `continuity_state` (a state-side continuity hypothesis the first version's
own necessity check had already flagged as unused): the theorem's conclusion makes no
claim about the final hop's state relative to the origin's, only its intent, so unlike
`fields_preserved_holds` there was no empirical counterpart making it worth fixing --
removed rather than kept as dead weight, per the review's own stated fallback for this
specific premise.

**Re-ran the mechanical necessity check after the fix: 19/19 load-bearing, 0 redundant**
(up from 17/19) -- every hypothesis across both files, with no disclosed exceptions
remaining. The sentence this licenses, close to verbatim from the review that requested
it: *the premise Ab11 shows is empirically necessary is also formally necessary here:
removing it breaks the mechanized proof.*

## A real gap caught and fixed before finalizing, in the first version

The first working draft had `scope_member` asserting `Omega j (I_s j)` for *every* hop
$j$ directly, as a raw hypothesis -- which made `scope_compatible` (the compatible-scope
premise the theorem explicitly names) completely unused. Changed `scope_member` to
assert only the *origin* hop's membership (`scope_member_origin`), and added
`multihop_scope_propagation` as a genuine induction that does not go through without
`scope_compatible`.

## Two real Coq bugs found and fixed while compiling the first version (disclosed, not hidden)

1. `multihop_composition_soundness`'s first draft used `repeat split` on a multi-way
   conjunction; `repeat split` additionally unfolded `T_d_auth_at`'s own internal `/\ `
   (a `Definition`, not opaque), shifting every subsequent proof bullet by one position.
   Fixed with an explicit, fully-bracketed `split` chain.
2. `multihop_scope_propagation`'s first draft used `rewrite <-` in the wrong direction,
   then a second gap (`I_d j'` not syntactically recognized as `M_I j' (I_s j')`) needed
   an explicit `rewrite (I_d_is_MI j')` step. Both fixed before the first clean compile
   was treated as final.

## Mechanical hypothesis-necessity check

`check_necessity.py`: for every named `Hypothesis` across both `.v` files, comments out
that one hypothesis, recompiles the file standalone with `coqc`, and records whether the
proof still succeeds without it (success = redundant = bad; failure = load-bearing =
expected). Run twice: once against the first version (17/19 load-bearing, 2 disclosed
exceptions -- see above), and again after the `fields_preserved_holds` fix
(**19/19 load-bearing, 0 redundant**, current state, 4 in `ADPRelationalSoundness.v` + 15
in `ADPMultiHopComposition.v`). Full current per-hypothesis results:
`necessity_check_results.json`.

This is the mechanized-proof analog of the paper's own ablation results (Ab0-Ab11):
every premise named in both mechanized theorems is load-bearing, verified mechanically
rather than asserted, with the one premise that has a direct empirical counterpart
(Ab11/`fields_preserved_holds`) now agreeing both formally and empirically.

## Scope, stated honestly (the file's own header makes the same disclosures)

- **Theorem 2 and Theorem 3 are NOT separately re-mechanized here** -- their
  *conclusions* are taken as per-hop hypotheses (`ObligationsPreserved`'s propagation is
  now proved, not assumed, as described above; `AssuranceNotAmplified` remains a direct
  per-hop hypothesis), exactly as A1-A13 are assumed rather than re-derived in
  `ADPRelationalSoundness.v`.
- **Freshness/replay/finality** (`TemporalValid j`) is an abstract per-hop `Prop`
  hypothesis. Important: the paper's body statement of Theorem 4 specifies freshness is
  checked against the *origin timestamp* $t_0$, not each hop's local receipt time (a
  distinction the paper's own on-chain origin-timestamp experiment, S6, shows matters --
  per-hop-local freshness does not imply end-to-end freshness). **$t_0$ itself is NOT
  represented in this file** -- `TemporalValid` carries no timestamp structure. The
  manuscript's wording was narrowed accordingly (Abstract/Contributions/Conclusion all
  say "the composition argument is mechanized," not that the whole theorem including its
  freshness mechanism is).

## Why this is a real addition, not busywork

Table 8 (`tab:claim-evidence`) previously listed Theorem 4 as pen-and-paper only. Its
composition argument -- including, after the fix above, the specific premise Ab11
demonstrates matters empirically -- is now machine-checked and confirmed load-bearing by
an independent mechanical sweep, not merely hand-verified or asserted.

## Reproduction

```
cd evidence/coq
coqc ADPRelationalSoundness.v
coqc ADPMultiHopComposition.v
python3 check_necessity.py   # mechanical hypothesis-necessity check, ~1 minute
```
Exit code 0 for both `coqc` runs; `check_necessity.py` reports 19/19 load-bearing, 0
redundant.
