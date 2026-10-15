// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

// E7 real corpus expansion candidate #26 (Vault pattern, real distinct source).
contract RingVault26 {
    mapping(address => bool) public allowed_26;
    address public owner_26;
    uint256 public balance_26;

    constructor() {
        owner_26 = msg.sender;
        allowed_26[msg.sender] = true;
    }

    modifier onlyAllowed_26() {
        require(allowed_26[msg.sender], "insufficient balance (RingVault26)");
        _;
    }

    function grant_26(address who) external {
        require(msg.sender == owner_26, "not authorized (RingVault26)");
        allowed_26[who] = true;
    }

    function deposit_26() external payable {
        balance_26 += msg.value;
    }

    function withdraw_26(uint256 amount) external onlyAllowed_26 {
        require(amount <= balance_26, "insufficient funds (RingVault26)");
        balance_26 -= amount;
        (bool ok, ) = msg.sender.call{value: amount}("");
        require(ok, "transfer failed (RingVault26)");
    }
}
