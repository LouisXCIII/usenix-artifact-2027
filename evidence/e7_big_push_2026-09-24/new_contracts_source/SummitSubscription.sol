// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

contract SummitSubscription {
    mapping(address => uint256) public ExpiresAtN;
    address public OwnerN;
    uint256 public PricePerPeriodN;
    uint256 public PeriodLengthN;

    constructor(uint256 _price, uint256 _period) {
        OwnerN = msg.sender;
        PricePerPeriodN = _price;
        PeriodLengthN = _period;
    }

    function subscribe() external payable {
        require(msg.value >= PricePerPeriodN, "insufficient payment");
        uint256 base = ExpiresAtN[msg.sender] > block.timestamp ? ExpiresAtN[msg.sender] : block.timestamp;
        ExpiresAtN[msg.sender] = base + PeriodLengthN;
    }

    function isActive(address user) external view returns (bool) {
        return ExpiresAtN[user] > block.timestamp;
    }

    function withdraw() external {
        require(msg.sender == OwnerN, "only owner");
        (bool ok, ) = OwnerN.call{value: address(this).balance}("");
        require(ok, "withdraw failed");
    }
}
