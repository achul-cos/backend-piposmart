# Large Seeder Bug Fixes & Progress UX - Sprint 11c

## 1. Ringkasan

Sprint 11c adalah **koreksi & penyempurnaan** dari `seed demo --preset=large` yang dibangun di Sprint 11b.
Saat 11b pertama kali diuji end-to-end, ditemukan 3 bug yang membuat data hasil seed tidak sesuai
spesifikasi awal, plus satu permintaan UX (progress bar). Dokumen ini mencatat root cause, fix, dan
hasil validasi final.

| Item | Nilai |
|------|-------|
| Project | Backend CRM Piposmart |
| Sprint | Sprint 11c - Large Seeder Bug Fixes |
| Tanggal | 24 Juli 2026 |
| Status | ✅ Fixed, tested end-to-end, terdokumentasi |
| File utama | `internal/platform/seeder/seeder_large.go`, `internal/platform/factory/factory.go` |

**Catatan penting**: Sebagian angka di `docs/sprint-11b/README.md` (bagian 2 "Data Scale" dan bagian 5
"Expected SQL Output") sudah **tidak akurat** setelah fix ini — lihat bagian 3 di bawah untuk angka aktual.
Dokumen 11b tetap dipertahankan sebagai riwayat implementasi awal, tapi rujuk dokumen ini untuk perilaku
seeder yang sebenarnya berjalan sekarang.

## 2. Bug yang Ditemukan & Diperbaiki

### Bug #1 — UNIQUE constraint `customer_leads.owner_id` dilanggar

**Gejala**: `lookup customer_leads.code=OWN-000001-LEAD-02: sql: no rows in result set` saat membuat lead
ke-2 untuk owner yang sama.

**Root cause**: Migrasi `20260723000400_lead_ownership_assignment.sql` menambahkan
`UNIQUE KEY uq_customer_leads_owner_id (owner_id)` — artinya skema mewajibkan **tepat 1 lead per owner**
(lead merepresentasikan status pipeline customer, bukan record yang bisa berulang). Desain awal 11b yang
membuat 1-6 lead acak per owner melanggar constraint ini sejak lead kedua.

**Fix**: Redesain `seedDemoLarge()` agar setiap owner memiliki **tepat 1 lead**. Volume data tetap besar
lewat banyaknya interaksi per lead (5-15) dikalikan 18.000 owner.

### Bug #2 — Timeline generator berhenti di ~2.400 owner, bukan 18.000

**Gejala**: Seeder selesai tanpa error tapi hanya membuat ~2.400 owner, jauh dari target 18.000.

**Root cause**: Algoritma awal memakai `daysPerBatch := totalDays / targetCount`. Dengan rentang
2020-01-01 s.d. `as-of` (~2.400 hari) dan target 18.000, hasil pembagian integer `2400/18000 = 0`,
dibulatkan minimum ke 1 hari per iterasi — timeline habis begitu hari terakhir tercapai, jauh sebelum
18.000 entri terbentuk.

**Fix**: Ditulis ulang `generateGrowthTimeline()` memakai **largest remainder method**:
1. Hitung bobot tiap hari berdasarkan fase growth curve (startup/growth/acceleration/plateau + random spike).
2. Alokasikan `floor(bobot_hari / total_bobot * targetCount)` owner per hari.
3. Sisa alokasi (pembulatan) dibagikan ke hari-hari dengan pecahan terbesar.

Hasilnya **selalu tepat `targetCount` timestamp**, dengan banyak owner per hari yang proporsional terhadap
fase growth curve — tidak lagi tergantung jumlah hari yang tersedia.

### Bug #3 — Semua owner tercatat `created_at` di tahun berjalan (2026), bukan tersebar 2020-2026

**Gejala**: Setelah Bug #1 dan #2 diperbaiki, seeder sukses membuat 18.000 owner, tapi query
`GROUP BY YEAR(created_at)` menunjukkan 100% owner ada di tahun 2026 — timeline generator sudah benar
tapi tidak dipakai.

**Root cause**: `factory.Owner` tidak punya field `CreatedAt`, dan `CreateOwner()` melakukan
`INSERT INTO owners (...)` tanpa kolom `created_at` — MySQL otomatis mengisi `DEFAULT CURRENT_TIMESTAMP`,
sehingga timestamp hasil `generateGrowthTimeline()` (yang sudah benar tersebar 2020-2026) dihitung tapi
tidak pernah dikirim ke database.

**Fix**:
- `factory.Owner` ditambah field `CreatedAt time.Time`.
- `Factory.CreateOwner()` menyertakan `created_at` di query INSERT (fallback ke `time.Now()` jika kosong,
  agar caller lain yang belum set field ini tidak terpengaruh).
