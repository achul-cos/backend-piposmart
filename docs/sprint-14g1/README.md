# Sprint 14g1 - Analytics Foundation, Customer Ops, dan Peta Indonesia

## 1. Informasi Dokumen

| Item | Nilai |
| --- | --- |
| Project | Backend CRM Piposmart |
| Sprint | 14g1 |
| Tanggal Perencanaan | 29 Juli 2026 |
| Fokus | Fondasi endpoint analytics per diagram, owner/outlet/lead/activity/training, dan peta Indonesia |
| Target Pengguna | Frontend Engineer, Backend Engineer, System Analyst, Data Analyst |

Dokumentasi hasil pengujian API implementasi sprint ini tersedia di [api-testing.md](api-testing.md).

## 2. Tujuan Sprint 14g1

Sprint 14g1 menjadi fondasi seluruh modul diagram. Fokus utamanya:

- membekukan kontrak request/response analytics;
- membangun endpoint per diagram, bukan satu endpoint generik yang terlalu abstrak;
- memastikan semua diagram bisa difilter berdasarkan:
  - rentang tanggal
  - rentang bulan
  - rentang tahun
- memastikan semua diagram bisa melakukan comparison;
- memastikan response analytics tidak hanya berisi angka, tetapi juga:
  - nama diagram
  - fungsi diagram
  - tujuan diagram
  - cara membaca diagram
  - nilai delta
  - arah perubahan positif/negatif
  - kesimpulan analisis
- menghadirkan diagram peta Indonesia untuk persebaran owner dan outlet.

## 3. Roadmap 5 Fase Analytics

| Sprint | Fokus | Modul |
| --- | --- | --- |
| 14g1 | Fondasi analytics + customer ops + geo | owner, outlet, lead, interaction, training |
| 14g2 | Sales, catalog, closing, target, KPI | catalog, closing, target, KPI |
| 14g3 | Finance, wallet, subscription, reconciliation | wallet, topup, subscription, reconciliation |
| 14g4 | Partner, commission, governance, audit | partner, commission, audit trail |
| 14g5 | Importing, executive analytics, advanced comparison, export center | importing, executive dashboard, custom comparison, export |

## 4. Kontrak Endpoint Analytics Umum

## 4.1 Pola Route

Setiap diagram akan memiliki endpoint spesifik:

```text
POST /api/v1/analytics/{module}/{diagram_key}/query
POST /api/v1/analytics/{module}/{diagram_key}/export
GET  /api/v1/analytics/catalog
GET  /api/v1/analytics/catalog/{module}
GET  /api/v1/analytics/catalog/{module}/{diagram_key}
GET  /api/v1/analytics/exports/{job_id}
```

Contoh:

```text
POST /api/v1/analytics/owners/growth-trend/query
POST /api/v1/analytics/outlets/indonesia-distribution-map/query
POST /api/v1/analytics/leads/funnel/query
POST /api/v1/analytics/interactions/follow-up-compliance/query
```

## 4.2 Request Schema Umum

```json
{
  "time_filter": {
    "mode": "date_range",
    "date_from": "2026-07-01",
    "date_to": "2026-07-29",
    "month_from": null,
    "month_to": null,
    "year_from": null,
    "year_to": null,
    "granularity": "day"
  },
  "comparison": {
    "enabled": true,
    "mode": "previous_period",
    "baseline_time_filter": null,
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
    "owner_count"
  ],
  "dimensions": [
    "date"
  ],
  "filters": {
    "province": [
      "Jawa Barat"
    ],
    "city": [],
    "sales_id": [],
    "supervisor_id": [],
    "owner_id": [],
    "outlet_id": [],
    "status": []
  },
  "options": {
    "limit": 10,
    "sort": "-value",
    "include_table": true,
    "include_summary": true,
    "include_previous_points": true
  }
}
```

## 4.3 Aturan Filter Waktu

- `mode=date_range`
  - gunakan `date_from` dan `date_to`
- `mode=month_range`
  - gunakan `month_from` dan `month_to` dengan format `YYYY-MM`
