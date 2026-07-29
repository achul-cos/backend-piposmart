# Sprint 14g3 - Wallet, Topup, Subscription, dan Reconciliation Analytics

## 1. Informasi Dokumen

| Item | Nilai |
| --- | --- |
| Project | Backend CRM Piposmart |
| Sprint | 14g3 |
| Tanggal Perencanaan | 29 Juli 2026 |
| Fokus | Finance dan subscription analytics |
| Kontrak API | Mengikuti kontrak umum Sprint 14g1 |

Dokumentasi hasil pengujian API implementasi sprint ini tersedia di [api-testing.md](api-testing.md).

## 2. Tujuan Sprint 14g3

Sprint 14g3 fokus pada:

- omset topup;
- penggunaan saldo;
- health subscription;
- renewal;
- issue reconciliation;
- pemisahan metrik revenue dan performa closing.

## 3. Daftar Diagram Sprint 14g3

| No | Diagram | Tipe | Endpoint Query | Tujuan | Cara baca | Analisis yang dicari | Kesimpulan/aksi | Perubahan baik bila |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | Topup Revenue Trend | line | `/api/v1/analytics/wallets/topup-revenue-trend/query` | Melihat omset topup | Lihat nominal per periode | Revenue topup naik/turun | Forecast cash | revenue naik |
| 2 | Topup Transaction Count | line | `/api/v1/analytics/wallets/topup-transaction-count/query` | Melihat frekuensi topup | Lihat jumlah transaksi | Banyak transaksi kecil/besar | Analisis perilaku bayar | count naik |
| 3 | Distribusi Saldo Owner | histogram | `/api/v1/analytics/wallets/owner-balance-distribution/query` | Melihat sebaran saldo owner | Lihat bucket saldo | Banyak saldo 0 atau tinggi | Strategi aktivasi beli paket | balance usable naik |
| 4 | Topup Used vs Unused | donut | `/api/v1/analytics/wallets/topup-used-vs-unused/query` | Mengetahui konversi topup | Compare terpakai vs mengendap | Banyak saldo tertahan | Follow-up owner | unused turun |
| 5 | Topup to Subscribe Lag | histogram | `/api/v1/analytics/wallets/topup-to-subscribe-lag/query` | Melihat jeda topup ke pembelian | Bucket hari antar kejadian | Topup cepat/lambat dikonversi | Dorong pembelian paket | avg lag turun |
| 6 | Zero vs Non-zero Balance | donut | `/api/v1/analytics/wallets/zero-vs-nonzero-balance/query` | Mengetahui kesiapan transaksi owner | Compare saldo 0 vs >0 | Banyak owner belum siap beli | Program topup drive | owner bersaldo naik |
| 7 | Active Subscription Trend | line | `/api/v1/analytics/subscriptions/active-subscription-trend/query` | Melihat basis pelanggan aktif | Lihat total sub aktif | Retensi basis pelanggan | Ukur pertumbuhan | active subscription naik |
| 8 | Activation vs Expiry Trend | line | `/api/v1/analytics/subscriptions/activation-vs-expiry-trend/query` | Membandingkan sub masuk vs habis | Dua garis activation/expiry | Net growth positif atau tidak | Fokus renewal | activation > expiry |
| 9 | Renewal Rate | line | `/api/v1/analytics/subscriptions/renewal-rate/query` | Mengukur perpanjangan | Lihat % renewal | Retensi bagus atau lemah | Program renewal | renewal rate naik |
| 10 | Expiry Forecast 30/60/90 | grouped bar | `/api/v1/analytics/subscriptions/expiry-forecast/query` | Memprediksi beban jatuh tempo | Bandingkan bucket 30/60/90 | Risiko jatuh tempo massal | Siapkan follow-up | bucket 30 hari terkendali |
| 11 | Subscription Package Mix | donut | `/api/v1/analytics/subscriptions/package-mix/query` | Melihat komposisi paket aktif | Compare package aktif | Basis user per package | Strategi upsell | package target naik |
| 12 | Subscription Tenure Mix | donut | `/api/v1/analytics/subscriptions/tenure-mix/query` | Melihat durasi sub aktif | Compare tenure aktif | Durasi mayoritas | Atur promo | tenure panjang naik |
| 13 | Days Remaining Histogram | histogram | `/api/v1/analytics/subscriptions/days-remaining-histogram/query` | Melihat sisa masa aktif | Bucket sisa hari | Banyak akun hampir habis | Reminder renewal | bucket kritis turun |
| 14 | Churn Bucket Trend | stacked bar | `/api/v1/analytics/subscriptions/churn-bucket-trend/query` | Melihat EXPIRED vs NOT SUBSCRIBE | Bandingkan bucket churn | Churn jangka pendek vs berat | Win-back strategy | churn turun |
| 15 | Reconciliation Success Rate | line | `/api/v1/analytics/reconciliation/success-rate/query` | Mengukur keberhasilan match | Lihat % confirmed | Matching engine sehat atau tidak | Perbaiki logic/data | success rate naik |
| 16 | Issue by Type | bar | `/api/v1/analytics/reconciliation/issue-by-type/query` | Mengetahui issue dominan | Urutkan jenis issue | Masalah utama | Prioritas fixing | issue count turun |
| 17 | Issue Aging | bar | `/api/v1/analytics/reconciliation/issue-aging/query` | Melihat issue lama tertahan | Bucket umur issue | Issue menggantung | Tambah SLA admin | avg aging turun |
| 18 | Auto vs Manual Reconciliation | donut | `/api/v1/analytics/reconciliation/auto-vs-manual/query` | Mengukur efisiensi sistem | Compare auto vs manual | Rule cukup pintar atau belum | Tingkatkan engine | auto ratio naik |
| 19 | Hanging Transaction Trend | line | `/api/v1/analytics/reconciliation/hanging-transaction-trend/query` | Melihat transaksi menggantung | Lihat trend issue open | Risiko laporan salah | Monitoring harian | hanging turun |
| 20 | Revenue vs Closing Period Compare | grouped bar | `/api/v1/analytics/reconciliation/revenue-vs-closing-period-compare/query` | Memisahkan topup revenue dan performa sales | Bandingkan bulan topup vs bulan closing | Hindari double counting | Edukasi laporan manajemen | selisih terjelaskan |

