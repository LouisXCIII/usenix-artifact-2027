# S11 — Vá amount không dương và assurance-claim trên chaincode authcc (seq 44, 2026-10-06)

Phạm vi: đóng hai lỗ hổng đã biết trong báo cáo S3/S10. (1) `SubmitDecisionFull` chấp nhận amount ≤ 0 (26/26 ca). (2) Nửa assurance-claim: `SubmitDecisionAssuranceV2` nhận `semBound`/`evAssuranceOK` từ caller.

## Thay đổi mã
- Guard `amount <= 0` → lỗi "amount must be positive" trên 6 entry point production: SubmitDecision, SubmitDecisionFull, SubmitDecisionObligationV2, SubmitDecisionAssuranceV2, SubmitDecisionTrustlessV7, SubmitDecisionTrustlessV7Obligation.
- `SubmitDecisionAssuranceV3` (assurance_v3.go): semBound và evAssuranceOK được suy từ chứng cứ thật (ZK proof, header Besu, proven mlScanPassed/approvals, feature binding), như V7Obligation. assureOK chỉ lấy từ verifiedTier. V2 giữ nguyên cho các đo đạc đã công bố.
- Biến thể ablation/baseline (ví dụ SubmitDecisionTrustlessWeakQuorum, SubmitDecisionFullNoObligationCheck, SubmitDecisionB0/B2/B3) KHÔNG đổi: vẫn chấp nhận amount không dương. Đây là chủ đích để giữ so sánh đã đo.

## Kiểm thử
- `go build`, `go vet`, `go test ./chaincode` (repo `~/fabric-network/authcc`): đạt.
- `amount_guard_test.go` (mới): 6 entry point × amount {0, -1, -100000} đều trả lỗi amount-positive và không ghi state. Đạt.
- Trên chain (query, không ghi): amount -5 → "amount must be positive (got -5)". Hàm B1 đã gỡ → "function not found".

## Kiểm thử V3 trên chain (block 619309, ledger clock đã SetClock về now+300)
| Ca | Chứng cứ | Yêu cầu | Kết quả | semBound | evOK | overClaim |
|---|---|---|---|---|---|---|
| A | honest biến thể 04 | ClassicalDecisionAuth | EXACT | true | true | — |
| B | honest 04 | PQDecisionAuth | REJECT | true | true | — |
| C | honest 04, khai pq/xác minh classical | ClassicalDecisionAuth | EXACT | true | true | true |
| D | forged 04 (featureHash không khớp) | ClassicalDecisionAuth | REJECT | false | true | — |

Ý nghĩa: B cho thấy cổng assurance hoạt động; C cho thấy over-claim được ghi nhận nhưng không cấp quyền thêm; D cho thấy chứng cứ giả bị chặn ở tầng semBound. Bản ghi test vẫn nằm trên chain.

## Triển khai
- Sequence 44, version 2.32, cả hai org đã approve, định nghĩa đã commit trên authchannel (xác nhận `querycommitted`).
- Dữ liệu cũ vẫn đọc được (`ReadDecision` của `s8-t-c000-1790605271`).
- Mã nguồn commit tại `~/fabric-network/authcc` (aa547cc). Không commit các file B1 có trọng số mô hình.

## Hạn chế cần nêu trong bài
- Bộ kiểm thử V3 là 4 ca trên một proof; chưa có bộ ca thử quy mô.
- Ledger clock được đặt lại (SetClock) trong lúc thử; giá trị chỉ tăng.
- Registry cũ `0x2671c586…` vẫn còn trên chain và chưa gỡ được.
- Số liệu của `SubmitDecisionFull` với amount không dương trong các bản báo cáo trước đã lỗi thời cho entry point production.
