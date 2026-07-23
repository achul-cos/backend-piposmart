# Sprint 09 - Payment dan Wallet Ledger

## Sprint

Sprint 09

## Periode

23 Juli 2026 - 24 Juli 2026

## Status

`GREEN`

Sprint Goal tercapai: top-up dan saldo owner dapat dicatat secara aman, auditable, idempotent, serta dapat direkap sebagai revenue top-up berdasarkan `paid_at`.

## Sprint Goal

Top-up dan saldo owner tercatat secara aman dan dapat diaudit.

## Committed Deliverables

- Wallet account.
- Payment/top-up.
- Wallet transaction ledger.
- Credit, debit, adjustment, dan refund.
- Row locking dan idempotency external reference.
- Revenue top-up berdasarkan `paid_at`.
- PaymentFactory dan WalletTransactionFactory.
- Seed top-up terpakai dan belum terpakai.

## Completed

- [x] Migration `wallet_accounts`, `wallet_payments`, dan `wallet_transactions`.
- [x] API list wallet, detail wallet owner, list payment, detail payment, list ledger, dan ledger per owner.
- [x] API create top-up, debit, adjustment, dan refund.
- [x] Top-up membuat payment + ledger credit + update balance dalam satu database transaction.
- [x] Debit/refund/adjustment debit menolak saldo negatif.
- [x] Idempotency diterapkan pada top-up dan manual ledger mutation.
- [x] Admin-only untuk mutasi wallet.
- [x] Supervisor/Sales read access mengikuti visibility owner/lead.
- [x] Money handling memakai decimal string, bukan `float64`.
- [x] Unit test decimal, saldo negatif, dan idempotency.
- [x] Factory/seeder wallet untuk top-up terpakai dan belum terpakai.
- [x] OpenAPI diperbarui ke `0.9.0-sprint-9`.
- [x] README root diperbarui dengan route wallet/payment/ledger.
- [x] API Testing Report Sprint 09 diperluas dengan route coverage, query params, success cases, error cases, dan smoke evidence.

## Not Completed / Carry Over

Tidak ada carry over untuk scope Sprint 09.

Catatan roadmap:

- Wallet debit pada Sprint 09 masih menyediakan debit manual.
- Integrasi order/subscription/reconciliation otomatis masuk Sprint 10 sesuai roadmap.

## Demo Evidence

Smoke test API lokal dijalankan pada port `18092`.

| Area | Evidence | Result |
| --- | --- | --- |
| Auth | Login Admin dan Sales dummy | PASS |
| Wallet read | `GET /api/v1/wallets` | PASS |
| Wallet detail | `GET /api/v1/owners/{owner_id}/wallet` | PASS |
| Top-up | `POST /api/v1/owners/{owner_id}/wallet/topups` | PASS |
| Idempotency | Repeat top-up dengan idempotency key sama | PASS |
| Payment detail | `GET /api/v1/wallet-payments/{payment_id}` | PASS |
| Debit | `POST /api/v1/owners/{owner_id}/wallet/debits` | PASS |
| Payment report | `GET /api/v1/wallet-payments?paid_from=2026-07-01&paid_to=2026-07-31` | PASS |
| Ledger report | `GET /api/v1/owners/{owner_id}/wallet/transactions` | PASS |
| Error handling | Missing idempotency | PASS |
| Error handling | Debit melebihi saldo | PASS |
| RBAC | Sales forbidden membuat adjustment | PASS |
| Ledger consistency | `balance == ledger_balance` | PASS |

Actual smoke output:

```text
Case                       Status Pass
----                       ------ ----
Admin login                   200 True
Sales login                   200 True
List wallets                  200 True
Create top-up                 201 True
Idempotent top-up             201 True
Detail payment                200 True
Create debit                  201 True
Missing idempotency error     400 True
Over debit error              400 True
Sales forbidden adjustment    403 True
Get owner wallet              200 True
List payments                 200 True
List owner ledger             200 True

OWNER_ID=1
PAYMENT_ID=7
WALLET_BALANCE=3000000.00
WALLET_LEDGER_BALANCE=3000000.00
```

