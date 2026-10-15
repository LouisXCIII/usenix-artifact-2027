// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

contract ApexAuction {
    address public SellerL;
    uint256 public EndTimeL;
    address public HighestBidderL;
    uint256 public HighestBidL;
    mapping(address => uint256) public PendingReturnsL;
    bool public EndedL;

    constructor(uint256 biddingTime) {
        SellerL = msg.sender;
        EndTimeL = block.timestamp + biddingTime;
    }

    function bid() external payable {
        require(block.timestamp < EndTimeL, "auction ended");
        require(msg.value > HighestBidL, "bid too low");
        if (HighestBidL != 0) {
            PendingReturnsL[HighestBidderL] += HighestBidL;
        }
        HighestBidderL = msg.sender;
        HighestBidL = msg.value;
    }

    function withdraw() external returns (bool) {
        uint256 amount = PendingReturnsL[msg.sender];
        if (amount > 0) {
            PendingReturnsL[msg.sender] = 0;
            (bool ok, ) = msg.sender.call{value: amount}("");
            if (!ok) {
                PendingReturnsL[msg.sender] = amount;
                return false;
            }
        }
        return true;
    }

    function endAuction() external {
        require(block.timestamp >= EndTimeL, "auction not yet ended");
        require(!EndedL, "already ended");
        EndedL = true;
        (bool ok, ) = SellerL.call{value: HighestBidL}("");
        require(ok, "transfer to seller failed");
    }
}
