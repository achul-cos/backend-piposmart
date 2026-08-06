# Sprint 16e — Saldo Aplikasi Owner-Outlet, Status Kelolaan Outlet, dan Floating Dropdown Remarks

## Ringkasan

Sprint 16e berfokus pada penyempurnaan pengalaman pengguna (UX) dan transparansi informasi pada modul CRM Piposmart:
1. Menampilkan kembali kolom dan ringkasan **Saldo Aplikasi** (Wallet Balance) pada modul Owner-Outlet.
2. Menyederhanakan istilah status outlet pada tab Informasi Umum modul Kelolaan Outlet dari `"New Existing"` menjadi `"New"`.
3. Memperbaiki masalah visual menu dropdown **Remarks Customer** pada Form Laporan Aktivitas Call/Lead agar melayang bebas (*floating*) menggunakan React DOM Portal tanpa terpotong oleh kontainer modal maupun footer.

---

## Perubahan Frontend (`crm_piposmart`)

### 1. Integrasi Saldo Aplikasi Owner-Outlet
- **Halaman Utama Owner (`app/menu/owner-outlet/page.tsx`)**:
  - Menambahkan kolom header `<th>Saldo Aplikasi</th>` dan sel baris data `<WalletBalanceCell ownerId={owner.id} />` yang diformat rata kiri (*align left*).
  - Mengatur `colSpan` pesan *loading* dan *empty state* menjadi `11`.
- **Detail Owner (`app/menu/owner-outlet/[id]/page.tsx`)**:
  - Memasang komponen `<OwnerOverviewCard ownerId={ownerId} />` di bagian atas halaman detail untuk menampilkan kartu statistik **Saldo Berjalan**, **Total Transfer**, **Total Top Up**, dan **Total Terpakai**.
- **Kontrol Visibilitas Kolom (`app/components/table/ColumnVisibilityControl.tsx`)**:
  - Mengubah aturan `shouldHideByDefault` agar kolom Saldo Aplikasi langsung muncul secara otomatis saat tabel pertama kali dibuka.

### 2. Standarisasi Terminology Status Kelolaan Outlet (`app/menu/kelolaan-outlet/page.tsx`)
- Mengganti label filter `TIME_STATUS_OPTIONS` dari `"New Existing"` menjadi `"New"`.
- Mengubah fungsi helper `getOutletTimeStatus` agar mengembalikan status `"New"` untuk outlet baru yang terdaftar di bulan filter.
- Menyesuaikan `getTimeStatusBadgeClass` agar badge warna biru tampil secara presisi untuk status `"New"`.

### 3. Floating Dropdown Menu Remarks Customer (`app/menu/lead/call/remarks/page.tsx` & `app/menu/owner-outlet/call/remarks/page.tsx`)
- Menggunakan `React DOM Portal` (`createPortal`) ke `document.body` dengan `position: fixed` dan `z-index: 99999`.
- Mengatur posisi menu dropdown agar **selalu muncul terbuka ke bawah (`top: rect.bottom + 8px`)** tepat di bawah tombol input trigger "Pilih Remarks".
- Menambahkan pembatas `maxHeight` dinamis dan internal scroll agar menu tetap aman dan tidak melampaui batas layar.
- Mempertahankan penulisan `export const getRemarkScoreFromValue` dan `export const getRemarkLabelFromValue` demi menjaga kompatibilitas impor modul induk.

---

## Dokumentasi Terkait

- [sprint-16e.md](./sprint-16e.md) — Laporan resmi hasil pelaksanaan Sprint 16e.
