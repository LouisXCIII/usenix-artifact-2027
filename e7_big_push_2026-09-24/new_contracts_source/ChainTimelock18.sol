// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

// E7 real corpus expansion candidate #18 (Timelock pattern, real distinct source).
contract ChainTimelock18 {
    address public beneficiary_18;
    uint256 public unlockTime_18;
    address public depositor_18;

    constructor(address _beneficiary_18, uint256 _unlockTime_18) payable {
        depositor_18 = msg.sender;
        beneficiary_18 = _beneficiary_18;
        unlockTime_18 = _unlockTime_18;
    }

    function claim_18() external {
        require(msg.sender == beneficiary_18, "insufficient balance (ChainTimelock18)");
        require(block.timestamp >= unlockTime_18, "not authorized (ChainTimelock18)");
        uint256 amt = address(this).balance;
        (bool ok, ) = beneficiary_18.call{value: amt}("");
        require(ok, "insufficient funds (ChainTimelock18)");
    }
}
