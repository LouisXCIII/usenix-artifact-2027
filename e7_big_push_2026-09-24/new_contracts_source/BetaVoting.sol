// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

contract BetaVoting {
    mapping(address => bool) public VotedB;
    mapping(uint256 => uint256) public TallyvB;
    address public AdminB;
    bool public OpenB;

    constructor() {
        AdminB = msg.sender;
        OpenB = true;
    }

    function closeVoting() external {
        require(msg.sender == AdminB, "only admin");
        OpenB = false;
    }

    function vote(uint256 choice) external {
        require(OpenB, "voting closed");
        require(!VotedB[msg.sender], "already voted");
        VotedB[msg.sender] = true;
        TallyvB[choice] += 1;
    }
}
