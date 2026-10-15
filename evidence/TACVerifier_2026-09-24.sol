// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

/// @notice S19/A10 real on-chain guard ("đào sâu và phát triển đưa S19/A10 lên rất tốt"). Ports
/// the standalone Transfer-Admissibility Certificate (TAC) construction from
/// `s19_tac_substitution.js` (eq:architecture-tac) into a real, deployed contract, closing that
/// script's disclosed gap ("a standalone, additive TAC module... has not been wired into either
/// real network's actual transaction-execution path"). The 11-field digest packing exactly
/// mirrors `tacFields()`/`tacDigest()`: keccak256(hDs, hPis, keccak256(mapID), keccak256(assurID),
/// hWSigma, hWP, hEta, keccak256(classLabel), delta, exp, issuer), signed with the EIP-191
/// personal-sign scheme (`ethers.Wallet.signMessage` / `ecrecover` over the prefixed digest) --
/// the same real scheme the standalone script already used and this contract's `issue()`
/// independently re-verifies at issuance time.
///
/// `enforceFieldBinding` (set once, at construction) selects which of the two properties this
/// deployed instance checks: `true` is S19 (both the presented context must match the issued
/// certificate's own bound fields, AND the issuer's signature must be valid) -- deploy one
/// instance this way. `false` is A10's ablation (only signature validity is checked; the
/// presented context is accepted without comparison) -- deploy a second, separate instance this
/// way. Both instances are fed the IDENTICAL real certificates, so the same substitution attempts
/// can be replayed against both and their real divergence observed on-chain.
contract TACVerifier {
    struct TACContext {
        bytes32 hDs;
        bytes32 hPis;
        bytes32 hMapID;      // raw mapID string is hashed on-chain, matching h(ctx.mapID) off-chain
        bytes32 hAssurID;    // raw assurID string is hashed on-chain, matching h(ctx.assurID)
        bytes32 hWSigma;
        bytes32 hWP;
        bytes32 hEta;
        bytes32 hClassLabel; // raw classLabel string is hashed on-chain, matching h(ctx.classLabel)
        bytes32 delta;
        string  exp;
        address issuer;
    }

    struct TAC {
        TACContext ctx;
        bytes32 digest;
        bytes signature;
        bool exists;
    }

    bool public immutable enforceFieldBinding;
    mapping(bytes32 => TAC) public certs;

    event Issued(bytes32 indexed certId, address indexed issuer);
    event Verified(bytes32 indexed certId, bool valid, bool fieldMismatch, bool sigValid);

    constructor(bool _enforceFieldBinding) {
        enforceFieldBinding = _enforceFieldBinding;
    }

    function computeDigest(TACContext calldata ctx) public pure returns (bytes32) {
        return keccak256(abi.encodePacked(
            ctx.hDs, ctx.hPis, ctx.hMapID, ctx.hAssurID,
            ctx.hWSigma, ctx.hWP, ctx.hEta,
            ctx.hClassLabel, ctx.delta, ctx.exp, ctx.issuer
        ));
    }

    function ethSignedDigest(bytes32 digest) public pure returns (bytes32) {
        return keccak256(abi.encodePacked("\x19Ethereum Signed Message:\n32", digest));
    }

    function recoverSigner(bytes32 digest, bytes memory sig) public pure returns (address) {
        require(sig.length == 65, "bad sig length");
        bytes32 r; bytes32 s; uint8 v;
        assembly {
            r := mload(add(sig, 32))
            s := mload(add(sig, 64))
            v := byte(0, mload(add(sig, 96)))
        }
        if (v < 27) v += 27;
        return ecrecover(ethSignedDigest(digest), v, r, s);
    }

    /// @notice Issue a real certificate. Independently re-verifies the supplied signature against
    /// the recomputed digest and the claimed issuer BEFORE storing -- a malformed or forged
    /// certificate is rejected at issuance, not silently accepted.
    function issue(bytes32 certId, TACContext calldata ctx, bytes calldata signature) external {
        require(!certs[certId].exists, "cert already issued");
        bytes32 digest = computeDigest(ctx);
        address recovered = recoverSigner(digest, signature);
        require(recovered == ctx.issuer, "signature does not match claimed issuer");
        certs[certId] = TAC({ ctx: ctx, digest: digest, signature: signature, exists: true });
        emit Issued(certId, ctx.issuer);
    }

    /// @notice Real on-chain verification, recorded as a transaction (not a view-only call) so
    /// every attempt -- honest or a substitution attack -- is itself a real, mined case.
    function verifyContext(bytes32 certId, TACContext calldata presented)
        external returns (bool valid, bool fieldMismatch, bool sigValid)
    {
        TAC storage c = certs[certId];
        require(c.exists, "no such certificate");

        sigValid = (recoverSigner(c.digest, c.signature) == c.ctx.issuer);

        if (enforceFieldBinding) {
            fieldMismatch =
                presented.hDs != c.ctx.hDs ||
                presented.hPis != c.ctx.hPis ||
                presented.hMapID != c.ctx.hMapID ||
                presented.hAssurID != c.ctx.hAssurID ||
                presented.hWSigma != c.ctx.hWSigma ||
                presented.hWP != c.ctx.hWP ||
                presented.hEta != c.ctx.hEta ||
                presented.hClassLabel != c.ctx.hClassLabel ||
                presented.delta != c.ctx.delta ||
                keccak256(bytes(presented.exp)) != keccak256(bytes(c.ctx.exp)) ||
                presented.issuer != c.ctx.issuer;
            valid = sigValid && !fieldMismatch;
        } else {
            // A10 ablation: field binding is not checked at all -- the presented context is
            // accepted without comparison, matching "field-binding check removed from
            // verification, leaving only signature validity".
            fieldMismatch = false;
            valid = sigValid;
        }

        emit Verified(certId, valid, fieldMismatch, sigValid);
    }
}
