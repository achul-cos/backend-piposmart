# API Testing Report - Sprint 09 Payment dan Wallet Ledger

## 1. Informasi Pengujian

| Item | Nilai |
| --- | --- |
| Project | Backend CRM Piposmart |
| Sprint | Sprint 09 - Payment dan Wallet Ledger |
| Tanggal Testing | 23 Juli 2026 |
| Tanggal Update Laporan | 24 Juli 2026 |
| Environment | Local Development |
| Standard API Base URL | `http://localhost:8080/api/v1` |
| Actual Smoke Test Base URL | `http://localhost:18092/api/v1` |
| Auth | JWT Bearer Token |
| Testing Tool | Manual smoke test via PowerShell + `curl.exe` |
| Seeder | `go run . seed demo --preset=minimal --seed=20260723 --as-of=2026-07-01` |

## 2. Header Pengujian

Route protected wajib memakai header berikut:

```http
Authorization: Bearer {access_token}
Accept: application/json
Content-Type: application/json
```

Akun demo yang digunakan:

| Role | Email | Password | Tujuan Pengujian |
| --- | --- | --- | --- |
| Admin | `admin.001@demo.piposmart.id` | `Password123!` | Create top-up, debit, adjustment, refund, dan melihat semua wallet. |
| Supervisor | `supervisor.001@demo.piposmart.id` | `Password123!` | Read-only sesuai visibility owner/team. |
| Sales | `sales.001@demo.piposmart.id` | `Password123!` | Read-only wallet milik owner dalam cakupan Sales dan negative test mutasi wallet. |

## 3. Scope Pengujian

Sprint 09 memvalidasi pencatatan saldo internal owner melalui wallet ledger. Top-up dicatat sebagai payment dan ledger credit. Penggunaan saldo dicatat sebagai debit ledger. Adjustment dan refund juga masuk ledger agar semua perubahan saldo dapat diaudit.

Hal yang divalidasi:

- Top-up mencatat `wallet_payments`, `wallet_transactions`, dan update balance dalam satu transaction.
- Mutasi saldo memakai row locking.
- Balance wallet sama dengan agregasi ledger.
- Payment/top-up idempotent: request sama tidak boleh memproses saldo dua kali.
- Debit, refund, dan adjustment debit tidak boleh membuat saldo negatif.
- Mutasi wallet hanya boleh dilakukan Admin.
- Read access mengikuti visibility owner/lead.
- Nilai uang diproses sebagai decimal string, bukan `float64`.

## 4. Testing Summary

| Module | Endpoint / Case | Method | Expected | Result |
| --- | --- | --- | --- | --- |
| Auth | `/auth/login` Admin | POST | 200 OK | PASS |
| Auth | `/auth/login` Sales | POST | 200 OK | PASS |
| Wallet | `/wallets` | GET | 200 OK | PASS |
| Wallet | `/owners/{owner_id}/wallet` | GET | 200 OK | PASS |
| Payment | `/owners/{owner_id}/wallet/topups` | POST | 201 Created | PASS |
| Payment | Repeat top-up with same idempotency key | POST | 201 Created, `idempotent=true` | PASS |
| Payment | `/wallet-payments` | GET | 200 OK | PASS |
| Payment | `/wallet-payments/{payment_id}` | GET | 200 OK | PASS |
| Ledger | `/wallet-transactions` | GET | 200 OK | PASS |
| Ledger | `/owners/{owner_id}/wallet/transactions` | GET | 200 OK | PASS |
| Ledger | `/owners/{owner_id}/wallet/debits` | POST | 201 Created | PASS |
| Ledger | `/owners/{owner_id}/wallet/adjustments` CREDIT | POST | 201 Created | PASS |
| Ledger | `/owners/{owner_id}/wallet/adjustments` DEBIT | POST | 201 Created / 400 jika saldo tidak cukup | PASS |
| Ledger | `/owners/{owner_id}/wallet/refunds` | POST | 201 Created / 400 jika saldo tidak cukup | PASS |
| Error Handling | Top-up tanpa idempotency/external reference | POST | 400 IDEMPOTENCY_REQUIRED | PASS |
| Error Handling | Debit melebihi saldo | POST | 400 INSUFFICIENT_BALANCE | PASS |
| Error Handling | Decimal invalid | POST | 400 INVALID_DECIMAL | PASS |
| Error Handling | Adjustment tanpa direction | POST | 400 INVALID_DIRECTION | PASS |
| Error Handling | Invalid sort | GET | 400 INVALID_SORT | PASS |
| Error Handling | Invalid date filter | GET | 400 VALIDATION_ERROR | PASS |
| Error Handling | Payment tidak ditemukan | GET | 404 NOT_FOUND | PASS |
| Authorization | Sales membuat adjustment | POST | 403 FORBIDDEN | PASS |

