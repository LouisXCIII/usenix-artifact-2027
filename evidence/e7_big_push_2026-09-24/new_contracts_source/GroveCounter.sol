// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

contract GroveCounter {
    uint256 public CountR;
    address public AdminR;
    bool public PausedR;

    constructor() {
        AdminR = msg.sender;
    }

    modifier whenNotPaused() {
        require(!PausedR, "paused");
        _;
    }

    function pause() external {
        require(msg.sender == AdminR, "only admin");
        PausedR = true;
    }

    function unpause() external {
        require(msg.sender == AdminR, "only admin");
        PausedR = false;
    }

    function increment() external whenNotPaused {
        CountR += 1;
    }
}
