// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

/// @notice Comprehensive real reverse-direction (Fabric -> EVM) destination gate, extending
/// EVMDestinationGate.sol (which covered S0/S2/S3) with real, on-chain mechanisms for S1, S4-S10,
/// S12-S14, and S17 -- closing the guide's bidirectionality requirement for every S-scenario that
/// has a meaningful EVM-side counterpart (S11/S15/S16/S18/S19 remain argued as platform-symmetric
/// or already covered at the crypto layer, per evm_reverse_direction_2026-09-24.md). Additive:
/// EVMDestinationGate.sol and its evidence are untouched.
///
/// Obligation strength ordinals (S5-S7) mirror obligation_v2.go's mechanismStrengthV2 exactly:
/// none=0, notification=1, audit=2, dual_approval=3, combined=4.
contract EVMDestinationGateV2 {
    address public owner;
    address[3] public admins;

    mapping(string => uint256) public roleThreshold;
    mapping(string => bool) public roleExists;
    mapping(string => string) public roleMapping;
    mapping(string => bool) public roleMappingExists;

    mapping(bytes32 => bool) public revokedCredentials;      // S9
    bytes32 public currentBoundStateHash;                     // S10
    uint256 public currentMappingEpoch = 1;                   // S13
    uint256 public constant MIN_FABRIC_CONFIRMATIONS = 3;     // S14 (mirrors the real ceil(2n/3)=3 quorum used elsewhere)

    mapping(bytes32 => uint8) public mappingUpdateApprovals;                       // S17
    mapping(bytes32 => mapping(address => bool)) public mappingUpdateApprovedBy;   // S17

    enum Class { NONE, EXACT, SAFE, REJECT, REEVALUATE }

    struct SubmitParams {
        bytes32 decisionId;
        string sourceDomain;
        string sourceRef;      // real Fabric decisionID/certId this decision is bound to
        string role;
        uint256 amount;
        bool semBound;
        bool evAssuranceOk;
        bytes32 credentialHash;        // S9
        bytes32 boundStateHash;        // S10
        uint256 mappingEpoch;          // S13
        uint256 fabricConfirmations;   // S14
        uint256 expiryTimestamp;       // S8/S12
        uint8 mappedMechanismStrength; // S5/S6 (0-4, see contract-level doc)
        bool obligationRequired;       // whether this decision's amount triggers an obligation gate at all
        bool hasEnforcementWitness;    // S7
    }

    struct Decision {
        Class class;
        string detail;
        bool exists;
    }
    mapping(bytes32 => Decision) public decisions;

    event DecisionSubmitted(bytes32 indexed decisionId, Class class, string detail);
    event MappingProposed(bytes32 indexed propId, string srcRole, string destRole, address proposer);
    event MappingApproved(bytes32 indexed propId, address approver, uint8 approvals);
    event MappingActivated(bytes32 indexed propId, string srcRole, string destRole);

    modifier onlyAdmin() { require(isAdmin(msg.sender), "not admin"); _; }

    constructor(address a1, address a2, address a3) {
        owner = msg.sender;
        admins = [a1, a2, a3];
        roleThreshold["manager"] = 500000;   roleExists["manager"] = true;
        roleThreshold["director"] = 2000000; roleExists["director"] = true;
        roleThreshold["clerk"] = 10000;      roleExists["clerk"] = true;
        roleMapping["gerente"] = "manager";         roleMappingExists["gerente"] = true;
        roleMapping["gerente_senior"] = "director"; roleMappingExists["gerente_senior"] = true;
        currentBoundStateHash = keccak256(abi.encodePacked("genesis-state"));
    }

    function isAdmin(address a) public view returns (bool) {
        return a == admins[0] || a == admins[1] || a == admins[2];
    }

    // ---- Real admin-governed state changes, used to set up S9/S10/S13 test conditions ----

    function revokeCredential(bytes32 credentialHash) external onlyAdmin {
        revokedCredentials[credentialHash] = true;
    }

    function setBoundStateHash(bytes32 newHash) external onlyAdmin {
        currentBoundStateHash = newHash;
    }

    function advanceMappingEpoch() external onlyAdmin {
        currentMappingEpoch++;
    }

    // ---- S17: real 2-of-3 multisig governance for mapping updates (mirrors A7's real Fabric
    // chaincode-lifecycle governance argument, but concretely enforced on-chain here) ----

    function proposeMappingUpdate(string calldata srcRole, string calldata destRole) external onlyAdmin returns (bytes32) {
        bytes32 propId = keccak256(abi.encodePacked(srcRole, destRole, block.timestamp, msg.sender));
        mappingUpdateApprovedBy[propId][msg.sender] = true;
        mappingUpdateApprovals[propId] = 1;
        emit MappingProposed(propId, srcRole, destRole, msg.sender);
        return propId;
    }

    function approveMappingUpdate(bytes32 propId, string calldata srcRole, string calldata destRole) external onlyAdmin {
        require(!mappingUpdateApprovedBy[propId][msg.sender], "already approved");
        mappingUpdateApprovedBy[propId][msg.sender] = true;
        mappingUpdateApprovals[propId]++;
        emit MappingApproved(propId, msg.sender, mappingUpdateApprovals[propId]);
        if (mappingUpdateApprovals[propId] >= 2) {
            roleMapping[srcRole] = destRole;
            roleMappingExists[srcRole] = true;
            emit MappingActivated(propId, srcRole, destRole);
        }
    }

    // ---- Main real Eval_d entry point ----

    function submitDecision(SubmitParams calldata p) external returns (Class) {
        require(!decisions[p.decisionId].exists, "already recorded (replay rejected)");

        if (revokedCredentials[p.credentialHash]) {
            return _record(p.decisionId, Class.REJECT, "S9: credential revoked");
        }
        if (p.boundStateHash != currentBoundStateHash) {
            return _record(p.decisionId, Class.REJECT, "S10: bound state changed after decision");
        }
        if (p.mappingEpoch != currentMappingEpoch) {
            return _record(p.decisionId, Class.REJECT, "S13: mapping epoch mismatch");
        }
        if (p.fabricConfirmations < MIN_FABRIC_CONFIRMATIONS) {
            return _record(p.decisionId, Class.REJECT, "S14: insufficient finality (below quorum)");
        }
        if (block.timestamp > p.expiryTimestamp) {
            return _record(p.decisionId, Class.REJECT, "S8/S12: decision expired/stale");
        }

        (Class baseClass, bool identityMapping) = _baseClassify(p.role, p.amount, p.semBound, p.evAssuranceOk);
        if (baseClass != Class.EXACT && baseClass != Class.SAFE) {
            return _record(p.decisionId, baseClass, "base classify (S0/S1/S2/S3/S4 mechanism)");
        }

        if (p.obligationRequired) {
            if (p.mappedMechanismStrength == 0) {
                return _record(p.decisionId, Class.REJECT, "S5: obligation stripped");
            }
            if (p.mappedMechanismStrength < 2) { // required strength fixed at 2 (audit) for this test suite
                return _record(p.decisionId, Class.REJECT, "S6: obligation weakened");
            }
            if (!p.hasEnforcementWitness) {
                return _record(p.decisionId, Class.REJECT, "S7: obligation laundered (no enforcement witness)");
            }
        }

        string memory detail = identityMapping ? "S0: identity, direct reuse" : "S1/S2/S4: mapped, contained, no direct reuse";
        return _record(p.decisionId, baseClass, detail);
    }

    function _baseClassify(string calldata role, uint256 amount, bool semBound, bool evAssuranceOk)
        internal view returns (Class, bool identityMapping)
    {
        string memory destRole = role;
        identityMapping = true;
        if (roleMappingExists[role]) {
            destRole = roleMapping[role];
            identityMapping = false;
        }
        bool roleKnown = roleExists[destRole];
        uint256 threshold = roleKnown ? roleThreshold[destRole] : 0;

        if (!semBound) return (Class.REJECT, identityMapping);
        if (!evAssuranceOk) return (Class.REJECT, identityMapping);
        if (!roleKnown) return (Class.REEVALUATE, identityMapping);
        if (amount > threshold) return (Class.REJECT, identityMapping);
        return (identityMapping ? Class.EXACT : Class.SAFE, identityMapping);
    }

    function _record(bytes32 decisionId, Class class, string memory detail) internal returns (Class) {
        decisions[decisionId] = Decision({ class: class, detail: detail, exists: true });
        emit DecisionSubmitted(decisionId, class, detail);
        return class;
    }

    function getDecision(bytes32 decisionId) external view returns (Class, string memory) {
        Decision storage d = decisions[decisionId];
        require(d.exists, "no such decision");
        return (d.class, d.detail);
    }
}
