Sprint: 14 — Import Framework dan Data Customer
Periode: 26 Juli 2026 (dikerjakan dalam satu sesi, mengikuti siklus sprint yang berlaku)
Status: GREEN

Sprint Goal:
- Admin dapat mengimpor owner, outlet, dan lead dari Excel dengan aman.

Committed Deliverables:
- Import batch dan import rows.
- Upload, SHA-256, profile detection, staging, validation, preview, dan commit.
- Background job import.
- Profil Owner & Outlet dan Non-Register.
- Sanitasi OTP.
- Normalisasi Excel serial date dan nomor telepon.
- File hasil rejected rows.

Completed:
- Domain baru `internal/importing` (types/errors/repository/excel/service/worker/handler), dua tabel baru (`import_batches`, `import_rows`), dependency baru `github.com/xuri/excelize/v2`.
- Upload multipart dengan dedup SHA-256 (`file_sha256` UNIQUE) — file yang sama mengembalikan batch existing, tidak pernah duplikat. Nama file di disk diturunkan dari hash (bukan input user) untuk mencegah path traversal.
- Deteksi profil dua mode: eksplisit (`profile` form field) atau otomatis dari header kolom — marker header diambil langsung dari file Excel asli kantor (`c:/piposmart/data_admin/`), bukan ditebak. Header-hunting memindai 5 baris pertama, menangani file dengan judul gabungan di row 1 (header sungguhan di row 2, seperti file "Data Belum Registrasi").
- Validasi & commit sepenuhnya async lewat `job_queue` Sprint 13 (`IMPORT_VALIDATE`, `IMPORT_COMMIT`) — request upload/commit tidak pernah menahan HTTP untuk file besar. Job queue dipakai apa adanya (generik), tidak dibangun ulang.
- Sanitasi OTP: kolom `Kode OTP`/`Created Date Kode OTP` di-strip di level index header (dua index terpisah: satu untuk deteksi profil yang butuh tahu OTP ada, satu lagi—aman—untuk ekstraksi nilai) sehingga OTP mentah tidak pernah masuk ke `raw_payload` dalam bentuk apa pun. Diverifikasi langsung terhadap nilai OTP asli yang ada di file sumber.
- Normalisasi: nomor telepon lewat `customer.NormalizePhone` (reuse Sprint 4), tanggal serial Excel lewat `excelize.ExcelDateToTime` plus fallback string format (termasuk `DD/MM/YY` yang ditemukan di data nyata).
- Commit membuat Owner/Outlet/Lead lewat service yang sudah ada dan teruji (`customer.Service`, `lead.Service`) — bukan SQL baru — diproses per-baris (bukan satu transaksi raksasa) sehingga satu baris gagal tidak menggagalkan seluruh batch.
- File hasil rejected rows: `GET /imports/{id}/rejected-rows/export` mengunduh CSV baris invalid berikut alasannya.
- Role gating: seluruh endpoint import ADMIN saja, sesuai literal roadmap ("Admin dapat mengimpor").
- OpenAPI diperbarui ke `0.15.0-sprint-14` (120 path, 90 schema), unit test untuk deteksi profil/header-hunting (dengan fixture header asli), sanitasi OTP, dan validasi baris.

Not Completed / Carry Over:
- (tidak ada)

Demo Evidence:
- Endpoint/Swagger: `POST /imports`, `GET /imports`, `GET /imports/{id}`, `GET /imports/{id}/rows`, `GET /imports/{id}/rejected-rows/export`, `POST /imports/{id}/commit` — `internal/platform/httpserver/openapi.yaml` v0.15.0-sprint-14.
- Skenario nyata: upload `01. Owner & Outlet 2026 (Copy).xlsx` (105 baris) tanpa `profile` → auto-detect `OWNER_OUTLET`, validasi 91 valid/5 invalid, commit 91/91 sukses (90 owner + 91 outlet, termasuk satu owner dengan 2 outlet lewat reuse kode). Upload `04. Data Belum Registrasi 2026 - User Temp (Copy).xlsx` dengan `profile=NON_REGISTER` eksplisit → validasi 48 valid/8 invalid, commit 48/48 sukses (owner minimal + lead per baris, nomor telepon duplikat antar-baris ditangani idempoten).
- Screenshot/log/test report: `docs/sprint-14/README.md` (skenario lengkap dengan request/response nyata, termasuk lima bug yang ditemukan & diperbaiki selama testing terhadap file asli).

