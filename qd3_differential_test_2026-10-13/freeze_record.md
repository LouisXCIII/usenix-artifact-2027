# Freeze record — `independent_oblcovered.py`

- SHA-256: `0b34d92652053dc7497f18a97be3e5184a1269d7bfa62f316b094364bc3fe3a6`
- Frozen (UTC): `2026-10-13T04:22:58Z`
- Written by: a fresh `general-purpose` agent spawned with ONLY the prose spec below (no
  chaincode source, no file access to `~/fabric-network`, no web search), per the
  independence discipline used for the earlier S3/Cedar pilot. The agent is an LLM, same
  model family as the rest of this project's tooling — disclosed explicitly, not two
  independent human experts.
- This hash/timestamp was computed and recorded BEFORE running `independent_oblcovered.py`
  against any real or synthetic case data, per the plan's "freeze before comparing" rule.

## Strength ordering the agent chose (not stated in the paper's prose, inferred by the agent)

`empty(0) < notification(1) < audit(2) < dual_approval(3) <= combined(4)`, with its own
stated rationale (preventive dual-approval ranked above detective audit). The agent explicitly
flagged this as the one point where a different person given the same prose might disagree
(e.g. ranking an audit trail above a single extra approver).

## Three ambiguities the agent resolved (see file header comment for full text)

1. `obligationProfile == "empty"` → vacuously covered, no discharge/witness checks needed.
2. Non-empty profile with no discharge claimed at all (`obligationDischargeRef == ""`) →
   treated as not covered (same as never-discharged), not as "not applicable."
3. `enforcementWitnessVerified` required `== true` for genuine coverage, even though the
   paper is explicit this flag is unverified/caller-trusted in the real system (the open S7
   gap) — the agent reasoned the task asked for "genuinely, fully covered," not "what the
   deliberately incomplete real system currently accepts."
