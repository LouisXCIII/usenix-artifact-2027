// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

// E7 real corpus expansion candidate #2 (Counter pattern, real distinct source).
contract PlainCounter2 {
    uint256 public count_2;
    address public owner_2;
    bool public paused_2;

    constructor() {
        owner_2 = msg.sender;
    }

    modifier onlyAllowed_2() {
        require(!paused_2, "insufficient balance (PlainCounter2)");
        _;
    }

    function pause_2() external {
        require(msg.sender == owner_2, "not authorized (PlainCounter2)");
        paused_2 = true;
    }

    function unpause_2() external {
        require(msg.sender == owner_2, "not authorized (PlainCounter2)");
        paused_2 = false;
    }

    function increment_2() external onlyAllowed_2 {
        count_2 += 1;
    }

    function reset_2() external {
        require(msg.sender == owner_2, "not authorized (PlainCounter2)");
        count_2 = 0;
    }
}
