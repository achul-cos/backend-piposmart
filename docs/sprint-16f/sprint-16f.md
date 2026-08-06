# Report Laporan Sprint 16f

## Informasi Umum
- **Fokus Sprint:** Penyatuan branch backend `satria` + `fix/owner-outlet-mitra` ke `achul`, audit
  besar-besaran lintas proyek (backend + frontend) untuk kesiapan beta testing, perbaikan seluruh temuan.
- **Modul Terdampak:** Customer/Owner, Outlet, Lead, Partner, Analytics, Reporting, Seeder — backend;
  Kelolaan Mitra, Owner-Outlet — frontend.
- **Repo:** `backend_crm_piposmart` (branch `achul`), `crm_piposmart` (branch `main`).

## Tujuan
1. Menyatukan dua branch backend yang berjalan paralel (`satria`: export data, seeder `--preset=real`,
   penyesuaian owner/outlet; `fix/owner-outlet-mitra`: refactor customer/lead outlet-centric, riwayat tipe
   partner) ke `achul` tanpa saling menghilangkan progress.
2. Mengaudit hasil penyatuan secara adversarial — bukan sekadar `go build` hijau — karena penyatuan dua
   branch independen yang menyentuh entitas tumpang tindih (owner, outlet, partner type) punya risiko nyata
   saling menimpa logika satu sama lain secara diam-diam.
3. Memperluas audit ke frontend `crm_piposmart` sebagai satu kesatuan proyek CRM, karena frontend sudah
   punya branch pasangan (`satria`, `fix/owner-outlet`, `feat/owner-outlet-mitra`) yang ter-merge ke `main`
   lebih dulu — ditulis terhadap versi backend yang **belum** diperbaiki.
4. Memperbaiki setiap celah yang ditemukan di kedua sisi, memverifikasi lewat build/test statis **dan**
   smoke test langsung terhadap backend yang benar-benar berjalan.

---

## Metodologi Audit

Audit dilakukan dalam dua gelombang, masing-masing 5 dan 3 subagent paralel per domain, dengan **setiap
temuan diverifikasi ulang secara manual** (baca kode langsung, hitung placeholder SQL, trace diff terhadap
baseline pre-merge) sebelum dicatat sebagai bug nyata — bukan sekadar dilaporkan mentah dari subagent.
Beberapa klaim awal subagent ditolak setelah verifikasi (lihat bagian "Temuan yang Ditolak").

Baseline pre-merge: commit `8ca85b7` (achul sebelum penyatuan). Diff penuh: `git diff 8ca85b7 HEAD` —
34 file, ~2.500 baris berubah.

---

## Rincian Perubahan — Backend

### 1. `ExportOwnerOutlets` tidak ter-scope (Kritis — kebocoran data)
- **Penyebab:** `internal/customer/repository.go` — fungsi menerima parameter `actor` dan `params` tapi
  query SQL-nya hardcoded, tidak memakai keduanya sama sekali. `GET /owners/export` mengembalikan seluruh
  owner + outlet + saldo wallet ke role apa pun yang login, tanpa peduli visibilitas per-role, dan filter
  apa pun yang dikirim client diam-diam diabaikan.
- **Perbaikan:** memakai `ownerWhere(actor, params)` — helper yang sama persis dipakai `ListOwners` —
  sehingga scoping visibilitas (`1=1` untuk ADMIN, `EXISTS(...)` untuk SUPERVISOR/SALES) dan semua filter
  (`Query`, `Province`, `City`, dst.) berlaku identik dengan endpoint list biasa.
- **Verifikasi live:** `GET /owners/export` sebagai ADMIN → `200`, data ter-scope benar.

### 2. Download Excel owner/outlet tanpa RBAC (Kritis — kebocoran data)
- **Penyebab:** `downloadOwnerOutletExcel` dan `downloadOwnerExcel` (`internal/customer/handler.go`)
  membaca file Excel mentah langsung dari disk (`asset/data_admin/...`, 2.8MB, berisi PII owner asli) dan
  stream ke response — tanpa memanggil `currentActor(c)` atau cek role sama sekali. Karena jalur ini
  bypass DB sepenuhnya, tidak ada mekanisme scoping per baris yang bisa diterapkan.
- **Perbaikan:** digate `actorCanManageOwners(currentActor(c))` (ADMIN-only) — konsisten dengan pola 18+
  operasi sensitif lain di file yang sama.
