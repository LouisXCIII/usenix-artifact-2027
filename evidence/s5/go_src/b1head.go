// b1head.go -- S5-A: exact fixed-point re-execution in Go of the relation proved by circuits_v4/ml_scan_proof_v4.circom, i.e. the WHOLE
// classifier head 576 -> 512 (LayerNorm+ReLU) -> 256 (LayerNorm+ReLU) -> 12, about 429,000 multiply-accumulates. It mirrors the circuit's
// integer semantics: signed constant division and division by std truncate toward zero, std = floor(sqrt(var + 168)), ReLU by sign.
// Protocol: evidence/s5_deep_dive_protocol_2026-09-28.md. Not part of the deployed chaincode.
package chaincode

import (
	"bytes"
	"encoding/json"
	"math"
	"strconv"
)

const (
	hN0, hN1, hN2, hC = 576, 512, 256, 12
	hScale            = 4096
	hEpsS2            = 168 // round(1e-5 * 4096 * 4096), as in the circuit
)

type headModel struct {
	W0          []int64 // [512][576] row-major
	B0, G1, Be1 []int64
	W4          []int64 // [256][512]
	B4, G5, Be5 []int64
	W8          []int64 // [12][256]
	B8          []int64
}

func numsToInt64(raw []json.Number) []int64 {
	out := make([]int64, len(raw))
	for i, n := range raw {
		v, err := strconv.ParseInt(n.String(), 10, 64)
		if err != nil {
			panic("value outside int64: " + n.String())
		}
		out[i] = v
	}
	return out
}

func flat2(m [][]json.Number) []int64 {
	var out []int64
	for _, r := range m {
		out = append(out, numsToInt64(r)...)
	}
	return out
}

// headLoad decodes a circuit input file (the same JSON the circuit consumes) into the model and the input vector x0.
func headLoad(raw []byte) (*headModel, []int64, error) {
	var d struct {
		X0  []json.Number   `json:"x0"`
		W0  [][]json.Number `json:"W0"`
		B0  []json.Number   `json:"b0"`
		G1  []json.Number   `json:"gamma1"`
		Be1 []json.Number   `json:"beta1"`
		W4  [][]json.Number `json:"W4"`
		B4  []json.Number   `json:"b4"`
		G5  []json.Number   `json:"gamma5"`
		Be5 []json.Number   `json:"beta5"`
		W8  [][]json.Number `json:"W8"`
		B8  []json.Number   `json:"b8"`
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	if err := dec.Decode(&d); err != nil {
		return nil, nil, err
	}
	m := &headModel{W0: flat2(d.W0), B0: numsToInt64(d.B0), G1: numsToInt64(d.G1), Be1: numsToInt64(d.Be1),
		W4: flat2(d.W4), B4: numsToInt64(d.B4), G5: numsToInt64(d.G5), Be5: numsToInt64(d.Be5),
		W8: flat2(d.W8), B8: numsToInt64(d.B8)}
	return m, numsToInt64(d.X0), nil
}

func isqrt(n int64) int64 {
	if n <= 0 {
		return 0
	}
	x := int64(math.Sqrt(float64(n)))
	for x*x > n {
		x--
	}
	for (x+1)*(x+1) <= n {
		x++
	}
	return x
}

// dense computes out[j] = b[j] + sum_i w[j*in+i]*x[i], unrolled by four (the loop a careful implementer would write in Go).
func dense(out, x, w, b []int64, in int) {
	for j := range out {
		row := w[j*in : j*in+in]
		s := b[j]
		i := 0
		for ; i+4 <= in; i += 4 {
			s += row[i]*x[i] + row[i+1]*x[i+1] + row[i+2]*x[i+2] + row[i+3]*x[i+3]
		}
		for ; i < in; i++ {
			s += row[i] * x[i]
		}
		out[j] = s
	}
}

func layerNormReLU(z, gamma, beta []int64) []int64 {
	dim := int64(len(z))
	var sum int64
	for _, v := range z {
		sum += v
	}
	mean := sum / dim
	var sumsq int64
	cen := make([]int64, len(z))
	for i, v := range z {
		c := v - mean
		cen[i] = c
		sumsq += c * c
	}
	std := isqrt(sumsq/dim + hEpsS2)
	if std == 0 {
		std = 1
	}
	out := make([]int64, len(z))
	for i := range z {
		norm := (cen[i] * hScale) / std
		ln := (norm*gamma[i])/hScale + beta[i]
		if ln < 0 {
			ln = 0
		}
		out[i] = ln
	}
	return out
}

// headForward returns the 12 logits (scale 4096^2) for the input x0.
func headForward(m *headModel, x0 []int64) [hC]int64 {
	z0 := make([]int64, hN1)
	dense(z0, x0, m.W0, m.B0, hN0)
	for i := range z0 {
		z0[i] /= hScale
	}
	a0 := layerNormReLU(z0, m.G1, m.Be1)
	z4 := make([]int64, hN2)
	dense(z4, a0, m.W4, m.B4, hN1)
	for i := range z4 {
		z4[i] /= hScale
	}
	a4 := layerNormReLU(z4, m.G5, m.Be5)
	var logits [hC]int64
	dense(logits[:], a4, m.W8, m.B8, hN2)
	return logits
}
