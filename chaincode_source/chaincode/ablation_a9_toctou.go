// ablation_a9_toctou.go -- E8 limitation fix, going deeper on A9 (TOCTOU/state binding). The
// earlier pass (evidence/e8_remaining_ablations_2026-09-24.md) explained A9 as not ablatable
// because Fabric's MVCC read-set validation is automatic for every GetState call. That is still
// true for GetState itself -- but clock.go's own real code already contains a genuine,
// pre-existing toggle for exactly this property: `ledgerNow(ctx)` reads the freshness clock via
// GetState(clock:now) (real, version-tracked, therefore genuinely bound into the transaction's
// MVCC read-set -- a concurrent SetClock races against it for real), while the alternative branch
// (`useLedgerClock=false`, its own comment: "tests may disable") uses
// ctx.GetStub().GetTxTimestamp() -- the CALLER's own chosen transaction timestamp, which involves
// no GetState call at all and therefore adds NOTHING to the read-set: no amount of concurrent
// clock activity can ever conflict with it, because nothing about time is being read from
// versioned state in the first place. This file exposes that second path as a real, additive,
// separately invocable entry point (not by mutating the shared global, which would affect every
// other real entry point) so it can be compared against the real, state-bound path under a
// genuine concurrent race -- mirroring A7's finding that a second, real, separately-configurable
// mechanism existed after all.
package chaincode

import (
	"fmt"

	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
)

// SubmitDecisionTrustlessCallerTimestamp is SubmitDecisionTrustless with its time reference taken
// directly from ctx.GetStub().GetTxTimestamp() (the caller's own chosen transaction timestamp)
// instead of ledgerNow(ctx) (the real, state-bound ledger clock). verifySourceState itself is
// reused completely unchanged -- only how "now" is obtained differs. A9 ablated: the freshness
// check still runs, but its time reference is no longer bound into the transaction's MVCC
// read-set, so no concurrent ledger-clock change can ever be detected as a race against it.
func (c *AuthContract) SubmitDecisionTrustlessCallerTimestamp(ctx contractapi.TransactionContextInterface,
	decisionID string, sourceDomain string, role string, amount int64, sourceLimit int64,
	codeHash string, proofJSON string, publicSignalsJSON string,
	headerJSON string, registryProofJSON string) (string, error) {

	exists, err := c.decisionExists(ctx, decisionID)
	if err != nil {
		return "", err
	}
	if exists {
		return "", fmt.Errorf("decision %s already recorded (replay rejected)", decisionID)
	}
	ch, err := unhexBytes(codeHash)
	if err != nil || len(ch) != 32 {
		return "", fmt.Errorf("codeHash must be 32 bytes of hex")
	}
	// A9 ablated: GetTxTimestamp(), NOT ledgerNow(ctx) -- no GetState call, no read-set entry.
	ts, err := ctx.GetStub().GetTxTimestamp()
	if err != nil {
		return "", err
	}
	nowSec := ts.GetSeconds()
	src := verifySourceState(nowSec, ch, headerJSON, registryProofJSON)
	return submitDecisionTrustlessAblated(ctx, decisionID, sourceDomain, role, amount, sourceLimit,
		codeHash, proofJSON, publicSignalsJSON, headerJSON, registryProofJSON, src,
		"A9 ablated: time reference is the caller's own GetTxTimestamp, not the state-bound ledger clock")
}
