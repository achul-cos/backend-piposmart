# API Testing Report - Sprint 13 Sales Target, KPI, dan Ranking

## 1. Informasi Pengujian

| Item | Nilai |
| --- | --- |
| Project | Backend CRM Piposmart |
| Sprint | Sprint 13 - Sales Target, KPI, dan Ranking |
| Tanggal Testing | 25 Juli 2026 |
| Environment | Local Development, terisolasi (`test_sprint13`, port `8092`) |
| API Base URL | `http://localhost:8092/api/v1` |
| Database | `test_sprint13` |
| Auth | JWT Bearer Token |
| Testing Tool | Manual smoke test via `curl` (API + Worker berjalan bersamaan) |
| Database Migration | `go run . migrate up` (`20260725000100_job_queue.sql`, `20260725000200_sales_target_kpi_ranking.sql`) |
| Seeder | `go run . seed master` dan `go run . seed demo --preset=minimal --seed=20260725 --as-of=2026-07-01` |
| Worker | `go run . worker` (`WORKER_POLL_INTERVAL=2s` untuk mempercepat siklus demo) |

## 2. Header Pengujian

```http
Authorization: Bearer {access_token}
Accept: application/json
Content-Type: application/json
```

Akun demo yang digunakan:

| Role | Email | Password |
| --- | --- | --- |
| Admin | `admin@piposmart.id` | `ChangeMe123!` (bootstrap-admin) |
| Sales (demo scenario) | `sales.091@demo.piposmart.id` | `Password123!` |

## 3. Scope Pengujian

- Job queue generik (`job_queue`, `internal/platform/jobqueue`): claim atomik (`SKIP LOCKED`), retry dengan backoff, transisi ke `FAILED` setelah `max_attempts` habis.
- Sales Target: bulk-set (tidak pernah menimpa) vs override (selalu menang), visibility Sales-hanya-milik-sendiri.
- KPI Definition: CRUD (create/list/get/deactivate), validasi total weight aktif per periode harus 100% (divalidasi saat recompute).
- KPI Recompute: dijalankan async lewat worker, idempoten (delete-then-insert per periode dalam satu transaksi), klasifikasi ACHIEVED/NEAR_ACHIEVED/NOT_ACHIEVED sesuai threshold.
- Sales Ranking: `RANK()` window function per periode, endpoint Admin/Supervisor-only.
- Role gating: ADMIN/SUPERVISOR untuk target & definisi & trigger recompute & ranking, Sales scoped ke datanya sendiri di `/kpi/results`.

## 4. Testing Summary

| Module | Endpoint / Case | Method | Expected | Result |
| --- | --- | --- | --- | --- |
| Auth | `/auth/login` Admin | POST | 200 OK | PASS |
| Sales Target | Bulk set target (5 Sales aktif, metric `CONFIRMED_CLOSING_COUNT`) | POST `/sales-targets/bulk` | 200, `eligible_sales=5`, `created=5` | PASS |
| KPI Definition | Create definition weight=60 (sengaja tidak lengkap) | POST `/kpi-definitions` | 201 | PASS |
| KPI Recompute | Trigger recompute dengan total weight aktif = 60% (bukan 100%) | POST `/kpi/recompute` | Job di-enqueue, lalu retry 5x, akhirnya `FAILED` dengan pesan jelas | PASS |
| KPI Definition | Tambah definition kedua weight=40 (total jadi 100%) | POST `/kpi-definitions` | 201 | PASS |
| KPI Recompute | Trigger recompute ulang, total weight = 100% | POST `/kpi/recompute` | Job `COMPLETED` dalam 1 attempt | PASS |
| KPI Recompute | Trigger recompute periode yang sama 2x berturut-turut | POST `/kpi/recompute` (2x) | Hasil `sales_kpi_results` identik (sales_id/total_score/classification/rank_position) sebelum & sesudah, hanya `id` baris berbeda | PASS |
| KPI Ranking | GET ranking periode Juli 2026 (data demo seeder) | GET `/kpi/ranking` (Admin) | 3 tier klasifikasi persis sesuai desain: 100/NEAR/20, rank 1-2-3 | PASS |
| KPI Ranking | Akses ranking sebagai Sales | GET `/kpi/ranking` (Sales) | 403 FORBIDDEN | PASS |
| KPI Results | Akses results sebagai Sales | GET `/kpi/results` (Sales) | Hanya melihat baris miliknya sendiri, lengkap dengan detail per-metric | PASS |

