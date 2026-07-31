Sprint: 15a - Post-pitching Model Changes (Owner/Outlet/Top Up/Transfer/Subscribe/Partner)
Periode: 30 Juli 2026
Status: GREEN

Sprint Goal:
- Menerapkan serangkaian perubahan model data & alur hasil rapat pitching klien, di luar urutan
  roadmap 18-sprint resmi (informal, seperti Sprint 11b/11c sebelumnya).
- Wallet jadi murni milik Owner (bukan Outlet), Top Up dapat status lifecycle nyata, Transfer jadi
  modul terpisah dengan reconciliation matching ke Top Up, Subscribe dapat outcome PARTIAL_CONFIRM
  + toleransi balance untuk order admin manual, Partner disederhanakan jadi 3 tipe dengan komisi
  per-plan, dan closing/order bisa memakai lebih dari satu promotion sekaligus.

Committed Deliverables (semua diverifikasi live terhadap server yang berjalan, bukan cuma unit test):
- §1 Owner overview rollup: `GET /owners/{id}/overview` (wallet balance, total_transferred,
  total_topup, total_spent, age_status, subscription_status, subscribed_outlet_count) — wallet
  dihapus dari response Outlet (dulu duplikat angka yang sama per outlet milik owner yang sama).
- §5 Partner: seeder persis 3 partner type (REFERRAL/PARTNERSHIP/STRATEGIC, semua FIXED), 6 tipe
  demo lama dinonaktifkan (bukan dihapus — partner lama masih mereferensikannya). `commission_rules`
  direscope dari `package_id` ke `plan_id`. Matriks komisi 7 plan x 3 tipe = 21 baris di-seed persis
  dari `data_admin/Ringkasan_Komisi_Piposmart.pdf`. Harga 7 plan yang di-cover PDF disesuaikan persis
  (Basic/Business/Pro 12/18/24 bulan). `POST /partners` menerima `self_assign_pic` (Sales bisa jadi
  PIC atas mitra yang dia buat sendiri, atomic). `GET /partners/{id}/activity?month=YYYY-MM` →
  BELUM_MEMBERIKAN_REFERAL / TELAH_MEMBERIKAN_REFERAL.
- §2 Top Up lifecycle: `wallet_payments.status` PENDING/REJECTED/EXPIRED/ACCEPTED (dulu cuma PAID
  instan). Balance & ledger HANYA berubah saat ACCEPTED. Sesi PENDING 24 jam, auto-EXPIRE lewat
  worker tick (`ExpireStaleTopups`, bukan job type baru — bulk UPDATE idempoten tiap tick).
  `PATCH /wallet-payments/{id}/accept|reject`, `.../transfer-date`. `unique_code` dicatat terpisah
  dari nominal yang dianggap omset. Copy `MANUAL_TRANSFER`/fallback `MANUAL` → `TF/BRI`.
- §3 Transfer (modul baru `internal/transfer`): `owner_transfers` table, `POST /owners/{id}/transfers`,
  `GET /owners/{id}/transfers/suggestions` (read-only, heuristik tanggal+nominal terhadap Top Up
  PENDING owner tsb, deteksi unique code vs mismatch nyata), `POST /transfers/{id}/confirm-match`
  (accept Top Up + tandai transfer MATCHED, satu operasi logis), `.../reject-match`.
- §4 Subscribe: reconciliation outcome `PARTIAL_CONFIRM` (omset closing di-override ke angka admin,
  yang benar-benar mengalir ke `sales_closings.final_amount` sehingga commission sync partner ikut
  benar — bukan cuma catatan audit). Order admin manual yang melebihi balance owner TIDAK diblokir
  lagi — tetap dibuat, balance di-clamp ke 0 (CHECK constraint DB tidak mengizinkan negatif),
  `balance_shortfall_amount` dicatat di order. Non-Admin tetap hard-block (real live debit).
- §4b Multi-promotion: `sales_closings`/`subscription_orders` bisa memakai >1 promotion sekaligus
  (tabel baru `sales_closing_promotions`/`subscription_order_promotions`), selama semua eligible
  untuk plan yang sama — satu saja tidak eligible, seluruh request ditolak. Diskon (`additional_charge`)
  dan ekstensi durasi (FREE_DURATION benefit) dijumlah lintas SEMUA promotion, bukan cuma yang
  pertama (bug yang ditemukan & diperbaiki saat verifikasi: durasi awalnya cuma menghitung 1
  promotion dari daftar berapapun yang dipakai).

Completed (verifikasi live, per bagian):
- §1: dibuat owner+2 outlet+topup+transfer+order asli, angka rollup cocok aritmatik
  (500.000 topup - 99.000 spent = 401.000 ledger, dsb).
- §5: seeder ulang → `commission_rules` persis 21 baris sesuai PDF; partner dibuat sebagai Sales
  (bukan Admin) dengan `self_assign_pic=true` → `PartnerAssignment` langsung ada tanpa panggilan
  kedua; dikonfirmasi tidak ada endpoint update-in-place nominal commission rule (hanya
  create+deactivate — commission yang sudah didapat mitra tidak berubah retroaktif kalau rate
  diubah nanti).
- §2: top up PENDING (balance tidak berubah) → accept (balance credit + unique_code + transfer
  date override tercatat, ledger konsisten) → reject (balance tetap tidak berubah) → expire
  (dites dengan backdate `session_expires_at`, worker benar-benar mem-flip status dalam satu tick,
  log `"worker expired stale top-ups","count":1`).
