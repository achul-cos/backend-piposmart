# Sprint 14b - API Change Notes dan Panduan Frontend

## 1. Informasi

| Item | Nilai |
| --- | --- |
| Project | Backend CRM Piposmart |
| Sprint | Sprint 14b |
| Tanggal | 27 Juli 2026 |
| Fokus | Trash/Unscoped listing, wallet semua owner, outlet global |
| Verifikasi | `go build ./...` dan `go test ./...` PASS |

Dokumen ini fokus pada perubahan API yang ditambahkan pada Sprint 14b. Contoh response di bawah adalah **contoh bentuk payload** agar frontend mudah integrasi.

## 2. Header

```http
Authorization: Bearer {access_token}
Accept: application/json
Content-Type: application/json
```

## 3. Ringkasan Perubahan

| Modul | Perubahan |
| --- | --- |
| Owner | Tambah route list `trash` dan `unscoped`. |
| Outlet (nested) | Tambah route list `trash` dan `unscoped` per owner. |
| Outlet (global) | Tambah route list semua outlet, trash outlet, unscoped outlet, dan detail outlet global. |
| Wallet | `GET /wallets` sekarang menampilkan **semua owner**, termasuk yang belum pernah top-up dengan saldo `0.00`. |
| Owner Wallet | `GET /owners/{owner_id}/wallet` sekarang tetap bisa mengembalikan wallet sintetis `0.00` bila owner belum pernah top-up. |
| Subscription | `GET /subscriptions` tambah query `outlet_id`. |
| Catalog | Tambah route list `trash` dan `unscoped` untuk packages, plans, promotions. |
| Closing | Tambah route list `trash` dan `unscoped`. |

## 4. Route Baru / Berubah

### 4.1 Owner Trash dan Unscoped

| Method | Path | Fungsi |
| --- | --- | --- |
| GET | `/api/v1/owners/trash` | Menampilkan owner yang soft delete. |
| GET | `/api/v1/owners/unscoped` | Menampilkan owner aktif + soft delete. |

Contoh:

```http
GET /api/v1/owners/trash?page=1&limit=10&sort=-updated_at
Authorization: Bearer {access_token}
Accept: application/json
```

Query tambahan yang didukung tetap sama dengan list owner biasa:
`q`, `code`, `name`, `phone`, `brand_name`, `province`, `city`, `page`, `limit`, `sort`.

### 4.2 Outlet Trash / Unscoped per Owner

| Method | Path | Fungsi |
| --- | --- | --- |
| GET | `/api/v1/owners/{owner_id}/outlets/trash` | Menampilkan outlet soft delete milik owner tertentu. |
| GET | `/api/v1/owners/{owner_id}/outlets/unscoped` | Menampilkan seluruh outlet milik owner tertentu. |

Contoh:

```http
GET /api/v1/owners/12/outlets/unscoped?sort=name
Authorization: Bearer {access_token}
Accept: application/json
```

### 4.3 Global Outlet

| Method | Path | Fungsi |
| --- | --- | --- |
| GET | `/api/v1/outlets` | Menampilkan semua outlet aktif lintas owner sesuai visibility actor. |
| GET | `/api/v1/outlets/trash` | Menampilkan outlet soft delete lintas owner. |
| GET | `/api/v1/outlets/unscoped` | Menampilkan semua outlet aktif + soft delete lintas owner. |
| GET | `/api/v1/outlets/{outlet_id}` | Detail outlet global berisi info owner, wallet owner, dan ringkasan subscription. |

Query list outlet global:

| Query | Contoh | Fungsi |
| --- | --- | --- |
| `q` | `Outlet` | Search outlet + owner ringkas. |
| `code` | `OUT-0001` | Filter kode outlet. |
| `name` | `Outlet Pusat` | Filter nama outlet. |
| `phone` | `0812` | Filter telepon outlet. |
| `province` | `Jawa Barat` | Filter provinsi. |
| `city` | `Bandung` | Filter kota. |
| `brand_name` | `Laundry Cerah` | Filter brand owner. |
| `owner_id` | `12` | Filter outlet milik owner tertentu. |
| `subscription_status` | `ACTIVE` | Filter outlet yang punya subscription dengan status tertentu. |
| `subscription_month` | `2026-07` | Filter outlet yang subscription-nya overlap dengan bulan tertentu. |
| `page` | `1` | Halaman. |
| `limit` | `10` | Jumlah data. |
| `sort` | `-created_at` | Sorting outlet. |

Contoh list:

```http
GET /api/v1/outlets?subscription_status=ACTIVE&subscription_month=2026-07&page=1&limit=10
Authorization: Bearer {access_token}
Accept: application/json
```

Contoh bentuk response:

```json
{
  "data": {
    "items": [
      {
        "id": 31,
        "owner": {
          "id": 12,
          "code": "OWN-00012",
          "name": "Laundry Cerah",
          "phone": "6281234567890",
          "email": "owner12@example.com",
          "brand_name": "Laundry Cerah"
        },
        "wallet": {
          "id": 5,
          "account_code": "WALLET-OWNER-000012",
          "currency": "IDR",
          "balance": "150000.00",
          "ledger_balance": "150000.00",
          "status": "ACTIVE"
        },
        "code": "OUT-00031",
        "name": "Laundry Cerah Outlet 1",
        "phone": "6281234567891",
        "province": "Jawa Barat",
        "city": "Bandung",
        "status": "ACTIVE",
        "subscription_summary": {
          "total_subscriptions": 3,
          "active_subscriptions": 1,
          "latest_subscription_status": "ACTIVE",
          "latest_subscription_start": "2026-07-01",
          "latest_subscription_end": "2027-06-26"
        }
      }
    ],
    "pagination": {
      "page": 1,
      "limit": 10,
      "total": 1
    }
  }
}
```

