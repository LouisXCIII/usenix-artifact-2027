# Destination tự dẫn xuất cả SemBound và EvAssuranceOK (SubmitDecisionTrustless) — 2026-09-21

Đây là file tổng hợp cho việc đóng Limitations item 1 ("destination tin relay"). File
`real_destination_zk_verification_2026-09-21.md` là nửa 1 (Groth16); file này là nửa 2 (light client
Besu) và phép ghép hai nửa. Lưu ý: nửa 1 đã được **sửa** trong lượt này (xem "Sửa đổi so với nửa 1").

## Cơ chế (chaincode authcc v1.6, sequence 7)

`SubmitDecisionTrustless(decisionID, sourceDomain, role, amount, sourceLimit, codeHash, proof,
publicSignals, besuHeaderJSON, registryProofJSON)` — **không còn** tham số boolean `semBound` /
`evAssuranceOk`.

- `SemBound` = (a) Groth16 proof verify được với verification key nhúng, **và** (b) public signals đạt
  chính sách của destination (`maxVulnClass==0`, `minConfidence>=7000`), **và** (c) public
  `solidityCodeHash` == `codeHash` của chính quyết định này (không thay proof của hợp đồng khác được),
  **và** (d) `mlScanPassed[codeHash]` của RegistryContract, được chứng minh từ state Besu, là true.
- `EvAssuranceOK` = header Besu được verify với validator set đã ghim **và**
  `approvalCount[codeHash] >= 2` được chứng minh từ state.
- Light client (Go, `chaincode/lightclient.go`): dựng lại block hash từ các trường header thô, dựng lại
  committed-seal hash theo `BftBlockHashing.calculateDataHashForCommittedSeal` của Besu, `ecrecover`
  từng commit seal, yêu cầu ≥ ceil(2n/3)=3 validator phân biệt thuộc tập genesis đã ghim; sau đó verify
  Merkle-Patricia proof (`eth_getProof`) của tài khoản registry và của 2 storage slot dưới `stateRoot`
  của header. **Storage key do chaincode tự tính** từ `codeHash` (không nhận từ request).
- Thư viện: `gnark-crypto` (Groth16/BN254), `decred/secp256k1` (ecrecover), `x/crypto/sha3`; RLP và MPT
  tự viết (~200 dòng), không kéo go-ethereum vào chaincode.
- Fail-closed: bằng chứng không verify được → ghi bản ghi REJECT kèm lý do (không abort giao dịch), trừ
  request sai định dạng (lỗi).

## Xác minh (thật, không giả lập)

1. **Prototype độc lập** trên dữ liệu thật của mạng Besu chainId 1337 (đọc qua SSH tunnel sau khi bật lại
   10 VM OpenStack): header block 0x5fc0c dựng lại đúng hash; 3/4 commit seal phục hồi ra 3 validator
   khác nhau trong tập genesis; slot `approvalCount`/`mlScanPassed` chứng minh đúng (HONEST: 2/true;
   DISHONEST: 2/false — `false` được chứng minh từ trie). 4 ca âm bị từ chối (đổi timestamp, giả
   stateRoot, sửa 1 node proof, sai tập validator).
2. **13 Go test** trên code chaincode thật với fixture thật (`chaincode/testdata/`): 13/13 PASS, gồm cả
   4 test Groth16 của nửa 1 (chạy lại sau khi `go get` hạ gnark-crypto v0.17.0→v0.14.0 để khớp x/crypto).
3. **Deploy thật** lên mạng Fabric `authcc` đang chạy: v1.6 / sequence 7, approve bởi cả 2 org.
4. **Chạy end-to-end thật** (`relay_trustless.py`, run "r2"; run "r1" chạy trước đó có cùng kết quả phân
   loại nhưng harness đọc lại sớm 1 s trước khi block commit nên không in được trường chi tiết):

| Ca | Bằng chứng | Kết quả | Vì sao |
|---|---|---|---|
| T1 honest | proof SafeBank thật + header thật + state proof thật | **SAFE** | semBound=true, evOK=true (3/4 seal, approvals=2 chứng minh) |
| T2 relay đổi proof+state của honest sang quyết định dishonest | proof honest cho codeHash dishonest | **REJECT** | proof bị ràng codeHash khác; không có storage proof cho slot 5 của codeHash này |
| T3 dishonest, state thật của chính nó | header+state thật, không có proof hợp lệ cho codeHash này | **REJECT** | evOK=true (2 approval) nhưng semBound=false — đúng kịch bản Theorem T0 |
| T4 relay giả header (stateRoot) | header sửa | **REJECT** | block hash dựng lại ≠ hash khai báo |
| T5 relay sửa proof | pi_a sửa 1 chữ số | **REJECT** | pairing check thất bại |
| Đối chiếu: `SubmitDecision` cũ, relay nói dối `semBound=true`, `evAssuranceOk=true` cho hợp đồng dishonest | — | **SAFE** | chứng minh lỗ hổng cũ là thật, không chỉ lý thuyết |

## Sửa đổi so với nửa 1 (nói thẳng)

- Nửa 1 kiểm chứng proof nhưng **không ràng proof với quyết định**: proof_A_v5 có
  `solidityCodeHash=0x037cf1…`, KHÔNG phải codeHash HONEST của registry (`0x1f6621…` = SafeBank.sol). Một relay
  có thể ghép mọi proof hợp lệ với mọi mục registry. Đã sửa bằng điều kiện (c) ở trên, và bằng cách sinh
  **proof v5 thật cho SafeBank** (pipeline `cli_scan_zkp_v5.py`, cùng zkey/vkey đã nhúng — đã cmp
  giống hệt; public `solidityCodeHash` khớp đúng `0x1f6621…`).
- Ngưỡng `minConfidence`: lần chạy đầu với 8500 (mặc định) thất bại ở witness generation vì classifier
  thật chỉ cho SafeBank độ tin cậy **0.7739**; bản nháp 8000 cũng sẽ loại nó. Đặt 7000 **sau khi thấy số
  này** — đây là giá trị chính sách minh họa, không phải tham số bảo mật độc lập. Đã ghi vào comment mã.

## Giới hạn còn lại (không được nói là đã đóng)

