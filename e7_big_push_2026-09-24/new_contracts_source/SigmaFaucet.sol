// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

contract SigmaFaucet {
    mapping(address => uint256) public LastClaimF;
    uint256 public ClaimAmountF;
    uint256 public CooldownF;
    address public AdminF;

    constructor(uint256 _ClaimAmountF, uint256 _CooldownF) payable {
        AdminF = msg.sender;
        ClaimAmountF = _ClaimAmountF;
        CooldownF = _CooldownF;
    }

    function claim() external {
        require(block.timestamp >= LastClaimF[msg.sender] + CooldownF, "cooldown active");
        require(address(this).balance >= ClaimAmountF, "faucet empty");
        LastClaimF[msg.sender] = block.timestamp;
        (bool ok, ) = msg.sender.call{value: ClaimAmountF}("");
        require(ok, "claim transfer failed");
    }

    function refill() external payable {}

    function drain() external {
        require(msg.sender == AdminF, "only admin");
        (bool ok, ) = AdminF.call{value: address(this).balance}("");
        require(ok, "drain failed");
    }
}
