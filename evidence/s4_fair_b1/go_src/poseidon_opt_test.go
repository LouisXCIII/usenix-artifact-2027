package chaincode

import (
	"encoding/json"
	"math/big"
	"os"
	"testing"

	"github.com/consensys/gnark-crypto/ecc/bn254/fr"
)

func TestPoseidon16MatchesCircomlibjs(t *testing.T) {
	var v struct {
		Hash16 []struct {
			In  []string `json:"in"`
			Out string   `json:"out"`
		} `json:"hash16"`
		Chain []struct {
			Vals    []string `json:"vals"`
			NChunks int      `json:"nChunks"`
			Out     string   `json:"out"`
		} `json:"chain"`
	}
	b, err := os.ReadFile("testdata/poseidon_vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, &v); err != nil {
		t.Fatal(err)
	}
	for i, c := range v.Hash16 {
		var in [16]fr.Element
		for j, s := range c.In {
			n, _ := new(big.Int).SetString(s, 10)
			in[j].SetBigInt(n)
		}
		got := poseidon16(&in)
		if got.String() != c.Out {
			t.Fatalf("hash16 case %d: got %s want %s", i, got.String(), c.Out)
		}
	}
	for i, c := range v.Chain {
		vals := make([]int64, len(c.Vals))
		for j, s := range c.Vals {
			n, _ := new(big.Int).SetString(s, 10)
			vals[j] = n.Int64()
		}
		got := poseidonChain(vals, c.NChunks)
		if got.String() != c.Out {
			t.Fatalf("chain case %d (with negatives): got %s want %s", i, got.String(), c.Out)
		}
	}
	t.Logf("%d raw hashes and %d chained digests (with negative inputs) match circomlibjs", len(v.Hash16), len(v.Chain))
}

func TestRealWeightsDigestMatchesPinnedHash(t *testing.T) {
	var m struct {
		WeightsHash string    `json:"weights_hash"`
		W           [][]int64 `json:"W_q"`
		B           []int64   `json:"b_q"`
	}
	b, err := os.ReadFile("testdata/b1_model_weights.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	flat := make([]int64, 0, 3084)
	for _, row := range m.W {
		flat = append(flat, row...)
	}
	flat = append(flat, m.B...)
	got := poseidonChain(flat, 206)
	if got.String() != pinnedModelWeightsHash {
		t.Fatalf("weights digest %s != pinned %s", got.String(), pinnedModelWeightsHash)
	}
	if m.WeightsHash != pinnedModelWeightsHash {
		t.Fatalf("cache hash %s != pinned %s", m.WeightsHash, pinnedModelWeightsHash)
	}
}
