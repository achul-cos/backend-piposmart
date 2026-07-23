# API Testing Report - Sprint 10 Subscription, Order, dan Reconciliation

## 1. Informasi Pengujian

| Item | Nilai |
| --- | --- |
| Project | Backend CRM Piposmart |
| Sprint | Sprint 10 - Subscription, Order, dan Reconciliation |
| Tanggal Testing | 24 Juli 2026 |
| Environment | Local Development |
| API Base URL | `http://localhost:8080/api/v1` |
| Auth | JWT Bearer Token |
| Testing Tool | Manual smoke test via PowerShell `Invoke-RestMethod` / `Invoke-WebRequest` |
| Seeder | `go run . seed demo --preset=minimal --seed=20260723 --as-of=2026-07-01` |

## 2. Header Pengujian

Route protected wajib memakai JWT:

```http
Authorization: Bearer {access_token}
Accept: application/json
Content-Type: application/json
```

Akun demo yang digunakan:

| Role | Email | Password | Tujuan Pengujian |
| --- | --- | --- | --- |
| Admin | `admin.001@demo.piposmart.id` | `Password123!` | Create order, manual reconciliation, dan melihat seluruh data subscription. |
| Supervisor | `supervisor.001@demo.piposmart.id` | `Password123!` | Akses reconciliation sesuai cakupan ownership. |
| Sales | `sales.001@demo.piposmart.id` | `Password123!` | Negative test create order karena Sales tidak boleh membuat order subscription. |

Login:

```http
POST /api/v1/auth/login
Accept: application/json
Content-Type: application/json
```

```json
{
  "email": "admin.001@demo.piposmart.id",
  "password": "Password123!"
}
```

Response `200 OK`:

```json
{
  "data": {
    "access_token": "{jwt_access_token}",
    "refresh_token": "{refresh_token}",
    "token_type": "Bearer",
    "expires_in": 899,
    "user": {
      "id": 1,
      "email": "admin.001@demo.piposmart.id",
      "role": "ADMIN"
    }
  },
  "meta": {
    "request_id": "generated-request-id"
  }
}
```

## 3. Scope Pengujian

Sprint 10 memvalidasi alur pembelian paket dari saldo wallet owner sampai menjadi subscription aktif dan dapat direkonsiliasi dengan laporan closing Sales.

Hal yang divalidasi:

- Subscription order mendebit wallet, membuat ledger `DEBIT`, membuat subscription, dan membuat subscription period secara atomic.
- Durasi subscription memakai aturan fixed `tenure_months x 30 hari` dan tambahan benefit free duration bila ada.
- Order yang dibuat dari closing dapat auto reconciliation dan mengubah closing menjadi `CONFIRMED`.
- Order tanpa closing masuk reconciliation issue queue sebagai `HANGING_ORDER`.
- Manual reconciliation dapat menghubungkan order dengan closing dan mencatat `amount_difference`.
- Revenue top-up tetap berdasarkan `wallet_payments.paid_at`; pembelian subscription tidak dihitung sebagai revenue baru agar tidak double counting.
- Idempotency wajib pada create order.
- Saldo wallet tidak boleh negatif.
- Admin-only untuk create order; Admin/Supervisor untuk reconciliation.

## 4. Testing Summary