- §3: transfer dibuat → suggestion computed benar (unique_code "123" utk selisih Rp 123) → confirm
  → wallet payment ACCEPTED + transfer MATCHED, `total_transferred` di Owner overview ikut naik.
  Reject-match dan double-confirm (409 ALREADY_MATCHED) juga dites.
- §4: order admin melebihi balance (Rp 2.596.000 vs balance Rp 1.500.000) tetap tercipta,
  `balance_shortfall_amount="1.096.000"`, balance jadi 0 (bukan negatif). PARTIAL_CONFIRM pada
  closing yang datanya beda dari order (Rp 3.200.000 vs Rp 2.596.000 admin) → `sales_closings.
  final_amount` benar-benar berubah jadi angka admin.
- §4b: order dengan 2 promotion pada plan yang sama → `additional_charge` terjumlah benar,
  `duration_days` bertambah benar dari kedua promotion; promotion yang tidak eligible untuk plan
  tsb → seluruh order ditolak (bukan partial-apply).
- `go build ./...`, `go vet ./...`, `go test ./...` bersih di seluruh repo di akhir sprint.

Bug ditemukan & diperbaiki di luar scope awal (side-effect dari verifikasi live, bukan bagian
rencana):
- Migration timestamp collision `20260730000100` antara migration Sprint 15a session ini
  (`sprint15_import_profiles`, sudah applied sebelumnya) dan migration `create_discussion_threads`
  yang di-commit sesi paralel lain (`7c19d5f`, belum pernah applied) — di-renumber ke
  `20260730000600` (rename file saja, tanpa ubah isi).
- `wallet.ListParams.Status` sudah ada sebagai field tapi tidak pernah dipakai di `paymentWhere` —
  filter status pada `ListPayments` diam-diam tidak berfungsi. Ditambahkan.
- `totalDurationDays` (helper lama) hanya membaca promotion PERTAMA dari daftar — kalau promotion
  kedua/ketiga yang punya FREE_DURATION benefit, durasinya akan salah dihitung. Diganti
  `totalDurationDaysMulti` yang menjumlah dari seluruh daftar promotion.

Not Completed / Carry Over:
- **Modul Katalog (sisa)**: user menyebut "modul katalog" dua kali sebagai bagian perubahan, tapi
  detail yang diberikan (sebelum pesan lompat ke Top Up) hanya berisi multi-promotion (§4b). Belum
  dikonfirmasi apakah ada perubahan katalog lain yang dimaksud di luar itu.
- **Integrasi admin dashboard**: skema (`source`/`external_reference` pada `owner_transfers`)
  sengaja dibuat integration-agnostic, tapi konektor otomatisnya (API / akses DB langsung / project
  scraper terpisah) belum dipilih maupun dibangun — eksplisit "masih rencana" per pernyataan user.
- Frontend (owner tab reorganisasi, badge status outlet baru/lama/tidak-aktif, filter tanggal
  jatuh-tempo 3-arah, halaman Mitra Sales) — di luar scope repo backend ini, dicatat untuk sesi
  frontend terpisah.
- Estimasi: Katalog perlu klarifikasi user (~5 menit tanya-jawab) sebelum dikerjakan atau ditutup
  sebagai "tidak ada tambahan"; integrasi admin dashboard perlu keputusan arsitektur terpisah
  (kemungkinan besar effort tersendiri, bukan sisipan sprint ini).

Demo Evidence:
- Endpoint yang terbukti berjalan end-to-end (curl terhadap server lokal, database MySQL
  diperiksa langsung via `mysql` CLI setelah tiap langkah):
  - `GET /owners/{id}/overview`
  - `POST /owners/{id}/wallet/topups` → `PATCH /wallet-payments/{id}/accept|reject`
  - `POST /owners/{id}/transfers` → `GET .../suggestions` → `POST /transfers/{id}/confirm-match`
  - `POST /owners/{id}/subscription-orders` (balance shortfall case)
  - `POST /subscription-orders/{id}/reconcile` (action=PARTIAL_CONFIRM)
  - `POST /owners/{id}/subscription-orders` dengan `promotion_ids:[1,3]`
  - `POST /partners` dengan `self_assign_pic:true`, `GET /partners/{id}/activity`
- Migration: `20260730000300_owner_transfers.sql`, `20260730000400_commission_rules_by_plan.sql`,
  `20260730000500_wallet_topup_lifecycle.sql`, `20260730000700_subscription_partial_confirm.sql`,
  `20260730000800_multi_promotion.sql` (+ `20260730000600_create_discussion_threads.sql`, rename
  saja, bukan milik sprint ini).

Catatan silang-referensi:
- Sprint 15 (import framework) SEMENTARA DIJEDA sebelum sprint ini dimulai, karena profil
  `BONUS_MITRA` menyentuh domain commission yang direstrukturisasi di §5 — melanjutkan Sprint 15
  di atas skema lama akan berarti membangun dua kali. Dilanjutkan setelah sprint ini.
- `docs/sprint-15/sprint-15.md` (ditulis sesi sebelumnya) mengklaim smoke test `MONTHLY_ACTIVE` dan
  `BONUS_MITRA` sudah lulus — ini TIDAK cocok dengan status kerja yang saya lacak (kedua profil itu
  belum diimplementasikan: `parseMonthlyActiveRow`/`parseBonusMitraRow` belum ada). Perlu diverifikasi
  ulang sebelum dipercaya; tidak diperbaiki di sini karena di luar scope sprint ini.
