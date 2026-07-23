# Sprint 08 - Closing dan Laporan Penjualan

## Sprint

Sprint 08

## Periode

23 Juli 2026

## Status

`GREEN`

Sprint Goal tercapai untuk scope Sprint 08: remark score `3` menghasilkan laporan closing yang konsisten, atomic, memiliki snapshot catalog, dan dapat dikelola sampai status `PENDING_RECONCILIATION`, `CONFIRMED`, atau `REJECTED`.

Catatan roadmap: integrasi reconciliation penuh dengan wallet/order/subscription memang direncanakan pada Sprint 10, sehingga tidak dianggap blocker Sprint 08.

## Sprint Goal

Remark score `3` menghasilkan laporan closing yang konsisten dan dapat diaudit, dengan snapshot package, tenor, harga, promo, serta perhitungan nominal.

## Committed Deliverables

- Sales closing.
- Snapshot package, tenor, harga, dan promo.
- Perhitungan `base_price`, `discount_amount`, `additional_charge`, `unique_transfer_code`, dan `final_amount`.
- Status `PENDING_RECONCILIATION`, `CONFIRMED`, dan `REJECTED`.
- Remark score `3` dan closing dalam satu database transaction.
- ClosingFactory.
- Seed closing gratis, berbayar, dan tanpa promo.
- API documentation dan smoke test.

## Completed

- [x] Migration `sales_closings`.
- [x] API list/detail/create/confirm/reject closing.
- [x] Soft delete, restore, force delete, dan bulk mutation closing.
- [x] Snapshot package, plan, dan promotion pada transaksi closing.
- [x] Perhitungan nominal tanpa `float64`.
- [x] Status lifecycle closing: pending, confirmed, rejected.
- [x] Create closing membuat interaction remark score `3` dan stage history `CLOSING` dalam satu database transaction.
- [x] Endpoint interaction biasa menolak remark score `3`, sehingga closing tidak dapat dicatat tanpa laporan penjualan.
- [x] ClosingFactory dan demo seeder untuk closing tanpa promo, promo gratis, dan promo berbayar.
- [x] OpenAPI diperbarui ke `0.8.0-sprint-8`.
- [x] README root diperbarui dengan route closing.
- [x] API Testing Report Sprint 08 diperluas dengan route coverage dan negative/error cases.

## Not Completed / Carry Over

Tidak ada carry over untuk scope Sprint 08.

Catatan dependency roadmap:

- Item: Reconciliation penuh antara closing, wallet, order, dan subscription.
- Penyebab: Sudah direncanakan sebagai scope Sprint 10.
- Estimasi ulang: Tidak perlu re-estimate Sprint 08; lanjut sesuai roadmap Sprint 10.

## Demo Evidence

Smoke test API lokal dijalankan pada port `18080` agar tidak terkena proses API lama yang berjalan di port `8080`.

| Area | Evidence | Result |
| --- | --- | --- |
| Auth | Login Admin dan Sales dummy | PASS |
| Catalog lookup | Business 12 bulan dan promo FREE | PASS |
| Create closing | Sales pemilik lead membuat closing | PASS |
| Atomic rule | Closing mencatat interaction remark 3 + stage history | PASS |
| Snapshot | Detail closing menampilkan package/plan/promo snapshot | PASS |
| Confirm | Admin confirm pending closing | PASS |
| Reject | Admin reject pending closing | PASS |
| List/filter | Filter status, date range, search, pagination, sorting | PASS |
| Delete lifecycle | Soft delete, restore, force delete | PASS |
| Bulk lifecycle | Bulk soft delete, restore, force delete | PASS |
| RBAC | Sales tidak bisa closing lead Sales lain | PASS |
| Error handler | Duplicate closing, invalid promo, final negative, invalid decimal, invalid sort, invalid date | PASS |
| Activity guard | Remark 3 via interaction biasa ditolak | PASS |

Contoh evidence utama:

- Sales pemilik lead membuat closing: `201 Created`, status `PENDING_RECONCILIATION`, `final_amount = 1698723.00`.
- Detail closing: `plan_snapshot.code = BUSINESS_12_MONTHS`, `promotion_snapshot.code = FREE_1_MONTH_BUSINESS_12`.
- Admin confirm closing: `200 OK`, status `CONFIRMED`.
- Sales lain membuat closing untuk lead tersebut: `403 FORBIDDEN`.
- Discount terlalu besar: `400 FINAL_AMOUNT_NEGATIVE`.
- Promo tidak eligible: `400 INVALID_PROMOTION`.
- Remark 3 lewat endpoint interaction biasa: `400 INVALID_TRANSITION`.

Dokumen API Testing detail:

- `docs/sprint-08/README.md`

## Seeder Evidence

| Command | Result |
| --- | --- |
| `go run . migrate up` | PASS |
| `go run . seed master` | PASS |
| `go run . seed demo --preset=minimal --seed=20260723 --as-of=2026-07-01` | PASS |
| `go run . migrate down` untuk Sprint 08 lalu `go run . migrate up` | PASS |
| Demo seed setelah migration down/up | PASS |

Seeder demo menghasilkan:

- closing tanpa promo;
- closing dengan promo gratis/free duration;
- closing dengan promo berbayar/device bundle.

## Quality

| Quality Gate | Result |
| --- | --- |
| `go test ./...` | PASS |
| `go vet ./...` | PASS |
| `go build .` | PASS |
| Migration up/down | PASS |
| Seeder master/demo | PASS |
| Smoke test API Sprint 08 | PASS |
| OpenAPI updated | PASS |
| README route updated | PASS |

## Defect Terbuka

Tidak ada defect blocker atau critical untuk scope Sprint 08.

## Impediments

- Sandbox normal sempat menolak read/exec/edit file existing karena ACL Windows.
- Mitigasi: Validasi dijalankan memakai command escalated terbatas pada folder project.

## Risiko Baru

| Risiko | Dampak | Mitigasi | Owner |
| --- | --- | --- | --- |
| Reconciliation penuh baru masuk Sprint 10 | Closing confirmed Sprint 08 belum otomatis mengaktifkan subscription/order/wallet debit | Pending closing belum dihitung KPI confirmed final; Sprint 10 menyambungkan order, wallet debit, subscription, dan reconciliation | Backend Engineer |
| Domain finansial rawan double counting | Revenue top-up dan closing dapat salah dibaca bila dicampur | Dokumentasi menegaskan closing adalah performance Sales, bukan revenue top-up | Backend Engineer |

## Keputusan yang Dibutuhkan

Tidak ada keputusan baru untuk Sprint 08.

## Rencana Sprint Berikutnya

Sprint 09: Payment dan Wallet Ledger.

Fokus Sprint 09:

- top-up owner;
- wallet account;
- ledger credit/debit/adjustment/refund;
- row locking;
- idempotency external reference;
- seed top-up terpakai dan belum terpakai.
