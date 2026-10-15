package destination_authz

# -----------------------------------------------------------------------------
# Destination Authorization Policy
#
# Inputs (all already-authenticated by an upstream verification layer; this
# policy does not verify signatures, proofs, or ledger state):
#   input.role   (string)  one of: clerk, manager, director, gerente, gerente_senior
#   input.amount (number)  requested transfer amount in USD, positive integer
#   input.tier   (number)  achieved risk tier for this transfer, 0-3
#
# Business rule:
#   - tier 0            -> always DENY
#   - tier 1            -> amount <= 10%  of role's full cap
#   - tier 2            -> amount <= 50%  of role's full cap
#   - tier 3            -> amount <= 100% of role's full cap
#   - GRANT iff tier >= 1 AND amount <= cap(role) * fraction(tier)
# -----------------------------------------------------------------------------

# Full approved transfer cap per role, in USD.
caps = {
	"clerk": 10000,
	"manager": 500000,
	"director": 2000000,
	"gerente": 500000,
	"gerente_senior": 2000000,
}

# Allowed fraction of the role's cap per achieved risk tier, expressed as an
# exact integer numerator/denominator pair (avoids floating-point rounding).
# Tier 0 is intentionally absent: any lookup against it is undefined, which
# makes `allow` undefined/false for tier 0, matching "DENY outright".
tier_fraction_num = {1: 1, 2: 1, 3: 1}

tier_fraction_den = {1: 10, 2: 2, 3: 1}

# allow is true exactly when the transfer should be GRANTed.
default allow = false

allow {
	input.tier >= 1
	input.tier <= 3
	cap := caps[input.role]
	num := tier_fraction_num[input.tier]
	den := tier_fraction_den[input.tier]

	# amount <= cap * (num/den)  <=>  amount * den <= cap * num
	input.amount * den <= cap * num
}

# decision is a human-readable "GRANT"/"DENY" mirror of `allow`, for callers
# that prefer a string result over a boolean.
default decision = "DENY"

decision = "GRANT" {
	allow
}
