// zkverify.go — real, on-chain Groth16 verification of the SemBound proof
// at the destination, closing the gap named in Limitations item 1
// (Manuscriptsp2_Full_v1.tex): SubmitDecision/SubmitDecisionFull take
// semBound as a caller-supplied boolean that the destination never
// independently checks. SubmitDecisionVerified instead takes the real
// Groth16 proof and public signals produced by the ml_scan_proof_v5
// circuit (Section subsec:impl-sembound-source) and re-verifies the
// pairing equation itself, in Go, against the embedded verification key --
// the same key already used by MLScanVerifierV5.sol on the source chain --
// rather than trusting a value the relay read there and forwarded.
package chaincode

import (
	"encoding/json"
	"fmt"
	"math/big"

	_ "embed"

	"github.com/consensys/gnark-crypto/ecc/bn254"
	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
)

//go:embed verification_key_v5.json
var verificationKeyV5JSON []byte

//go:embed verification_key_v6.json
var verificationKeyV6JSON []byte

//go:embed verification_key_v7.json
var verificationKeyV7JSON []byte

// v6 policy (relation ml_scan_proof_v6, 6 public signals: scanPassed, solidityCodeHash, commitmentHash,
// maxVulnClass, minMargin, modelWeightsHash). The model is pinned, and the confidence floor is replaced by a
// margin floor implied by it: a winner leading every other logit by m has softmax confidence of at least
// e^m/(e^m+11) over 12 classes, so confidence >= 0.70 is implied by m >= ln(0.7*11/0.3) = 3.2452 logits,
// i.e. 54,445,307 in the quantized scale 4096^2. It is derived, not tuned to the corpus.
const (
	requiredMinMargin      = 54445307
	pinnedModelWeightsHash = "10494038872765858080845244269484429820162559076895853253651143169770011946707"
)

// allowLegacyV5 controls whether v5 proofs (5 public signals) are still accepted. In v5 the model hash
// and the confidence are prover-declared private inputs (attacks A3, A2d of the X1 experiment), so the
// default is to refuse them; tests flip it to exercise the legacy path.
var allowLegacyV5 = false

// destination-owned policy for the ml_scan_proof_v5 public signals -- these
// are NOT read from the proof; they are the destination's own required
// scan policy, checked against what the prover declared. A dishonest relay
// that submits a genuine proof for a weaker policy (e.g. maxVulnClass=11,
// "any class passes") is rejected here even though the pairing check alone
// would accept it, which is exactly the gap a pairing-check-only verifier
// would still miss.
const (
	requiredMaxVulnClass = "0" // only the "safe" (class 0) verdict is accepted
	// requiredMinConfidence is the destination's confidence floor (basis points). It was set to
	// 7000 AFTER observing that the real classifier scores the paper's HONEST contract (SafeBank)
	// at 0.7739; an earlier draft used 8000, which would have rejected that contract. It is an
	// illustrative policy value, not an independently derived security parameter.
	requiredMinConfidence = 7000
)

type groth16VKey struct {
	VkAlpha1 []string   `json:"vk_alpha_1"`
	VkBeta2  [][]string `json:"vk_beta_2"`
	VkGamma2 [][]string `json:"vk_gamma_2"`
	VkDelta2 [][]string `json:"vk_delta_2"`
	IC       [][]string `json:"IC"`
}

type groth16Proof struct {
	PiA []string   `json:"pi_a"`
	PiB [][]string `json:"pi_b"`
	PiC []string   `json:"pi_c"`
}

func g1FromStrings(s []string) (bn254.G1Affine, error) {
	var p bn254.G1Affine
	x, ok := new(big.Int).SetString(s[0], 10)
	if !ok {
		return p, fmt.Errorf("bad G1.x %q", s[0])
	}
	y, ok := new(big.Int).SetString(s[1], 10)
	if !ok {
		return p, fmt.Errorf("bad G1.y %q", s[1])
	}
	p.X.SetBigInt(x)
	p.Y.SetBigInt(y)
	return p, nil
}

// g2FromStrings assumes the natural snarkjs (c0,c1) ordering, empirically
// confirmed (scratch self-test against the real, already Solidity-verified
// proof_A_v5.json/verification_key_v5.json pair) to match gnark-crypto's
// G2Affine{X:{A0,A1},Y:{A0,A1}} field assignment directly, with no swap.
func g2FromStrings(s [][]string) (bn254.G2Affine, error) {
	var p bn254.G2Affine
	x0, ok := new(big.Int).SetString(s[0][0], 10)
	if !ok {
		return p, fmt.Errorf("bad G2.x0 %q", s[0][0])
	}
	x1, ok := new(big.Int).SetString(s[0][1], 10)
	if !ok {
		return p, fmt.Errorf("bad G2.x1 %q", s[0][1])
	}
	y0, ok := new(big.Int).SetString(s[1][0], 10)
	if !ok {
		return p, fmt.Errorf("bad G2.y0 %q", s[1][0])
	}
	y1, ok := new(big.Int).SetString(s[1][1], 10)
	if !ok {
		return p, fmt.Errorf("bad G2.y1 %q", s[1][1])
	}
	p.X.A0.SetBigInt(x0)
	p.X.A1.SetBigInt(x1)
	p.Y.A0.SetBigInt(y0)
	p.Y.A1.SetBigInt(y1)
	return p, nil
}

