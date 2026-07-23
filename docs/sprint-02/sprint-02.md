# Sprint 2 - Baseline Migration, Factory, dan Seeder

## Sprint

Sprint 2

## Periode

23 Juli 2026

## Status

`AMBER`

Sprint Goal tercapai untuk baseline schema, factory, seeder, dan command lokal.
Status belum `GREEN` karena runtime Docker tidak dapat diuji pada workstation
ini dan data harga resmi dari PDF belum dapat diekstrak/validasi otomatis.

## Sprint Goal

Database kosong dapat dibentuk dan diisi data awal yang deterministik untuk
demo Sprint berikutnya.

## Committed Deliverables

- Mengganti migration prototipe dengan baseline Goose baru.
- Schema identity, reference data, customer dasar, dan catalog dasar.
- Framework factory `Build` dan `Create`.
- Random generator deterministik.
- Tabel `seed_runs`.
- Master seeder untuk role, permission, remark reason, partner type, metric
  code, package, plan, dan promo awal.
- Demo seeder preset `minimal`.
- Command:
  - `go run . migrate up`
  - `go run . migrate down`
  - `go run . seed master`
  - `go run . seed demo --preset=minimal --seed=20260723 --as-of=2026-07-01`

## Completed

- [x] Migration lama prototype dihapus dari folder `migrations`.
- [x] Baseline migration baru dibuat:
  `migrations/20260723000100_baseline_crm_schema.sql`.
- [x] Schema awal mencakup:
  - `roles`
  - `permissions`
  - `role_permissions`
  - `users`
  - `audit_logs`
  - `seed_runs`
  - `remark_reasons`
  - `partner_types`
  - `metric_codes`
  - `subscription_packages`
  - `subscription_plans`
  - `promotions`
  - `promotion_plan_eligibilities`
  - `promotion_benefits`
  - `owners`
  - `outlets`
  - `customer_leads`
- [x] Seeder master dibuat dan bersifat idempotent.
- [x] Seeder demo preset `minimal` dibuat.
- [x] Demo seeder ditolak saat `APP_ENV=production`.
- [x] Password dummy user di-hash dengan Argon2id.
- [x] Factory deterministic untuk user, owner, outlet, dan lead.
- [x] CLI `seed` ditambahkan ke executable CRM.
- [x] Service Compose `seed` opsional ditambahkan dengan profile `seed`.
- [x] `.env.example` disesuaikan agar command lokal memakai path lokal.
- [x] Entrypoint dipindahkan ke root `main.go` agar command lokal lebih pendek:
  `go run . ...`.
- [x] Folder legacy prototype root dibersihkan:
  `controllers`, `database`, `models`, `repositories`, `requests`,
  `responses`, `routes`, `seeders`, dan `services`.
- [x] README disesuaikan dengan command baru dan struktur project baru.
- [x] OpenAPI version dinaikkan ke `0.2.0-sprint-2`.

## Not Completed / Carry Over

- Item: Verifikasi Docker runtime.
- Penyebab: Docker CLI tidak tersedia pada workstation implementasi.
- Estimasi ulang: 0.5 hari saat Docker tersedia.

- Item: Validasi nominal harga/promo resmi dari file PDF.
- Penyebab: Tool ekstraksi PDF tidak tersedia pada workstation implementasi.
- Estimasi ulang: dibawa ke Sprint 7 saat catalog package/promo menjadi fokus
  utama dan stakeholder memvalidasi daftar harga.

## Demo Evidence

Endpoint/Swagger:

```text
GET /health/live
GET /health/ready
GET /api/v1/status
GET /swagger/index.html
```

Command:

```powershell
go run . migrate up
go run . seed master
go run . seed demo --preset=minimal --seed=20260723 --as-of=2026-07-01
go run . help
```

Skenario seed:

- Master data role: Admin, Supervisor, Sales.
- Permission matrix awal untuk Admin, Supervisor, dan Sales.
- Remark reason skor 0 sampai 3.
- Partner type awal.
- Metric code awal untuk KPI/reporting.
- Paket Basic, Business, dan Pro.
- Plan tenor 1, 9, 12, 18, dan 24 bulan dengan durasi `tenure x 30 hari`.
- Promo gratis dan promo berbayar awal.
- Demo user Admin, Supervisor, dan dua Sales.
- Demo owner satu outlet dan multi-outlet.
- Demo lead yang sudah memiliki PIC Sales aktif.

## Quality

- Unit/integration test: `go test ./...` lulus.
- Static analysis: `go vet ./...` lulus.
- Build: `go build .` lulus.
- Migration status: `go run . migrate up` berhasil pada database lokal.
- Docker build: belum diverifikasi lokal karena Docker CLI tidak tersedia.
- Defect terbuka: tidak ada defect kode yang diketahui.

## Impediments

- Docker Engine/CLI tidak tersedia pada workstation.
- PDF harga belum dapat diekstrak otomatis.

## Risiko Baru

- Risiko: Harga seed awal belum angka final resmi.
- Dampak: Demo catalog dapat berbeda dari harga bisnis sebenarnya.
- Mitigasi: Perlakukan harga Sprint 2 sebagai placeholder dan validasi daftar
  harga pada Sprint 7.
- Owner: Product Owner / stakeholder marketing.

- Risiko: Fresh baseline menghapus migration prototype.
- Dampak: Database prototype lama tidak bisa dimigrasi incremental.
- Mitigasi: Keputusan fresh schema sudah dikunci; gunakan database kosong untuk
  development baru.
- Owner: Backend Engineer.

## Keputusan yang Dibutuhkan

- Konfirmasi daftar harga dan promo resmi sebelum Sprint 7.
- Konfirmasi apakah Docker akan diuji di mesin developer lain atau di CI saja.

## Rencana Sprint Berikutnya

- Sprint 3: Authentication, RBAC, dan Sales Management.
- Implement login, refresh, logout, me, dan change password.
- Implement auth session dan refresh token rotation.
- Implement middleware authentication dan permission.
- Implement bootstrap Admin.
- Implement CRUD Sales oleh Admin dan Supervisor.
- Tambahkan factory/seeder untuk user Admin, Supervisor, dan Sales sesuai flow
  Sprint 3.
