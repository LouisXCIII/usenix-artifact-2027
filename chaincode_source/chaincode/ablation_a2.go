// ablation_a2.go -- E8 limitation fix: A2 ("remove R_P/M_T relational semantic
// preservation, i.e. the sem_covered check") was previously only exercised offline,
// against the formal E1 corpus (e8_ablation.py), with no real on-chain analog -- unlike
// A10/A11, which already have one (a10_tac_binding_ablation.js, X5a). For the "role"
// family, sem_covered IS exactly classify()'s amount<=threshold non-widening check
// (confirmed identical in e8_ablation.py's family=="role" branch). This file adds a
// real, additive, ablated entry point that skips ONLY that one check, so the real
// chaincode -- not a Python reimplementation -- demonstrates A2's consequence on-chain.
package chaincode

import (
	"encoding/json"
	"fmt"

	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
)

// SubmitDecisionNoSemCheck is SubmitDecision with exactly one change: the
// amount<=threshold (non-widening / sem_covered) check is skipped -- any amount is
// treated as within policy once semBound/evAssuranceOK/role are satisfied. This is the
// real-chaincode instance of E8's A2 ablation, mirroring the offline classifier's
// `ablation == "A2": sem_covered = True` branch (e8_ablation.py) exactly.
func (c *AuthContract) SubmitDecisionNoSemCheck(ctx contractapi.TransactionContextInterface,
	decisionID string, sourceDomain string, role string, amount int64,
	sourceLimit int64, semBound bool, evAssuranceOK bool) (string, error) {

	exists, err := c.decisionExists(ctx, decisionID)
	if err != nil {
		return "", err
	}
	if exists {
		return "", fmt.Errorf("decision %s already recorded (replay rejected)", decisionID)
	}

	threshold, known := RoleThreshold[role]
	if !known {
		threshold = 0
	}

	class := classifyA2Ablated(semBound, evAssuranceOK, known)

	rec := PortableDecisionRecord{
		DecisionID:    decisionID,
		SourceDomain:  sourceDomain,
		Role:          role,
		Amount:        amount,
		SourceLimit:   sourceLimit,
		SemBound:      semBound,
		EvAssuranceOK: evAssuranceOK,
		TransferClass: class,
		DestThreshold: threshold,
	}

	recJSON, err := json.Marshal(rec)
	if err != nil {
		return "", err
	}
	if err := ctx.GetStub().PutState(decisionID, recJSON); err != nil {
		return "", fmt.Errorf("failed to write state: %v", err)
	}
	return class, nil
}

// classifyA2Ablated is classify() with the amount<=threshold check forced true --
// exactly A2's definition, nothing else changed (Theorem T0's semBound/evAssuranceOK
// ordering and the roleKnown->REEVALUATE branch are untouched, isolating this one
// conjunct the same way e8_ablation.py's classify_ablation() does).
func classifyA2Ablated(semBound bool, evAssuranceOK bool, roleKnown bool) string {
	if !semBound {
		return "REJECT"
	}
	if !evAssuranceOK {
		return "REJECT"
	}
	if !roleKnown {
		return "REEVALUATE"
	}
	return "SAFE" // A2 ablated: amount<=threshold not checked, always treated as satisfied
}
