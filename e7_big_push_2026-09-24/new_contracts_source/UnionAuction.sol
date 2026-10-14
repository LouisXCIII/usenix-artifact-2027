// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

contract UnionAuction {
    address public SellerZ;
    uint256 public EndTimeZ;
    address public HighestBidderZ;
    uint256 public HighestBidZ;
    mapping(address => uint256) public PendingReturnsZ;
    bool public EndedZ;

    constructor(uint256 biddingTime) {
        SellerZ = msg.sender;
        EndTimeZ = block.timestamp + biddingTime;
    }

    function bid() external payable {
        require(block.timestamp < EndTimeZ, "auction ended");
        require(msg.value > HighestBidZ, "bid too low");
        if (HighestBidZ != 0) {
            PendingReturnsZ[HighestBidderZ] += HighestBidZ;
        }
        HighestBidderZ = msg.sender;
        HighestBidZ = msg.value;
    }

    function withdraw() external returns (bool) {
        uint256 amount = PendingReturnsZ[msg.sender];
        if (amount > 0) {
            PendingReturnsZ[msg.sender] = 0;
            (bool ok, ) = msg.sender.call{value: amount}("");
            if (!ok) {
                PendingReturnsZ[msg.sender] = amount;
                return false;
            }
        }
        return true;
    }

    function endAuction() external {
        require(block.timestamp >= EndTimeZ, "auction not yet ended");
        require(!EndedZ, "already ended");
        EndedZ = true;
        (bool ok, ) = SellerZ.call{value: HighestBidZ}("");
        require(ok, "transfer to seller failed");
    }
}
