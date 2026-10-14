# Giao thức tuần 2 (đăng ký trước khi chạy), 2026-10-06

Mục đích: tăng sức mạnh bằng chứng cho các cơ chế có cỡ mẫu on-chain nhỏ, và thêm một đối chứng
type-level (kiểu ICS-27 AllowMessages) để kiểm tra luận điểm chính. Giao thức này được viết TRƯỚC khi
chạy bất kỳ thí nghiệm mới nào, để không chỉnh tiêu chí theo kết quả.


## 0. Sửa đổi trước khi chạy (đăng ký trước, không sửa sau khi có số)
Bản đầu đề xuất "lặp cùng 5 loại đột biến với tham số khác". Đây là pseudo-replication: các đột biến
tất định trên cùng một header không cho thông tin độc lập, nên cận trên không có ý nghĩa thống kê.
Thay bằng: (a) mỗi mẫu dùng một header Besu thật KHÁC (lấy từ các block khác nhau trên chain), (b) vị trí
và giá trị đột biến chọn ngẫu nhiên với seed cố định (seed = 20261006), (c) số mẫu độc lập = n như mục 2.
Mục 2.1 và 2.2 dưới đây được hiểu theo cách này.

## 1. Việc chưa được phê duyệt (không chạy cho tới khi có đồng ý)
- Ghi thêm giao dịch lên chain demo (authchannel, mạng Besu 1337). Ước tính: khoảng 200 giao dịch
  cho các mục 2.1–2.3, cộng khoảng 60 giao dịch cho mục 2.4.
- Đổi chaincode (thêm hàm baseline ICS-27). Đây là deploy sequence mới, cần cả hai org approve.

## 2. Tăng n cho các cơ chế nhỏ (đánh giá: cận trên 95% một phía, Clopper–Pearson)
Mục tiêu: n đường thực đủ để cận trên dưới 5% khi lỗi = 0 (cần n ≥ 59).
- 2.1 Ab1 (trạng thái/header): hiện n=5. Mục tiêu n = 60 bằng cách sinh 60 biến thể malform
  (cùng 5 loại, lặp với tham số khác nhau). Tiêu chí: real path từ chối 60/60; ablated path chấp nhận ≥1.
- 2.2 Ab5 (finality): hiện n=5. Mục tiêu n = 60 (quét số seal 0–4, mỗi mức lặp 12 lần).
- 2.3 Ab4 (tươi mới): hiện n=1, không thể tăng vì Besu cắt lịch sử trạng thái. KHÔNG tăng n;
  bài ghi rõ n=1 và không công bố cận cho mục này.
- 2.4 S18 (đa chặng): hiện n=1 on-chain. Mục tiêu n = 30 chuỗi đa chặng (h ∈ {2,3,4,5}, mỗi h có
  mẫu bảo toàn và mẫu mất thông tin). Tiêu chí: composed từ chối 100% mẫu mất thông tin; pairwise
  chấp nhận ≥1 mẫu.

Quy tắc dừng: không thêm mẫu sau khi thấy kết quả; báo cáo mọi lần chạy, kể cả lỗi hạ tầng.

## 3. Đối chứng ICS-27 (type-level allowlist)
Đối chứng: cùng các kịch bản widening theo giá trị (S2–S4 trên corpus hình thức và các ca on-chain
đã có), nhưng quyết định chỉ dựa trên danh sách loại hàm được phép (không đọc số tiền).
- Giả thuyết H0: đối chứng ICS-27 chặn được mọi trường hợp gọi sai hàm, nhưng chấp nhận mọi
  widening theo giá trị trong hàm được phép.
- Tiêu chí: tỉ lệ chấp nhận sai cho widening theo giá trị phải ≥ 50%. Nếu thấp hơn, luận điểm
  "type-level không đủ" phải được hạ bậc.
- Đo trên cùng bộ kịch bản với ADP; không thêm ca sau khi có kết quả.

## 4. Những gì sẽ không làm
- Không tăng n cho Ab2, Ab3, Ab6, Ab8, Ab9, Ab10, Ab11 (đã đủ hoặc đã có bằng chứng khác).
- Không đổi định nghĩa FAR, FRR hay cận đã báo cáo.
- Không dùng bộ held-out RAG để chỉnh tiêu chí.

## 5. Kết quả mong đợi và rủi ro
- Nếu 2.1–2.2 đạt, cận trên của hai cơ chế đó giảm xuống dưới 5% (hiện 45%).
- Nếu S18 mẫu mất thông tin có ca lọt qua, bài phải báo cáo và sửa luận điểm multi-hop.
- Rủi ro hạ tầng: 2.1 và 2.2 chạy trên chain tải nặng; kiểm tra peer count trước mỗi đợt.

Trạng thái: ĐÃ ĐƯỢC ĐỒNG Ý (mục 1 và 3). Thực hiện theo mục 0.
