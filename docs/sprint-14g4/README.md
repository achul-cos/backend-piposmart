# Sprint 14g4 - Partner, Commission, Governance, dan Audit Analytics

## 1. Informasi Dokumen

| Item | Nilai |
| --- | --- |
| Project | Backend CRM Piposmart |
| Sprint | 14g4 |
| Tanggal Perencanaan | 29 Juli 2026 |
| Fokus | Partner performance, commission, governance, audit trail |
| Kontrak API | Mengikuti kontrak umum Sprint 14g1 |

Dokumentasi hasil pengujian API implementasi sprint ini tersedia di [api-testing.md](api-testing.md).

## 2. Tujuan Sprint 14g4

Sprint 14g4 fokus pada:

- kualitas dan kontribusi partner;
- komisi, payout, unpaid aging;
- histori perubahan rule;
- governance data dan audit.

## 3. Daftar Diagram Sprint 14g4

| No | Diagram | Tipe | Endpoint Query | Tujuan | Cara baca | Analisis yang dicari | Kesimpulan/aksi | Perubahan baik bila |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | Partner Growth Trend | line | `/api/v1/analytics/partners/partner-growth-trend/query` | Melihat pertumbuhan partner | Lihat partner baru per periode | Program partner tumbuh atau tidak | Evaluasi akuisisi partner | partner aktif naik |
| 2 | Partner Type Distribution | donut | `/api/v1/analytics/partners/partner-type-distribution/query` | Mengetahui komposisi jenis partner | Compare tiap type | Type dominan | Fokus type terbaik | type target tumbuh |
| 3 | Referral Count per Partner | horizontal bar | `/api/v1/analytics/partners/referral-count-per-partner/query` | Mengukur kontribusi referral | Urutkan partner by referral | Partner produktif | Prioritas relationship | referral count naik |
| 4 | Referral Conversion per Partner | horizontal bar | `/api/v1/analytics/partners/referral-conversion-per-partner/query` | Mengukur kualitas referral | Compare referral vs closing | Partner berkualitas tinggi/rendah | Atur insentif | conversion rate naik |
| 5 | Partner PIC Workload | stacked bar | `/api/v1/analytics/partners/partner-pic-workload/query` | Melihat beban PIC partner | Bandingkan jumlah partner per sales | PIC overload atau merata | Redistribusi PIC | workload lebih merata |
| 6 | Call Mitra Frequency | line | `/api/v1/analytics/partners/call-mitra-frequency/query` | Mengukur intensitas hubungan partner | Lihat trend call mitra | Partner dirawat atau tidak | Jadwalkan silaturahmi | call frequency sehat naik |
| 7 | Partner Inactivity Aging | bar | `/api/v1/analytics/partners/partner-inactivity-aging/query` | Mengetahui partner dorman | Bucket hari tanpa aktivitas | Partner pasif | Re-engagement | inactivity turun |
| 8 | Partner Region Distribution | bar | `/api/v1/analytics/partners/partner-region-distribution/query` | Melihat sebaran partner per wilayah | Compare wilayah | Area kuat/lemah | Rekrut partner di area kosong | coverage naik |
| 9 | Commission Earned Trend | line | `/api/v1/analytics/commissions/commission-earned-trend/query` | Melihat accrual komisi | Lihat nominal per periode | Beban komisi naik/turun | Forecast payout | earning sehat |
| 10 | Paid vs Unpaid Commission | donut | `/api/v1/analytics/commissions/paid-vs-unpaid/query` | Mengetahui kewajiban komisi | Compare paid dan unpaid | Banyak beban belum dibayar | Atur payout | unpaid ratio turun |
| 11 | Commission Aging | bar | `/api/v1/analytics/commissions/commission-aging/query` | Melihat umur komisi belum dibayar | Bucket umur earning | Komisi menumpuk | Proses payout | avg aging turun |
| 12 | Commission by Partner Type | bar | `/api/v1/analytics/commissions/commission-by-partner-type/query` | Mengetahui type partner mahal/murah | Compare nominal per type | Type komisi terbesar | Evaluasi rule | payout lebih efisien |
| 13 | Commission by Package | bar | `/api/v1/analytics/commissions/commission-by-package/query` | Mengetahui paket penyumbang komisi | Compare per package | Paket pemicu komisi besar | Review profitabilitas | margin sehat |
| 14 | Payout Waterfall | waterfall | `/api/v1/analytics/commissions/payout-waterfall/query` | Melihat alur earning ke paid | Earned → approved → paid | Titik delay payout | Perbaiki approval | flow lebih lancar |
| 15 | Commission Rule History Timeline | line | `/api/v1/analytics/commissions/rule-history-timeline/query` | Melihat perubahan rule komisi | Timeline perubahan | Rule terlalu sering berubah? | Governance komisi | perubahan tak perlu turun |
| 16 | Snapshot vs Current Commission | grouped bar | `/api/v1/analytics/commissions/snapshot-vs-current/query` | Memastikan historical commission statis | Compare snapshot vs rule sekarang | Historical aman | Validasi akuntansi | mismatch nol |
| 17 | Audit Log Volume by Module | bar | `/api/v1/analytics/audit/log-volume-by-module/query` | Mengetahui modul paling sering berubah | Urutkan count audit | Modul paling dinamis | Fokus pengawasan | sesuai ekspektasi |
| 18 | Actor Activity Chart | horizontal bar | `/api/v1/analytics/audit/actor-activity-chart/query` | Melihat siapa paling aktif mengubah data | Bandingkan per actor | Aktivitas edit abnormal atau normal | Audit user | outlier bisa diinvestigasi |
| 19 | Restore vs Delete Trend | line | `/api/v1/analytics/audit/restore-vs-delete-trend/query` | Mengukur kualitas operasi delete | Compare restore dan delete | Banyak salah hapus atau tidak | Perbaiki UX/otorisasi | restore akibat salah hapus turun |
| 20 | Backend Error Code Frequency | bar | `/api/v1/analytics/audit/backend-error-code-frequency/query` | Mengetahui error backend paling sering | Group code error | VALIDATION/FORBIDDEN/CONFLICT dominan | Prioritaskan fixing | error kritis turun |

