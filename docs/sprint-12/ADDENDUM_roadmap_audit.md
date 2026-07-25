# Addendum — Roadmap Audit & Concurrency Fix (pre-Sprint 13)

## Konteks

Sebelum memulai Sprint 13, dilakukan audit progress Sprint 1–12 terhadap `BACKEND_PLAN_SPRINT.md`
(roadmap resmi 18-sprint) untuk memastikan implementasi tidak melebar (scope creep) atau kurang dari
komitmen (under-delivered). Dokumen ini mencatat temuan audit dan satu fix yang langsung dikerjakan
sebagai hasilnya, sebelum Sprint 13 dimulai. Sprint 12 sendiri (`sprint-12.md`) tidak diubah — dokumen
ini murni addendum berisi temuan setelahnya.

## Temuan Audit

### 1. Scope creep — Large seeder (Sprint 11b/11c) adalah deliverable Sprint 17

Roadmap menaruh **"Demo seeder preset `standard` dan `large`"** secara eksplisit di **Sprint 17**
(Hardening, Performance, dan Dataset Besar — persis sebelum UAT). "Sprint 11b/11c" (penomoran informal,
bukan bagian dari 1-18 resmi) mengerjakan seeder 18.000-owner lengkap dengan growth curve simulation —
itu deliverable Sprint 17, dikerjakan ~6 sprint lebih awal, menghabiskan beberapa siklus debug (4 bug:
unique constraint, timeline algorithm, timestamp, note collision). Kerjaan itu tetap berguna, tapi
dikerjakan di luar urutan roadmap dan alokasi kapasitas (1 engineer, 1 minggu/sprint) yang direncanakan.

**Status**: Tidak dibatalkan (sudah berjalan & tervalidasi), tapi dicatat sebagai deviasi jadwal.
Rekomendasi: saat Sprint 17 tiba nanti, seeder ini sudah "selesai lebih awal" — kapasitas Sprint 17 bisa
dialokasikan ke item lain (load test, security review) tanpa perlu membangun ulang seeder besar.

### 2. Under-scope — Sprint 12 Commission lebih sempit dari rencana roadmap

| Deliverable roadmap Sprint 12 | Realita | Gap |
|---|---|---|
| Commission rule **fixed/percentage/tier** | Hanya PERCENTAGE/FIXED (`CHECK` constraint `chk_partner_commissions_mode`) | **TIER mode tidak ada** |
| **Effective date** & **package scope** pada commission rule | `partner_types.commission_mode/value` flat, tanpa `effective_from/to`, tanpa scoping per package (beda dengan `subscription_plans`/`promotions` Sprint 7 yang punya pola ini) | **Tidak ada rule versioning/scoping** |
| **Payout dan payout item** (entity terpisah) | Tidak ada tabel `payouts`/`payout_items`. Pembayaran per-commission individual via `PATCH .../pay` | **Payout batch entity tidak ada** — DoD "Total payout = jumlah payout item" tidak bisa dipenuhi secara struktural |

**Status**: Dicatat sebagai backlog resmi (lihat bagian Backlog di bawah), bukan dikerjakan sekarang.
Sprint 13 (Target/KPI/Ranking) tidak bergantung ke gap-gap ini sehingga aman untuk lanjut, tapi
sebaiknya ditutup sebelum Sprint 16 (Dashboard/Reporting yang akan menampilkan data payout & komisi).

### 3. Bug nyata — Partner assignment tidak punya DB-level concurrency guard ✅ FIXED

Sprint 11 DoD mengharuskan **"satu mitra hanya memiliki satu PIC aktif"**, sama seperti aturan
`customer_leads` (Sprint 5) yang diamankan dengan generated column + `UNIQUE KEY`
(`uq_customer_leads_owner_id` / `uq_lead_assignments_one_active`). `partner_assignments` (Sprint 11)
hanya punya index biasa (`idx_partner_assignments_partner_active`), **bukan constraint** — service layer
melakukan check-then-deactivate-then-insert lewat 3 panggilan repository terpisah **tanpa transaksi**,
sehingga dua panggilan `AssignPIC` bersamaan bisa menghasilkan dua PIC aktif sekaligus (race condition).

