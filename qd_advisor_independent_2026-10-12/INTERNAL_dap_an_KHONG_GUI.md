# NỘI BỘ — KHÔNG GỬI FILE NÀY CHO THẦY

Đáp án tham khảo cho `cases.json`, suy ra trực tiếp từ logic thật đang chạy (`oblCoveredV2` trong
`obligation_v2.go` + đoạn kiểm tra discharge trong `q_d_domain_attestation.go`), dùng để đối
chiếu với phán đoán độc lập của thầy sau khi thầy gửi lại — không dùng để "mách" trước.

| Ca | covered (đáp án) | Lý do |
|---|---|---|
| qd-adv-01 | **true** | audit(2) yêu cầu, audit(2) ánh xạ — đủ; discharge khớp đúng domain/role/amount, chưa bị trích dẫn trước; có witness |
| qd-adv-02 | **false** | dual_approval(3) yêu cầu, mappedMechanism rỗng — nghĩa vụ bị tước bỏ (S5) |
| qd-adv-03 | **false** | combined(4) yêu cầu, notification(1) ánh xạ — bị làm yếu (S6) |
| qd-adv-04 | **false** | mechanism đủ mạnh, discharge khớp đúng, nhưng enforcementWitnessVerified=false — giải trừ rửa (S7) |
| qd-adv-05 | **false** | **CA BẪY**: discharge record có `obligationDischarged=true` nhưng `amount=700000` ≠ amount của quyết định (300000) — confused-deputy, nếu chỉ đọc cờ `obligationDischarged` mà không so khớp 3 trường sẽ SAI thành true |
| qd-adv-06 | **false** | discharge khớp đúng mọi trường, có witness, NHƯNG `alreadyCitedByAnotherDecision=true` — một lần giải trừ chỉ dùng được 1 lần |

Lưu ý khi đối chiếu: 5/6 ca là `false`, chỉ 1/6 là `true` — không cân bằng, nhưng mục đích là
kiểm từng lý do fail riêng biệt (S5/S6/S7/confused-deputy/one-time-use), không phải đo tỉ lệ.
Nếu thầy trả lời khác đáp án ở đây, ĐÓ LÀ KẾT QUẢ THẬT (phát hiện đặc tả mơ hồ hoặc tôi viết sai
đặc tả) — không tự "sửa" câu trả lời của thầy để khớp bảng này, phải báo cáo trung thực cả hai.
