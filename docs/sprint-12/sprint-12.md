# Sprint 12 - Partner Commission

## Sprint

Sprint 12

## Periode

24 Juli 2026

## Status

`GREEN`

Sprint Goal tercapai: partner mendapatkan komisi otomatis saat lead yang direferensikannya mencapai closing `CONFIRMED`, dengan ledger komisi (PENDING → APPROVED → PAID, atau CANCELLED) yang bisa diaudit per closing.

## Sprint Goal

Menghitung dan mencatat komisi mitra (partner) dari referral yang berujung closing confirmed, tanpa mengubah data komisi historis saat rate partner_type berubah di kemudian hari.

## Catatan Penting: Desain Rate Sudah Ada Sejak Baseline

`partner_types.commission_mode` (`PERCENTAGE`/`FIXED`) dan `partner_types.commission_value` sudah ada di skema sejak migrasi baseline Sprint 1 (`20260723000100_baseline_crm_schema.sql`), namun belum pernah dipakai — layer aplikasi (repository/service/handler) tidak pernah membaca atau menulis kedua kolom ini, dan `CreatePartnerType` bahkan tidak menyertakannya di query `INSERT` (berpotensi gagal karena `commission_mode` adalah `NOT NULL` tanpa default).

Implementasi awal Sprint 12 sempat menambahkan kolom `commission_rate_percent` baru langsung di tabel `partners` sebelum menyadari kolom `commission_mode`/`commission_value` di `partner_types` sudah dirancang untuk peran ini sejak awal. Desain final **memindahkan rate ke level partner_type** (sesuai skema asli) dan membatalkan kolom baru tersebut, sehingga:

- Semua partner dengan `partner_type_id` yang sama otomatis berbagi rate komisi yang sama.
- `PERCENTAGE` menerapkan `commission_value` sebagai persentase `final_amount` closing (0-100).
- `FIXED` membayar `commission_value` flat, **tidak dibatasi (uncapped)** oleh besar closing — keputusan bisnis eksplisit agar komisi tetap (mis. bonus agen regional) tidak terpotong pada closing kecil.

## Committed Deliverables

- Migrasi database `20260724001000_partner_commission.sql` — tabel `partner_commissions` (ledger).
- `PartnerType` CRUD dilengkapi `commission_mode` & `commission_value` (baca dan tulis, sebelumnya tidak terhubung sama sekali ke layer aplikasi).
- Sync komisi dari confirmed closing (`POST /partners/{id}/commissions/sync`), idempotent.
- Approve / Pay / Cancel commission dengan role gating (ADMIN/SUPERVISOR untuk approve & cancel, ADMIN saja untuk pay).
- List & detail commission, discoped per partner (cross-partner ID leakage dicegah).
- OpenAPI diperbarui ke `0.12.0-sprint-12`.
- Unit test package `internal/partner` untuk kalkulasi komisi (`money.go`) dan response mapping.
- API Testing Report Sprint 12.

## Completed

- [x] Migration `partner_commissions` (unique per `closing_id`, snapshot `commission_mode`/`commission_value` saat kalkulasi).
- [x] `PartnerType`: field `commission_mode`/`commission_value` terhubung penuh — `CreatePartnerType`, `GetPartnerTypeByID`, `ListPartnerTypes`, `UpdatePartnerType`.
- [x] Validasi `commission_value` sesuai mode (`PERCENTAGE` 0-100, `FIXED` ≥ 0) di service layer, dengan pemetaan error yang benar ke `400 VALIDATION_ERROR` (bukan `500`).
- [x] `POST /partners/{partnerID}/commissions/sync` — scan closing `CONFIRMED` dari referral partner, buat commission `PENDING` untuk closing yang belum tersinkron (unique constraint pada `closing_id` mencegah duplikat).
- [x] `PATCH .../approve` (`PENDING → APPROVED`), `.../pay` (`APPROVED → PAID`), `.../cancel` (`PENDING`/`APPROVED` → `CANCELLED`) — transisi status invalid ditolak `400 INVALID_STATUS`.
- [x] Role gating: sync & approve & cancel oleh ADMIN/SUPERVISOR; pay hanya ADMIN (uang benar-benar keluar dari bisnis).
- [x] `GET /partners/{partnerID}/commissions` & `.../commissions/{commissionID}` — commission discoped ke partner yang benar (404 jika ID valid tapi milik partner lain).
- [x] Kalkulasi cents-based (hindari floating point) dengan rounding half-up, konsisten dengan pola `money.go` modul lain (`closing`, `wallet`).
- [x] OpenAPI diperbarui ke `0.12.0-sprint-12`.
- [x] Unit test `internal/partner`: parsing/kalkulasi commission (PERCENTAGE & FIXED), response mapping (`PartnerCommission`, `PartnerType`).
- [x] Smoke test end-to-end pada environment terisolasi (lihat bagian Demo Evidence).

