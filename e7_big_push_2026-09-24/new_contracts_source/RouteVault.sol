// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

contract RouteVault {
    mapping(address => bool) public AllowedL;
    address public OwnerL;
    uint256 public BalL;

    constructor() {
        OwnerL = msg.sender;
        AllowedL[msg.sender] = true;
    }

    modifier onlyAllowed() {
        require(AllowedL[msg.sender], "not allowed");
        _;
    }

    function grant(address who) external {
        require(msg.sender == OwnerL, "only owner");
        AllowedL[who] = true;
    }

    function deposit() external payable {
        BalL += msg.value;
    }

    function withdraw(uint256 amount) external onlyAllowed {
        require(amount <= BalL, "insufficient funds");
        BalL -= amount;
        (bool ok, ) = msg.sender.call{value: amount}("");
        require(ok, "transfer failed");
    }
}