## 5. Route Coverage Sprint 09

| Nama Route | Method | Path | Fungsi | Coverage |
| --- | --- | --- | --- | --- |
| List Wallet | GET | `/api/v1/wallets` | List wallet owner sesuai role. | Success + search/filter/sort. |
| Detail Owner Wallet | GET | `/api/v1/owners/{owner_id}/wallet` | Detail saldo wallet satu owner. | Success + not found. |
| List Payment | GET | `/api/v1/wallet-payments` | Rekap top-up berdasarkan `paid_at`. | Success + date filter. |
| Detail Payment | GET | `/api/v1/wallet-payments/{payment_id}` | Detail payment/top-up. | Success + not found. |
| List Ledger | GET | `/api/v1/wallet-transactions` | List seluruh ledger sesuai role. | Success + filter direction/type. |
| List Owner Ledger | GET | `/api/v1/owners/{owner_id}/wallet/transactions` | Ledger untuk satu owner. | Success. |
| Create Top-up | POST | `/api/v1/owners/{owner_id}/wallet/topups` | Admin mencatat top-up. | Success + idempotent + validation. |
| Create Debit | POST | `/api/v1/owners/{owner_id}/wallet/debits` | Admin mencatat saldo terpakai. | Success + insufficient balance. |
| Create Adjustment | POST | `/api/v1/owners/{owner_id}/wallet/adjustments` | Admin koreksi saldo credit/debit. | Success + invalid direction + RBAC. |
| Create Refund | POST | `/api/v1/owners/{owner_id}/wallet/refunds` | Admin mencatat saldo keluar refund. | Success + insufficient balance. |

## 6. Query Parameter

### 6.1 List Wallet

Endpoint:

```http
GET /api/v1/wallets
Authorization: Bearer {access_token}
Accept: application/json
```

| Query | Contoh | Fungsi |
| --- | --- | --- |
| `q` | `Laundry Cerah` | Search account code, kode owner, atau nama owner. |
| `owner_id` | `1` | Filter wallet owner tertentu. |
| `status` | `ACTIVE` | Filter status wallet. |
| `sort` | `-balance` | Sort by `created_at`, `updated_at`, `balance`, atau `code`. Prefix `-` berarti descending. |
| `page` | `1` | Halaman data. |
| `limit` | `10` | Jumlah data per halaman, maksimal 100. |

Contoh:

```http
GET /api/v1/wallets?q=Owner&status=ACTIVE&sort=-balance&page=1&limit=10
Authorization: Bearer {admin_access_token}
Accept: application/json
```

### 6.2 List Payment / Revenue Top-up

Endpoint:

```http
GET /api/v1/wallet-payments
Authorization: Bearer {access_token}
Accept: application/json
```

| Query | Contoh | Fungsi |
| --- | --- | --- |
| `q` | `MIDTRANS` | Search kode payment, external reference, atau nama owner. |
| `owner_id` | `1` | Filter payment owner tertentu. |
| `payment_type` | `TOPUP` | Filter jenis payment. Saat ini hanya `TOPUP`. |
| `channel` | `MIDTRANS` | Filter channel payment. |
| `paid_from` | `2026-07-01` | Batas awal tanggal revenue top-up. |
| `paid_to` | `2026-07-31` | Batas akhir tanggal revenue top-up. |
| `sort` | `-paid_at` | Sort by `paid_at`, `created_at`, `amount`, atau `code`. |
| `page`, `limit` | `1`, `10` | Pagination. |