## Not Completed / Carry Over

- Sinkronisasi komisi masih dipicu manual via endpoint (`commissions/sync`), belum otomatis saat closing dikonfirmasi — worker (`internal/app.RunWorker`) saat ini masih stub heartbeat-only, belum ada job queue nyata untuk dikaitkan. Kandidat Sprint 13: hook otomatis di alur `ConfirmClosing` atau job worker terjadwal.
- Belum ada endpoint pembayaran massal (bulk pay) atau ekspor laporan komisi partner — carry over jika dibutuhkan tim finance.
- Seeder demo (`seed demo --preset=minimal|large`) belum membuat data referral+commission contoh; seeder terkait (`seeder.go`, `seeder_large.go`, `factory.go`) sedang aktif dikerjakan paralel untuk Sprint 11c (large seeder bug fixes) sehingga sengaja tidak disentuh pada sprint ini untuk menghindari conflict. Data komisi pada smoke test dibuat manual via API (lihat API Testing Report).

## Isolasi Development (Menghindari Konflik dengan Sprint 11c)

Sprint 11c (large seeder bug fixes) berjalan paralel menggunakan port default (`APP_PORT=8080`) dan database `test_piposmart` dari `.env`. Untuk menghindari konflik, seluruh pengembangan dan smoke test Sprint 12 dijalankan dengan **override environment variable saat invocation**, tanpa mengubah `.env`:

```bash
export DB_NAME=piposmart_sprint12
export APP_PORT=8090
go run . migrate up
go run . seed master
go run . seed demo --preset=minimal --seed=20260724 --as-of=2026-07-24
go run . bootstrap-admin
go run . api
```

Database `piposmart_sprint12` dibuat baru dan terpisah total dari `test_piposmart`; API berjalan di port `8090`, terpisah dari port `8080` milik sprint 11c.

## Demo Evidence

Smoke test API dijalankan pada environment terisolasi (`DB_NAME=piposmart_sprint12`, `APP_PORT=8090`).

