// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

contract TerraEscrow {
    address public BuyerH;
    address public SellerH;
    uint256 public AmtH;
    bool public RelH;

    constructor(address _SellerH) payable {
        BuyerH = msg.sender;
        SellerH = _SellerH;
        AmtH = msg.value;
    }

    function release() external {
        require(msg.sender == BuyerH, "only buyer");
        require(!RelH, "already released");
        RelH = true;
        (bool ok, ) = SellerH.call{value: AmtH}("");
        require(ok, "transfer failed");
    }

    function refund() external {
        require(msg.sender == SellerH, "only seller");
        require(!RelH, "already released");
        RelH = true;
        (bool ok, ) = BuyerH.call{value: AmtH}("");
        require(ok, "transfer failed");
    }
}
