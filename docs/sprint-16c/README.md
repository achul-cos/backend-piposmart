# Sprint 16c — Lead Outlet-Centric Architecture & Multi-Outlet Support

## Ringkasan

Sprint 16c menyempurnakan modul Lead pada backend dan frontend CRM Piposmart agar sepenuhnya berpusat pada **Outlet** (*Outlet-Centric Architecture*).

Sebelumnya, data Lead ditarik berbasis `owner_id` (1 baris per owner). Jika 1 Owner memiliki 2 atau lebih outlet (misal: "deny mart" dan "deny pos"), Lead hanya menampilkan 1 outlet saja.

Pada Sprint 16c, arsitektur query Lead diubah menjadi berbasis **Outlet**:
- Setiap outlet yang dimiliki oleh Owner memiliki baris data Lead tersendiri.
- Jika 1 Owner memiliki 2 outlet, daftar Lead akan menampilkan 2 baris data Lead secara terpisah (masing-masing memiliki Nama Outlet, Kode Outlet, dan No HP Outlet sendiri).
- Menyandingkan kolom **Nama Outlet** dan **Nama Owner** secara jelas pada tabel utama Lead.

## Perubahan Backend

### 1. Query Lead Berbasis Outlet (`internal/lead/repository.go`)
- Mengubah tabel utama query `ListLeads` dan `leadSelect()` dari `customer_leads cl` menjadi `outlets ot JOIN owners o ON o.id = ot.owner_id`.
- Menambahkan pembacaan otomatis via `COALESCE` ke outlet default owner jika lead belum memiliki `outlet_id` spesifik:
  - `COALESCE(cl.outlet_id, ot.id) AS outlet_id`
  - `COALESCE(ot.code, ot_default.code) AS outlet_code`
  - `COALESCE(ot.name, ot_default.name) AS outlet_name`
  - `COALESCE(ot.phone, ot_default.phone) AS outlet_phone`

### 2. Auto Link Outlet saat Pembuatan Lead Baru (`internal/customer/repository.go`)
- Memperbarui fungsi `createLeadForOwner` saat pembuatan Owner/Outlet baru agar langsung mengambil dan mengisi `outlet_id` pertama milik akun tersebut ke tabel `customer_leads`.

### 3. Perbaikan SQL Count & Slice Argument Mutation (`internal/lead/repository.go`)
- Memperbaiki query `COUNT(*)` pada `ListLeads` dengan menyertakan `JOIN outlets ot`.
- Mengisolasi parameter slice query `countArgs` dan `selectArgs` menggunakan `copy(...)` untuk mencegah kesalahan *slice argument mutation* saat pagination (`LIMIT ? OFFSET ?`) dan pencarian data.

### 4. Perluasan Filter Search & Order (`internal/lead/repository.go`)
- Memperluas `leadWhere` agar kata kunci pencarian `params.Query` (pencarian teks) mendukung pencarian nama outlet (`ot.name`) dan kode outlet (`ot.code`).
- Memperbarui `leadOrderBy` agar mendukung sorting berdasarkan tanggal dibuat outlet/lead dan nama outlet.

### 5. Logging Error Handler (`internal/lead/handler.go`)
- Menambahkan `slog.Error` pada `writeError` agar pesan kesalahan database/server pada modul Lead langsung tercetak di log konsol backend secara detail.

## Perubahan Frontend (`crm_piposmart`)

1. **Tabel Lead ([app/menu/lead/page.tsx](file:///d:/PT_piposmart/frontend/crm_piposmart/app/menu/lead/page.tsx))**:
   - Menambahkan kolom **Nama Outlet** dan **Nama Owner** secara terpisah dan berdampingan.
   - Kolom **Kode** dan **Kontak** memprioritaskan Kode Outlet dan No HP Outlet.
2. **Detail Lead ([app/menu/lead/[id]/page.tsx](file:///d:/PT_piposmart/frontend/crm_piposmart/app/menu/lead/[id]/page.tsx))**:
   - Header title menampilkan `Detail Lead: [Nama Outlet]`.
3. **Telepon/Call Lead ([app/menu/lead/call/page.tsx](file:///d:/PT_piposmart/frontend/crm_piposmart/app/menu/lead/call/page.tsx))**:
   - Profil customer utama memprioritaskan **Nama Outlet** sebagai pelanggan yang sedang difollow-up.

## Dokumentasi Terkait

- [sprint-16c.md](./sprint-16c.md) — Report resmi hasil Sprint 16c.
- [api-testing.md](./api-testing.md) — Panduan & contoh skenario API testing untuk modul Lead.
- [frontend-briefing.md](./frontend-briefing.md) — Acuan integrasi frontend untuk arsitektur Lead per-outlet.
