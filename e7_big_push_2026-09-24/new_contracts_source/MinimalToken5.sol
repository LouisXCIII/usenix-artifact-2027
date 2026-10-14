// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

// E7 real corpus expansion candidate #5 (Token pattern, real distinct source).
contract MinimalToken5 {
    mapping(address => uint256) public balance_5;
    uint256 public totalSupply_5;
    address public owner_5;

    constructor(uint256 initialtotalSupply_5) {
        owner_5 = msg.sender;
        totalSupply_5 = initialtotalSupply_5;
        balance_5[msg.sender] = initialtotalSupply_5;
    }

    function transfer_5(address to, uint256 value) external returns (bool) {
        require(balance_5[msg.sender] >= value, "insufficient balance (MinimalToken5)");
        balance_5[msg.sender] -= value;
        balance_5[to] += value;
        return true;
    }

    function balanceOf_5(address who) external view returns (uint256) {
        return balance_5[who];
    }
}
