# Sprint 16f — Penyatuan Branch `satria` + `fix/owner-outlet-mitra`, Audit Lintas Proyek, dan Perbaikan Beta-Readiness

## Ringkasan

Sprint 16f bukan sprint fitur baru, melainkan **penyatuan (merge) dua branch backend yang sudah berjalan
paralel** ke `achul`, diikuti **audit besar-besaran lintas proyek** (backend `backend_crm_piposmart` +
frontend `crm_piposmart`) untuk memastikan keduanya siap dibangun sebagai satu proyek beta testing.

1. Merge `origin/satria` (export data, seeder `--preset=real`, penyesuaian entitas owner/outlet) ke `achul`
   — fast-forward bersih, 1 commit.
2. Merge `origin/fix/owner-outlet-mitra` (refactor customer/lead outlet-centric, fitur riwayat tipe partner)
   ke `achul` — auto-merge tanpa conflict.
3. Audit mendalam (5 subagent paralel per domain + verifikasi manual tiap temuan) menemukan **8 bug nyata**
   yang lolos dari kedua branch sumber — 3 kritis (kebocoran data / bypass RBAC), 4 tinggi (SQL pasti gagal,
   breaking API contract, transaction leak), 1 sedang (timezone). Semua diperbaiki di `achul`.
4. Audit lanjutan ke frontend `crm_piposmart` (yang branch pasangannya — `satria`, `fix/owner-outlet`,
   `feat/owner-outlet-mitra` — sudah ter-merge ke `main` lebih dulu, ditulis terhadap backend versi BELUM
   diperbaiki) menemukan 2 titik frontend yang perlu disesuaikan supaya konsisten dengan backend yang sudah
   diperbaiki (gating RBAC partner-type, gating export Excel owner-outlet).
5. Semua perbaikan diverifikasi hidup: `go build`/`go vet`/`go test` (backend), `tsc --noEmit`/`npm run
   build` (frontend), **dan live smoke test** terhadap backend yang benar-benar dijalankan (login sungguhan,
   panggil endpoint yang tadinya pasti gagal, konfirmasi sekarang 200 dengan data benar; konfirmasi role
   SALES sekarang ditolak 403 di endpoint yang tadinya terbuka untuk semua role).

---

## Perubahan Backend (`backend_crm_piposmart`)

### 1. Kebocoran data & bypass RBAC (Kritis)
- **`internal/customer/repository.go` `ExportOwnerOutlets`**: mengabaikan `actor`/`params` sepenuhnya —
  endpoint `GET /owners/export` mengembalikan seluruh data owner/outlet/saldo wallet ke role apa pun yang
  login. Diperbaiki dengan memakai `ownerWhere(actor, params)` yang sudah dipakai `ListOwners`, sehingga
  scoping visibilitas dan filter berlaku sama seperti endpoint list biasa.
- **`internal/customer/handler.go` `downloadOwnerOutletExcel` / `downloadOwnerExcel`**: nol pengecekan
  role — siapa pun yang login bisa unduh file Excel asli 2.8MB berisi PII owner. Karena kedua endpoint ini
  membaca file mentah di disk (bukan query DB), tidak bisa di-scope per baris — digate ADMIN-only
  (`actorCanManageOwners`), konsisten dengan pola endpoint sensitif lain di file yang sama.
- **`internal/partner/handler.go` `DeletePartnerType`**: nol pengecekan actor/role — role mana pun bisa
  hapus tipe partner master. Ditambah `DeletePartnerTypeWithMeta` (gate `canManagePartnerType`, plus audit
  log `partner_type.delete` yang sebelumnya tidak pernah tercatat).

### 2. Query yang pasti gagal setiap dipanggil (Tinggi)
- **`internal/analytics/repository.go` `outletSubscriptionStatusCounts`**: query butuh 13 placeholder `?`
  tapi hanya diberi 12 argumen statis — untuk actor ADMIN (kasus paling umum, 0 argumen visibilitas
  tambahan) selalu error `sql: expected 13 arguments, got 12`. Diperbaiki dengan memetakan ulang argumen ke
  posisi placeholder yang benar (branch NOT_SUBSCRIBE/EXPIRED/NEW dikembalikan persis seperti sebelum
  refactor; 3 branch baru AKAN_JATUH_TEMPO/JATUH_TEMPO/TELAH_JATUH_TEMPO memakai `monthEnd` sebagai titik
  referensi, konsisten dengan tier yang sama di `classifyOutletSubscription`).
- **`internal/reporting/repository.go`**: trailing comma sebelum `FROM` pada query `ReportAdminOwner` dan
  `ReportAdminOwnerOutlet` — syntax error MySQL murni, kedua laporan itu tidak pernah berhasil generate.
  Koma dihapus.

### 3. Breaking API contract (Tinggi)
- **`internal/customer/types.go` `NewOwnerResponse`**: field `status` pada response owner berubah arti dari
  status lifecycle asli (`ACTIVE`/`DELETED`) menjadi status langganan (`SUBSCRIBE`/`TRIAL`/`NOT_SUBSCRIBE`)
  — field `subscription_status` baru ditambahkan membawa nilai yang sama, tapi `status` asli hilang total.
  Dikembalikan ke `owner.Status` (lifecycle asli); `subscription_status` tetap ada sebagai field terpisah.

### 4. Resource leak (Tinggi)
- **`internal/partner/repository.go` `CreateCommissionRule`**: `defer tx.Rollback()` terhapus saat
  menambah UPDATE baru di transaksi yang sama — 4 jalur error sebelum `tx.Commit()` membocorkan koneksi DB.
  `defer tx.Rollback()` dikembalikan.

