// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

contract EastVoting {
    mapping(address => bool) public VotedW;
    mapping(uint256 => uint256) public TallyvW;
    address public AdminW;
    bool public OpenW;

    constructor() {
        AdminW = msg.sender;
        OpenW = true;
    }

    function closeVoting() external {
        require(msg.sender == AdminW, "only admin");
        OpenW = false;
    }

    function vote(uint256 choice) external {
        require(OpenW, "voting closed");
        require(!VotedW[msg.sender], "already voted");
        VotedW[msg.sender] = true;
        TallyvW[choice] += 1;
    }
}
