# Thực nghiệm concurrency/TOCTOU (thật, on-chain) — mạng authcc — 2026-09-21

## Lý do
Đọc kỹ toàn bộ bài (~2200 dòng), mục Limitations tự nêu rõ 2 khoảng trống liên quan:
- "No concurrency or scalability measurement... every result is a sequential real transaction"
- "TOCTOU is closed architecturally but not stress-tested... under concurrent policy/state churn"

User xác nhận làm cả 2 phần: (a) race condition thật trên cùng 1 decisionID, (b) đo latency thật
dưới tải đồng thời (so với N=10 tuần tự đã có, mean 2278ms).

## Phần (a): Race condition trên cùng decisionID

5 giao dịch `SubmitDecision` đồng thời, cùng `decisionID="decision-race-001"`, cùng tham số.

**Kết quả thật**: cả 5 đều được endorse (đọc cùng pre-state "chưa tồn tại"), cả 5 được order vào
CÙNG 1 block (block 20), nhưng khi validate: **chỉ 1 giao dịch commit VALID** (payload SAFE), **4
giao dịch còn lại bị invalidate với status MVCC_READ_CONFLICT** — đúng cơ chế optimistic
concurrency control chuẩn của Fabric. Xác nhận bằng `ReadDecision` sau đó: ledger chỉ có đúng 1
bản ghi, khớp với giao dịch thắng. Không có double-write.

## Phần (b): Latency dưới tải đồng thời thật

10 giao dịch `SubmitDecision` đồng thời, mỗi giao dịch 1 decisionID riêng
(`decision-concurrent-1..10`), đo wall-clock từ lúc gửi đến khi nhận VALID.

**Kết quả**: mean=100.8ms, sd=1.62ms, min=98ms, max=103ms — **nhanh hơn >20 lần** so với N=10
tuần tự đã có trong bài (mean 2278ms, sd 110ms).

**Giải thích cơ chế (đã xác minh, không suy đoán)**: fetch trực tiếp block 18-21 bằng
`peer channel fetch` + decode bằng `configtxlator`, đếm số giao dịch mỗi block:
- Block 18: 1 tx, Block 19: 1 tx (giao dịch tuần tự trước đó — mỗi tx 1 block riêng)
- Block 20: 5 tx (batch của phần (a))
- **Block 21: 10 tx** (batch của phần (b) — cả 10 giao dịch đồng thời rơi vào ĐÚNG 1 block)

→ Xác nhận: orderer cắt block theo **BatchSize** (ngưỡng số lượng giao dịch) khi đủ giao dịch đến
cùng lúc, thay vì đợi hết **BatchTimeout** (~2s mặc định) như khi tuần tự (mỗi giao dịch đứng 1
mình trong 1 block, phải đợi hết timeout). Đây là cơ chế thật của Fabric ordering service, không
phải hiện tượng lạ chưa giải thích được.

## Cập nhật: user chỉ ra tôi mới test "một phần" — mở rộng thêm 2 hướng

Sau khi báo cáo xong, user hỏi "sao không test hết" — đúng, tôi chỉ mới test race trùng lặp
(không phải TOCTOU thật: policy đổi giữa chừng) và chỉ 1 mức concurrency (N=10, không phải sweep).
Đã làm thêm:

### Quét concurrency vượt BatchSize (N=10, 20, 50)
Xác nhận `max_message_count=10` (BatchSize thật của channel, đọc từ channel config qua
`configtxlator`). Kết quả đo thật (fetch+decode block để đếm tx/block):

| N | mean latency | sd | số block thật |
|---|---|---|---|
| 10 | 100.8ms | 1.62ms | 1 |
| 20 | 165.3ms | 8.18ms | 2 |
| 50 | 361.5ms | 45.16ms | 5 |

Xu hướng tăng gần tuyến tính theo số block — không còn là 1 điểm dữ liệu duy nhất.

### TOCTOU thật (không phải race trùng lặp)
Sửa chaincode: thêm `SetRoleThreshold` (ghi threshold lên world-state) và
`SubmitDecisionDynamicPolicy` (đọc threshold từ world-state thay vì hard-code) — deploy v1.2/seq3.

Race thật: `SetRoleThreshold(manager, 400000)` chạy **đồng thời** với 1 quyết định
role=manager/amount=450000 (đã endorse dựa trên threshold cũ 500000):
- Giao dịch đổi policy: **VALID**.
- Giao dịch quyết định (đọc threshold cũ, đã lỗi thời): **MVCC_READ_CONFLICT** — bị từ chối, không
  hề "lọt" dưới policy cũ.
- Resubmit sau khi policy ổn định: **REJECT** đúng (450000 > 400000 mới).

→ Cơ chế: threshold giờ là 1 read on-chain tường minh, nên bị cuốn vào read-set của giao dịch —
Fabric MVCC tự động bắt được race chính sách bằng đúng cơ chế đã bắt race trùng lặp decisionID.
Đây là bằng chứng CƠ CHẾ thật (không chỉ giả định kiến trúc) cho lý do TOCTOU "closed".

## Đã viết vào bài
`submit_paper2/Manuscriptsp2_Full_v1.tex`:
- Thêm `\subsection{Concurrent submission...}` (label `subsec:eval-concurrency`) ngay sau bảng
  `tab:evidence-cost`, trước `\section{System Architecture...}`.
- Sửa 2 mục trong Limitations (TOCTOU, concurrency/scalability) để phản ánh đã test 1 phần, còn
  lại (policy/state churn giữa lúc cấp chứng chỉ, quy mô >10 giao dịch đồng thời, độ phức tạp
  policy/hop count) vẫn mở.

Build PDF sạch, 30 trang (từ 29) — sau đó thêm 2 phần mở rộng ở trên, build lại sạch, **31 trang**.

## File liên quan
- Script test: `/tmp/claude-1001/.../scratchpad/race_test.sh`,
  `concurrent_latency_test.sh` (tạm, trong scratchpad phiên này).
- Ledger thêm: `decision-race-001` (1 bản ghi thắng), `decision-concurrent-1..10` (10 bản ghi) —
  không đụng dữ liệu cũ.
- Block thật đã fetch/decode: block 18-21 trên channel `authchannel`.