// verifyGroth16 checks the standard Groth16 pairing equation
// e(A,B) == e(alpha,beta) . e(vk_x,gamma) . e(C,delta), i.e.
// e(-A,B) . e(alpha,beta) . e(vk_x,gamma) . e(C,delta) == 1, against the
// embedded verification key. It does not interpret the public signals;
// callers must independently check they satisfy the destination's policy.
func verifyGroth16(vk groth16VKey, pf groth16Proof, public []string) (bool, error) {
	if len(public) != len(vk.IC)-1 {
		return false, fmt.Errorf("public input count mismatch: got %d want %d", len(public), len(vk.IC)-1)
	}

	alpha1, err := g1FromStrings(vk.VkAlpha1)
	if err != nil {
		return false, err
	}
	beta2, err := g2FromStrings(vk.VkBeta2)
	if err != nil {
		return false, err
	}
	gamma2, err := g2FromStrings(vk.VkGamma2)
	if err != nil {
		return false, err
	}
	delta2, err := g2FromStrings(vk.VkDelta2)
	if err != nil {
		return false, err
	}
	A, err := g1FromStrings(pf.PiA)
	if err != nil {
		return false, err
	}
	B, err := g2FromStrings(pf.PiB)
	if err != nil {
		return false, err
	}
	C, err := g1FromStrings(pf.PiC)
	if err != nil {
		return false, err
	}

	ic0, err := g1FromStrings(vk.IC[0])
	if err != nil {
		return false, err
	}
	vkxJac := new(bn254.G1Jac).FromAffine(&ic0)
	for i, pubStr := range public {
		scalar, ok := new(big.Int).SetString(pubStr, 10)
		if !ok {
			return false, fmt.Errorf("bad public input %q", pubStr)
		}
		icPoint, err := g1FromStrings(vk.IC[i+1])
		if err != nil {
			return false, err
		}
		var icJac bn254.G1Jac
		icJac.FromAffine(&icPoint)
		icJac.ScalarMultiplication(&icJac, scalar)
		vkxJac.AddAssign(&icJac)
	}
	var vkx bn254.G1Affine
	vkx.FromJacobian(vkxJac)

	var negA bn254.G1Affine
	negA.Neg(&A)

	return bn254.PairingCheck(
		[]bn254.G1Affine{negA, alpha1, vkx, C},
		[]bn254.G2Affine{B, beta2, gamma2, delta2},
	)
}

