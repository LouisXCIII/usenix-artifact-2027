// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

contract CrestFaucet {
    mapping(address => uint256) public LastClaimM;
    uint256 public ClaimAmountM;
    uint256 public CooldownM;
    address public AdminM;

    constructor(uint256 _ClaimAmountM, uint256 _CooldownM) payable {
        AdminM = msg.sender;
        ClaimAmountM = _ClaimAmountM;
        CooldownM = _CooldownM;
    }

    function claim() external {
        require(block.timestamp >= LastClaimM[msg.sender] + CooldownM, "cooldown active");
        require(address(this).balance >= ClaimAmountM, "faucet empty");
        LastClaimM[msg.sender] = block.timestamp;
        (bool ok, ) = msg.sender.call{value: ClaimAmountM}("");
        require(ok, "claim transfer failed");
    }

    function refill() external payable {}

    function drain() external {
        require(msg.sender == AdminM, "only admin");
        (bool ok, ) = AdminM.call{value: address(this).balance}("");
        require(ok, "drain failed");
    }
}