1. **Feature forgery / P3**: circuit v5 ràng argmax(W·x+b) và hash trọng số lớp cuối, nhưng **không ràng
   vector đặc trưng x với mã nguồn có `solidityCodeHash` đó**. Prover độc hại có thể chứng minh "class 0"
   cho codeHash của hợp đồng có lỗ hổng bằng cách đưa x của hợp đồng sạch. Destination giờ xác minh
   "có người chạy đúng lớp cuối của mô hình đã cam kết cho ra class 0 và gắn nhãn codeHash này", chưa phải
   "hợp đồng này sạch". (Review mục 2.4 nói đúng; CLI gốc cũng tự ghi "vẫn KHÔNG đạt P3 đầy đủ".)
2. `RegistryContract.recordMLScanPass` không có kiểm soát truy cập — bit `mlScanPassed` chứng minh được từ
   state nhưng bản thân nó có thể do bất kỳ ai ghi. Vì vậy SemBound không dựa riêng vào bit đó mà đòi thêm
   proof hợp lệ đã ràng codeHash; nhưng ràng buộc này chỉ mạnh đến mức giới hạn 1.
3. **Trust anchor tĩnh**: validator set (4 địa chỉ genesis) và địa chỉ registry được ghim cứng trong
   chaincode; chỉ verify header của MỘT block, không theo dõi thay đổi validator (header có tập validator
   khác bị từ chối). `EvAssuranceOK` = "2/3 admin đã approve trên Besu" — vẫn **không** phải mức assurance
   NIST hay ML-DSA (review mục 2.6 đúng).
4. Mạng Besu nguồn **đã dừng** (block cuối 2026-09-15, `net_peerCount=0`: mesh `pqc_tunnel` chưa bật lại sau
   reboot). Header dùng là block cuối cùng thật; ràng buộc độ tươi 30 ngày được chọn để chấp nhận nó —
   kiểm thử độ tươi chỉ bằng unit test (header quá hạn bị từ chối), chưa có ca on-chain.
5. Phạm vi thực nghiệm: 1 hợp đồng honest + 1 dishonest, 5 ca; **chưa** chạy lại 24-case corpus /
   7 ca dishonest qua đường trustless. Chi phí chỉ được chặn thô (n=10 mỗi loại, qua peer CLI): simulate 34 ms (trustless) vs 33 ms (boolean); invoke→commit (`--waitForEvent`) 2047 ms vs 2043 ms — cả hai bị chi phối bởi BatchTimeout 2 s của orderer, nên chỉ kết luận chi phí verify thêm ở mức vài ms, không tách riêng được. (Lần đo đầu 38 ms là sai: `peer chaincode invoke` không có `--waitForEvent` trả về trước khi commit.)
6. Relay độc hại được thử ở 3 dạng (thay proof, giả header, sửa proof); chưa thử replay chéo đích, hop độc
   hại, relay chặn/trì hoãn (chỉ chứng minh được an toàn, không chứng minh liveness).

## Hạ tầng (ghi chú vận hành)

Máy vừa reboot lúc 16:54 ngày 21/9: 10 VM OpenStack SHUTOFF (đã bật lại), SSH tunnel 18545 dựng lại,
container Fabric `authcc-*` dừng (đã `docker start`, ledger còn nguyên, height 55→). Runbook
`REBOOT_RECOVERY.md` mục Fabric đã lỗi thời (mạng thật nằm ở `~/authcc-network`, cổng 17050/17051/19051).

## Chạy toàn bộ 24-case corpus qua đường trustless (chaincode v1.7, seq 8) — cùng ngày

Kết quả thô: `trustless_corpus_c1_results_2026-09-21.json`. Kịch bản: `pqc-hardhat/experiments/portability/stage6_register_corpus_besu.js`
(đăng ký) + driver scratch `stage6_trustless_corpus.py`.

- Sau reboot: bật lại mesh `pqc_tunnel` theo runbook Bước 4 (4 VM × 1 server + 3 client) → Besu nguồn sinh block lại (~2 s/block).
- **Registry mới** `0x4BDF7d0CC77F907FeDfc1Ab6B39fa9f4336869c7` (registry cũ đã có 2 approval cho SafeBank/VulnerableBank, trong khi corpus cố ý
  đặt evAssuranceOk=false cho corpus-000/001; approval không gỡ được). 63 giao dịch Besu thật: 1 deploy + 16 `recordMLScanPass` (g1 PASS)
  + 46 `approve` (22 case × 2, 2 case × 1). Chaincode ghim cả 2 registry; ràng buộc độ tươi hạ từ 30 ngày xuống **24 giờ**.
- Với mỗi hợp đồng: chạy scan + proof v5 thật (chính sách đích: class 0, confidence ≥ 0.70). Verdict g1 đã ghi khớp 24/24 với scan sống.
  Có proof cho chính codeHash của nó: **13/24**; không có: **11/24** (8 g1 BLOCK, 3 g1 PASS nhưng không chứng minh được:
  `05_weak_access_control_safe` conf 0.6348, `10_gas_exception_vuln` conf 0.5333, `11_external_bug_safe` scan ra class Timestamp).
- Relay trung thực đưa proof của chính hợp đồng nếu có; nếu không, ca tấn công duy nhất khả dĩ là đưa proof SafeBank (thay thế).
- **Kết quả: 24 giao dịch thật, khớp ground truth B1 22/24.** 2 ca lệch đều là *từ chối nhầm* (B1=SAFE, trustless=REJECT):
  `05_weak_access_control_safe`, `11_external_bug_safe`. Ca thứ 3 cùng loại (`10_gas_exception_vuln`) trùng lớp vì role clerk vượt ngưỡng.
  Đây là chênh lệch giữa luật PASS của gateway và chính sách chứng minh của đích — KHÔNG chỉnh lại chính sách để xoá nó.
- Đường trustless không sửa được lỗi của classifier: g1 cho PASS 4 hợp đồng thực sự có lỗ hổng (`03_dos_vuln`, `08_timestamp_vuln`,
  `09_mishandled_exception_vuln`, `10_gas_exception_vuln`); đích chấp nhận SAFE 2 trong số đó (03, 09), 2 ca kia chỉ bị từ chối do ngưỡng role.
  Ground truth của corpus lấy verdict g1 làm SemBound nên B1 cũng làm vậy — đây là đúng điểm review 2.2 ("nhãn theo một oracle đã biết là sai").
