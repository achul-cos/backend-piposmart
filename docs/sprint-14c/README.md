# Sprint 14c - Rekap Status Langganan Outlet per Bulan

## 1. Informasi

| Item | Nilai |
| --- | --- |
| Project | Backend CRM Piposmart |
| Sprint | Sprint 14c |
| Tanggal | 27 Juli 2026 |
| Fokus | Rekap tabel langganan outlet bulanan |
| Verifikasi | `go build ./...` PASS, `go test ./...` PASS |

Dokumen ini menjelaskan endpoint backend baru untuk tabel langganan outlet bulanan. Fokus endpoint ini adalah **rekapan status per akhir bulan filter**, bukan menyesuaikan struktur frontend secara langsung.

## 2. Endpoint Baru

| Method | Path | Fungsi |
| --- | --- | --- |
| GET | `/api/v1/outlets/subscription-statuses` | Menampilkan seluruh outlet aktif beserta status langganan bulanannya, termasuk outlet yang belum pernah subscribe. |

## 3. Header

```http
Authorization: Bearer {access_token}
Accept: application/json
```

## 4. Query Parameters

| Query | Contoh | Wajib | Fungsi |
| --- | --- | --- | --- |
| `month` | `2026-06` | tidak | Bulan rekap dalam format `YYYY-MM`. Default bulan saat ini. |
| `subscription_status` | `EXPIRED` | tidak | Filter status langganan bulanan. |
| `q` | `Outlet` | tidak | Search outlet + owner. |
| `code` | `OUT-0001` | tidak | Filter kode outlet. |
| `name` | `Outlet Pusat` | tidak | Filter nama outlet. |
| `phone` | `0812` | tidak | Filter telepon outlet. |
| `brand_name` | `Laundry Cerah` | tidak | Filter brand owner. |
| `province` | `Jawa Barat` | tidak | Filter provinsi outlet. |
| `city` | `Bandung` | tidak | Filter kota outlet. |
| `owner_id` | `12` | tidak | Filter owner tertentu. |
| `page` | `1` | tidak | Halaman data. |
| `limit` | `10` | tidak | Jumlah data per halaman. |
| `sort` | `subscription_end_date` | tidak | Sorting. Field: `created_at`, `updated_at`, `code`, `name`, `city`, `province`, `subscription_start_date`, `subscription_end_date`. |

## 5. Aturan Rekap Bulanan

Status dihitung berdasarkan **akhir bulan filter**, bukan berdasarkan hari saat request dikirim.

Contoh:

- jika `month=2026-06`, maka tanggal acuan rekap adalah `30 Juni 2026`
- jika `month=2026-07`, maka tanggal acuan rekap adalah `31 Juli 2026`

Endpoint akan mengambil **subscription terakhir yang mulai berlaku pada atau sebelum akhir bulan filter** untuk setiap outlet.

### 5.1 NOT SUBSCRIBE

Status ini berlaku bila:

- outlet belum pernah subscribe sama sekali, atau
- subscription terakhir sudah berakhir lebih dari 60 hari sebelum akhir bulan filter

Perilaku response:

- jika belum pernah subscribe:
  - `remaining_days = null`
  - `remaining_days_display = "TIDAK PERNAH"`
  - `last_subscription_end_display = "TIDAK PERNAH"`
- jika pernah subscribe namun sudah lama sekali habis:
  - `remaining_days` bernilai negatif

### 5.2 EXPIRED

Status ini berlaku bila subscription terakhir berakhir sebelum bulan filter, tetapi masih berada dalam jendela 60 hari ke belakang dari akhir bulan filter.

Contoh `month=2026-06`:

- akhir bulan: `2026-06-30`
- subscription end antara `2026-05-01` sampai sebelum `2026-06-01` ? `EXPIRED`

### 5.3 JATUH TEMPO

Status ini berlaku bila subscription terakhir berakhir **di dalam bulan filter**.

Contoh `month=2026-06`:

- subscription end `2026-06-01` s.d. `2026-06-30` ? `JATUH TEMPO`

### 5.4 BERLANGGANAN

Status ini berlaku bila outlet masih berada dalam masa langganan normal, dan tidak termasuk `NEW` maupun pengecualian paket 1 bulan.

### 5.5 NEW

