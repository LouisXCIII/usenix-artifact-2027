// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

contract DeltaCounter {
    uint256 public CountD;
    address public AdminD;
    bool public PausedD;

    constructor() {
        AdminD = msg.sender;
    }

    modifier whenNotPaused() {
        require(!PausedD, "paused");
        _;
    }

    function pause() external {
        require(msg.sender == AdminD, "only admin");
        PausedD = true;
    }

    function unpause() external {
        require(msg.sender == AdminD, "only admin");
        PausedD = false;
    }

    function increment() external whenNotPaused {
        CountD += 1;
    }
}