Contoh detail:

```http
GET /api/v1/outlets/31
Authorization: Bearer {access_token}
Accept: application/json
```

Response detail memiliki struktur yang sama untuk owner, wallet, outlet, dan `subscription_summary`, sehingga frontend detail outlet bisa langsung memakainya.

### 4.4 Wallet Semua Owner

Route utama tetap:

| Method | Path | Perubahan |
| --- | --- | --- |
| GET | `/api/v1/wallets` | Sekarang berbasis **semua owner**, bukan hanya owner yang sudah punya row `wallet_accounts`. |
| GET | `/api/v1/owners/{owner_id}/wallet` | Jika owner belum pernah top-up, tetap ada response wallet dengan saldo `0.00`. |

Contoh:

```http
GET /api/v1/wallets?page=1&limit=10&sort=-balance
Authorization: Bearer {access_token}
Accept: application/json
```

Contoh owner tanpa top-up:

```json
{
  "data": {
    "id": 0,
    "owner": {
      "id": 44,
      "code": "OWN-00044",
      "name": "Prospek Baru"
    },
    "account_code": "WALLET-OWNER-000044",
    "currency": "IDR",
    "balance": "0.00",
    "ledger_balance": "0.00",
    "status": "ACTIVE"
  }
}
```

### 4.5 Filter Subscription per Outlet

| Method | Path | Perubahan |
| --- | --- | --- |
| GET | `/api/v1/subscriptions` | Tambah query `outlet_id`. |

Contoh:

```http
GET /api/v1/subscriptions?outlet_id=31&status=ACTIVE
Authorization: Bearer {access_token}
Accept: application/json
```

### 4.6 Catalog Trash dan Unscoped

| Method | Path |
| --- | --- |
| GET | `/api/v1/catalog/packages/trash` |
| GET | `/api/v1/catalog/packages/unscoped` |
| GET | `/api/v1/catalog/plans/trash` |
| GET | `/api/v1/catalog/plans/unscoped` |
| GET | `/api/v1/catalog/promotions/trash` |
| GET | `/api/v1/catalog/promotions/unscoped` |

### 4.7 Closing Trash dan Unscoped

| Method | Path |
| --- | --- |
| GET | `/api/v1/closings/trash` |
| GET | `/api/v1/closings/unscoped` |

## 5. Error Case Penting

### Invalid path id

```http
GET /api/v1/outlets/abc
```

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "ID tidak valid"
  }
}
```

### Invalid sort

```http
GET /api/v1/outlets?sort=unknown_field
```

```json
{
  "error": {
    "code": "INVALID_SORT",
    "message": "sort tidak valid"
  }
}
```

### Resource tidak ditemukan

```http
GET /api/v1/outlets/999999
```

```json
{
  "error": {
    "code": "NOT_FOUND",
    "message": "data tidak ditemukan"
  }
}
```

### Forbidden

Contoh Sales mengakses route manajemen yang tidak diizinkan:

```http
GET /api/v1/closings/unscoped
Authorization: Bearer {sales_access_token}
```

Response mengikuti permission handler yang berlaku, misalnya:

```json
{
  "error": {
    "code": "FORBIDDEN",
    "message": "akses ditolak"
  }
}
```

## 6. Mapping Pemakaian Frontend

### Modul Owner

- Tabel Informasi Umum Owner: `GET /api/v1/owners`
- Tabel Sampah Owner: `GET /api/v1/owners/trash`
- Tabel Saldo Owner: `GET /api/v1/wallets`
- Detail Owner: `GET /api/v1/owners/{owner_id}`
- Tabel Outlet Owner: `GET /api/v1/owners/{owner_id}/outlets`
- Tabel Sampah Outlet Owner: `GET /api/v1/owners/{owner_id}/outlets/trash`

### Modul Outlet

- Tabel Informasi Umum Outlet: `GET /api/v1/outlets`
- Tabel Langganan Outlet (filter per bulan/status):
  - list ringkas: `GET /api/v1/outlets?subscription_status=ACTIVE&subscription_month=2026-07`
  - riwayat detail: `GET /api/v1/subscriptions?outlet_id={outlet_id}`
- Tabel Sampah Outlet: `GET /api/v1/outlets/trash`
- Detail Outlet: `GET /api/v1/outlets/{outlet_id}`
- Riwayat topup owner dari detail outlet: `GET /api/v1/wallet-payments?owner_id={owner_id}`
- Riwayat ledger owner dari detail outlet: `GET /api/v1/owners/{owner_id}/wallet/transactions`

## 7. Status Verifikasi

Per 27 Juli 2026:

- `go build ./...` -> PASS
- `go test ./...` -> PASS

Belum dilakukan manual smoke test HTTP pada dokumen ini; contoh request/response di atas disediakan sebagai panduan integrasi frontend dan pembacaan struktur payload.
