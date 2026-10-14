// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

contract PrimeCounter {
    uint256 public CountK;
    address public AdminK;
    bool public PausedK;

    constructor() {
        AdminK = msg.sender;
    }

    modifier whenNotPaused() {
        require(!PausedK, "paused");
        _;
    }

    function pause() external {
        require(msg.sender == AdminK, "only admin");
        PausedK = true;
    }

    function unpause() external {
        require(msg.sender == AdminK, "only admin");
        PausedK = false;
    }

    function increment() external whenNotPaused {
        CountK += 1;
    }
}