## 4. Catatan Perhitungan Positif/Negatif

Contoh rule polarity:

- `topup_revenue`: naik = positif
- `unused_balance`: naik = negatif
- `renewal_rate`: naik = positif
- `issue_count`: turun = positif
- `avg_issue_age_days`: turun = positif

## 5. Contoh Insight

```json
{
  "summary": "Renewal rate naik 8.4% dibanding bulan sebelumnya.",
  "conclusion": "Program follow-up renewal berjalan lebih baik pada bulan ini.",
  "recommendation": "Pertahankan reminder untuk outlet dengan sisa hari di bawah 14 hari."
}
```

## 6. Progress Implementasi Backend

Backend Sprint 14g3 saat ini sudah mengaktifkan query endpoint untuk seluruh diagram pada sprint ini melalui kontrak umum analytics:

- `GET /api/v1/analytics/catalog`
- `GET /api/v1/analytics/catalog/:module`
- `GET /api/v1/analytics/catalog/:module/:diagram`
- `POST /api/v1/analytics/:module/:diagram/query`

Diagram 14g3 yang sudah aktif di backend:

- wallets
  - `topup-revenue-trend`
  - `topup-transaction-count`
  - `owner-balance-distribution`
  - `topup-used-vs-unused`
  - `topup-to-subscribe-lag`
  - `zero-vs-nonzero-balance`
- subscriptions
  - `active-subscription-trend`
  - `activation-vs-expiry-trend`
  - `renewal-rate`
  - `expiry-forecast`
  - `package-mix`
  - `tenure-mix`
  - `days-remaining-histogram`
  - `churn-bucket-trend`
- reconciliation
  - `success-rate`
  - `issue-by-type`
  - `issue-aging`
  - `auto-vs-manual`
  - `hanging-transaction-trend`
  - `revenue-vs-closing-period-compare`

Contoh request untuk diagram revenue vs closing compare:

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
    "supervisor_id": [3]
  },
  "options": {
    "include_table": true,
    "include_summary": true
  }
}
```

Catatan implementasi saat ini:

- topup revenue dihitung dari `wallet_payments.paid_at`;
- closing revenue snapshot dihitung dari snapshot closing historis tanpa unique code;
- renewal rate saat ini memakai pendekatan pragmatic: expired pada periode terpilih lalu renewed maksimal 30 hari setelah masa aktif sebelumnya berakhir;
- churn bucket trend memakai definisi status outlet dari Sprint 14c, khusus bucket `EXPIRED` dan `NOT_SUBSCRIBE`;
- export diagram dan compare-series lintas entitas masih menjadi carry over.
