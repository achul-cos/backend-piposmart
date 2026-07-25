# Addendum 2 — Commission Rules (TIER/Effective-Date/Package Scope) & Payout Batching

## 1. Konteks

`ADDENDUM_roadmap_audit.md` mencatat 3 gap Sprint 12 sebagai "Backlog Resmi" terhadap
`BACKEND_PLAN_SPRINT.md`: TIER commission mode, effective-dated & package-scoped commission
rule, dan entity Payout/PayoutItem untuk batch pembayaran. Dokumen ini mencatat implementasi
yang menutup ketiganya sekaligus, dikerjakan sebagai lanjutan langsung dari audit tersebut
(bukan sprint baru dalam penomoran resmi).

**Keputusan bisnis yang dikonfirmasi user sebelum desain**:
- Basis TIER = volume closing confirmed kumulatif **per partner per bulan kalender** (bukan
  per-besar-closing, bukan lifetime kumulatif).
- Payout = **satu payout per partner**, membatch commission APPROVED. Endpoint individual
  `PATCH .../commissions/{id}/pay` tetap ada untuk kasus one-off.

## 2. Desain

### commission_rules — overlay opsional, additive

`commission_rules` **tidak menggantikan** `partner_types.commission_mode/value` — kalau tidak
ada rule yang cocok (partner_type + package opsional + tanggal), kalkulasi **fallback** ke rate
flat partner_type persis seperti Sprint 12. Ini dikonfirmasi lewat regression test: partner
dengan 0 baris `commission_rules` tetap menghasilkan commission dengan rate flat yang benar dan
`commission_rule_id = null`.

**Precedence resolusi** (satu query, `resolveCommissionRule` di `repository.go`):
```sql
SELECT id, mode, value FROM commission_rules
WHERE partner_type_id = ? AND active = TRUE
  AND effective_from <= ? AND (effective_to IS NULL OR effective_to >= ?)
  AND (package_id = ? OR package_id IS NULL)
ORDER BY (package_id IS NOT NULL) DESC, effective_from DESC, id DESC
LIMIT 1
```
Package-specific mengalahkan type-wide; kalau seri, `effective_from` terbaru menang. Tanggal
pembanding adalah `sales_closings.confirmed_at` (satu-satunya tanggal yang relevan karena sync
cuma memproses closing berstatus CONFIRMED).

### TIER — volume bulanan, dihitung live dari sales_closings

`monthlyClosingOrdinal` menghitung rank ke berapa (1-based) sebuah closing di antara closing
CONFIRMED partner pada bulan kalender yang sama (`YEAR/MONTH(confirmed_at)`), diurutkan
`confirmed_at` lalu `id`. `resolveTier` mencari baris `commission_tiers` yang range
`[min_closings, max_closings]`-nya mencakup ordinal tersebut. Dihitung dari `sales_closings`
langsung (bukan dari `partner_commissions`) supaya urutan sync tidak memengaruhi tier bracket.

### Rule management: create + list + get + deactivate, tanpa update

Rule effective-dated berarti **di-supersede**, bukan diedit di tempat — `DeactivateCommissionRule`
set `active=FALSE` dan menutup `effective_to` yang masih open-ended ke `CURDATE()`, konsisten
dengan pola `subscription_plans`/`promotions` (Sprint 7) yang juga tidak punya endpoint update.

### Payout — soft-release, bukan hard delete

`partner_payout_items` punya `released_at` + generated column `active_commission_key`
(`VIRTUAL`, bukan `STORED` — lihat gotcha di bawah) yang collapse ke NULL saat released, dengan
`UNIQUE KEY` di atasnya. Ini **pola yang sama persis** dengan fix concurrency `partner_assignments`
hari ini (`active_partner_key`/`uq_partner_assignments_one_active`, migration `...001100`) —
dipilih supaya idiom konsisten dan riwayat audit finansial tidak pernah dihapus (cancel payout
cuma set `released_at`, tidak menyentuh baris commission sama sekali).

**Commission tidak pernah berubah status selagi "dipegang" payout PENDING** — tetap APPROVED.
Status cuma berubah jadi PAID saat payout-nya benar-benar dibayar (`MarkPayoutPaid`). Ini
menghindari perlu status baru ("RESERVED") dan membuat cancel payout otomatis benar tanpa logic
revert tambahan — "release kembali ke APPROVED" itu benar by construction karena commission
memang tidak pernah diubah.

### Double-pay prevention

`MarkCommissionPaid` dan `CancelCommission` (yang sudah ada sejak Sprint 12) ditambah guard
`ensureCommissionNotInPayout` — `SELECT 1 FROM partner_payout_items WHERE commission_id=? AND
released_at IS NULL FOR UPDATE` di dalam transaksi yang sama dengan row lock status. `CreatePayout`
mengunci kandidat commission dengan `SELECT ... FOR UPDATE` + `NOT EXISTS` terhadap payout_items
aktif, menyerialkan terhadap panggilan individual pay/cancel maupun `CreatePayout` lain yang
konkuren untuk partner yang sama — replikasi persis pola row-lock `AssignPIC` (migration `...001100`).

## 3. Gotcha MySQL (konsisten dengan temuan hari ini)

