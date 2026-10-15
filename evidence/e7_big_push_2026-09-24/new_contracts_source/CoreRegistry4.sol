// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

// E7 real corpus expansion candidate #4 (Registry pattern, real distinct source).
contract CoreRegistry4 {
    mapping(bytes32 => bool) public recorded_4;
    mapping(bytes32 => address) public ownerOf_4;
    address public admin_4;

    constructor() {
        admin_4 = msg.sender;
    }

    function register_4(bytes32 key) external {
        require(!recorded_4[key], "insufficient balance (CoreRegistry4)");
        recorded_4[key] = true;
        ownerOf_4[key] = msg.sender;
    }

    function revoke_4(bytes32 key) external {
        require(msg.sender == admin_4 || msg.sender == ownerOf_4[key], "not authorized (CoreRegistry4)");
        recorded_4[key] = false;
    }
}
