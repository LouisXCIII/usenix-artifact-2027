package main

import (
	"fmt"

	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"

	"github.com/authportability/authcc/chaincode"
)

func main() {
	authChaincode, err := contractapi.NewChaincode(&chaincode.AuthContract{})
	if err != nil {
		fmt.Printf("Error creating auth-portability chaincode: %v\n", err)
		return
	}
	if err := authChaincode.Start(); err != nil {
		fmt.Printf("Error starting auth-portability chaincode: %v\n", err)
	}
}
