# Sprint 14g5 - Importing, Executive Analytics, Advanced Comparison, dan Export Center

## 1. Informasi Dokumen

| Item | Nilai |
| --- | --- |
| Project | Backend CRM Piposmart |
| Sprint | 14g5 |
| Tanggal Perencanaan | 29 Juli 2026 |
| Fokus | Import analytics, executive dashboard, custom comparison, export XLSX/PDF/PNG |
| Kontrak API | Mengikuti kontrak umum Sprint 14g1 |

Dokumentasi hasil pengujian API implementasi sprint ini tersedia di [api-testing.md](api-testing.md).

## 2. Tujuan Sprint 14g5

Sprint 14g5 adalah fase pematangan analytics:

- quality dashboard untuk importing;
- executive board lintas modul;
- comparison builder yang lebih fleksibel;
- export center untuk excel/pdf/png.

## 3. Daftar Diagram Sprint 14g5

| No | Diagram | Tipe | Endpoint Query | Tujuan | Cara baca | Analisis yang dicari | Kesimpulan/aksi | Perubahan baik bila |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | Import Batches per Profile | bar | `/api/v1/analytics/imports/batches-per-profile/query` | Melihat profil import dominan | Compare jumlah batch per profile | Profil tersibuk | Fokus template/profile | proses stabil |
| 2 | Import Success vs Failed | donut | `/api/v1/analytics/imports/success-vs-failed/query` | Mengukur kesehatan import | Compare batch sukses/gagal | Import stabil atau tidak | Prioritas stabilisasi | fail rate turun |
| 3 | Invalid Rows Distribution | histogram | `/api/v1/analytics/imports/invalid-rows-distribution/query` | Melihat kualitas file | Bucket invalid rows | Batch kotor atau bersih | Edukasi uploader | invalid rows turun |
| 4 | Validation Error by Profile | bar | `/api/v1/analytics/imports/validation-error-by-profile/query` | Mengetahui profil paling rawan | Group error per profile | Profil sulit dipakai | Perbaiki template | error turun |
| 5 | Duplicate Detection Rate | line | `/api/v1/analytics/imports/duplicate-detection-rate/query` | Mengukur duplikat data import | Lihat rate duplicate | Sumber data noisy | Perketat dedupe | duplicate rate turun |
| 6 | Import Duration Trend | line | `/api/v1/analytics/imports/import-duration-trend/query` | Mengukur performa proses import | Lihat durasi per batch | Worker melambat atau normal | Optimasi worker | duration turun |
| 7 | Batch Status Funnel | funnel | `/api/v1/analytics/imports/batch-status-funnel/query` | Melihat upload → validate → commit | Baca penyusutan tiap tahap | Banyak batch berhenti di mana | Fokus fixing tahap itu | batch committed naik |
| 8 | Import Uploader Activity | horizontal bar | `/api/v1/analytics/imports/uploader-activity/query` | Mengetahui user paling aktif upload | Urutkan batch per uploader | Distribusi kerja admin | Evaluasi operasional | distribusi sehat |
| 9 | File History Usage | line | `/api/v1/analytics/imports/file-history-usage/query` | Melihat pemakaian viewer/download | Lihat open/download trend | Fitur histori dipakai atau tidak | Validasi UX histori | usage sehat |
| 10 | End-to-End Business Funnel | funnel | `/api/v1/analytics/executive/end-to-end-funnel/query` | Melihat alur bisnis penuh | Owner → Lead → Training → Closing → Subscription | Titik bocor utama | Fokus lintas tim | conversion antar tahap naik |
| 11 | Revenue vs Closing vs Active Subscription | grouped bar | `/api/v1/analytics/executive/revenue-closing-active-subscription-board/query` | Memisahkan 3 metrik utama | Baca masing-masing metrik terpisah | Bisnis naik dari sisi mana | Hindari salah tafsir | ketiganya sehat |
| 12 | Monthly Operating Review Board | mixed | `/api/v1/analytics/executive/monthly-operating-review-board/query` | Ringkasan bulanan manajemen | Lihat KPI utama dalam satu board | Kondisi bisnis bulanan | Bahan rapat manajemen | warna indikator membaik |
| 13 | North Star KPI Trend | line | `/api/v1/analytics/executive/north-star-kpi-trend/query` | Melihat tren KPI inti | Line beberapa KPI utama | Arah bisnis jangka menengah | Keputusan manajerial | KPI inti naik |
| 14 | Data Quality Score by Module | bar | `/api/v1/analytics/executive/data-quality-score-by-module/query` | Mengukur kualitas data per modul | Bandingkan score tiap modul | Modul paling rawan data | Fokus cleansing | score naik |
| 15 | Custom Multi-series Trend | line | `/api/v1/analytics/custom/multi-series-trend/query` | Membandingkan beberapa metrik sekaligus | Banyak garis dalam satu chart | Korelasi antar metrik | Analisis custom analyst | sesuai tujuan user |
| 16 | Custom Metric Comparison Board | grouped bar | `/api/v1/analytics/custom/metric-comparison-board/query` | Membandingkan beberapa metrik dalam satu periode | Bandingkan card/bar metric | Metric mana unggul/tertinggal | Review kinerja | gap positif |
| 17 | Custom Region Comparison Board | grouped bar | `/api/v1/analytics/custom/region-comparison-board/query` | Membandingkan wilayah | Compare provinsi/kota | Area terbaik/terburuk | Prioritas ekspansi | region target naik |
| 18 | Subscription Cohort Retention | heatmap | `/api/v1/analytics/subscriptions/cohort-retention/query` | Melihat ketahanan cohort subscription | Baca cohort per bulan aktivasi | Cohort kuat/lemah | Evaluasi onboarding | retention naik |
| 19 | Forecast Summary Board | mixed | `/api/v1/analytics/executive/forecast-summary-board/query` | Ringkasan prediksi expiry, churn, issue | Lihat card forecast | Risiko bulan depan | Persiapan operasional | risk score turun |
| 20 | Comparison Impact Summary | bar | `/api/v1/analytics/custom/comparison-impact-summary/query` | Meringkas dampak comparison | Compare delta banyak diagram | Area paling positif/negatif | Fokus perbaikan cepat | delta positif bertambah |

