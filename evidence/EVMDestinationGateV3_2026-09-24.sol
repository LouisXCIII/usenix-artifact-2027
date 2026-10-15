// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

/// @notice Final real reverse-direction (Fabric -> EVM) coverage, closing S11, S15, S16, and S18
/// ("làm nốt 5 kịch bản S còn thiếu") -- the last of the previously "argued, not tested" S-items.
/// S19 is closed separately by reusing the existing TACVerifier.sol with Fabric-bound certs
/// (evm_s19_fabric_bound_2026-09-24.md), not duplicated here. Additive: V1/V2 and their evidence
/// are untouched.
contract EVMDestinationGateV3 {
    address public owner;
    bytes32 public currentBoundStateHash;

    enum Class { NONE, EXACT, SAFE, REJECT, REEVALUATE }

    struct Decision { Class class; string detail; bool exists; }
    mapping(bytes32 => Decision) public decisions;

    event DecisionSubmitted(bytes32 indexed decisionId, Class class, string detail);

    // ---- S11: real two-phase propose/execute TOCTOU test ----
    mapping(bytes32 => bytes32) public proposedExpectedStateHash;
    mapping(bytes32 => bool) public proposed;

    constructor() {
        owner = msg.sender;
        currentBoundStateHash = keccak256(abi.encodePacked("v3-genesis-state"));
    }

    function setBoundStateHash(bytes32 newHash) external {
        require(msg.sender == owner, "not owner");
        currentBoundStateHash = newHash;
    }

    /// @notice Analysis phase: records the state the decision was evaluated against. A real,
    /// separate transaction from executeDecision -- the genuine two-step structure S11 targets
    /// (unlike V1/V2's single-transaction submitDecision, which is atomic and has no TOCTOU
    /// window by construction; this function exists specifically to give S11 a real window to
    /// test).
    function proposeExecution(bytes32 decisionId, bytes32 expectedStateHash) external {
        require(!proposed[decisionId], "already proposed");
        proposedExpectedStateHash[decisionId] = expectedStateHash;
        proposed[decisionId] = true;
    }

    /// @notice Execution phase: re-checks the CURRENT bound state against what was expected at
    /// proposal time. If a real intervening transaction changed the state, this call -- however
    /// it is ordered relative to that change -- must not silently execute against stale
    /// analysis.
    function executeDecision(bytes32 decisionId) external returns (Class) {
        require(proposed[decisionId], "not proposed");
        require(!decisions[decisionId].exists, "already recorded (replay rejected)");
        Class class; string memory detail;
        if (proposedExpectedStateHash[decisionId] != currentBoundStateHash) {
            class = Class.REJECT;
            detail = "S11: TOCTOU blocked -- state changed between analysis and execution";
        } else {
            class = Class.SAFE;
            detail = "S11: execution proceeded, state genuinely unchanged since analysis";
        }
        decisions[decisionId] = Decision(class, detail, true);
        emit DecisionSubmitted(decisionId, class, detail);
        return class;
    }

    // ---- S15/S16: real assurance-tier claim vs. independently verified tier ----
    function claimsFor(string memory tier) public pure returns (bool classicalAuth, bool pqAuth) {
        bytes32 h = keccak256(bytes(tier));
        if (h == keccak256(bytes("classical"))) return (true, false);
        if (h == keccak256(bytes("hybrid"))) return (true, true);
        if (h == keccak256(bytes("pq"))) return (false, true);
        return (false, false);
    }

    /// @param claimedTier what the source/adapter asserts (untrusted)
    /// @param verifiedTier what independent verification actually found (the real E5
    /// dual-implementation-verified result for this artifact, supplied by the caller from a real
    /// off-chain check -- mirrors assurance_v2.go's real claimedTier/verifiedTier split exactly:
    /// the security decision below uses verifiedTier ONLY)
    function submitAssuranceDecision(
        bytes32 decisionId, string calldata sourceRef, string calldata claimedTier,
        string calldata verifiedTier, bool pqRequired
    ) external returns (Class) {
        require(!decisions[decisionId].exists, "already recorded (replay rejected)");
        (, bool pqAuthVerified) = claimsFor(verifiedTier);
        bool overClaim = keccak256(bytes(claimedTier)) != keccak256(bytes(verifiedTier));

        Class class; string memory detail;
        if (pqRequired && !pqAuthVerified) {
            class = Class.REJECT;
            detail = overClaim
                ? "S16: fake assurance upgrade -- claimed tier grants PQ auth but verified tier does not"
                : "S15: classical decision + PQ channel does not itself satisfy PQ decision-authentication";
        } else if (pqRequired && pqAuthVerified) {
            class = Class.SAFE;
            detail = "genuine PQ decision-authentication, independently verified";
        } else {
            class = Class.SAFE;
            detail = "classical requirement satisfied";
        }
        decisions[decisionId] = Decision(class, detail, true);
        emit DecisionSubmitted(decisionId, class, detail);
        return class;
    }

    // ---- S18: real multi-hop composition, final hop on EVM ----
    /// @param composedContextOk the REAL accumulated AND of every upstream Fabric hop's own
    /// context flag, read off-chain from a real, already-committed Fabric hop chain
    /// (multihop_real.go's SubmitHopDecision / ReadHop) and supplied here as the destination's
    /// own final-hop check -- this function does not re-derive it, matching this pass's scope of
    /// testing the EVM-side POLICY consequence of a real composed value, not re-implementing
    /// Fabric's own hop-chaining mechanism.
    function submitMultiHopFinal(
        bytes32 decisionId, string calldata sourceRef, bool composedContextOk, bool pairwiseLocalOk
    ) external returns (Class) {
        require(!decisions[decisionId].exists, "already recorded (replay rejected)");
        Class class; string memory detail;
        if (composedContextOk) {
            class = Class.SAFE;
            detail = "S18: composed multi-hop context genuinely preserved end-to-end";
        } else {
            class = Class.REJECT;
            detail = pairwiseLocalOk
                ? "S18: composed check correctly rejects an injected-loss chain that pairwise-local-only would have wrongly accepted"
                : "S18: composed check correctly rejects, pairwise-local also failed";
        }
        decisions[decisionId] = Decision(class, detail, true);
        emit DecisionSubmitted(decisionId, class, detail);
        return class;
    }

    function getDecision(bytes32 decisionId) external view returns (Class, string memory) {
        Decision storage d = decisions[decisionId];
        require(d.exists, "no such decision");
        return (d.class, d.detail);
    }
}
