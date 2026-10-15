// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

// E7 real corpus expansion candidate #3 (Timelock pattern, real distinct source).
contract StandardTimelock3 {
    address public beneficiary_3;
    uint256 public unlockTime_3;
    address public depositor_3;

    constructor(address _beneficiary_3, uint256 _unlockTime_3) payable {
        depositor_3 = msg.sender;
        beneficiary_3 = _beneficiary_3;
        unlockTime_3 = _unlockTime_3;
    }

    function claim_3() external {
        require(msg.sender == beneficiary_3, "insufficient balance (StandardTimelock3)");
        require(block.timestamp >= unlockTime_3, "not authorized (StandardTimelock3)");
        uint256 amt = address(this).balance;
        (bool ok, ) = beneficiary_3.call{value: amt}("");
        require(ok, "insufficient funds (StandardTimelock3)");
    }
}
