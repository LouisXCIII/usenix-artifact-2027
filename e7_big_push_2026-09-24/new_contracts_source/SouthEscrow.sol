// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

contract SouthEscrow {
    address public BuyerV;
    address public SellerV;
    uint256 public AmtV;
    bool public RelV;

    constructor(address _SellerV) payable {
        BuyerV = msg.sender;
        SellerV = _SellerV;
        AmtV = msg.value;
    }

    function release() external {
        require(msg.sender == BuyerV, "only buyer");
        require(!RelV, "already released");
        RelV = true;
        (bool ok, ) = SellerV.call{value: AmtV}("");
        require(ok, "transfer failed");
    }

    function refund() external {
        require(msg.sender == SellerV, "only seller");
        require(!RelV, "already released");
        RelV = true;
        (bool ok, ) = BuyerV.call{value: AmtV}("");
        require(ok, "transfer failed");
    }
}
