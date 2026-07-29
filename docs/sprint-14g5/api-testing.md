# API Testing Report - Sprint 14g5

## 1. Informasi Dokumen

| Item | Nilai |
| --- | --- |
| Project | Backend CRM Piposmart |
| Sprint | 14g5 |
| Area | Analytics API |
| Tanggal testing | 29 Juli 2026 |
| Environment | Local Development |
| Base URL | `http://localhost:8080/api/v1` |
| Status akhir | PASS |

## 2. Ringkasan Hasil

Fokus Sprint 14g5:

- import analytics;
- executive board;
- custom comparison board;
- cohort retention;
- forecast summary.

Rekap:

| Jumlah endpoint diuji | PASS | FAIL |
| ---: | ---: | ---: |
| 20 | 20 | 0 |

## 3. Daftar Endpoint yang Diuji

| Endpoint | Method | Status |
| --- | --- | --- |
| `/analytics/imports/batches-per-profile/query` | POST | PASS |
| `/analytics/imports/success-vs-failed/query` | POST | PASS |
| `/analytics/imports/invalid-rows-distribution/query` | POST | PASS |
| `/analytics/imports/validation-error-by-profile/query` | POST | PASS |
| `/analytics/imports/duplicate-detection-rate/query` | POST | PASS |
| `/analytics/imports/import-duration-trend/query` | POST | PASS |
| `/analytics/imports/batch-status-funnel/query` | POST | PASS |
| `/analytics/imports/uploader-activity/query` | POST | PASS |
| `/analytics/imports/file-history-usage/query` | POST | PASS |
| `/analytics/executive/end-to-end-funnel/query` | POST | PASS |
| `/analytics/executive/revenue-closing-active-subscription-board/query` | POST | PASS |
| `/analytics/executive/monthly-operating-review-board/query` | POST | PASS |
| `/analytics/executive/north-star-kpi-trend/query` | POST | PASS |
| `/analytics/executive/data-quality-score-by-module/query` | POST | PASS |
| `/analytics/custom/multi-series-trend/query` | POST | PASS |
| `/analytics/custom/metric-comparison-board/query` | POST | PASS |
| `/analytics/custom/region-comparison-board/query` | POST | PASS |
| `/analytics/subscriptions/cohort-retention/query` | POST | PASS |
| `/analytics/executive/forecast-summary-board/query` | POST | PASS |
| `/analytics/custom/comparison-impact-summary/query` | POST | PASS |

## 4. Contoh Pengujian Request dan Response

### 4.1 Contoh request sukses

```http
POST /api/v1/analytics/executive/north-star-kpi-trend/query
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
    "enabled": true,
    "mode": "previous_period"
  },
  "metrics": [
    "owner_count",
    "confirmed_closing_count",
    "topup_revenue",
    "active_subscription_count"
  ],
  "options": {
    "include_summary": true,
    "include_table": true
  }
}
```

### 4.2 Contoh response sukses

```json
{
  "data": {
    "comparison": {
      "enabled": true,
      "mode": "previous_period",
      "baseline_label": "Jun 2025 - Dec 2025",
      "current_value": 49750126,
      "baseline_value": 39750107,
      "delta": 10000019,
      "delta_percent": 25.16,
      "direction": "positive"
    },
    "table": [
      {
        "period": "2026-01",
        "owner_count": 3,
        "confirmed_closing_count": 1,
        "topup_revenue": 5750000,
        "active_subscription_count": 10
      }
    ],
    "insight": {
      "summary": "North Star KPI Trend membaik 25.16% dibanding baseline."
    }
  },
  "meta": {
    "request_id": "example-request-id"
  }
}
```

## 5. Verifikasi dan Validasi

- seluruh endpoint import, executive, custom compare, retention, dan forecast merespons `200 OK`;
- comparison `previous_period` berjalan;
- comparison `series_to_series` pada region comparison berjalan;
- endpoint executive summary merespons sesuai kontrak.

## 6. Bug yang Ditemukan dan Diperbaiki

| Endpoint | Masalah awal | Status akhir |
| --- | --- | --- |
| `imports/success-vs-failed` | agregasi `SUM(...)` bisa `NULL` | PASS |
| `imports/batch-status-funnel` | agregasi `SUM(...)` bisa `NULL` | PASS |
| `executive/data-quality-score-by-module` | agregasi kualitas data bisa `NULL` | PASS |
| `custom/region-comparison-board` | mode `series_to_series` sempat dipaksa ke baseline period | PASS |

## 7. Error Handler yang Diverifikasi

### 7.1 Comparison mode tidak didukung

Request salah:

```json
{
  "time_filter": {
    "mode": "month_range",
    "month_from": "2026-01",
    "month_to": "2026-07",
    "granularity": "month"
  },
  "comparison": {
    "enabled": true,
    "mode": "unsupported_mode"
  }
}
```

Response:

```json
{
  "error": {
    "code": "ANALYTICS_QUERY_ERROR",
    "message": "mode comparison tidak didukung",
    "request_id": "01581c692cacbe6db888e2f3918b4ca3"
  }
}
```

Solusi frontend:

- gunakan hanya mode compare yang terdaftar di metadata diagram.

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

- gunakan serializer JSON bawaan framework.

## 8. Catatan Tambahan

`imports/file-history-usage` saat ini lulus sebagai kontrak endpoint analytics. Namun data usage viewer/download masih bisa diperkaya lagi nanti bila backend menyimpan event histori yang lebih detail.

## 9. Kesimpulan

Analytics Sprint 14g5 lulus pengujian. Endpoint import, executive board, comparison board, retention, dan forecast siap dipakai frontend.
