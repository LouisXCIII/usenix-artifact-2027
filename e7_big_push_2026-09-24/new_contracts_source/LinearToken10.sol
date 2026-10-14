// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

// E7 real corpus expansion candidate #10 (Token pattern, real distinct source).
contract LinearToken10 {
    mapping(address => uint256) public balance_10;
    uint256 public totalSupply_10;
    address public owner_10;

    constructor(uint256 initialtotalSupply_10) {
        owner_10 = msg.sender;
        totalSupply_10 = initialtotalSupply_10;
        balance_10[msg.sender] = initialtotalSupply_10;
    }

    function transfer_10(address to, uint256 value) external returns (bool) {
        require(balance_10[msg.sender] >= value, "insufficient balance (LinearToken10)");
        balance_10[msg.sender] -= value;
        balance_10[to] += value;
        return true;
    }

    function balanceOf_10(address who) external view returns (uint256) {
        return balance_10[who];
    }
}
