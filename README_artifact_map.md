# Artifact map — Manuscriptsp2_Full_v1 (cập nhật 2026-09-21)

Nguồn: chaincode `~/fabric-network/authcc/` (Go, `vendor/`), hợp đồng/mạch/script `~/pqc-hardhat/`, mạng Fabric `~/authcc-network/fabric-samples/test-network`.
Chưa phát hành: cần URL/DOI ẩn danh và quyết định double-blind (xem mục A8). Khoá proving/witness chứa trọng số lượng tử hóa → không phát hành.

## Phiên bản chaincode `authcc` (mỗi bản chỉ THÊM entry point, không sửa cái cũ)
| Version / seq | Thêm | Dùng cho |
|---|---|---|
| 1.0 / 1 | `SubmitDecision`, `classify`, `ReadDecision` | corpus 24 ca, baseline, relay, ladder |
| 1.1–1.3 / 2–4 | `SubmitDecisionRBAC`, chính sách động, `SubmitDecisionFull`/`DischargeObligation` | RBAC vs ABAC, TOCTOU, 5 lớp |
| 1.4–1.5 / 5–6 | `SubmitDecisionVerified` (Groth16 v5) | nửa 1 của trustless |
| 1.6 / 7 | `SubmitDecisionTrustless` (header Besu + MPT) | §eval-trustless (5 ca) |
| 1.7 / 8 | ghim 2 registry, độ tươi 24 h | chạy 24 ca qua trustless (22/24) |
| 1.8 / 9 | `SubmitDecisionCommitted` (`commit.go`) | §eval-content (X3) |
| 1.9 / 10 | relation v6, từ chối v5 | §eval-attacks (v6, 17/24) |
| 2.0 / 11 | `SubmitDecisionTwoHop` (`twohop.go`) | §eval-multihop (X5b) |
| 2.1 / 12 | chẩn đoán `TxTimestamp` | X4: timestamp do client chọn |
| 2.3 / 15 | `clock.go` (`SetClock`, `LedgerClock`, `ledgerNow`) | X4-bis: đồng hồ sổ cái (2.2 / 13 và 2.2-ts / 14 là bước trung gian) |
| 2.4 / 16 | `featurebind.go`, `SubmitDecisionTrustlessV7` (mạch v7, P4-tối-thiểu) | X8: đơn-quyết-định, `FeatureCommitRegistry` đầu tiên (`scannerAuthority`=ví B) |
| 2.5 / 17 | sửa lỗi hiển thị `detail` (không đổi `semBound`/`class`) | X9: P4 ở quy mô corpus 11 hợp đồng, cùng registry ví B |
| 2.6 / 18 | đổi hằng `featureCommitRegistryAddress` sang registry mới | X13–X15: `scannerAuthority` độc lập (ví C, không phải validator) |
30 Go test (chưa có test riêng cho `clock.go`; kiểm tra trực tiếp trên mạng, xem X4-bis) (`go test ./chaincode/`) + 3 benchmark (`-bench .`) — fixture thật trong `chaincode/testdata/`.

## Hợp đồng và mạch
`contracts/RegistryContract.sol` (stage 3 `0x93254ACd…`, stage 6 `0x4BDF7d0C…`), `DecisionCommitRegistry.sol` (`0x891aFA35…`), `Ibft2RegistryAttester.sol` (B1: `0x8816CD3d…`, `0x102d294E…`),
`MultiHopRelay.sol` (B1 `0x61b88297…`, không hạn chế truy cập — phản ví dụ X5a). Mạch `circuits_v1..v6/` (v6: `build_v6.sh`, 436,454 ràng buộc, vk `verification_key_v6.json`).