- `mode=year_range`
  - gunakan `year_from` dan `year_to`
- `granularity`
  - `day`
  - `week`
  - `month`
  - `quarter`
  - `year`

Aturan:

- hanya satu `mode` aktif per request;
- comparison dapat memakai:
  - `previous_period`
  - `previous_month`
  - `previous_year`
  - `custom_period`
  - `series_to_series`

## 4.4 Response Schema Umum

```json
{
  "data": {
    "diagram": {
      "key": "growth-trend",
      "module": "owners",
      "name": "Tren Pertumbuhan Owner",
      "type": "line",
      "purpose": "Melihat pertumbuhan owner baru dari waktu ke waktu.",
      "how_to_read": "Semakin tinggi garis, semakin banyak owner baru pada periode tersebut.",
      "analysis_goal": "Mengetahui apakah akuisisi owner meningkat atau menurun."
    },
    "time_filter": {
      "mode": "date_range",
      "granularity": "day",
      "label": "01 Jul 2026 - 29 Jul 2026"
    },
    "comparison": {
      "enabled": true,
      "mode": "previous_period",
      "baseline_label": "02 Jun 2026 - 30 Jun 2026",
      "current_value": 124,
      "baseline_value": 105,
      "delta": 19,
      "delta_percent": 18.1,
      "direction": "positive",
      "polarity_rule": "higher_is_better",
      "status_value": 1
    },
    "series": [
      {
        "key": "owner_count",
        "label": "Owner Baru",
        "points": [
          {
            "x": "2026-07-01",
            "y": 4
          },
          {
            "x": "2026-07-02",
            "y": 7
          }
        ]
      }
    ],
    "table": [
      {
        "period": "2026-07-01",
        "owner_count": 4
      }
    ],
    "insight": {
      "summary": "Owner baru naik 19 data dibanding periode sebelumnya.",
      "conclusion": "Akuisisi owner membaik dan tren harian cenderung stabil naik.",
      "recommendation": "Pertahankan sumber lead yang aktif di wilayah dengan pertumbuhan tertinggi."
    }
  },
  "meta": {
    "request_id": "analytics-req-001"
  }
}
```

### Interpretasi `status_value`

- `1` = positif
- `0` = netral
- `-1` = negatif

Catatan:

- positif/negatif mengikuti `polarity_rule`, bukan selalu berarti angka naik itu baik;
- contoh:
  - `owner_count`: naik = positif
  - `issue_count`: turun = positif

## 4.5 Export Contract

```json
{
  "format": "xlsx",
  "include_chart": true,
  "include_raw_data": true,
  "include_summary": true,
  "file_name": "owner-growth-july-2026",
  "layout": "landscape"
}
```

Format yang harus didukung dalam roadmap:

- `xlsx`
- `pdf`
- `png`

Target output:

- XLSX: data tabel + insight + metadata filter
- PDF: chart + insight + filter ringkas
- PNG: visual chart saja untuk embed presentasi

## 5. Tipe Diagram yang Dipakai di 14g1

- line
- bar
- stacked bar
- donut
- sankey
- histogram
- heatmap
- choropleth map Indonesia

## 6. Daftar Diagram Sprint 14g1

