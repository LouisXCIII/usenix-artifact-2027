// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

contract WestVault {
    mapping(address => bool) public AllowedX;
    address public OwnerX;
    uint256 public BalX;

    constructor() {
        OwnerX = msg.sender;
        AllowedX[msg.sender] = true;
    }

    modifier onlyAllowed() {
        require(AllowedX[msg.sender], "not allowed");
        _;
    }

    function grant(address who) external {
        require(msg.sender == OwnerX, "only owner");
        AllowedX[who] = true;
    }

    function deposit() external payable {
        BalX += msg.value;
    }

    function withdraw(uint256 amount) external onlyAllowed {
        require(amount <= BalX, "insufficient funds");
        BalX -= amount;
        (bool ok, ) = msg.sender.call{value: amount}("");
        require(ok, "transfer failed");
    }
}
