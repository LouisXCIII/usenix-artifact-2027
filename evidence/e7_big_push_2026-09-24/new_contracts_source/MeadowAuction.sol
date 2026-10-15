// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

contract MeadowAuction {
    address public SellerS;
    uint256 public EndTimeS;
    address public HighestBidderS;
    uint256 public HighestBidS;
    mapping(address => uint256) public PendingReturnsS;
    bool public EndedS;

    constructor(uint256 biddingTime) {
        SellerS = msg.sender;
        EndTimeS = block.timestamp + biddingTime;
    }

    function bid() external payable {
        require(block.timestamp < EndTimeS, "auction ended");
        require(msg.value > HighestBidS, "bid too low");
        if (HighestBidS != 0) {
            PendingReturnsS[HighestBidderS] += HighestBidS;
        }
        HighestBidderS = msg.sender;
        HighestBidS = msg.value;
    }

    function withdraw() external returns (bool) {
        uint256 amount = PendingReturnsS[msg.sender];
        if (amount > 0) {
            PendingReturnsS[msg.sender] = 0;
            (bool ok, ) = msg.sender.call{value: amount}("");
            if (!ok) {
                PendingReturnsS[msg.sender] = amount;
                return false;
            }
        }
        return true;
    }

    function endAuction() external {
        require(block.timestamp >= EndTimeS, "auction not yet ended");
        require(!EndedS, "already ended");
        EndedS = true;
        (bool ok, ) = SellerS.call{value: HighestBidS}("");
        require(ok, "transfer to seller failed");
    }
}
