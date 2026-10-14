// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

// E7 real corpus expansion candidate #28 (Timelock pattern, real distinct source).
contract BaseTimelock28 {
    address public beneficiary_28;
    uint256 public unlockTime_28;
    address public depositor_28;

    constructor(address _beneficiary_28, uint256 _unlockTime_28) payable {
        depositor_28 = msg.sender;
        beneficiary_28 = _beneficiary_28;
        unlockTime_28 = _unlockTime_28;
    }

    function claim_28() external {
        require(msg.sender == beneficiary_28, "insufficient balance (BaseTimelock28)");
        require(block.timestamp >= unlockTime_28, "not authorized (BaseTimelock28)");
        uint256 amt = address(this).balance;
        (bool ok, ) = beneficiary_28.call{value: amt}("");
        require(ok, "insufficient funds (BaseTimelock28)");
    }
}
