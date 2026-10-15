// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

contract PortFaucet {
    mapping(address => uint256) public LastClaimH;
    uint256 public ClaimAmountH;
    uint256 public CooldownH;
    address public AdminH;

    constructor(uint256 _ClaimAmountH, uint256 _CooldownH) payable {
        AdminH = msg.sender;
        ClaimAmountH = _ClaimAmountH;
        CooldownH = _CooldownH;
    }

    function claim() external {
        require(block.timestamp >= LastClaimH[msg.sender] + CooldownH, "cooldown active");
        require(address(this).balance >= ClaimAmountH, "faucet empty");
        LastClaimH[msg.sender] = block.timestamp;
        (bool ok, ) = msg.sender.call{value: ClaimAmountH}("");
        require(ok, "claim transfer failed");
    }

    function refill() external payable {}

    function drain() external {
        require(msg.sender == AdminH, "only admin");
        (bool ok, ) = AdminH.call{value: address(this).balance}("");
        require(ok, "drain failed");
    }
}
