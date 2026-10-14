// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

contract GammaVault {
    mapping(address => bool) public AllowedC;
    address public OwnerC;
    uint256 public BalC;

    constructor() {
        OwnerC = msg.sender;
        AllowedC[msg.sender] = true;
    }

    modifier onlyAllowed() {
        require(AllowedC[msg.sender], "not allowed");
        _;
    }

    function grant(address who) external {
        require(msg.sender == OwnerC, "only owner");
        AllowedC[who] = true;
    }

    function deposit() external payable {
        BalC += msg.value;
    }

    function withdraw(uint256 amount) external onlyAllowed {
        require(amount <= BalC, "insufficient funds");
        BalC -= amount;
        (bool ok, ) = msg.sender.call{value: amount}("");
        require(ok, "transfer failed");
    }
}
