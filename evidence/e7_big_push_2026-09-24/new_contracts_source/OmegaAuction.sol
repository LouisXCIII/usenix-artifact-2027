// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

contract OmegaAuction {
    address public SellerE;
    uint256 public EndTimeE;
    address public HighestBidderE;
    uint256 public HighestBidE;
    mapping(address => uint256) public PendingReturnsE;
    bool public EndedE;

    constructor(uint256 biddingTime) {
        SellerE = msg.sender;
        EndTimeE = block.timestamp + biddingTime;
    }

    function bid() external payable {
        require(block.timestamp < EndTimeE, "auction ended");
        require(msg.value > HighestBidE, "bid too low");
        if (HighestBidE != 0) {
            PendingReturnsE[HighestBidderE] += HighestBidE;
        }
        HighestBidderE = msg.sender;
        HighestBidE = msg.value;
    }

    function withdraw() external returns (bool) {
        uint256 amount = PendingReturnsE[msg.sender];
        if (amount > 0) {
            PendingReturnsE[msg.sender] = 0;
            (bool ok, ) = msg.sender.call{value: amount}("");
            if (!ok) {
                PendingReturnsE[msg.sender] = amount;
                return false;
            }
        }
        return true;
    }

    function endAuction() external {
        require(block.timestamp >= EndTimeE, "auction not yet ended");
        require(!EndedE, "already ended");
        EndedE = true;
        (bool ok, ) = SellerE.call{value: HighestBidE}("");
        require(ok, "transfer to seller failed");
    }
}
