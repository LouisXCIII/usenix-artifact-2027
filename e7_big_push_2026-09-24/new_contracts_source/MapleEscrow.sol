// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

contract MapleEscrow {
    address public BuyerQ;
    address public SellerQ;
    uint256 public AmtQ;
    bool public RelQ;

    constructor(address _SellerQ) payable {
        BuyerQ = msg.sender;
        SellerQ = _SellerQ;
        AmtQ = msg.value;
    }

    function release() external {
        require(msg.sender == BuyerQ, "only buyer");
        require(!RelQ, "already released");
        RelQ = true;
        (bool ok, ) = SellerQ.call{value: AmtQ}("");
        require(ok, "transfer failed");
    }

    function refund() external {
        require(msg.sender == SellerQ, "only seller");
        require(!RelQ, "already released");
        RelQ = true;
        (bool ok, ) = BuyerQ.call{value: AmtQ}("");
        require(ok, "transfer failed");
    }
}
