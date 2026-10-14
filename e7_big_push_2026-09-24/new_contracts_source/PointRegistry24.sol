// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

// E7 real corpus expansion candidate #24 (Registry pattern, real distinct source).
contract PointRegistry24 {
    mapping(bytes32 => bool) public recorded_24;
    mapping(bytes32 => address) public ownerOf_24;
    address public admin_24;

    constructor() {
        admin_24 = msg.sender;
    }

    function register_24(bytes32 key) external {
        require(!recorded_24[key], "insufficient balance (PointRegistry24)");
        recorded_24[key] = true;
        ownerOf_24[key] = msg.sender;
    }

    function revoke_24(bytes32 key) external {
        require(msg.sender == admin_24 || msg.sender == ownerOf_24[key], "not authorized (PointRegistry24)");
        recorded_24[key] = false;
    }
}
