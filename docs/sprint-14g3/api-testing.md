# API Testing Report - Sprint 14g3

## 1. Informasi Dokumen

| Item | Nilai |
| --- | --- |
| Project | Backend CRM Piposmart |
| Sprint | 14g3 |
| Area | Analytics API |
| Tanggal testing | 29 Juli 2026 |
| Environment | Local Development |
| Base URL | `http://localhost:8080/api/v1` |
| Status akhir | PASS |

## 2. Ringkasan Hasil

Fokus Sprint 14g3:

- wallet analytics;
- topup analytics;
- subscription analytics;
- reconciliation analytics.

Rekap:

| Jumlah endpoint diuji | PASS | FAIL |
| ---: | ---: | ---: |
| 20 | 20 | 0 |

## 3. Daftar Endpoint yang Diuji

| Endpoint | Method | Status |
| --- | --- | --- |
| `/analytics/wallets/topup-revenue-trend/query` | POST | PASS |
| `/analytics/wallets/topup-transaction-count/query` | POST | PASS |
| `/analytics/wallets/owner-balance-distribution/query` | POST | PASS |
| `/analytics/wallets/topup-used-vs-unused/query` | POST | PASS |
| `/analytics/wallets/topup-to-subscribe-lag/query` | POST | PASS |
| `/analytics/wallets/zero-vs-nonzero-balance/query` | POST | PASS |
| `/analytics/subscriptions/active-subscription-trend/query` | POST | PASS |
| `/analytics/subscriptions/activation-vs-expiry-trend/query` | POST | PASS |
| `/analytics/subscriptions/renewal-rate/query` | POST | PASS |
| `/analytics/subscriptions/expiry-forecast/query` | POST | PASS |
| `/analytics/subscriptions/package-mix/query` | POST | PASS |
| `/analytics/subscriptions/tenure-mix/query` | POST | PASS |
| `/analytics/subscriptions/days-remaining-histogram/query` | POST | PASS |
| `/analytics/subscriptions/churn-bucket-trend/query` | POST | PASS |
| `/analytics/reconciliation/success-rate/query` | POST | PASS |
| `/analytics/reconciliation/issue-by-type/query` | POST | PASS |
| `/analytics/reconciliation/issue-aging/query` | POST | PASS |
| `/analytics/reconciliation/auto-vs-manual/query` | POST | PASS |
| `/analytics/reconciliation/hanging-transaction-trend/query` | POST | PASS |
| `/analytics/reconciliation/revenue-vs-closing-period-compare/query` | POST | PASS |

## 4. Contoh Pengujian Request dan Response

### 4.1 Contoh request sukses

```http
POST /api/v1/analytics/subscriptions/renewal-rate/query
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
    "series": [
      {
        "key": "renewal_rate",
        "label": "Renewal Rate (%)",
        "points": [
          { "x": "2026-01", "y": 0 },
          { "x": "2026-02", "y": 0 },
          { "x": "2026-03", "y": 0 }
        ]
      }
    ],
    "table": [
      {
        "period": "2026-01",
        "expired_candidate_count": 2,
        "renewed_count": 0,
        "renewal_rate": 0
      }
    ]
  },
  "meta": {
    "request_id": "example-request-id"
  }
}
```

## 5. Verifikasi dan Validasi

- seluruh endpoint wallet, subscription, dan reconciliation merespons `200 OK`;
- `renewal-rate` yang awalnya error sudah berhasil diuji ulang;
- `tenure-mix` stabil setelah penyesuaian query;
- perbandingan revenue vs closing period merespons sesuai kontrak.

## 6. Bug yang Ditemukan dan Diperbaiki

| Endpoint | Masalah awal | Status akhir |
| --- | --- | --- |
| `subscriptions/renewal-rate` | `renewed_count` bisa `NULL` saat scan | PASS |
| `subscriptions/tenure-mix` | query sensitif `ONLY_FULL_GROUP_BY` | PASS |

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

- hanya kirim mode comparison yang memang didukung metadata diagram.

## 8. Kesimpulan

Analytics Sprint 14g3 lulus pengujian. Endpoint finance, subscription, dan reconciliation siap dipakai frontend.
