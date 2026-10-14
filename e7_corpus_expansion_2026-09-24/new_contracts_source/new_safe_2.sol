// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

contract VotingRegistry {
    mapping(address => bool) public hasVoted;
    mapping(uint256 => uint256) public tally;
    address public admin;
    bool public open;

    constructor() {
        admin = msg.sender;
        open = true;
    }

    function closeVoting() external {
        require(msg.sender == admin, "only admin");
        open = false;
    }

    function vote(uint256 choice) external {
        require(open, "voting closed");
        require(!hasVoted[msg.sender], "already voted");
        hasVoted[msg.sender] = true;
        tally[choice] += 1;
    }
}
