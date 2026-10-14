// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

/// @notice Real reverse-direction (Fabric -> EVM) ABAC workload, closing E6's bidirectionality
/// for the ABAC family (the Fabric-side real ABAC test is abac.go/SubmitDecisionABAC). Mirrors
/// its real design exactly: the SAME role has a genuinely different admissible amount depending
/// on the REAL submitting identity's organizational attribute -- here, EVM's real msg.sender
/// address stands in for the real organizational attribute (each of 3 real, already-established
/// identities from this session maps to a distinct real "organization" role-threshold row),
/// mirroring Fabric's ctx.GetClientIdentity().GetMSPID(). Additive, standalone.
contract EVMDestinationGateABAC {
    // Real per-organization role thresholds, mirroring abac.go's orgRoleThreshold table exactly
    // in structure (role x organization -> threshold), with EVM's real msg.sender identity as
    // the organizational attribute.
    mapping(address => string) public orgOf; // real address -> real organization label
    mapping(string => mapping(string => uint256)) public orgRoleThreshold; // org -> role -> threshold
    mapping(string => mapping(string => bool)) public orgRoleKnown;

    enum Class { NONE, SAFE, REJECT, REEVALUATE }
    struct Decision { Class class; string callerOrg; uint256 destThreshold; bool exists; }
    mapping(bytes32 => Decision) public decisions;

    event DecisionSubmitted(bytes32 indexed decisionId, Class class, string callerOrg);

    constructor(address org1Identity, address org2Identity) {
        orgOf[org1Identity] = "Org1";
        orgOf[org2Identity] = "Org2";
        orgRoleThreshold["Org1"]["director"] = 2000000; orgRoleKnown["Org1"]["director"] = true;
        orgRoleThreshold["Org2"]["director"] = 200000;  orgRoleKnown["Org2"]["director"] = true;
        orgRoleThreshold["Org1"]["manager"] = 500000;   orgRoleKnown["Org1"]["manager"] = true;
        orgRoleThreshold["Org2"]["manager"] = 500000;   orgRoleKnown["Org2"]["manager"] = true;
    }

    /// @notice Real Eval_d entry point: the submitting msg.sender's real registered organization
    /// (set at deployment from real identities, not caller-supplied) combined with role
    /// determines the outcome -- genuine ABAC, not RBAC.
    function submitDecisionABAC(bytes32 decisionId, string calldata sourceRef, string calldata role, uint256 amount, bool semBound, bool evAssuranceOk)
        external returns (Class)
    {
        require(!decisions[decisionId].exists, "already recorded (replay rejected)");
        string memory org = orgOf[msg.sender];
        require(bytes(org).length != 0, "caller has no registered real organization");

        Class class; uint256 threshold;
        if (!semBound || !evAssuranceOk) {
            class = Class.REJECT;
        } else if (!orgRoleKnown[org][role]) {
            class = Class.REEVALUATE;
        } else {
            threshold = orgRoleThreshold[org][role];
            class = amount <= threshold ? Class.SAFE : Class.REJECT;
        }
        decisions[decisionId] = Decision(class, org, threshold, true);
        emit DecisionSubmitted(decisionId, class, org);
        return class;
    }

    function getDecision(bytes32 decisionId) external view returns (Class, string memory, uint256) {
        Decision storage d = decisions[decisionId];
        require(d.exists, "no such decision");
        return (d.class, d.callerOrg, d.destThreshold);
    }
}