## Kết quả → script → bằng chứng
| Kết quả trong bài | Script (`~/pqc-hardhat/experiments/portability/`) | Bằng chứng (`evidence/`) |
|---|---|---|
| 24 ca qua trustless, 22/24 | `stage6_register_corpus_besu.js`, `drivers/stage6_trustless_corpus.py`, `drivers/fetch_corpus_state.js` | `trustless_corpus_c1_results_*.json` |
| Ma trận tấn công v5 (A2, A3, A2d, A4) | `x1_attacks.py`, `x1_register_attack.js`, `drivers/x1_trustless.py` | `x1_*` |
| Relation v6 + 17/24 + A3 bị chặn | `circuits_v6/`, `x6_v6_prove.py`, `drivers/x6_corpus.py` | `x6_*` |
| Ràng buộc nội dung (X3) | `x3_deploy_commit.js`, `x3_fetch_state.js`, `drivers/x3_run.py` | `x3_*` |
| Phản ví dụ hop (X5a) | `x5a_launder.js`, `drivers/x5_fetch.js` | `x5a_*` |
| Hop có chứng thực (X5b) | `x5b_test_local.js`, `x5b_deploy_attest.js`, `x5b_fetch_b1.js` | `x5b_*` |
| Chi phí (CLI và Go benchmark) | `drivers/cost.py`, `drivers/cost2.py`, `chaincode/bench_test.go` | `trustless_destination_verification_*.md` (mục X7) |
| Timestamp client chọn + đồng hồ sổ cái (X4) | `drivers/tsclient/main.go`, `x4_commit_fresh.js`, `x4_fetch_state.js`, `drivers/x4_clock_run.py` | `x4_*`, mục X4 và X4-bis |
| Concurrency / TOCTOU / RBAC / 5 lớp | (phiên trước) | `concurrency_*`, `rbac_*`, `all_five_*` |
| P4-tối-thiểu: v7 + FeatureCommitRegistry chặn A2 (X8) | `circuits_v7/build_v7.sh`, `gateway/feature_digest_cli.js`, `x8_v7_featurebind.py`, `x8_deploy_featurecommit.js`, `x8_chain_test.js`, `x8_invoke.py` | `x8_summary.json`, `x8_chain_result.json`, `x8_chain_test_state.json`, `x8_invoke_r3_results.json`, mục X8 |
| P4 ở quy mô corpus 11 hợp đồng, 11/11→0/11 (X9) | `x9_v7_corpus_forge.py`, `x9_deploy_featurecommit.js`, `x9_invoke.py` (tái dùng `x7_variants.json`/`x7_registry_result.json`) | `x9_summary.json`, `x9_chain_state.json`, `x9_invoke_r1_results.json`, mục X9 |
| Baseline ngoài UCAN: B2's widening failure trên `go-ucan` thật | `evidence/ucan_baseline/main.go` (module Go độc lập) | `evidence/ucan_baseline/*`, mục "Baseline hệ thống ngoài — UCAN" |
| `scannerAuthority` độc lập, kết quả P4 không đổi (X13–X15) | `x13_deploy_featurecommit_independent.js`, `x14_fetch_combined_proofs.js`, `x15_invoke_independent.py` | `x13_featurecommit_independent_2026-09-22.json`, `x14_combined_state_2026-09-22.json`, `x15_invoke_independent_r1_results_2026-09-22.json`, mục X13–X15 |
| Baseline ngoài UCAN mở rộng: widening 9 ca, 3 role | `evidence/ucan_widening_corpus/main.go` (module Go độc lập) | `evidence/ucan_widening_corpus/*`, mục "Baseline UCAN mở rộng" |
Ghi chú: các driver trong `drivers/` chạy từ chính thư mục đó và đọc `corpus_state.json`, `x5b_state.json` (sinh bởi `fetch_*`) và `/tmp/corpus_proofs` (bản sao trong `drivers/corpus_v5_proofs/`).

## Còn thiếu để phát hành
- Bản sao ẩn danh + URL/DOI; hash SHA-256 của proving/verification key; danh sách địa chỉ hợp đồng theo từng thí nghiệm.
- Mạng nguồn Besu (chainId 1337/1338) và Fabric phải được bật lại theo `REBOOT_RECOVERY.md` (Bước 4, 4.5, 6-bis) trước khi chạy lại bất kỳ driver nào.
- Các bib entry bài đồng hành đã có dạng ẩn danh (`anon_pqc_companion`, `anon_zkml_companion`, `anon_composition_companion`); tiêu đề của mục cuối là chỗ giữ chỗ cần tác giả xác nhận.
