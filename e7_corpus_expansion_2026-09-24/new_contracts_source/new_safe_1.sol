// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

contract SimpleEscrow {
    address public buyer;
    address public seller;
    uint256 public amount;
    bool public released;

    constructor(address _seller) payable {
        buyer = msg.sender;
        seller = _seller;
        amount = msg.value;
    }

    function release() external {
        require(msg.sender == buyer, "only buyer");
        require(!released, "already released");
        released = true;
        (bool ok, ) = seller.call{value: amount}("");
        require(ok, "transfer failed");
    }

    function refund() external {
        require(msg.sender == seller, "only seller");
        require(!released, "already released");
        released = true;
        (bool ok, ) = buyer.call{value: amount}("");
        require(ok, "transfer failed");
    }
}
