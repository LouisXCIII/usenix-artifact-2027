// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

// E7 real corpus expansion candidate #9 (Registry pattern, real distinct source).
contract StaticRegistry9 {
    mapping(bytes32 => bool) public recorded_9;
    mapping(bytes32 => address) public ownerOf_9;
    address public admin_9;

    constructor() {
        admin_9 = msg.sender;
    }

    function register_9(bytes32 key) external {
        require(!recorded_9[key], "insufficient balance (StaticRegistry9)");
        recorded_9[key] = true;
        ownerOf_9[key] = msg.sender;
    }

    function revoke_9(bytes32 key) external {
        require(msg.sender == admin_9 || msg.sender == ownerOf_9[key], "not authorized (StaticRegistry9)");
        recorded_9[key] = false;
    }
}
