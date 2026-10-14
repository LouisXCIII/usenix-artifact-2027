> **ĐÃ ĐƯỢC SỬA — đọc `trustless_destination_verification_2026-09-21.md` trước.** Hai điểm trong file này đã lỗi thời: (1) proof_A_v5 dùng ở đây KHÔNG được ràng với quyết định (public `solidityCodeHash` ≠ codeHash trong registry) — đã sửa bằng ràng buộc codeHash + proof SafeBank thật; (2) ngưỡng `minConfidence>=8000` đã đổi thành 7000 sau khi thấy độ tin cậy thật của SafeBank là 0.7739. Chaincode hiện là v1.6 (seq 7).

# Destination tự verify Groth16 proof thật cho SemBound — 2026-09-21

## Vấn đề đóng (Limitations item 1, đã thêm cùng ngày)

`SubmitDecision`/`SubmitDecisionFull` nhận `semBound`/`evAssuranceOk` như tham số
boolean do caller/relay truyền vào; destination chưa từng tự verify zk-SNARK
proof hay finality. Đây là điểm phê bình nặng nhất trong bài review bên ngoài
đã đọc cùng ngày (mục 2.2/2.3), và đã tự xác nhận bằng cách đọc trực tiếp
`authchaincode.go` (dòng 76-77 cũ: `semBound bool, evAssuranceOK bool` là
tham số hàm).

## Đã làm: `SubmitDecisionVerified` — verify Groth16 thật trong chaincode

- File mới `fabric-network/authcc/chaincode/zkverify.go`: verifier Groth16
  (BN254 pairing) viết bằng Go, dùng `github.com/consensys/gnark-crypto`
  (v0.17.0, ghim thấp để tương thích toolchain Go 1.22/1.23 sẵn có trong môi
  trường build chaincode — bản mới nhất v0.21.0 đòi Go≥1.25, không dùng được
  trong container build vendor hiện tại).
- Verification key **tái dùng đúng file thật** đã tồn tại từ trước
  (`verification_key_v5.json`, sinh bởi snarkjs cho circuit `ml_scan_proof_v5`,
  cùng key `MLScanVerifierV5.sol` trên Besu source domain đã dùng) — embed
  bằng `go:embed`, không phát minh key mới.
- Đúng theo T0's public-signal contract của circuit v5: `[scanPassed,
  solidityCodeHash, commitmentHash, maxVulnClass, minConfidence]`. Destination
  không chỉ pairing-check proof mà còn tự áp **chính sách của mình**
  (`maxVulnClass` phải đúng bằng 0, `minConfidence` phải ≥ 8000) thay vì tin
  ngưỡng do prover tự khai — đóng đúng lỗ hổng review đã chỉ ra ("proof hợp lệ
  cho một policy yếu hơn vẫn được pairing-check chấp nhận nếu không kiểm thêm").
- `SubmitDecisionVerified` không còn tham số `semBound` — về cấu trúc, không
  còn đường nào để caller tự khai giá trị này; nó luôn được tính từ
  `verifySemBoundProof()`.

## Xác minh từng lớp (thật, không giả lập)

1. **Prototype độc lập** (`/tmp/.../scratchpad/zkverify/`): verify
   `proof_A_v5.json`/`public_A_v5.json`/`verification_key_v5.json` thật (đã
   tồn tại từ phiên trước, đã được `MLScanVerifierV5.sol` trên Besu xác nhận
   hợp lệ) bằng gnark-crypto → `valid=true` (đúng ngay lần thử "không hoán vị
   tọa độ G2" — xác nhận quy ước snarkjs/gnark-crypto khớp trực tiếp, không
   cần transform). Thử 2 ca âm: proof bị sửa 1 chữ số → `valid=false`; public
   signal `maxVulnClass` bị sửa 0→5 → `valid=false`.
2. **Go test thật trên code chaincode** (`zkverify_test.go`, 4 test, dùng
   đúng file test data thật trong `chaincode/testdata/`): genuine → pass,
   tampered proof → reject, policy mismatch → reject, malformed JSON → error.
   Toàn bộ 4/4 PASS.
3. **Deploy thật lên mạng Fabric đang chạy** (`authcc-network`, project
   compose "authcc"): chaincode v1.4→v1.5 (sequence 5→6, v1.5 chỉ sửa
   metadata tag `optional` để `ReadDecision` không bị lỗi schema — một bug có
   sẵn từ trước, không phải do lần sửa này gây ra, nhưng phải sửa mới đọc lại
   được record).
4. **2 giao dịch thật, đã mined**, gọi `SubmitDecisionVerified` qua
   `peer chaincode invoke` thật (không dùng `eth_estimateGas`/dry-run kiểu
   nào):
   - `zk-genuine-001`: proof thật, public signals thật → **SAFE**.
   - `zk-tampered-001`: cùng public signals, proof bị sửa 1 chữ số ở `pi_a`
     → **REJECT**.
5. **Đọc lại on-chain** qua `ReadDecision` (không phải log giao dịch, mà state
   thật đã commit), xác nhận trường mới `zkVerified`/`zkVerificationDetail`:
   ```json
   {"decisionId":"zk-genuine-001", ..., "semBound":true, "transferClass":"SAFE",
    "zkVerified":true, "zkVerificationDetail":"Groth16 proof verified on-chain
    at the destination; public signals meet destination policy (maxVulnClass,
    minConfidence)"}
   {"decisionId":"zk-tampered-001", ..., "semBound":false, "transferClass":"REJECT",
    "zkVerified":true, "zkVerificationDetail":"Groth16 pairing check failed:
    proof does not verify against the embedded verification key"}
   ```

## Việc chưa làm (thuộc phần 2, đang làm tiếp)

- `EvAssuranceOK` vẫn là boolean do caller cung cấp trong
  `SubmitDecisionVerified` — threat (a) "malicious relay" mới đóng cho nửa
  SemBound, chưa đóng cho nửa EvAssuranceOK/finality. Cần light-client Besu
  (header + chữ ký ≥2f+1 validator + Merkle-Patricia proof của storage slot)
  — đang triển khai tiếp, xem file evidence riêng khi xong.
- Chưa cập nhật `Manuscriptsp2_Full_v1.tex` (Implementation/Evaluation/
  Limitations) với kết quả này — sẽ làm sau khi Component B (light-client)
  xong, để viết một lượt thống nhất cho cả hai nửa RQ3.
- Chưa relay corpus-scale (mới 2 giao dịch minh họa, chưa chạy lại toàn bộ
  24-case corpus / 7-case dishonest-provider qua đường `SubmitDecisionVerified`
  mới).
