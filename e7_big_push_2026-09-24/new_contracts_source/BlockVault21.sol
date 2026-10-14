// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

// E7 real corpus expansion candidate #21 (Vault pattern, real distinct source).
contract BlockVault21 {
    mapping(address => bool) public allowed_21;
    address public owner_21;
    uint256 public balance_21;

    constructor() {
        owner_21 = msg.sender;
        allowed_21[msg.sender] = true;
    }

    modifier onlyAllowed_21() {
        require(allowed_21[msg.sender], "insufficient balance (BlockVault21)");
        _;
    }

    function grant_21(address who) external {
        require(msg.sender == owner_21, "not authorized (BlockVault21)");
        allowed_21[who] = true;
    }

    function deposit_21() external payable {
        balance_21 += msg.value;
    }

    function withdraw_21(uint256 amount) external onlyAllowed_21 {
        require(amount <= balance_21, "insufficient funds (BlockVault21)");
        balance_21 -= amount;
        (bool ok, ) = msg.sender.call{value: amount}("");
        require(ok, "transfer failed (BlockVault21)");
    }
}