Dokumen API Testing detail:

- `docs/sprint-09/README.md`

## Seeder Evidence

Seeder demo:

```powershell
go run . seed demo --preset=minimal --seed=20260723 --as-of=2026-07-01
```

Data demo yang dibuat:

| Scenario | Owner | Top-up | Debit | Expected Balance |
| --- | --- | ---: | ---: | ---: |
| Top-up terpakai sebagian | Owner demo pertama | 2.000.000 | 500.000 | 1.500.000 |
| Top-up belum terpakai | Owner demo kedua | 1.250.000 | 0 | 1.250.000 |

Command evidence:

| Command | Result |
| --- | --- |
| `go run . migrate up` | PASS |
| `go run . migrate down` untuk migration Sprint 09 | PASS |
| `go run . migrate up` setelah rollback | PASS |
| `go run . seed master` | PASS |
| `go run . seed demo --preset=minimal --seed=20260723 --as-of=2026-07-01` | PASS |
| `go run . migrate status` | PASS, database berada pada `20260723000700_wallet_payment_ledger.sql` |

## Quality

| Quality Gate | Result |
| --- | --- |
| `go test ./...` | PASS |
| `go vet ./...` | PASS |
| `go build .` | PASS |
| Unit test wallet money/idempotency | PASS |
| Migration up/down | PASS |
| Seeder master/demo | PASS |
| Smoke test API Sprint 09 | PASS |
| OpenAPI updated | PASS |
| README route updated | PASS |

## Defect Found During Testing

| Defect | Dampak | Root Cause | Fix | Status |
| --- | --- | --- | --- | --- |
| Create top-up/debit sempat menghasilkan `500 Internal Server Error` pada smoke test ketika request berbeda memakai `paid_at` atau `occurred_at` yang sama. | Mutasi wallet gagal karena duplicate unique code. | Generator `code` payment/ledger memakai tanggal bisnis (`paid_at` / `occurred_at`) sebagai komponen waktu. | Generator code diubah memakai waktu pembuatan aktual `time.Now().UTC()`. Tanggal bisnis tetap tersimpan pada `paid_at` / `occurred_at`. | CLOSED |

## Defect Terbuka

Tidak ada defect blocker atau critical untuk scope Sprint 09.

## Impediments

- PowerShell/curl memiliki beberapa kendala quoting JSON saat smoke test manual.
- Mitigasi: Payload JSON untuk curl ditulis ke file sementara di workspace agar request body persis dan dapat direproduksi.
- Sandbox Windows/ACL masih memerlukan command escalated terbatas pada folder project untuk read/write/test tertentu.

## Risiko Baru

| Risiko | Dampak | Mitigasi | Owner |
| --- | --- | --- | --- |
| Domain finansial rawan race condition | Saldo bisa tidak akurat jika mutasi paralel tidak dikunci | Row locking pada wallet saat top-up/debit/adjustment/refund | Backend Engineer |
| Double processing payment | Saldo/revenue bisa tercatat dua kali | Idempotency key dan external reference unik | Backend Engineer |
| Double counting revenue | Top-up dan closing bisa salah dijumlah sebagai satu metrik | Dokumentasi memisahkan revenue top-up (`paid_at`) dan performance closing | Backend Engineer |
| Integrasi subscription belum ada | Debit manual belum otomatis mengaktifkan subscription | Sprint 10 mengerjakan order, subscription, dan reconciliation | Backend Engineer |

## Keputusan yang Dibutuhkan

Tidak ada keputusan baru untuk Sprint 09.

## Rencana Sprint Berikutnya

Sprint 10 - Subscription, Order, dan Reconciliation.

Fokus Sprint 10:

- Subscription order.
- Wallet debit untuk pembelian paket.
- Subscription dan subscription period.
- Reconciliation otomatis/manual antara closing, top-up, order, dan subscription.
- Issue queue untuk transaksi menggantung.
- Demo utama: top-up April menjadi revenue April; pembelian/closing Juli menjadi performance Sales Juli tanpa revenue ganda.