- **Verifikasi live:** sebagai SALES → `403`; sebagai ADMIN → `200` dengan file `.xlsx` valid (dikonfirmasi
  `file` command mendeteksi "Microsoft Excel 2007+").

### 3. `DeletePartnerType` tanpa RBAC (Kritis — bypass otorisasi)
- **Penyebab:** `internal/partner/handler.go` — handler delete tidak pernah memanggil
  `identity.CurrentUser(c)`, service-nya (`s.repo.DeletePartnerType(ctx, id)`) tidak menerima parameter
  actor sama sekali. Dibandingkan `UpdatePartnerType` yang sudah benar gate `canManagePartnerType`, delete
  ini terbuka untuk role apa pun.
- **Perbaikan:** ditambah `Service.DeletePartnerTypeWithMeta(ctx, actor, id, meta)` — mengikuti pola
  `CreatePartnerTypeWithMeta`/`UpdatePartnerTypeWithMeta` yang sudah ada: gate `canManagePartnerType`,
  catat `before` state, dan tulis audit log `partner_type.delete` (sebelumnya penghapusan sama sekali tidak
  tercatat di riwayat tipe partner).
- **Verifikasi live:** sebagai SALES, `DELETE /partner-types/{id}` → `403 FORBIDDEN` (sebelumnya akan
  berhasil untuk role apa pun).

### 4. `outletSubscriptionStatusCounts` — argumen SQL kurang satu (Tinggi — pasti gagal)
- **Penyebab:** `internal/analytics/repository.go` — rewrite untuk memecah bucket lama `JATUH_TEMPO`
  (whole-month match) menjadi 3 bucket baru (`AKAN_JATUH_TEMPO`/`JATUH_TEMPO`/`TELAH_JATUH_TEMPO`, mirror
  tier yang sama dengan `classifyOutletSubscription`) menambah 3 placeholder `?` baru di query (dari 10
  jadi 13), tapi daftar argumen statis hanya ditambah 2 (dari 9 jadi 11). Untuk actor ADMIN (0 argumen
  visibilitas tambahan — kasus paling umum), query **selalu** gagal dengan
  `sql: expected 13 arguments, got 12`.
- **Perbaikan:** argumen dipetakan ulang persis 13 posisi — 7 posisi pertama (NOT_SUBSCRIBE, EXPIRED, NEW)
  dikembalikan **identik** dengan nilai pre-merge (diverifikasi lewat `git show 8ca85b7:...`); 5 posisi
  baru (AKAN/JATUH/TELAH) memakai `monthEnd` sebagai titik referensi "hari ini", konsisten dengan konvensi
  yang sama di `classifyOutletSubscription` (lihat Sprint sebelumnya — penyatuan `satria`+`fix/owner-
  outlet-mitra` juga sempat merusak fungsi itu, sudah diperbaiki di commit awal sprint ini).
- **Verifikasi live:** `POST /analytics/outlets/subscription-status-recap/query` → `200` dengan 7 series
  bucket data asli (sebelumnya `500` di setiap panggilan).

### 5. Trailing comma sebelum `FROM` (Tinggi — pasti gagal)
- **Penyebab:** `internal/reporting/repository.go` — query `ReportAdminOwner` dan `ReportAdminOwnerOutlet`
  (`case ReportAdminOwner` / `case ReportAdminOwnerOutlet`) diakhiri `..., AS jumlah_outlet,\nFROM owners o`
  — koma sebelum `FROM` adalah syntax error MySQL murni. Kedua jenis laporan itu tidak pernah berhasil
  generate sejak rewrite.
- **Perbaikan:** koma dihapus di kedua query.
- **Verifikasi live:** `GET /reports/admin_owner` dan `GET /reports/admin_owner_outlet` → `200` dengan
  baris data asli (sebelumnya syntax error di setiap panggilan).

### 6. `NewOwnerResponse.Status` — breaking API contract (Tinggi)
- **Penyebab:** `internal/customer/types.go` — sebelum sprint sebelumnya, `Status: owner.Status` memetakan
  status lifecycle asli DB (`ACTIVE`/`DELETED`) ke field JSON `status`. Rewrite `fix/owner-outlet-mitra`
  mengubahnya jadi `Status: subStatus` (status langganan `SUBSCRIBE`/`TRIAL`/`NOT_SUBSCRIBE`) sambil
  menambah field baru `subscription_status` membawa nilai yang sama — field `status` asli (ACTIVE/DELETED)
  hilang total dari response, padahal frontend sudah ditulis mengharapkan `status` berarti lifecycle
  (dikonfirmasi via audit frontend, lihat bawah).
