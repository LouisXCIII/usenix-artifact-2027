// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

contract AsterFaucet {
    mapping(address => uint256) public LastClaimO;
    uint256 public ClaimAmountO;
    uint256 public CooldownO;
    address public AdminO;

    constructor(uint256 _ClaimAmountO, uint256 _CooldownO) payable {
        AdminO = msg.sender;
        ClaimAmountO = _ClaimAmountO;
        CooldownO = _CooldownO;
    }

    function claim() external {
        require(block.timestamp >= LastClaimO[msg.sender] + CooldownO, "cooldown active");
        require(address(this).balance >= ClaimAmountO, "faucet empty");
        LastClaimO[msg.sender] = block.timestamp;
        (bool ok, ) = msg.sender.call{value: ClaimAmountO}("");
        require(ok, "claim transfer failed");
    }

    function refill() external payable {}

    function drain() external {
        require(msg.sender == AdminO, "only admin");
        (bool ok, ) = AdminO.call{value: address(this).balance}("");
        require(ok, "drain failed");
    }
}
