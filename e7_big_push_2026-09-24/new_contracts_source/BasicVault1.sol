// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

// E7 real corpus expansion candidate #1 (Vault pattern, real distinct source).
contract BasicVault1 {
    mapping(address => bool) public allowed_1;
    address public owner_1;
    uint256 public balance_1;

    constructor() {
        owner_1 = msg.sender;
        allowed_1[msg.sender] = true;
    }

    modifier onlyAllowed_1() {
        require(allowed_1[msg.sender], "insufficient balance (BasicVault1)");
        _;
    }

    function grant_1(address who) external {
        require(msg.sender == owner_1, "not authorized (BasicVault1)");
        allowed_1[who] = true;
    }

    function deposit_1() external payable {
        balance_1 += msg.value;
    }

    function withdraw_1(uint256 amount) external onlyAllowed_1 {
        require(amount <= balance_1, "insufficient funds (BasicVault1)");
        balance_1 -= amount;
        (bool ok, ) = msg.sender.call{value: amount}("");
        require(ok, "transfer failed (BasicVault1)");
    }
}
