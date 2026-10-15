package authz.obligation

# -----------------------------------------------------------------------
# Role dollar caps (hard upper limit regardless of evidence)
# -----------------------------------------------------------------------
cap_map := {
	"clerk": 10000,
	"manager": 500000,
	"director": 2000000,
	"gerente": 500000,
	"gerente_senior": 2000000,
}

# -----------------------------------------------------------------------
# Half-of-cap thresholds (precomputed, as given in the spec table)
# -----------------------------------------------------------------------
half_cap_map := {
	"clerk": 5000,
	"manager": 250000,
	"director": 1000000,
	"gerente": 250000,
	"gerente_senior": 1000000,
}

# -----------------------------------------------------------------------
# Evidence strength levels
# -----------------------------------------------------------------------
strength_map := {
	"none": 0,
	"notification": 1,
	"audit": 2,
	"dual_approval": 3,
	"combined": 4,
}

# strength(e): looks up the strength of a given evidence_type string.
# Any evidence_type not present in strength_map (including "none", which is
# listed explicitly but also happens to equal the default) resolves to 0.
strength(e) = s {
	s := strength_map[e]
} else = 0 {
	true
}

# -----------------------------------------------------------------------
# allow: GRANT/DENY decision
# -----------------------------------------------------------------------
default allow = false

# Case 1: amount is at or below half of the role's cap -> no extra
# evidence required, subject to the role cap.
allow {
	input.amount > 0
	cap := cap_map[input.role]
	input.amount <= cap
	half := half_cap_map[input.role]
	input.amount <= half
}

# Case 2: amount is strictly above half of the role's cap (but still at or
# below the full cap) -> requires evidence strength >= 3.
allow {
	input.amount > 0
	cap := cap_map[input.role]
	input.amount <= cap
	half := half_cap_map[input.role]
	input.amount > half
	strength(input.evidence_type) >= 3
}

# -----------------------------------------------------------------------
# decision: string mirror of allow, for callers that prefer GRANT/DENY text
# -----------------------------------------------------------------------
decision = "GRANT" {
	allow
}

decision = "DENY" {
	not allow
}
