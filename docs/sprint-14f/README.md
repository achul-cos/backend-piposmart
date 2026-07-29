# API Documentation Update - Sprint 14f Full List Routes, Identity, & OpenAPI

## 1. Informasi Dokumen

| Item | Nilai |
| --- | --- |
| Project | Backend CRM Piposmart |
| Sprint | 14f - Full List Routes, Identity, & OpenAPI Update |
| Tanggal Update | 29 Juli 2026 |
| Environment | Backend Local Development |
| Fokus | Penambahan route `/all`, `/all-deleted`, account management Admin/Supervisor, dan update OpenAPI |

## 2. Ringkasan Tujuan

Sprint 14f menambahkan opsi endpoint baru agar frontend bisa mengambil seluruh data tanpa terbatasi pagination default maksimal 100 item, sekaligus menambahkan route manajemen akun untuk Admin dan Supervisor.

Perubahan ini dibuat dengan prinsip:

- route lama tetap dipertahankan;
- response shape tetap kompatibel;
- filter query tetap mengikuti route list lama;
- route baru hanya menambah opsi, bukan mengganti perilaku lama;
- penambahan manajemen akun tidak mengubah flow login/auth existing.

## 3. Aturan Umum Route Baru

### 3.1 Route Tambahan

Setiap modul/list utama yang relevan sekarang memiliki pola:

- `GET /...`
  - list normal dengan pagination
- `GET /.../all`
  - list semua data tanpa limit/pagination query
- `GET /.../all-deleted`
  - list semua data termasuk soft deleted jika modul tersebut mendukung soft delete

### 3.2 Perilaku Response

Walaupun route `/all` tidak memakai pagination untuk mengambil data, backend tetap mengembalikan metadata pagination/list meta agar frontend lama tidak rusak.

Artinya:

- struktur response lama tetap dipakai;
- `items` berisi seluruh data hasil filter;
- `pagination.limit` atau `meta.limit` akan berisi jumlah item hasil return pada route `/all`.

### 3.3 Catatan Penting untuk Frontend

- Untuk tabel biasa yang memang memakai paging, tetap gunakan route list lama.
- Untuk kebutuhan dropdown besar, export helper, bulk selector, sinkronisasi cache lokal, atau preload data, gunakan route `/all`.
- Untuk trash table / data sampah, gunakan:
  - `/trash` bila ingin tetap berpaginasi
  - `/all-deleted` bila ingin mengambil semuanya sekaligus

## 4. Modul yang Sudah Mendapatkan Route Baru

### 4.1 Customer / Owner / Outlet

- `GET /api/v1/owners/all`
- `GET /api/v1/owners/all-deleted`
- `GET /api/v1/owners/{owner_id}/outlets/all`
- `GET /api/v1/owners/{owner_id}/outlets/all-deleted`
- `GET /api/v1/outlets/all`
- `GET /api/v1/outlets/all-deleted`
- `GET /api/v1/outlets/subscription-statuses/all`

### 4.2 Catalog

- `GET /api/v1/catalog/packages/all`
- `GET /api/v1/catalog/packages/all-deleted`
- `GET /api/v1/catalog/plans/all`
- `GET /api/v1/catalog/plans/all-deleted`
- `GET /api/v1/catalog/promotions/all`
- `GET /api/v1/catalog/promotions/all-deleted`

### 4.3 Lead & Activity

- `GET /api/v1/leads/all`
- `GET /api/v1/leads/all-deleted`
- `GET /api/v1/customer-interactions/all`
- `GET /api/v1/customer-interactions/all-deleted`
- `GET /api/v1/follow-ups/all`
- `GET /api/v1/follow-ups/all-deleted`
- `GET /api/v1/leads/{lead_id}/interactions/all`
- `GET /api/v1/leads/{lead_id}/interactions/all-deleted`
- `GET /api/v1/trainings/all`
- `GET /api/v1/trainings/all-deleted`
- `GET /api/v1/leads/{lead_id}/trainings/all`
- `GET /api/v1/leads/{lead_id}/trainings/all-deleted`

### 4.4 Closing

- `GET /api/v1/closings/all`
- `GET /api/v1/closings/all-deleted`

### 4.5 Wallet

- `GET /api/v1/wallets/all`
- `GET /api/v1/wallets/all-deleted`
- `GET /api/v1/wallet-payments/all`
- `GET /api/v1/wallet-payments/all-deleted`
- `GET /api/v1/wallet-transactions/all`
- `GET /api/v1/wallet-transactions/all-deleted`
- `GET /api/v1/owners/{owner_id}/wallet/transactions/all`
- `GET /api/v1/owners/{owner_id}/wallet/transactions/all-deleted`

### 4.6 Subscription & Reconciliation

- `GET /api/v1/subscription-orders/all`
- `GET /api/v1/subscription-orders/all-deleted`
- `GET /api/v1/subscriptions/all`
- `GET /api/v1/subscriptions/all-deleted`
- `GET /api/v1/reconciliations/all`
- `GET /api/v1/reconciliations/all-deleted`
- `GET /api/v1/reconciliation-issues/all`
- `GET /api/v1/reconciliation-issues/all-deleted`

### 4.7 Importing

- `GET /api/v1/imports/all`
- `GET /api/v1/imports/all-deleted`
- `GET /api/v1/imports/{id}/rows/all`
- `GET /api/v1/imports/{id}/rows/all-deleted`

### 4.8 Target

