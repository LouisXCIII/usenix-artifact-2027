// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

contract RateLimitedFaucet {
    mapping(address => uint256) public lastClaim;
    uint256 public claimAmount;
    uint256 public cooldown;
    address public admin;

    constructor(uint256 _claimAmount, uint256 _cooldown) payable {
        admin = msg.sender;
        claimAmount = _claimAmount;
        cooldown = _cooldown;
    }

    function claim() external {
        require(block.timestamp >= lastClaim[msg.sender] + cooldown, "cooldown active");
        require(address(this).balance >= claimAmount, "faucet empty");
        lastClaim[msg.sender] = block.timestamp;
        (bool ok, ) = msg.sender.call{value: claimAmount}("");
        require(ok, "claim transfer failed");
    }

    function refill() external payable {}

    function drain() external {
        require(msg.sender == admin, "only admin");
        (bool ok, ) = admin.call{value: address(this).balance}("");
        require(ok, "drain failed");
    }
}
