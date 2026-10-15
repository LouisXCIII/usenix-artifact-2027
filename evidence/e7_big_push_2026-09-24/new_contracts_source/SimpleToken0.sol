// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

// E7 real corpus expansion candidate #0 (Token pattern, real distinct source).
contract SimpleToken0 {
    mapping(address => uint256) public balance_0;
    uint256 public totalSupply_0;
    address public owner_0;

    constructor(uint256 initialtotalSupply_0) {
        owner_0 = msg.sender;
        totalSupply_0 = initialtotalSupply_0;
        balance_0[msg.sender] = initialtotalSupply_0;
    }

    function transfer_0(address to, uint256 value) external returns (bool) {
        require(balance_0[msg.sender] >= value, "insufficient balance (SimpleToken0)");
        balance_0[msg.sender] -= value;
        balance_0[to] += value;
        return true;
    }

    function balanceOf_0(address who) external view returns (uint256) {
        return balance_0[who];
    }
}
