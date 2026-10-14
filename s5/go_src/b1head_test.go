package chaincode

// S5-A harness. env: S5_DIR = evidence/s5 (oracle + real v4 proof/vkey), S5_OUT = output prefix. Skipped when S5_DIR is unset.

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"testing"
	"time"
)

type s5Oracle struct {
	File               string  `json:"file"`
	PyLogits           []int64 `json:"py_logits"`
	PyPred             int     `json:"py_pred"`
	CircuitSatisfiable bool    `json:"circuit_satisfiable"`
	CircuitRunningMax  []int64 `json:"circuit_running_max"`
	PyRunningMax       []int64 `json:"py_running_max"`
}

func s5Dir(t testing.TB) string {
	d := os.Getenv("S5_DIR")
	if d == "" {
		t.Skip("S5_DIR not set")
	}
	return d
}

func s5Oracles(t testing.TB) map[string]s5Oracle {
	raw, err := os.ReadFile(filepath.Join(s5Dir(t), "s5_oracle_v4.json"))
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]s5Oracle
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	return m
}

func runningMax(l [hC]int64) []int64 {
	out := make([]int64, hC)
	cur := l[0]
	for j := 0; j < hC; j++ {
		if l[j] > cur {
			cur = l[j]
		}
		out[j] = cur
	}
	return out
}

// Parity: the Go re-execution equals the independent Python fixed-point reference on all 23 real inputs, and the circuit's own witness
// (running maximum of the logits) on the 22 inputs for which the circuit is satisfiable.
func TestS5HeadParity(t *testing.T) {
	or := s5Oracles(t)
	names := make([]string, 0, len(or))
	for n := range or {
		names = append(names, n)
	}
	sort.Strings(names)
	circ, py := 0, 0
	for _, n := range names {
		o := or[n]
		raw, err := os.ReadFile(o.File)
		if err != nil {
			t.Fatal(err)
		}
		m, x0, err := headLoad(raw)
		if err != nil {
			t.Fatal(err)
		}
		l := headForward(m, x0)
		for j := 0; j < hC; j++ {
			if l[j] != o.PyLogits[j] {
				t.Fatalf("%s: logit %d Go %d != Python reference %d", n, j, l[j], o.PyLogits[j])
			}
		}
		py++
		if o.CircuitSatisfiable {
			rm := runningMax(l)
			for j := range rm {
				if rm[j] != o.CircuitRunningMax[j] {
					t.Fatalf("%s: running max %d Go %d != circuit witness %d", n, j, rm[j], o.CircuitRunningMax[j])
				}
			}
			circ++
		}
	}
	t.Logf("Go head re-execution equals the Python reference on %d/%d inputs (all 12 logits) and the circuit witness on %d inputs", py, len(names), circ)
}

func med(a []int64) int64 {
	b := append([]int64(nil), a...)
	sort.Slice(b, func(i, j int) bool { return b[i] < b[j] })
	return b[len(b)/2]
}