| No | Diagram | Tipe | Endpoint Query | Tujuan | Cara baca | Analisis yang dicari | Kesimpulan/aksi | Perubahan baik bila |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | Tren Pertumbuhan Owner | line | `/api/v1/analytics/owners/growth-trend/query` | Melihat owner baru per periode | Lihat garis per hari/bulan/tahun | Owner naik atau turun | Evaluasi akuisisi owner | `owner_count` naik |
| 2 | Distribusi Kepemilikan Owner | stacked bar | `/api/v1/analytics/owners/ownership-distribution/query` | Melihat owner milik Admin/Supervisor/Sales | Bandingkan stack per role | Ownership menumpuk di level mana | Redistribusi owner | distribusi lebih merata |
| 3 | Distribusi Owner per Provinsi | bar | `/api/v1/analytics/owners/province-distribution/query` | Mengetahui persebaran pasar owner | Bandingkan nilai per provinsi | Provinsi dominan dan lemah | Fokus regional | provinsi target tumbuh |
| 4 | Top 10 Kota Owner | bar | `/api/v1/analytics/owners/city-top10/query` | Mengetahui kota owner terbanyak | Urutkan kota dengan nilai tertinggi | Kota prioritas akuisisi | Fokus kota potensial | kota target tumbuh |
| 5 | Tren Soft Delete Owner | line | `/api/v1/analytics/owners/soft-delete-trend/query` | Melihat owner yang dibuang | Lihat tren deleted owner | Banyak data buruk atau tidak | Audit input owner | `soft_deleted_count` turun |
| 6 | Tren Pertumbuhan Outlet | line | `/api/v1/analytics/outlets/growth-trend/query` | Melihat pertumbuhan outlet | Lihat outlet baru per periode | Pertumbuhan jaringan outlet | Baca skala customer | `outlet_count` naik |
| 7 | Distribusi Outlet per Owner | histogram | `/api/v1/analytics/outlets/outlet-per-owner-histogram/query` | Mengetahui owner single vs multi outlet | Lihat sebaran bucket outlet | Struktur basis customer | Segmentasi owner | owner multi-outlet bertambah |
| 8 | Rekap Status Langganan Outlet | stacked bar | `/api/v1/analytics/outlets/subscription-status-recap/query` | Melihat NEW/BERLANGGANAN/JATUH TEMPO/EXPIRED/NOT SUBSCRIBE | Baca stack status per bulan | Kesehatan basis outlet | Fokus retention/reaktivasi | ACTIVE/NEW naik, EXPIRED turun |
| 9 | Tren Outlet Not Subscribe | line | `/api/v1/analytics/outlets/not-subscribe-trend/query` | Melihat outlet yang lama tidak berlangganan | Lihat trend outlet mati | Potensi lost customer | Program win-back | `not_subscribe_count` turun |
| 10 | Peta Indonesia Persebaran Owner | choropleth map | `/api/v1/analytics/owners/indonesia-distribution-map/query` | Menampilkan persebaran owner di Indonesia | Warna lebih pekat = owner lebih banyak | Provinsi paling padat owner | Fokus penetrasi nasional | provinsi target tumbuh |
| 11 | Peta Indonesia Persebaran Outlet | choropleth map | `/api/v1/analytics/outlets/indonesia-distribution-map/query` | Menampilkan persebaran outlet | Lihat sebaran outlet per provinsi | Kekuatan outlet nasional | Fokus support regional | outlet target tumbuh |
| 12 | Funnel Lead | funnel | `/api/v1/analytics/leads/funnel/query` | Melihat perjalanan lead | Baca penurunan per stage | Tahap bocor terbesar | Intervensi stage tersebut | conversion naik |
| 13 | Aging Lead per Stage | bar | `/api/v1/analytics/leads/aging-by-stage/query` | Mengetahui lead yang lama tertahan | Bandingkan hari rata-rata per stage | Stage macet | Tambah SLA follow-up | `avg_age_days` turun |
| 14 | Distribusi Assignment Lead | stacked bar | `/api/v1/analytics/leads/assignment-distribution/query` | Melihat lead per supervisor/sales | Bandingkan jumlah lead PIC | Workload timpang atau tidak | Redistribusi lead | distribusi lebih merata |
| 15 | Sankey Perpindahan Kepemilikan Lead | sankey | `/api/v1/analytics/leads/ownership-transfer-sankey/query` | Menelusuri aliran Admin → Supervisor → Sales | Ikuti arah panah | Banyak lead muter atau lancar | Audit kualitas data & sales | transfer berulang turun |
| 16 | Tren Volume Interaksi | line | `/api/v1/analytics/interactions/volume-trend/query` | Mengukur aktivitas call/chat | Lihat jumlah interaksi per waktu | Sales aktif atau tidak | Monitoring aktivitas | `interaction_count` naik |
| 17 | Distribusi Remark 0-3 | donut | `/api/v1/analytics/interactions/remark-distribution/query` | Melihat hasil call customer | Bandingkan proporsi remark | Banyak invalid atau potensial | Evaluasi kualitas komunikasi | remark 2-3 naik, remark 0 turun |
| 18 | Kepatuhan Follow-up | bar | `/api/v1/analytics/interactions/follow-up-compliance/query` | Melihat due vs done | Bandingkan executed terhadap scheduled | Sales disiplin atau tidak | Penguatan SOP | compliance naik |
| 19 | Jeda Kontak Pertama | histogram | `/api/v1/analytics/interactions/first-response-lag/query` | Mengetahui kecepatan first contact | Lihat sebaran jam/hari | Respons lambat atau cepat | Tetapkan SLA | `avg_first_response_hours` turun |
| 20 | Jadwal vs Selesai Training | bar | `/api/v1/analytics/trainings/scheduled-vs-completed/query` | Mengukur eksekusi training | Bandingkan scheduled dan completed | Banyak cancel/no-show | Perbaiki reminder & jadwal | completion rate naik |

