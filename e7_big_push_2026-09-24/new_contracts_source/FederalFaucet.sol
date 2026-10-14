// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

contract FederalFaucet {
    mapping(address => uint256) public LastClaimA;
    uint256 public ClaimAmountA;
    uint256 public CooldownA;
    address public AdminA;

    constructor(uint256 _ClaimAmountA, uint256 _CooldownA) payable {
        AdminA = msg.sender;
        ClaimAmountA = _ClaimAmountA;
        CooldownA = _CooldownA;
    }

    function claim() external {
        require(block.timestamp >= LastClaimA[msg.sender] + CooldownA, "cooldown active");
        require(address(this).balance >= ClaimAmountA, "faucet empty");
        LastClaimA[msg.sender] = block.timestamp;
        (bool ok, ) = msg.sender.call{value: ClaimAmountA}("");
        require(ok, "claim transfer failed");
    }

    function refill() external payable {}

    function drain() external {
        require(msg.sender == AdminA, "only admin");
        (bool ok, ) = AdminA.call{value: address(this).balance}("");
        require(ok, "drain failed");
    }
}