// Timing: Groth16 verification of the REAL v4 proof (with argument and key decoding) versus re-executing the whole head, interleaved.
func TestS5HeadTiming(t *testing.T) {
	dir := s5Dir(t)
	or := s5Oracles(t)
	rawA, _ := os.ReadFile(or["A_v4"].File)
	m, _, err := headLoad(rawA)
	if err != nil {
		t.Fatal(err)
	}
	pfJ, _ := os.ReadFile(filepath.Join(dir, "v4_proof.json"))
	pubJ, _ := os.ReadFile(filepath.Join(dir, "v4_public.json"))
	vkJ, _ := os.ReadFile(filepath.Join(dir, "v4_vkey.json"))
	var pf groth16Proof
	var pub []string
	var vk groth16VKey
	_ = json.Unmarshal(pfJ, &pf)
	_ = json.Unmarshal(pubJ, &pub)
	_ = json.Unmarshal(vkJ, &vk)
	if ok, err := verifyGroth16(vk, pf, pub); err != nil || !ok {
		t.Fatalf("the real v4 proof does not verify in the Go verifier: %v %v", ok, err)
	}
	var xs [][]int64
	var xJSON []string
	names := make([]string, 0)
	for n, o := range or {
		if o.CircuitSatisfiable {
			names = append(names, n)
		}
	}
	sort.Strings(names)
	for _, n := range names {
		raw, _ := os.ReadFile(or[n].File)
		_, x0, _ := headLoad(raw)
		xs = append(xs, x0)
		b, _ := json.Marshal(x0)
		xJSON = append(xJSON, string(b))
	}
	var sink int64
	g16 := func() int64 {
		t0 := time.Now()
		var a groth16Proof
		var b []string
		var k groth16VKey
		_ = json.Unmarshal(pfJ, &a)
		_ = json.Unmarshal(pubJ, &b)
		_ = json.Unmarshal(vkJ, &k)
		ok, _ := verifyGroth16(k, a, b)
		el := time.Since(t0).Nanoseconds()
		if !ok {
			t.Fatal("verify failed")
		}
		return el
	}
	head := func(i int, sha bool) int64 {
		t0 := time.Now()
		var x []int64
		_ = json.Unmarshal([]byte(xJSON[i]), &x)
		l := headForward(m, x)
		best := 0
		for j := 1; j < hC; j++ {
			if l[j] > l[best] {
				best = j
			}
		}
		if sha {
			buf := make([]byte, 0, 8*len(x))
			for _, v := range x {
				buf = binary.BigEndian.AppendUint64(buf, uint64(v))
			}
			d := sha256.Sum256(buf)
			sink += int64(d[0])
		}
		el := time.Since(t0).Nanoseconds()
		sink += int64(best)
		return el
	}
	reps := 30
	var G, H, HS []int64
	for w := 0; w < 10; w++ {
		g16()
		head(0, false)
	}
	for i := range xs {
		for r := 0; r < reps; r++ {
			runtime.GC()
			for k := 0; k < 3; k++ {
				switch (r + k) % 3 {
				case 0:
					G = append(G, g16())
				case 1:
					H = append(H, head(i, false))
				case 2:
					HS = append(HS, head(i, true))
				}
			}
		}
	}
	res := map[string]interface{}{"inputs": len(xs), "reps_per_input": reps, "groth16_v4_verify_ns": G, "head_reexec_ns": H, "head_reexec_plus_sha_ns": HS,
		"macs": 576*512 + 512*256 + 256*12, "public_signals": len(pub), "sink": sink}
	b, _ := json.Marshal(res)
	_ = os.WriteFile(os.Getenv("S5_OUT")+"_head_timing.json", b, 0o644)
	t.Logf("Groth16 v4 verify median %.3f ms; head re-execution median %.3f ms; with SHA-256 %.3f ms (%d MACs)",
		float64(med(G))/1e6, float64(med(H))/1e6, float64(med(HS))/1e6, 576*512+512*256+256*12)
}

// Synthetic scaling (LABELLED SYNTHETIC): one dense layer of random small integers; measures the Go loop only.
func TestS5SyntheticScaling(t *testing.T) {
	s5Dir(t)
	rnd := rand.New(rand.NewSource(20260928))
	type row struct {
		In, Out  int
		Macs     int
		Reps     int
		MedNs    int64
		NsPerMac float64
	}
	var rows []row
	for _, sz := range [][2]int{{256, 12}, {256, 128}, {512, 256}, {576, 512}, {1024, 1024}, {2048, 1280}, {2048, 2048}, {4096, 4096}, {8192, 8192}} {
		in, out := sz[0], sz[1]
		w := make([]int64, in*out)
		for i := range w {
			w[i] = int64(rnd.Intn(8193) - 4096)
		}
		x := make([]int64, in)
		for i := range x {
			x[i] = int64(rnd.Intn(8193) - 4096)
		}
		b := make([]int64, out)
		o := make([]int64, out)
		macs := in * out
		reps := 200
		if macs > 100000 {
			reps = 40
		}
		if macs > 3000000 {
			reps = 15
		}
		if macs > 30000000 {
			reps = 5
		}
		for i := 0; i < 3; i++ {
			dense(o, x, w, b, in)
		}
		var s []int64
		for r := 0; r < reps; r++ {
			t0 := time.Now()
			dense(o, x, w, b, in)
			s = append(s, time.Since(t0).Nanoseconds())
		}
		md := med(s)
		rows = append(rows, row{in, out, macs, reps, md, float64(md) / float64(macs)})
		t.Logf("%5d x %-5d = %9d MACs  median %10.3f ms  %.3f ns/MAC", in, out, macs, float64(md)/1e6, float64(md)/float64(macs))
	}
	b, _ := json.MarshalIndent(rows, "", " ")
	_ = os.WriteFile(os.Getenv("S5_OUT")+"_synthetic_scaling.json", b, 0o644)
}
