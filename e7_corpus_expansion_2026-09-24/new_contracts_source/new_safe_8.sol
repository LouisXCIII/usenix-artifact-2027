// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

contract PausableCounter {
    uint256 public count;
    address public admin;
    bool public paused;

    constructor() {
        admin = msg.sender;
    }

    modifier whenNotPaused() {
        require(!paused, "paused");
        _;
    }

    function pause() external {
        require(msg.sender == admin, "only admin");
        paused = true;
    }

    function unpause() external {
        require(msg.sender == admin, "only admin");
        paused = false;
    }

    function increment() external whenNotPaused {
        count += 1;
    }
}
