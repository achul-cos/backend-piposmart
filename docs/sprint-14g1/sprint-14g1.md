Sprint: 14g1 - Analytics Foundation, Customer Ops, dan Peta Indonesia
Periode: 29 Juli 2026
Status: IN PROGRESS

Sprint Goal:
- Membekukan kontrak endpoint analytics per diagram.
- Menyediakan analytics prioritas tinggi untuk owner, outlet, lead, interaction, training.
- Menyediakan diagram peta Indonesia untuk persebaran owner dan outlet.

Committed Deliverables:
- Base route `/api/v1/analytics/...`
- Analytics catalog endpoint
- Query contract dengan filter tanggal/bulan/tahun
- Comparison engine v1
- Insight payload (`summary`, `conclusion`, `recommendation`)
- 20 endpoint diagram Sprint 14g1

Completed:
- Menambahkan modul backend baru `internal/analytics`.
- Menambahkan route:
  - `GET /api/v1/analytics/catalog`
  - `GET /api/v1/analytics/catalog/:module`
  - `GET /api/v1/analytics/catalog/:module/:diagram`
  - `POST /api/v1/analytics/:module/:diagram/query`
- Menambahkan registry 20 diagram Sprint 14g1.
- Menambahkan parser time filter:
  - `date_range`
  - `month_range`
  - `year_range`
- Menambahkan comparison mode:
  - `previous_period`
  - `previous_month`
  - `previous_year`
  - `custom_period`
- Menambahkan payload response analytics:
  - metadata diagram
  - summary comparison
  - status positif / negatif
  - summary / conclusion / recommendation
- Mengimplementasikan query backend untuk 20 diagram Sprint 14g1.
- Menambahkan test dasar untuk parser time filter, comparison resolver, dan registry diagram.

Verification:
- `go build ./...` PASS
- `go test ./internal/analytics/...` PASS
- `go test ./internal/platform/httpserver/...` PASS, dengan catatan cleanup warning temporary binary di Windows setelah package test selesai `ok`

Not Completed / Carry Over:
- OpenAPI analytics belum diperbarui pada implementasi awal ini.
- Export analytics (`xlsx`, `pdf`, `png`) masih menjadi bagian roadmap fase berikutnya.

Dependencies:
- Definisi filter wilayah owner/outlet
- Mapping province code untuk Indonesia map
- Keputusan final format comparison

Risiko:
- Risiko: kontrak analytics terlalu generik dan sulit dipakai frontend.
- Mitigasi: setiap diagram tetap punya endpoint spesifik, bukan satu endpoint serba bebas.

Definition of Done Saat Ini:
- Fondasi backend analytics sudah tersedia.
- Endpoint query untuk 20 diagram Sprint 14g1 sudah aktif di backend.
- Frontend dapat mulai integrasi melalui catalog dan route query analytics.
