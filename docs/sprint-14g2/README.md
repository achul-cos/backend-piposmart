# Sprint 14g2 - Sales, Catalog, Closing, Target, dan KPI Analytics

## 1. Informasi Dokumen

| Item | Nilai |
| --- | --- |
| Project | Backend CRM Piposmart |
| Sprint | 14g2 |
| Tanggal Perencanaan | 29 Juli 2026 |
| Fokus | Diagram sales performance, catalog, closing, target, KPI |
| Kontrak API | Mengikuti kontrak umum Sprint 14g1 |

Dokumentasi hasil pengujian API implementasi sprint ini tersedia di [api-testing.md](api-testing.md).

## 2. Tujuan Sprint 14g2

Sprint 14g2 berfokus pada analytics yang langsung terasa untuk tim sales dan supervisor:

- paket, plan, promo, histori harga;
- closing sales;
- target sales;
- KPI dan ranking;
- perbandingan aktivitas vs hasil.

## 3. Daftar Diagram Sprint 14g2

| No | Diagram | Tipe | Endpoint Query | Tujuan | Cara baca | Analisis yang dicari | Kesimpulan/aksi | Perubahan baik bila |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | Training to Closing Conversion | bar | `/api/v1/analytics/trainings/training-to-closing-conversion/query` | Mengukur efektivitas training | Bandingkan training selesai vs jadi closing | Demo efektif atau tidak | Tingkatkan materi demo | conversion rate naik |
| 2 | Popularitas Paket | bar | `/api/v1/analytics/catalog/package-popularity/query` | Mengetahui paket terlaris | Urutkan jumlah closing per package | Paket dominan | Fokus positioning | count paket target naik |
| 3 | Popularitas Tenure | bar | `/api/v1/analytics/catalog/tenure-popularity/query` | Mengetahui tenor paling laku | Bandingkan 1/9/12/18/24 bulan | Tenure favorit | Rancang promo | tenure target naik |
| 4 | Heatmap Package x Tenure | heatmap | `/api/v1/analytics/catalog/package-tenure-heatmap/query` | Melihat kombinasi paling laku | Lihat sel paling pekat | Bundle unggulan | Fokus kombinasi terbaik | sel target naik |
| 5 | Promo Adoption Rate | donut | `/api/v1/analytics/catalog/promo-adoption-rate/query` | Mengetahui pemakaian promo | Compare no promo/free/paid promo | Promo yang benar-benar dipilih | Bersihkan promo lemah | adoption promo efektif naik |
| 6 | Additional Charge Adoption | bar | `/api/v1/analytics/catalog/additional-charge-adoption/query` | Mengukur promo berbayar | Bandingkan closing dengan biaya tambahan | Sensitivitas harga pasar | Strategi upsell | upsell rate sehat naik |
| 7 | Timeline Riwayat Harga Paket | line | `/api/v1/analytics/catalog/price-history-timeline/query` | Melihat histori perubahan harga | Baca titik perubahan per effective date | Dampak perubahan harga | Audit pricing | stabil/terkontrol |
| 8 | Timeline Riwayat Promo | line | `/api/v1/analytics/catalog/promotion-history-timeline/query` | Melihat perubahan promo | Timeline perubahan promo | Promo terlalu sering berubah? | Governance promo | perubahan liar turun |
| 9 | Tren Closing | line | `/api/v1/analytics/closings/closing-trend/query` | Melihat closing per waktu | Lihat tren harian/bulanan | Output sales naik/turun | Monitoring penjualan | closing_count naik |
| 10 | Closing by Sales | horizontal bar | `/api/v1/analytics/closings/closing-by-sales/query` | Ranking sales berdasarkan closing | Urutkan jumlah per sales | Top dan low performer | Reward/coaching | closing per sales naik |
| 11 | Closing by Supervisor | horizontal bar | `/api/v1/analytics/closings/closing-by-supervisor/query` | Mengukur output tim supervisor | Bandingkan total tim | Tim terkuat/terlemah | Evaluasi leadership | output tim naik |
| 12 | Closing by Package | bar | `/api/v1/analytics/closings/closing-by-package/query` | Melihat paket paling laku di transaksi closing | Bandingkan count/value | Paket dominan di lapangan | Fokus penawaran | paket target naik |
| 13 | Closing by Tenure | bar | `/api/v1/analytics/closings/closing-by-tenure/query` | Mengetahui tenor penjualan | Bandingkan jumlah per tenure | Durasi favorit customer | Rancang script sales | tenure target naik |
| 14 | Distribusi Status Closing | donut | `/api/v1/analytics/closings/status-distribution/query` | Melihat pending/confirmed/rejected | Bandingkan proporsi status | Banyak pending atau reject | Audit proses | confirmed naik, rejected turun |
| 15 | Tren Average Ticket Size | line | `/api/v1/analytics/closings/average-ticket-size-trend/query` | Melihat rata-rata nilai closing | Lihat tren nilai rata-rata | Penjualan naik karena volume atau nilai | Atur target | avg ticket naik |
| 16 | Waterfall Nilai Closing | waterfall | `/api/v1/analytics/closings/closing-amount-waterfall/query` | Menjelaskan pembentuk final amount | Base price → discount → add charge → final | Faktor utama pembentuk nilai | Edukasi pricing | final amount sehat |
| 17 | Target vs Actual Sales | grouped bar | `/api/v1/analytics/targets/target-vs-actual/query` | Membandingkan target dan realisasi | Lihat gap tiap sales | Sales tertinggal atau unggul | Intervensi supervisor | gap negatif turun |
| 18 | Target Burn-up | line | `/api/v1/analytics/targets/target-burnup/query` | Melihat progres target sepanjang periode | Garis actual vs target | Sales akan capai atau tertinggal | Coaching tengah bulan | actual mendekati/melewati target |
| 19 | KPI Leaderboard | horizontal bar | `/api/v1/analytics/kpi/leaderboard/query` | Ranking KPI | Urutkan skor KPI | Top performer berdasarkan KPI | Reward/coaching | skor KPI naik |
| 20 | Scatter Activity vs Closing | scatter | `/api/v1/analytics/kpi/activity-vs-closing-scatter/query` | Melihat korelasi aktivitas dan hasil | Titik di kuadran aktivitas/closing | Sales sibuk tapi kurang efektif | Coaching kualitas kerja | sales bergerak ke kuadran tinggi-tinggi |

