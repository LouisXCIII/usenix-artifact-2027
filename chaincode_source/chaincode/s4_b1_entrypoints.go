// s4_b1_entrypoints.go -- additive, timing-instrumented entry points for the S4 fair-B1 measurement
// (evidence/s4_fair_b1_protocol_2026-09-28.md). Nothing in any existing file is modified and no existing entry
// point calls anything here.
//
//	SubmitDecisionTrustlessV7Timed  B4: SubmitDecisionTrustlessV7's exact logic (Groth16 verification, source-state
//	                                verification, Poseidon feature-commitment proof, classify, PutState), timed per step.
//	SubmitDecisionB1StoreTimed      B1: the Groth16 step is replaced by re-executing the attested relation in Go
//	                                (Poseidon digest of x, stored pinned model, logits/margin/class). Everything else identical.
//	SubmitDecisionB1NoStoreTimed    as above, but the caller supplies (W,b) and the pinned-model digest is recomputed per call.
//	SubmitDecisionB1ShaTimed        as B1Store, but x is bound by a SHA-256 commitment held in a second on-chain registry
//	                                (amendment A1), proven by Merkle-Patricia proof under the same verified header.
//
// All four return timing JSON and perform the real PutState, so `peer chaincode query` measures endorsement only and
// `peer chaincode invoke` measures submission through commit. Use disposable decisionIDs for invokes.
package chaincode

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
)

// Second FeatureCommitRegistry (same contract as featureCommitRegistryAddress, scanner authority = the same independent
// wallet) holding SHA-256(x) commitments. Deployed by evidence/s4_fair_b1/s4_register_besu.js.
const b1ShaRegistryAddress = "0xc425C8245052eb0F5D2bD041632cf9c9567836CA"

type s4Timing struct {
	Variant       string `json:"variant"`
	Class         string `json:"class"`
	ExistsNs      int64  `json:"existsNs"`
	VerifyNs      int64  `json:"verifyNs"` // Groth16 verification (B4) or re-execution of the attested relation (B1)
	LedgerNowNs   int64  `json:"ledgerNowNs"`
	SourceStateNs int64  `json:"sourceStateNs"`
	FeatureBindNs int64  `json:"featureBindNs"`
	ClassifyNs    int64  `json:"classifyNs"`
	PutStateNs    int64  `json:"putStateNs"`
	TotalNs       int64  `json:"totalNs"`
	VerifyOK      bool   `json:"verifyOk"`
	Detail        string `json:"detail"`
}

// s4Verify is the variant-specific step. It returns whether the attested relation held, a detail string, and the function
// that binds the features to the on-chain commitment (which registry and which digest depends on the variant).
type s4Verify func() (relationOK bool, detail string, bind func(stateRoot, codeHash []byte, featureProofJSON string) (bool, string), err error)

func (c *AuthContract) s4Run(ctx contractapi.TransactionContextInterface, variant string, timed bool,
	decisionID, sourceDomain, role string, amount, sourceLimit int64, codeHash string,
	headerJSON, registryProofJSON, featureProofJSON string, verify s4Verify) (string, error) {

	tStart := time.Now()
	t0 := time.Now()
	exists, err := c.decisionExists(ctx, decisionID)
	tExists := time.Since(t0)
	if err != nil {
		return "", err
	}
	if exists {
		return "", fmt.Errorf("decision %s already recorded (replay rejected)", decisionID)
	}
	ch, err := unhexBytes(codeHash)
	if err != nil || len(ch) != 32 {
		return "", fmt.Errorf("codeHash must be 32 bytes of hex")
	}

	t1 := time.Now()
	relOK, relDetail, bind, err := verify()
	tVerify := time.Since(t1)
	if err != nil {
		return "", fmt.Errorf("malformed verification request: %v", err)
	}

	t2 := time.Now()
	nowSec, err := ledgerNow(ctx)
	tLedger := time.Since(t2)
	if err != nil {
		return "", err
	}

	t3 := time.Now()
	src := verifySourceState(nowSec, ch, headerJSON, registryProofJSON)
	tSrc := time.Since(t3)

	featureOK, featureDetail := false, "source header not verified, so no feature commitment could be proven"
	t4 := time.Now()
	if src.Verified {
		featureOK, featureDetail = bind(src.StateRoot, ch, featureProofJSON)
	}
	tFeat := time.Since(t4)

	semBound := relOK && src.Verified && src.MlPassed && featureOK
	evOK := src.Verified && src.Approvals >= minApprovals
	detail := relDetail
	if relOK && src.Verified && !src.MlPassed {
		detail = "relation holds and is bound, but the registry's proven mlScanPassed[codeHash] is false"
	} else if relOK && src.Verified && src.MlPassed {
		detail = featureDetail
	}

	t5 := time.Now()
	threshold, known := RoleThreshold[role]
	if !known {
		threshold = 0
	}
	class := classify(semBound, evOK, amount, threshold, known)
	tClassify := time.Since(t5)

	rec := PortableDecisionRecord{
		DecisionID: decisionID, SourceDomain: sourceDomain, Role: role, Amount: amount,
		SourceLimit: sourceLimit, SemBound: semBound, EvAssuranceOK: evOK, TransferClass: class,
		DestThreshold: threshold, ZKVerified: variant == "b4-groth16", ZKVerificationDetail: detail,
		CodeHash: codeHash, LCVerified: src.Verified, LCDetail: src.Detail,
		SourceBlockNumber: src.BlockNumber, SourceBlockHash: src.BlockHash,
		ProvenApprovals: src.Approvals, ProvenMlScanPassed: src.MlPassed,
	}
	recJSON, err := json.Marshal(rec)
	if err != nil {
		return "", err
	}
	t6 := time.Now()
	putErr := ctx.GetStub().PutState(decisionID, recJSON)
	tPut := time.Since(t6)
	if putErr != nil {
		return "", fmt.Errorf("failed to write state: %v", putErr)
	}
	out := s4Timing{Variant: variant, Class: class, ExistsNs: tExists.Nanoseconds(), VerifyNs: tVerify.Nanoseconds(),
		LedgerNowNs: tLedger.Nanoseconds(), SourceStateNs: tSrc.Nanoseconds(), FeatureBindNs: tFeat.Nanoseconds(),
		ClassifyNs: tClassify.Nanoseconds(), PutStateNs: tPut.Nanoseconds(), TotalNs: time.Since(tStart).Nanoseconds(),
		VerifyOK: relOK, Detail: detail}
	if !timed {
		return class, nil // deterministic response, so two endorsing peers produce identical payloads
	}
	b, _ := json.Marshal(out)
	return string(b), nil
}