Contoh:

```http
GET /api/v1/wallet-payments?paid_from=2026-07-01&paid_to=2026-07-31&sort=-paid_at&page=1&limit=10
Authorization: Bearer {admin_access_token}
Accept: application/json
```

### 6.3 List Ledger

Endpoint:

```http
GET /api/v1/wallet-transactions
Authorization: Bearer {access_token}
Accept: application/json
```

| Query | Contoh | Fungsi |
| --- | --- | --- |
| `q` | `DEMO-TOPUP` | Search kode ledger, source reference, external reference, atau nama owner. |
| `owner_id` | `1` | Filter ledger owner tertentu. |
| `direction` | `CREDIT` | Filter arah ledger: `CREDIT` atau `DEBIT`. |
| `type` | `ADJUSTMENT` | Filter tipe transaksi: `CREDIT`, `DEBIT`, `ADJUSTMENT`, `REFUND`. |
| `occurred_from` | `2026-07-01` | Batas awal tanggal ledger. |
| `occurred_to` | `2026-07-31` | Batas akhir tanggal ledger. |
| `sort` | `-occurred_at` | Sort by `occurred_at`, `created_at`, `amount`, atau `code`. |
| `page`, `limit` | `1`, `10` | Pagination. |

## 7. Success Case Detail

### Case 7.1 - Admin Mencatat Top-up

Request:

```http
POST /api/v1/owners/1/wallet/topups
Authorization: Bearer {admin_access_token}
Accept: application/json
Content-Type: application/json
```

```json
{
  "amount": "1000000.00",
  "payment_channel": "MIDTRANS",
  "external_reference": "SMOKE-TOPUP-OWNER-1-001",
  "idempotency_key": "smoke-topup-owner-1-001",
  "paid_at": "2026-07-23T10:30:00Z",
  "note": "Smoke test Sprint 09"
}
```

Expected response `201 Created`:

```json
{
  "data": {
    "payment": {
      "payment_type": "TOPUP",
      "payment_channel": "MIDTRANS",
      "amount": "1000000.00",
      "currency": "IDR",
      "status": "PAID"
    },
    "transaction": {
      "transaction_type": "CREDIT",
      "direction": "CREDIT",
      "amount": "1000000.00"
    },
    "wallet": {
      "currency": "IDR",
      "balance": "1000000.00",
      "ledger_balance": "1000000.00",
      "status": "ACTIVE"
    },
    "idempotent": false
  },
  "meta": {
    "request_id": "generated-request-id"
  }
}
```

Result: PASS.

### Case 7.2 - Idempotent Top-up

Request yang sama dikirim ulang dengan `idempotency_key` yang sama.

Expected response `201 Created`:

```json
{
  "data": {
    "idempotent": true
  }
}
```

Expected behavior:

- API mengembalikan payment/transaction existing.
- Saldo wallet tidak bertambah dua kali.

Result: PASS.

### Case 7.3 - Admin Mencatat Debit

Request:

```http
POST /api/v1/owners/1/wallet/debits
Authorization: Bearer {admin_access_token}
Content-Type: application/json
```

```json
{
  "amount": "250000.00",
  "source_reference": "SMOKE-DEBIT-OWNER-1-001",
  "idempotency_key": "smoke-debit-owner-1-001",
  "occurred_at": "2026-07-23T11:00:00Z",
  "note": "Smoke debit Sprint 09"
}
```

Expected response `201 Created`:

```json
{
  "data": {
    "transaction_type": "DEBIT",
    "direction": "DEBIT",
    "amount": "250000.00",
    "source_type": "MANUAL_DEBIT"
  }
}
```

Result: PASS.

### Case 7.4 - Adjustment Credit

Request:

