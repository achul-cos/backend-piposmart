# Sprint 16d — Penyelarasan UI Kelolaan Outlet & Mitra dan Perbaikan Build

## Ringkasan

Sprint 16d berfokus pada standarisasi antarmuka pengguna (UI) untuk modul "Kelolaan Outlet" dan "Kelolaan Mitra" pada aplikasi frontend, serta pembersihan environment backend dari file pengujian yang mengganggu proses *build*.

Meskipun pada backend hanya dilakukan perbaikan minor (*hotfix*), perubahan pada frontend mencakup perombakan besar untuk memastikan konsistensi fungsi antar menu.

## Perubahan Backend

### 1. Perbaikan Build & Dependensi (`test_query.go`)
- Menghapus file `test_query.go` (berisi eksperimen package `github.com/jmoiron/sqlx`) yang secara tidak sengaja masuk ke *repository* dan menyebabkan *compiler error* karena *missing module*.
- Memperbarui modul proyek dengan `go mod tidy` agar build environment backend kembali bersih, stabil, dan siap untuk integrasi atau *deployment*.

## Perubahan Frontend (`crm_piposmart`)

### 1. Sinkronisasi UI Kelolaan Outlet (`app/menu/kelolaan-outlet/page.tsx`)
- Merombak halaman utama Kelolaan Outlet agar memiliki desain yang identik dengan menu Owner.
- Menyederhanakan struktur Tab menjadi satu tabel terpadu "Daftar Outlet".
- Menambahkan **Stat Cards** di atas tabel (Total Outlet, Berlangganan, Trial, Belum Berlangganan).
- Mengaktifkan fitur pencarian canggih berbasis Autocomplete (*Filter Pintar*).
- Menambahkan fungsionalitas **Bulk Action** dengan *drag-to-select* untuk Ubah/Hapus massal.

### 2. Modul Sampah Kelolaan Outlet (`app/menu/kelolaan-outlet/trash/page.tsx`)
- Membuat halaman Trash (Sampah) baru untuk mengelola *Logical Delete* pada outlet.
- Mendukung fitur pengembalian data (*Restore*) ke daftar aktif, serta pembersihan data secara permanen (*Force Delete*).

### 3. Standarisasi Ikon Kelolaan Mitra
- Mengganti teks tombol aksi di halaman **Kelolaan Mitra** dan **Jenis Mitra** menjadi ikon berwarna (Biru untuk Detail, Oranye untuk Edit, Merah untuk Nonaktif, Hijau untuk Restore).
- Memberikan transisi efek warna pada saat *hover* untuk kenyamanan navigasi yang lebih modern.

## Dokumentasi Terkait

- [sprint-16d.md](./sprint-16d.md) — Laporan resmi hasil pelaksanaan Sprint 16d.
