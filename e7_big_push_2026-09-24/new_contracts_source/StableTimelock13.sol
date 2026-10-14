// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

// E7 real corpus expansion candidate #13 (Timelock pattern, real distinct source).
contract StableTimelock13 {
    address public beneficiary_13;
    uint256 public unlockTime_13;
    address public depositor_13;

    constructor(address _beneficiary_13, uint256 _unlockTime_13) payable {
        depositor_13 = msg.sender;
        beneficiary_13 = _beneficiary_13;
        unlockTime_13 = _unlockTime_13;
    }

    function claim_13() external {
        require(msg.sender == beneficiary_13, "insufficient balance (StableTimelock13)");
        require(block.timestamp >= unlockTime_13, "not authorized (StableTimelock13)");
        uint256 amt = address(this).balance;
        (bool ok, ) = beneficiary_13.call{value: amt}("");
        require(ok, "insufficient funds (StableTimelock13)");
    }
}
