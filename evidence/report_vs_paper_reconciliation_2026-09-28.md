# Đối chiếu: Báo cáo E1–E10 (26/09) ↔ Bản thảo (`Manuscriptsp_m2_r3_usenix.tex`, 28/09) ↔ Evidence thô

Cách làm: mỗi con số của báo cáo được (a) tìm trong bản thảo bằng regex, (b) với E1, E4, E5, E6 (chiều ngược), E7, E8, E9, E10 mở file evidence JSON thô để đối chiếu.
Không đối chiếu thô: bảng phân rã độ trễ 4.1 (chỉ khớp báo cáo↔bài; file `e9perf_component_timed_result.json` chưa mở), Bảng 6b (N=10/20/50), số chặng h=2 block 0x606cb.

## 1. Kết quả theo nhóm

| Nhóm | Báo cáo | Bài | Evidence thô | Kết luận |
|---|---|---|---|---|
| E1 | n=2000; 1850 unsafe/150 safe; FAR 0 (≤0,16%), FRR 0 (≤1,98%); 24/50 sound; 4 lỗi cấy, phát hiện 4/4 | khớp | `e1_formal_benchmark_results`: 1850/150, 24 sound/26 unsound, 2000/2000 đúng | KHỚP. Lệch về diễn giải (mục 3.1) |
| E2 | 8 tx thật 5 lớp; 5 unit test 5/5 | khớp | (không mở lại) | KHỚP bài↔báo cáo |
| E3 | 20 kịch bản; 0/880, 0/501, 0/689, 0/153+1/1, 0/293, 0/385, 0/618, S19 0/60, S18 0/1; chiều ngược 31 tx | 0/675, 0/501, 0/689, 0/153+0/1000, 0/293, 0/385, 0/618, S19 0/50, S18 0/1 | crosswalk cũ: S2=400, S3=480, S4=400 (mẫu con thô); S19: 10 đối chứng dương + 50 thay thế | 3 SỐ LỆCH, dùng số của bài (mục 2) |
| E4 | N=1000: 991 từ chối + 9 MVCC = 1000/1000; cận 0,30% | khớp | `e4_replay_depth_1000`: 991 rõ ràng; `e4_block333_validation`: tx0 VALID, 9 tx MVCC_READ_CONFLICT | KHỚP |
| E5 | n=31, 0 nâng cấp sai, cận 9,21%; precompile 24/24, gas 34.320 | khớp | `e5_dual_implementation_v4`: 24 ca, 0 bất đồng; `e5_precompile_v5`: 24/24 | KHỚP |
| E6 | ABAC 6/6 hai chiều; chiều ngược 22/22 (≤12,73%); quét N=20/50/100: 300/300, trung vị 4.673–4.722 ms | khớp (ABAC mô tả ở phụ lục, không có chuỗi "6/6") | `e6_reverse_scaling_tiers`: N20 20/20, trung vị 4722, 0,845 tx/s | KHỚP |
| E7 | 190 ca; 126 unsafe/59 safe/5 reevaluate; B0 100%, B2 56,3% (71/126), B3 0,8% (1/126), B4 0%; cận B4 2,35%/4,95% | khớp | `e7_big_push/b0b4_recompute`: 71, 1; `..._ci_final`: 0,0235, 0,0495 | KHỚP. B1 = 0% là "by construction" (mục 3.2) |
| E8 | Ab0–Ab9: 0; 0,70; 3,95; 2,49; 3,62; 1,30; 4,59; 3,19; 0,70; 0,92 | có thêm Ab10, Ab11 | `e8_ablation_result`: A1 13/1850 = 0,703%, A2 73 = 3,946%, A3 46, A4 67, A5 24 | KHỚP (Ab0–Ab5 đã đối chiếu thô; Ab6–Ab9 khớp bài↔báo cáo) |
| E9 | 25 ca mất thông tin, 0/25 (≤11,29%); pairwise nhận sai 25/25; 56 tx thật; h=5 6/6 | khớp | `e9_multihop_extension`: 31 ca (6 đối chứng + 25), nUnsafeComposed 0, nUnsafePairwise 25 | KHỚP |
| E10 | 100→10.000: 18.800 tx, 55,4–57,9 ms/tx; N=10^5 không thử | khớp | `e10_concurrency_full_sweep`: 100+200+500+1000+2000+5000+10000 = 18.800, đều commit đủ | KHỚP |

## 2. Ba số báo cáo cần sửa theo bài
1. **S2–S4: 0/880 → 0/675.** 675 là số đã khử trùng (200 + 317 − 42 giao + 200, tính lại từ generator seed cố định). 880 không có căn cứ trong bài; nguồn có thể là cộng thô các mẫu con (crosswalk cũ ghi 400/480/400).
2. **S19: 0/60 → 0/50.** 60 = 10 đối chứng dương (đúng là `valid=true`) + 50 lần thay thế. Thống kê đúng là 0/50, cận 5,82%. Chính báo cáo ở đoạn văn cũng dùng 50.
3. **TOCTOU/phát lại: "0/153 (mô phỏng) + 1/1 (thật)" → "0/153 (mô phỏng) + 0/1000 (phát lại thật); Ab9 TOCTOU 8/8 đường thật đúng và 8/8 đường bị cắt sai".**