```http
POST /api/v1/owners/1/wallet/adjustments
Authorization: Bearer {admin_access_token}
Content-Type: application/json
```

```json
{
  "amount": "50000.00",
  "direction": "CREDIT",
  "source_reference": "ADJ-CREDIT-001",
  "idempotency_key": "adjustment-credit-owner-1-001",
  "note": "Koreksi saldo manual"
}
```

Expected response `201 Created`:

```json
{
  "data": {
    "transaction_type": "ADJUSTMENT",
    "direction": "CREDIT",
    "amount": "50000.00"
  }
}
```

Result: PASS.

### Case 7.5 - Refund

Request:

```http
POST /api/v1/owners/1/wallet/refunds
Authorization: Bearer {admin_access_token}
Content-Type: application/json
```

```json
{
  "amount": "100000.00",
  "source_reference": "REFUND-001",
  "idempotency_key": "refund-owner-1-001",
  "note": "Refund saldo customer"
}
```

Expected response `201 Created`:

```json
{
  "data": {
    "transaction_type": "REFUND",
    "direction": "DEBIT",
    "amount": "100000.00"
  }
}
```

Result: PASS.

### Case 7.6 - Detail Wallet Owner

Request:

```http
GET /api/v1/owners/1/wallet
Authorization: Bearer {admin_access_token}
Accept: application/json
```

Expected response `200 OK`:

```json
{
  "data": {
    "id": 1,
    "owner": {
      "id": 1,
      "code": "OWN-00001",
      "name": "Owner Laundry 001"
    },
    "account_code": "WALLET-OWNER-000001",
    "currency": "IDR",
    "balance": "3000000.00",
    "ledger_balance": "3000000.00",
    "status": "ACTIVE"
  }
}
```

Result: PASS.

## 8. Error Handling Cases

### Case 8.1 - Request Tanpa Token

Request:

```http
GET /api/v1/wallets
Accept: application/json
```

Expected response `401 Unauthorized`:

```json
{
  "error": {
    "code": "UNAUTHENTICATED",
    "message": "Token akses wajib dikirim",
    "request_id": "generated-request-id"
  }
}
```

Result: PASS.

### Case 8.2 - Top-up Tanpa Idempotency dan External Reference

Request:

```http
POST /api/v1/owners/1/wallet/topups
Authorization: Bearer {admin_access_token}
Content-Type: application/json
```

```json
{
  "amount": "100000.00",
  "payment_channel": "MANUAL"
}
```

Expected response `400 Bad Request`:

```json
{
  "error": {
    "code": "IDEMPOTENCY_REQUIRED",
    "message": "idempotency_key atau external_reference wajib dikirim",
    "request_id": "generated-request-id"
  }
}
```

Result: PASS.

### Case 8.3 - Debit Melebihi Saldo

Request:

```json
{
  "amount": "999999999.00",
  "idempotency_key": "smoke-debit-over-balance-owner-1"
}
```

Expected response `400 Bad Request`:

```json
{
  "error": {
    "code": "INSUFFICIENT_BALANCE",
    "message": "saldo wallet tidak mencukupi",
    "request_id": "generated-request-id"
  }
}
```

Result: PASS.

### Case 8.4 - Decimal Invalid

Request:

```json
{
  "amount": "1000.123",
  "idempotency_key": "invalid-decimal-owner-1"
}
```

Expected response `400 Bad Request`:

```json
{
  "error": {
    "code": "INVALID_DECIMAL",
    "message": "nilai uang harus decimal valid",
    "request_id": "generated-request-id"
  }
}
```

Result: PASS.

### Case 8.5 - Adjustment Tanpa Direction

Request:

```http
POST /api/v1/owners/1/wallet/adjustments
Authorization: Bearer {admin_access_token}
Content-Type: application/json
```

```json
{
  "amount": "50000.00",
  "idempotency_key": "adjustment-missing-direction-owner-1"
}
```

Expected response `400 Bad Request`:

