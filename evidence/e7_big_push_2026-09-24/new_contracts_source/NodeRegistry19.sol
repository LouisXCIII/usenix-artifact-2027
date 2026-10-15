// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

// E7 real corpus expansion candidate #19 (Registry pattern, real distinct source).
contract NodeRegistry19 {
    mapping(bytes32 => bool) public recorded_19;
    mapping(bytes32 => address) public ownerOf_19;
    address public admin_19;

    constructor() {
        admin_19 = msg.sender;
    }

    function register_19(bytes32 key) external {
        require(!recorded_19[key], "insufficient balance (NodeRegistry19)");
        recorded_19[key] = true;
        ownerOf_19[key] = msg.sender;
    }

    function revoke_19(bytes32 key) external {
        require(msg.sender == admin_19 || msg.sender == ownerOf_19[key], "not authorized (NodeRegistry19)");
        recorded_19[key] = false;
    }
}