## 3. Chỗ báo cáo nói mạnh hơn bài (nên hạ trước khi báo cáo)
1. **E1 "nhãn độc lập từ Z3, không phải kiểm tra hình thức".** Bài nói E1 là "regression test": nhãn và bộ phân loại cùng đánh giá một đặc tả, nên FAR=0 chưa phải bằng chứng độc lập. Độc lập đến từ kiểm thử đột biến (4/4) và kiểm tra chéo (snarkjs, Liboqs/CIRCL).
2. **B1 = 0% ở bảng E7.** Đây là 0% theo cấu tạo (cùng công thức sinh nhãn), không phải phát hiện thực nghiệm. Bài ghi rõ; báo cáo không.
3. **"Tất cả 10 nhóm đạt mức xuất sắc".** Đây là tiêu chí nội bộ; bài không dùng chữ này. Khi báo cáo nên nói "đã có đợt kiểm chứng độc lập cho từng nhóm".
4. **"Không có chấp nhận không an toàn" (mục 1, 9).** Chỉ đúng trong các kịch bản đã thử, với giả định A1–A13. Xem mục 5.

## 4. Chỗ báo cáo đã cũ so với bài (bài mới hơn, phiên 28/09)
| Nội dung | Báo cáo (26/09) | Bài / evidence hiện tại |
|---|---|---|
| Chaincode | v2.26/seq38 | v2.29/seq41 (thêm file cộng thêm; `authchaincode.go` không đổi) |
| RQ4/H7 | "Tái sử dụng rẻ hơn 43–49% so với B1 — đây là câu trả lời thật" | Đúng chỉ so với chạy lại cả pipeline. Cùng runtime, cùng quan hệ: verify 0,85 ms nhanh hơn Poseidon (1,6 ms) nhưng chậm hơn SHA-256 (0,03 ms); trên cả classifier head 0,74 vs 0,21 ms. Lợi ích là không lộ đặc trưng x (866 byte), không phải độ trễ |
| Thời gian sinh proof | "chưa đo được (thiếu RAM)" | Đã đo: khoảng 8,0 s (7,9–8,1), khoảng 5 GB, khóa proving 199 MB (mạch last-layer); cả head 8–15 s và 5–11 GB |
| Lộ dữ liệu | không có | Probe tuyến tính 643 hợp đồng, AUC 0,74–0,94; sau khi kiểm soát nguồn gốc chỉ còn 3 thuộc tính có ích lợi dương có ý nghĩa |
| Freshness đa chặng (t₀) | bài trước ghi là khoảng trống | Đã hiện thực trên Fabric thật: 42 bản ghi path, 184 tx, 0 lỗi; path quá hạn: composed từ chối 6/6, per-hop chấp nhận 6/6 (cận trên 39,3% cho 0/6); Δ=120 s, khoảng cách 50 s là dựng sẵn |
| Ablation | Ab0–Ab9 | Ab0–Ab11 |
| Đo B1 công bằng | 22 file Python | thêm B1 cùng runtime Go, trên Fabric+Besu thật (Poseidon 2,81 ms vs B4 2,02 ms; SHA-256 1,04 ms) |

## 5. Điều chưa có trong bài lẫn báo cáo, nên biết trước khi báo cáo
- **Số tiền không dương được chấp nhận.** Kiểm tra độc lập (28/09) cho thấy `SubmitDecisionFull` trả EXACT/SAFE cho amount ≤ 0 (26/26 ca thử). Ngoài đặc tả (chỉ xác định cho số dương), chưa vá, và phần này đã bị gỡ khỏi bài cùng S3.
- **Audit mã nguồn (28/09):** 13 điểm nghi vấn, chưa có điểm nào xác nhận bằng chạy thật; đáng chú ý là `SetRoleThreshold`/`DischargeObligation` không có kiểm tra danh tính người gọi và các entry point ablation nằm trong contract production. Chi tiết ở `~/security-audit-skill/authcc/run-1/REPORT.md`.
- Hai điểm trên ảnh hưởng cách phát biểu "0 unsafe accept": nên nói rõ "trong các kịch bản và giả định đã nêu".
- Survey S1 và S3 (Cedar/adversary) đã bị gỡ khỏi bài theo quyết định của tác giả; evidence vẫn nằm trong `evidence/`.

## 6. Lỗi nhỏ trong báo cáo
- Hai bảng cùng đánh số "Bảng 8" (B0–B4 ở mục 4.3 và ablation ở mục 5); Bảng 9 cũng bị trùng.
- Mục 8: "10 cảnh báo overfull"; hiện là 11 (bản USENIX).