// B4: SubmitDecisionTrustlessV7's checks, unchanged, timed.
func (c *AuthContract) SubmitDecisionTrustlessV7Timed(ctx contractapi.TransactionContextInterface,
	decisionID string, sourceDomain string, role string, amount int64, sourceLimit int64,
	codeHash string, proofJSON string, publicSignalsJSON string,
	headerJSON string, registryProofJSON string, featureProofJSON string) (string, error) {

	verify := func() (bool, string, func([]byte, []byte, string) (bool, string), error) {
		ch, _ := unhexBytes(codeHash)
		zkOK, zkDetail, err := verifySemBoundProof(proofJSON, publicSignalsJSON)
		if err != nil {
			return false, "", nil, err
		}
		var pub []string
		_ = json.Unmarshal([]byte(publicSignalsJSON), &pub)
		if zkOK && len(pub) != 7 {
			zkOK = false
			zkDetail = fmt.Sprintf("requires a v7 proof (7 public signals), got %d", len(pub))
		}
		if zkOK {
			bound, ok := new(big.Int).SetString(pub[1], 10)
			if !ok || bound.Cmp(new(big.Int).SetBytes(ch)) != 0 {
				zkOK = false
				zkDetail = "proof verifies but its public solidityCodeHash differs from this decision's codeHash (proof substitution rejected)"
			}
		}
		bind := func(root, chb []byte, fp string) (bool, string) {
			if len(pub) == 7 {
				return verifyFeatureBinding(root, chb, fp, pub[6])
			}
			return false, "no v7 public featureHash signal present"
		}
		return zkOK, zkDetail, bind, nil
	}
	return c.s4Run(ctx, "b4-groth16", true, decisionID, sourceDomain, role, amount, sourceLimit, codeHash,
		headerJSON, registryProofJSON, featureProofJSON, verify)
}

func s4B1Verify(xJSON, wbJSON string) s4Verify {
	return func() (bool, string, func([]byte, []byte, string) (bool, string), error) {
		ok, fh, detail, err := b1Reevaluate(xJSON, wbJSON, 0, requiredMinMargin)
		if err != nil {
			return false, "", nil, err
		}
		bind := func(root, chb []byte, fp string) (bool, string) {
			if fh == "" {
				return false, "feature vector rejected before hashing"
			}
			return verifyFeatureBinding(root, chb, fp, fh)
		}
		return ok, detail, bind, nil
	}
}

// B1, stored pinned model, x bound through the Poseidon commitment (the scheme the proof's circuit uses).
func (c *AuthContract) SubmitDecisionB1StoreTimed(ctx contractapi.TransactionContextInterface,
	decisionID string, sourceDomain string, role string, amount int64, sourceLimit int64,
	codeHash string, xJSON string,
	headerJSON string, registryProofJSON string, featureProofJSON string) (string, error) {
	return c.s4Run(ctx, "b1-store", true, decisionID, sourceDomain, role, amount, sourceLimit, codeHash,
		headerJSON, registryProofJSON, featureProofJSON, s4B1Verify(xJSON, ""))
}

