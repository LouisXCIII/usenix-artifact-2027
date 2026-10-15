// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

contract SealedBidAuction {
    address public seller;
    uint256 public endTime;
    address public highestBidder;
    uint256 public highestBid;
    mapping(address => uint256) public pendingReturns;
    bool public ended;

    constructor(uint256 biddingTime) {
        seller = msg.sender;
        endTime = block.timestamp + biddingTime;
    }

    function bid() external payable {
        require(block.timestamp < endTime, "auction ended");
        require(msg.value > highestBid, "bid too low");
        if (highestBid != 0) {
            pendingReturns[highestBidder] += highestBid;
        }
        highestBidder = msg.sender;
        highestBid = msg.value;
    }

    function withdraw() external returns (bool) {
        uint256 amount = pendingReturns[msg.sender];
        if (amount > 0) {
            pendingReturns[msg.sender] = 0;
            (bool ok, ) = msg.sender.call{value: amount}("");
            if (!ok) {
                pendingReturns[msg.sender] = amount;
                return false;
            }
        }
        return true;
    }

    function endAuction() external {
        require(block.timestamp >= endTime, "auction not yet ended");
        require(!ended, "already ended");
        ended = true;
        (bool ok, ) = seller.call{value: highestBid}("");
        require(ok, "transfer to seller failed");
    }
}
