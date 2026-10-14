# Đặc tả: khi nào một "nghĩa vụ" (obligation) được coi là đáp ứng đầy đủ

Đây là đặc tả bằng lời cho đúng một câu hỏi: với dữ liệu của một quyết định cross-chain, nghĩa
vụ gắn với quyết định đó có được đích (destination) đáp ứng đầy đủ (`covered = true`) hay không?

**Phạm vi**: chỉ xét 4 trường liên quan tới nghĩa vụ — `obligationProfile`, `mappedMechanism`,
`obligationDischargeRef` (và dữ liệu của bản ghi nó trỏ tới), `enforcementWitnessVerified`.
KHÔNG xét vai trò/số tiền/ngưỡng, KHÔNG xét bằng chứng ZK/header — những phần đó thuộc một kiểm
tra khác của chain chính, không phải phần thầy cần xác nhận.

## Quy tắc 1 — Mức độ yêu cầu của nghĩa vụ nguồn (`obligationProfile`)

Mỗi `obligationProfile` có một mức "độ mạnh yêu cầu" (bảng thứ tự, số lớn hơn = yêu cầu khắt
khe hơn):

| obligationProfile | độ mạnh yêu cầu |
|---|---|
| `"empty"` (không có nghĩa vụ gì) | 0 |
| `"notification"` | 1 |
| `"audit"` | 2 |
| `"dual_approval"` | 3 |
| `"combined"` | 4 |

Nếu độ mạnh yêu cầu = 0 → nghĩa vụ coi là đáp ứng ngay (`covered = true`), không cần xét gì
thêm. Nếu độ mạnh yêu cầu > 0, tiếp tục Quy tắc 2.

## Quy tắc 2 — Cơ chế đích (`mappedMechanism`) có đủ mạnh không

`mappedMechanism` cũng có độ mạnh riêng theo đúng bảng trên (`""` nghĩa là không ánh xạ sang cơ
chế nào cả, độ mạnh = 0).

- Nếu `mappedMechanism` là chuỗi rỗng (`""`) **và** độ mạnh yêu cầu (Quy tắc 1) > 0 → nghĩa vụ
  bị **tước bỏ hoàn toàn** → `covered = false`. (Dừng, không cần xét tiếp.)
- Nếu độ mạnh của `mappedMechanism` **nhỏ hơn** độ mạnh yêu cầu → nghĩa vụ bị **làm yếu** →
  `covered = false`. (Dừng, không cần xét tiếp.)
- Nếu độ mạnh của `mappedMechanism` **lớn hơn hoặc bằng** độ mạnh yêu cầu → đạt, tiếp tục Quy
  tắc 3.

## Quy tắc 3 — Bản ghi "giải trừ nghĩa vụ" (discharge) có thật và khớp đúng giao dịch này không

Mỗi ca có thể có một `obligationDischargeRef` (tham chiếu tới một bản ghi khác) và dữ liệu đầy
đủ của bản ghi đó (`referencedDischargeRecord`). Nghĩa vụ chỉ được coi là đã giải trừ hợp lệ
(`obligationDischarged = true`) khi **TẤT CẢ** các điều sau đúng — thiếu một điều cũng khiến
`covered = false`:

a. `referencedDischargeRecord.obligationDischarged` phải là `true` (bản ghi tự xác nhận đã giải
   trừ — nếu `false` hoặc bản ghi không tồn tại → chưa giải trừ → `covered = false`).

b. **Ba trường `sourceDomain`, `role`, `amount` của bản ghi được tham chiếu phải khớp CHÍNH XÁC**
   với `sourceDomain`, `role`, `amount` của chính quyết định đang xét (không phải của quyết định
   khác). Nếu bất kỳ trường nào trong 3 trường này khác nhau → đây là lỗi "confused-deputy": một
   lần giải trừ hợp lệ cho một giao dịch KHÔNG được dùng để hợp thức hóa một giao dịch khác có
   vai trò/số tiền khác, dù cờ `obligationDischarged` vẫn là `true` → `covered = false`.

c. Trường `alreadyCitedByAnotherDecision` của bản ghi được tham chiếu phải là `false`. Một lần
   giải trừ chỉ được dùng cho đúng MỘT quyết định — nếu bản ghi đã từng được một quyết định khác
   trích dẫn trước đó (`alreadyCitedByAnotherDecision = true`), quyết định hiện tại không được
   phép dùng lại nó → `covered = false`.

Nếu cả (a), (b), (c) đều đúng → nghĩa vụ coi là đã giải trừ hợp lệ cho chính giao dịch này, tiếp
tục Quy tắc 4.

## Quy tắc 4 — Có nhân chứng thực thi độc lập không (`enforcementWitnessVerified`)

Dù đã giải trừ hợp lệ theo Quy tắc 3, nếu `enforcementWitnessVerified = false` → việc giải trừ
được coi là "giải trừ rửa" (laundered): tự khai báo đã giải trừ nhưng không có bên thứ ba nào
xác minh độc lập việc thực thi đó → `covered = false`.

Chỉ khi `enforcementWitnessVerified = true` → `covered = true`.

## Tóm tắt quy trình (áp dụng tuần tự, dừng ngay khi có kết luận)

1. Độ mạnh yêu cầu = 0? → `covered = true`, dừng.
2. `mappedMechanism` rỗng (và yêu cầu > 0)? → `covered = false`, dừng.
3. Độ mạnh `mappedMechanism` < độ mạnh yêu cầu? → `covered = false`, dừng.
4. Bản ghi discharge không tồn tại / `obligationDischarged != true`? → `covered = false`, dừng.
5. `sourceDomain`/`role`/`amount` của bản ghi discharge không khớp quyết định đang xét?
   → `covered = false`, dừng.
6. Bản ghi discharge đã bị một quyết định khác trích dẫn trước đó? → `covered = false`, dừng.
7. `enforcementWitnessVerified = false`? → `covered = false`, dừng.
8. Nếu đi hết các bước trên mà không dừng ở đâu → `covered = true`.