| Area | Evidence | Result |
| --- | --- | --- |
| Auth | Login Admin & Sales demo | PASS |
| Partner Type Commission | `GET /api/v1/partner-types` menampilkan `commission_mode`/`commission_value` sesuai seed (`SUPPLIER`=PERCENTAGE 5.00, `AGENT`=FIXED 150000.00, dst.) | PASS |
| Nested Partner Type | `GET /api/v1/partners/{id}` — field `partner_type.commission_mode`/`commission_value` terisi benar (awalnya kosong, diperbaiki saat smoke test — lihat Bug Ditemukan) | PASS |
| Referral | `POST /api/v1/partners/3/referrals` (REF-001 → lead 3) | PASS |
| Sync PERCENTAGE | `POST /api/v1/partners/3/commissions/sync` — closing `final_amount=3768703.00`, rate `3.00%` → `commission_amount=113061.09` (tepat) | PASS |
| Sync Idempotent | Sync kedua kali → `created: 0` (tidak duplikat) | PASS |
| Approve → Pay | `PATCH .../1/approve` → `APPROVED`, `PATCH .../1/pay` → `PAID`, `approved_by`/`paid_by` terisi dari JWT actor | PASS |
| Invalid Transition | `PATCH .../1/approve` setelah `PAID` → `400 INVALID_STATUS` | PASS |
| Sync FIXED | Partner baru tipe `AGENT` (FIXED 150000.00) → referral ke closing `final_amount=1698702.00` → `commission_amount=150000.00` (flat, tidak dipotong) | PASS |
| Cancel | `PATCH .../2/cancel` (`PENDING → CANCELLED`) dengan note, lalu cancel kedua kali → `400 INVALID_STATUS` | PASS |
| Role Gating | Sales mencoba sync/pay → `403 FORBIDDEN` | PASS |
| Cross-Partner Isolation | `GET /partners/3/commissions/2` (commission 2 milik partner 4) → `404 NOT_FOUND` | PASS |
| Validasi Commission Value | `PERCENTAGE=150.00` dan `FIXED=-500.00` saat create partner type → `400 INVALID_COMMISSION_VALUE` (awalnya `500`, lihat Bug Ditemukan) | PASS |
| Validasi Mode | `commission_mode="BOGUS"` ditolak binding `oneof` → `400 VALIDATION_ERROR` | PASS |

## Bug Ditemukan & Diperbaiki Selama Smoke Test

1. **Nested `partner_type` kosong pada response Partner** — service hanya menyalin `Code`/`Name` saat menyusun ulang objek `partner_type` bersarang dalam response Partner, sehingga `commission_mode`/`commission_value` selalu kosong. Diperbaiki dengan helper `attachPartnerType()` yang menyalin keempat field sekaligus, dipakai konsisten di `CreatePartner`, `GetPartnerByID`, `GetPartnerByCode`, `ListPartners`, `UpdatePartner`.
2. **Validasi commission value mengembalikan `500` bukan `400`** — handler `CreatePartnerType`/`UpdatePartnerType` tidak memetakan `ErrInvalidCommissionRate`/`ErrInvalidMoney`/`ErrInvalidCommissionMode` ke status HTTP yang tepat, jatuh ke `default: INTERNAL_ERROR`. Diperbaiki dengan menambah case eksplisit → `400 INVALID_COMMISSION_VALUE`.

## File yang Diubah

| File | Perubahan |
|------|-----------|
| `migrations/20260724001000_partner_commission.sql` | Tabel baru `partner_commissions` (snapshot `commission_mode`/`commission_value`, unique per `closing_id`) |
| `internal/partner/types.go` | `PartnerType.CommissionMode/Value`, `PartnerCommission*` types, `attachPartnerType` support fields |
| `internal/partner/money.go` | Baru — `parseMoneyToCents`, `parseCommissionRate`, `validateCommissionValue`, `calculateCommissionAmountCents` (mode-aware) |
| `internal/partner/errors.go` | `ErrInvalidMoney`, `ErrInvalidCommissionRate`, `ErrInvalidCommissionMode`, `ErrInvalidCommissionStatus`, `ErrCommissionAlreadyExists` |
| `internal/partner/repository.go` | `PartnerType` CRUD menyertakan commission columns; sync join `partner_referrals` → `sales_closings` → `partner_types`; commission CRUD & status transitions |
| `internal/partner/service.go` | Validasi commission value di `CreatePartnerType`/`UpdatePartnerType`; `attachPartnerType` helper; commission service methods dengan role gating |
| `internal/partner/handler.go` | Routes & handlers commission; error mapping commission value ke `400` |
| `internal/platform/httpserver/openapi.yaml` | Versi `0.12.0-sprint-12`; schema & path commission baru; `commission_mode`/`commission_value` pada partner-type schemas |
| `internal/partner/money_test.go`, `partner_test.go` | Unit test kalkulasi & response mapping |

Tidak ada perubahan pada `internal/platform/seeder/*` atau `internal/platform/factory/factory.go` — keduanya sedang dikerjakan paralel untuk Sprint 11c.
