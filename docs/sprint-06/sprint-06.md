# Sprint 6 - Call Customer, Remark, Follow-up, dan Training

## Sprint

Sprint 6

## Periode

23 Juli 2026

## Status

`AMBER`

Sprint Goal tercapai untuk vertical slice call/chat customer, remark, follow-up,
stage history, dan training. Status belum `GREEN` karena integration test DB
otomatis belum lengkap dan rule remark 3 baru disiapkan sebagai stage
`CLOSING` sementara sampai modul closing Sprint 8.

## Sprint Goal

Sales dapat mencatat aktivitas follow-up customer tanpa kehilangan histori.
Setiap remark dapat memperbarui stage lead sesuai policy, dan training/demo
dapat dijadwalkan, di-reschedule, diselesaikan, atau dibatalkan.

## Completed

- [x] Migration `20260723000500_customer_interactions_training.sql`.
- [x] Tabel `customer_interactions` untuk call/chat append-only.
- [x] Tabel `lead_stage_histories` untuk riwayat perubahan stage/score.
- [x] Tabel `training_reports` untuk jadwal dan laporan training.
- [x] Endpoint activity:
  - `GET /api/v1/customer-interactions`
  - `GET /api/v1/follow-ups`
  - `GET /api/v1/leads/{lead_id}/interactions`
  - `POST /api/v1/leads/{lead_id}/interactions`
  - `GET /api/v1/leads/{lead_id}/stage-history`
- [x] Endpoint training:
  - `GET /api/v1/trainings`
  - `GET /api/v1/trainings/{training_id}`
  - `GET /api/v1/leads/{lead_id}/trainings`
  - `POST /api/v1/leads/{lead_id}/trainings`
  - `POST /api/v1/trainings/{training_id}/reschedule`
  - `POST /api/v1/trainings/{training_id}/complete`
  - `POST /api/v1/trainings/{training_id}/cancel`
- [x] Remark policy:
  - `0` menjadi `INVALID` dan ownership kembali ke Supervisor.
  - `1` menjadi `POSSIBLE`.
  - `1` tidak menurunkan `POTENTIAL`.
  - `2` menjadi `POTENTIAL`.
  - `3` menjadi `CLOSING` sementara sampai Sprint 8.
- [x] Unit test sticky potential.
- [x] Factory/seeder demo remark 0-3 dan training.
- [x] OpenAPI diperbarui ke `0.6.0-sprint-6`.

## Demo Evidence

Smoke test API: 12/12 PASS.

- Login Admin, Supervisor, Sales.
- Lead dibuat dan diberikan ke Sales.
- Remark 2 mengubah lead menjadi `POTENTIAL`.
- Remark 1 tidak menurunkan `POTENTIAL`.
- Follow-up schedule dapat difilter.
- Training berhasil dijadwalkan.
- Training berhasil di-reschedule.
- Training berhasil diselesaikan.
- Remark 3 mencatat stage `CLOSING`.
- Remark 0 membuat `INVALID` dan ownership kembali ke Supervisor.
- Sales kehilangan visibility setelah remark 0.
- Sales lain tidak melihat interaction lead bukan miliknya.
- Stage history tercatat untuk perubahan stage.

## Quality

- Migration up: PASS.
- Demo seed: PASS.
- `go test ./internal/activity ./internal/platform/factory ./internal/platform/seeder ./internal/platform/httpserver`: PASS.
- Smoke test API manual: PASS.

## Impediments

- Docker CLI tidak tersedia pada shell Codex, sehingga Docker runtime tetap perlu
  diverifikasi langsung di workstation.
- Integration test DB otomatis untuk activity/training belum dibuat lengkap.

## Risiko Baru

- Risiko: Remark 3 belum terhubung ke closing transaction.
- Dampak: Stage `CLOSING` masih bersifat sementara dan belum boleh dihitung
  sebagai KPI/closing confirmed.
- Mitigasi: Pada Sprint 8, remark 3 dan closing dibuat dalam satu transaksi.
- Owner: Backend Engineer.

## Rencana Sprint Berikutnya

- Sprint 7: Package, Plan, Promotion, dan Benefit.
- Menyiapkan master data paket, tenor, harga, promo, dan eligibility.
- Menjaga agar transaksi historis nantinya memakai snapshot, bukan master data
  yang berubah.
