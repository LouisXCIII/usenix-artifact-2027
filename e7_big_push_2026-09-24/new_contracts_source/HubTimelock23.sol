// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

// E7 real corpus expansion candidate #23 (Timelock pattern, real distinct source).
contract HubTimelock23 {
    address public beneficiary_23;
    uint256 public unlockTime_23;
    address public depositor_23;

    constructor(address _beneficiary_23, uint256 _unlockTime_23) payable {
        depositor_23 = msg.sender;
        beneficiary_23 = _beneficiary_23;
        unlockTime_23 = _unlockTime_23;
    }

    function claim_23() external {
        require(msg.sender == beneficiary_23, "insufficient balance (HubTimelock23)");
        require(block.timestamp >= unlockTime_23, "not authorized (HubTimelock23)");
        uint256 amt = address(this).balance;
        (bool ok, ) = beneficiary_23.call{value: amt}("");
        require(ok, "insufficient funds (HubTimelock23)");
    }
}
