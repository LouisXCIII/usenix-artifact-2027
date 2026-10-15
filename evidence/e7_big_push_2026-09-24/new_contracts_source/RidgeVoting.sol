// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

contract RidgeVoting {
    mapping(address => bool) public VotedP;
    mapping(uint256 => uint256) public TallyvP;
    address public AdminP;
    bool public OpenP;

    constructor() {
        AdminP = msg.sender;
        OpenP = true;
    }

    function closeVoting() external {
        require(msg.sender == AdminP, "only admin");
        OpenP = false;
    }

    function vote(uint256 choice) external {
        require(OpenP, "voting closed");
        require(!VotedP[msg.sender], "already voted");
        VotedP[msg.sender] = true;
        TallyvP[choice] += 1;
    }
}
