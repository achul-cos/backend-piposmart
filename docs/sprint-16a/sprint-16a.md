# Sprint 16a Report

Sprint: 16a  
Periode: 3 Agustus 2026  
Status: GREEN

## Sprint Goal

- Menambahkan fondasi historical created date pada modul-modul operasional.
- Menambahkan filter tanggal dibuat lintas modul utama.
- Menambahkan export berbasis filter termasuk PDF.
- Menyempurnakan template report admin agar makin dekat ke workbook kantor.

## Completed

- Owner, outlet, global outlet menerima `created_at` opsional saat create.
- Lead menerima `created_at` opsional saat create.
- Package, plan, dan promotion menerima `created_at` opsional saat create.
- Activity interaction dan training menerima `created_at` opsional saat create.
- Wallet account, payment, dan ledger menerima filter `created_from/created_to`.
- Closing menerima filter `created_from/created_to`.
- Subscription order, subscription, reconciliation, dan issue menerima filter `created_from/created_to`.
- Partner module diperluas dengan historical created date pada:
  - partner type
  - partner
  - assignment
  - interaction
  - referral
- Importing module diperluas dengan:
  - filter `created_from/created_to` untuk batch dan rows
  - filter `uploaded_from/uploaded_to` untuk batch
- Reporting menerima filter:
  - `created_from`
  - `created_to`
- Export format baru:
  - `PDF`
- Report key admin tambahan:
  - `admin_owner_outlet`
  - `admin_new_subscribe`
  - `admin_nasabah_baru_provinsi`
- OpenAPI diperbarui untuk:
  - enum report key baru
  - format export `PDF`
  - filter `created_from/created_to`
  - filter `uploaded_from/uploaded_to`
  - request `created_at` pada partner-related create endpoints

## Not Completed / Catatan Lanjutan

- Template workbook admin yang sangat kompleks dan multi-sheet belum disalin 100%.
- Report admin baru masih fokus pada struktur data dan kompatibilitas export, belum pixel-perfect
  mengikuti seluruh visual workbook kantor.
- Export admin untuk workbook lain di `data_admin` masih bisa dilanjutkan satu per satu tanpa
  mengubah kontrak route reporting yang sudah ada.

## Risiko

- Sebagian workbook admin lama memiliki rumus manual dan susunan kolom yang sangat spesifik.
- Filter historis lintas modul menambah variasi query sehingga dokumentasi frontend harus sangat
  jelas agar tidak salah mengirim pasangan filter tanggal.
- PDF export saat ini fokus pada keterbacaan tabel, bukan reproduksi visual 100% workbook Excel.

## Mitigasi

- Memecah workbook admin menjadi report key terpisah.
- Menjaga route reporting generik tetap stabil.
- Menjaga perubahan schema seminimal mungkin dengan tetap memakai `created_at` yang sudah ada.
- Menambah dokumentasi request/response/error agar frontend dapat melakukan validasi sebelum kirim.
