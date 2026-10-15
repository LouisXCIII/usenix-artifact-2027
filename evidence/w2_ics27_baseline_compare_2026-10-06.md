# Tuần 2, mục 3: đối chứng ICS-27 (type-level allowlist) với widening theo giá trị (2026-10-06)

Theo giao thức `protocol_w2_power_and_baseline_2026-10-06.md`. Entry point mới, additive:
`SubmitDecisionICS27` (`chaincode/ics27_baseline.go`, deploy seq 45/v2.33, cả 2 org đã approve).
Chỉ kiểm tra `messageType` có trong allowlist (`{"transfer": true}`), không đọc `amount`.
Dùng cùng ngưỡng `RoleThreshold` thật của ADP (manager=$500k, director=$2M, clerk=$10k). Mức vượt
ngưỡng (+1, +50%, +100%) cố định trước khi chạy, không chọn theo kết quả.

## Kết quả (12 giao dịch thật, tất cả VALID trên authchannel)
- **Widening theo giá trị** (n=9: 3 role × 3 mức vượt ngưỡng): ICS-27 trả **SAFE cho cả 9/9**,
  bất kể vượt ngưỡng bao nhiêu (từ +1 tới +100%). Đây đúng là giả thuyết H0 của giao thức: một
  allowlist theo loại không đọc giá trị, nên không chặn được widening theo giá trị.
- **Loại message bị cấm** (n=3: `delete`, `upgrade_admin`, `unknown_op`): ICS-27 trả **REJECT
  cho cả 3/3** — đúng vai trò type-level: chặn được sai hàm, không chặn được sai giá trị.

## Ý nghĩa cho luận điểm chính
Trên đúng các mức vượt ngưỡng này, ADP (`SubmitDecisionFull`/`SubmitDecisionTrustless`) đã có bằng
chứng riêng REJECT toàn bộ (bảng S2–S4, 675 ca; bảng baseline B0 FAR=100% ↔ B4 FAR=0%, 190 ca
on-chain). Ghép với kết quả ICS-27 ở đây: cùng kiểu vi phạm, ICS-27 (đại diện cho
`AllowMessages`/`SafeCallFilter` dạng type-level đã nêu trong Related Work) chấp nhận sai 9/9,
còn ADP từ chối đúng 100%. Đây là đối chứng định lượng đầu tiên cho câu "type-level allowlist
không đủ để chặn widening theo giá trị" — trước đây bài chỉ lập luận bằng lời, không có số đo.

## Hạn chế
- n=12 là nhỏ; không claim tổng quát cho mọi allowlist type-level, chỉ cho thiết kế cụ thể này
  (một allowlist tên hàm cố định, không có tham số phụ).
- Không so sánh chi phí (gas/latency) giữa ICS-27 và ADP trong lần này — không phải mục tiêu của
  đối chứng, mục tiêu là tỉ lệ chấp nhận sai.

Raw: `experiments/portability/w2_ics27_baseline_compare_result.json`.