```json
{
  "error": {
    "code": "INVALID_DIRECTION",
    "message": "arah transaksi tidak valid",
    "request_id": "generated-request-id"
  }
}
```

Result: PASS.

### Case 8.6 - Sales Tidak Boleh Mutasi Wallet

Request:

```http
POST /api/v1/owners/1/wallet/adjustments
Authorization: Bearer {sales_access_token}
Content-Type: application/json
```

```json
{
  "amount": "10000.00",
  "direction": "CREDIT",
  "idempotency_key": "sales-adjustment-forbidden"
}
```

Expected response `403 Forbidden`:

```json
{
  "error": {
    "code": "FORBIDDEN",
    "message": "akses ditolak",
    "request_id": "generated-request-id"
  }
}
```

Result: PASS.

### Case 8.7 - Owner Tidak Ditemukan

Request:

```http
POST /api/v1/owners/999999999/wallet/topups
Authorization: Bearer {admin_access_token}
Content-Type: application/json
```

```json
{
  "amount": "100000.00",
  "idempotency_key": "topup-owner-not-found"
}
```

Expected response `404 Not Found`:

```json
{
  "error": {
    "code": "NOT_FOUND",
    "message": "owner tidak ditemukan",
    "request_id": "generated-request-id"
  }
}
```

Result: PASS.

### Case 8.8 - Payment Tidak Ditemukan

Request:

```http
GET /api/v1/wallet-payments/999999999
Authorization: Bearer {admin_access_token}
Accept: application/json
```

Expected response `404 Not Found`:

```json
{
  "error": {
    "code": "NOT_FOUND",
    "message": "data wallet tidak ditemukan",
    "request_id": "generated-request-id"
  }
}
```

Result: PASS.

### Case 8.9 - Invalid Sort

Request:

```http
GET /api/v1/wallets?sort=-unknown_field
Authorization: Bearer {admin_access_token}
Accept: application/json
```

Expected response `400 Bad Request`:

```json
{
  "error": {
    "code": "INVALID_SORT",
    "message": "sort tidak valid",
    "request_id": "generated-request-id"
  }
}
```

Result: PASS.

### Case 8.10 - Invalid Date Filter

Request:

```http
GET /api/v1/wallet-payments?paid_from=24-07-2026
Authorization: Bearer {admin_access_token}
Accept: application/json
```

Expected response `400 Bad Request`:

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "paid_from harus format YYYY-MM-DD",
    "request_id": "generated-request-id"
  }
}
```

Result: PASS.

## 9. Actual Smoke Test Evidence

Final smoke test dijalankan pada API lokal port `18092` setelah patch generator code wallet.

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

## 10. Quality Gate

| Check | Result |
| --- | --- |
| `go test ./...` | PASS |
| `go vet ./...` | PASS |
| `go build .` | PASS |
| `go run . migrate up` | PASS |
| `go run . migrate down` lalu `go run . migrate up` | PASS |
| `go run . seed master` | PASS |
| `go run . seed demo --preset=minimal --seed=20260723 --as-of=2026-07-01` | PASS |
| Smoke test route Sprint 09 | PASS |

## 11. Defect Found During Testing

| Defect | Dampak | Perbaikan | Status |
| --- | --- | --- | --- |
| Kode payment/ledger awalnya memakai `paid_at` / `occurred_at`, sehingga request berbeda dengan tanggal sama dapat duplicate code. | Create top-up/debit bisa menghasilkan 500 karena unique constraint `code`. | Generator code API diubah memakai waktu pembuatan aktual `time.Now().UTC()`, sedangkan `paid_at` / `occurred_at` tetap menjadi tanggal bisnis. | CLOSED |

## 12. Conclusion

Sprint 09 berhasil menyediakan wallet ledger yang auditable untuk top-up dan saldo owner. Semua route utama, role access, idempotency, insufficient balance, dan error handler utama berhasil diuji. Revenue top-up dapat direkap berdasarkan `paid_at`, sementara closing tetap dipisahkan sebagai performance Sales agar tidak terjadi double counting.
