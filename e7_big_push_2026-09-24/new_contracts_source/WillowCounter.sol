// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

contract WillowCounter {
    uint256 public CountT;
    address public AdminT;
    bool public PausedT;

    constructor() {
        AdminT = msg.sender;
    }

    modifier whenNotPaused() {
        require(!PausedT, "paused");
        _;
    }

    function pause() external {
        require(msg.sender == AdminT, "only admin");
        PausedT = true;
    }

    function unpause() external {
        require(msg.sender == AdminT, "only admin");
        PausedT = false;
    }

    function increment() external whenNotPaused {
        CountT += 1;
    }
}
