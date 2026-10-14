// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

// E7 real corpus expansion candidate #29 (Registry pattern, real distinct source).
contract GridRegistry29 {
    mapping(bytes32 => bool) public recorded_29;
    mapping(bytes32 => address) public ownerOf_29;
    address public admin_29;

    constructor() {
        admin_29 = msg.sender;
    }

    function register_29(bytes32 key) external {
        require(!recorded_29[key], "insufficient balance (GridRegistry29)");
        recorded_29[key] = true;
        ownerOf_29[key] = msg.sender;
    }

    function revoke_29(bytes32 key) external {
        require(msg.sender == admin_29 || msg.sender == ownerOf_29[key], "not authorized (GridRegistry29)");
        recorded_29[key] = false;
    }
}
