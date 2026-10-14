# Nhờ thầy đóng vai trò "domain thứ hai" độc lập thật — kiểm tra nghĩa vụ (obligation coverage)

## Bối cảnh ngắn (vì sao cần việc này)

Hệ thống ADP có một thực nghiệm tên $q_D$ (domain diversity): kiểm tra việc "giải trừ nghĩa vụ"
(obligation discharge) của một giao dịch cross-chain được xác nhận bởi **hai domain quản trị
độc lập**, không chỉ một. Thực nghiệm trước (dùng AWS) đã bị review đúng chỉ ra: AWS và máy chủ
chính đều do cùng một người (em) vận hành — nên dù là 2 máy khác nhau, đó chỉ là "đa dạng hạ
tầng", không phải "đa dạng quản trị" thật.

Để có kết quả thật, cần một người **hoàn toàn độc lập** — không phải tác giả, tự giữ khóa, tự
đưa ra phán đoán riêng — xác nhận lại đúng một câu hỏi hẹp: *với dữ liệu cho sẵn, nghĩa vụ
(obligation) có được đáp ứng đầy đủ hay không?* Việc này tách biệt hoàn toàn khỏi các bước khác
của hệ thống (bằng chứng ZK, xác thực header Besu, ngưỡng vai trò) — những bước đó chain chính
đã tự kiểm tra, không thuộc phạm vi này.

**Thời gian ước tính: 30-60 phút.** Không cần cài đặt phức tạp, không cần mở cổng mạng, không
cần truy cập vào hệ thống của em.

## Thầy cần làm gì (4 bước)

### Bước 0 — Chuẩn bị (một lần, trên Windows)
1. Cài **Go** cho Windows: vào https://go.dev/dl/, tải file `.msi` (ví dụ
   `go1.2x.x.windows-amd64.msi`), chạy cài đặt như một phần mềm bình thường (Next/Next/Finish).
   Sau khi cài xong, **mở lại** PowerShell (đóng cửa sổ cũ, mở cửa sổ mới) để nó nhận lệnh `go`.
2. Giải nén thư mục này ra (chuột phải → "Extract All..." như mọi file zip khác trên Windows).
3. Mở **PowerShell** (bấm Start, gõ "PowerShell", Enter), dùng lệnh `cd` để vào đúng thư mục
   `signer` vừa giải nén, ví dụ:
   ```
   cd C:\Users\<ten-may-cua-thay>\Downloads\qd_advisor_independent_2026-10-12\signer
   ```
4. Chạy:
   ```
   go mod tidy
   ```
   (chỉ cần internet một lần để Go tự tải 2 thư viện mật mã cần dùng.)

Mọi lệnh ở các bước dưới đều chạy trong đúng cửa sổ PowerShell này, y hệt cách gõ trên máy
Linux/Mac — không cần cài WSL hay Linux gì cả, Go chạy thẳng trên Windows.

### Bước 1 — Tạo khóa riêng của thầy
```
go run main.go keygen
```
Lệnh này tạo một khóa mật mã (secp256k1) **lưu trên máy của thầy**, không gửi đi đâu cả. Nó in
ra một "địa chỉ công khai" (dạng `0x...`). **Gửi lại đúng dòng địa chỉ đó cho em** (không gửi
file `advisor_domain_key.hex` — đó là khóa riêng, phải giữ lại trên máy thầy).

### Bước 2 — Đọc đặc tả quy tắc
Đọc file `dac_ta_quy_tac_nghia_vu.md` — mô tả đúng 4 quy tắc xác định một nghĩa vụ có được đáp
ứng đầy đủ hay không. **Đây là đặc tả bằng lời, không phải code** — xin thầy tự áp dụng các quy
tắc này bằng chính phán đoán của mình, không chạy lại bất kỳ chương trình nào của em để "tính ra"
kết quả (nếu làm vậy thì phán đoán không còn độc lập nữa, mất hết giá trị của thực nghiệm này).

### Bước 3 — Xét 6 ca trong `cases.json`
Mỗi ca là một tình huống cụ thể (file `cases.json`). Với mỗi ca, áp dụng đặc tả ở Bước 2, tự kết
luận: nghĩa vụ có được đáp ứng đầy đủ (`covered`) hay không — chỉ `true` hoặc `false`.

### Bước 4 — Ký và gửi lại
Với mỗi ca, chạy đúng 1 dòng lệnh này (thay đúng giá trị từ `cases.json` của ca đó, và
`--passed` là kết luận của thầy ở Bước 3) — gõ liền một dòng, không xuống dòng giữa câu lệnh:
```
go run main.go sign --decision-id=<decisionId> --discharge-ref=<obligationDischargeRef> --source-domain=<sourceDomain> --role=<role> --amount=<amount> --passed=true|false
```
Ví dụ cụ thể cho ca `qd-adv-01` nếu thầy kết luận `covered = true`:
```
go run main.go sign --decision-id=qd-adv-case-01 --discharge-ref=discharge-rec-01 --source-domain=BankA --role=manager --amount=300000 --passed=true
```
Lệnh in ra một chữ ký (chuỗi hex). **Gửi lại cho em đúng 3 thứ cho mỗi ca**: `decisionId`, kết
luận `passed` (true/false) của thầy, và chữ ký in ra. Có thể gửi dạng bảng/liệt kê đơn giản.

## Vì sao cần làm đúng cách này (xin đọc trước khi bắt đầu)

- **Xin đừng hỏi em kết quả "đúng" trước khi làm** — mục đích của thực nghiệm là xem phán đoán
  độc lập của thầy (chỉ dựa vào đặc tả) có khớp với logic thật của hệ thống hay không. Nếu thầy
  biết trước đáp án, kết quả sẽ không còn giá trị để báo cáo.
- Một trong 6 ca có một "cái bẫy" có chủ đích (không nói trước ca nào) — để kiểm tra liệu chỉ đọc
  một cờ boolean đơn lẻ có đủ hay cần đọc kỹ toàn bộ dữ liệu liên quan. Đây là thiết kế thực
  nghiệm bình thường, không phải đánh đố.
- Nếu thầy thấy đặc tả mơ hồ ở điểm nào, **đó cũng là một phát hiện đáng ghi lại** — xin ghi chú
  lại chỗ mơ hồ đó khi gửi kết quả, không cần tự đoán ý em.

## Sau khi thầy gửi lại

Em sẽ: (1) đối chiếu phán đoán của thầy với logic thật đang chạy trên chain để xem có khớp
không — khớp hay lệch đều là kết quả thật đáng công bố, không giấu; (2) đưa địa chỉ công khai
của thầy vào chaincode như một "domain authority" thứ hai; (3) chạy các giao dịch thật trên
chain với chữ ký của thầy, ghi lại làm bằng chứng cho bài báo (có disclose rõ vai trò của thầy,
không giấu danh tính nếu thầy đồng ý được ghi tên/vai trò trong phần cảm ơn hoặc phương pháp).
