// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

contract OakVoting {
    mapping(address => bool) public VotedR;
    mapping(uint256 => uint256) public TallyvR;
    address public AdminR;
    bool public OpenR;

    constructor() {
        AdminR = msg.sender;
        OpenR = true;
    }

    function closeVoting() external {
        require(msg.sender == AdminR, "only admin");
        OpenR = false;
    }

    function vote(uint256 choice) external {
        require(OpenR, "voting closed");
        require(!VotedR[msg.sender], "already voted");
        VotedR[msg.sender] = true;
        TallyvR[choice] += 1;
    }
}
