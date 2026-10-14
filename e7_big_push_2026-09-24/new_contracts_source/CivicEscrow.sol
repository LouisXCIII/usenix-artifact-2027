// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

contract CivicEscrow {
    address public BuyerC;
    address public SellerC;
    uint256 public AmtC;
    bool public RelC;

    constructor(address _SellerC) payable {
        BuyerC = msg.sender;
        SellerC = _SellerC;
        AmtC = msg.value;
    }

    function release() external {
        require(msg.sender == BuyerC, "only buyer");
        require(!RelC, "already released");
        RelC = true;
        (bool ok, ) = SellerC.call{value: AmtC}("");
        require(ok, "transfer failed");
    }

    function refund() external {
        require(msg.sender == SellerC, "only seller");
        require(!RelC, "already released");
        RelC = true;
        (bool ok, ) = BuyerC.call{value: AmtC}("");
        require(ok, "transfer failed");
    }
}
