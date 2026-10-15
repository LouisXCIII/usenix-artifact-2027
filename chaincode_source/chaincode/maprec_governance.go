// maprec_governance.go -- a real, per-transaction-comparable ablation of Ab7 as formally
// defined: accepting a MapRec whose own governance signature is absent or invalid.
//
// Context. Section~sec:model's formal MapRec = <id, scope, M_I, M_T, R_Sigma, R_P, M_O, ...> is
// meant to be a governance-approved record establishing the destination's role-mapping policy.
// In the real deployed system, the mapping (RoleMapping/RoleThreshold, authchaincode.go) is
// compiled directly into the chaincode source -- there is no separately-deployed, independently
// signed MapRec object at all, so Ab7 "as formally defined" (an entry point that takes a MapRec
// parameter and checks its own governance signature) had no real per-transaction twin.
// evidence/e8_a7_governance_2026-09-24.md built a real analog instead, honestly disclosed as
// testing a DIFFERENT Fabric platform mechanism (transaction endorsement policy), not this one.
//
// This file builds the real thing. A MapRec is now a real, explicit, caller-supplied data
// structure (id, scope, role->cap table), authenticated by a real secp256k1 signature from a
// single pinned "mapping-governance authority" key -- the same RecoverCompact/digest pattern
// already used three times in this codebase (lightclient.go's verifyIbft2Header,
// ablation_a5_quorum.go's verifyIbft2HeaderCustomQuorum, q_d_domain_attestation.go's
// verifyAwsDomainAttestation). SubmitDecisionMapRecVerified checks that signature before
// trusting the MapRec's claimed caps; SubmitDecisionMapRecUnverified is its deliberate ablated
// twin, identical except it skips the signature check entirely -- exactly Ab7 as formally
// defined, with a real on-chain consequence (a forged MapRec can claim an attacker-favorable
// cap for a role, and only the signature check stands between that claim and a genuine
// unsafe-accept at the amount/role decision level).
//
// SemBound/EvAssuranceOK are taken as caller-supplied booleans here, matching the isolation
// level of the existing Ab7 endorsement-policy analog -- this experiment targets the
// MapRec-authentication guard specifically, not the full ZK/light-client pipeline tested
// elsewhere in this codebase.
package chaincode

import (
	"encoding/json"
	"fmt"
	"sort"

	secpecdsa "github.com/decred/dcrd/dcrec/secp256k1/v4/ecdsa"
	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
)

// mappingGovernanceAuthorityAddress is the pinned Ethereum-style address
// (keccak256(uncompressed pubkey)[12:]) of a secp256k1 key generated locally by
// tools/maprec_sign (2026-10-11) specifically for this experiment; the private key is held only
// in that tool's local key file, never embedded in the chaincode. Same hardcoded-trust-anchor
// pattern as pinnedValidatorHex (lightclient.go), policyAuthorityAddress (commit.go), and
// awsDomainAuthorityAddress (q_d_domain_attestation.go).
const mappingGovernanceAuthorityAddress = "0xf94d8b83004c9adaeefd8696a70cf864ca79053e"

// mapRecDigest is the canonical byte layout a MapRec's governance signature covers:
// keccak256(id || scope || role_1 || cap_1(32B BE) || role_2 || cap_2(32B BE) || ...), roles
// sorted lexicographically so the digest is independent of map iteration order and of the order
// fields were supplied in -- the same determinism concern decisionHash (commit.go) already
// solves for a different record type.
func mapRecDigest(id, scope string, caps map[string]int64) []byte {
	roles := make([]string, 0, len(caps))
	for r := range caps {
		roles = append(roles, r)
	}
	sort.Strings(roles)
	parts := [][]byte{[]byte(id), []byte(scope)}
	for _, r := range roles {
		parts = append(parts, []byte(r), word(caps[r]))
	}
	return keccak256(parts...)
}

// verifyMapRecSignature recovers the signer of a 65-byte compact signature (same layout as
// lightclient.go's commit-seal recovery and q_d_domain_attestation.go's AWS attestation:
// compact[0]=27+recoveryID, compact[1:33]=r, compact[33:65]=s) over digest, and reports whether
// it matches the pinned mapping-governance authority.
func verifyMapRecSignature(compactSigHex string, digest []byte) (bool, string, error) {
	sig, err := unhexBytes(compactSigHex)
	if err != nil || len(sig) != 65 {
		return false, "", fmt.Errorf("MapRec signature must be 65 bytes of hex, got %d bytes", len(sig))
	}
	pub, _, err := secpecdsa.RecoverCompact(sig, digest)
	if err != nil {
		return false, "", fmt.Errorf("MapRec signature recovery failed: %v", err)
	}
	addr := keccak256(pub.SerializeUncompressed()[1:])[12:]
	want, werr := unhexBytes(mappingGovernanceAuthorityAddress)
	if werr != nil {
		return false, "", werr
	}
	recovered := fmt.Sprintf("0x%x", addr)
	if len(addr) != len(want) {
		return false, recovered, nil
	}
	for i := range addr {
		if addr[i] != want[i] {
			return false, recovered, nil
		}
	}
	return true, recovered, nil
}