- `buildLargeOwner()` di seeder_large.go menerima parameter `createdAt` dari timeline dan meneruskannya
  ke `factory.Owner.CreatedAt`.

### Bug #4 — Interaksi saling menimpa akibat kolisi `note` pada upsert

**Gejala**: Setelah Bug #1-#3 diperbaiki, seeder sukses membuat 18.000 owner dengan distribusi temporal
benar, tapi rata-rata interaksi per lead hanya **3.8** — jauh di bawah target 5-15/lead
(`interactionsPerLeadMin=5`, `interactionsPerLeadSpan=11`). Total interaksi hanya ~68.000, bukan
~180.000 yang diharapkan.

**Root cause**: `Factory.CreateInteraction()` mengemulasikan upsert dengan pola
`DELETE FROM customer_interactions WHERE lead_id=? AND note=?` sebelum `INSERT`. Sementara
`Factory.BuildInteraction(index, score)` men-generate `Note` hanya dari `score`
(`"Demo Sprint 06 remark %d"`), **tidak dari `index`**. Karena `remarkScore := rng.Intn(4)` di
`seedDemoLarge()` menghasilkan skor 0-3 secara acak untuk tiap interaksi dalam 1 lead, interaksi
dengan skor yang sama (sangat mungkin terjadi berulang kali dalam 5-15 iterasi) memiliki `note` yang
identik — insert berikutnya menghapus insert sebelumnya dengan `note` sama. Hasilnya, jumlah interaksi
yang benar-benar tersimpan per lead terbatas ke jumlah skor unik yang muncul (maksimum 4), bukan jumlah
iterasi yang dimaksud (5-15).

**Fix**: `BuildInteraction()` diubah agar `Note` menyertakan `index`, bukan hanya `score`:
```go
Note: fmt.Sprintf("Demo Sprint 06 remark %d-%d", score, index),
```
Dicek dulu bahwa perubahan ini aman untuk pemakai lain — `seedDemoMinimal()` di `seeder.go:633` hanya
memanggil `CreateInteraction` **sekali per lead**, jadi kolisi note tidak pernah relevan di sana; format
teks yang berubah tidak memengaruhi perilakunya.

### Event masa depan (guard sejak awal 11b)

`clampToAsOf()` dipertahankan untuk memastikan `InteractionAt` dan `ClosedAt` tidak pernah melewati
tanggal `--as-of` (mencegah data yang secara logis "belum terjadi").

## 3. Progress Bar (UX Enhancement)

Sebelumnya seeder hanya mencetak `Created 1000 owners...` setiap kelipatan 1.000 tanpa indikasi progres
keseluruhan — sulit tahu apakah proses masih berjalan atau macet untuk run yang memakan waktu 7-9 menit.

**Implementasi**: `progressBar` (di `seeder_large.go`) mencetak bar in-place ke `stderr` menggunakan `\r`
(carriage return tanpa newline), mirip installer aplikasi:

```
[#############################-]  99.7%  17944/18000 owners  elapsed 8m40s  eta 1s
```

- Update dibatasi maksimum ~6-7x/detik (throttle 150ms) supaya print itu sendiri tidak jadi bottleneck.
- Menampilkan persentase, hitungan `current/total`, waktu berjalan, dan estimasi sisa waktu (ETA) yang
  dihitung dari kecepatan rata-rata sejauh ini.
- Baris terakhir diakhiri `\n` (lewat `finish()`) supaya output berikutnya tidak menimpa bar.

## 4. Hasil Validasi End-to-End (Final, Setelah Semua Fix)

Command yang dijalankan:
```bash
go run . migrate up
go run . seed master
go run . seed demo --preset=large --seed=20260724 --as-of=2026-07-24
```

