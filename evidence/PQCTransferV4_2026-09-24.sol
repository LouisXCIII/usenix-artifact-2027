// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

interface IPQCVerifierOracle {
    function getAttestation(bytes32 pubKeyHash, bytes32 msgHash, bytes32 sigHash)
        external view returns (bool exists, bool verified, address attestor, uint256 ts);
}

/// @notice E5 further depth pass ("dao sau va phat trien dua E5 len rat tot"): PQCTransferV3
/// already closed single-party self-attestation by requiring an independent oracle attestation
/// (a separate identity re-runs the SAME Liboqs C/JNI implementation). The remaining disclosed
/// weakness was that both the submitter's self-check and the independent oracle still ultimately
/// depend on ONE implementation of ML-DSA-65 verification -- a bug shared by that single codebase
/// would not be caught. This contract requires TWO independent attestations from TWO SEPARATE
/// oracle contracts, each fed by a DIFFERENT verifier identity running a DIFFERENT, independently
/// authored implementation of ML-DSA-65 (Liboqs 0.15.0, C, Open Quantum Safe project; and
/// Cloudflare CIRCL v1.6.3, pure Go, FIPS 204 mldsa65 package) -- neither implementation calls or
/// links the other. VERIFIED is granted only if BOTH independently agree the signature is valid;
/// any disagreement between the two real implementations is itself recorded and exposed, not
/// silently resolved.
contract PQCTransferV4 {
    uint256 public constant DILITHIUM3_PUB_KEY_SIZE = 1976;

    address public owner;
    IPQCVerifierOracle public oracleLiboqs;
    IPQCVerifierOracle public oracleCircl;

    struct Bank {
        string name;
        bytes  pqcPublicKey;
        bytes32 pqcPublicKeyHash;
        bool isRegistered;
    }

    enum TransferStatus { NONE, VERIFIED, REJECTED }

    struct Transfer {
        address fromBank;
        address toBank;
        bytes32 payloadHash;
        bytes32 sigHash;
        bool claimedVerified;
        bool liboqsVerified;
        bool circlVerified;
        bool implementationsDisagree;
        TransferStatus status;
    }

    mapping(address => Bank) public banks;
    mapping(bytes32 => Transfer) public transfers;
    bytes32[] public allTransferIds;

    event BankRegistered(address indexed bankAddress, string name);
    event TransferSubmitted(
        bytes32 indexed transferId, address indexed fromBank, address indexed toBank,
        bool liboqsVerified, bool circlVerified, bool implementationsDisagree, TransferStatus status
    );

    modifier onlyOwner() { require(msg.sender == owner, "not owner"); _; }

    constructor(address _oracleLiboqs, address _oracleCircl) {
        owner = msg.sender;
        oracleLiboqs = IPQCVerifierOracle(_oracleLiboqs);
        oracleCircl = IPQCVerifierOracle(_oracleCircl);
    }

    function registerBank(address bankAddress, string memory name, bytes memory pqcPublicKey) external onlyOwner {
        require(pqcPublicKey.length == DILITHIUM3_PUB_KEY_SIZE, "PQC key must be 1976 bytes");
        banks[bankAddress] = Bank({ name: name, pqcPublicKey: pqcPublicKey,
            pqcPublicKeyHash: keccak256(pqcPublicKey), isRegistered: true });
        emit BankRegistered(bankAddress, name);
    }

    function submitTransfer(
        address toBank, bytes32 payloadHash, bytes memory pqcSignature, bool claimedVerified
    ) external returns (bytes32 transferId) {
        require(banks[msg.sender].isRegistered, "sender bank not registered");
        require(banks[toBank].isRegistered, "recipient bank not registered");

        bytes32 sigHash = keccak256(pqcSignature);
        bytes32 pubKeyHash = banks[msg.sender].pqcPublicKeyHash;

        (bool existsL, bool verifiedL, , ) = oracleLiboqs.getAttestation(pubKeyHash, payloadHash, sigHash);
        require(existsL, "no liboqs attestation for this (pubkey,msg,sig) tuple");
        (bool existsC, bool verifiedC, , ) = oracleCircl.getAttestation(pubKeyHash, payloadHash, sigHash);
        require(existsC, "no circl attestation for this (pubkey,msg,sig) tuple");

        bool disagree = (verifiedL != verifiedC);
        bool bothVerified = verifiedL && verifiedC;
        TransferStatus status = bothVerified ? TransferStatus.VERIFIED : TransferStatus.REJECTED;

        transferId = keccak256(abi.encodePacked(msg.sender, toBank, payloadHash, sigHash, block.timestamp));
        transfers[transferId] = Transfer({
            fromBank: msg.sender, toBank: toBank, payloadHash: payloadHash, sigHash: sigHash,
            claimedVerified: claimedVerified, liboqsVerified: verifiedL, circlVerified: verifiedC,
            implementationsDisagree: disagree, status: status
        });
        allTransferIds.push(transferId);

        emit TransferSubmitted(transferId, msg.sender, toBank, verifiedL, verifiedC, disagree, status);
    }

    function getTransfer(bytes32 transferId) external view
        returns (address, address, bool, bool, bool, bool, TransferStatus)
    {
        Transfer storage t = transfers[transferId];
        return (t.fromBank, t.toBank, t.liboqsVerified, t.circlVerified,
            t.implementationsDisagree, t.claimedVerified, t.status);
    }

    function getTotalTransfers() external view returns (uint256) { return allTransferIds.length; }
}