Sama seperti migration `partner_assignments`, `partner_payout_items.active_commission_key`
memakai `VIRTUAL`. Kali ini kolomnya didefinisikan saat `CREATE TABLE` (bukan `ALTER TABLE` pada
tabel yang sudah ada FK-nya), jadi `STORED` kemungkinan aman — tapi dipertahankan `VIRTUAL` demi
konsistensi idiom dan menghindari perlu re-test asumsi itu di bawah tekanan waktu.

## 4. Hasil Validasi End-to-End

Smoke test manual di database bersih (`test_piposmart`, fixture dibuat langsung via SQL untuk 5
partner referral + closing, lalu seluruh alur rule/sync/approve/payout diuji lewat API):

| Skenario | Hasil |
|---|---|
| **(a) TIER escalation** — rule 1-3 closing→2%, 4+→5%, 5 closing bulan sama | Closing ordinal 1-3 → 2% (Rp20.000 dari Rp1.000.000), ordinal 4-5 → 5% (Rp50.000, Rp100.000). ✅ Tepat sesuai bracket |
| **(b) Effective-date + package-scope** — rule type-wide 3% (lama), rule package-specific 7% (aktif), rule type-wide 99% (`effective_from` 2099, masa depan) | Closing dengan `package_id=3` confirmed hari ini memilih rule 7% (`commission_rule_id=2`), **bukan** TIER type-wide maupun rule 99% masa depan. ✅ |
| **(c) Payout batching + double-pay block** | 2 commission APPROVED (Rp20.000+Rp20.000) berhasil dibatch jadi 1 payout `total_amount=40.000`. `PATCH .../commissions/{id}/pay` individual pada commission yang sudah dibatch → `409 COMMISSION_IN_PAYOUT`. ✅ |
| **(d) Payout pay → cascade PAID** | `PATCH .../payouts/{id}/pay` memindahkan payout dan **kedua** commission di dalamnya jadi PAID dengan `paid_by`/`paid_at` terisi. ✅ |
| **(e) Payout cancel → release** | Payout PENDING baru (2 commission) dibatalkan → kedua commission kembali `APPROVED`, tanpa `active_payout_id`. Individual pay pada salah satunya sesudahnya **berhasil**. ✅ |
| **(f) Regression — fallback rate lama** | Partner tipe `SUPPLIER` (tanpa `commission_rules` sama sekali) di-sync → commission tetap pakai rate flat partner_type (5%), `commission_rule_id=null`. Perilaku Sprint 12 tidak berubah. ✅ |
| **Concurrency spot-check** | 3 `POST .../payouts` bersamaan untuk partner yang sama dengan 2 commission APPROVED tersisa → tepat 1 request sukses (membatch keduanya), 2 lainnya `400 NO_PAYABLE_COMMISSIONS`. Tidak ada double-batch. ✅ |

## 5. File yang Diubah

| File | Perubahan |
|------|-----------|
| `migrations/20260724001200_commission_rules_tiers_payouts.sql` | 4 tabel baru (`commission_rules`, `commission_tiers`, `partner_payouts`, `partner_payout_items`) + `ALTER partner_commissions` (kolom traceability `commission_rule_id`, `tier_ordinal`) |
| `internal/partner/types.go` | Const `CommissionModeTier`, `PayoutStatus*`; struct `CommissionRule`, `CommissionTier`, `PartnerPayout`, `PartnerPayoutItem` + response/request counterparts; extend `PartnerCommission(Response)` dengan 3 field traceability |
| `internal/partner/errors.go` | 6 error baru: `ErrInvalidCommissionTier`, `ErrNoMatchingTier`, `ErrCommissionInPayout`, `ErrNoPayableCommissions`, `ErrMixedCurrency`, `ErrInvalidPayoutStatus` |
| `internal/partner/money.go` | `validateRuleCommissionValue`, `validateCommissionTiers` |
| `internal/partner/repository.go` | `resolveCommissionRule`, `monthlyClosingOrdinal`, `resolveTier`; `SyncCommissions` diupdate untuk konsultasi rule sebelum fallback; guard `ensureCommissionNotInPayout` di `MarkCommissionPaid`/`CancelCommission`; CRUD `CommissionRule` & `PartnerPayout` lengkap |
| `internal/partner/service.go` | Method baru dengan role gating identik pola commission (approve/cancel/rule = ADMIN+SUPERVISOR, pay = ADMIN saja) |
| `internal/partner/handler.go` | 9 route baru (`/partner-types/{id}/commission-rules[...]`, `/partners/{id}/payouts[...]`), extend `writeCommissionError` |
| `internal/platform/httpserver/openapi.yaml` | Versi `0.13.0-sprint-12-addendum`; 9 path + 8 schema baru; extend `PartnerCommissionResponse` |
| `internal/partner/money_test.go`, `internal/partner/partner_test.go` | Test baru untuk validasi tier/rule dan response builder (table-driven, pola sama dengan test existing) |

Semua backlog resmi dari `ADDENDUM_roadmap_audit.md` sekarang **CLOSED**. `go build`, `go vet`,
`go test ./internal/partner/...` bersih; migration reversibel (up/down/up teruji).
