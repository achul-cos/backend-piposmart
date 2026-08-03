# Sprint 16b Report

Sprint: 16b  
Periode: 3 Agustus 2026  
Status: GREEN

## Sprint Goal

- Menambahkan sistem upgrade paket subscription yang sedang aktif.
- Mendukung tanggal efektif upgrade yang dapat diatur admin untuk backfill data.
- Menjaga histori subscription lama dan subscription hasil upgrade tetap dapat diaudit.

## Completed

- Route baru `POST /api/v1/subscriptions/{subscription_id}/upgrades`.
- Validasi bahwa upgrade hanya bisa dilakukan pada subscription aktif.
- Validasi bahwa plan target harus berasal dari package dengan level lebih tinggi.
- Perhitungan prorata berdasarkan sisa hari subscription aktif.
- Upgrade dapat dikaitkan langsung ke closing sales melalui `closing_id`.
- Auto reconciliation untuk order upgrade jika `closing_id` dikirim.
- Auto partial confirm untuk upgrade jika nominal closing sales berbeda dengan nominal prorata upgrade.
- Omset closing dipin ke nominal upgrade aktual saat partial confirm otomatis terjadi.
- Wallet owner didebit otomatis untuk nilai upgrade prorata.
- Subscription lama dipotong pada `effective_start_date`.
- Subscription baru dibuat untuk sisa hari dengan `source_type = SUBSCRIPTION_UPGRADE`.
- Subscription order upgrade menyimpan histori:
  - `order_type`
  - `source_subscription_id`
  - `upgrade_effective_start_date`
  - `upgrade_original_end_date`
  - `upgrade_remaining_days`
  - `upgrade_daily_price`
  - snapshot package/plan lama
- OpenAPI diperbarui untuk route dan request baru.
- Unit test dasar ditambahkan untuk:
  - hitungan prorata
  - validasi tanggal efektif upgrade

## Not Completed / Catatan Lanjutan

- Upgrade belum otomatis dihubungkan ke closing sales tertentu.
- Response OpenAPI masih memakai envelope generik proyek, belum diperkaya dengan schema detail khusus order upgrade.
- Formula upgrade saat ini mengikuti briefing: owner membayar prorata harga plan target, bukan selisih harga old plan vs new plan.

## Risiko

- Logika prorata upgrade cukup tidak umum sehingga frontend dan admin perlu contoh konkret.
- Bila di masa depan perusahaan ingin “upgrade bayar selisih”, rumus ini harus dipisah sebagai mode bisnis berbeda.

## Mitigasi

- Menyimpan histori subscription sumber dan snapshot package/plan lama.
- Menulis dokumentasi API testing dengan contoh request/response dan error case.
- Menjaga implementasi upgrade berdiri sebagai route baru agar tidak merusak flow order biasa.
