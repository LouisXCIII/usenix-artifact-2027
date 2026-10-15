// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

contract MultiSigApproval {
    address[] public signers;
    mapping(address => bool) public isSigner;
    mapping(bytes32 => mapping(address => bool)) public approvals;
    mapping(bytes32 => uint256) public approvalCount;
    uint256 public threshold;

    constructor(address[] memory _signers, uint256 _threshold) {
        require(_threshold <= _signers.length, "threshold too high");
        for (uint256 i = 0; i < _signers.length; i++) {
            signers.push(_signers[i]);
            isSigner[_signers[i]] = true;
        }
        threshold = _threshold;
    }

    function approve(bytes32 actionId) external {
        require(isSigner[msg.sender], "not a signer");
        require(!approvals[actionId][msg.sender], "already approved");
        approvals[actionId][msg.sender] = true;
        approvalCount[actionId] += 1;
    }

    function isApproved(bytes32 actionId) external view returns (bool) {
        return approvalCount[actionId] >= threshold;
    }
}
