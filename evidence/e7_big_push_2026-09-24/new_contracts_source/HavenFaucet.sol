// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

contract HavenFaucet {
    mapping(address => uint256) public LastClaimT;
    uint256 public ClaimAmountT;
    uint256 public CooldownT;
    address public AdminT;

    constructor(uint256 _ClaimAmountT, uint256 _CooldownT) payable {
        AdminT = msg.sender;
        ClaimAmountT = _ClaimAmountT;
        CooldownT = _CooldownT;
    }

    function claim() external {
        require(block.timestamp >= LastClaimT[msg.sender] + CooldownT, "cooldown active");
        require(address(this).balance >= ClaimAmountT, "faucet empty");
        LastClaimT[msg.sender] = block.timestamp;
        (bool ok, ) = msg.sender.call{value: ClaimAmountT}("");
        require(ok, "claim transfer failed");
    }

    function refill() external payable {}

    function drain() external {
        require(msg.sender == AdminT, "only admin");
        (bool ok, ) = AdminT.call{value: address(this).balance}("");
        require(ok, "drain failed");
    }
}