| Module | Endpoint / Case | Method | Expected | Result |
| --- | --- | --- | --- | --- |
| Auth | `/auth/login` Admin | POST | 200 OK | PASS |
| Subscription Order | `/subscription-orders` | GET | 200 OK | PASS |
| Subscription Order | `/subscription-orders/{order_id}` | GET | 200 OK | PASS |
| Subscription Order | `/owners/{owner_id}/subscription-orders` | POST | 201 Created | PASS |
| Subscription Order | Repeat create dengan idempotency sama | POST | 201 Created, `idempotent=true` | PASS |
| Reconciliation | `/subscription-orders/{order_id}/reconcile` manual confirm | POST | 200 OK | PASS |
| Subscription | `/subscriptions` | GET | 200 OK | PASS |
| Subscription | `/subscriptions/{subscription_id}` | GET | 200 OK | PASS |
| Reconciliation | `/reconciliations` | GET | 200 OK | PASS |
| Reconciliation Issue | `/reconciliation-issues?status=OPEN` | GET | 200 OK | PASS |
| Error Handling | Tanpa JWT | GET | 401 UNAUTHENTICATED | PASS |
| Error Handling | Invalid sort | GET | 400 INVALID_SORT | PASS |
| Error Handling | Order tidak ditemukan | GET | 404 NOT_FOUND | PASS |
| Error Handling | Missing idempotency | POST | 400 IDEMPOTENCY_REQUIRED | PASS |
| Error Handling | Saldo tidak cukup | POST | 400 INSUFFICIENT_BALANCE | PASS |
| Error Handling | Sales membuat order | POST | 403 FORBIDDEN | PASS |
| Error Handling | Reconcile order yang sudah reconciled | POST | 409 ORDER_ALREADY_RECONCILED | PASS |
| Error Handling | Format tanggal invalid | GET | 400 VALIDATION_ERROR | PASS |

## 5. Route Coverage Sprint 10

| Nama Route | Method | Path | Fungsi | Coverage |
| --- | --- | --- | --- | --- |
| List Subscription Order | GET | `/api/v1/subscription-orders` | Menampilkan pembelian paket subscription dari wallet owner. | Success + filter/sort + invalid sort. |
| Detail Subscription Order | GET | `/api/v1/subscription-orders/{order_id}` | Detail order, snapshot paket/plan/promo, wallet transaction, dan status reconciliation. | Success + not found. |
| Create Owner Subscription Order | POST | `/api/v1/owners/{owner_id}/subscription-orders` | Admin membuat pembelian paket dari saldo wallet owner. | Success + idempotency + saldo tidak cukup + RBAC. |
| Manual Reconcile Order | POST | `/api/v1/subscription-orders/{order_id}/reconcile` | Admin/Supervisor confirm/reject order terhadap closing. | Success manual confirm + already reconciled error. |
| List Subscription | GET | `/api/v1/subscriptions` | Melihat subscription owner aktif/expired/cancelled. | Success + date filter error. |
| Detail Subscription | GET | `/api/v1/subscriptions/{subscription_id}` | Detail subscription aktif dan period. | Success. |
| List Reconciliation | GET | `/api/v1/reconciliations` | Melihat hasil auto/manual reconciliation order dengan closing. | Success. |
| List Reconciliation Issue | GET | `/api/v1/reconciliation-issues` | Melihat hanging transaction atau issue mismatch yang perlu review. | Success + filter status. |

## 6. Query Parameter

### 6.1 List Subscription Order

Endpoint:

```http
GET /api/v1/subscription-orders
Authorization: Bearer {access_token}
Accept: application/json
```

| Query | Contoh | Fungsi |
| --- | --- | --- |
| `q` | `Owner Laundry` | Search kode order, kode owner, nama owner, atau external reference. |
| `owner_id` | `4` | Filter order berdasarkan owner. |
| `closing_id` | `9` | Filter order yang terhubung dengan closing tertentu. |
| `sales_id` | `3` | Filter order berdasarkan Sales dari closing/lead. |
| `supervisor_id` | `2` | Filter order berdasarkan Supervisor dari closing/lead. |
| `plan_id` | `1` | Filter order berdasarkan subscription plan. |
| `status` | `RECONCILED` | Filter status: `PENDING_RECONCILIATION`, `PAID`, `RECONCILED`, `REJECTED`. |
| `purchased_from` | `2026-07-01` | Batas awal tanggal pembelian. |
| `purchased_to` | `2026-07-31` | Batas akhir tanggal pembelian. |
| `sort` | `-purchased_at` | Sort by `purchased_at`, `created_at`, `updated_at`, `final_amount`, atau `code`. Prefix `-` berarti descending. |
| `page`, `limit` | `1`, `10` | Pagination. |

