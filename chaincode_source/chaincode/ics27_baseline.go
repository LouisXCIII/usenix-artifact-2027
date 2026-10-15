package chaincode

import (
	"encoding/json"
	"fmt"

	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
)

// ics27AllowMessages is the type-level allowlist baseline (ICS-27 AllowMessages style): a message
// type is either permitted or not, with no access to the transferred amount.
var ics27AllowMessages = map[string]bool{"transfer": true}

// SubmitDecisionICS27 records the decision of a type-level allowlist that never reads amount, so it
// is the comparison point for value-sensitive widening. Used only as an evaluation baseline.
func (c *AuthContract) SubmitDecisionICS27(ctx contractapi.TransactionContextInterface,
	decisionID string, sourceDomain string, messageType string, amount int64) (string, error) {

	exists, err := c.decisionExists(ctx, decisionID)
	if err != nil {
		return "", err
	}
	if exists {
		return "", fmt.Errorf("decision %s already recorded (replay rejected)", decisionID)
	}
	if amount <= 0 {
		return "", fmt.Errorf("amount must be positive (got %d)", amount)
	}

	class := "REJECT"
	if ics27AllowMessages[messageType] {
		class = "SAFE"
	}
	rec := PortableDecisionRecord{
		DecisionID: decisionID, SourceDomain: sourceDomain, Role: messageType, Amount: amount,
		TransferClass: class,
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
