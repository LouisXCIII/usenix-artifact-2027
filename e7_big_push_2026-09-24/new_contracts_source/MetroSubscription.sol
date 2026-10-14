// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

contract MetroSubscription {
    mapping(address => uint256) public ExpiresAtB;
    address public OwnerB;
    uint256 public PricePerPeriodB;
    uint256 public PeriodLengthB;

    constructor(uint256 _price, uint256 _period) {
        OwnerB = msg.sender;
        PricePerPeriodB = _price;
        PeriodLengthB = _period;
    }

    function subscribe() external payable {
        require(msg.value >= PricePerPeriodB, "insufficient payment");
        uint256 base = ExpiresAtB[msg.sender] > block.timestamp ? ExpiresAtB[msg.sender] : block.timestamp;
        ExpiresAtB[msg.sender] = base + PeriodLengthB;
    }

    function isActive(address user) external view returns (bool) {
        return ExpiresAtB[user] > block.timestamp;
    }

    function withdraw() external {
        require(msg.sender == OwnerB, "only owner");
        (bool ok, ) = OwnerB.call{value: address(this).balance}("");
        require(ok, "withdraw failed");
    }
}
