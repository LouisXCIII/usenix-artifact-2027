// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

contract PublicVoting {
    mapping(address => bool) public VotedD;
    mapping(uint256 => uint256) public TallyvD;
    address public AdminD;
    bool public OpenD;

    constructor() {
        AdminD = msg.sender;
        OpenD = true;
    }

    function closeVoting() external {
        require(msg.sender == AdminD, "only admin");
        OpenD = false;
    }

    function vote(uint256 choice) external {
        require(OpenD, "voting closed");
        require(!VotedD[msg.sender], "already voted");
        VotedD[msg.sender] = true;
        TallyvD[choice] += 1;
    }
}
