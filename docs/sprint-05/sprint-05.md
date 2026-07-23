# Sprint 5 - Lead dan Assignment Sales

## Sprint

Sprint 5

## Periode

23 Juli 2026

## Status

`AMBER`

Sprint Goal tercapai untuk vertical slice ownership dan assignment lead sesuai
briefing terbaru. Status belum `GREEN` karena integration test database otomatis
dan verifikasi Docker runtime dari environment Codex belum tersedia.

## Sprint Goal

Data owner/customer dapat dibagikan dari Admin ke Supervisor, dari Supervisor ke
Sales, dan dapat kembali ke Supervisor saat Sales menandai customer invalid.
Seluruh perpindahan ownership terekam pada assignment history.

## Completed

- [x] Migration `20260723000400_lead_ownership_assignment.sql`.
- [x] Customer lead ownership pada tabel `customer_leads`:
  - `current_owner_user_id`
  - `current_owner_role`
  - `supervisor_id`
  - `current_score`
  - `invalidated_at`
  - `invalidated_by_sales_id`
- [x] Tabel `lead_assignments` untuk tracking perpindahan ownership.
- [x] Constraint satu active assignment per lead melalui generated unique key.
- [x] Owner yang dibuat Admin otomatis dibuatkan customer lead milik Admin.
- [x] Role-scoped visibility:
  - Admin melihat semua owner/lead.
  - Supervisor melihat owner/lead miliknya dan Sales di bawahnya.
  - Sales hanya melihat owner/lead miliknya.
- [x] Endpoint Lead:
  - `GET /api/v1/leads`
  - `POST /api/v1/leads`
  - `GET /api/v1/leads/{lead_id}`
  - `GET /api/v1/leads/{lead_id}/assignment-history`
  - `POST /api/v1/leads/{lead_id}/assign-supervisor`
  - `POST /api/v1/leads/{lead_id}/assign-sales`
  - `POST /api/v1/leads/{lead_id}/release`
  - `POST /api/v1/leads/{lead_id}/mark-invalid`
  - `POST /api/v1/leads/bulk/assign-supervisor`
  - `POST /api/v1/leads/bulk/assign-sales`
  - `POST /api/v1/leads/bulk/release`
- [x] Filter lead berdasarkan `q`, `ownership`, `stage`, `status`, `score`,
  `supervisor_id`, `sales_id`, `follow_up_from`, dan `follow_up_to`.
- [x] Factory/seeder demo diperbarui agar lead memiliki current ownership dan
  active assignment history.
- [x] OpenAPI diperbarui ke `0.5.0-sprint-5`.

## Demo Evidence

Smoke test API tambahan: 11/11 PASS.

- Admin membuat owner dan lead otomatis terbentuk.
- Supervisor tidak dapat melihat owner yang masih milik Admin.
- Admin assign lead ke Supervisor.
- Supervisor dapat melihat owner yang sudah menjadi miliknya.
- Supervisor assign lead ke Sales.
- Sales yang ditugaskan dapat melihat owner tersebut.
- Sales lain tidak dapat melihat owner/lead tersebut.
- Sales menandai lead invalid.
- Lead kembali ke Supervisor dengan `current_score = 0`.
- Sales kehilangan visibility setelah invalid.
- Assignment history mencatat perpindahan ownership.

## Quality

- Migration up: PASS.
- `go test ./...`: PASS.
- `go vet ./...`: PASS.
- `go build .`: PASS.
- Smoke test API manual: PASS.

## Impediments

- Docker CLI belum tersedia pada environment shell Codex, meskipun user sudah
  menginstall Docker di workstation.
- Integration test DB otomatis belum tersedia.

## Risiko Baru

- Risiko: Lead assignment kini menjadi sumber kebenaran visibility owner.
- Dampak: Endpoint lama yang membaca owner harus selalu melewati visibility
  policy agar Sales tidak melihat data Sales lain.
- Mitigasi: Tambahkan integration test role visibility pada hardening Sprint 5/6.
- Owner: Backend Engineer.

## Rencana Sprint Berikutnya

- Sprint 6: Call Customer, Remark, Follow-up, dan Training.
- Implement remark score 0-3 dari call customer.
- Integrasikan remark `0` dengan endpoint invalid yang sudah ada.
- Tambahkan follow-up schedule dan stage history.
