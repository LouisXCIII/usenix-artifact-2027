// multihop_t0.go -- S6 (evidence/s6_t0_multihop_protocol_2026-09-28.md): closes the gap the manuscript discloses for multi-hop freshness.
// The model requires freshness at hop j to be evaluated against the ORIGIN timestamp t0 carried in the path state, so that the final
// destination's bound is checked against the total path age; the earlier real multi-hop chaincode (multihop_real.go) propagates only a boolean
// context flag. This file adds, without touching any existing entry point:
//   SubmitHopDecisionT0       composed: context flag AND age-since-origin <= DeltaDecision; OriginT0 is COPIED from the previous hop's
//                             ledger record (no timestamp argument exists), so an intermediate hop cannot move it.
//   SubmitHopDecisionT0Local  same context propagation, but freshness is per hop: gap since the PREVIOUS hop <= DeltaDecision. This is the
//                             "pairwise-local" behaviour the model says is insufficient.
// Time is the ledger clock (ledgerNow), never GetTxTimestamp (submitter-chosen, see clock.go). Both return only the class string, so the two
// endorsing peers produce identical responses.
package chaincode

import (
	"encoding/json"
	"fmt"

	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
)

// DeltaDecision is the destination's own end-to-end freshness bound (seconds); it is deliberately not a caller argument.
const hopDeltaDecisionSec int64 = 120

type HopRecordT0 struct {
	ChainID           string `json:"chainId"`
	HopIndex          int64  `json:"hopIndex"`
	ContextAtHop      bool   `json:"contextAtHop"`
	ComposedContextOK bool   `json:"composedContextOk"`
	Mode              string `json:"mode"` // "composed" or "local"
	Class             string `json:"class"`
	OriginT0          int64  `json:"originT0"`
	HopTime           int64  `json:"hopTime"`
	AgeSinceOrigin    int64  `json:"ageSinceOrigin"`
	GapSincePrev      int64  `json:"gapSincePrev"`
	FreshOK           bool   `json:"freshOk"`
}

func hopT0Key(chainID, mode string, idx int64) string {
	return fmt.Sprintf("hopt0:%s:%s:%d", mode, chainID, idx)
}

func (c *AuthContract) submitHopT0(ctx contractapi.TransactionContextInterface, mode, chainID string, hopIndex int64, contextAtHop bool) (string, error) {
	key := hopT0Key(chainID, mode, hopIndex)
	if existing, err := ctx.GetStub().GetState(key); err != nil {
		return "", fmt.Errorf("failed to read state: %v", err)
	} else if existing != nil {
		return "", fmt.Errorf("hop %d of chain %s already recorded (replay rejected)", hopIndex, chainID)
	}
	now, err := ledgerNow(ctx)
	if err != nil {
		return "", err
	}
	rec := HopRecordT0{ChainID: chainID, HopIndex: hopIndex, ContextAtHop: contextAtHop, Mode: mode, HopTime: now}
	prevOK := true
	if hopIndex == 0 {
		rec.OriginT0 = now
	} else {
		prevBytes, err := ctx.GetStub().GetState(hopT0Key(chainID, mode, hopIndex-1))
		if err != nil {
			return "", fmt.Errorf("failed to read previous hop: %v", err)
		}
		if prevBytes == nil {
			return "", fmt.Errorf("previous hop %d of chain %s not found -- state-binding continuity broken", hopIndex-1, chainID)
		}
		var prev HopRecordT0
		if err := json.Unmarshal(prevBytes, &prev); err != nil {
			return "", fmt.Errorf("corrupt previous hop record: %v", err)
		}
		if now < prev.HopTime {
			return "", fmt.Errorf("ledger clock moved backwards relative to the previous hop (fail-closed)")
		}
		rec.OriginT0 = prev.OriginT0 // copied from the ledger, never supplied by the caller
		rec.GapSincePrev = now - prev.HopTime
		prevOK = prev.ComposedContextOK
	}
	rec.AgeSinceOrigin = now - rec.OriginT0
	switch mode {
	case "composed":
		rec.FreshOK = rec.AgeSinceOrigin <= hopDeltaDecisionSec
	default: // "local": only the gap to the previous hop is examined
		rec.FreshOK = rec.GapSincePrev <= hopDeltaDecisionSec
	}
	rec.ComposedContextOK = prevOK && contextAtHop && rec.FreshOK
	rec.Class = "REJECT"
	if rec.ComposedContextOK {
		rec.Class = "SAFE"
	}
	b, err := json.Marshal(rec)
	if err != nil {
		return "", err
	}
	if err := ctx.GetStub().PutState(key, b); err != nil {
		return "", fmt.Errorf("failed to write state: %v", err)
	}
	return rec.Class, nil
}

func (c *AuthContract) SubmitHopDecisionT0(ctx contractapi.TransactionContextInterface, chainID string, hopIndex int64, contextAtHop bool) (string, error) {
	return c.submitHopT0(ctx, "composed", chainID, hopIndex, contextAtHop)
}

func (c *AuthContract) SubmitHopDecisionT0Local(ctx contractapi.TransactionContextInterface, chainID string, hopIndex int64, contextAtHop bool) (string, error) {
	return c.submitHopT0(ctx, "local", chainID, hopIndex, contextAtHop)
}

// ReadHopT0 returns a stored record of either mode ("composed" or "local").
func (c *AuthContract) ReadHopT0(ctx contractapi.TransactionContextInterface, chainID string, mode string, hopIndex int64) (*HopRecordT0, error) {
	b, err := ctx.GetStub().GetState(hopT0Key(chainID, mode, hopIndex))
	if err != nil {
		return nil, fmt.Errorf("failed to read state: %v", err)
	}
	if b == nil {
		return nil, fmt.Errorf("hop %d of chain %s (%s) does not exist", hopIndex, chainID, mode)
	}
	var rec HopRecordT0
	if err := json.Unmarshal(b, &rec); err != nil {
		return nil, err
	}
	return &rec, nil
}