- **Perbaikan:** dikembalikan `Status: owner.Status`; `SubscriptionStatus: subStatus` tetap sebagai field
  terpisah (tidak menghapus fitur baru, hanya mengembalikan field lama yang tergeser).
- **Verifikasi live:** `GET /owners` → `"status":"ACTIVE"`, `"subscription_status":"TRIAL"` — dua field
  independen seperti seharusnya.

### 7. Transaction leak di `CreateCommissionRule` (Tinggi)
- **Penyebab:** `internal/partner/repository.go` — `defer tx.Rollback()` terhapus saat menambah query
  UPDATE baru (menonaktifkan rule lama sebelum insert rule baru) di transaksi yang sama. 4 jalur error
  sebelum `tx.Commit()` tidak lagi rollback maupun commit — setiap kegagalan (constraint violation,
  validasi tier, dst.) membocorkan satu koneksi DB dari pool.
- **Perbaikan:** `defer tx.Rollback()` dikembalikan tepat setelah `BeginTx`.

### 8. `parseDateOnly` memakai timezone lokal (Sedang)
- **Penyebab:** `internal/customer/handler.go` — `time.ParseInLocation(..., time.Local)` menggantikan
  `time.UTC` sebelumnya. Filter tanggal (`created_from`/`start_date`/dst.) jadi ter-parse memakai timezone
  proses server, bukan UTC — tidak konsisten dengan seluruh codebase yang selalu `.UTC()`.
- **Perbaikan:** dikembalikan `time.UTC`.

### 9. Bypass `RoleCode == ""` (Rendah — dead code, dibersihkan)
- **Penyebab:** `canManagePartnerType` (`internal/partner/partner_type_history.go`) dan
  `CreateCommissionRule` (`internal/partner/service.go`) punya klausa `|| actor.RoleCode == ""` — sisa pola
  yang sah untuk *background job* tanpa actor HTTP (lihat `internal/reporting/service.go:159`, konteks job
  queue), tapi dipakai keliru di jalur handler HTTP murni di mana `identity.CurrentUser(c)` sesudah
  `AuthMiddleware` seharusnya selalu mengembalikan role non-kosong. Bukan exploit aktif yang terbukti
  (butuh data integritas rusak di DB untuk tercapai), tapi tetap sisa kode yang tidak seharusnya ada.
- **Perbaikan:** kedua klausa dihapus.

### 10. Dependency mati `sqlx` (Rendah — kebersihan)
- **Penyebab:** `go.mod` menambah `github.com/jmoiron/sqlx` tapi tidak ada satu file pun yang
  mengimpornya (dikonfirmasi `grep -rl` kosong di seluruh repo).
- **Perbaikan:** dihapus lewat `go mod tidy`.

### 11. Risiko terbuka — tabrakan ID lead virtual (didokumentasikan, belum diperbaiki)
- **Temuan:** `internal/lead/repository.go` — lead "virtual" (outlet yang belum punya row
  `customer_leads`) memakai `COALESCE(cl.id, ot.id)` sebagai id lead di response list. Karena
  `customer_leads.id` dan `outlets.id` adalah sequence auto-increment independen, id kecil bisa bertabrakan.
  Lookup (`FindLeadByID`/`lockLead`) memakai `WHERE (cl.id = ? OR ot.id = ?) ORDER BY CASE WHEN cl.id = ?
  THEN 0 ELSE 1 END LIMIT 1` — kalau lead id=7 (real, milik owner lain) DAN outlet id=7 (virtual, belum
  punya lead) sama-sama ada, query akan selalu mengembalikan lead #7 yang salah, bukan virtual lead outlet
  #7 yang dimaksud.