## 4. Catatan Governance

Diagram histori perubahan wajib memakai:

- actor user
- waktu perubahan
- nilai lama
- nilai baru
- request_id bila tersedia

Tujuannya agar dashboard governance tidak hanya informatif, tetapi juga bisa diaudit.

## 5. Progress Implementasi Backend

Backend Sprint 14g4 saat ini sudah mengaktifkan query endpoint untuk seluruh diagram pada sprint ini melalui kontrak umum analytics:

- `GET /api/v1/analytics/catalog`
- `GET /api/v1/analytics/catalog/:module`
- `GET /api/v1/analytics/catalog/:module/:diagram`
- `POST /api/v1/analytics/:module/:diagram/query`

Diagram 14g4 yang sudah aktif di backend:

- partners
  - `partner-growth-trend`
  - `partner-type-distribution`
  - `referral-count-per-partner`
  - `referral-conversion-per-partner`
  - `partner-pic-workload`
  - `call-mitra-frequency`
  - `partner-inactivity-aging`
  - `partner-region-distribution`
- commissions
  - `commission-earned-trend`
  - `paid-vs-unpaid`
  - `commission-aging`
  - `commission-by-partner-type`
  - `commission-by-package`
  - `payout-waterfall`
  - `rule-history-timeline`
  - `snapshot-vs-current`
- audit
  - `log-volume-by-module`
  - `actor-activity-chart`
  - `restore-vs-delete-trend`
  - `backend-error-code-frequency`

Contoh request untuk diagram commission aging:

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
    "mode": "previous_month"
  },
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

- referral conversion partner memakai referral vs commission/closing yang berhasil terbentuk;
- partner region distribution saat ini diturunkan dari wilayah owner pada referral partner, karena tabel partner belum punya kolom provinsi/kota terstruktur;
- commission rule history timeline saat ini membaca histori `partner.type` pada audit log untuk melacak perubahan mode/nilai komisi master;
- backend error code frequency saat ini hanya menghitung failure backend yang benar-benar persisted di database, yaitu:
  - `VALIDATION_FAILED` dari `import_batches`
  - `COMMIT_FAILED` dari `import_batches`
  - `JOB_FAILED` dari `job_queue`
- jadi chart error backend ini belum merepresentasikan seluruh HTTP error response global, dan itu masih menjadi carry over desain observability berikutnya.
