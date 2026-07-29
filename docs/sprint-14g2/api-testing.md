# API Testing Report - Sprint 14g2

## 1. Informasi Dokumen

| Item | Nilai |
| --- | --- |
| Project | Backend CRM Piposmart |
| Sprint | 14g2 |
| Area | Analytics API |
| Tanggal testing | 29 Juli 2026 |
| Environment | Local Development |
| Base URL | `http://localhost:8080/api/v1` |
| Status akhir | PASS |

## 2. Ringkasan Hasil

Fokus Sprint 14g2:

- training conversion;
- package, plan, promo, dan histori harga;
- closing analytics;
- target analytics;
- KPI analytics.

Rekap:

| Jumlah endpoint diuji | PASS | FAIL |
| ---: | ---: | ---: |
| 20 | 20 | 0 |

## 3. Header dan Payload Baseline

```http
Content-Type: application/json
Authorization: Bearer {access_token}
Accept: application/json
```

```json
{
  "time_filter": {
    "mode": "month_range",
    "month_from": "2026-01",
    "month_to": "2026-07",
    "granularity": "month"
  },
  "comparison": {
    "enabled": false
  },
  "metrics": [
    "confirmed_closing_count"
  ],
  "options": {
    "include_summary": true,
    "include_table": true
  }
}
```

## 4. Daftar Endpoint yang Diuji

| Endpoint | Method | Status |
| --- | --- | --- |
| `/analytics/trainings/training-to-closing-conversion/query` | POST | PASS |
| `/analytics/catalog/package-popularity/query` | POST | PASS |
| `/analytics/catalog/tenure-popularity/query` | POST | PASS |
| `/analytics/catalog/package-tenure-heatmap/query` | POST | PASS |
| `/analytics/catalog/promo-adoption-rate/query` | POST | PASS |
| `/analytics/catalog/additional-charge-adoption/query` | POST | PASS |
| `/analytics/catalog/price-history-timeline/query` | POST | PASS |
| `/analytics/catalog/promotion-history-timeline/query` | POST | PASS |
| `/analytics/closings/closing-trend/query` | POST | PASS |
| `/analytics/closings/closing-by-sales/query` | POST | PASS |
| `/analytics/closings/closing-by-supervisor/query` | POST | PASS |
| `/analytics/closings/closing-by-package/query` | POST | PASS |
| `/analytics/closings/closing-by-tenure/query` | POST | PASS |
| `/analytics/closings/status-distribution/query` | POST | PASS |
| `/analytics/closings/average-ticket-size-trend/query` | POST | PASS |
| `/analytics/closings/closing-amount-waterfall/query` | POST | PASS |
| `/analytics/targets/target-vs-actual/query` | POST | PASS |
| `/analytics/targets/target-burnup/query` | POST | PASS |
| `/analytics/kpi/leaderboard/query` | POST | PASS |
| `/analytics/kpi/activity-vs-closing-scatter/query` | POST | PASS |

## 5. Contoh Pengujian Request dan Response

### 5.1 Contoh request sukses

```http
POST /api/v1/analytics/closings/closing-by-sales/query
Authorization: Bearer {admin_token}
Content-Type: application/json
Accept: application/json
```

```json
{
  "time_filter": {
    "mode": "month_range",
    "month_from": "2026-01",
    "month_to": "2026-07",
    "granularity": "month"
  },
  "comparison": {
    "enabled": false
  },
  "metrics": [
    "confirmed_closing_count"
  ],
  "options": {
    "include_summary": true,
    "include_table": true
  }
}
```

### 5.2 Contoh response sukses

```json
{
  "data": {
    "series": [
      {
        "key": "confirmed_closing_count",
        "label": "Confirmed Closing Count",
        "points": [
          { "x": "Sales Demo 002", "y": 3 },
          { "x": "Sales Demo 003", "y": 3 },
          { "x": "Sales Demo 091", "y": 3 }
        ]
      }
    ],
    "table": [
      { "sales_name": "Sales Demo 002", "confirmed_closing_count": 3 },
      { "sales_name": "Sales Demo 003", "confirmed_closing_count": 3 },
      { "sales_name": "Sales Demo 091", "confirmed_closing_count": 3 }
    ]
  },
  "meta": {
    "request_id": "example-request-id"
  }
}
```

## 6. Verifikasi dan Validasi

- seluruh endpoint catalog, closing, target, dan KPI merespons `200 OK`;
- chart leaderboard dan scatter KPI merespons normal;
- histori harga dan histori promosi dapat dibuka;
- top 3 closing sales berhasil terbaca dari hasil query.

## 7. Error Handler yang Diverifikasi

### 7.1 Diagram tidak ditemukan

Request:

```http
POST /api/v1/analytics/catalog/not-found/query
Authorization: Bearer {access_token}
Content-Type: application/json
```

Response:

```json
{
  "error": {
    "code": "NOT_FOUND",
    "message": "diagram tidak ditemukan",
    "request_id": "6bb3fb5d7d47d4fccde7c28ca18cc37e"
  }
}
```

Solusi frontend:

- gunakan catalog sebagai sumber daftar diagram, jangan hardcode key yang belum terdaftar.

### 7.2 JSON request rusak

Request salah:

```json
{"time_filter":
```

Response:

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Payload request tidak valid",
    "details": {
      "error": "unexpected EOF"
    },
    "request_id": "f9334edc26248e059898dea567017e9e"
  }
}
```

Solusi frontend:

- gunakan serializer JSON resmi dari framework.

## 8. Kesimpulan

Analytics Sprint 14g2 lulus pengujian. Endpoint sales, closing, target, KPI, dan catalog analytics siap dipakai frontend.