- **Ca hết hạn on-chain**: header 15/9 bị từ chối, "header is 560784s old, exceeds the 86400s freshness bound" — thay thế việc chỉ có unit test.
- Vẫn chưa làm: 7-case dishonest-provider (B_semless), replay chéo đích, hop độc hại; chi phí chỉ chặn thô.

## X1 — tấn công đầu vào giả mạo lên relation v5 (cùng ngày; kết quả đảo ngược một số nhận định trước)

Mã: `pqc-hardhat/experiments/portability/x1_attacks.py`, `x1_register_attack.js`; driver scratch `x1_trustless.py`.
Kết quả thô: `x1_prover_attacks_summary_*.json`, `x1_trustless_results_*.json`, `x1_registry_result_*.json`.
Mọi tấn công đi qua ĐÚNG pipeline prover thật (`MLScanProofV5.generate`, circuit/zkey/snarkjs thật) — chỉ đầu vào do prover kiểm soát khác đi.

V' = `05_weak_access_control_vuln.sol` + comment cuối (vẫn có lỗ hổng, hash không nằm trong registry nào). Scan thật: class Timestamp, conf 0.9992.

| Tấn công | Prover trung thực | Proof v5 hợp lệ | Destination (chaincode v1.7, block header mới) |
|---|---|---|---|
| H0 prover trung thực trên V' | không có proof (class≠0) | không | REJECT (relay chỉ còn thay proof SafeBank) |
| A2 đặc trưng giả (x của `00_clean_token`, hash của V') | — | **có** | **SAFE** cho hợp đồng có lỗ hổng |
| A3 thay trọng số (W=0, b thiên lớp 0, hash tính lại) | — | **có** | **SAFE** |
| A2d confidence giả (thật 0.6348, khai 0.9999) | không có proof (dưới ngưỡng) | **có** | **SAFE** (trước đó REJECT do "từ chối nhầm") |
| A4 bit registry giả | — | — | ví **non-admin** ngẫu nhiên, chưa nạp tiền, gọi `recordMLScanPass(V')` thành công (block 393499); + 2 approve của admin A,B như mọi ca corpus → registry: mlScanPassed=true, approvals=2 |
| A5 relay thay proof | — | — | REJECT (T2 và các ca corpus) |
| A6 header cũ | — | — | REJECT (freshness, on-chain) |

Xác nhận từ code (khớp review): `modelWeightsHash` là input riêng tư, không có trong public list → chaincode không ghim được mô hình;
`confidence` là input riêng tư chỉ đi vào `confCheck`, không suy ra từ logits → ngưỡng `minConfidence` chỉ ràng buộc prover trung thực.
Hệ quả: hai ca "từ chối nhầm" ở corpus chỉ xảy ra với prover trung thực; prover gian chỉ cần khai confidence cao.

Bài đã sửa theo kết quả: (1) câu kết Bảng ladder ("pinpoints exactly where SemBound stops being fabricable") thành "locates where a declared-class attack
stops working… does not show SemBound is unfabricable"; (2) Abstract và Conclusion; (3) chú thích Bảng ladder ("SAFE" ≠ không thể giả mạo);
(4) mục con mới `subsec:eval-attacks` + Bảng attacks; (5) Limitations (1)/(4) trỏ tới đo đạc.

Giới hạn: A4 giả định admin approve các mục có bit quét — chỉ chứng minh destination không phân biệt được, không chứng minh admin thật bị lừa. Chưa thử: tối ưu hóa x
(A2b), sửa mạch (public modelWeightsHash, margin thay confidence, ràng buộc đặc trưng) vì cần circuit + zkey mới; chưa ràng buộc role/amount (B0.3).

## X3 — ràng buộc NỘI DUNG quyết định (role, amount, …) vào state nguồn (chaincode v1.8, seq 9)

Mã: `contracts/DecisionCommitRegistry.sol` (slot 0 = authority, slot 1 = `decisionCommit`), `experiments/portability/x3_deploy_commit.js`,
`x3_fetch_state.js`; chaincode `commit.go` (`SubmitDecisionCommitted`) + `commit_test.go`. Registry: `0x891aFA35016D434782CA036fd6DBfc56A64666bc`,
authority = admin A. Kết quả thô: `x3_content_binding_results_2026-09-21.json`, `x3_commit_registry_result_2026-09-21.json`.

hD = keccak256(abi.encode(keccak(domain), keccak(intent), codeHash, keccak(role), amount, sourceLimit, epoch, expiry, nonce)). Chaincode tự tính hD
từ các trường nhận vào, chứng minh `decisionCommit[hD] == authority` từ header đã xác minh, kiểm hạn dùng/epoch, khoá replay = hD (không còn do người gọi chọn).
Unit test 19/19 PASS, gồm kiểm tra hD Go == hD ethers/ABI tính ở nguồn.

Kết quả thật (hợp đồng `03_dos_safe`, quyết định nguồn: clerk, 5000, limit 10000):
| Ca | Kết quả |
|---|---|
| quyết định đã commit | SAFE |
| nộp lại đúng quyết định đó | bị từ chối (replay theo hD) |
| relay nâng lên director 700000 | REJECT (không ai commit nội dung này) |
| relay đổi amount 5000→9999 | REJECT |
| relay đổi role clerk→manager | REJECT |
| quyết định quá hạn | REJECT (expired) |
| commit cho đích khác (domain khác) | REJECT |
| chưa từng commit (ví non-authority thử commit → giao dịch revert, status 0) | REJECT |
| đối chứng: `SubmitDecisionTrustless` cũ, cùng phép nâng quyền | **SAFE** |

Giới hạn: authority là 1 địa chỉ được ghim, không bị tấn công trong thí nghiệm; không kiểm tra quyết định của authority có đúng không; hạn dùng/độ tươi dựa trên
timestamp giao dịch do client Fabric đặt (chưa kiểm chứng client có dịch chuyển được không — mục X4 còn mở); chỉ 1 quyết định (clerk) được thử, chưa chạy 24-case.

## Các bản vá văn bản đã áp dụng (P1–P6, A2, A3, A4, A5) — cùng ngày
P1 methodology; P2 TOCTOU; P3 non-goals; P4 conclusion + đoạn giới hạn cuối bài; P5 obligation (3 chỗ); P6 T6 (5 Reevaluate + 1 Reject, corpus-003);
A2 abstract mới (bỏ câu "worked counterexample" không có thật; nêu tình trạng chứng minh T0–T6); A3 thứ tự assurance dạng cặp (cl, q), ECDSA=(L1,0), sửa chứng minh T4
cho thứ tự tích; A4 bỏ ngôn ngữ nội bộ (KHÔNG thêm cite mới: cần bạn cung cấp ref cho các bài đồng hành); A5 định nghĩa metric (B2: 8/24 exact, 13/24 accept-vs-not) và nói rõ
B1 và B4 cùng luật. Sửa thêm 2 chỗ lỗi thời: câu "chaincode commit ở sequence 1, không redeploy" và câu "mạng Besu sinh block bình thường trong mọi phép đo".
Chưa làm: A7 (related work, các trích dẫn chưa xác minh), A8 (Artifact).

## Relation v6 (sửa hai trong ba tấn công X1) — chaincode v1.9, seq 10

Mạch: `pqc-hardhat/circuits_v6/ml_scan_proof_v6.circom`, build `build_v6.sh` (436,454 ràng buộc so với 435,604 của v5; 6 public signal:
scanPassed, solidityCodeHash, commitmentHash, maxVulnClass, minMargin, modelWeightsHash). Thay đổi: (1) `modelWeightsHash` thành public để đích ghim mô hình;
(2) bỏ input tự do `confidence`, thay bằng margin tính từ logit (winner phải dẫn mọi logit khác ≥ minMargin công khai). Setup Groth16 mới trên pot20 sẵn có,
1 đóng góp phase-2 của chính mình (`snarkjs zkey verify` → "ZKey Ok!"), cùng mức tin cậy với v5 (một bên). Build mất ~76 giây (ghi chú cũ "2–3 giờ" không còn đúng).
Ngưỡng margin KHÔNG chỉnh theo dữ liệu: margin ≥ ln(0.7·11/0.3)=3.2452 logit ⇒ softmax confidence ≥ 0.70 (12 lớp) ⇒ 54,445,307 ở thang 4096². Ghim `modelWeightsHash` = hash thật của lớp cuối.
Chaincode: dispatch theo số public signal (6 = v6, 5 = v5 bị TỪ CHỐI mặc định, `allowLegacyV5=false`); 25/25 Go test PASS (gồm A3 bị bắt ở model hash, A2 vẫn verify — ghi nhận giới hạn).

Kết quả thật:
| Tấn công | v5 (đích) | v6 (đích) |
|---|---|---|
| A1 khai lớp | không có proof | không có proof |
| A2 đặc trưng giả | SAFE | **SAFE** (vẫn thành công; donor margin 5.53) |
| A3 thay trọng số | SAFE | **REJECT** ("modelWeightsHash is not the pinned model") |
| A2d confidence giả | SAFE | không diễn đạt được (không còn input confidence) |
| proof v5 nộp cho chaincode v1.9 | — | REJECT (v5 không được chấp nhận) |

Đánh đổi (nói thẳng): chỉ **6/24** hợp đồng có proof v6 hợp lệ (v5: 13/24), vì ngưỡng margin là điều kiện đủ nên chặt hơn ngưỡng confidence cũ;
corpus 24 ca đồng thuận với ground truth **17/24**, 7 ca lệch đều là từ chối nhầm (`05_weak_access_control_safe`, `07_tod_safe`, `08_timestamp_safe`,
`09_mishandled_exception_safe`, `10_gas_exception_safe`, `11_external_bug_safe`, `00_clean_voting`). Không hạ ngưỡng để xoá chúng.
Còn lại: features chưa được ràng vào mã nguồn (A2). Kết quả thô: `x6_v6_corpus_and_attacks_*.json`, `x6_v6_prover_summary_*.json`.

## X7 — chi phí verifier tách riêng (Go benchmark, `chaincode/bench_test.go`)
i7-13620H, 1 luồng, 5 lần × 200 vòng, median: header + 2 slot registry 0.407 ms; Groth16 v6 pairing + policy 0.847 ms; commit proof (hD + MPT) 0.016 ms;
tổng ≈ 1.3 ms/quyết định. Đây đo mã verify, KHÔNG phải đường endorse của peer (đo qua CLI chỉ chặn được ở mức vài ms vì BatchTimeout 2 s).

## X4 — nguồn thời gian (đọc mã, chưa thử client sửa)
`core/endorser/msgvalidation.go` (Fabric main) không nhắc tới Timestamp: proposal không bị kiểm timestamp với đồng hồ peer ⇒ `GetTxTimestamp` do client đặt.
Relay chính là người nộp ⇒ relay độc hại có thể làm header cũ/quyết định quá hạn trông như mới; ràng buộc 24 h và expiry chỉ bảo vệ khỏi trễ vô tình.
Bài đã sửa mọi chỗ nói "stale" bị từ chối thành "khi timestamp của người nộp trung thực". Chưa làm: clock on-ledger do authority ký; chưa thử client sửa timestamp thật.

## X5a — phản ví dụ thật cho hợp thành 2-hop (T5), mạng settlement chainId 1338

Hạ tầng bật lại sau reboot: cầu nối netns (Bước 2) + 16 tiến trình `pqc_tunnel` settlement (Bước 4.5, 4 VM) → peers 3, block chạy lại. `MultiHopRelay` (0x61b882…AddE, có sẵn) có
`relayDecision(decisionId, semBoundIn, …)` KHÔNG kiểm soát truy cập và nhận semBoundIn từ người gọi; hop không hề kiểm B0.
Thí nghiệm (`x5a_launder.js`): ví ngẫu nhiên non-owner, chưa nạp tiền, gọi relayDecision(semBound=true) cho quyết định DISHONEST (VulnerableBank, bit registry B0 = false) → block 72132, hop record semBound=true.
- Đường boolean, theo từng hop (relay chuyển record của B1 cho `SubmitDecision`): **SAFE** — sai đầu-cuối dù mỗi bước "hợp lệ".
- Đường trustless (đích tự xác minh B0; B1 chỉ là kênh): **REJECT** — header B0 verify được, mlScanPassed chứng minh false, approvals=2 (đúng kịch bản T0); proof của relay bị ràng codeHash khác.
Kết quả thô: `x5a_two_hop_counterexample_*.json`, `x5a_b1_launder_tx_*.json`. Điều chưa chứng minh: hợp thành trustless QUA một hop có chứng thực (X5b).

## X5b — hợp thành trustless qua hop có chứng thực (chaincode v2.0, seq 11)

Hợp đồng `contracts/Ibft2RegistryAttester.sol` (trên B1, chainId 1338): `attest()` tự dựng lại hash header B0 + hash committed-seal, ecrecover seal, ≥3 signer phân biệt thuộc tập validator B0 ghim,
rồi verify MPT của tài khoản registry và của mlScanPassed[h]/approvalCount[h]; không có setter của owner. Sinh mã Solidity sau 2 lần sửa "stack too deep" (không bật viaIR).
Test cục bộ trên EVM in-process với DỮ LIỆU THẬT của B0 (`x5b_test_local.js`): honest attest (745k gas); dishonest, header sửa timestamp, node proof hỏng đều revert đúng lý do.
Trên mạng settlement thật: attester stage-3 `0x8816CD3d…439A` và attester stage-6 `0x102d294E…74f4`; honest status 1 (744k/889k gas), dishonest status 0, laundering (hash dishonest + proof của honest) status 0; attested {honest:true, dishonest:false}.
Đích (`twohop.go`, 5 Go test PASS): ghim 4 validator B1 + địa chỉ attester; verify header B1 + proof slot attested[h] + Groth16 v6 trực tiếp.
Kết quả thật: honest **SAFE**; dishonest **REJECT**; laundering bằng proof của honest **REJECT**; hop không hạn chế của X5a đưa ra làm attester **REJECT** (không được ghim).
Thêm phụ thuộc: ≥3/4 validator B1 trung thực + đúng đắn bytecode attester (chưa audit); vẫn giới hạn: tập validator tĩnh, 1 header/chuỗi, không hỗ trợ node inline, timestamp do client đặt, chỉ 2 hợp đồng.
Ở 2 dòng dishonest đích cũng từ chối vì proof bị thay; việc hop tự từ chối được chứng minh ở B1 (status 0), không phải chỉ ở đích.
Hạ tầng: cầu nối netns + 16 pqc_tunnel settlement dựng lại (Bước 2 + 4.5) — nên cập nhật REBOOT_RECOVERY nếu cần.

## X7 — tấn công đặc trưng giả ở quy mô corpus (v6, chaincode v2.0)
`x7_forge_variants.py`, `x7_register_variants.js`, `drivers/x7_run.py`. 11 hợp đồng có lỗ hổng của corpus (gồm 7 ca dishonest-provider) → mỗi cái 1 biến thể (source + comment, hash chưa có trong registry nào).
Donor = `04_call_to_unknown_safe` (margin 5.53). Registry stage 6: 1 ví non-admin gọi `recordMLScanPass` cho 11 biến thể + 2 approve của admin A,B (33 tx thật).
Kết quả: bằng chứng trung thực → destination chấp nhận **2/11** (đúng 2 ca classifier tự cho qua: `03_dos_vuln`, `08_timestamp_vuln`); đặc trưng giả → chấp nhận **11/11**.
(Với biến thể có comment, `09_mishandled_exception_vuln` có margin 3.220 < sàn 3.245 nên không còn proof trung thực; bản gốc là 3.248.)

## X4 — thử client sửa timestamp (đã làm, chaincode v2.1 seq 12)
Thêm hàm chẩn đoán chỉ-đọc `TxTimestamp()` (trả về `GetTxTimestamp().Seconds`). Client Go tự dựng và ký proposal (`drivers/tsclient/main.go`, danh tính Admin@org1, ECDSA P-256, low-S),
đặt `ChannelHeader.Timestamp` = now + offset, gửi thẳng `Endorser.ProcessProposal` tới peer0.org1:
offset 0 → 1790014345; −30 ngày → 1787422345; +30 ngày → 1792606345; peer status 200 và chaincode trả về ĐÚNG giá trị đã giả. ⇒ timestamp hoàn toàn do client chọn;
ràng buộc độ tươi 24 h và expiry chỉ chống trễ vô tình, không chống relay độc hại. Chưa làm: clock on-ledger do authority ký.

## X4-bis — đồng hồ trên sổ cái (chaincode v2.3, seq 15), 2026-09-22
Mã: `chaincode/clock.go` (`SetClock` chỉ `Org1MSP`/CN `org1admin`, chỉ tăng; `LedgerClock`; `ledgerNow` fail-closed nếu chưa đặt). `SubmitDecisionTrustless`, `SubmitDecisionCommitted`, `SubmitDecisionTwoHop` dùng `ledgerNow`; `TxTimestamp` giữ làm chẩn đoán. Triển khai v2.2/seq 13 → (biến thể đối chứng v2.2-ts/seq 14, `useLedgerClock=false`) → bản cuối v2.3/seq 15 (đồng hồ còn nguyên sau nâng cấp, chứng minh state giữ lại).
Client: `drivers/tsclient` (thêm `-args`, `-org`). Scripts: `x4_commit_fresh.js` (X4_MODE=short cho quyết định TTL 100 s), `x4_fetch_state.js`, `drivers/x4_clock_run.py`.

| Phép thử | Kết quả |
|---|---|
| `LedgerClock` khi chưa đặt | lỗi "no ledger clock … (fail-closed)" |
| `SetClock` bởi Admin Org2 (mô phỏng) | bị từ chối: "only Org1MSP/org1admin may set the clock" |
| `SetClock(now)` bởi Admin Org1 (giao dịch thật) | status 200 |
| `SetClock(now-100)` | bị từ chối: "clock may not move backwards" |
| `LedgerClock` với timestamp giả 0 / −30 ngày / +30 ngày | luôn trả đúng 1790014669 (giá trị sổ cái); `TxTimestamp` vẫn trả đúng giá trị giả |
| E_fresh, E_expired (expiry 1700000000) với timestamp giả 0 / −90.000.000 s / +30 ngày | SAFE / REJECT ở cả ba mức (E_expired vẫn hết hạn cả khi lùi, nên không phân biệt được hai cơ chế) |
| S_short (TTL 100 s, header trước hạn), sau khi hết hạn, **đồng hồ chưa được nâng**, ts thật và ts giả −110 s | **SAFE cả hai** (giới hạn còn lại: tươi = so với lần cập nhật đồng hồ gần nhất) |
| S_short, đồng hồ đã nâng lên 1790014935, ts 0 / −110 / +30 ngày | REJECT cả ba |
| S_short trên biến thể **dựa timestamp giao dịch** (seq 14) ts thật / −110 / −115 | REJECT / **SAFE** / **SAFE** → tấn công lùi timestamp thành công trên biến thể cũ |
| S_short trên bản cuối (seq 15) ts 0 / −110 / −115 | REJECT ×3 |
| Giao dịch thật bản cuối: E_fresh, E_expired | SAFE (commitVerified, semBound) / REJECT ("decision expired at 1700000000, transaction time 1790014935") |

Phạm vi và giới hạn: chỉ đường `SubmitDecisionCommitted`, một hợp đồng (03_dos_safe, proof v6 c05), một quyền lực đồng hồ (Org1 admin, test-network một tổ chức); hai entry point còn lại dùng cùng `ledgerNow` nhưng không chạy lại trực tiếp (30 unit test vẫn PASS); corpus 24 ca và ma trận tấn công X1/X7 chạy TRƯỚC khi có đồng hồ (dùng timestamp giao dịch) và không chạy lại. Header vượt đồng hồ quá 300 s bị từ chối → quyền lực phải nâng đồng hồ gần bằng tốc độ sinh block của nguồn. Không kiểm tra header cũ hơn 24 h trên đường sống với đồng hồ (chỉ unit test). Tấn công lùi timestamp chỉ chứng minh trên biến thể seq 14 bằng mô phỏng endorsement (tsclient), không commit.
Kết quả JSON: `x4_clock_set.json`, `x4_clock_mined.json`, `x4_result.json`, `x4_short_result.json`.

## X8 — P4-tối-thiểu: ràng buộc đặc trưng qua một scanner authority được pin (mạch v7, chaincode v2.5, seq 17), 2026-09-22

Mục tiêu: đóng property P4 (Table `tab:secprops`) — hiện A2 (đặc trưng giả) phá `SemBound` ở mọi phiên bản đã xây kể cả v6, corpus-scale 11/11 hợp đồng lỗ hổng được chấp nhận `Safe`. Thiết kế: `circuits_v7/ml_scan_proof_v7.circom` (kế thừa v6, 474.110 constraint so với 436.454 của v6) thêm public input `featureHash` = digest Poseidon-chunked (18 vòng, CHUNK=15, khớp thuật toán `modelWeightsHash` đã có) của vector đặc trưng `x[256]`. Hợp đồng mới `FeatureCommitRegistry.sol` (slot0 `scannerAuthority`, slot1 mapping `featureHash`, ghi một lần, chỉ `scannerAuthority` được ghi) triển khai thật tại `0x5774D352bF8c6eBa283cD5904AD69B1689BE4CC2` (deploy block 413202, `scannerAuthority` = ví B = `0x3ff00c0960daf07eD230C4a789dB66E43fD77600`, cũng là một trong 4 validator được pin — hạn chế do chỉ có 2 ví có quỹ trong `.env`). Chaincode: `featurebind.go` (mới, không đổi `SubmitDecisionTrustless` cũ) — entry point `SubmitDecisionTrustlessV7` yêu cầu proof 7 tín hiệu công khai và chứng minh `featureHash[codeHash]` bằng MPT proof dưới cùng header đã xác minh, khớp `public[6]` của proof.

Build v7: `bash circuits_v7/build_v7.sh` (compile+setup+contribute+export, ~90 s), `snarkjs zkey verify` OK. `go build/vet/test` (30 test cũ) vẫn PASS sau khi thêm `featurebind.go` + case 7 trong `zkverify.go`.

Script trích đặc trưng thật: `x8_v7_featurebind.py` (dùng lại `gateway.MLScanner`, `lastlayer_witness_v5.py`, digest mới `gateway/feature_digest_cli.js`).

| Ca | Mô tả | valid (circuit) | featureHash proof khớp registry? |
|---|---|---|---|
| HONEST | `04_call_to_unknown_safe` (margin 5.53), đặc trưng thật của chính nó | true | có (thật) |
| V' honest scan | `05_weak_access_control_vuln`+comment, class 8 (Timestamp), margin 7.15 — giá trị mà scanner trung thực SẼ commit | — | (tham chiếu) |
| A2 v7 forged | đặc trưng của donor, gắn codeHash của V' | true | KHÁC honest_featureHash_for_victim (đúng như thiết kế) |

Deploy+commit thật (`x8_deploy_featurecommit.js`): commit featureHash(donor) block 413203, commit featureHash(victim, trung thực) block 413205, **ví lạ (không phải scannerAuthority) thử ghi đè bằng giá trị giả → status 0 (revert)**.

Giao dịch mined thật trên `SubmitDecisionTrustlessV7` (chaincode v2.5/seq 17, đồng hồ sổ cái đã nâng lên hiện tại trước khi chạy — nếu không sẽ bị "header timestamp is in the future"):

| decisionID | codeHash | semBound | evAssuranceOk | class | lý do (zkVerificationDetail) |
|---|---|---|---|---|---|
| x8-v7-honest-r3 | donor (thật) | true | true | SAFE | "featureHash proven to match the pinned scanner authority's on-chain commitment" |
| x8-v7-A2-r3 | V' (giả) | false | true | REJECT | "proof's public featureHash does not match the pinned scanner authority's on-chain commitment for this codeHash (forged-feature attack rejected)" |

Phạm vi và giới hạn: chỉ thử lại dạng đơn-quyết-định của A2 (donor tốt nhất, giống thí nghiệm A2 gốc) — quy mô corpus 11 hợp đồng được làm tiếp ở mục X9 ngay dưới đây. `scannerAuthority` không được chứng minh trung thực — cơ chế chỉ dời biên tin cậy từ "mọi prover" xuống "một khóa được pin", và ở testbed này khóa đó trùng với một validator đồng thuận (không độc lập hoàn toàn với compromise ở tầng consensus). A4 (`recordMLScanPass` không kiểm soát truy cập) vẫn còn nguyên — V' vẫn được đăng ký `mlScanPassed=true` bởi một ví non-admin, evAssuranceOk vẫn `true` trong ca A2 vì đó là trục riêng.
Kết quả JSON: `x8_out/x8_summary.json`, `x8_chain_result.json`, `x8_chain_test_state.json`, `x8_invoke_r3_results.json`.

## X9 — P4 ở quy mô corpus 11 hợp đồng (tái sử dụng đúng 11 biến thể + đăng ký của X7), 2026-09-22

Dùng lại nguyên vẹn `x7_variants.json`/`x7_registry_result.json` (11 hợp đồng lỗ hổng, biến thể `+ "// x7-variant\n"`, đã có `mlScanPassed=true`/2 approvals trên registry `0x4BDF7d0CC77F907FeDfc1Ab6B39fa9f4336869c7`, cùng donor `04_call_to_unknown_safe` margin 5.532). Mới: `x9_v7_corpus_forge.py` tính `featureHash` trung thực cho từng biến thể + proof v7 (honest bằng đặc trưng thật, forged bằng đặc trưng donor) — resumable (bỏ qua case đã có file, vì máy có lúc bị OOM do 10 VM OpenStack chạy song song, một lần `snarkjs prove` bị SIGKILL và chạy lại thành công). Kết quả circuit-level: **honest valid 2/11** (đúng `03_dos_vuln` margin 5.509, `08_timestamp_vuln` margin 4.568 — khớp chính xác 2 ca v6 đã chấp nhận), **forged valid 11/11** (khớp v6).

`x9_deploy_featurecommit.js`: commit `featureHash` trung thực cho cả 11 codeHash lên `FeatureCommitRegistry` (block 415102–415116, cùng registry đã deploy ở X8), rồi fetch header + MPT proof (ml-registry slot 3/5, feature-registry slot 1) cho cả 11 ở cùng một block (415116).

`x9_invoke.py`: 13 giao dịch mined thật lên `SubmitDecisionTrustlessV7` (2 honest + 11 forged, sau khi nâng đồng hồ sổ cái lên hiện tại):

| Loại | Kết quả |
|---|---|
| Honest (i=04, 03_dos_vuln) | SAFE, semBound=true |
| Honest (i=14, 08_timestamp_vuln) | SAFE, semBound=true |
| Forged (cả 11, i=00,02,04,06,08,10,12,14,16,18,20) | **REJECT ×11**, semBound=false, lý do: "proof's public featureHash does not match the pinned scanner authority's on-chain commitment for this codeHash (forged-feature attack rejected)" |

Kết luận: cấu trúc tấn công từng khiến destination chấp nhận **11/11** hợp đồng lỗ hổng dưới v6 nay bị chặn **0/11** dưới v7 + FeatureCommitRegistry, đường honest không bị ảnh hưởng (vẫn 2/11 như v6, đúng do sàn margin không đổi). Giới hạn: `scannerAuthority`/A4 như đã nêu ở X8; chưa test trường hợp `scannerAuthority` tự làm sai (không nằm trong phạm vi P4-tối-thiểu).
Kết quả JSON: `x9_summary.json`, `x9_chain_state.json`, `x9_invoke_r1_results.json`.

## Baseline hệ thống ngoài — UCAN (`go-ucan` v1.2.0), 2026-09-22

Bối cảnh: thử tìm baseline hệ thống ngoài thật (không tự dựng). Weaver/Cacti (Hyperledger) — đã thử nghiêm túc, tự tay deploy chaincode `interop` thật (nguồn `weaver-dlt-interoperability`, weaver-labs, repo đã archived 04/02/2026) lên **chính `authchannel` đang sống** (song song `authcc`, version 1.0/seq 1 — `querycommitted` xác nhận cả 2 org approve), pull thành công 4 image Docker thật (`weaver-relay-server:1.6.0`, `weaver-fabric-driver:1.6.0`, `cacti-weaver-relay-server:latest`, `cacti-weaver-driver-fabric:latest`). Nhưng phát hiện quyết định: **không có logic xác minh Besu ở phía Fabric trong BẤT KỲ thế hệ nào của Weaver** (`grep -rln besu` trên cả interop-cc cũ lẫn mới đều rỗng — chỉ có `besu/view_data.proto` là định nghĩa kiểu dữ liệu, không có code xác minh). Làm cho đúng cặp Besu→Fabric sẽ phải tự viết cả driver gRPC lẫn logic xác minh Besu bên trong chính chaincode `interop` của Weaver — quay lại đúng rủi ro "tự dựng" mà việc tìm baseline ngoài muốn tránh. Dừng hướng Weaver, chuyển sang UCAN theo quyết định của người dùng.

`go-ucan` v1.2.0 (`github.com/ucan-wg/go-ucan`, thật, đang duy trì, `go get` thành công, dùng fixture khóa thật `didtest.PersonaAlice/Bob`) — mã hóa đúng ví dụ chạy xuyên suốt bài (§motivation-auth-vs-admissibility): nguồn cho `manager` duyệt tới \$1.000.000 (policy UCAN thật `["<=", ".amount", 1000000]`), đích chỉ cho tới \$500.000 (luật riêng, UCAN không biết). Gọi thẳng `Token.ExecutionAllowed()` — hàm xác minh thật của thư viện, không sửa:

| Ca | amount | Kết quả `ExecutionAllowed()` |
|---|---|---|
| Trong ngưỡng | \$300.000 | ALLOWED |
| Widening (đúng ví dụ bài) | \$700.000 | **ALLOWED** (đúng thất bại kiểu B2 — chấp nhận dù vượt ngưỡng đích) |
| Đối chứng: vượt cả ngưỡng nguồn | \$1.500.000 | REJECTED, lỗi thật: `the following UCAN policy is not satisfied: ["<=", ".amount", 1000000]` |

Ca đối chứng chứng minh verifier có kiểm tra chính sách thật (không phải rubber-stamp), nên kết quả ALLOWED ở \$700K có ý nghĩa: UCAN không sai, nó chỉ không biết khái niệm "ngưỡng đích riêng biệt".
Kết quả: `evidence/ucan_baseline/main.go`, `go.mod`, `go.sum`, `ucan_baseline_output.txt`.

## X13–X15 — tách `scannerAuthority` khỏi validator (chaincode v2.6, seq 18), 2026-09-22

Mục tiêu: đóng giới hạn nêu ở X8/X9 — `scannerAuthority` ban đầu (ví B) trùng một trong 4 validator IBFT2, nên compromise của nó không độc lập với compromise tầng consensus. Phát hiện thêm (chưa sửa, ngoài phạm vi lần này): ví A (deployer + `policyAuthorityAddress` của `DecisionCommitRegistry`/T7) **cũng** là một validator — không đụng tới trong lần sửa này vì người dùng chỉ yêu cầu sửa P4/scanner authority.

`x13_deploy_featurecommit_independent.js`: tạo ví C hoàn toàn mới (`0x4A3234209922FCCc56a358E2801502d5074B7d5b`, không phải validator, không phải ví A/B), deploy `FeatureCommitRegistry` mới tại `0x7Cb4419EebEE62Cd823DD119407C424e835537AE` (block 422470, `scannerAuthority`=ví C), commit lại **13 featureHash thật** (donor + victim X8 + 11 biến thể X9, dùng đúng giá trị đã tính trước đó, không tính ML lại) — 13 giao dịch mined, tất cả status 1.

`featurebind.go`: đổi hằng `featureCommitRegistryAddress` sang địa chỉ mới. `go build` sạch. Redeploy chaincode `authcc` v2.6/seq 18 (`network.sh deployCC`), cả 2 org approve, commit thành công.

`x14_fetch_combined_proofs.js`: fetch lại cả ml-registry proof (registry cũ, không đổi) và feature-registry proof (registry mới) ở **cùng một block** (422617) cho 13 codeHash, vì header chỉ có một stateRoot.

`x15_invoke_independent.py`: nâng đồng hồ sổ cái, chạy lại đúng 13 case (X8: honest + A2 forged; X9: 2 honest + 11 forged) qua `SubmitDecisionTrustlessV7` trên chaincode v2.6/seq 18:

| Loại | Kết quả |
|---|---|
| Honest (X8 donor, X9 i=04, i=14) | **SAFE ×3**, semBound=true |
| Forged (X8 A2, X9 11 biến thể) | **REJECT ×12**, semBound=false, cùng lý do featureHash-mismatch |

Kết quả **giống hệt** lần chạy với ví B (validator) — cơ chế không phụ thuộc việc `scannerAuthority` có phải validator hay không, nên tách ra không đổi hành vi, chỉ loại bỏ tương quan compromise với consensus. Bài đã cập nhật: §eval-p4 (đoạn "This does not remove the trust..."), caption Table `tab:featurebind` (nay ghi rõ v2.6/seq.18 + registry độc lập), đoạn corpus-scale, bullet residual-trust "Scanner authority compromise", dải phiên bản chaincode (934, Open Science).
Kết quả JSON: `x13_featurecommit_independent_2026-09-22.json`, `x14_combined_state_2026-09-22.json`, `x15_invoke_independent_r1_results_2026-09-22.json`.

## Baseline UCAN mở rộng — widening corpus 9 case, 3 role (`ucan_widening_corpus`), 2026-09-22

Vấn đề phát hiện khi định mở rộng UCAN lên corpus 24-case: corpus đó (`stage4_corpus_and_baselines.js`) chỉ biến thiên `semBound`/`evAssuranceOk`, **không có source limit riêng cho từng role** — trong khi UCAN chỉ có thể biểu diễn đúng trục widening (source limit ≠ destination limit), không hề có khái niệm SemBound/EvAssuranceOK. Chạy UCAN qua 24 case sẽ đo sai thứ (không phải mở rộng trung thực của running example). Đã hỏi người dùng và được chọn phương án: bộ case widening mới, tự thiết kế, dùng đúng tỉ lệ đã công bố cho `manager` (source = 2× destination) áp cho cả 3 role đích đã có trong `RoleThreshold` (`manager≤500K, director≤2M, clerk≤10K`), amount tại 0.6×/1.4×/3.0× ngưỡng đích (đúng điểm tương đối \$300K/\$700K/\$1.5M của running example với `manager`). Các số này là lựa chọn của chúng tôi cho lần mở rộng này, không phải số đã đo/công bố ở đâu khác, cố định trước khi chạy `go-ucan`.

`ucan_widening_corpus/main.go` (module Go riêng, cùng `go-ucan` v1.2.0 pin qua `go.sum`, tái dùng đúng API `delegation.Root`/`invocation.New`/`Token.ExecutionAllowed` như `ucan_baseline/main.go`): 9 case thật (3 role × 3 điểm amount).

| Role | in-bounds | widening | control |
|---|---|---|---|
| manager (dest \$500K, src \$1M) | ALLOWED | **ALLOWED (sai)** | REJECTED |
| director (dest \$2M, src \$4M) | ALLOWED | **ALLOWED (sai)** | REJECTED |
| clerk (dest \$10K, src \$20K) | ALLOWED | **ALLOWED (sai)** | REJECTED |

Kết luận: lỗi widening của UCAN không đặc thù riêng role `manager`/mức \$700K của running example — cùng lỗi lặp lại đúng như vậy trên cả 3 role đích của bài, 3/3 in-bounds đúng, 3/3 widening sai (ALLOWED), 3/3 control đúng (REJECTED, cùng thông báo lỗi thật của thư viện). Bài đã cập nhật: Abstract, §eval-corpus (đoạn mới "Widening at corpus scale" + bảng `tab:ucan-widening`), Limitations (mục "No corpus-scale..."), Contributions item cuối, Open Science.
Kết quả: `evidence/ucan_widening_corpus/main.go`, `go.mod`, `go.sum`, `ucan_widening_output.txt`.
