# RBAC-only vs ABAC (thật, on-chain) — mạng authcc — 2026-09-21

## Bối cảnh
Bài đã có sẵn framework baseline B0-B4 + Bsemless (24-case corpus), trong đó B2 ("syntactic
mapping" — chỉ nhận diện role, bỏ qua SemBound/EvAssuranceOK) gần tương đương RBAC-thuần. NHƯNG
các baseline này được đánh giá **offline** ("by direct application of their own weaker rule, not
re-implemented as a second copy of the chaincode logic" — trích nguyên văn bài). Thí nghiệm này bổ
sung: triển khai RBAC-thuần **thành chaincode thật**, gọi **giao dịch thật** trên mạng `authcc`
đang chạy, không phải đánh giá offline.

## Thay đổi chaincode
- Thêm `SubmitDecisionRBAC` + `classifyRBAC` vào `authchaincode.go` (song song `SubmitDecision`/
  `classify` hiện có, không sửa code cũ).
- Logic: dùng đúng bảng `RoleThreshold` y hệt ABAC, chỉ check `amount ≤ threshold` theo role —
  **bỏ hoàn toàn** `SemBound`/`EvAssuranceOK`.
- Compile sạch (`go build ./...`).
- Deploy lên mạng thật: nâng cấp chaincode definition `authcc` từ version 1.0/sequence 1 lên
  **version 1.1/sequence 2** (package, install cả 2 peer, approve cả Org1MSP+Org2MSP, commit) —
  đúng quy trình lifecycle chuẩn Fabric, không đụng `network.sh up/down` (tránh rủi ro đã gặp
  trước đây).

## Kết quả 3 ca thật (invoke thật, waitForEvent, commit VALID trên cả 2 peer)

| Ca | Role | Amount | SemBound | ABAC (`SubmitDecision`) | RBAC-thuần (`SubmitDecisionRBAC`) | Đồng thuận? |
|---|---|---|---|---|---|---|
| 1 — tấn công (giả lập decision không ràng buộc ngữ nghĩa) | manager | $300,000 | **false** | **REJECT** | **SAFE** | **KHÁC — RBAC sai** |
| 2 — hợp lệ thật | manager | $300,000 | true | SAFE | SAFE | Giống nhau |
| 3 — vượt ngưỡng | clerk | $50,000 (ngưỡng $10,000) | true | REJECT | REJECT | Giống nhau |

Cả 6 giao dịch đều commit VALID trên cả peer0.org1 và peer0.org2 (txid thật, có trong log).

## Ý nghĩa
- Xác nhận đúng dự đoán của B2 (offline) **bằng giao dịch thật trên ledger thật**: nếu dùng RBAC-
  thuần làm chính sách destination, ca tấn công (role hợp lệ, amount trong ngưỡng, nhưng thiếu
  ràng buộc ngữ nghĩa) sẽ được **APPROVE SAI** — đúng là kiểu lỗi Theorem T0 cảnh báo.
- KHÔNG cherry-pick: báo cáo đầy đủ cả 2 ca đồng thuận (hợp lệ + vượt ngưỡng), không chỉ ca khác
  biệt.
- Không thay đổi số liệu B2 đã đo trên corpus 24-case (8/24, 11/17 unsafe accepted) — đây là bằng
  chứng bổ sung, xác nhận số liệu đó không phải hiện tượng riêng của cách đánh giá offline.

## Đã viết vào bài
`submit_paper2/Manuscriptsp2_Full_v1.tex` — thêm 1 đoạn `\paragraph{}` ngay sau bảng
`tab:corpus-results`, trước `\subsection{B_semless...}`. Build PDF sạch, 29 trang.

## File liên quan
- `/home/deployer/fabric-network/authcc/chaincode/authchaincode.go` — code đã sửa (thêm, không
  đổi code cũ).
- Chaincode definition `authcc` hiện tại: version 1.1, sequence 2, committed trên channel
  `authchannel`, cả Org1MSP+Org2MSP approved.
- Ledger hiện có thêm 6 decision mới: `decision-cmp-abac-001/002/003`,
  `decision-cmp-rbac-001/002/003` — không đụng 5 decision cũ (`decision-001..005`).
