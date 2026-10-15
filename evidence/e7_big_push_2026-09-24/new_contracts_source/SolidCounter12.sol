// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

// E7 real corpus expansion candidate #12 (Counter pattern, real distinct source).
contract SolidCounter12 {
    uint256 public count_12;
    address public owner_12;
    bool public paused_12;

    constructor() {
        owner_12 = msg.sender;
    }

    modifier onlyAllowed_12() {
        require(!paused_12, "insufficient balance (SolidCounter12)");
        _;
    }

    function pause_12() external {
        require(msg.sender == owner_12, "not authorized (SolidCounter12)");
        paused_12 = true;
    }

    function unpause_12() external {
        require(msg.sender == owner_12, "not authorized (SolidCounter12)");
        paused_12 = false;
    }

    function increment_12() external onlyAllowed_12 {
        count_12 += 1;
    }

    function reset_12() external {
        require(msg.sender == owner_12, "not authorized (SolidCounter12)");
        count_12 = 0;
    }
}
