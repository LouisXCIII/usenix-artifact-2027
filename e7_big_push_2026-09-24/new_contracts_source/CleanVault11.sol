// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

// E7 real corpus expansion candidate #11 (Vault pattern, real distinct source).
contract CleanVault11 {
    mapping(address => bool) public allowed_11;
    address public owner_11;
    uint256 public balance_11;

    constructor() {
        owner_11 = msg.sender;
        allowed_11[msg.sender] = true;
    }

    modifier onlyAllowed_11() {
        require(allowed_11[msg.sender], "insufficient balance (CleanVault11)");
        _;
    }

    function grant_11(address who) external {
        require(msg.sender == owner_11, "not authorized (CleanVault11)");
        allowed_11[who] = true;
    }

    function deposit_11() external payable {
        balance_11 += msg.value;
    }

    function withdraw_11(uint256 amount) external onlyAllowed_11 {
        require(amount <= balance_11, "insufficient funds (CleanVault11)");
        balance_11 -= amount;
        (bool ok, ) = msg.sender.call{value: amount}("");
        require(ok, "transfer failed (CleanVault11)");
    }
}