## 4. Contoh Endpoint Spesifik

### 4.1 Query Closing by Sales

```text
POST /api/v1/analytics/closings/closing-by-sales/query
POST /api/v1/analytics/closings/closing-by-sales/export
```

Metric yang dapat dipilih:

- `closing_count`
- `confirmed_closing_count`
- `gross_revenue_snapshot`
- `average_ticket_size`

Comparison yang didukung:

- previous period
- previous month
- previous year
- sales vs sales

## 5. Catatan Frontend

- grafik leaderboard sebaiknya mendukung klik menuju detail sales;
- scatter chart perlu tooltip yang menampilkan:
  - sales name
  - interaction_count
  - closing_count
  - conversion_rate
- waterfall closing perlu label nilai uang yang jelas.

## 6. Progress Implementasi Backend

Backend Sprint 14g2 saat ini sudah mengaktifkan query endpoint untuk seluruh diagram yang direncanakan pada sprint ini melalui kontrak umum analytics:

- `GET /api/v1/analytics/catalog`
- `GET /api/v1/analytics/catalog/:module`
- `GET /api/v1/analytics/catalog/:module/:diagram`
- `POST /api/v1/analytics/:module/:diagram/query`

Diagram 14g2 yang sudah aktif di backend:

- trainings
  - `training-to-closing-conversion`
- catalog
  - `package-popularity`
  - `tenure-popularity`
  - `package-tenure-heatmap`
  - `promo-adoption-rate`
  - `additional-charge-adoption`
  - `price-history-timeline`
  - `promotion-history-timeline`
- closings
  - `closing-trend`
  - `closing-by-sales`
  - `closing-by-supervisor`
  - `closing-by-package`
  - `closing-by-tenure`
  - `status-distribution`
  - `average-ticket-size-trend`
  - `closing-amount-waterfall`
- targets
  - `target-vs-actual`
  - `target-burnup`
- kpi
  - `leaderboard`
  - `activity-vs-closing-scatter`

Contoh request untuk diagram closing trend:

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
  "metrics": ["gross_revenue_snapshot"],
  "filters": {
    "sales_id": [5]
  },
  "options": {
    "include_table": true,
    "include_summary": true
  }
}
```

Catatan implementasi saat ini:

- analytics closing memakai snapshot omzet historis tanpa unique transfer code;
- analytics target dan KPI membaca tabel target/KPI yang sudah ada, lalu menyesuaikan visibilitas role;
- timeline histori price dan promotion dibaca dari `audit_logs`;
- export diagram dan compare series lintas sales/package belum diaktifkan pada Sprint 14g2 ini, jadi masih menjadi carry over.