// verifySemBoundProof re-verifies the ml_scan_proof_v5 Groth16 proof and
// checks its public signals ([scanPassed, solidityCodeHash, commitmentHash,
// maxVulnClass, minConfidence], matching the circuit's declared public
// list) against the destination's own policy, rather than the prover's.
// Returns (semBoundVerified, humanReadableDetail, error). error is non-nil
// only for a malformed request (bad JSON); a well-formed but invalid proof
// or a proof that fails the destination's own policy check returns
// semBoundVerified=false with no error, so the caller can still record a
// REJECT decision rather than aborting the transaction.
func verifySemBoundProof(proofJSON, publicJSON string) (bool, string, error) {
	var pf groth16Proof
	if err := json.Unmarshal([]byte(proofJSON), &pf); err != nil {
		return false, "", fmt.Errorf("malformed proof JSON: %v", err)
	}
	var public []string
	if err := json.Unmarshal([]byte(publicJSON), &public); err != nil {
		return false, "", fmt.Errorf("malformed public-signals JSON: %v", err)
	}
	var keyJSON []byte
	switch len(public) {
	case 5:
		if !allowLegacyV5 {
			return false, "v5 proofs are not accepted: in that relation the model hash and the confidence are prover-declared", nil
		}
		keyJSON = verificationKeyV5JSON
	case 6:
		keyJSON = verificationKeyV6JSON
	case 7:
		keyJSON = verificationKeyV7JSON
	default:
		return false, fmt.Sprintf("expected 5, 6 or 7 public signals, got %d", len(public)), nil
	}
	var vk groth16VKey
	if err := json.Unmarshal(keyJSON, &vk); err != nil {
		return false, "", fmt.Errorf("embedded verification key is corrupt: %v", err)
	}

	pairingOK, err := verifyGroth16(vk, pf, public)
	if err != nil {
		return false, fmt.Sprintf("pairing check error: %v", err), nil
	}
	if !pairingOK {
		return false, "Groth16 pairing check failed: proof does not verify against the embedded verification key", nil
	}
	if public[0] != "1" {
		return false, "proof verifies, but its own scanPassed output signal is 0 (circuit-internal check failed)", nil
	}
	if public[3] != requiredMaxVulnClass {
		return false, fmt.Sprintf("proof verifies, but was produced for maxVulnClass=%s; destination policy requires exactly %s", public[3], requiredMaxVulnClass), nil
	}

	if len(public) == 6 || len(public) == 7 {
		if public[5] != pinnedModelWeightsHash {
			return false, "proof verifies, but its modelWeightsHash is not the pinned model: it was computed over other weights", nil
		}
		margin, ok := new(big.Int).SetString(public[4], 10)
		if !ok {
			return false, fmt.Sprintf("malformed minMargin public signal %q", public[4]), nil
		}
		if margin.Cmp(big.NewInt(requiredMinMargin)) < 0 {
			return false, fmt.Sprintf("proof verifies, but its minMargin=%s is below destination policy minimum %d", public[4], requiredMinMargin), nil
		}
		// v7 adds public[6] = featureHash, a circuit-internal consistency check only (Poseidon digest of
		// the private feature vector x). This function cannot tell whether it is the HONEST digest of the
		// contract behind solidityCodeHash -- that requires Besu state the caller has and this function
		// does not (same reasoning as the solidityCodeHash-vs-decision binding, checked by the caller).
		// SubmitDecisionTrustlessV7 (featurebind.go) proves featureHash[codeHash] separately and rejects
		// a mismatch; a v7 proof that verifies here is therefore necessary but not sufficient for SemBound.
		if len(public) == 7 {
			return true, "Groth16 v7 proof verified at the destination; model hash and margin meet destination policy; featureHash still needs proving against the on-chain scanner commitment", nil
		}
		return true, "Groth16 v6 proof verified at the destination; model hash matches the pinned model and the margin meets destination policy", nil
	}

	minConfidence, ok := new(big.Int).SetString(public[4], 10)
	if !ok {
		return false, fmt.Sprintf("malformed minConfidence public signal %q", public[4]), nil
	}
	if minConfidence.Cmp(big.NewInt(requiredMinConfidence)) < 0 {
		return false, fmt.Sprintf("proof verifies, but its minConfidence=%s is below destination policy minimum %d", public[4], requiredMinConfidence), nil
	}
	return true, "Groth16 proof verified on-chain at the destination; public signals meet destination policy (maxVulnClass, minConfidence)", nil
}

// SubmitDecisionVerified is the destination entry point that closes
// Limitations item 1: instead of taking semBound as a caller-supplied
// boolean, it takes the real Groth16 proof and public signals and
// re-verifies them on-chain, at the destination, before computing SemBound.
// A malicious or compromised relay that substitutes semBound=true for a
// decision where it is genuinely 0 -- threat (a) of
// Section~subsec:model-trust, named in the threat model but previously
// undefended -- is now caught here: either the pairing check fails, or the
// proof's own declared maxVulnClass/minConfidence do not meet the
// destination's policy, in which case semBoundVerified is false and the
// same classify() ordering (Theorem T0 first) that every other entry point
// uses applies to it.
func (c *AuthContract) SubmitDecisionVerified(ctx contractapi.TransactionContextInterface,
	decisionID string, sourceDomain string, role string, amount int64, sourceLimit int64,
	proofJSON string, publicSignalsJSON string, evAssuranceOK bool) (string, error) {

	exists, err := c.decisionExists(ctx, decisionID)
	if err != nil {
		return "", err
	}
	if exists {
		return "", fmt.Errorf("decision %s already recorded (replay rejected)", decisionID)
	}

	semBoundVerified, detail, err := verifySemBoundProof(proofJSON, publicSignalsJSON)
	if err != nil {
		return "", fmt.Errorf("malformed verification request: %v", err)
	}

	threshold, known := RoleThreshold[role]
	if !known {
		threshold = 0
	}
	class := classify(semBoundVerified, evAssuranceOK, amount, threshold, known)

	rec := PortableDecisionRecord{
		DecisionID:           decisionID,
		SourceDomain:         sourceDomain,
		Role:                 role,
		Amount:               amount,
		SourceLimit:          sourceLimit,
		SemBound:             semBoundVerified,
		EvAssuranceOK:        evAssuranceOK,
		TransferClass:        class,
		DestThreshold:        threshold,
		ZKVerified:           true,
		ZKVerificationDetail: detail,
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
