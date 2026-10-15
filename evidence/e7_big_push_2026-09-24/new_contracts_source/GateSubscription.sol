// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

contract GateSubscription {
    mapping(address => uint256) public ExpiresAtI;
    address public OwnerI;
    uint256 public PricePerPeriodI;
    uint256 public PeriodLengthI;

    constructor(uint256 _price, uint256 _period) {
        OwnerI = msg.sender;
        PricePerPeriodI = _price;
        PeriodLengthI = _period;
    }

    function subscribe() external payable {
        require(msg.value >= PricePerPeriodI, "insufficient payment");
        uint256 base = ExpiresAtI[msg.sender] > block.timestamp ? ExpiresAtI[msg.sender] : block.timestamp;
        ExpiresAtI[msg.sender] = base + PeriodLengthI;
    }

    function isActive(address user) external view returns (bool) {
        return ExpiresAtI[user] > block.timestamp;
    }

    function withdraw() external {
        require(msg.sender == OwnerI, "only owner");
        (bool ok, ) = OwnerI.call{value: address(this).balance}("");
        require(ok, "withdraw failed");
    }
}
