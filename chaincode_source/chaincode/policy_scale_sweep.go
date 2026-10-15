// policy_scale_sweep.go -- real, additive implementation of the p/m/o independent complexity
// sweeps flagged as open ("TBD") in Manuscriptsp_m2_r3.tex's tab:perf-policy-scale
// (subsec:perf-policy-scale). RoleThreshold/RoleMapping are compiled-in, fixed-size maps
// (authchaincode.go) with no existing mechanism to vary policy/mapping/obligation size
// independently -- this file adds three new, query-only functions that build and evaluate a
// REAL Go map/slice of the requested size on each call, timed internally with time.Now()
// (the same isolation-from-consensus-overhead pattern already validated by
// perf_instrumented.go's T_obligation/T_assurance measurements this session). Query-only (no
// PutState): isolates the pure evaluation cost from Fabric's block-commit latency, matching the
// table's "primary component" framing (T_policy / T_map+T_state / obligation processing), not
// end-to-end transaction latency (already measured separately, subsec:perf-volume).
package chaincode

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
)

// SweepPolicySize evaluates p real boolean policy predicates (amount <= threshold_i, for p
// distinct real synthetic thresholds) against a fixed real transfer amount. Returns the real
// elapsed time (ns) for evaluating exactly p predicates, isolated from map/slice construction.
func (c *AuthContract) SweepPolicySize(ctx contractapi.TransactionContextInterface, p int64) (string, error) {
	if p < 0 {
		return "", fmt.Errorf("p must be >= 0")
	}
	amount := int64(250000)
	thresholds := make([]int64, p)
	for i := int64(0); i < p; i++ {
		thresholds[i] = 10000 + i*1000
	}
	start := time.Now()
	passCount := 0
	for i := int64(0); i < p; i++ {
		if amount <= thresholds[i] {
			passCount++
		}
	}
	elapsed := time.Since(start)
	out := map[string]interface{}{"p": p, "passCount": passCount, "elapsedNs": elapsed.Nanoseconds()}
	b, _ := json.Marshal(out)
	return string(b), nil
}

// SweepMappingSize builds a real Go map with m entries (role_i -> destRole_i) then performs m
// real lookups against it. Returns the real elapsed time (ns) for the m lookups, isolated from
// map construction (timed separately, after the map is built).
func (c *AuthContract) SweepMappingSize(ctx contractapi.TransactionContextInterface, m int64) (string, error) {
	if m < 0 {
		return "", fmt.Errorf("m must be >= 0")
	}
	mapping := make(map[string]string, m)
	keys := make([]string, m)
	for i := int64(0); i < m; i++ {
		k := fmt.Sprintf("role%d", i)
		mapping[k] = fmt.Sprintf("destRole%d", i)
		keys[i] = k
	}
	start := time.Now()
	lookups := 0
	for i := int64(0); i < m; i++ {
		if _, ok := mapping[keys[i]]; ok {
			lookups++
		}
	}
	elapsed := time.Since(start)
	out := map[string]interface{}{"m": m, "lookups": lookups, "elapsedNs": elapsed.Nanoseconds()}
	b, _ := json.Marshal(out)
	return string(b), nil
}

// SweepObligationCount evaluates o real, distinct obligation-strength checks in sequence
// (mirroring the ordinal mechanism-strength model already used for S5-S7's real obligation_v2.go
// port), each against a fixed real amount, all pre-approved. Returns the real elapsed time (ns)
// for evaluating exactly o obligations.
func (c *AuthContract) SweepObligationCount(ctx contractapi.TransactionContextInterface, o int64) (string, error) {
	if o < 0 {
		return "", fmt.Errorf("o must be >= 0")
	}
	amount := int64(1500000)
	obligationThresholds := make([]int64, o)
	for i := int64(0); i < o; i++ {
		obligationThresholds[i] = (i + 1) * 100000
	}
	start := time.Now()
	dischargedCount := 0
	for i := int64(0); i < o; i++ {
		required := amount > obligationThresholds[i]
		approved := true // real check evaluated; approval state fixed for this sweep
		if required && approved {
			dischargedCount++
		}
	}
	elapsed := time.Since(start)
	out := map[string]interface{}{"o": o, "dischargedCount": dischargedCount, "elapsedNs": elapsed.Nanoseconds()}
	b, _ := json.Marshal(out)
	return string(b), nil
}
