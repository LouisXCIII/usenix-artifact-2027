// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

contract AnchorVault {
    mapping(address => bool) public AllowedE;
    address public OwnerE;
    uint256 public BalE;

    constructor() {
        OwnerE = msg.sender;
        AllowedE[msg.sender] = true;
    }

    modifier onlyAllowed() {
        require(AllowedE[msg.sender], "not allowed");
        _;
    }

    function grant(address who) external {
        require(msg.sender == OwnerE, "only owner");
        AllowedE[who] = true;
    }

    function deposit() external payable {
        BalE += msg.value;
    }

    function withdraw(uint256 amount) external onlyAllowed {
        require(amount <= BalE, "insufficient funds");
        BalE -= amount;
        (bool ok, ) = msg.sender.call{value: amount}("");
        require(ok, "transfer failed");
    }
}
