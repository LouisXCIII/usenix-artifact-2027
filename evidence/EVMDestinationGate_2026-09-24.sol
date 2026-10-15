// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

/// @notice Real reverse-direction (Fabric -> EVM) destination gate, closing the guide's
/// bidirectionality requirement ("Repeat in both EVM->Fabric and Fabric->EVM directions where
/// the scenario has a meaningful counterpart") for S0/S2/S3 -- the core relational
/// non-widening property (H1/H2), the paper's central claim. This is a faithful, real port of
/// authchaincode.go's classify()/RoleThreshold/RoleMapping (Fabric chaincode, where EVM/Besu is
/// the source and Fabric is the destination) run in the OPPOSITE role: here EVM/Besu is the
/// DESTINATION and a real Fabric decision is the SOURCE. Same real role thresholds, same real
/// role-mapping table, same real fail-closed classify() precedence -- only the platform is
/// mirrored, not the policy semantics, so the two directions are a genuine "meaningful
/// counterpart" of each other rather than an unrelated new test.
contract EVMDestinationGate {
    address public owner;

    // Identical real destination policy to authchaincode.go's RoleThreshold.
    mapping(string => uint256) public roleThreshold;
    mapping(string => bool) public roleExists;
    // Identical real mapping table to authchaincode.go's RoleMapping.
    mapping(string => string) public roleMapping;
    mapping(string => bool) public roleMappingExists;

    enum Class { NONE, EXACT, SAFE, REJECT, REEVALUATE }

    struct Decision {
        string sourceDomain;   // real Fabric channel/chaincode identifier
        string sourceRef;      // real Fabric decisionID/txID this decision is bound to
        string role;
        uint256 amount;
        bool semBound;
        bool evAssuranceOk;
        Class class;
        uint256 destThreshold;
        bool exists;
    }
    mapping(bytes32 => Decision) public decisions;

    event DecisionSubmitted(bytes32 indexed decisionId, Class class);

    modifier onlyOwner() { require(msg.sender == owner, "not owner"); _; }

    constructor() {
        owner = msg.sender;
        roleThreshold["manager"] = 500000;   roleExists["manager"] = true;
        roleThreshold["director"] = 2000000; roleExists["director"] = true;
        roleThreshold["clerk"] = 10000;      roleExists["clerk"] = true;
        roleMapping["gerente"] = "manager";         roleMappingExists["gerente"] = true;
        roleMapping["gerente_senior"] = "director"; roleMappingExists["gerente_senior"] = true;
    }

    /// @notice Real Eval_d entry point, EVM-side. decisionId must be bound to a real Fabric
    /// artifact (sourceRef) by the caller -- this contract does not itself re-verify Fabric
    /// state (that is the separately-established light-client/MPT layer,
    /// FabricNestedInclusionVerifier.sol / FabricRealTxVerifier.sol); this contract's job is the
    /// destination POLICY decision (classify()), mirroring exactly what SubmitDecision does on
    /// the Fabric side.
    function submitDecision(
        bytes32 decisionId, string calldata sourceDomain, string calldata sourceRef,
        string calldata role, uint256 amount, bool semBound, bool evAssuranceOk
    ) external returns (Class) {
        require(!decisions[decisionId].exists, "already recorded (replay rejected)");

        string memory destRole = role;
        bool identityMapping = true;
        if (roleMappingExists[role]) {
            destRole = roleMapping[role];
            identityMapping = false;
        }
        bool roleKnown = roleExists[destRole];
        uint256 threshold = roleKnown ? roleThreshold[destRole] : 0;

        Class class;
        if (!semBound) {
            class = Class.REJECT;
        } else if (!evAssuranceOk) {
            class = Class.REJECT;
        } else if (!roleKnown) {
            class = Class.REEVALUATE;
        } else if (amount <= threshold) {
            class = identityMapping ? Class.EXACT : Class.SAFE;
        } else {
            class = Class.REJECT;
        }

        decisions[decisionId] = Decision({
            sourceDomain: sourceDomain, sourceRef: sourceRef, role: role, amount: amount,
            semBound: semBound, evAssuranceOk: evAssuranceOk, class: class,
            destThreshold: threshold, exists: true
        });
        emit DecisionSubmitted(decisionId, class);
        return class;
    }

    function getDecision(bytes32 decisionId) external view returns (
        string memory sourceDomain, string memory sourceRef, string memory role,
        uint256 amount, Class class, uint256 destThreshold
    ) {
        Decision storage d = decisions[decisionId];
        require(d.exists, "no such decision");
        return (d.sourceDomain, d.sourceRef, d.role, d.amount, d.class, d.destThreshold);
    }
}