Contoh:

```http
GET /api/v1/subscription-orders?status=RECONCILED&purchased_from=2026-07-01&purchased_to=2026-07-31&sort=-purchased_at&page=1&limit=10
Authorization: Bearer {admin_access_token}
Accept: application/json
```

### 6.2 List Subscription

Endpoint:

```http
GET /api/v1/subscriptions
Authorization: Bearer {access_token}
Accept: application/json
```

| Query | Contoh | Fungsi |
| --- | --- | --- |
| `q` | `OWN-00004` | Search kode subscription, kode owner, atau nama owner. |
| `owner_id` | `4` | Filter subscription owner tertentu. |
| `order_id` | `3` | Filter subscription dari order tertentu. |
| `plan_id` | `1` | Filter berdasarkan plan. |
| `status` | `ACTIVE` | Filter status subscription: `ACTIVE`, `EXPIRED`, `CANCELED`. |
| `active_from` | `2026-07-01` | Filter subscription aktif mulai tanggal tertentu. |
| `active_to` | `2026-07-31` | Filter subscription aktif sampai tanggal tertentu. |
| `sort` | `-active_from` | Sort by `active_from`, `active_until`, `created_at`, `updated_at`, atau `code`. |
| `page`, `limit` | `1`, `10` | Pagination. |

### 6.3 List Reconciliation

Endpoint:

```http
GET /api/v1/reconciliations
Authorization: Bearer {access_token}
Accept: application/json
```

| Query | Contoh | Fungsi |
| --- | --- | --- |
| `q` | `DEMO-REC` | Search kode reconciliation, kode order, kode closing, atau nama owner. |
| `order_id` | `1` | Filter berdasarkan order. |
| `closing_id` | `9` | Filter berdasarkan closing. |
| `owner_id` | `3` | Filter berdasarkan owner. |
| `status` | `CONFIRMED` | Filter status: `PENDING`, `CONFIRMED`, `REJECTED`. |
| `sort` | `-created_at` | Sort by `created_at`, `updated_at`, `confirmed_at`, `amount_difference`, atau `code`. |
| `page`, `limit` | `1`, `10` | Pagination. |

### 6.4 List Reconciliation Issue

Endpoint:

```http
GET /api/v1/reconciliation-issues
Authorization: Bearer {access_token}
Accept: application/json
```

| Query | Contoh | Fungsi |
| --- | --- | --- |
| `q` | `HANGING` | Search kode issue, kode order, atau nama owner. |
| `order_id` | `2` | Filter issue order tertentu. |
| `owner_id` | `4` | Filter issue berdasarkan owner. |
| `issue_type` | `HANGING_ORDER` | Filter tipe issue: `HANGING_ORDER`, `CLOSING_MISMATCH`, `MANUAL_REVIEW`. |
| `status` | `OPEN` | Filter status issue: `OPEN`, `RESOLVED`. |
| `sort` | `-detected_at` | Sort by `detected_at`, `created_at`, `updated_at`, atau `code`. |
| `page`, `limit` | `1`, `10` | Pagination. |

## 7. Success Case

### 7.1 GET `/subscription-orders`

Request:

```http
GET /api/v1/subscription-orders?sort=-purchased_at&limit=5
Authorization: Bearer {admin_access_token}
Accept: application/json
```

Response `200 OK`:

```json
{
  "data": {
    "items": [
      {
        "id": 1,
        "code": "DEMO-ORD-000003-RDER-APRIL-TOPUP-JULY-PURCHASE-OWNER-003",
        "owner": {
          "id": 3,
          "code": "OWN-00003",
          "name": "Owner Laundry 003"
        },
        "closing": {
          "id": 9,
          "code": "DEMO-CLS-000003-PRO_12_MONTHS"
        },
        "plan": {
          "id": 13,
          "code": "PRO_12_MONTHS",
          "name": "Pro 12 Bulan"
        },
        "promotion": {
          "id": 3,
          "code": "PRO_12_ANDROID_POS_BUNDLE",
          "name": "Pro 12 Bulan Bonus Alat POS"
        },
        "duration_days": 360,
        "base_price": "2268600.00",
        "additional_charge": "1500000.00",
        "final_amount": "3768703.00",
        "status": "RECONCILED",
        "purchased_at": "2026-07-10T13:00:00Z",
        "subscription_start_date": "2026-07-10"
      }
    ],
    "pagination": {
      "page": 1,
      "limit": 5,
      "total": 4
    }
  },
  "meta": {
    "request_id": "generated-request-id"
  }
}
```

Catatan: `final_amount` pada order yang berasal dari closing memakai snapshot transaksi sehingga tidak berubah walaupun master package/promo diedit.

### 7.2 POST `/owners/{owner_id}/subscription-orders`

Request success tanpa closing:

```http
POST /api/v1/owners/4/subscription-orders
Authorization: Bearer {admin_access_token}
Accept: application/json
Content-Type: application/json
```

```json
{
  "plan_id": 1,
  "idempotency_key": "smoke:sprint10:create-owner-004-basic-001",
  "external_reference": "SMOKE-SPRINT10-OWNER004-BASIC001",
  "purchased_at": "2026-07-24T03:00:00Z",
  "subscription_start_date": "2026-07-24",
  "note": "Smoke test Sprint 10 create subscription order tanpa closing"
}
```

Response `201 Created`:

```json
{
  "data": {
    "order": {
      "id": 3,
      "owner": {
        "id": 4,
        "code": "OWN-00004",
        "name": "Owner Laundry 004"
      },
      "wallet_transaction_id": 13,
      "plan": {
        "id": 1,
        "code": "BASIC_01_MONTHS",
        "name": "Basic 1 Bulan"
      },
      "tenure_months": 1,
      "duration_days": 30,
      "final_amount": "99000.00",
      "status": "PAID",
      "idempotency_key": "subscription_order:smoke:sprint10:create-owner-004-basic-001"
    },
    "subscription": {
      "id": 3,
      "status": "ACTIVE",
      "active_from": "2026-07-24",
      "active_until": "2026-08-23",
      "total_duration_days": 30
    },
    "period": {
      "period_index": 1,
      "start_date": "2026-07-24",
      "end_date": "2026-08-23",
      "duration_days": 30
    },
    "issue": {
      "issue_type": "HANGING_ORDER",
      "status": "OPEN"
    },
    "idempotent": false
  },
  "meta": {
    "request_id": "generated-request-id"
  }
}
```

Request yang sama jika dikirim ulang:

```json
{
  "data": {
    "order": {
      "id": 3,
      "status": "PAID"
    },
    "issue": {
      "issue_type": "HANGING_ORDER"
    },
    "idempotent": true
  },
  "meta": {
    "request_id": "generated-request-id"
  }
}
```

### 7.3 POST `/subscription-orders/{order_id}/reconcile`

Request manual confirm:

```http
POST /api/v1/subscription-orders/4/reconcile
Authorization: Bearer {admin_access_token}
Accept: application/json
Content-Type: application/json
```

```json
{
  "action": "CONFIRM",
  "closing_id": 7,
  "note": "Smoke test manual confirm order dengan closing pending"
}
```

Response `200 OK`:

```json
{
  "data": {
    "order": {
      "id": 4,
      "status": "RECONCILED"
    },
    "reconciliation": {
      "status": "CONFIRMED",
      "match_type": "MANUAL",
      "amount_difference": "-101.00"
    }
  },
  "meta": {
    "request_id": "generated-request-id"
  }
}
```

### 7.4 GET `/subscriptions/{subscription_id}`

Request:

```http
GET /api/v1/subscriptions/1
Authorization: Bearer {admin_access_token}
Accept: application/json
```

Response `200 OK`:

```json
{
  "data": {
    "id": 1,
    "code": "DEMO-SUB-000001",
    "owner": {
      "id": 3,
      "code": "OWN-00003",
      "name": "Owner Laundry 003"
    },
    "order": {
      "id": 1,
      "code": "DEMO-ORD-000003-RDER-APRIL-TOPUP-JULY-PURCHASE-OWNER-003"
    },
    "plan": {
      "id": 13,
      "code": "PRO_12_MONTHS",
      "name": "Pro 12 Bulan"
    },
    "status": "ACTIVE",
    "active_from": "2026-07-10",
    "active_until": "2027-07-05",
    "total_duration_days": 360
  },
  "meta": {
    "request_id": "generated-request-id"
  }
}
```

### 7.5 GET `/reconciliations`

Request:

```http
GET /api/v1/reconciliations?status=CONFIRMED&limit=5
Authorization: Bearer {admin_access_token}
Accept: application/json
```

Response `200 OK`:

```json
{
  "data": {
    "items": [
      {
        "id": 1,
        "code": "DEMO-REC-000001",
        "order": {
          "id": 1
        },
        "closing": {
          "id": 9
        },
        "owner": {
          "id": 3,
          "name": "Owner Laundry 003"
        },
        "status": "CONFIRMED",
        "match_type": "AUTO",
        "amount_difference": "0.00",
        "confirmed_at": "2026-07-10T13:00:00Z"
      }
    ],
    "pagination": {
      "page": 1,
      "limit": 5,
      "total": 2
    }
  },
  "meta": {
    "request_id": "generated-request-id"
  }
}
```

### 7.6 GET `/reconciliation-issues`

Request:

```http
GET /api/v1/reconciliation-issues?status=OPEN&limit=5
Authorization: Bearer {admin_access_token}
Accept: application/json
```

Response `200 OK`:

```json
{
  "data": {
    "items": [
      {
        "id": 1,
        "code": "DEMO-RIS-000002",
        "order": {
          "id": 2
        },
        "owner": {
          "id": 4,
          "code": "OWN-00004",
          "name": "Owner Laundry 004"
        },
        "issue_type": "HANGING_ORDER",
        "status": "OPEN",
        "description": "Demo hanging order: order subscription belum terhubung dengan closing.",
        "detected_at": "2026-07-12T13:00:00Z"
      }
    ],
    "pagination": {
      "page": 1,
      "limit": 5,
      "total": 2
    }
  },
  "meta": {
    "request_id": "generated-request-id"
  }
}
```

## 8. Error Handling Cases

### 8.1 Tanpa JWT

Request:

```http
GET /api/v1/subscription-orders
Accept: application/json
```

Response `401 Unauthorized`:

```json
{
  "error": {
    "code": "UNAUTHENTICATED",
    "message": "Token akses wajib dikirim",
    "request_id": "generated-request-id"
  }
}
```

### 8.2 Invalid Sort

Request:

```http
GET /api/v1/subscription-orders?sort=unknown
Authorization: Bearer {admin_access_token}
Accept: application/json
```

Response `400 Bad Request`:

```json
{
  "error": {
    "code": "INVALID_SORT",
    "message": "sort tidak valid",
    "request_id": "generated-request-id"
  }
}
```

### 8.3 Order Tidak Ditemukan

Request:

```http
GET /api/v1/subscription-orders/999999
Authorization: Bearer {admin_access_token}
Accept: application/json
```

Response `404 Not Found`:

```json
{
  "error": {
    "code": "NOT_FOUND",
    "message": "data subscription tidak ditemukan",
    "request_id": "generated-request-id"
  }
}
```

### 8.4 Missing Idempotency

Request:

```http
POST /api/v1/owners/4/subscription-orders
Authorization: Bearer {admin_access_token}
Accept: application/json
Content-Type: application/json
```

```json
{
  "plan_id": 1,
  "purchased_at": "2026-07-20T10:00:00Z",
  "subscription_start_date": "2026-07-20"
}
```

Response `400 Bad Request`:

