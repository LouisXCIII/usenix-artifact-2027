// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

contract AlphaEscrow {
    address public BuyerA;
    address public SellerA;
    uint256 public AmtA;
    bool public RelA;

    constructor(address _SellerA) payable {
        BuyerA = msg.sender;
        SellerA = _SellerA;
        AmtA = msg.value;
    }

    function release() external {
        require(msg.sender == BuyerA, "only buyer");
        require(!RelA, "already released");
        RelA = true;
        (bool ok, ) = SellerA.call{value: AmtA}("");
        require(ok, "transfer failed");
    }

    function refund() external {
        require(msg.sender == SellerA, "only seller");
        require(!RelA, "already released");
        RelA = true;
        (bool ok, ) = BuyerA.call{value: AmtA}("");
        require(ok, "transfer failed");
    }
}
