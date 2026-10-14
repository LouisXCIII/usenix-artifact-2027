package authz.rolemap

# Source-to-destination role translation table.
# Each source_role (left column of the spec's table) maps to its
# destination cap. Two different source roles may map to the same cap
# (gerente_senior and diretor both -> "director" / 2,000,000); that is
# intentional per the spec and is simply two entries with the same value.
role_caps := {
	"gerente": 500000,
	"gerente_senior": 2000000,
	"funcionario": 10000,
	"diretor": 2000000,
	"supervisor": 500000,
}

default allow := false

# GRANT iff:
#   1. source_role is one of the five known source-side strings (lookup
#      into role_caps succeeds), AND
#   2. amount is a positive integer (amount > 0), AND
#   3. amount does not exceed that role's destination cap.
#
# If input.source_role is not a key in role_caps, `role_caps[input.source_role]`
# is undefined, the rule body fails, and `allow` falls back to the
# `default allow := false` above -> DENY. No fallback/default cap is ever
# substituted for an unrecognized role.
allow {
	input.amount > 0
	cap := role_caps[input.source_role]
	input.amount <= cap
}

# String mirror of the decision, for convenience / human-readable output.
decision := "GRANT" {
	allow
} else := "DENY" {
	true
}
