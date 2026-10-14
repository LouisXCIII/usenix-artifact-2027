// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

contract HarborAuction {
    address public SellerG;
    uint256 public EndTimeG;
    address public HighestBidderG;
    uint256 public HighestBidG;
    mapping(address => uint256) public PendingReturnsG;
    bool public EndedG;

    constructor(uint256 biddingTime) {
        SellerG = msg.sender;
        EndTimeG = block.timestamp + biddingTime;
    }

    function bid() external payable {
        require(block.timestamp < EndTimeG, "auction ended");
        require(msg.value > HighestBidG, "bid too low");
        if (HighestBidG != 0) {
            PendingReturnsG[HighestBidderG] += HighestBidG;
        }
        HighestBidderG = msg.sender;
        HighestBidG = msg.value;
    }

    function withdraw() external returns (bool) {
        uint256 amount = PendingReturnsG[msg.sender];
        if (amount > 0) {
            PendingReturnsG[msg.sender] = 0;
            (bool ok, ) = msg.sender.call{value: amount}("");
            if (!ok) {
                PendingReturnsG[msg.sender] = amount;
                return false;
            }
        }
        return true;
    }

    function endAuction() external {
        require(block.timestamp >= EndTimeG, "auction not yet ended");
        require(!EndedG, "already ended");
        EndedG = true;
        (bool ok, ) = SellerG.call{value: HighestBidG}("");
        require(ok, "transfer to seller failed");
    }
}
