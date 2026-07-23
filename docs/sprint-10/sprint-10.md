# Sprint 10 - Subscription, Order, dan Reconciliation

## Sprint

Sprint 10

## Periode

24 Juli 2026

## Status

`GREEN`

Sprint Goal tercapai: closing dapat dipertemukan dengan pembelian paket subscription tanpa double counting revenue. Subscription order mendebit wallet, mengaktifkan subscription, membuat period fixed 30 hari, dan menyediakan auto/manual reconciliation beserta issue queue.

## Sprint Goal

Closing dapat dipertemukan dengan pembelian paket tanpa double counting revenue.

## Committed Deliverables

- Subscription order.
- Subscription dan subscription period.
- Wallet debit untuk pembelian.
- Reconciliation otomatis/manual.
- Reconciliation issue queue.
- Fixed 30-day duration.
- Subscription/Reconciliation factories.
- Seed top-up April dan pembelian Juli.

## Completed

- [x] Migration `subscription_orders`, `subscriptions`, `subscription_periods`, `subscription_reconciliations`, dan `reconciliation_issues`.
- [x] API list/detail subscription order.
- [x] API create subscription order dari wallet owner.
- [x] API list/detail subscription.
- [x] API list reconciliation dan reconciliation issue queue.
- [x] API manual reconciliation order ke closing.
- [x] Auto reconciliation ketika order dibuat dengan `closing_id`.
- [x] Wallet debit atomic untuk pembelian paket.
- [x] Subscription activation dan period creation atomic bersama order.
- [x] Durasi subscription memakai fixed `tenure_months x 30 hari` dan benefit free duration dari promo snapshot.
- [x] Idempotency create order memakai `idempotency_key` atau `external_reference`.
- [x] Saldo wallet tidak dapat negatif.
- [x] Snapshot package, plan, dan promotion disimpan di order agar transaksi historis tidak berubah ketika master data diubah.
- [x] Visibility read mengikuti ownership owner/lead: Admin all, Supervisor/Sales sesuai cakupan data.
- [x] Create subscription order Admin-only.
- [x] Manual reconciliation Admin/Supervisor.
- [x] Factory subscription order, subscription, period, reconciliation, dan issue.
- [x] Seeder demo skenario top-up April dan pembelian Juli.
- [x] OpenAPI diperbarui ke `0.10.0-sprint-10`.
- [x] API Testing Report Sprint 10 dibuat.

## Not Completed / Carry Over

Tidak ada carry over blocker untuk scope Sprint 10.

Catatan teknis untuk sprint berikutnya:

- Saat masuk domain komisi Sprint 12, `subscription_reconciliations.status = CONFIRMED` dapat menjadi source-of-truth untuk mengaktifkan earning mitra.
- Saat masuk KPI Sprint 13, KPI closing sebaiknya membaca closing yang sudah confirmed melalui reconciliation, bukan pending closing.
- Untuk production hardening, perlu diputuskan apakah manual confirm dengan selisih nominal non-zero tetap boleh langsung confirmed atau harus masuk approval/review tambahan.

## Demo Evidence

Smoke test API lokal dijalankan pada port `8080`.

| Area | Evidence | Result |
| --- | --- | --- |
| Auth | Login Admin dummy | PASS |
| List Order | `GET /api/v1/subscription-orders?sort=-purchased_at&limit=5` | PASS |
| Detail Order | `GET /api/v1/subscription-orders/1` | PASS |
| Create Order | `POST /api/v1/owners/4/subscription-orders` | PASS |
| Idempotency | Repeat create order dengan idempotency key sama | PASS |
| List Subscription | `GET /api/v1/subscriptions?limit=5` | PASS |
| Detail Subscription | `GET /api/v1/subscriptions/1` | PASS |
| Auto Reconciliation | Seed order owner 3 linked closing 9 | PASS |
| Manual Reconciliation | `POST /api/v1/subscription-orders/4/reconcile` dengan closing 7 | PASS |
| Issue Queue | `GET /api/v1/reconciliation-issues?status=OPEN&limit=5` | PASS |
| Error Handling | Tanpa JWT | PASS |
| Error Handling | Invalid sort | PASS |
| Error Handling | Missing idempotency | PASS |
| Error Handling | Saldo tidak cukup | PASS |
| Error Handling | Sales forbidden create order | PASS |
| Error Handling | Already reconciled order | PASS |
| Error Handling | Invalid date filter | PASS |

Actual smoke output ringkas:

