Sprint: 14f - Full List Routes `/all`, Identity Route, dan OpenAPI Update  
Periode: 29 Juli 2026  
Status: GREEN

Sprint Goal:
- Menambahkan route tambahan `/all` dan `/all-deleted` pada modul-modul list utama tanpa mengganggu route lama.
- Menambahkan route manajemen akun Admin/Supervisor dan reset password lintas role.
- Memperbarui OpenAPI agar frontend memiliki dokumentasi resmi untuk route baru tersebut.

Committed Deliverables:
- Route `/all` pada modul list utama.
- Route `/all-deleted` pada modul yang memakai soft delete, serta alias kompatibilitas pada modul lain.
- Reuse filter query yang sama dengan endpoint GET lama.
- Route create Admin dan create Supervisor.
- Route reset password Admin, Supervisor, dan Sales.
- OpenAPI diperbarui untuk route baru.
- Dokumentasi Sprint 14f.

Completed:
- Menambahkan pola route full list tanpa pagination pada domain:
  - owner
  - outlet
  - outlet subscription statuses
  - catalog packages/plans/promotions
  - leads
  - customer interactions/follow-ups/trainings
  - closings
  - wallets/payments/transactions
  - subscription orders/subscriptions/reconciliations/issues
  - imports/import rows
  - sales targets
  - partner list/interactions/commissions/payouts
- Menjaga route lama tetap utuh agar frontend existing tidak rusak.
- Menjaga response shape lama tetap kompatibel, termasuk metadata pagination/list meta.
- Menambahkan flag internal/service/repository agar query list bisa berjalan tanpa `LIMIT/OFFSET` saat route `/all` dipakai.
- Menambahkan dokumentasi OpenAPI untuk route baru `/all` dan `/all-deleted`.
- Menambahkan route identity baru:
  - `POST /api/v1/admins`
  - `POST /api/v1/admins/{id}/reset-password`
  - `POST /api/v1/supervisors`
  - `POST /api/v1/supervisors/{id}/reset-password`
- Mempertahankan route reset password Sales:
  - `POST /api/v1/sales/{id}/reset-password`
- Menjaga boundary permission:
  - Supervisor tetap hanya mengelola Sales
  - Admin dapat membuat Admin/Supervisor dan reset password semua role manajemen
- Menambahkan penjelasan pada OpenAPI untuk membedakan:
  - route yang benar-benar mengembalikan data soft deleted,
  - dan route alias kompatibilitas frontend pada modul yang belum memiliki soft delete.
- Memperbarui versi OpenAPI menjadi `0.15.4-sprint-14f-identity`.
- Menambahkan dokumentasi:
  - `docs/sprint-14f/README.md`
  - `docs/sprint-14f/sprint-14f.md`
  - `docs/sprint-14f/api-testing.md`

Not Completed / Carry Over:
- Item: sinkronisasi seluruh route baru `/all` ke README utama project.
- Penyebab: fokus Sprint 14f dibatasi pada implementasi backend + OpenAPI + dokumentasi sprint.
- Estimasi ulang: 20-30 menit jika ingin dibuat dokumentasi route-by-route tambahan di README project.

Demo Evidence:
- Endpoint/Swagger:
  - route `/all` dan `/all-deleted` pada OpenAPI file `internal/platform/httpserver/openapi.yaml`
  - route identity:
    - `POST /api/v1/admins`
    - `POST /api/v1/admins/{id}/reset-password`
    - `GET /api/v1/supervisors`
    - `POST /api/v1/supervisors`
    - `POST /api/v1/supervisors/{id}/reset-password`
    - `POST /api/v1/sales/{id}/reset-password`
- Command verifikasi:
  - `go build ./...`
- Dokumentasi:
  - `docs/sprint-14f/README.md`
  - `docs/sprint-14f/api-testing.md`

Quality:
- Unit/integration test:
  - `go test ./internal/identity/...` PASS
  - `go test ./internal/platform/httpserver/...` PASS
  - ditambahkan dokumentasi manual API testing untuk route Sprint 14f
- Migration status:
  - tidak ada migration baru.
- Docker build:
  - tidak ada perubahan Dockerfile/compose pada Sprint 14f.
- Defect terbuka:
  - tidak ada defect compile/runtime yang teridentifikasi dari perubahan ini.

Impediments:
- OpenAPI saat ini dikelola manual dalam satu file YAML besar, sehingga penambahan banyak path membutuhkan kehati-hatian lebih tinggi saat maintenance.

Risiko Baru:
- Risiko: frontend memakai route `/all` untuk dataset sangat besar pada halaman yang sebenarnya cukup memakai pagination.
- Dampak: payload response bisa besar, memperlambat render UI dan transfer network.
- Mitigasi: dokumentasi Sprint 14f menegaskan route `/all` hanya dipakai untuk kebutuhan bulk/full preload, bukan default tabel biasa.
- Owner: Frontend Engineer / System Analyst

- Risiko: frontend atau QA mencoba route create/reset Admin-Supervisor menggunakan akun non-Admin.
- Dampak: akan selalu menerima `403 FORBIDDEN` dan disangka bug.
- Mitigasi: dokumentasi Sprint 14f menegaskan boundary otorisasi role untuk route identity baru.
- Owner: Backend Engineer / Frontend Engineer

Keputusan yang Dibutuhkan:
- Apakah pada sprint berikutnya perlu ditambahkan guard atau batas maksimum optional untuk route `/all` pada modul tertentu yang berpotensi sangat besar.

Rencana Sprint Berikutnya:
- Jika diperlukan, lanjutkan ke dokumentasi route baru pada README utama project.
- Review bersama frontend modul mana yang sebaiknya tetap memakai pagination dan mana yang layak memakai `/all`.
- Lanjut ke backlog Sprint 15 setelah kebutuhan frontend terkait full list routes dianggap cukup.
