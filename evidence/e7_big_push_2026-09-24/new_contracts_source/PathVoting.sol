// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

contract PathVoting {
    mapping(address => bool) public VotedK;
    mapping(uint256 => uint256) public TallyvK;
    address public AdminK;
    bool public OpenK;

    constructor() {
        AdminK = msg.sender;
        OpenK = true;
    }

    function closeVoting() external {
        require(msg.sender == AdminK, "only admin");
        OpenK = false;
    }

    function vote(uint256 choice) external {
        require(OpenK, "voting closed");
        require(!VotedK[msg.sender], "already voted");
        VotedK[msg.sender] = true;
        TallyvK[choice] += 1;
    }
}
