// clock.go — X4 mitigation: an authoritative on-ledger clock.
//
// Fabric does not check ChannelHeader.Timestamp against the peer's clock (tsclient, X4), so a submitter chooses
// the value GetTxTimestamp returns and any freshness or expiry test built on it is defeatable by that submitter.
// The verifying entry points therefore use ledgerNow instead: a clock kept in world state that only the pinned
// clock authority can advance (monotonically). A submitter can no longer move it. What remains is trust in that
// authority and its update cadence: a decision or header is "fresh" relative to the last clock update, not to
// wall-clock time.
package chaincode

import (
	"fmt"
	"strconv"

	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
)

const (
	clockKey          = "clock:now"
	clockAuthorityMSP = "Org1MSP"
	clockAuthorityCN  = "org1admin"
)

// useLedgerClock selects the ledger clock for the verifying entry points; tests may disable it.
var useLedgerClock = true

func readClock(ctx contractapi.TransactionContextInterface) (int64, error) {
	b, err := ctx.GetStub().GetState(clockKey)
	if err != nil {
		return 0, err
	}
	if b == nil {
		return 0, fmt.Errorf("no ledger clock: the clock authority has not set one (fail-closed)")
	}
	return strconv.ParseInt(string(b), 10, 64)
}

// ledgerNow is the time used for freshness and expiry.
func ledgerNow(ctx contractapi.TransactionContextInterface) (int64, error) {
	if !useLedgerClock {
		ts, err := ctx.GetStub().GetTxTimestamp()
		if err != nil {
			return 0, err
		}
		return ts.GetSeconds(), nil
	}
	return readClock(ctx)
}

// SetClock advances the ledger clock. Only the pinned authority identity may call it, and it never moves backwards.
func (c *AuthContract) SetClock(ctx contractapi.TransactionContextInterface, t int64) error {
	id := ctx.GetClientIdentity()
	msp, err := id.GetMSPID()
	if err != nil {
		return err
	}
	cert, err := id.GetX509Certificate()
	if err != nil {
		return err
	}
	if msp != clockAuthorityMSP || cert.Subject.CommonName != clockAuthorityCN {
		return fmt.Errorf("only %s/%s may set the clock (caller %s/%s)", clockAuthorityMSP, clockAuthorityCN, msp, cert.Subject.CommonName)
	}
	if old, err := readClock(ctx); err == nil && t < old {
		return fmt.Errorf("clock may not move backwards (%d < %d)", t, old)
	}
	return ctx.GetStub().PutState(clockKey, []byte(strconv.FormatInt(t, 10)))
}

// LedgerClock returns the ledger clock; unlike TxTimestamp it does not depend on anything the submitter chooses.
func (c *AuthContract) LedgerClock(ctx contractapi.TransactionContextInterface) (int64, error) {
	return readClock(ctx)
}
