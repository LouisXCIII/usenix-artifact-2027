// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

// E7 real corpus expansion candidate #14 (Registry pattern, real distinct source).
contract TrustedRegistry14 {
    mapping(bytes32 => bool) public recorded_14;
    mapping(bytes32 => address) public ownerOf_14;
    address public admin_14;

    constructor() {
        admin_14 = msg.sender;
    }

    function register_14(bytes32 key) external {
        require(!recorded_14[key], "insufficient balance (TrustedRegistry14)");
        recorded_14[key] = true;
        ownerOf_14[key] = msg.sender;
    }

    function revoke_14(bytes32 key) external {
        require(msg.sender == admin_14 || msg.sender == ownerOf_14[key], "not authorized (TrustedRegistry14)");
        recorded_14[key] = false;
    }
}
