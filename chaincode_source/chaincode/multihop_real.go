// multihop_real.go -- E9 limitation fix: the project's only fully-on-chain multi-hop evidence was
// X5a/X5b at h=2 (evidence/e9_multihop_extension_2026-09-23.md's "Honest scope notes"); extending
// beyond h=2 had only been done via a JS/Keccak256 hash-chain SIMULATION, not real Fabric
// transactions. This file makes every hop of a multi-hop path a genuine, separate, real on-chain
// transaction: each hop reads the PREVIOUS hop's real committed record via GetState (real
// state-binding, real MVCC-eligible read) before writing its own. Two real, additive entry points
// mirror the composed-vs-pairwise-local contrast already established (X5a, e9_multihop_extension.js):
//   - SubmitHopDecision (composed/real): the accumulated context-preservation flag is the AND of
//     every hop's own contextAtHop value, genuinely propagated hop-by-hop through real reads.
//   - SubmitHopDecisionPairwiseLocal (ablated): each hop still genuinely reads and requires the
//     previous hop to exist (real state-binding continuity), but does NOT propagate or examine the
//     accumulated context truth at all -- matching X5a's real boolean_path behavior exactly.
// Does not modify any existing entry point or struct.
package chaincode

import (
	"encoding/json"
	"fmt"

	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
)

type HopRecord struct {
	ChainID           string `json:"chainId"`
	HopIndex          int64  `json:"hopIndex"`
	ContextAtHop      bool   `json:"contextAtHop"`
	ComposedContextOK bool   `json:"composedContextOk"`
	Mode              string `json:"mode"` // "composed" or "pairwise"
	Class             string `json:"class"`
}

func hopKey(chainID string, mode string, hopIndex int64) string {
	return fmt.Sprintf("hop:%s:%s:%d", mode, chainID, hopIndex)
}

// SubmitHopDecision: real, composed multi-hop verification. hopIndex 0 is the origin; each
// subsequent hop genuinely reads the previous real hop record (GetState) and ANDs its own
// contextAtHop into the accumulated ComposedContextOK. The class returned reflects the FULL,
// real accumulated history, not just this hop's own value.
func (c *AuthContract) SubmitHopDecision(ctx contractapi.TransactionContextInterface,
	chainID string, hopIndex int64, contextAtHop bool) (string, error) {

	key := hopKey(chainID, "composed", hopIndex)
	existing, err := ctx.GetStub().GetState(key)
	if err != nil {
		return "", fmt.Errorf("failed to read state: %v", err)
	}
	if existing != nil {
		return "", fmt.Errorf("hop %d of chain %s already recorded (replay rejected)", hopIndex, chainID)
	}

	composed := contextAtHop
	if hopIndex > 0 {
		prevKey := hopKey(chainID, "composed", hopIndex-1)
		prevBytes, err := ctx.GetStub().GetState(prevKey)
		if err != nil {
			return "", fmt.Errorf("failed to read previous hop: %v", err)
		}
		if prevBytes == nil {
			return "", fmt.Errorf("previous hop %d of chain %s not found -- real state-binding continuity broken", hopIndex-1, chainID)
		}
		var prev HopRecord
		if err := json.Unmarshal(prevBytes, &prev); err != nil {
			return "", fmt.Errorf("corrupt previous hop record: %v", err)
		}
		composed = prev.ComposedContextOK && contextAtHop
	}

	class := "REJECT"
	if composed {
		class = "SAFE"
	}
	rec := HopRecord{ChainID: chainID, HopIndex: hopIndex, ContextAtHop: contextAtHop,
		ComposedContextOK: composed, Mode: "composed", Class: class}
	recJSON, err := json.Marshal(rec)
	if err != nil {
		return "", err
	}
	if err := ctx.GetStub().PutState(key, recJSON); err != nil {
		return "", fmt.Errorf("failed to write state: %v", err)
	}
	return class, nil
}

// SubmitHopDecisionPairwiseLocal: ablated multi-hop verification (E9's "pairwise-local" design).
// Each hop still genuinely requires the previous hop to be really, on-chain present (real
// GetState existence check -- state-binding continuity is NOT removed), but the returned class
// NEVER examines the accumulated context truth: it always reports SAFE once continuity holds,
// exactly matching X5a's real boolean_path behavior.
func (c *AuthContract) SubmitHopDecisionPairwiseLocal(ctx contractapi.TransactionContextInterface,
	chainID string, hopIndex int64, contextAtHop bool) (string, error) {

	key := hopKey(chainID, "pairwise", hopIndex)
	existing, err := ctx.GetStub().GetState(key)
	if err != nil {
		return "", fmt.Errorf("failed to read state: %v", err)
	}
	if existing != nil {
		return "", fmt.Errorf("hop %d of chain %s already recorded (replay rejected)", hopIndex, chainID)
	}

	if hopIndex > 0 {
		prevKey := hopKey(chainID, "pairwise", hopIndex-1)
		prevBytes, err := ctx.GetStub().GetState(prevKey)
		if err != nil {
			return "", fmt.Errorf("failed to read previous hop: %v", err)
		}
		if prevBytes == nil {
			return "", fmt.Errorf("previous hop %d of chain %s not found -- state-binding continuity broken", hopIndex-1, chainID)
		}
		// Pairwise-local ablated: the previous hop's existence is checked (real), but its
		// ComposedContextOK/context history is deliberately NOT read or propagated here.
	}

	// Pairwise-local ablated: contextAtHop is recorded but NEVER examined for the class decision.
	class := "SAFE"
	rec := HopRecord{ChainID: chainID, HopIndex: hopIndex, ContextAtHop: contextAtHop,
		ComposedContextOK: false, Mode: "pairwise", Class: class}
	recJSON, err := json.Marshal(rec)
	if err != nil {
		return "", err
	}
	if err := ctx.GetStub().PutState(key, recJSON); err != nil {
		return "", fmt.Errorf("failed to write state: %v", err)
	}
	return class, nil
}

// ReadHop returns a real hop record for either mode ("composed" or "pairwise").
func (c *AuthContract) ReadHop(ctx contractapi.TransactionContextInterface,
	chainID string, mode string, hopIndex int64) (*HopRecord, error) {
	key := hopKey(chainID, mode, hopIndex)
	b, err := ctx.GetStub().GetState(key)
	if err != nil {
		return nil, fmt.Errorf("failed to read state: %v", err)
	}
	if b == nil {
		return nil, fmt.Errorf("hop %d of chain %s (%s) does not exist", hopIndex, chainID, mode)
	}
	var rec HopRecord
	if err := json.Unmarshal(b, &rec); err != nil {
		return nil, err
	}
	return &rec, nil
}