func mapRecCapsFromJSON(capsJSON string) (map[string]int64, error) {
	var caps map[string]int64
	if err := json.Unmarshal([]byte(capsJSON), &caps); err != nil {
		return nil, fmt.Errorf("malformed MapRec caps JSON: %v", err)
	}
	return caps, nil
}

// SubmitDecisionMapRecVerified is the REAL entry point: the destination's role->cap table is
// not read from the hardcoded RoleThreshold (authchaincode.go) but from a caller-supplied
// MapRec (id, scope, role->cap JSON), trusted only if its governance signature recovers to the
// pinned mapping-governance authority. An invalid or absent signature is fail-closed REJECT --
// the cap table is never trusted, regardless of what it claims.
func (c *AuthContract) SubmitDecisionMapRecVerified(ctx contractapi.TransactionContextInterface,
	decisionID string, role string, amount int64, semBound bool, evAssuranceOK bool,
	mapRecID string, mapRecScope string, mapRecCapsJSON string, mapRecSigHex string) (string, error) {

	exists, err := c.decisionExists(ctx, decisionID)
	if err != nil {
		return "", err
	}
	if exists {
		return "", fmt.Errorf("decision %s already recorded (replay rejected)", decisionID)
	}
	caps, err := mapRecCapsFromJSON(mapRecCapsJSON)
	if err != nil {
		return "", err
	}
	digest := mapRecDigest(mapRecID, mapRecScope, caps)
	sigOK, signer, sigErr := verifyMapRecSignature(mapRecSigHex, digest)

	var detail string
	switch {
	case sigErr != nil:
		sigOK = false
		detail = fmt.Sprintf("MapRec governance signature malformed: %v", sigErr)
	case !sigOK:
		detail = fmt.Sprintf("MapRec governance signature recovered to %s, not the pinned mapping-governance authority %s -- MapRec rejected, cap table NOT trusted", signer, mappingGovernanceAuthorityAddress)
	default:
		detail = fmt.Sprintf("MapRec governance signature verified (signer %s); cap table trusted", signer)
	}

	var class string
	var threshold int64
	var known bool
	if !sigOK {
		class = "REJECT" // Ab7 as formally defined: an unsigned/invalidly-signed MapRec is never trusted
	} else {
		threshold, known = caps[role]
		class = classify(semBound, evAssuranceOK, amount, threshold, known)
	}

	rec := PortableDecisionRecord{
		DecisionID: decisionID, Role: role, Amount: amount,
		SemBound: semBound, EvAssuranceOK: evAssuranceOK, TransferClass: class, DestThreshold: threshold,
		MapRecVerified: sigOK, MapRecSigner: signer, MapRecDetail: detail, MapRecChecked: true,
	}
	recJSON, err := json.Marshal(rec)
	if err != nil {
		return "", err
	}
	if err := ctx.GetStub().PutState(decisionID, recJSON); err != nil {
		return "", fmt.Errorf("failed to write state: %v", err)
	}
	return class, nil
}

// SubmitDecisionMapRecUnverified is the deliberate ablated twin of SubmitDecisionMapRecVerified:
// identical in every respect except it never calls verifyMapRecSignature -- it trusts the
// caller-supplied MapRec's cap table unconditionally (the signature is not even a parameter
// here). This is Ab7 as formally defined: "accepting a MapRec whose own governance signature is
// absent or invalid."
func (c *AuthContract) SubmitDecisionMapRecUnverified(ctx contractapi.TransactionContextInterface,
	decisionID string, role string, amount int64, semBound bool, evAssuranceOK bool,
	mapRecID string, mapRecScope string, mapRecCapsJSON string) (string, error) {

	exists, err := c.decisionExists(ctx, decisionID)
	if err != nil {
		return "", err
	}
	if exists {
		return "", fmt.Errorf("decision %s already recorded (replay rejected)", decisionID)
	}
	caps, err := mapRecCapsFromJSON(mapRecCapsJSON)
	if err != nil {
		return "", err
	}
	_ = mapRecID
	_ = mapRecScope

	threshold, known := caps[role]
	class := classify(semBound, evAssuranceOK, amount, threshold, known)

	rec := PortableDecisionRecord{
		DecisionID: decisionID, Role: role, Amount: amount,
		SemBound: semBound, EvAssuranceOK: evAssuranceOK, TransferClass: class, DestThreshold: threshold,
		MapRecVerified: false, MapRecDetail: "ABLATED: MapRec governance signature never checked (Ab7 as formally defined)", MapRecChecked: true,
	}
	recJSON, err := json.Marshal(rec)
	if err != nil {
		return "", err
	}
	if err := ctx.GetStub().PutState(decisionID, recJSON); err != nil {
		return "", fmt.Errorf("failed to write state: %v", err)
	}
	return class, nil
}
