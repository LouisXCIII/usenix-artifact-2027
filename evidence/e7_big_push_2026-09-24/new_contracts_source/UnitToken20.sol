// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

// E7 real corpus expansion candidate #20 (Token pattern, real distinct source).
contract UnitToken20 {
    mapping(address => uint256) public balance_20;
    uint256 public totalSupply_20;
    address public owner_20;

    constructor(uint256 initialtotalSupply_20) {
        owner_20 = msg.sender;
        totalSupply_20 = initialtotalSupply_20;
        balance_20[msg.sender] = initialtotalSupply_20;
    }

    function transfer_20(address to, uint256 value) external returns (bool) {
        require(balance_20[msg.sender] >= value, "insufficient balance (UnitToken20)");
        balance_20[msg.sender] -= value;
        balance_20[to] += value;
        return true;
    }

    function balanceOf_20(address who) external view returns (uint256) {
        return balance_20[who];
    }
}
