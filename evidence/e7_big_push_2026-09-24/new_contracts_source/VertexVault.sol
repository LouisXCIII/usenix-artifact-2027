// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

contract VertexVault {
    mapping(address => bool) public AllowedJ;
    address public OwnerJ;
    uint256 public BalJ;

    constructor() {
        OwnerJ = msg.sender;
        AllowedJ[msg.sender] = true;
    }

    modifier onlyAllowed() {
        require(AllowedJ[msg.sender], "not allowed");
        _;
    }

    function grant(address who) external {
        require(msg.sender == OwnerJ, "only owner");
        AllowedJ[who] = true;
    }

    function deposit() external payable {
        BalJ += msg.value;
    }

    function withdraw(uint256 amount) external onlyAllowed {
        require(amount <= BalJ, "insufficient funds");
        BalJ -= amount;
        (bool ok, ) = msg.sender.call{value: amount}("");
        require(ok, "transfer failed");
    }
}
