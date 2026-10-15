// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

contract BeaconCounter {
    uint256 public CountF;
    address public AdminF;
    bool public PausedF;

    constructor() {
        AdminF = msg.sender;
    }

    modifier whenNotPaused() {
        require(!PausedF, "paused");
        _;
    }

    function pause() external {
        require(msg.sender == AdminF, "only admin");
        PausedF = true;
    }

    function unpause() external {
        require(msg.sender == AdminF, "only admin");
        PausedF = false;
    }

    function increment() external whenNotPaused {
        CountF += 1;
    }
}
