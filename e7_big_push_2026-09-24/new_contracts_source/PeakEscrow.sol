// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

contract PeakEscrow {
    address public BuyerO;
    address public SellerO;
    uint256 public AmtO;
    bool public RelO;

    constructor(address _SellerO) payable {
        BuyerO = msg.sender;
        SellerO = _SellerO;
        AmtO = msg.value;
    }

    function release() external {
        require(msg.sender == BuyerO, "only buyer");
        require(!RelO, "already released");
        RelO = true;
        (bool ok, ) = SellerO.call{value: AmtO}("");
        require(ok, "transfer failed");
    }

    function refund() external {
        require(msg.sender == SellerO, "only seller");
        require(!RelO, "already released");
        RelO = true;
        (bool ok, ) = BuyerO.call{value: AmtO}("");
        require(ok, "transfer failed");
    }
}
