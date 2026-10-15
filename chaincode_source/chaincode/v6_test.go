package chaincode

import (
	"encoding/json"
	"strings"
	"testing"
)

// The legacy v5 tests elsewhere exercise the v5 path; production refuses v5 by default.
func init() { allowLegacyV5 = true }

func TestV6_HonestProofVerifiesUnderPinnedPolicy(t *testing.T) {
	ok, detail, err := verifySemBoundProof(readTD(t, "v6_honest_proof.json"), readTD(t, "v6_honest_public.json"))
	if err != nil || !ok {
		t.Fatalf("honest v6 proof must verify: ok=%v err=%v detail=%s", ok, err, detail)
	}
}

func TestV6_SubstitutedWeightsRejectedByPinnedModelHash(t *testing.T) {
	// A valid v6 proof (the pairing check passes) computed over W=0 and a bias favouring class 0.
	ok, detail, err := verifySemBoundProof(readTD(t, "v6_a3_proof.json"), readTD(t, "v6_a3_public.json"))
	if err != nil {
		t.Fatal(err)
	}
	if ok || !strings.Contains(detail, "not the pinned model") {
		t.Fatalf("substituted weights must be rejected on the model hash, got ok=%v detail=%s", ok, detail)
	}
}

func TestV6_ForgedFeaturesStillVerify(t *testing.T) {
	// Documents the residual gap: the relation does not bind the features to the source, so a proof over
	// another contract's features, labelled with this contract's hash, verifies. The destination cannot tell.
	ok, detail, err := verifySemBoundProof(readTD(t, "v6_a2_proof.json"), readTD(t, "v6_a2_public.json"))
	if err != nil || !ok {
		t.Fatalf("forged-feature proof is expected to verify (known limitation), got ok=%v err=%v detail=%s", ok, err, detail)
	}
}

func TestV6_TamperedPublicSignalsRejected(t *testing.T) {
	var pub []string
	if err := json.Unmarshal([]byte(readTD(t, "v6_honest_public.json")), &pub); err != nil {
		t.Fatal(err)
	}
	for _, idx := range []int{1, 2, 4, 5} {
		p := append([]string{}, pub...)
		p[idx] = "1234567"
		b, _ := json.Marshal(p)
		ok, _, err := verifySemBoundProof(readTD(t, "v6_honest_proof.json"), string(b))
		if err != nil || ok {
			t.Fatalf("changing public signal %d must invalidate the proof (ok=%v err=%v)", idx, ok, err)
		}
	}
}

func TestV6_FixtureCarriesTheDerivedMarginFloor(t *testing.T) {
	var pub []string
	_ = json.Unmarshal([]byte(readTD(t, "v6_honest_public.json")), &pub)
	if pub[4] != "54445307" {
		t.Fatalf("fixture minMargin is %s, expected the derived floor 54445307", pub[4])
	}
}

func TestV5ProofsRefusedByDefault(t *testing.T) {
	allowLegacyV5 = false
	defer func() { allowLegacyV5 = true }()
	ok, detail, err := verifySemBoundProof(readTD(t, "safebank_proof_v5.json"), readTD(t, "safebank_public_v5.json"))
	if err != nil || ok || !strings.Contains(detail, "v5 proofs are not accepted") {
		t.Fatalf("v5 must be refused by default, got ok=%v err=%v detail=%s", ok, err, detail)
	}
}