// B1, model supplied per call and its Poseidon digest recomputed against the pinned hash.
func (c *AuthContract) SubmitDecisionB1NoStoreTimed(ctx contractapi.TransactionContextInterface,
	decisionID string, sourceDomain string, role string, amount int64, sourceLimit int64,
	codeHash string, xJSON string, wbJSON string,
	headerJSON string, registryProofJSON string, featureProofJSON string) (string, error) {
	if wbJSON == "" {
		return "", fmt.Errorf("the no-store variant needs the model argument")
	}
	return c.s4Run(ctx, "b1-nostore", true, decisionID, sourceDomain, role, amount, sourceLimit, codeHash,
		headerJSON, registryProofJSON, featureProofJSON, s4B1Verify(xJSON, wbJSON))
}

// B1, stored pinned model, x bound through a SHA-256 commitment in the second registry (amendment A1).
func (c *AuthContract) SubmitDecisionB1ShaTimed(ctx contractapi.TransactionContextInterface,
	decisionID string, sourceDomain string, role string, amount int64, sourceLimit int64,
	codeHash string, xJSON string,
	headerJSON string, registryProofJSON string, shaProofJSON string) (string, error) {
	return c.s4B1Sha(ctx, true, decisionID, sourceDomain, role, amount, sourceLimit, codeHash, xJSON,
		headerJSON, registryProofJSON, shaProofJSON)
}

func (c *AuthContract) s4B1Sha(ctx contractapi.TransactionContextInterface, timed bool,
	decisionID string, sourceDomain string, role string, amount int64, sourceLimit int64,
	codeHash string, xJSON string,
	headerJSON string, registryProofJSON string, shaProofJSON string) (string, error) {

	verify := func() (bool, string, func([]byte, []byte, string) (bool, string), error) {
		ok, digest, err := b1ReevaluateSHA(xJSON, 0, requiredMinMargin)
		if err != nil {
			return false, "", nil, err
		}
		detail := "relation not satisfied"
		if ok {
			detail = "relation satisfied"
		}
		bind := func(root, chb []byte, fp string) (bool, string) {
			var rp registryProof
			if err := json.Unmarshal([]byte(fp), &rp); err != nil {
				return false, fmt.Sprintf("malformed SHA-commitment proof JSON: %v", err)
			}
			if !strings.EqualFold(rp.Address, b1ShaRegistryAddress) {
				return false, "proof is for a contract that is not the pinned SHA-commitment registry"
			}
			v, err := provenMappingAt(root, rp, b1ShaRegistryAddress, chb, slotFeatureHash)
			if err != nil {
				return false, fmt.Sprintf("SHA commitment not proven: %v", err)
			}
			if v.Sign() == 0 {
				return false, "no SHA-256 commitment exists for this exact codeHash"
			}
			if v.Cmp(new(big.Int).SetBytes(digest[:])) != 0 {
				return false, "SHA-256 of the supplied features does not match the scanner's on-chain commitment"
			}
			return true, "SHA-256 commitment proven to match the supplied features"
		}
		return ok, detail, bind, nil
	}
	return c.s4Run(ctx, "b1-sha", timed, decisionID, sourceDomain, role, amount, sourceLimit, codeHash,
		headerJSON, registryProofJSON, shaProofJSON, verify)
}

// Deterministic (non-timed) B1 entry points: they return only the class, exactly like SubmitDecisionTrustlessV7, so a
// transaction endorsed by both organizations' peers assembles and commits. B4 for the end-to-end run is the deployed,
// unmodified SubmitDecisionTrustlessV7.
func (c *AuthContract) SubmitDecisionB1Store(ctx contractapi.TransactionContextInterface,
	decisionID string, sourceDomain string, role string, amount int64, sourceLimit int64,
	codeHash string, xJSON string,
	headerJSON string, registryProofJSON string, featureProofJSON string) (string, error) {
	return c.s4Run(ctx, "b1-store", false, decisionID, sourceDomain, role, amount, sourceLimit, codeHash,
		headerJSON, registryProofJSON, featureProofJSON, s4B1Verify(xJSON, ""))
}

func (c *AuthContract) SubmitDecisionB1NoStore(ctx contractapi.TransactionContextInterface,
	decisionID string, sourceDomain string, role string, amount int64, sourceLimit int64,
	codeHash string, xJSON string, wbJSON string,
	headerJSON string, registryProofJSON string, featureProofJSON string) (string, error) {
	if wbJSON == "" {
		return "", fmt.Errorf("the no-store variant needs the model argument")
	}
	return c.s4Run(ctx, "b1-nostore", false, decisionID, sourceDomain, role, amount, sourceLimit, codeHash,
		headerJSON, registryProofJSON, featureProofJSON, s4B1Verify(xJSON, wbJSON))
}

func (c *AuthContract) SubmitDecisionB1Sha(ctx contractapi.TransactionContextInterface,
	decisionID string, sourceDomain string, role string, amount int64, sourceLimit int64,
	codeHash string, xJSON string,
	headerJSON string, registryProofJSON string, shaProofJSON string) (string, error) {
	return c.s4B1Sha(ctx, false, decisionID, sourceDomain, role, amount, sourceLimit, codeHash, xJSON,
		headerJSON, registryProofJSON, shaProofJSON)
}
