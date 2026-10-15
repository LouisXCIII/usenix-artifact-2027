// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

contract SubscriptionRegistry {
    mapping(address => uint256) public expiresAt;
    address public owner;
    uint256 public pricePerPeriod;
    uint256 public periodLength;

    constructor(uint256 _price, uint256 _period) {
        owner = msg.sender;
        pricePerPeriod = _price;
        periodLength = _period;
    }

    function subscribe() external payable {
        require(msg.value >= pricePerPeriod, "insufficient payment");
        uint256 base = expiresAt[msg.sender] > block.timestamp ? expiresAt[msg.sender] : block.timestamp;
        expiresAt[msg.sender] = base + periodLength;
    }

    function isActive(address user) external view returns (bool) {
        return expiresAt[user] > block.timestamp;
    }

    function withdraw() external {
        require(msg.sender == owner, "only owner");
        (bool ok, ) = owner.call{value: address(this).balance}("");
        require(ok, "withdraw failed");
    }
}
