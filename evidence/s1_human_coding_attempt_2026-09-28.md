# S1 -- Human coding attempt (author-coded), 2026-09-28: result recorded, not used

Status: CLOSED by the author's decision ("save it in evidence and move on"). Recorded exactly as it happened. It is not used to
validate or to overturn the AI codings.

## What was done
One author (not a specialist in the systems; knows the paper's hypothesis, so NOT an independent coder) coded the 22-row sheet in the
terminal form (`s1_verification/human_coding_kit/code_interactive.py`) after reading the frozen rubric. 21 of 22 rows were filled
(R25 not filled; reason not given). Evidence and note fields were mostly left empty (evidence given for R2 only; no notes).
Raw files: `s1_verification/human_coding_kit/human1_ORIGINAL_submission_2026-09-28.csv` (read-only, sha256 prefix ed6bc28539804e3a)
and `human1_score_output_2026-09-28.txt`.

## Result
Agreement with the AI coders is at chance level.
| Comparison | D2 | D3 | D4 | D5 | pooled kappa |
|---|---|---|---|---|---|
| human vs coder 1 (raw) | 0.29 | 0.29 | 0.14 | 0.24 | -0.01 |
| human vs coder 2 (raw) | 0.24 | 0.24 | 0.14 | 0.24 | -0.05 |
| human vs adjudicated (raw) | 0.29 | 0.24 | 0.14 | 0.24 | -0.05 |
The human disagreed with BOTH AI coders on 61 cells (list in the score output).

## Pattern (exploratory, post hoc, n = 21 rows; not a finding)
| Cell | Letters the human used | Letters the AIs used |
|---|---|---|
| D2 | P 11, Y 9, U 1, never N | Y 11, P 5-7, N 2-3 |
| D3 | Y 9, P 6, U 3, N 3 | N 13-14, Y 3-5, P 3 |
| D4 | P 7, N 5, U 5, Y 4 | N 16-17, Y 2, P 2 |
| D5 | Y 9, N 7, P 3, U 2 | N 11, Y 4-6, P 2-5 |
On the cells where the two AI coders agree, swapping Y and P in the human's D2 raises raw agreement from 0.29 to 0.71 (n = 14); swapping Y and N
in D3 and D5 raises it to about 0.5 (n = 19, 15); no swap explains D4 (0.25). The swaps were chosen after seeing the data and are only a diagnostic:
they suggest the human read the letters under a different meaning than the rubric intended (for example "the concern exists" or "the mechanism could exist
in the ecosystem" instead of "this file enforces it"). The three diagnostic questions put to the author (what Y meant for D3-D5, whether `proto` rows were coded
from the listed file only, why R25 was skipped) were not answered because the author closed the exercise.

## What it means, and what it does not
- It does NOT show that the AI codings are wrong, and it does NOT show that the human is wrong.
- It shows that the rubric, as written and as presented in the tool, is not reliably usable by a human who reads it once. That is a real limitation of
  the survey's coding scheme and must be disclosed if the survey is reported.
- It also removes the one piece of reassurance that could have supported the AI codings: the AI-AI agreement (pooled kappa 0.63) may reflect a shared
  interpretive habit rather than validity.
- The survey's core evidence is unaffected: the file:line citations, the deployed-code checks and the production-configuration reads can be verified by
  anyone without relying on the letter codes. What is weakened is the aggregate count (20/22, 18/22), which depends on the coding scheme.
- If the survey is reported, the safest wording is qualitative (a table of what each receiver enforces, with citations) plus the counts labelled "by the
  authors' rubric, agreement between coders limited (see this note)".

## If this is ever revisited
Revise the rubric with worked examples and a decision tree; have a colleague who is not an author code after a short calibration on excluded rows (R21,
Weaver); report that person's agreement, not this one's.
