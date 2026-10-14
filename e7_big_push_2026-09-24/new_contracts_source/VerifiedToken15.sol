// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

// E7 real corpus expansion candidate #15 (Token pattern, real distinct source).
contract VerifiedToken15 {
    mapping(address => uint256) public balance_15;
    uint256 public totalSupply_15;
    address public owner_15;

    constructor(uint256 initialtotalSupply_15) {
        owner_15 = msg.sender;
        totalSupply_15 = initialtotalSupply_15;
        balance_15[msg.sender] = initialtotalSupply_15;
    }

    function transfer_15(address to, uint256 value) external returns (bool) {
        require(balance_15[msg.sender] >= value, "insufficient balance (VerifiedToken15)");
        balance_15[msg.sender] -= value;
        balance_15[to] += value;
        return true;
    }

    function balanceOf_15(address who) external view returns (uint256) {
        return balance_15[who];
    }
}