- **Keputusan:** tidak diperbaiki sprint ini. Perbaikan yang benar butuh skema penomoran ID baru (mis.
  namespace terpisah/negative-ID untuk virtual lead) yang mengubah kontrak `id` yang sudah dikonsumsi
  langsung oleh frontend (`app/menu/lead/page.tsx` membangun URL navigasi dari `lead.id` mentah) — perlu
  koordinasi desain lintas proyek, bukan perbaikan satu-file yang aman untuk dirush di sprint audit ini.
  Audit frontend mengonfirmasi eksposur saat ini rendah (id selalu diambil segar dari list yang baru
  difetch, tidak pernah dikonstruksi dari endpoint outlet lain) — risiko nyata tapi tidak akut.

---

## Rincian Perubahan — Frontend (`crm_piposmart`)

Frontend `main` sudah lebih dulu menggabungkan branch pasangannya sendiri (`satria`, `fix/owner-outlet`,
`feat/owner-outlet-mitra`) — ditulis terhadap backend versi **sebelum** perbaikan di atas. Audit
mengonfirmasi sebagian besar sudah kompatibel tanpa perubahan (lihat bagian "Kompatibel — Tidak Perlu
Perubahan"), dan menemukan 2 celah nyata:

### 1. Gating role tipe-partner tidak sinkron dengan 3 tingkat izin backend
- **Penyebab:** `app/menu/kelolaan-mitra/page.tsx` (`canManageTypes`) dan
  `app/menu/kelolaan-mitra/jenis-mitra/page.tsx` (`canManage`) menggabungkan SEMUA operasi tipe-partner +
  commission-rule ke satu flag `ADMIN || SUPERVISOR || ""`. Backend sebenarnya punya 3 tingkat berbeda:
  - Type CRUD (create/update/delete) → **ADMIN-only** (`canManagePartnerType`, diperketat di temuan #3
    di atas).
  - Commission-rule create → **ADMIN + SUPERVISOR** (`CreateCommissionRule`, tidak berubah).
  - Commission-rule deactivate → **ADMIN-only** (`DeactivateCommissionRule`, tidak berubah, ternyata sudah
    lebih ketat dari create sejak awal).

  Akibatnya SUPERVISOR (dan actor dengan role belum termuat, fallback `""`) masih melihat tombol
  create/edit/delete tipe partner yang sekarang akan gagal dengan 403 yang membingungkan.
- **Perbaikan:**
  - `kelolaan-mitra/page.tsx`: `canManageTypes` diperketat jadi `currentRole === "ADMIN"` saja (dipakai
    murni untuk type CRUD di file ini, dikonfirmasi tidak ada usage lain yang perlu tetap permisif).
  - `kelolaan-mitra/jenis-mitra/page.tsx`: `canManage` dipecah jadi 3 flag terpisah — `canManageTypes`
    (ADMIN-only, dipakai tombol create/edit type), `canCreateRules` (ADMIN+SUPERVISOR, dipakai tombol
    submit commission rule), `canDeactivateRules` (ADMIN-only, dipakai tombol nonaktifkan rule) — masing-
    masing dipetakan ke fungsi backend yang persis sesuai.

### 2. Export Excel Owner-Outlet tidak digate & pesan error generik
- **Penyebab:** `app/menu/owner-outlet/page.tsx` — tombol dropdown "Export Data" (memanggil
  `/owners/export/download` dan `/owners/export/download-owner`) tidak punya guard role sama sekali; menu
  "Owner" sendiri juga tidak difilter berdasarkan role di `layout.tsx`. Backend sekarang mengembalikan 403
  untuk non-ADMIN (temuan #2 di atas) — tanpa perubahan frontend, SALES/SUPERVISOR akan melihat tombol yang
  selalu gagal dengan alert generik "Terjadi kesalahan saat mengunduh data ekspor."
- **Perbaikan:**
  - Ditambah state `currentRole` (fetch via `getProfile()`, pola yang sama dipakai `kelolaan-mitra`) dan
    flag `canExportOwnerExcel = currentRole === "ADMIN"`.
  - Grup tombol "Export Data" hanya dirender jika `canExportOwnerExcel`.
  - `handleDownloadExcel` sekarang mendeteksi `res.status === 403` secara eksplisit dan menampilkan pesan
    "Anda tidak memiliki izin untuk mengunduh data ini." — sebagai fallback jika role belum termuat saat
    tombol sempat diklik.

### Kompatibel — Tidak Perlu Perubahan
- **Field `status` owner:** frontend SUDAH menyimpan `status` dan `subscription_status` sebagai dua field
  terpisah di tipe `BackendOwner` (`app/lib/api.ts`), dan halaman detail owner
  (`app/menu/owner-outlet/[id]/page.tsx`) sudah memakai `owner.status === "ACTIVE"` untuk badge hijau/merah
  — persis semantik lifecycle yang benar. Perbaikan backend di temuan #6 justru **memperbaiki** bug yang
  sebelumnya membuat badge ini salah tampil (`SUBSCRIBE`/`TRIAL` bukan `ACTIVE`), bukan meregresi.
- **`exportOwnerOutlets()` (JSON `/owners/export`):** didefinisikan di `api.ts` tapi tidak pernah dipanggil
  di mana pun — dead code, tidak terdampak.
- **`plan_id` vs `package_id`:** dikonfirmasi nol occurrence `package_id`/`packageId` tersisa di modul
  partner frontend — perbaikan sprint sebelumnya tidak ter-regresi oleh branch yang baru di-merge.
- **Bucket status analytics (7 bucket, termasuk `AKAN_JATUH_TEMPO`/`TELAH_JATUH_TEMPO`):** chart renderer
  (`Sprint14g1Board.tsx`) sudah generik, iterasi `result.series` secara dinamis tanpa hardcode daftar
  bucket — otomatis merender bucket baru begitu backend (temuan #4) mulai mengembalikan data. Filter
  dropdown outlet (`kelolaan-outlet/page.tsx`) bahkan sudah mencantumkan kedua bucket baru sebelum
  backend-nya berfungsi.
- **Commission rule `POST .../commission-rules`:** sudah digate ADMIN+SUPERVISOR di kedua halaman partner,
  tidak pernah SALES — cocok dengan backend, tidak berubah.

---

## Temuan yang Ditolak (Verifikasi Manual)

Beberapa klaim awal dari audit subagent ditolak setelah dibandingkan langsung dengan kode/baseline:
- **Sign `remaining_days` negatif di tier `WillBeDue`** (`classifyOutletSubscription`) — awalnya diklaim
  bug, ternyata konvensi asli yang sudah ada sejak sebelum merge, konsisten dengan test yang lolos. Bukan
  regresi.
- **SQL count-join di lead list** — awalnya diduga berisiko (pola bug klasik: COUNT pakai JOIN beda dari
  SELECT utama), dikonfirmasi COUNT dan SELECT memakai JOIN yang identik — pagination sudah benar.
- **Migration ordering/conflict** — dikonfirmasi tidak ada migration baru sama sekali di window merge ini
  (`git diff --name-status -- migrations/` kosong), sehingga risiko conflict tidak relevan.

---

## Verifikasi & Kualitas Kode

- **Backend:** `go build $(go list ./... | grep -v cmd/run_subs)` bersih; `go vet` bersih; `go test ./...`
  — seluruh package `ok` (kecuali `cmd/run_subs`, sudah rusak sejak sebelum sprint ini karena mengimpor
  package `pkg/mysql` yang tidak pernah ada di repo — tidak tersentuh perubahan sprint ini, dicatat sebagai
  temuan terpisah, bukan bagian dari cakupan perbaikan).
- **Frontend:** `npx tsc --noEmit` bersih; `npm run build` sukses penuh, 58 route ter-generate tanpa error.
- **Live smoke test** — backend dijalankan sungguhan (`go run . api`) terhadap database dev lokal
  (`127.0.0.1:3306`, data existing, bukan seed baru), login sungguhan sebagai user demo (`admin.001@demo
  .piposmart.id` / `sales.001@demo.piposmart.id`, password deterministik seeder `Password123!`):
  1. `POST /analytics/outlets/subscription-status-recap/query` — `200`, 7 series bucket data asli.
  2. `GET /reports/admin_owner` & `GET /reports/admin_owner_outlet` — `200`, baris data asli.
  3. `GET /owners/export` — `200`, data ter-scope sesuai actor.
  4. `GET /owners` — `status` dan `subscription_status` dua field independen dengan nilai benar.
  5. `DELETE /partner-types/{id}` sebagai SALES — `403 FORBIDDEN`.
  6. `GET /owners/export/download-owner` sebagai SALES — `403`; sebagai ADMIN — `200` + file `.xlsx` valid.
  7. Server dimatikan bersih setelah smoke test selesai; tidak ada seed/migrate destruktif dijalankan
     terhadap database dev bersama.
