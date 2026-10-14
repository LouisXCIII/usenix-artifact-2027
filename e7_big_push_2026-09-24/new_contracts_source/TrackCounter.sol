// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

contract TrackCounter {
    uint256 public CountM;
    address public AdminM;
    bool public PausedM;

    constructor() {
        AdminM = msg.sender;
    }

    modifier whenNotPaused() {
        require(!PausedM, "paused");
        _;
    }

    function pause() external {
        require(msg.sender == AdminM, "only admin");
        PausedM = true;
    }

    function unpause() external {
        require(msg.sender == AdminM, "only admin");
        PausedM = false;
    }

    function increment() external whenNotPaused {
        CountM += 1;
    }
}