Waktu eksekusi: **~9-10 menit** untuk 18.000 owner (single transaction; naik dari ~7-9 menit sebelum
Bug #4 diperbaiki karena sekarang benar-benar menyimpan 5-15 interaksi/lead, bukan tertimpa jadi ~1-4).

### Jumlah Data Aktual (setelah SEMUA fix termasuk Bug #4)

| Entity | Jumlah Aktual | Catatan |
|--------|---------------|---------|
| Owners | 18.000 | Tepat sesuai `largeOwnerCount` |
| Outlets | 18.000 | 1 outlet per owner |
| Leads | 18.000 | **Tepat 1 per owner** (bukan 1-6 seperti asumsi 11b awal — lihat Bug #1) |
| Interactions | **181.654** (min 5, max 16, avg 10.1/lead) | Sesuai target desain `interactionsPerLeadMin=5`/`Span=11` setelah Bug #4 (note collision) diperbaiki |
| Closings | 2.136 (~11.9%) | Sesuai target `closingRatePercent=12` |
| Sales users | 45 | Sesuai `largeSalesCount` |
| Supervisors | 9 | Sesuai `largeSupervisorCount` |

**Riwayat angka interactions** (untuk konteks debugging, bukan target):
- Sebelum Bug #4 fix: ~68.225 (avg 3.8/lead) — interaksi saling menimpa karena note collision
- Setelah Bug #4 fix: **181.654** (avg 10.1/lead) — sesuai desain awal

### Distribusi Temporal (Growth Curve) — Contoh Run `seed=20260724`

| Tahun | Owner | % |
|-------|-------|---|
| 2020 | 854 | 4.7% |
| 2021 | 2.250 | 12.5% |
| 2022 | 2.707 | 15.0% |
| 2023 | 2.922 | 16.2% |
| 2024 | 4.088 | 22.7% |
| 2025 | 4.129 | 22.9% |
| 2026 | 1.050 | 5.8% |

Pola ini sesuai desain growth curve: startup lambat (2020) → growth (2021-2023) → akselerasi (2024-2025)
→ plateau tahun berjalan (2026, karena `as-of` di tengah tahun).

### Integrity Check

```sql
-- 0 owner dengan >1 lead (constraint UNIQUE terjaga)
SELECT COUNT(*) FROM (
  SELECT owner_id FROM customer_leads GROUP BY owner_id HAVING COUNT(*) > 1
) x;
-- hasil: 0

-- 0 lead tanpa owner valid (no orphan)
SELECT COUNT(*) FROM customer_leads cl
LEFT JOIN owners o ON o.id = cl.owner_id
WHERE o.id IS NULL;
-- hasil: 0

-- 0 interaksi/closing melewati tanggal --as-of (clampToAsOf bekerja)
SELECT COUNT(*) FROM customer_interactions WHERE interaction_at > '2026-07-24 23:59:59';
-- hasil: 0
SELECT COUNT(*) FROM sales_closings WHERE closed_at > '2026-07-24 23:59:59';
-- hasil: 0
```

## 5. File yang Diubah

| File | Perubahan |
|------|-----------|
| `internal/platform/seeder/seeder_large.go` | 1 lead/owner, timeline largest-remainder, `clampToAsOf`, progress bar (`progressBar` struct + `newProgressBar`/`update`/`finish`), `buildLargeOwner` menerima `createdAt` |
| `internal/platform/factory/factory.go` | `Owner.CreatedAt` field baru; `CreateOwner()` menyertakan `created_at` di INSERT dengan fallback `time.Now()`; `BuildInteraction()` — `Note` kini menyertakan `index` agar unik per interaksi (fix Bug #4) |

Tidak ada perubahan skema database maupun API — sepenuhnya perbaikan logic seeder di layer aplikasi.

## 6. Cara Menjalankan (Updated)

```bash
cd backend_crm_piposmart

# Reset database (opsional, untuk hasil bersih)
mysql -u root -p -e "DROP DATABASE IF EXISTS crm_piposmart; CREATE DATABASE crm_piposmart;"

go run . migrate up
go run . seed master
go run . seed demo --preset=large --seed=20260724 --as-of=2026-07-24
```

Progress bar akan tampil otomatis di terminal (stderr) selama proses berjalan, contoh:
```
[##############################] 100.0%  18000/18000 owners  elapsed 8m41s  eta 0s
seed demo selesai (preset=large, seed=20260724, as_of=2026-07-24)
```

## 7. Known Limitations (Update dari 11b)

- Volume interaksi aktual (**181.654**, avg 10.1/lead) berbeda dari estimasi 11b awal (~378K, basis
  rata-rata 3 lead/owner) karena revisi ke 1 lead/owner (Bug #1). Basis 18.000 lead × avg 10.1 interaksi
  sudah sesuai desain (`interactionsPerLeadMin=5`, `Span=11`) setelah Bug #4 diperbaiki. Jika volume lebih
  besar dibutuhkan untuk testing tertentu, naikkan `interactionsPerLeadMin`/`interactionsPerLeadSpan` di
  `seeder_large.go`, bukan jumlah lead.
- Progress bar hanya mengukur progres pembuatan **owner** (proxy untuk keseluruhan proses, karena semua
  entity lain dibuat sinkron per-owner di loop yang sama) — bukan progres granular per jenis entity.
- Masih single transaction untuk 18K owner (~9-10 menit) — jika proses terputus (mis. sesi CLI ditutup),
  transaksi rollback total dan tidak ada partial data tersisa (ini WAI, bukan bug, tapi berarti dijalankan
  sebagai proses foreground/blocking lebih aman daripada background yang berisiko terputus).

---

**Sprint 11c - Large Seeder Bug Fixes & Progress UX**
**Status**: ✅ COMPLETE — Fixed, tested end-to-end (18.000 owners), documented
**Last Updated**: 24 Juli 2026
