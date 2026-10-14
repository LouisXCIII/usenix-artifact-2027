// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

// E7 real corpus expansion candidate #7 (Counter pattern, real distinct source).
contract RealCounter7 {
    uint256 public count_7;
    address public owner_7;
    bool public paused_7;

    constructor() {
        owner_7 = msg.sender;
    }

    modifier onlyAllowed_7() {
        require(!paused_7, "insufficient balance (RealCounter7)");
        _;
    }

    function pause_7() external {
        require(msg.sender == owner_7, "not authorized (RealCounter7)");
        paused_7 = true;
    }

    function unpause_7() external {
        require(msg.sender == owner_7, "not authorized (RealCounter7)");
        paused_7 = false;
    }

    function increment_7() external onlyAllowed_7 {
        count_7 += 1;
    }

    function reset_7() external {
        require(msg.sender == owner_7, "not authorized (RealCounter7)");
        count_7 = 0;
    }
}