Status ini berlaku bila tanggal mulai subscription berada dalam 30 hari terakhir menuju akhir bulan filter.

### 5.6 Perilaku Khusus Paket 1 Bulan

Jika outlet mengambil paket 1 bulan dan secara logika memenuhi kondisi `NEW` sekaligus `JATUH TEMPO`, endpoint akan mengembalikan:

- `subscription_status_code = "BERLANGGANAN"`
- `subscription_status_label = "BERLANGGANAN 1 BULAN"`

Jadi data ini tetap dianggap sedang berlangganan, tetapi diberi label khusus agar mudah dibedakan.

## 6. Contoh Request

### 6.1 Rekap default bulan berjalan

```http
GET /api/v1/outlets/subscription-statuses?page=1&limit=10
Authorization: Bearer {access_token}
Accept: application/json
```

### 6.2 Rekap bulan Juni 2026

```http
GET /api/v1/outlets/subscription-statuses?month=2026-06&page=1&limit=10&sort=subscription_end_date
Authorization: Bearer {access_token}
Accept: application/json
```

### 6.3 Filter hanya outlet expired bulan Juni 2026

```http
GET /api/v1/outlets/subscription-statuses?month=2026-06&subscription_status=EXPIRED
Authorization: Bearer {access_token}
Accept: application/json
```

## 7. Contoh Response

```json
{
  "data": {
    "reference_month": "2026-06",
    "reference_month_start": "2026-06-01",
    "reference_month_end": "2026-06-30",
    "items": [
      {
        "outlet_id": 31,
        "outlet_code": "OUT-00031",
        "outlet_name": "Laundry Cerah Outlet 1",
        "outlet_phone": "6281234567891",
        "outlet_province": "Jawa Barat",
        "outlet_city": "Bandung",
        "owner": {
          "id": 12,
          "code": "OWN-00012",
          "name": "Laundry Cerah",
          "phone": "6281234567890",
          "brand_name": "Laundry Cerah"
        },
        "subscription_status_code": "JATUH_TEMPO",
        "subscription_status_label": "JATUH TEMPO",
        "remaining_days": -10,
        "remaining_days_display": "-10",
        "last_subscription_end": "2026-06-20",
        "last_subscription_end_display": "2026-06-20",
        "subscription_start_date": "2026-03-20",
        "subscription_end_date": "2026-06-20",
        "package_plan": {
          "package_id": 2,
          "package_code": "BUSINESS",
          "package_name": "Business",
          "plan_id": 8,
          "plan_code": "BUSINESS-3M",
          "plan_name": "Business 3 Bulan",
          "tenure_months": 3
        }
      },
      {
        "outlet_id": 77,
        "outlet_code": "OUT-00077",
        "outlet_name": "Outlet Baru",
        "owner": {
          "id": 44,
          "code": "OWN-00044",
          "name": "Prospek Baru"
        },
        "subscription_status_code": "NOT_SUBSCRIBE",
        "subscription_status_label": "NOT_SUBSCRIBE",
        "remaining_days_display": "TIDAK PERNAH",
        "last_subscription_end_display": "TIDAK PERNAH",
        "package_plan": {}
      }
    ],
    "pagination": {
      "page": 1,
      "limit": 10,
      "total": 2
    }
  }
}
```

## 8. Error Cases

### Format bulan salah

```http
GET /api/v1/outlets/subscription-statuses?month=2026/06
```

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "month harus format YYYY-MM"
  }
}
```

### Sort tidak valid

```http
GET /api/v1/outlets/subscription-statuses?sort=unknown_field
```

```json
{
  "error": {
    "code": "INVALID_SORT",
    "message": "sort tidak valid"
  }
}
```

## 9. Catatan Implementasi

- endpoint ini dibuat additive sebagai route baru, agar endpoint outlet global yang sudah ada tidak diubah perilakunya secara besar
- data dasar diambil dari outlet aktif yang visible bagi actor
- subscription yang dipakai adalah subscription terakhir yang `active_from <= akhir bulan filter`
- frontend dapat menyesuaikan tampilan label/status dari payload ini tanpa backend perlu mengikuti struktur tabel frontend secara ketat

## 10. Verifikasi

Per 27 Juli 2026:

- `go build ./...` -> PASS
- `go test ./...` -> PASS