## 7. Contoh Diagram dan Cara Frontend Merender

### 7.1 Contoh Line Chart

Diagram: `Tren Pertumbuhan Owner`

```text
120 |                            *
100 |                      *   *
 80 |                *   *
 60 |            *  *
 40 |      *   *
 20 |   * *
    +------------------------------
      01 05 10 15 20 25 29 Jul
```

### 7.2 Contoh Choropleth Map Indonesia

Diagram:

- `Peta Indonesia Persebaran Owner`
- `Peta Indonesia Persebaran Outlet`

Cara baca:

- warna semakin pekat = jumlah owner/outlet semakin besar;
- tooltip minimal menampilkan:
  - nama provinsi
  - owner_count
  - outlet_count bila relevan
  - delta vs periode pembanding

Contoh response region:

```json
{
  "province_code": "ID-JB",
  "province_name": "Jawa Barat",
  "owner_count": 184,
  "delta": 21,
  "delta_percent": 12.9,
  "direction": "positive",
  "status_value": 1
}
```

## 8. Catatan Frontend

- frontend tidak perlu menebak nama diagram; backend sudah kirimkan `name`, `purpose`, `how_to_read`, dan `analysis_goal`;
- frontend tinggal merender metadata itu di card detail / tooltip / panel analisis;
- comparison panel sebaiknya bisa menampilkan:
  - nilai sekarang
  - nilai sebelumnya
  - delta absolut
  - delta persen
  - label positif/negatif

## 9. Definition of Ready Sprint 14g1

Sprint 14g1 dianggap siap diimplementasikan bila:

- kontrak request/response analytics dibekukan;
- daftar 20 diagram di atas disetujui frontend;
- daftar dimensi/filter yang diperlukan sudah final;
- keputusan format export async disetujui.

## 10. Progress Implementasi Backend

Per 29 Juli 2026, backend Sprint 14g1 sudah mulai diimplementasikan.

Route yang sudah aktif:

```text
GET  /api/v1/analytics/catalog
GET  /api/v1/analytics/catalog/{module}
GET  /api/v1/analytics/catalog/{module}/{diagram_key}
POST /api/v1/analytics/{module}/{diagram_key}/query
```

Status implementasi saat ini:

- modul baru `internal/analytics` sudah ditambahkan;
- 20 diagram Sprint 14g1 sudah memiliki query backend;
- catalog diagram sudah tersedia untuk frontend;
- comparison engine v1 sudah aktif;
- peta Indonesia owner dan outlet sudah tersedia dalam bentuk data region untuk frontend render.

Contoh request:

```http
POST /api/v1/analytics/owners/growth-trend/query
Authorization: Bearer {access_token}
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
    "mode": "previous_year"
  },
  "filters": {
    "province": ["Jawa Barat"]
  },
  "options": {
    "include_table": true,
    "include_summary": true
  }
}
```

Catatan:

- export belum diaktifkan di 14g1;
- OpenAPI analytics belum diperbarui pada implementasi awal ini;
- frontend dapat memakai endpoint catalog untuk membaca nama diagram, fungsi, tujuan, dan cara baca tanpa hardcode manual.