Quality:
- Unit/integration test: `go test ./...` — seluruh paket PASS, termasuk test baru `internal/importing` (deteksi profil dengan fixture header asli, sanitasi OTP, validasi baris, normalisasi tanggal).
- Migration status: `20260726000100_import_framework.sql` — up/down/up reversibel, teruji di database terisolasi.
- Docker build: tidak diverifikasi ulang eksplisit sprint ini (tidak ada perubahan Dockerfile/compose); dependency baru (`excelize`) adalah pure-Go, tidak menambah risiko build container.
- Defect terbuka: tidak ada.

Impediments:
- (tidak ada)

Risiko Baru:
- Risiko: profil "Non-Register" menghasilkan Owner minimal (kode `NONREG-<telepon>`, nama placeholder "Prospek <telepon>") yang secara desain memang tidak lengkap — data ini perlu dilengkapi manual oleh Sales/Admin setelah diimpor.
  Dampak: kalau tidak ada proses tindak lanjut, banyak Owner "Prospek" tanpa nama asli menumpuk di sistem.
  Mitigasi: sudah sesuai desain (mencerminkan "belum registrasi" pada data sumber); tindak lanjut manual adalah bagian dari alur kerja Sales yang sudah ada (assign lead, followed oleh remark/interaction Sprint 6), bukan gap teknis.
  Owner: Tim Sales (proses bisnis, bukan backend).
- Risiko: performa commit satu-per-baris (bukan bulk insert) untuk file impor sangat besar (ribuan baris) bisa lambat karena tiap baris memanggil service layer penuh (termasuk validasi ulang di `customer.Service`/`lead.Service`).
  Dampak: import besar bisa makan waktu lama diproses worker (meski tidak menahan HTTP, karena async).
  Mitigasi: dicatat sebagai item review Sprint 17 (index dan query optimization, load test), tidak memblokir Sprint 14 — desain per-baris sengaja dipilih demi isolasi kegagalan (satu baris gagal tidak menggagalkan batch), trade-off yang disadari.
  Owner: Backend Engineer (Sprint 17).

Keputusan yang Dibutuhkan:
- (tidak ada — pemetaan kolom kedua profil didasarkan langsung pada file Excel asli kantor yang diperiksa saat perencanaan, bukan asumsi yang perlu dikonfirmasi ulang)

Rencana Sprint Berikutnya (Sprint 15 — Import Transaksi, Mitra, dan Data Sales):
- Framework import (`import_batches`/`import_rows`/`job_queue`/pola deteksi-profil-dari-header-asli) sudah siap dipakai ulang — Sprint 15 tinggal menambah profil baru (New & Subscribe, Monthly Active, Data Bonus Mitra, Data Call & Chat Sales) mengikuti pola yang sama persis, bukan membangun ulang.
- File Excel asli untuk Sprint 15 sudah teridentifikasi di `c:/piposmart/data_admin/` (`02. New & Subscribe`, `03. Nasabah Baru Per Provinsi`, `05. Monthly Active`, `06. Data Bonus Mitra`) dan `c:/piposmart/data_sales/` — perlu diinspeksi header asli sebelum planning, mengikuti kebiasaan yang sama seperti Sprint 14.
- Audit roadmap standar akan dijalankan sebelum Sprint 15 dimulai, mengecek Sprint 14 delivered persis sesuai DoD roadmap — hasil audit ini sudah tercermin di laporan ini (status GREEN, semua Committed Deliverables Completed).

