// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

contract BridgeEscrow {
    address public BuyerJ;
    address public SellerJ;
    uint256 public AmtJ;
    bool public RelJ;

    constructor(address _SellerJ) payable {
        BuyerJ = msg.sender;
        SellerJ = _SellerJ;
        AmtJ = msg.value;
    }

    function release() external {
        require(msg.sender == BuyerJ, "only buyer");
        require(!RelJ, "already released");
        RelJ = true;
        (bool ok, ) = SellerJ.call{value: AmtJ}("");
        require(ok, "transfer failed");
    }

    function refund() external {
        require(msg.sender == SellerJ, "only seller");
        require(!RelJ, "already released");
        RelJ = true;
        (bool ok, ) = BuyerJ.call{value: AmtJ}("");
        require(ok, "transfer failed");
    }
}
