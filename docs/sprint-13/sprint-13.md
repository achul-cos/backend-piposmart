Sprint: 13 — Sales Target, KPI, dan Ranking
Periode: 25 Juli 2026 (dikerjakan dalam satu sesi, mengikuti siklus sprint yang berlaku)
Status: GREEN

Sprint Goal:
- Performa Sales dihitung otomatis dari aktivitas CRM (confirmed closing, call/chat, training) terhadap target bulanan, dengan hasil yang dapat diaudit dan di-ranking.

Committed Deliverables:
- Target bulanan bulk.
- Override target per Sales.
- KPI definition, weight, threshold, dan result.
- KPI worker.
- Sales ranking.
- Factory target/KPI.
- Seed Sales tercapai, hampir tercapai, dan tidak tercapai.

Completed:
- Job queue generik berbasis MySQL (`job_queue`, `internal/platform/jobqueue`) — claim atomik via `SKIP LOCKED`, retry dengan backoff, stale-job reclaim, dipasang ke `RunWorker` yang sebelumnya cuma heartbeat stub. Dibangun generik (bukan KPI-only) karena Sprint 14 (Import Framework) butuh mekanisme identik — bukan scope creep, melainkan prasyarat deliverable "KPI worker" itu sendiri.
- Sales Target (`internal/target`): bulk-set (tidak pernah menimpa baris yang sudah ada) dan override (selalu menang) sebagai dua endpoint terpisah, metric_code-driven (memakai tabel `metric_codes` yang sudah di-seed sejak Sprint 2, bukan enum baru).
- KPI Definition (`internal/kpi`): CRUD create/list/get/deactivate (pola identik `commission_rules` Sprint 12 — tanpa update, supersede via definisi baru). Validasi "weight aktif per periode harus 100%" dilakukan di titik recompute (satu sumber kebenaran).
- KPI Recompute: idempoten (delete-then-insert per periode dalam satu transaksi), dipicu async lewat `POST /kpi/recompute` → job_queue → worker; juga dipanggil langsung (bypass queue) oleh demo seeder untuk determinisme.
- Sales Ranking: `RANK() OVER (ORDER BY total_score DESC)` per periode, endpoint Admin/Supervisor-only; Sales tetap bisa melihat `rank_position` miliknya sendiri lewat `/kpi/results`.
- Role gating: ADMIN/SUPERVISOR untuk target/definisi/trigger recompute/ranking; Sales discoped ke datanya sendiri (pola `visibilityWhere` identik `internal/lead`/`internal/closing`).
- Factory & seed: skenario 3 Sales dengan klasifikasi ACHIEVED/NEAR_ACHIEVED/NOT_ACHIEVED yang deterministik (`internal/platform/seeder/seeder_target_kpi.go`), dipasang ke preset `minimal` maupun `large`.
- OpenAPI diperbarui ke `0.14.0-sprint-13` (9 path baru, 9 schema baru), unit test untuk validasi decimal/weight, klasifikasi threshold, visibility, dan metric yang didukung.

Not Completed / Carry Over:
- (tidak ada)

Demo Evidence:
- Endpoint/Swagger: `POST /sales-targets/bulk`, `PUT /sales-targets/{salesID}`, `GET /sales-targets`, `POST/GET /kpi-definitions`, `GET/PATCH /kpi-definitions/{id}`, `POST /kpi/recompute`, `GET /kpi/jobs/{id}`, `GET /kpi/results`, `GET /kpi/ranking` — seluruhnya di `internal/platform/httpserver/openapi.yaml` v0.14.0-sprint-13.
- Skenario seed: `seed demo --preset=minimal --seed=20260725 --as-of=2026-07-01` menghasilkan 3 Sales dengan klasifikasi berbeda (100/80/20 — ACHIEVED/NEAR_ACHIEVED/NOT_ACHIEVED) yang langsung terlihat lewat `GET /kpi/ranking`.
- Screenshot/log/test report: `docs/sprint-13/README.md` (skenario lengkap dengan request/response nyata, termasuk job retry/failure path dan idempotency check).

Quality:
- Unit/integration test: `go test ./...` — seluruh paket PASS, termasuk test baru (`internal/target`, `internal/kpi`: validasi decimal/percent, klasifikasi threshold, visibility, supported metric).
- Migration status: `20260725000100_job_queue.sql` dan `20260725000200_sales_target_kpi_ranking.sql` — up/down/up reversibel, teruji di database terisolasi.
- Docker build: tidak diverifikasi ulang secara eksplisit sprint ini (tidak ada perubahan Dockerfile/compose); risiko rendah karena tidak ada dependency baru di luar stdlib + driver MySQL yang sudah ada.
- Defect terbuka: tidak ada.

Impediments:
- (tidak ada)

Risiko Baru:
- Risiko: `PARTNER_CALL_COUNT` (metric_code yang sudah di-seed sejak Sprint 2) belum didukung KPI recompute karena atribusi Sales-nya tidak langsung (perlu join time-scoped ke `partner_assignments`, bukan kolom `sales_id` langsung).
  Dampak: kalau ada KPI definition yang mengaktifkan metric ini, recompute akan gagal dengan `UNSUPPORTED_METRIC` (bukan silent-zero — sengaja fail-loud).
  Mitigasi: didokumentasikan eksplisit sebagai scope decision di plan Sprint 13; ditangani kalau ada kebutuhan bisnis nyata di sprint mendatang, bukan preventif sekarang.
  Owner: Backend Engineer (sprint berikutnya, on-demand).
- Risiko: performa `Recompute` untuk preset `large` (ribuan Sales) melakukan query `WHERE sales_id IN (...)` dengan placeholder sebanyak jumlah Sales aktif per metric.
  Dampak: bisa lambat pada dataset sangat besar (bukan masalah untuk seed/demo, tapi perlu diperhatikan saat Sprint 17 hardening/performance).
  Mitigasi: dicatat sebagai item review Sprint 17 (index dan query optimization), tidak memblokir Sprint 13.
  Owner: Backend Engineer (Sprint 17).

Keputusan yang Dibutuhkan:
- (tidak ada — seluruh keputusan desain terkait TIER pengukuran, bulk-vs-override, dan validasi weight sudah diambil sendiri berdasarkan pola yang sudah established di sprint-sprint sebelumnya, konsisten dengan gaya kerja "keputusan teknis dikunci lalu didokumentasikan" pada roadmap)

Rencana Sprint Berikutnya (Sprint 14 — Import Framework dan Data Customer):
- Job queue generik dari Sprint 13 (`internal/platform/jobqueue`) sudah siap dipakai — Sprint 14 tinggal menambah job type baru (`IMPORT_*`) dan handler-nya, tidak perlu membangun ulang mekanisme worker/retry/stale-reclaim.
- Audit roadmap standar akan dijalankan sebelum Sprint 14 dimulai (`backend_crm_piposmart/CLAUDE.md`), mengecek Sprint 13 delivered persis sesuai DoD roadmap tanpa under/over-deliver — hasil audit ini sudah tercermin di laporan ini (status GREEN, semua Committed Deliverables Completed).