- `GET /api/v1/sales-targets/all`
- `GET /api/v1/sales-targets/all-deleted`

### 4.9 Partner

- `GET /api/v1/partners/all`
- `GET /api/v1/partners/all-deleted`
- `GET /api/v1/partners/{partnerID}/interactions/all`
- `GET /api/v1/partners/{partnerID}/interactions/all-deleted`
- `GET /api/v1/partners/{partnerID}/commissions/all`
- `GET /api/v1/partners/{partnerID}/commissions/all-deleted`
- `GET /api/v1/partners/{partnerID}/payouts/all`
- `GET /api/v1/partners/{partnerID}/payouts/all-deleted`

### 4.10 Identity / User Management

- `POST /api/v1/admins`
- `POST /api/v1/admins/{id}/reset-password`
- `GET /api/v1/supervisors`
- `POST /api/v1/supervisors`
- `POST /api/v1/supervisors/{id}/reset-password`
- `POST /api/v1/sales/{id}/reset-password`

Catatan otorisasi:

- Admin dapat membuat akun Admin.
- Admin dapat membuat akun Supervisor.
- Admin dapat reset password Admin, Supervisor, dan Sales.
- Supervisor tetap hanya dapat membuat/mengelola Sales sesuai permission yang sudah ada sebelumnya.

## 5. Modul yang `all-deleted` Hanya Bersifat Alias Kompatibilitas

Tidak semua modul memang punya soft delete pada list view-nya. Untuk modul seperti ini, route `/all-deleted` tetap disediakan agar frontend punya pola route yang seragam, tetapi secara perilaku saat ini hasilnya setara alias `/all`.

Contoh modul alias kompatibilitas:

- leads
- customer interactions
- follow-ups
- trainings
- wallet payments
- wallet transactions
- subscription orders
- reconciliations
- reconciliation issues
- imports
- sales targets
- beberapa list partner domain

## 6. Update OpenAPI

File OpenAPI dan dokumen testing telah diperbarui pada:

- [internal/platform/httpserver/openapi.yaml](/abs/path/C:/piposmart/backend_crm_piposmart/internal/platform/httpserver/openapi.yaml)
- [API Testing Sprint 14f](./api-testing.md)

Pembaruan OpenAPI mencakup:

- bump versi spec menjadi `0.15.4-sprint-14f-identity`;
- deskripsi baru pada `info.description`;
- dokumentasi path baru `/all` dan `/all-deleted`;
- dokumentasi route account management Admin/Supervisor;
- dokumentasi reset password untuk Admin/Supervisor/Sales;
- penjelasan mana yang benar-benar include soft deleted;
- penjelasan mana yang alias compatibility frontend.

## 7. Contoh Pemakaian Frontend

### 7.1 Ambil Semua Owner untuk Bulk Select

```http
GET /api/v1/owners/all?name=laundry&city=bandung&sort=name
Authorization: Bearer {access_token}
Accept: application/json
```

### 7.2 Ambil Semua Owner Termasuk Sampah

```http
GET /api/v1/owners/all-deleted?sort=-updated_at
Authorization: Bearer {access_token}
Accept: application/json
```

### 7.3 Ambil Semua Wallet Owner

```http
GET /api/v1/wallets/all?sort=-balance
Authorization: Bearer {access_token}
Accept: application/json
```

### 7.4 Ambil Semua Row Import Tanpa Paging

```http
GET /api/v1/imports/15/rows/all?status=INVALID
Authorization: Bearer {access_token}
Accept: application/json
```

### 7.5 Membuat Akun Supervisor

```http
POST /api/v1/supervisors
Authorization: Bearer {admin_access_token}
Content-Type: application/json
Accept: application/json
```

```json
{
  "code": "SPV-NEW-001",
  "name": "Supervisor Area Bandung",
  "email": "supervisor.bandung@piposmart.test",
  "phone": "081234567890",
  "password": "TempPass123"
}
```

### 7.6 Reset Password Admin / Supervisor / Sales

Contoh reset password Supervisor:

```http
POST /api/v1/supervisors/12/reset-password
Authorization: Bearer {admin_access_token}
Content-Type: application/json
Accept: application/json
```

```json
{
  "new_password": "ResetPass123"
}
```

## 8. Validasi Teknis

Validasi minimal yang dilakukan pada update Sprint 14f:

```powershell
go build ./...
go test ./internal/identity/...
go test ./internal/platform/httpserver/...
```

Hasil:

- PASS

Dokumentasi testing API lengkap tersedia pada:

- [api-testing.md](./api-testing.md)

## 9. Dampak ke Frontend

Dengan perubahan ini frontend sekarang punya 3 pilihan pola konsumsi data:

1. gunakan route list biasa untuk tabel paginated;
2. gunakan route `/trash` bila ingin tabel sampah tapi tetap paginated;
3. gunakan route `/all` atau `/all-deleted` bila ingin mengambil seluruh dataset sekaligus.

Ini terutama membantu pada:

- dropdown/select data besar,
- modal bulk action,
- trash table penuh,
- export helper,
- preload cache lokal frontend,
- sinkronisasi data antar-tab atau antar-state UI.

## 10. Catatan Implementasi

- Kode route lama tidak diubah perilaku bisnisnya.
- Route baru ditambahkan sebagai sub-route agar aman untuk frontend existing.
- Untuk modul soft delete, `/all-deleted` memetakan ke scope unscoped/full.
- Untuk modul non-soft-delete, `/all-deleted` tetap ada demi konsistensi kontrak frontend.
