# Sprint 4 - Owner dan Outlet

## Sprint

Sprint 4

## Periode

23 Juli 2026

## Status

`AMBER`

Sprint Goal tercapai untuk CRUD Owner dan Outlet, pencarian, normalisasi nomor
telepon, pagination, sorting, soft delete, restore, hard delete, bulk operation,
Swagger/OpenAPI, dan smoke test API.
Status belum `GREEN` karena Docker runtime dan integration test database
otomatis belum tersedia.

## Sprint Goal

Data owner laundry dan satu atau lebih outlet dapat dikelola melalui API
terproteksi.

## Completed

- [x] CRUD Owner:
  - `GET /api/v1/owners`
  - `POST /api/v1/owners`
  - `GET /api/v1/owners/{owner_id}`
  - `PATCH /api/v1/owners/{owner_id}`
  - `DELETE /api/v1/owners/{owner_id}`
- [x] CRUD Outlet per Owner:
  - `GET /api/v1/owners/{owner_id}/outlets`
  - `POST /api/v1/owners/{owner_id}/outlets`
  - `GET /api/v1/owners/{owner_id}/outlets/{outlet_id}`
  - `PATCH /api/v1/owners/{owner_id}/outlets/{outlet_id}`
  - `DELETE /api/v1/owners/{owner_id}/outlets/{outlet_id}`
- [x] Search owner berdasarkan kode, nama, telepon, brand, kota, dan provinsi.
- [x] Search outlet berdasarkan kode, nama, telepon, kota, dan provinsi.
- [x] Normalisasi nomor telepon ke format `62...`.
- [x] Pagination `page` dan `limit`.
- [x] Sorting whitelist, termasuk prefix `-` untuk descending.
- [x] Soft delete owner dan outlet.
- [x] Restore owner dan outlet:
  - `PATCH /api/v1/owners/{owner_id}/restore`
  - `PATCH /api/v1/owners/{owner_id}/outlets/{outlet_id}/restore`
- [x] Hard delete permanen owner dan outlet:
  - `DELETE /api/v1/owners/{owner_id}/force`
  - `DELETE /api/v1/owners/{owner_id}/outlets/{outlet_id}/force`
- [x] Bulk operation owner:
  - `POST /api/v1/owners/bulk`
  - `PATCH /api/v1/owners/bulk`
  - `DELETE /api/v1/owners/bulk`
  - `DELETE /api/v1/owners/bulk/force`
- [x] Bulk operation outlet:
  - `POST /api/v1/owners/{owner_id}/outlets/bulk`
  - `PATCH /api/v1/owners/{owner_id}/outlets/bulk`
  - `DELETE /api/v1/owners/{owner_id}/outlets/bulk`
  - `DELETE /api/v1/owners/{owner_id}/outlets/bulk/force`
- [x] Owner soft delete tidak melakukan cascade delete ke outlet.
- [x] Owner hard delete membuat outlet child menjadi orphan dengan `owner_id = NULL`.
- [x] Response default tidak memunculkan data soft-deleted.
- [x] Query list owner menghitung `outlet_count` dengan join agregat, tanpa N+1.
- [x] Endpoint Owner/Outlet diproteksi permission `owners.manage`.
- [x] Admin dan Supervisor dapat mengelola Owner/Outlet.
- [x] Sales ditolak mengakses Owner/Outlet.
- [x] OpenAPI diperbarui ke `0.4.0-sprint-4`.
- [x] Migration `20260723000300_owner_outlet_orphan_delete_policy.sql`
  ditambahkan untuk mendukung orphan behavior outlet.
- [x] OwnerFactory dan OutletFactory sudah tersedia dan dipakai demo seeder.
- [x] Demo seed tetap menyediakan owner satu outlet dan multi-outlet.

## Not Completed / Carry Over

- Item: Integration test DB otomatis untuk Owner/Outlet.
- Penyebab: Belum ada test database lifecycle khusus.
- Estimasi ulang: 1 hari saat environment test DB tersedia.

- Item: Docker runtime verification.
- Penyebab: Docker CLI belum tersedia pada workstation.
- Estimasi ulang: 0.5 hari saat Docker tersedia.

## Demo Evidence

Command:

```powershell
go run . migrate up
go run . api
```

Smoke test API:

- `GET /api/v1/owners` sebagai Admin berhasil.
- `POST /api/v1/owners` berhasil membuat owner dengan phone ternormalisasi.
- Search owner berdasarkan `q` dan `city` berhasil.
- `PATCH /api/v1/owners/{owner_id}` berhasil.
- Duplicate owner code mengembalikan `409`.
- Invalid phone mengembalikan `400`.
- `POST /api/v1/owners/{owner_id}/outlets` berhasil membuat outlet.
- List/detail/update outlet berhasil.
- Outlet untuk owner tidak ada mengembalikan `404`.
- Soft delete outlet membuat detail outlet mengembalikan `404`.
- Restore outlet mengembalikan outlet menjadi `ACTIVE`.
- Soft delete owner membuat detail owner mengembalikan `404`, tetapi outlet
  child tidak ikut soft-deleted.
- Restore owner membuat detail owner kembali dapat diakses.
- Bulk create/update owner berhasil.
- Bulk create/update outlet berhasil.
- Bulk force delete owner/outlet berhasil.
- Sales mengakses `/owners` mengembalikan `403`.
- Sort invalid mengembalikan `400`.

## Keputusan Domain Baru

- Semua entitas CRM yang memiliki `deleted_at` memakai pola:
  - restore single: `PATCH /api/v1/{entities}/{id}/restore`
  - hard delete single: `DELETE /api/v1/{entities}/{id}/force`
  - bulk create: `POST /api/v1/{entities}/bulk`
  - bulk update: `PATCH /api/v1/{entities}/bulk`
  - bulk soft delete: `DELETE /api/v1/{entities}/bulk`
  - bulk hard delete: `DELETE /api/v1/{entities}/bulk/force`
- Soft delete parent tidak menghapus child.
- Hard delete parent tidak menghapus child; FK child menjadi `NULL` jika schema
  mendukung.
- Customer/owner memiliki konsep kepemilikan operasional:
  - pertama dibuat Admin dan menjadi data milik Admin;
  - Admin dapat membagikan ke Supervisor;
  - Supervisor dapat membagikan data miliknya ke Sales;
  - Sales hanya melihat data miliknya;
  - remark `0` dari Sales mengembalikan customer ke Supervisor dengan status
    tidak potensial/invalid;
  - saat Supervisor membagikan ulang ke Sales lain, skor dapat di-reset menjadi
    `1` kemungkinan potensial;
  - seluruh perpindahan kepemilikan customer wajib tercatat pada assignment
    history dan dapat dilihat Admin pada detail customer.

## Quality

- Unit test: `go test ./...` lulus.
- Static analysis: `go vet ./...` lulus.
- Build: `go build .` lulus.
- Smoke test API manual utama: 18/18 PASS.
- Smoke test tambahan restore/force/bulk: 12/12 PASS.
- Docker build/runtime: belum diverifikasi lokal.
- Defect terbuka: tidak ada defect kode yang diketahui.

## Impediments

- Docker Engine/CLI belum tersedia.
- Integration test DB otomatis belum tersedia.

## Risiko Baru

- Risiko: Hard delete parent menghasilkan orphan data pada child.
- Dampak: Frontend perlu menampilkan parent yang tidak tersedia dengan pesan
  ramah, bukan menganggap data child rusak.
- Mitigasi: Response detail child pada domain berikutnya perlu menyediakan
  fallback label seperti `data owner tidak tersedia` saat parent sudah hard
  deleted.
- Owner: Product Owner dan Frontend Engineer.

- Risiko: Validation error Gin masih menampilkan nama struct Go.
- Dampak: Pesan error validasi kurang ramah untuk end user.
- Mitigasi: Tambahkan validation translator saat hardening API response.
- Owner: Backend Engineer.

## Keputusan yang Dibutuhkan

- Apakah Sales pada sprint mendatang perlu read-only access owner/outlet milik
  lead yang sedang dia handle.
- Model kepemilikan customer Admin -> Supervisor -> Sales dan tracking
  perpindahannya dimasukkan ke Sprint 5 sebagai bagian dari Lead Assignment
  History.

## Rencana Sprint Berikutnya

- Sprint 5: Lead dan Assignment Sales.
- Implement customer lead.
- Implement single dan bulk assignment.
- Implement release dan redistribution.
- Implement constraint satu assignment aktif per lead.
- Implement filter unassigned, assigned, stage, Sales, dan follow-up.
