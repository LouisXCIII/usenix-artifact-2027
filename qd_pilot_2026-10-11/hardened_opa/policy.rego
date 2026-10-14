# hardened_opa/policy.rego -- the "hardened" arm (full-knowledge author = me), implementing
# spec_v1.md's rule exactly for all 5 roles. Upper-bound baseline, not given to the fresh
# context-free naive author.
package hardened

import future.keywords.if

role_cap := {
	"clerk": 10000,
	"manager": 500000,
	"director": 2000000,
	"gerente": 500000,
	"gerente_senior": 2000000,
}

tier_fraction_bps := {0: 0, 1: 1000, 2: 5000, 3: 10000}

decision := "GRANT" if {
	input.tier >= 1
	cap := role_cap[input.role]
	input.amount * 10000 <= cap * tier_fraction_bps[input.tier]
} else := "DENY"
