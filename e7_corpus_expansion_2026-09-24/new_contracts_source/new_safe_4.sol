// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

contract AccessControlledVault {
    mapping(address => bool) public allowed;
    address public owner;
    uint256 public balance;

    constructor() {
        owner = msg.sender;
        allowed[msg.sender] = true;
    }

    modifier onlyAllowed() {
        require(allowed[msg.sender], "not allowed");
        _;
    }

    function grant(address who) external {
        require(msg.sender == owner, "only owner");
        allowed[who] = true;
    }

    function deposit() external payable {
        balance += msg.value;
    }

    function withdraw(uint256 amount) external onlyAllowed {
        require(amount <= balance, "insufficient funds");
        balance -= amount;
        (bool ok, ) = msg.sender.call{value: amount}("");
        require(ok, "transfer failed");
    }
}
