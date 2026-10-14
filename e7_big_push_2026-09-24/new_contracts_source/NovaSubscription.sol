// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

contract NovaSubscription {
    mapping(address => uint256) public ExpiresAtG;
    address public OwnerG;
    uint256 public PricePerPeriodG;
    uint256 public PeriodLengthG;

    constructor(uint256 _price, uint256 _period) {
        OwnerG = msg.sender;
        PricePerPeriodG = _price;
        PeriodLengthG = _period;
    }

    function subscribe() external payable {
        require(msg.value >= PricePerPeriodG, "insufficient payment");
        uint256 base = ExpiresAtG[msg.sender] > block.timestamp ? ExpiresAtG[msg.sender] : block.timestamp;
        ExpiresAtG[msg.sender] = base + PeriodLengthG;
    }

    function isActive(address user) external view returns (bool) {
        return ExpiresAtG[user] > block.timestamp;
    }

    function withdraw() external {
        require(msg.sender == OwnerG, "only owner");
        (bool ok, ) = OwnerG.call{value: address(this).balance}("");
        require(ok, "withdraw failed");
    }
}