```json
{
  "error": {
    "code": "IDEMPOTENCY_REQUIRED",
    "message": "idempotency_key atau external_reference wajib dikirim",
    "request_id": "generated-request-id"
  }
}
```

### 8.5 Saldo Tidak Cukup

Request:

```http
POST /api/v1/owners/4/subscription-orders
Authorization: Bearer {admin_access_token}
Accept: application/json
Content-Type: application/json
```

```json
{
  "plan_id": 13,
  "idempotency_key": "smoke:sprint10:insufficient-balance-owner004",
  "purchased_at": "2026-07-24T04:00:00Z",
  "subscription_start_date": "2026-07-24"
}
```

Response `400 Bad Request`:

```json
{
  "error": {
    "code": "INSUFFICIENT_BALANCE",
    "message": "saldo wallet tidak mencukupi",
    "request_id": "generated-request-id"
  }
}
```

### 8.6 Sales Tidak Boleh Membuat Order

Request:

```http
POST /api/v1/owners/4/subscription-orders
Authorization: Bearer {sales_access_token}
Accept: application/json
Content-Type: application/json
```

```json
{
  "plan_id": 1,
  "idempotency_key": "smoke:sprint10:sales-forbidden",
  "purchased_at": "2026-07-24T04:30:00Z",
  "subscription_start_date": "2026-07-24"
}
```

Response `403 Forbidden`:

```json
{
  "error": {
    "code": "FORBIDDEN",
    "message": "akses ditolak",
    "request_id": "generated-request-id"
  }
}
```

### 8.7 Reconcile Order yang Sudah Reconciled

Request:

```http
POST /api/v1/subscription-orders/1/reconcile
Authorization: Bearer {admin_access_token}
Accept: application/json
Content-Type: application/json
```

```json
{
  "action": "CONFIRM",
  "closing_id": 9,
  "note": "negative test already reconciled"
}
```

Response `409 Conflict`:

```json
{
  "error": {
    "code": "ORDER_ALREADY_RECONCILED",
    "message": "order sudah direkonsiliasi",
    "request_id": "generated-request-id"
  }
}
```

### 8.8 Format Tanggal Invalid

Request:

```http
GET /api/v1/subscriptions?active_from=24-07-2026
Authorization: Bearer {admin_access_token}
Accept: application/json
```

Response `400 Bad Request`:

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "active_from harus format YYYY-MM-DD",
    "request_id": "generated-request-id"
  }
}
```

## 9. Demo Scenario dari Seeder

Seeder minimal Sprint 10 membuat skenario utama roadmap:

| Scenario | Data | Expected |
| --- | --- | --- |
| Top-up April, pembelian Juli | Owner 3 top-up April, closing dan pembelian Juli paket Pro 12 bulan + promo alat POS | Revenue top-up masuk April; performa Sales masuk Juli setelah reconciliation confirmed; tidak ada revenue ganda pada Juli. |
| Hanging order | Owner 4 top-up April lalu beli Basic 1 bulan tanpa closing | Order `PAID`, subscription aktif, dan muncul issue `HANGING_ORDER` untuk review. |
| Manual reconciliation smoke | Owner 1 order Basic 1 bulan direconcile ke closing pending | Order menjadi `RECONCILED`, reconciliation `CONFIRMED`, match type `MANUAL`, amount difference tercatat. |

Summary smoke terakhir:

```text
orders=4
subscriptions=4
reconciliations=2
openIssues=2
latestOrderStatus=RECONCILED
latestOrderAmount=99000.00
```

## 10. Conclusion

Berdasarkan manual smoke test, seluruh endpoint Sprint 10 yang diuji berjalan sesuai ekspektasi.

Summary:

- Subscription Order: PASS
- Subscription Activation: PASS
- Wallet Debit Atomic Flow: PASS
- Auto Reconciliation dari closing seed: PASS
- Manual Reconciliation: PASS
- Hanging Issue Queue: PASS
- RBAC dan Error Handling: PASS

Overall API Testing Status: PASS