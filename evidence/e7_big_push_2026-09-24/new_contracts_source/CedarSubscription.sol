// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

contract CedarSubscription {
    mapping(address => uint256) public ExpiresAtP;
    address public OwnerP;
    uint256 public PricePerPeriodP;
    uint256 public PeriodLengthP;

    constructor(uint256 _price, uint256 _period) {
        OwnerP = msg.sender;
        PricePerPeriodP = _price;
        PeriodLengthP = _period;
    }

    function subscribe() external payable {
        require(msg.value >= PricePerPeriodP, "insufficient payment");
        uint256 base = ExpiresAtP[msg.sender] > block.timestamp ? ExpiresAtP[msg.sender] : block.timestamp;
        ExpiresAtP[msg.sender] = base + PeriodLengthP;
    }

    function isActive(address user) external view returns (bool) {
        return ExpiresAtP[user] > block.timestamp;
    }

    function withdraw() external {
        require(msg.sender == OwnerP, "only owner");
        (bool ok, ) = OwnerP.call{value: address(this).balance}("");
        require(ok, "withdraw failed");
    }
}
