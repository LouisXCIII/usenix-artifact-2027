# risk_tier_policy.rego -- a faithful re-implementation, in OPA/Rego, of the role/amount/tier
# gate in the REAL chaincode entry point SubmitDecisionRiskTierVerified
# (~/fabric-network/authcc/chaincode/risk_tier.go, function tierFromFloorSignal +
# SubmitDecisionRiskTierVerified). Same constants (tierT1Floor/T2/T3, tierFractionBps,
# RoleThreshold), same decision logic. The ONE thing it does NOT do, because Rego's builtin
# function set has no such primitive, is verify that input.declaredFloor was actually produced
# by a genuine Groth16 proof over bn254 (the pairing check in zkverify.go/verifyGroth16, done in
# Go via the gnark-crypto library) -- it treats input.declaredFloor as an already-trustworthy
# attribute, exactly the way every Cedar/OPA authorization model is designed to consume its
# inputs (policy evaluation over given facts, not cryptographic verification of how those facts
# were established).
package risktier

import future.keywords.if

tier_t1_floor := 54445422
tier_t2_floor := 67108864
tier_t3_floor := 83886080

tier_fraction_bps := {0: 0, 1: 1000, 2: 5000, 3: 10000}

role_threshold := {
	"clerk": 10000,
	"manager": 500000,
	"director": 2000000,
	"gerente": 500000,
	"gerente_senior": 2000000,
}

# Exactly tierFromFloorSignal's logic, ported line-for-line.
achieved_tier := 3 if input.declaredFloor >= tier_t3_floor
else := 2 if input.declaredFloor >= tier_t2_floor
else := 1 if input.declaredFloor >= tier_t1_floor
else := 0

fraction := tier_fraction_bps[achieved_tier]

adjusted_cap := cap * fraction / 10000 if cap := role_threshold[input.role]

decision := "SAFE" if {
	achieved_tier > 0
	input.amount <= adjusted_cap
} else := "REJECT"
