# Data-leakage investigation for the risk-tier FN-rate-by-tier calibration (2026-10-11)

Question: does the pinned ML scanner model (`fold0_epoch14_f10.9661.pt`, used throughout this
project and by `fn_rate_by_tier_v2.py`) have any real overlap between its own training data and
the 253-contract external corpus (SmartBugs-curated, WeakAccessMod) used to measure its
false-negative rate by tier?

## Steps and findings

1. `grep -rl "weakaccess" ~/Desktop/projectAI/buildVocab/` found the string inside
   `trainning/dataset/results*/solidity_vulnerabilities.json` -- the original trigger for
   flagging this as a risk.
2. The `weakaccess` corpus rows in `s5_corpus_features.json` point to
   `~/Desktop/projectAI/buildVocab/SourceCodesCleaned/WeakAccessMod/` directly -- confirming
   it is the same physical directory the training-adjacent pipeline references by name.
3. `grep` of `buildVocab/trainning/dataset/*.py` for `WeakAccessMod`/`fold`/`train_test_split`
   found `dedup_and_group_split.py`, a `GroupShuffleSplit`-based train/val/test splitter that
   explicitly handles `WeakAccessMod` as a rare class (~900 groups) during stratification --
   real evidence THIS pipeline uses WeakAccessMod.
4. Timestamp check: `find ... -printf '%T@ ...'` on every file in
   `SourceCodesCleaned/WeakAccessMod/` and every `*.pt` output of this pipeline
   (`outpt_v2_final/`, `outpt_v3_final/`, `outputs_v2_allfixes/`) -- all dated August 2026
   (earliest 2026-08-13). The pinned model file `fold0_epoch14_f10.9661.pt` is dated
   **2026-02-05**, six months earlier. This pipeline's outputs cannot be its training source.
5. `find / -iname "*fold0_epoch14*"` located the actual checkpoint at
   `~/pqc-hardhat/gateway/fold0_epoch14_f10.9661.pt` (2026-02-05) and a differently-scored
   sibling `~/Desktop/projectAI/fold0_epoch14_f10.9522.pt` (not the pinned one).
6. `grep -rl "0.9661" ... *.log` across the filesystem found
   `[unrelated path on disk]/lstm/logs/enhanced_lstm_20260203_225232.log` -- a completely
   separate project tree (a different user home, an LSTM training pipeline).
7. Read that log directly: it trains `BiLSTM2L_H256_attention_TA` over 5 folds, loads data from
   `[unrelated path on disk]/finalSlicing/test/outputs/processed_data.pt` (952,126 samples, 12
   classes), and saves `fold0_epoch14_f10.9661.pt` at **2026-02-05 13:28:02** -- an exact match
   to the pinned model's filename, F1 score, and file timestamp. This is the real training run.
8. Checked all other `enhanced_lstm_*.log` files in the same directory (11 total, Feb-2026
   through Mar-2026 runs): every one loads from the same
   `finalSlicing/test/outputs/processed_data.pt` path; `grep -il "smartbugs\|weakaccess"`
   across all of them returned zero matches.
9. `find / -iname "*finalSlicing*"` found nothing -- the actual training corpus lived on a
   separate (Windows) machine and was never transferred to this filesystem, so no direct
   file-hash comparison against the 253-contract evaluation corpus was possible.

## Conclusion

The specific leakage path originally flagged (the `WeakAccessMod` string appearing in a
training-adjacent dataset file) does not hold up: that file belongs to a different, later,
apparently unconnected dataset-preparation effort, not the pipeline that actually produced the
pinned model. No evidence of smartbugs/weakaccess involvement was found in any of the 11 real
training-run logs. This is not a complete proof of no leakage -- the real training corpus was
never directly inspectable -- but the concrete risk raised is not supported by the evidence
found. Written up in `evidence/rp_checker_tier_2026-10-11.md`'s Part 2b.
