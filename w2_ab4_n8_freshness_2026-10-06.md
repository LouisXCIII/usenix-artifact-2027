# Ab4 (freshness): điều tra sâu và tăng từ n=1 lên n=8 (2026-10-06)

Trả lời yêu cầu "Ab4 vẫn còn n=1 (không tăng được vì giới hạn kỹ thuật thật của Besu). điều tra sâu,
tìm phương án cải thiện" sau tuyên bố trước đó (chưa kiểm chứng kỹ) rằng Ab4 không thể tăng n.

## Gốc rễ thật (đọc trực tiếp mã nguồn, không suy đoán)
- `maxHeaderAgeSeconds = 24 * 3600` (`lightclient.go`) — ngưỡng "cũ" để `checkFreshness` từ chối.
- Besu chỉ giữ trạng thái lịch sử trong một cửa sổ ~500-700 block (~17-23 phút), xác nhận lại bằng
  `eth_getProof` thật (block tip-100 OK, tip-700 lỗi "World state unavailable").
- Hai số này KHÔNG giao nhau: không có block nào vừa "còn provable bằng eth_getProof ngay bây giờ"
  vừa "đã quá 24 giờ tuổi". Vì vậy không thể tạo mẫu "cũ" mới trong ngày bằng cách gọi eth_getProof
  trên block lịch sử (cách Ab1/Ab5 dùng cho mẫu của chúng) — đây là trần kỹ thuật thật.

## Phương án tìm ra: tái sử dụng fixture (header, registryProof) thật đã fetch từ các phiên trước
`checkFreshness` so sánh timestamp NHÚNG SẴN trong header JSON với ledger clock hiện tại — không
cần gọi lại `eth_getProof` cho header đó. Một proof Merkle-Patricia đã fetch vẫn còn đúng cấu trúc
mãi mãi (xác minh lại trên stateRoot của chính header đó, không phụ thuộc trạng thái Besu hiện tại).
Đây đúng là cách bản gốc n=1 đã làm (`testdata/besu_header_0x5fc0c.json`, từ 2026-09-21). Tìm thêm
trên máy: có **8 cặp (header, registryProof) thật, riêng biệt** (8 block khác nhau: 392204, 394017,
401766, 401785, 413206, 413365, 415116, 422490), từ 7 file thí nghiệm khác nhau ngày 2026-09-21/22,
nay tự nhiên đã 14-21 ngày tuổi (>>24h).

## Kết quả (16 giao dịch thật, script `w2_ab4_n8_freshness.js`)
- **Đường thật** (`SubmitDecisionTrustless`): **8/8** từ chối đúng vì lý do cũ ("header is Xs old,
  exceeds the 86400s freshness bound", X từ 1.207.991s tới 1.833.804s ≈ 14-21 ngày).
- **Đường ablated** (`SubmitDecisionTrustlessNoFreshnessCheck`): **6/8** bị khai thác sạch
  (`lcVerified=true`, "header age NOT checked") — cùng cơ chế bản gốc n=1 đã minh họa.
- **2/8 còn lại** (x8_chain_result.json, x13_featurecommit_independent.json) không dùng được cho
  phía ablated: registryProof của chúng là proof đối với MỘT CONTRACT KHÁC (địa chỉ
  `0x5774D352...`/`0x7Cb4419...`, feature-commit registry của X8/X13), không phải ML-registry đã
  pin trong chaincode (`0x4BDF7d0CC77F907FeDfc1Ab6B39fa9f4336869c7`) — lỗi "proof is for a contract
  that is..." phản ánh đúng sự thật này, không phải lỗi tham số có thể sửa. Phía thật của cả 2 vẫn
  đúng (từ chối vì cũ), chỉ phía ablated bị nhiễu bởi vấn đề khác, không liên quan tới freshness.

## Kết luận trung thực
- n thật tăng từ 1 lên 8 cho phía "real path correctly rejects" (cận trên 95% Clopper–Pearson,
  lỗi=0/8: **31,23%** — vẫn rộng, nhưng là cận có ý nghĩa đầu tiên cho cơ chế này).
- n cho phía "ablated path exploited" là 6/8 sạch (2/8 bị nhiễu bởi registry-address không khớp,
  đã công bố rõ, không gộp vào số đo).
- Không thể vượt n=8 bằng dữ liệu hiện có trên máy — đây là số fixture thật duy nhất tìm được.
  Muốn tăng thêm, cần tích lũy theo thời gian (mỗi phiên làm việc mới tạo ra fixture mới, sau 24h
  sẽ dùng được), không có cách tạo ngay trong một ngày.

Raw: `experiments/portability/w2_ab4_n8_result.json`.
