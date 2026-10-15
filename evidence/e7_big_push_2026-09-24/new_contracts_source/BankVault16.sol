// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

// E7 real corpus expansion candidate #16 (Vault pattern, real distinct source).
contract BankVault16 {
    mapping(address => bool) public allowed_16;
    address public owner_16;
    uint256 public balance_16;

    constructor() {
        owner_16 = msg.sender;
        allowed_16[msg.sender] = true;
    }

    modifier onlyAllowed_16() {
        require(allowed_16[msg.sender], "insufficient balance (BankVault16)");
        _;
    }

    function grant_16(address who) external {
        require(msg.sender == owner_16, "not authorized (BankVault16)");
        allowed_16[who] = true;
    }

    function deposit_16() external payable {
        balance_16 += msg.value;
    }

    function withdraw_16(uint256 amount) external onlyAllowed_16 {
        require(amount <= balance_16, "insufficient funds (BankVault16)");
        balance_16 -= amount;
        (bool ok, ) = msg.sender.call{value: amount}("");
        require(ok, "transfer failed (BankVault16)");
    }
}
