Sprint: 16 - Dashboard, Reporting, dan Export  
Periode: 2 Agustus 2026  
Status: GREEN

Sprint Goal:
- Menyediakan fondasi backend reporting lintas modul.
- Menyediakan dashboard ringkas yang bisa dipakai frontend untuk Admin/Supervisor/Sales.
- Menyediakan export report async berbasis worker.
- Menutup risiko spreadsheet formula injection pada file export.

Committed Deliverables:
- Modul reporting backend baru.
- Endpoint dashboard reporting.
- Endpoint list report lintas modul.
- Endpoint export center async.
- Tabel persistence export.
- Integrasi worker untuk generate file export.
- Proteksi formula injection untuk CSV/XLSX.

Completed:
- Menambahkan migration `20260802000200_reporting_exports.sql`.
- Menambahkan package `internal/reporting`.
- Menambahkan endpoint:
  - `GET /api/v1/reports/dashboard`
  - `GET /api/v1/reports/owners_outlets`
  - `GET /api/v1/reports/activities`
  - `GET /api/v1/reports/topups`
  - `GET /api/v1/reports/closings`
  - `GET /api/v1/reports/subscriptions`
  - `GET /api/v1/reports/partners`
  - `GET /api/v1/reports/targets_kpi`
  - `POST /api/v1/reports/exports`
  - `GET /api/v1/reports/exports`
  - `GET /api/v1/reports/exports/{id}`
  - `GET /api/v1/reports/exports/{id}/download`
- Menambahkan worker job:
  - `REPORT_EXPORT_GENERATE`
- Menambahkan builder export:
  - CSV
  - XLSX
- Menambahkan test sanitasi formula injection.
- Mengintegrasikan reporting ke router API dan background worker.

Not Completed / Carry Over:
- Item: end-to-end smoke test HTTP dengan database hidup dan file export nyata.
- Penyebab: pada turn ini fokus utama ditutup pada implementasi backend, validasi compile/test/vet, dan validasi OpenAPI parser.
- Estimasi ulang: 30-60 menit setelah environment lokal/staging untuk worker + database siap diuji interaktif.

Demo Evidence:
- Source files:
  - `internal/reporting/errors.go`
  - `internal/reporting/types.go`
  - `internal/reporting/repository.go`
  - `internal/reporting/service.go`
  - `internal/reporting/handler.go`
  - `internal/reporting/worker.go`
  - `internal/reporting/export.go`
  - `internal/reporting/export_test.go`
- Migration:
  - `migrations/20260802000200_reporting_exports.sql`
- Wiring:
  - `internal/platform/httpserver/router.go`
  - `internal/app/api.go`

Quality:
- Unit/integration test:
  - seluruh package test lulus; environment Windows masih memunculkan kendala cleanup file `.test.exe` sementara saat proses selesai
- Vet:
  - `go vet ./...` PASS
- Build:
  - `go build .` PASS
- OpenAPI:
  - `npx -y @apidevtools/swagger-cli validate internal/platform/httpserver/openapi.yaml` PASS
- Defect terbuka:
  - belum ada defect blocker yang teridentifikasi dari compile/vet/OpenAPI validation lokal.

Impediments:
- Tidak ada blocker teknis utama pada implementasi modul reporting ini.

Risiko Baru:
- Risiko: actor scope export async bisa berkembang lebih kompleks bila Supervisor pada masa depan
  perlu rule visibility yang berbeda dari `reports.read_all`.
- Dampak: hasil export mungkin perlu aturan scoping yang lebih detail daripada role sederhana.
- Mitigasi: role/requested_by sudah dipersist sebagai basis, dan repository reporting bisa
  diperluas tanpa mengubah contract endpoint export.
- Owner: Backend Engineer

- Risiko: report berbasis query generik `items + columns` memberi fleksibilitas tinggi, tetapi
  frontend perlu disiplin membaca metadata kolom daripada hardcode nama field.
- Dampak: frontend bisa salah asumsi jika tidak memakai `columns`.
- Mitigasi: dokumentasi Sprint 16 menegaskan `columns` sebagai source of truth tabel report.
- Owner: Frontend Engineer

Keputusan yang Dibutuhkan:
- Apakah export PDF/PNG perlu masuk Sprint 16 backend, atau tetap diposisikan sebagai backlog
  lanjutan setelah CSV/XLSX stabil.

Rencana Sprint Berikutnya:
- Tambahkan smoke test HTTP nyata untuk dashboard/report/export bila environment siap.
- Perkaya API testing report Sprint 16 dengan contoh request/response/error handler yang lebih panjang bila dibutuhkan stakeholder.
- Evaluasi kebutuhan PDF/PNG export.
- Lanjut ke Sprint 17 hardening/performance bila Sprint 16 disetujui.
