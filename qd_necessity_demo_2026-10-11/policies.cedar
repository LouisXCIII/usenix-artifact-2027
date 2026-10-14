// policies.cedar -- a faithful, representative port (director role only, matching the same
// forged/honest test case used for the OPA/Rego side of this experiment) of the real tier gate
// in ~/fabric-network/authcc/chaincode/risk_tier.go (tierFromFloorSignal +
// SubmitDecisionRiskTierVerified). Real constants: tierT1Floor=54445422, tierT2Floor=67108864,
// tierT3Floor=83886080; RoleThreshold["director"]=2000000; tierFractionBps {1:1000,2:5000,
// 3:10000} i.e. 10%/50%/100% of cap, pre-multiplied here as literal dollar caps (200000/
// 1000000/2000000) because Cedar's Long arithmetic is not needed to express the real necessity
// point -- only the >= / <= comparisons that Cedar genuinely supports are used.
//
// What is structurally missing, and cannot be added to this file no matter how it is written:
// a check that context.declaredFloor was the output of a real Groth16 proof verified against
// the embedded bn254 verification key (zkverify.go/verifyGroth16, gnark-crypto's
// bn254.PairingCheck). Cedar's expression grammar has no elliptic-curve or pairing primitive,
// and more fundamentally context.declaredFloor here is just a Long the caller supplies --
// nothing in the Cedar authorization model inspects how that Long came to be.

permit(principal == User::"submitter", action == Action::"submit", resource == Decision::"d")
when {
    context.role == "director" &&
    context.declaredFloor >= 83886080 &&
    context.amount <= 2000000
};

permit(principal == User::"submitter", action == Action::"submit", resource == Decision::"d")
when {
    context.role == "director" &&
    context.declaredFloor >= 67108864 &&
    context.declaredFloor < 83886080 &&
    context.amount <= 1000000
};

permit(principal == User::"submitter", action == Action::"submit", resource == Decision::"d")
when {
    context.role == "director" &&
    context.declaredFloor >= 54445422 &&
    context.declaredFloor < 67108864 &&
    context.amount <= 200000
};
