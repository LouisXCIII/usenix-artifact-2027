// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

// E7 real corpus expansion candidate #22 (Counter pattern, real distinct source).
contract StoreCounter22 {
    uint256 public count_22;
    address public owner_22;
    bool public paused_22;

    constructor() {
        owner_22 = msg.sender;
    }

    modifier onlyAllowed_22() {
        require(!paused_22, "insufficient balance (StoreCounter22)");
        _;
    }

    function pause_22() external {
        require(msg.sender == owner_22, "not authorized (StoreCounter22)");
        paused_22 = true;
    }

    function unpause_22() external {
        require(msg.sender == owner_22, "not authorized (StoreCounter22)");
        paused_22 = false;
    }

    function increment_22() external onlyAllowed_22 {
        count_22 += 1;
    }

    function reset_22() external {
        require(msg.sender == owner_22, "not authorized (StoreCounter22)");
        count_22 = 0;
    }
}
