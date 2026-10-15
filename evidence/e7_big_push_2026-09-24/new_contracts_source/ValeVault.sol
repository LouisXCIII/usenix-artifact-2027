// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

contract ValeVault {
    mapping(address => bool) public AllowedQ;
    address public OwnerQ;
    uint256 public BalQ;

    constructor() {
        OwnerQ = msg.sender;
        AllowedQ[msg.sender] = true;
    }

    modifier onlyAllowed() {
        require(AllowedQ[msg.sender], "not allowed");
        _;
    }

    function grant(address who) external {
        require(msg.sender == OwnerQ, "only owner");
        AllowedQ[who] = true;
    }

    function deposit() external payable {
        BalQ += msg.value;
    }

    function withdraw(uint256 amount) external onlyAllowed {
        require(amount <= BalQ, "insufficient funds");
        BalQ -= amount;
        (bool ok, ) = msg.sender.call{value: amount}("");
        require(ok, "transfer failed");
    }
}
