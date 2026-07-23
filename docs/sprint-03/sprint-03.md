# Sprint 3 - Authentication, RBAC, dan Sales Management

## Sprint

Sprint 3

## Periode

23 Juli 2026

## Status

`AMBER`

Sprint Goal tercapai untuk auth, RBAC dasar, bootstrap Admin, dan Sales
Management. Status belum `GREEN` karena Docker runtime belum tersedia pada
workstation dan integration test DB masih berupa smoke test lokal manual.

## Sprint Goal

Pengguna dapat login dan akses API dibatasi berdasarkan peran. Admin dan
Supervisor dapat membuat serta mengelola akun Sales, sementara Sales tidak
dapat mengakses data user global.

## Completed

- [x] Migration `auth_sessions` untuk rotating refresh token.
- [x] Kolom user Sprint 3:
  - `password_changed_at`
  - `deactivated_at`
- [x] Config JWT, refresh token TTL, cookie auth, dan bootstrap Admin.
- [x] JWT access token HS256.
- [x] Refresh token random, disimpan sebagai SHA-256 hash.
- [x] Refresh token rotation.
- [x] Logout/revoke session.
- [x] Middleware authentication Bearer token.
- [x] Middleware/check permission berbasis role permission.
- [x] Endpoint auth:
  - `POST /api/v1/auth/login`
  - `POST /api/v1/auth/refresh`
  - `POST /api/v1/auth/logout`
  - `GET /api/v1/auth/me`
  - `POST /api/v1/auth/change-password`
- [x] Command `go run . bootstrap-admin`.
- [x] CRUD dasar Sales:
  - `GET /api/v1/sales`
  - `POST /api/v1/sales`
  - `GET /api/v1/sales/{id}`
  - `PATCH /api/v1/sales/{id}`
- [x] Activation, deactivation, dan reset password Sales.
- [x] Admin dan Supervisor dapat membuat Sales melalui permission
  `users.manage_sales`.
- [x] Jalur API Sales selalu membuat role `SALES`, sehingga Supervisor tidak
  memiliki endpoint untuk membuat Admin/Supervisor.
- [x] Audit log untuk login, logout, change password, create/update Sales,
  activate/deactivate Sales, dan reset password.
- [x] OpenAPI diperbarui ke `0.3.0-sprint-3`.
- [x] README diperbarui dengan flow bootstrap, login, dan create Sales.

## Not Completed / Carry Over

- Item: Integration test DB otomatis untuk login, refresh rotation, dan Sales
  management.
- Penyebab: Belum ada test database lifecycle khusus.
- Estimasi ulang: 1 hari, bisa digabung saat test container MySQL tersedia.

- Item: Docker runtime verification.
- Penyebab: Docker CLI belum tersedia pada workstation.
- Estimasi ulang: 0.5 hari saat Docker tersedia.

## Demo Evidence

Command:

```powershell
go run . migrate up
go run . seed master
go run . bootstrap-admin
go run . api
```

Smoke test lokal:

- `go run . migrate up` berhasil sampai version `20260723000200`.
- `go run . seed master` berhasil.
- `go run . bootstrap-admin` berhasil membuat/memastikan Admin
  `admin@piposmart.id`.
- Login Admin berhasil.
- `GET /api/v1/auth/me` berhasil mengembalikan role `ADMIN` dan permission.
- `POST /api/v1/sales` berhasil membuat Sales baru dan mengembalikan temporary
  password.

## Quality

- Unit test: `go test ./...` lulus.
- Static analysis: `go vet ./...` lulus.
- Build: `go build .` lulus.
- Migration: `go run . migrate up` lulus pada database lokal.
- Docker build/runtime: belum diverifikasi lokal.
- Defect terbuka: tidak ada defect kode yang diketahui.

## Impediments

- Docker Engine/CLI belum tersedia.
- Integration test DB otomatis belum tersedia.

## Risiko Baru

- Risiko: Auth sudah berjalan, tetapi integration test DB belum otomatis.
- Dampak: Regression pada flow login/refresh/Sales bisa terlambat diketahui.
- Mitigasi: Tambahkan test DB lifecycle saat environment Docker/CI MySQL siap.
- Owner: Backend Engineer.

- Risiko: Temporary password dikembalikan di response create/reset Sales.
- Dampak: Frontend harus menampilkan hanya sekali dan tidak mencatatnya di log.
- Mitigasi: Kontrak frontend harus memperlakukan field ini sebagai one-time
  secret.
- Owner: Backend Engineer dan Frontend Engineer.

## Keputusan yang Dibutuhkan

- Tentukan apakah refresh token akan dikirim via body saja atau juga cookie
  HttpOnly saat frontend auth mulai diintegrasikan.
- Tentukan kebijakan password final untuk production.

## Rencana Sprint Berikutnya

- Sprint 4: Owner dan Outlet.
- Implement CRUD owner.
- Implement CRUD outlet per owner.
- Implement pagination, filter, sorting, dan soft delete.
- Tambahkan OwnerFactory dan OutletFactory sesuai endpoint baru.
