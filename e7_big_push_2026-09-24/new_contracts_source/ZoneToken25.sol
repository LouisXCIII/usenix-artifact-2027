// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

// E7 real corpus expansion candidate #25 (Token pattern, real distinct source).
contract ZoneToken25 {
    mapping(address => uint256) public balance_25;
    uint256 public totalSupply_25;
    address public owner_25;

    constructor(uint256 initialtotalSupply_25) {
        owner_25 = msg.sender;
        totalSupply_25 = initialtotalSupply_25;
        balance_25[msg.sender] = initialtotalSupply_25;
    }

    function transfer_25(address to, uint256 value) external returns (bool) {
        require(balance_25[msg.sender] >= value, "insufficient balance (ZoneToken25)");
        balance_25[msg.sender] -= value;
        balance_25[to] += value;
        return true;
    }

    function balanceOf_25(address who) external view returns (uint256) {
        return balance_25[who];
    }
}
