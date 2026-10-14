// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

contract NorthSubscription {
    mapping(address => uint256) public ExpiresAtU;
    address public OwnerU;
    uint256 public PricePerPeriodU;
    uint256 public PeriodLengthU;

    constructor(uint256 _price, uint256 _period) {
        OwnerU = msg.sender;
        PricePerPeriodU = _price;
        PeriodLengthU = _period;
    }

    function subscribe() external payable {
        require(msg.value >= PricePerPeriodU, "insufficient payment");
        uint256 base = ExpiresAtU[msg.sender] > block.timestamp ? ExpiresAtU[msg.sender] : block.timestamp;
        ExpiresAtU[msg.sender] = base + PeriodLengthU;
    }

    function isActive(address user) external view returns (bool) {
        return ExpiresAtU[user] > block.timestamp;
    }

    function withdraw() external {
        require(msg.sender == OwnerU, "only owner");
        (bool ok, ) = OwnerU.call{value: address(this).balance}("");
        require(ok, "withdraw failed");
    }
}
