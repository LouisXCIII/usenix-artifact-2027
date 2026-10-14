// Công cụ ký độc lập cho người xác minh thứ hai (domain authority) của thực nghiệm q_D.
//
// Công cụ này CHỈ làm 2 việc máy móc: (1) tạo khóa secp256k1, (2) ký một "digest" cố định
// trên dữ liệu của một ca cụ thể cộng với kết luận "passed" do người dùng tự xác định.
// Nó KHÔNG chứa bất kỳ logic nào để tự tính ra "passed" -- người chạy công cụ này phải tự
// đọc đặc tả (dac_ta_quy_tac_nghia_vu.md) và tự quyết định giá trị --passed, nếu không thì
// phán đoán không còn độc lập.
//
// Lược đồ digest và cách suy ra địa chỉ công khai khớp CHÍNH XÁC với
// q_d_domain_attestation.go (awsAttestationDigest / verifyAwsDomainAttestation) của chaincode
// authcc, để chữ ký tạo ra ở đây xác minh được trực tiếp trên chain thật sau khi địa chỉ công
// khai được đưa vào làm một "domain authority" mới.
package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"os"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/decred/dcrd/dcrec/secp256k1/v4/ecdsa"
	"golang.org/x/crypto/sha3"
)

const keyFile = "advisor_domain_key.hex"

func keccak256(parts ...[]byte) []byte {
	h := sha3.NewLegacyKeccak256()
	for _, p := range parts {
		h.Write(p)
	}
	return h.Sum(nil)
}

func addressFromPub(pub *secp256k1.PublicKey) string {
	full := pub.SerializeUncompressed()[1:]
	addr := keccak256(full)[12:]
	return "0x" + hex.EncodeToString(addr)
}

func cmdKeygen() {
	if _, err := os.Stat(keyFile); err == nil {
		fmt.Printf("CẢNH BÁO: %s đã tồn tại -- không tạo khóa mới (tránh ghi đè khóa đang dùng).\n", keyFile)
		os.Exit(1)
	}
	priv, err := secp256k1.GeneratePrivateKey()
	if err != nil {
		fmt.Println("Lỗi tạo khóa:", err)
		os.Exit(1)
	}
	if err := os.WriteFile(keyFile, []byte(hex.EncodeToString(priv.Serialize())), 0o600); err != nil {
		fmt.Println("Lỗi lưu khóa:", err)
		os.Exit(1)
	}
	addr := addressFromPub(priv.PubKey())
	fmt.Println("Đã tạo khóa mới, lưu cục bộ tại:", keyFile)
	fmt.Println("KHÔNG gửi file này cho bất kỳ ai -- đây là khóa riêng.")
	fmt.Println()
	fmt.Println("Địa chỉ công khai (gửi lại đúng dòng này):")
	fmt.Println(addr)
}

func loadKey() *secp256k1.PrivateKey {
	data, err := os.ReadFile(keyFile)
	if err != nil {
		fmt.Println("Chưa có khóa -- chạy 'go run main.go keygen' trước:", err)
		os.Exit(1)
	}
	b, err := hex.DecodeString(string(data))
	if err != nil {
		fmt.Println("File khóa hỏng:", err)
		os.Exit(1)
	}
	return secp256k1.PrivKeyFromBytes(b)
}

// digestFor khớp chính xác awsAttestationDigest trong q_d_domain_attestation.go:
// keccak256(decisionID || obligationDischargeRef || sourceDomain || role || amount(8B BE) || passedByte)
func digestFor(decisionID, dischargeRef, sourceDomain, role string, amount int64, passed bool) []byte {
	amountBytes := make([]byte, 8)
	for i := 0; i < 8; i++ {
		amountBytes[7-i] = byte(amount >> (8 * i))
	}
	passedByte := byte(0x00)
	if passed {
		passedByte = 0x01
	}
	return keccak256([]byte(decisionID), []byte(dischargeRef), []byte(sourceDomain), []byte(role), amountBytes, []byte{passedByte})
}

func cmdSign(args []string) {
	fs := flag.NewFlagSet("sign", flag.ExitOnError)
	decisionID := fs.String("decision-id", "", "decisionId của ca đang ký")
	dischargeRef := fs.String("discharge-ref", "", "obligationDischargeRef của ca đang ký")
	sourceDomain := fs.String("source-domain", "", "sourceDomain của ca đang ký")
	role := fs.String("role", "", "role của ca đang ký")
	amount := fs.Int64("amount", 0, "amount của ca đang ký")
	passed := fs.String("passed", "", "'true' hoặc 'false' -- kết luận ĐỘC LẬP của người ký, không copy từ đâu khác")
	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}
	if *passed != "true" && *passed != "false" {
		fmt.Println("--passed phải là 'true' hoặc 'false'")
		os.Exit(1)
	}
	passedBool := *passed == "true"

	priv := loadKey()
	digest := digestFor(*decisionID, *dischargeRef, *sourceDomain, *role, *amount, passedBool)
	sig := ecdsa.SignCompact(priv, digest, true)

	fmt.Println("decisionId:", *decisionID)
	fmt.Println("passed (kết luận của người ký):", *passed)
	fmt.Println("Chữ ký (gửi lại đúng chuỗi hex dưới đây):")
	fmt.Println(hex.EncodeToString(sig))
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Dùng: go run main.go keygen")
		fmt.Println("  hoặc: go run main.go sign --decision-id=... --discharge-ref=... --source-domain=... --role=... --amount=... --passed=true|false")
		os.Exit(1)
	}
	switch os.Args[1] {
	case "keygen":
		cmdKeygen()
	case "sign":
		cmdSign(os.Args[2:])
	default:
		fmt.Println("Lệnh không hợp lệ:", os.Args[1])
		os.Exit(1)
	}
}
