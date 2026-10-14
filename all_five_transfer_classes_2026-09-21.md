# Lấp khoảng trống EXACT/CONDITIONAL — mạng authcc — 2026-09-21

## Phát hiện (từ đọc kỹ docx blueprint + đối chiếu code thật)

So sánh `Authorization_Decision_Portability_Manuscript_v2.docx` (bản blueprint kế hoạch, có các
"STAGE UPDATE" ghi tiến độ đo thật) với `Manuscriptsp2_Full_v1.tex` + chaincode thật, phát hiện:

Bài định nghĩa hình thức 5 lớp phân loại: `EXACT, SAFE, CONDITIONAL, REEVALUATE, REJECT`. Đọc
trực tiếp `classify()` (hàm thật, dùng cho toàn bộ 24-case corpus + mọi thí nghiệm trước đó):
```go
if !semBound: REJECT
if !evAssuranceOK: REJECT
if !roleKnown: REEVALUATE
if amount <= threshold: SAFE
else: REJECT
```
→ **Chỉ 3/5 lớp từng được trả về thật** (`REJECT`, `REEVALUATE`, `SAFE`). `EXACT` và
`CONDITIONAL` chỉ xuất hiện trong phần lý thuyết/định nghĩa (nhiều chỗ), **không một lần nào**
trong số liệu đo — kể cả 24-case corpus, RBAC/ABAC, concurrency, TOCTOU.

Nguyên nhân gốc: `classify()` không phân biệt (a) mapping identity vs không-identity (điều kiện
cần để có EXACT riêng biệt với SAFE), và (b) không có obligation thật nào gắn với quyết định
(luôn $O_s=\emptyset$) nên không có đường nào dẫn tới CONDITIONAL.

## Đã sửa: thêm `SubmitDecisionFull` + `DischargeObligation`

Chaincode mới (deploy v1.3/sequence 4), không sửa các hàm cũ:
- `RoleMapping` map mới: `gerente→manager`, `gerente_senior→director` (tên role nước ngoài,
  khác hẳn tên role đích) — tạo mapping KHÔNG-identity thật để phân biệt với mapping identity.
- `SubmitDecisionFull`: nếu `role` đã trùng tên đích (identity) → ứng viên EXACT; nếu qua
  `RoleMapping` → ứng viên SAFE. Nếu `amount > threshold/2` → cần "second approval" (obligation
  thật) → CONDITIONAL nếu chưa có, SAFE/EXACT nếu có/đã discharge.
- `DischargeObligation`: giao dịch thật thứ 2, chuyển 1 quyết định CONDITIONAL đã lưu trên ledger
  thành EXACT hoặc SAFE (tùy loại mapping), đọc lại được qua `ReadDecision` y như các bản ghi
  khác — không phải nhãn tính xong rồi bỏ.

## Kết quả 7 giao dịch thật (tất cả commit VALID)

| # | Role (mapping) | Amount | Obligation | Kết quả |
|---|---|---|---|---|
| 1 | manager (identity) | $200,000 | — | **EXACT** |
| 2 | gerente→manager (non-identity) | $200,000 | — | **SAFE** |
| 3 | manager (identity) | $450,000 | chưa discharge | **CONDITIONAL** |
| 6 | (discharge #3) | | đã discharge | **EXACT** |
| 3b | gerente→manager | $450,000 | chưa discharge | **CONDITIONAL** |
| 7 | (discharge #3b) | | đã discharge | **SAFE** |
| 4 | auditor (unknown) | $50,000 | — | **REEVALUATE** |
| 5 | manager, semBound=false | $200,000 | — | **REJECT** |

**Cả 5 lớp phân loại đều có bằng chứng thật trên chain**, kể cả chu trình chuyển đổi
CONDITIONAL→EXACT/SAFE qua giao dịch discharge thật thứ 2.

## Đã viết vào bài
`submit_paper2/Manuscriptsp2_Full_v1.tex` — thêm `\subsection{All five transfer classes,
exercised as real transactions}` (label `subsec:eval-all-five-classes`) + bảng
`tab:all-five-classes`, ngay sau phần định nghĩa Baselines, trước phần corpus 24-case.

Build PDF sạch, **32 trang** (từ 31).

## Việc chưa làm (không nằm trong phạm vi yêu cầu lần này)
- Chưa cập nhật lại các con số "N giao dịch tuần tự" rải rác trong bài (ví dụ dòng ~1101 nói "43
  sequential real transactions") để cộng thêm các giao dịch mới của phiên này (RBAC/ABAC,
  concurrency, TOCTOU, 7 giao dịch full-classifier) — cần 1 lượt rà soát tổng số giao dịch toàn
  bài nếu muốn con số tuyệt đối chính xác trước khi nộp.
- Obligation model vẫn đơn giản (chỉ 1 loại: second-approval theo ngưỡng cố định); Theorem T3
  giờ có ít nhất 1 ca $O_s\neq\emptyset$ thật, nhưng chưa phải obligation calculus đầy đủ.
