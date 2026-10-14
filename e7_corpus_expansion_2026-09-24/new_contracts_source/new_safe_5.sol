// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

contract TimedLock {
    address public beneficiary;
    uint256 public unlockTime;
    address public depositor;

    constructor(address _beneficiary, uint256 _unlockTime) payable {
        depositor = msg.sender;
        beneficiary = _beneficiary;
        unlockTime = _unlockTime;
    }

    function claim() external {
        require(msg.sender == beneficiary, "only beneficiary");
        require(block.timestamp >= unlockTime, "still locked");
        uint256 amt = address(this).balance;
        (bool ok, ) = beneficiary.call{value: amt}("");
        require(ok, "transfer failed");
    }
}
