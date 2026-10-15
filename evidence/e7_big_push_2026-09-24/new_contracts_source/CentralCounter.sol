// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

contract CentralCounter {
    uint256 public CountY;
    address public AdminY;
    bool public PausedY;

    constructor() {
        AdminY = msg.sender;
    }

    modifier whenNotPaused() {
        require(!PausedY, "paused");
        _;
    }

    function pause() external {
        require(msg.sender == AdminY, "only admin");
        PausedY = true;
    }

    function unpause() external {
        require(msg.sender == AdminY, "only admin");
        PausedY = false;
    }

    function increment() external whenNotPaused {
        CountY += 1;
    }
}