```text
Create order status=201
Repeat create idempotent=true
Manual reconcile status=200
Manual reconcile result=CONFIRMED / MANUAL
Manual reconcile amount_difference=-101.00
orders=4
subscriptions=4
reconciliations=2
openIssues=2
```

Dokumen API Testing detail:

- `docs/sprint-10/README.md`

## Seeder Evidence

Command seed demo:

```powershell
go run . seed demo --preset=minimal --seed=20260723 --as-of=2026-07-01
```

Data demo Sprint 10:

| Scenario | Owner | Alur | Expected |
| --- | --- | --- | --- |
| Top-up April dan pembelian Juli | Owner 3 | Top-up April, closing dan pembelian Juli Pro 12 bulan + promo alat POS | Revenue top-up berada di April; performa Sales aktif Juli setelah reconciliation confirmed; tidak ada double counting revenue Juli. |
| Hanging transaction | Owner 4 | Top-up April lalu pembelian Basic 1 bulan tanpa closing | Order `PAID`, subscription aktif, issue `HANGING_ORDER` terbuka. |
| Manual reconciliation smoke | Owner 1 | Order Basic 1 bulan direconcile manual ke closing pending | Order `RECONCILED`, reconciliation `CONFIRMED`, amount difference tercatat. |

## Quality

| Quality Gate | Result | Catatan |
| --- | --- | --- |
| Migration up | PASS | Database berhasil naik sampai `20260723000800_subscription_order_reconciliation.sql`. |
| Seeder master/demo | PASS | `seed master` dan `seed demo minimal` berhasil. |
| `go build .` | PASS | Binary root berhasil dibuat. |
| `go vet ./...` | PASS | Tidak ada output error. |
| `go test -p 1 ./...` | PASS with environment note | Semua package tampil `ok`; proses Go di Windows mengembalikan exit code karena gagal cleanup temp exe `Access is denied`. |
| OpenAPI updated | PASS | Version `0.10.0-sprint-10`, route Sprint 10 ditambahkan. |
| API smoke test | PASS | Success/error cases terdokumentasi di `docs/sprint-10/README.md`. |
| Docker build | NOT VERIFIED | Command `docker` belum dikenali oleh terminal saat verifikasi lokal; perlu cek PATH Docker Desktop pada environment user. |

## Defect Found During Testing

| Defect | Dampak | Root Cause | Fix | Status |
| --- | --- | --- | --- | --- |
| OpenAPI schema `TopupResponse` dan `CreateOwnerRequest` menempel dalam satu baris. | YAML OpenAPI berpotensi invalid dan Swagger dapat gagal parsing schema setelah wallet section. | Formatting dari update sebelumnya tidak memberi newline sebelum `CreateOwnerRequest`. | Newline diperbaiki dan route/schema Sprint 10 ditambahkan. | CLOSED |

## Defect Terbuka

Tidak ada defect blocker atau critical untuk scope Sprint 10.

## Impediments

- Docker CLI belum tersedia di PATH terminal verifikasi lokal, sehingga Docker build belum dapat divalidasi dari sesi ini.
- Go test pada Windows sempat mengembalikan exit code non-zero karena cleanup binary test di temp folder ditolak OS, walaupun semua package test menampilkan `ok`.

## Risiko Baru

| Risiko | Dampak | Mitigasi | Owner |
| --- | --- | --- | --- |
| Manual reconciliation dengan `amount_difference` non-zero saat ini masih dapat confirmed selama owner cocok. | Stakeholder mungkin ingin mismatch nominal wajib review, bukan langsung confirmed. | Validasi aturan bisnis saat Sprint Review sebelum domain KPI/komisi memakai data confirmed. | Product Owner + Backend Engineer |
| Reconciliation menjadi source untuk KPI/komisi berikutnya. | Salah status reconciliation dapat memengaruhi performa Sales dan komisi. | KPI/komisi hanya membaca `CONFIRMED`, tambah test regresi pada Sprint 12-13. | Backend Engineer |

## Keputusan yang Dibutuhkan

- Apakah manual confirm dengan selisih nominal non-zero boleh tetap confirmed, atau harus dipaksa reject/review?
- Apakah subscription yang order-nya rejected manual harus tetap aktif, dibatalkan, atau diberi status khusus? Saat ini scope Sprint 10 fokus pada status order/reconciliation.

## Rencana Sprint Berikutnya

Sprint 11 - Partner, PIC, Referral, dan Call Mitra.

Fokus berikutnya:

- Partner type dan partner.
- Rekening partner terenkripsi dan response masked.
- Partner assignment/PIC aktif.
- Partner interaction/call.
- Partner referral ke customer lead.
- Factory dan demo scenario partner/referral.