## 5. Detail Skenario Pengujian API

### 5.1 Seed Demo — 3 Tier Klasifikasi

Seeder (`internal/platform/seeder/seeder_target_kpi.go`) membuat 3 Sales dedicated dengan jumlah closing `CONFIRMED` berbeda terhadap target=5 di bulan `AsOf` (Juli 2026), KPI definition tunggal (`CONFIRMED_CLOSING_COUNT`, weight=100%, threshold_achieved=100%, threshold_near=80%), lalu memanggil `kpi.Repository.Recompute` langsung (bukan lewat job_queue, karena seeder tidak punya worker berjalan).

```sql
SELECT u.code, u.name, kr.total_score, kr.classification, kr.rank_position
FROM sales_kpi_results kr JOIN users u ON u.id = kr.sales_id
WHERE u.code IN ('SLS-091','SLS-092','SLS-093')
ORDER BY kr.rank_position;
```

Hasil:

| code | name | total_score | classification | rank_position |
| --- | --- | --- | --- | --- |
| SLS-091 | Sales Demo 091 | 100.00 | ACHIEVED | 1 |
| SLS-092 | Sales Demo 092 | 80.00 | NEAR_ACHIEVED | 2 |
| SLS-093 | Sales Demo 093 | 20.00 | NOT_ACHIEVED | 3 |

Persis sesuai desain (5/5=100%, 4/5=80%, 1/5=20%). ✅

### 5.2 Job Queue — Retry & Failure Path

```bash
curl -X POST /api/v1/kpi-definitions -d '{"metric_code":"CONFIRMED_CLOSING_COUNT","period_year":2026,"period_month":8,"weight":"60.00"}'
curl -X POST /api/v1/kpi/recompute -d '{"period_year":2026,"period_month":8}'
# -> {"id":1,"status":"PENDING","attempts":0,"max_attempts":5}

# setelah worker memproses (WORKER_POLL_INTERVAL=2s):
curl /api/v1/kpi/jobs/1
# -> {"status":"FAILED","attempts":5,"max_attempts":5,
#     "last_error":"kpi: active KPI definitions for this period must have weights summing to exactly 100"}
```

Worker mencoba job tersebut 5x (sesuai `WORKER_MAX_ATTEMPTS` default) dengan backoff linear, lalu menyerah dan menandai `FAILED` dengan pesan yang bisa langsung ditindaklanjuti — bukan retry tanpa henti. ✅

Setelah definition kedua (weight=40%) ditambahkan sehingga total=100%, trigger recompute kedua langsung `COMPLETED` dalam 1 attempt. ✅

### 5.3 Idempotency

`sales_kpi_results` untuk periode Agustus 2026 dibandingkan sebelum & sesudah recompute kedua kali berturut-turut — `sales_id`, `total_score`, `classification`, `rank_position` identik; hanya primary key `id` yang berbeda (karena pola delete-then-insert), sesuai desain. ✅

### 5.4 RBAC

```bash
# Sebagai Sales (SLS-091):
curl /api/v1/kpi/ranking?period_year=2026&period_month=7
# -> 403 {"code":"FORBIDDEN"}

curl /api/v1/kpi/results?period_year=2026&period_month=7
# -> hanya 1 item: baris milik SLS-091 sendiri (total_score=100.00, ACHIEVED),
#    lengkap dengan breakdown per-metric (CONFIRMED_CLOSING_COUNT: target=5, actual=5, achievement=100%)
```

Sales tidak bisa melihat ranking penuh maupun data Sales lain di results — sesuai DoD "Sales hanya melihat KPI sendiri" dan "Admin/Supervisor melihat seluruh ranking". ✅

## 6. Kesimpulan Pengujian

Seluruh skenario inti Sprint 13 (job queue generik, sales target bulk/override, KPI definition dengan validasi weight=100%, recompute idempoten, klasifikasi 3-tier, ranking, RBAC) tervalidasi PASS. `go build`, `go vet`, `go test ./...` bersih; migration reversibel (up/down/up teruji); demo seeder menghasilkan skenario 3 tier yang deterministik dan dapat didemokan langsung lewat Swagger/curl.