## 4. Comparison Builder yang Direncanakan

Sprint 14g5 menambahkan comparison yang lebih fleksibel:

- sales vs sales
- supervisor vs supervisor
- province vs province
- city vs city
- current period vs previous period
- current month vs previous month
- current year vs previous year
- custom period vs custom period
- metric A vs metric B dalam satu board

Contoh request:

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
    "mode": "series_to_series",
    "compare_series": [
      {
        "field": "province",
        "label": "Jawa Barat",
        "value": "Jawa Barat"
      },
      {
        "field": "province",
        "label": "DKI Jakarta",
        "value": "DKI Jakarta"
      }
    ]
  },
  "metrics": [
    "owner_count",
    "outlet_count"
  ]
}
```

## 5. Export Center yang Direncanakan

Route:

```text
POST /api/v1/analytics/{module}/{diagram_key}/export
GET  /api/v1/analytics/exports/{job_id}
```

Format:

- `xlsx`
- `pdf`
- `png`

Output:

- XLSX: tabel + metadata filter + insight
- PDF: chart + summary + conclusion
- PNG: chart image siap presentasi

Export dijalankan async melalui job queue agar aman untuk file besar.

## 6. Progress Implementasi Backend

Status implementasi backend Sprint 14g5 per 29 Juli 2026:

- selesai: 20 query diagram analytics backend pada modul `imports`, `executive`, `custom`, dan `subscriptions`;
- selesai: update analytics catalog sehingga frontend dapat membaca metadata nama diagram, fungsi, tujuan, cara baca, dan endpoint query;
- selesai: dukungan comparison berbasis baseline period untuk board custom/executive, dan compare-series sederhana untuk region comparison board;
- selesai: executive board lintas modul untuk funnel, revenue vs closing vs subscription, north star KPI, quality score, forecast summary;
- selesai: cohort retention backend untuk subscription;
- belum: export center async `xlsx/pdf/png`;
- belum: event log usage viewer/download file import yang persisted, sehingga diagram `file-history-usage` masih mengembalikan placeholder contract + source note.

### Diagram yang Sudah Aktif di Backend

| Modul | Diagram Key | Status |
| --- | --- | --- |
| imports | `batches-per-profile` | ✅ Active |
| imports | `success-vs-failed` | ✅ Active |
| imports | `invalid-rows-distribution` | ✅ Active |
| imports | `validation-error-by-profile` | ✅ Active |
| imports | `duplicate-detection-rate` | ✅ Active |
| imports | `import-duration-trend` | ✅ Active |
| imports | `batch-status-funnel` | ✅ Active |
| imports | `uploader-activity` | ✅ Active |
| imports | `file-history-usage` | ⚠️ Placeholder source note |
| executive | `end-to-end-funnel` | ✅ Active |
| executive | `revenue-closing-active-subscription-board` | ✅ Active |
| executive | `monthly-operating-review-board` | ✅ Active |
| executive | `north-star-kpi-trend` | ✅ Active |
| executive | `data-quality-score-by-module` | ✅ Active |
| executive | `forecast-summary-board` | ✅ Active |
| custom | `multi-series-trend` | ✅ Active |
| custom | `metric-comparison-board` | ✅ Active |
| custom | `region-comparison-board` | ✅ Active |
| custom | `comparison-impact-summary` | ✅ Active |
| subscriptions | `cohort-retention` | ✅ Active |

### Catatan Penting untuk Frontend

- seluruh diagram Sprint 14g5 tetap memakai route generik yang sama:

```text
POST /api/v1/analytics/{module}/{diagram_key}/query
```

- frontend dianjurkan membaca catalog dulu dari:

```text
GET /api/v1/analytics/catalog
GET /api/v1/analytics/catalog/{module}
GET /api/v1/analytics/catalog/{module}/{diagram_key}
```

- diagram `region-comparison-board` mendukung `comparison.compare_series`;
- diagram `comparison-impact-summary` memanfaatkan baseline period dari `comparison.mode`;
- diagram `file-history-usage` belum memiliki source usage log nyata, jadi response berisi `source_note`;
- score pada `data-quality-score-by-module` adalah heuristic backend score 0-100, bukan skor final governance enterprise.
