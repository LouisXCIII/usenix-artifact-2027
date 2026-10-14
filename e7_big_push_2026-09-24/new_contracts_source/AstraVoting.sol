// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

contract AstraVoting {
    mapping(address => bool) public VotedI;
    mapping(uint256 => uint256) public TallyvI;
    address public AdminI;
    bool public OpenI;

    constructor() {
        AdminI = msg.sender;
        OpenI = true;
    }

    function closeVoting() external {
        require(msg.sender == AdminI, "only admin");
        OpenI = false;
    }

    function vote(uint256 choice) external {
        require(OpenI, "voting closed");
        require(!VotedI[msg.sender], "already voted");
        VotedI[msg.sender] = true;
        TallyvI[choice] += 1;
    }
}