### 5. Sedang / kebersihan
- **`internal/customer/handler.go` `parseDateOnly`**: `time.UTC` → `time.Local` (tidak konsisten dengan
  konvensi UTC di seluruh codebase). Dikembalikan ke `time.UTC`.
- **`internal/partner/partner_type_history.go` & `service.go`**: dua klausa bypass `RoleCode == ""` (sisa
  desain lama, tidak lagi tercapai lewat request HTTP normal setelah `AuthMiddleware`) dihapus dari
  `canManagePartnerType` dan `CreateCommissionRule`.
- **`go.mod`**: dependency `github.com/jmoiron/sqlx` ditambahkan tapi tidak pernah dipakai di file manapun
  — dihapus lewat `go mod tidy`.

### 6. Diketahui, belum diperbaiki (butuh keputusan desain lintas proyek)
- **`internal/lead/repository.go`**: lead "virtual" (outlet tanpa row `customer_leads`) memakai ID outlet
  sebagai id lead (`COALESCE(cl.id, ot.id)`). Karena `customer_leads.id` dan `outlets.id` adalah sequence
  auto-increment independen, tabrakan ID kecil bisa terjadi — `GET/POST /leads/{id}` bisa salah kena lead
  lain yang tidak terkait. Perbaikan menyeluruh butuh skema penomoran ID baru yang dikoordinasikan dengan
  frontend (yang saat ini mengonsumsi id ini langsung sebagai identitas navigasi) — di luar cakupan sprint
  ini, didokumentasikan sebagai risiko terbuka.

---

## Perubahan Frontend (`crm_piposmart`)

Audit menemukan bahwa branch pasangan frontend (`satria`, `fix/owner-outlet`, `feat/owner-outlet-mitra`)
sudah ter-merge duluan ke `main`, ditulis terhadap backend versi lama (sebelum bug di atas diperbaiki).
Sebagian besar sudah kompatibel dengan versi backend yang sudah diperbaiki tanpa perubahan apa pun (lihat
detail di `sprint-16f.md`). Dua celah nyata ditemukan dan diperbaiki:

### 1. Gating role tipe-partner tidak sinkron dengan backend
- **`app/menu/kelolaan-mitra/page.tsx`** & **`app/menu/kelolaan-mitra/jenis-mitra/page.tsx`**: tombol
  create/edit/delete tipe partner sebelumnya tampil untuk role `ADMIN`, `SUPERVISOR`, dan `""` (fallback
  loading) — backend sekarang ADMIN-only untuk operasi ini. SUPERVISOR yang mengklik akan dapat 403 yang
  membingungkan. Digate ulang jadi `currentRole === "ADMIN"` saja. `jenis-mitra/page.tsx` dipecah jadi 3
  flag terpisah (`canManageTypes` ADMIN-only, `canCreateRules` ADMIN+SUPERVISOR, `canDeactivateRules`
  ADMIN-only) supaya cocok persis dengan 3 tingkat izin berbeda di backend (type CRUD vs commission-rule
  create vs commission-rule deactivate).

### 2. Export Excel Owner-Outlet tidak digate & pesan error generik
- **`app/menu/owner-outlet/page.tsx`**: tombol "Export Data" (unduh Excel) tidak digate role sama sekali —
  backend sekarang ADMIN-only untuk endpoint ini (file mentah, tidak bisa di-scope per baris). Ditambah
  pengecekan role (`getProfile()` → `canExportOwnerExcel`) untuk sembunyikan tombol dari non-ADMIN, plus
  pesan error 403 yang jelas ("Anda tidak memiliki izin untuk mengunduh data ini.") menggantikan alert
  generik, untuk kasus race/fallback saat role belum termuat.

---

## Verifikasi

- **Backend**: `go build ./...`, `go vet ./...`, `go test ./...` — semua hijau (kecuali `cmd/run_subs`
  yang sudah rusak sejak sebelum sprint ini, tidak tersentuh perubahan ini).
- **Frontend**: `npx tsc --noEmit` bersih, `npm run build` sukses penuh (58 route ter-generate).
- **Live smoke test** (backend dijalankan sungguhan terhadap DB dev lokal, login sungguhan sebagai
  ADMIN dan SALES demo user):
  - `POST /analytics/outlets/subscription-status-recap/query` → sebelumnya pasti `500`, sekarang `200`
    dengan 7 bucket data asli.
  - `GET /reports/admin_owner` & `/reports/admin_owner_outlet` → sebelumnya pasti syntax error, sekarang
    `200` dengan baris data asli.
  - `GET /owners/export` → `200`, data ter-scope (bukan lagi dump semua tanpa filter).
  - `GET /owners` → field `status` sekarang `"ACTIVE"` (lifecycle), `subscription_status` terpisah
    `"TRIAL"`.
  - `DELETE /partner-types/{id}` sebagai SALES → `403 FORBIDDEN` (sebelumnya akan berhasil untuk role
    apa pun).
  - `GET /owners/export/download-owner` sebagai SALES → `403`; sebagai ADMIN → `200` dengan file `.xlsx`
    asli.

## Dokumentasi Terkait

- [sprint-16f.md](./sprint-16f.md) — Laporan resmi hasil pelaksanaan Sprint 16f, termasuk daftar lengkap
  temuan audit per domain dan hasil audit kompatibilitas frontend yang tidak memerlukan perubahan.
