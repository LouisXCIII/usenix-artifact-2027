// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

// E7 real corpus expansion candidate #17 (Counter pattern, real distinct source).
contract LedgerCounter17 {
    uint256 public count_17;
    address public owner_17;
    bool public paused_17;

    constructor() {
        owner_17 = msg.sender;
    }

    modifier onlyAllowed_17() {
        require(!paused_17, "insufficient balance (LedgerCounter17)");
        _;
    }

    function pause_17() external {
        require(msg.sender == owner_17, "not authorized (LedgerCounter17)");
        paused_17 = true;
    }

    function unpause_17() external {
        require(msg.sender == owner_17, "not authorized (LedgerCounter17)");
        paused_17 = false;
    }

    function increment_17() external onlyAllowed_17 {
        count_17 += 1;
    }

    function reset_17() external {
        require(msg.sender == owner_17, "not authorized (LedgerCounter17)");
        count_17 = 0;
    }
}
