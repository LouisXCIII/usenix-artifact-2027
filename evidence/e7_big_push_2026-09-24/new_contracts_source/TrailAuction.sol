// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

contract TrailAuction {
    address public SellerN;
    uint256 public EndTimeN;
    address public HighestBidderN;
    uint256 public HighestBidN;
    mapping(address => uint256) public PendingReturnsN;
    bool public EndedN;

    constructor(uint256 biddingTime) {
        SellerN = msg.sender;
        EndTimeN = block.timestamp + biddingTime;
    }

    function bid() external payable {
        require(block.timestamp < EndTimeN, "auction ended");
        require(msg.value > HighestBidN, "bid too low");
        if (HighestBidN != 0) {
            PendingReturnsN[HighestBidderN] += HighestBidN;
        }
        HighestBidderN = msg.sender;
        HighestBidN = msg.value;
    }

    function withdraw() external returns (bool) {
        uint256 amount = PendingReturnsN[msg.sender];
        if (amount > 0) {
            PendingReturnsN[msg.sender] = 0;
            (bool ok, ) = msg.sender.call{value: amount}("");
            if (!ok) {
                PendingReturnsN[msg.sender] = amount;
                return false;
            }
        }
        return true;
    }

    function endAuction() external {
        require(block.timestamp >= EndTimeN, "auction not yet ended");
        require(!EndedN, "already ended");
        EndedN = true;
        (bool ok, ) = SellerN.call{value: HighestBidN}("");
        require(ok, "transfer to seller failed");
    }
}
