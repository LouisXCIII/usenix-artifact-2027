// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

// E7 real corpus expansion candidate #8 (Timelock pattern, real distinct source).
contract FixedTimelock8 {
    address public beneficiary_8;
    uint256 public unlockTime_8;
    address public depositor_8;

    constructor(address _beneficiary_8, uint256 _unlockTime_8) payable {
        depositor_8 = msg.sender;
        beneficiary_8 = _beneficiary_8;
        unlockTime_8 = _unlockTime_8;
    }

    function claim_8() external {
        require(msg.sender == beneficiary_8, "insufficient balance (FixedTimelock8)");
        require(block.timestamp >= unlockTime_8, "not authorized (FixedTimelock8)");
        uint256 amt = address(this).balance;
        (bool ok, ) = beneficiary_8.call{value: amt}("");
        require(ok, "insufficient funds (FixedTimelock8)");
    }
}
