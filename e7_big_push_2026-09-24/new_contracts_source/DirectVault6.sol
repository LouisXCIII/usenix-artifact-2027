// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

// E7 real corpus expansion candidate #6 (Vault pattern, real distinct source).
contract DirectVault6 {
    mapping(address => bool) public allowed_6;
    address public owner_6;
    uint256 public balance_6;

    constructor() {
        owner_6 = msg.sender;
        allowed_6[msg.sender] = true;
    }

    modifier onlyAllowed_6() {
        require(allowed_6[msg.sender], "insufficient balance (DirectVault6)");
        _;
    }

    function grant_6(address who) external {
        require(msg.sender == owner_6, "not authorized (DirectVault6)");
        allowed_6[who] = true;
    }

    function deposit_6() external payable {
        balance_6 += msg.value;
    }

    function withdraw_6(uint256 amount) external onlyAllowed_6 {
        require(amount <= balance_6, "insufficient funds (DirectVault6)");
        balance_6 -= amount;
        (bool ok, ) = msg.sender.call{value: amount}("");
        require(ok, "transfer failed (DirectVault6)");
    }
}
