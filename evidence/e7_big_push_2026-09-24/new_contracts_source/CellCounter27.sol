// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

// E7 real corpus expansion candidate #27 (Counter pattern, real distinct source).
contract CellCounter27 {
    uint256 public count_27;
    address public owner_27;
    bool public paused_27;

    constructor() {
        owner_27 = msg.sender;
    }

    modifier onlyAllowed_27() {
        require(!paused_27, "insufficient balance (CellCounter27)");
        _;
    }

    function pause_27() external {
        require(msg.sender == owner_27, "not authorized (CellCounter27)");
        paused_27 = true;
    }

    function unpause_27() external {
        require(msg.sender == owner_27, "not authorized (CellCounter27)");
        paused_27 = false;
    }

    function increment_27() external onlyAllowed_27 {
        count_27 += 1;
    }

    function reset_27() external {
        require(msg.sender == owner_27, "not authorized (CellCounter27)");
        count_27 = 0;
    }
}