Ini beda kategori dari 2 temuan di atas — bukan scope gap, tapi **bug konkuren nyata** pada fitur yang
sudah "selesai". Diperbaiki langsung (murah untuk fix, pola sudah ada sebagai preseden):

**Fix**:
1. Migration `20260724001100_partner_assignment_concurrency_guard.sql` — menambahkan
   `active_partner_key` (generated column) + `UNIQUE KEY uq_partner_assignments_one_active`.
   - **Catatan teknis**: memakai `VIRTUAL`, bukan `STORED` seperti pola `lead_assignments`. Menambah
     kolom `GENERATED ... STORED` via `ALTER TABLE` pada MySQL 8.0.46 gagal dengan
     `Error 1215 (HY000): Cannot add foreign key constraint` — pesan menyesatkan, terjadi karena
     `partner_assignments` punya FK keluar (`partners`, `users`) dan InnoDB menolak rebuild inplace
     maupun copy untuk kombinasi STORED+FK ini. `VIRTUAL` menghindari jalur rebuild tersebut dan MySQL
     tetap mendukung index UNIQUE di atas kolom virtual — jaminan constraint-nya identik.
2. `Repository.AssignPIC()` baru (`internal/partner/repository.go`) — membungkus seluruh
   deactivate-lama + insert-baru dalam **satu transaksi** dengan `SELECT id FROM partners WHERE id=? FOR UPDATE`
   sebagai row lock, menyerialkan `AssignPIC` bersamaan untuk `partner_id` yang sama. Constraint
   `uq_partner_assignments_one_active` jadi backstop kalau lock ini pernah dilewati.
3. `mapDuplicateError()` menambah case `uq_partner_assignments_one_active` → `ErrInvalidAssignment`
   (error ini sudah ada sejak awal tapi sebelumnya tidak pernah bisa ter-trigger karena tidak ada
   constraint untuk dilanggar).
4. `Service.AssignPIC()` disederhanakan — dari 3 panggilan repo terpisah (racy) jadi 1 panggilan
   transactional.

**Validasi**:
- Insert manual 2 baris aktif langsung via SQL untuk `partner_id` yang sama → baris kedua ditolak
  (`ERROR 1062 Duplicate entry '1' for key 'uq_partner_assignments_one_active'`).
- End-to-end via API: `AssignPIC` berurutan (user 10 lalu user 11) → assignment lama otomatis
  `active=false`, yang baru `active=true`.
- **5 request `AssignPIC` bersamaan** (`user_id` 10-14, dikirim paralel via `curl ... &`) ke partner yang
  sama → seluruhnya sukses (diserialkan oleh row lock), hasil akhir **tepat 1 active assignment**
  (`SELECT COUNT(*) FROM partner_assignments WHERE partner_id=1 AND active=TRUE` → `1`).

## Backlog Resmi (Sprint 12 gap, belum dikerjakan)

Dicatat sebagai carry-over resmi, target penyelesaian sebelum Sprint 16:

1. **TIER commission mode** — tambah mode ketiga selain PERCENTAGE/FIXED di `partner_commissions`/`partner_types`
   (perlu desain tabel tier threshold, mis. `commission_tiers` dengan range closing amount → rate).
2. **Effective-dated commission rule + package scope** — pisahkan rate dari `partner_types` flat field
   menjadi entity `commission_rules` bertanggal (`effective_from`/`effective_to`, opsional `package_id`),
   mengikuti pola `subscription_plans`/`promotions` (Sprint 7). Perlu keputusan bisnis: apakah rate lama
   tetap berlaku untuk closing yang sudah PENDING saat rate berubah (snapshot sudah menangani ini secara
   parsial, tapi tanpa effective date, tidak ada cara menjadwalkan perubahan rate di masa depan).
3. **Payout & PayoutItem entity** — tabel `payouts` (batch, status, `paid_at`, `total_amount`) dan
   `payout_items` (link ke `partner_commissions`), agar satu pembayaran bisa mencakup banyak commission
   sekaligus dan DoD "Total payout = jumlah payout item" bisa diverifikasi.
