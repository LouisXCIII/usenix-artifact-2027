// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

contract PineVault {
    mapping(address => bool) public AllowedS;
    address public OwnerS;
    uint256 public BalS;

    constructor() {
        OwnerS = msg.sender;
        AllowedS[msg.sender] = true;
    }

    modifier onlyAllowed() {
        require(AllowedS[msg.sender], "not allowed");
        _;
    }

    function grant(address who) external {
        require(msg.sender == OwnerS, "only owner");
        AllowedS[who] = true;
    }

    function deposit() external payable {
        BalS += msg.value;
    }

    function withdraw(uint256 amount) external onlyAllowed {
        require(amount <= BalS, "insufficient funds");
        BalS -= amount;
        (bool ok, ) = msg.sender.call{value: amount}("");
        require(ok, "transfer failed");
    }
}
