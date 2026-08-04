# Sprint 16c Report

Sprint: 16c  
Periode: 4 Agustus 2026  
Status: GREEN  

## Sprint Goal

- Mengubah arsitektur modul Lead dari berbasis Owner menjadi berbasis **Outlet** (*Outlet-Centric Architecture*).
- Mendukung pemetaan multi-outlet per owner, sehingga setiap outlet memiliki baris Lead tersendiri.
- Mengatasi bug query `COUNT(*)` dan mutasi argumen slice parameter SQL pada endpoint Lead.
- Menampilkan kolom **Nama Outlet** dan **Nama Owner** secara berdampingan dan jelas di frontend CRM.

## Completed

- Refactoring query `ListLeads` dan `leadSelect()` di `internal/lead/repository.go` dengan menjadikan `outlets ot JOIN owners o ON o.id = ot.owner_id` sebagai basis utama.
- Penggunaan `COALESCE` untuk secara cerdas menggabungkan atribut `outlet_id`, `outlet_code`, `outlet_name`, dan `outlet_phone` bila `customer_leads` belum memiliki `outlet_id` terpisah.
- Pembaruan `createLeadForOwner` di `internal/customer/repository.go` untuk otomatis mengaitkan `outlet_id` pertama saat pembuatan akun baru.
- Isolasi parameter `countArgs` dan `selectArgs` menggunakan `copy(...)` untuk mencegah runtime slice argument corruption pada pagination (`LIMIT ? OFFSET ?`).
- Perluasan filter `leadWhere` agar kata kunci pencarian `params.Query` mendukung pencarian nama outlet dan kode outlet.
- Penambahan logging `slog.Error` pada `writeError` di `internal/lead/handler.go`.
- Penyesuaian antarmuka `BackendLead` di frontend (`app/lib/api.ts`) untuk memuat objek `outlet: { id, code, name, phone }`.
- Penyesuaian tabel Lead (`app/menu/lead/page.tsx`), detail Lead (`app/menu/lead/[id]/page.tsx`), dan halaman telepon Lead (`app/menu/lead/call/page.tsx`).

## Quality

- Unit / Integration Build:
  - `go build ./...` PASS
  - `npx tsc --noEmit` PASS (0 errors)
- Log Diagnostic:
  - `slog.Error` aktif pada handler error untuk pemantauan runtime.

## Risiko & Mitigasi

- **Risiko**: Data historis lama yang dibuat sebelum Sprint 16c mungkin belum memiliki `outlet_id` di tabel `customer_leads`.
  - **Dampak**: Query Lead lama bisa tidak menampilkan nama outlet jika outlet dihapus.
  - **Mitigasi**: `COALESCE(cl.outlet_id, ot.id)` dan fallback query otomatis mengambil outlet aktif pertama milik owner sehingga data terjamin aman dan konsisten.
