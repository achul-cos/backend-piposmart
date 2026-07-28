# API Testing & Frontend Integration Guide - Sprint 14 Import Framework dan Data Customer

## 1. Informasi Dokumen

| Item | Nilai |
| --- | --- |
| Project | Backend CRM Piposmart |
| Modul | Import Framework dan Data Customer |
| Sprint | Sprint 14 |
| Update Dokumen Terakhir | 28 Juli 2026 |
| Environment Pengujian | Local Development |
| API Base URL | `http://localhost:8093/api/v1` |
| Format Auth | JWT Bearer Token |
| Fokus Dokumen | Panduan implementasi frontend + laporan smoke test manual API |

Dokumen ini adalah pembaruan dari laporan Sprint 14 sebelumnya. Tujuan utamanya sekarang bukan hanya menunjukkan bahwa endpoint bekerja, tetapi juga memberi panduan yang jelas untuk frontend tentang:

- urutan endpoint yang benar,
- state yang mungkin muncul,
- kapan tombol/action di UI boleh diaktifkan,
- contoh request sukses,
- contoh response error,
- dan cara menangani retry atau race condition ringan di sisi client.

## 2. Ringkasan Perubahan per 28 Juli 2026

Per 28 Juli 2026 terdapat pembaruan perilaku backend pada modul importing:

1. `POST /imports/{id}/commit` sekarang lebih idempoten.
   - Jika batch masih `COMMITTING`, request commit ulang tidak lagi dianggap gagal secara bisnis.
   - Jika batch sudah `COMMITTED`, request commit ulang akan mengembalikan batch yang sudah selesai, bukan memaksa proses baru.

2. Error `INVALID_BATCH_STATUS` sekarang lebih informatif.
   - Response error menyertakan `action`, `current_status`, `allowed_statuses`, dan `hint`.
   - Ini dibuat agar frontend dapat menampilkan pesan yang lebih tepat dan memutuskan aksi berikutnya tanpa menebak-nebak.

3. Worker commit dibuat lebih tahan terhadap race condition ringan.
   - Jika job commit sudah masuk worker ketika status batch masih `VALIDATED` tetapi belum sempat ditulis menjadi `COMMITTING`, worker tetap dapat melanjutkan dengan aman.

4. Saat batch masuk fase `COMMITTING`, `progress_percentage` di-reset ke `0` lalu naik lagi selama commit berjalan.`r`n`r`n5. Pesan bentrok kode owner/outlet sekarang lebih jelas.`r`n   - Response `CODE_ALREADY_USED` menjelaskan bahwa penyebabnya bisa karena data memang sudah pernah terdaftar atau merupakan duplikat.`r`n   - Dokumentasi ini juga menambahkan panduan bagaimana frontend melakukan pengecekan kandidat duplikasi owner sebelum user memutuskan tetap menyimpan data baru.

## 3. Header Standar

Semua endpoint JSON pada modul import menggunakan header berikut:

```http
Authorization: Bearer {access_token}
Accept: application/json
```

Untuk upload file:

```http
Authorization: Bearer {access_token}
Accept: application/json
Content-Type: multipart/form-data
```

## 4. Cara Frontend Harus Menggunakan API Ini

Alur yang benar adalah sebagai berikut:

1. Login sebagai Admin.
2. Upload file Excel ke `POST /imports`.
3. Simpan `id` batch dari response upload.
4. Poll `GET /imports/{id}` sampai status berubah dari `UPLOADED` / `VALIDATING` menjadi salah satu final validasi:
   - `VALIDATED`
   - `VALIDATION_FAILED`
5. Jika status `VALIDATED`, frontend boleh:
   - menampilkan ringkasan total/valid/invalid,
   - menampilkan tabel row preview via `GET /imports/{id}/rows`,
   - menampilkan tombol download rejected rows,
   - mengaktifkan tombol commit.
6. Jika user menyetujui preview, panggil `POST /imports/{id}/commit`.
7. Poll lagi `GET /imports/{id}` sampai status:
   - `COMMITTING`
   - lalu final ke `COMMITTED`
8. Setelah `COMMITTED`, frontend dapat menampilkan hasil akhir dan menonaktifkan tombol commit.

Catatan penting untuk frontend:

- Jangan langsung memanggil commit tepat setelah upload, karena validasi berjalan async.
- Tombol commit hanya layak aktif jika status batch sudah `VALIDATED`.
- Jika user double click tombol commit, backend sekarang aman menangani retry untuk status `COMMITTING` atau `COMMITTED`.
- Jika file yang diupload identik byte-per-byte dengan file yang pernah diupload sebelumnya, backend dapat mengembalikan batch lama yang sama karena dedup SHA-256.

## 5. State Machine Batch yang Wajib Dipahami Frontend

### 5.1 Batch Status

| Status | Arti | Aksi Frontend yang Disarankan |
| --- | --- | --- |
| `UPLOADED` | File sudah diterima API, worker validasi belum mulai | tampilkan status “menunggu validasi”, polling detail batch |
| `VALIDATING` | Worker sedang parsing dan validasi file | tampilkan progress, tombol commit tetap disabled |
| `VALIDATED` | Preview siap, total/valid/invalid sudah final | tampilkan summary, izinkan commit |
| `VALIDATION_FAILED` | File gagal divalidasi total, mis. profil tak cocok / workbook rusak | tampilkan error, jangan izinkan commit |
| `COMMITTING` | Worker sedang membuat owner/outlet/lead | tampilkan progress commit, tombol commit disabled |
| `COMMITTED` | Proses commit selesai | tampilkan hasil akhir, tombol commit disabled |
| `COMMIT_FAILED` | Disediakan pada model, tetapi alur saat ini pada praktiknya lebih banyak mencatat gagal per-row ketimbang menggagalkan seluruh batch | tampilkan sebagai status gagal jika suatu saat dipakai |

### 5.2 Row Status

| Status | Arti |
| --- | --- |
| `VALID` | Baris lolos validasi dan siap dikomit |
| `INVALID` | Baris gagal validasi dan tidak akan dikomit |
| `COMMITTED` | Baris valid berhasil dibuat ke entity tujuan |
| `COMMIT_FAILED` | Baris valid gagal saat commit, contoh bentrok data atau error service |

## 6. Ringkasan Endpoint

| Endpoint | Method | Fungsi | Role |
| --- | --- | --- | --- |
| `/auth/login` | POST | Ambil access token JWT | semua user valid |
| `/imports` | POST | Upload file Excel import | Admin |
| `/imports` | GET | List batch import | Admin |
| `/imports/{id}` | GET | Detail batch import | Admin |
| `/imports/{id}/rows` | GET | List row preview / hasil commit | Admin |
| `/imports/{id}/rejected-rows/export` | GET | Download CSV row invalid | Admin |
| `/imports/{id}/commit` | POST | Jalankan commit batch tervalidasi | Admin |

## 7. Login untuk Mendapatkan JWT

### 7.1 Request

```http
POST /api/v1/auth/login
Content-Type: application/json
Accept: application/json
```

```json
{
  "email": "admin@piposmart.id",
  "password": "ChangeMe123!"
}
```

### 7.2 Response Sukses

**Status:** `200 OK`

```json
{
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIs...",
    "refresh_token": "...",
    "token_type": "Bearer",
    "expires_in": 3600,
    "refresh_token_expires_at": "2026-07-29T09:00:00Z",
    "user": {
      "id": 1,
      "name": "Admin User",
      "email": "admin@piposmart.id",
      "role": "ADMIN",
      "status": "ACTIVE"
    }
  },
  "meta": {
    "request_id": "req-login-001"
  }
}
```

Frontend selanjutnya cukup mengirim header:

```http
Authorization: Bearer {access_token}
```

## 8. Detail Endpoint Import

## 8.1 Upload Import Batch

### Endpoint

```http
POST /api/v1/imports
```

### Kapan Dipakai

Saat user Admin memilih file `.xlsx` untuk mulai proses import.

### Form Data

| Field | Tipe | Wajib | Keterangan |
| --- | --- | --- | --- |
| `file` | file | ya | file Excel `.xlsx` |
| `profile` | string | tidak | `OWNER_OUTLET` atau `NON_REGISTER`; kosongkan untuk auto-detect |

### Contoh Request

```bash
curl -X POST "http://localhost:8093/api/v1/imports" \
  -H "Authorization: Bearer {token}" \
  -H "Accept: application/json" \
  -F "file=@c:/piposmart/data_admin/01. Owner & Outlet 2026 (Copy).xlsx"
```

### Response Sukses

**Status:** `201 Created`

```json
{
  "data": {
    "id": 15,
    "code": "IMPORT-20260728-8e51fa9a1c30",
    "profile": "PENDING_DETECTION",
    "original_filename": "01. Owner & Outlet 2026 (Copy).xlsx",
    "status": "UPLOADED",
    "total_rows": 0,
    "valid_rows": 0,
    "invalid_rows": 0,
    "committed_rows": 0,
    "progress_percentage": 0,
    "uploaded_by": {
      "id": 1,
      "name": "Admin User"
    },
    "uploaded_at": "2026-07-28T09:10:00Z",
    "created_at": "2026-07-28T09:10:00Z",
    "updated_at": "2026-07-28T09:10:00Z"
  },
  "meta": {
    "request_id": "req-upload-001"
  }
}
```

### Arti Response untuk Frontend

- `id` harus disimpan untuk polling berikutnya.
- `profile=PENDING_DETECTION` bukan error. Ini berarti backend akan mendeteksi profil di worker.
- Setelah upload sukses, frontend **jangan** langsung memanggil commit.

### Error Cases Penting

#### A. File tidak dikirim

**Status:** `400 Bad Request`

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "file wajib diunggah",
    "request_id": "req-upload-err-001"
  }
}
```

#### B. Ekstensi bukan `.xlsx`

**Status:** `400 Bad Request`

```json
{
  "error": {
    "code": "INVALID_FILE_TYPE",
    "message": "importing: only .xlsx files are accepted",
    "request_id": "req-upload-err-002"
  }
}
```

#### C. File terlalu besar

**Status:** `400 Bad Request`

```json
{
  "error": {
    "code": "FILE_TOO_LARGE",
    "message": "importing: file exceeds the maximum allowed size",
    "request_id": "req-upload-err-003"
  }
}
```

#### D. Profile tidak dikenal

Contoh request salah:

```bash
curl -X POST "http://localhost:8093/api/v1/imports" \
  -H "Authorization: Bearer {token}" \
  -F "file=@./sample.xlsx" \
  -F "profile=OWNER"
```

**Status:** `400 Bad Request`

```json
{
  "error": {
    "code": "UNKNOWN_PROFILE",
    "message": "importing: unknown profile",
    "request_id": "req-upload-err-004"
  }
}
```

#### E. User bukan Admin

**Status:** `403 Forbidden`

```json
{
  "error": {
    "code": "FORBIDDEN",
    "message": "importing: forbidden",
    "request_id": "req-upload-err-005"
  }
}
```

#### F. Re-upload file identik

Jika file identik byte-per-byte pernah diupload sebelumnya, response dapat mengembalikan batch lama yang sama.

Ini **bukan error**. Frontend harus melihat `id` dan `status` batch yang dikembalikan, lalu melanjutkan dari state tersebut.

## 8.2 List Import Batch

### Endpoint

```http
GET /api/v1/imports?status=VALIDATED&profile=OWNER_OUTLET&page=1&limit=20
```

### Kapan Dipakai

Untuk halaman tabel daftar import batch.

### Query Parameter

| Param | Wajib | Keterangan |
| --- | --- | --- |
| `status` | tidak | filter status batch |
| `profile` | tidak | filter profil |
| `page` | tidak | default `1` |
| `limit` | tidak | default `20` |

### Response Sukses

**Status:** `200 OK`

```json
{
  "data": {
    "items": [
      {
        "id": 15,
        "code": "IMPORT-20260728-8e51fa9a1c30",
        "profile": "OWNER_OUTLET",
        "original_filename": "01. Owner & Outlet 2026 (Copy).xlsx",
        "status": "VALIDATED",
        "total_rows": 96,
        "valid_rows": 91,
        "invalid_rows": 5,
        "committed_rows": 0,
        "progress_percentage": 100,
        "uploaded_by": {
          "id": 1,
          "name": "Admin User"
        },
        "uploaded_at": "2026-07-28T09:10:00Z",
        "validated_at": "2026-07-28T09:10:09Z",
        "created_at": "2026-07-28T09:10:00Z",
        "updated_at": "2026-07-28T09:10:09Z"
      }
    ],
    "meta": {
      "page": 1,
      "limit": 20,
      "total": 1
    }
  },
  "meta": {
    "request_id": "req-list-batch-001"
  }
}
```

### Error Case

#### A. User bukan Admin

**Status:** `403 Forbidden`

```json
{
  "error": {
    "code": "FORBIDDEN",
    "message": "importing: forbidden",
    "request_id": "req-list-batch-err-001"
  }
}
```

## 8.3 Get Detail Batch

### Endpoint

```http
GET /api/v1/imports/{id}
```

### Kapan Dipakai

Untuk polling progress dan detail summary batch.

### Response Sukses Saat Validasi Masih Berjalan

```json
{
  "data": {
    "id": 15,
    "code": "IMPORT-20260728-8e51fa9a1c30",
    "profile": "PENDING_DETECTION",
    "original_filename": "01. Owner & Outlet 2026 (Copy).xlsx",
    "status": "VALIDATING",
    "total_rows": 0,
    "valid_rows": 0,
    "invalid_rows": 0,
    "committed_rows": 0,
    "progress_percentage": 42,
    "uploaded_by": {
      "id": 1,
      "name": "Admin User"
    },
    "uploaded_at": "2026-07-28T09:10:00Z",
    "created_at": "2026-07-28T09:10:00Z",
    "updated_at": "2026-07-28T09:10:04Z"
  },
  "meta": {
    "request_id": "req-batch-detail-001"
  }
}
```

### Response Sukses Saat Sudah Validated

```json
{
  "data": {
    "id": 15,
    "code": "IMPORT-20260728-8e51fa9a1c30",
    "profile": "OWNER_OUTLET",
    "original_filename": "01. Owner & Outlet 2026 (Copy).xlsx",
    "status": "VALIDATED",
    "total_rows": 96,
    "valid_rows": 91,
    "invalid_rows": 5,
    "committed_rows": 0,
    "progress_percentage": 100,
    "uploaded_by": {
      "id": 1,
      "name": "Admin User"
    },
    "uploaded_at": "2026-07-28T09:10:00Z",
    "validated_at": "2026-07-28T09:10:09Z",
    "created_at": "2026-07-28T09:10:00Z",
    "updated_at": "2026-07-28T09:10:09Z"
  },
  "meta": {
    "request_id": "req-batch-detail-002"
  }
}
```

### Response Sukses Saat Commit Selesai

```json
{
  "data": {
    "id": 15,
    "code": "IMPORT-20260728-8e51fa9a1c30",
    "profile": "OWNER_OUTLET",
    "original_filename": "01. Owner & Outlet 2026 (Copy).xlsx",
    "status": "COMMITTED",
    "total_rows": 96,
    "valid_rows": 91,
    "invalid_rows": 5,
    "committed_rows": 91,
    "progress_percentage": 100,
    "uploaded_by": {
      "id": 1,
      "name": "Admin User"
    },
    "committed_by": {
      "id": 1,
      "name": "Admin User"
    },
    "uploaded_at": "2026-07-28T09:10:00Z",
    "validated_at": "2026-07-28T09:10:09Z",
    "committed_at": "2026-07-28T09:10:21Z",
    "created_at": "2026-07-28T09:10:00Z",
    "updated_at": "2026-07-28T09:10:21Z"
  },
  "meta": {
    "request_id": "req-batch-detail-003"
  }
}
```

### Error Case

#### A. Batch tidak ditemukan

**Status:** `404 Not Found`

```json
{
  "error": {
    "code": "NOT_FOUND",
    "message": "importing: data not found",
    "request_id": "req-batch-detail-err-001"
  }
}
```

## 8.4 List Rows dalam Batch

### Endpoint

```http
GET /api/v1/imports/{id}/rows?status=INVALID&page=1&limit=50
```

### Kapan Dipakai

Untuk halaman preview import, tabel invalid rows, atau hasil commit per-row.

### Query Parameter

| Param | Wajib | Keterangan |
| --- | --- | --- |
| `status` | tidak | `VALID`, `INVALID`, `COMMITTED`, `COMMIT_FAILED` |
| `page` | tidak | default `1` |
| `limit` | tidak | default `50` |

### Response Sukses

```json
{
  "data": {
    "items": [
      {
        "id": 190,
        "batch_id": 15,
        "row_index": 37,
        "raw_payload": {
          "owner_code": "OWN-000321",
          "owner_name": "Laundry Maju Jaya",
          "owner_phone": "",
          "outlet_name": "Laundry Maju Jaya Cabang 1",
          "outlet_phone": "081234567890",
          "province": "Jawa Timur",
          "city": "Surabaya",
          "address": "Jl. Mawar No. 10"
        },
        "status": "INVALID",
        "validation_errors": [
          "owner_phone wajib diisi"
        ]
      }
    ],
    "meta": {
      "page": 1,
      "limit": 50,
      "total": 1
    }
  },
  "meta": {
    "request_id": "req-rows-001"
  }
}
```

### Error Cases

#### A. Batch ID tidak valid

**Status:** `400 Bad Request`

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "invalid batch ID",
    "request_id": "req-rows-err-001"
  }
}
```

#### B. Batch tidak ditemukan

**Status:** `404 Not Found`

```json
{
  "error": {
    "code": "NOT_FOUND",
    "message": "importing: data not found",
    "request_id": "req-rows-err-002"
  }
}
```

## 8.5 Export Rejected Rows

### Endpoint

```http
GET /api/v1/imports/{id}/rejected-rows/export
```

### Kapan Dipakai

Untuk mengunduh daftar row invalid sebagai CSV agar Admin bisa review atau koreksi sumber Excel.

### Response Sukses

**Status:** `200 OK`

**Content-Type:** `text/csv`

Contoh isi file:

```csv
row_index,raw_payload,validation_errors
37,"{""owner_code"":""OWN-000321"",...}","owner_phone wajib diisi"
41,"{""owner_code"":""OWN-000322"",...}","outlet_phone tidak valid"
```

### Catatan Frontend

- Endpoint ini tidak mengembalikan JSON saat sukses, tetapi file CSV.
- Jika jumlah row invalid = 0, backend tetap dapat mengembalikan CSV dengan header saja.
- Frontend sebaiknya menampilkan tombol export setelah batch sudah `VALIDATED` atau setelah user sudah membuka preview hasil validasi.

### Error Cases

#### A. Batch tidak ditemukan

**Status:** `404 Not Found`

```json
{
  "error": {
    "code": "NOT_FOUND",
    "message": "importing: data not found",
    "request_id": "req-export-err-001"
  }
}
```

#### B. User bukan Admin

**Status:** `403 Forbidden`

```json
{
  "error": {
    "code": "FORBIDDEN",
    "message": "importing: forbidden",
    "request_id": "req-export-err-002"
  }
}
```

## 8.6 Commit Batch

### Endpoint

```http
POST /api/v1/imports/{id}/commit
```

### Kapan Dipakai

Hanya setelah `GET /imports/{id}` menunjukkan `status=VALIDATED`.

### Request

Tidak perlu body.

```bash
curl -X POST "http://localhost:8093/api/v1/imports/15/commit" \
  -H "Authorization: Bearer {token}" \
  -H "Accept: application/json"
```

### Response Sukses Pertama Kali

**Status:** `202 Accepted`

```json
{
  "data": {
    "id": 15,
    "code": "IMPORT-20260728-8e51fa9a1c30",
    "profile": "OWNER_OUTLET",
    "original_filename": "01. Owner & Outlet 2026 (Copy).xlsx",
    "status": "COMMITTING",
    "total_rows": 96,
    "valid_rows": 91,
    "invalid_rows": 5,
    "committed_rows": 0,
    "progress_percentage": 0,
    "uploaded_by": {
      "id": 1,
      "name": "Admin User"
    },
    "uploaded_at": "2026-07-28T09:10:00Z",
    "validated_at": "2026-07-28T09:10:09Z",
    "created_at": "2026-07-28T09:10:00Z",
    "updated_at": "2026-07-28T09:10:12Z"
  },
  "meta": {
    "request_id": "req-commit-001"
  }
}
```

### Response Saat Commit Diulang Ketika Masih `COMMITTING`

**Status:** `202 Accepted`

```json
{
  "data": {
    "id": 15,
    "status": "COMMITTING",
    "progress_percentage": 37,
    "committed_rows": 0
  },
  "meta": {
    "request_id": "req-commit-002"
  }
}
```

Arti bisnisnya: request retry aman, frontend cukup lanjut polling `GET /imports/{id}`.

### Response Saat Commit Diulang Ketika Sudah `COMMITTED`

**Status:** `200 OK`

```json
{
  "data": {
    "id": 15,
    "status": "COMMITTED",
    "committed_rows": 91,
    "progress_percentage": 100,
    "committed_at": "2026-07-28T09:10:21Z"
  },
  "meta": {
    "request_id": "req-commit-003"
  }
}
```

Arti bisnisnya: backend tidak menjalankan commit kedua, hanya mengembalikan hasil akhir batch yang memang sudah selesai.

### Error Cases Penting

#### A. Commit dipanggil terlalu cepat, saat batch masih `UPLOADED`

**Status:** `400 Bad Request`

```json
{
  "error": {
    "code": "INVALID_BATCH_STATUS",
    "message": "importing: batch is not in a state that allows this action (action=commit, current_status=UPLOADED, allowed_statuses=VALIDATED, COMMITTING, COMMITTED)",
    "details": {
      "action": "commit",
      "current_status": "UPLOADED",
      "allowed_statuses": [
        "VALIDATED",
        "COMMITTING",
        "COMMITTED"
      ],
      "hint": "poll GET /imports/{id} until status VALIDATED before first commit; retry commit safely if status COMMITTING or COMMITTED"
    },
    "request_id": "req-commit-err-001"
  }
}
```

#### B. Commit dipanggil saat batch masih `VALIDATING`

**Status:** `400 Bad Request`

```json
{
  "error": {
    "code": "INVALID_BATCH_STATUS",
    "message": "importing: batch is not in a state that allows this action (action=commit, current_status=VALIDATING, allowed_statuses=VALIDATED, COMMITTING, COMMITTED)",
    "details": {
      "action": "commit",
      "current_status": "VALIDATING",
      "allowed_statuses": [
        "VALIDATED",
        "COMMITTING",
        "COMMITTED"
      ],
      "hint": "poll GET /imports/{id} until status VALIDATED before first commit; retry commit safely if status COMMITTING or COMMITTED"
    },
    "request_id": "req-commit-err-002"
  }
}
```

#### C. Commit dipanggil saat validasi total gagal

**Status:** `400 Bad Request`

```json
{
  "error": {
    "code": "INVALID_BATCH_STATUS",
    "message": "importing: batch is not in a state that allows this action (action=commit, current_status=VALIDATION_FAILED, allowed_statuses=VALIDATED, COMMITTING, COMMITTED)",
    "details": {
      "action": "commit",
      "current_status": "VALIDATION_FAILED",
      "allowed_statuses": [
        "VALIDATED",
        "COMMITTING",
        "COMMITTED"
      ],
      "hint": "poll GET /imports/{id} until status VALIDATED before first commit; retry commit safely if status COMMITTING or COMMITTED"
    },
    "request_id": "req-commit-err-003"
  }
}
```

#### D. Batch tidak ditemukan

**Status:** `404 Not Found`

```json
{
  "error": {
    "code": "NOT_FOUND",
    "message": "importing: data not found",
    "request_id": "req-commit-err-004"
  }
}
```

#### E. User bukan Admin

**Status:** `403 Forbidden`

```json
{
  "error": {
    "code": "FORBIDDEN",
    "message": "importing: forbidden",
    "request_id": "req-commit-err-005"
  }
}
```

## 9. Pola Integrasi Frontend yang Direkomendasikan

### 9.1 Setelah Upload

Frontend sebaiknya:

- simpan `batch_id`,
- pindah ke halaman detail/import preview,
- poll `GET /imports/{id}` tiap 2-5 detik,
- hentikan polling jika status sudah final validasi (`VALIDATED` atau `VALIDATION_FAILED`).

### 9.2 Saat Menentukan Tombol Commit

Gunakan aturan berikut:

| Status Batch | Tombol Commit |
| --- | --- |
| `UPLOADED` | disabled |
| `VALIDATING` | disabled |
| `VALIDATED` | enabled |
| `VALIDATION_FAILED` | disabled |
| `COMMITTING` | disabled, tapi boleh tampilkan badge “sedang diproses” |
| `COMMITTED` | disabled |

### 9.3 Saat Menentukan Tombol Export Rejected Rows

Rekomendasi UI:

- tampilkan setelah batch minimal sudah `VALIDATED` atau `COMMITTED`,
- jika `invalid_rows = 0`, tombol bisa tetap tampil tetapi beri label bahwa file mungkin hanya berisi header CSV.

### 9.4 Saat Mendapat `INVALID_BATCH_STATUS`

Frontend sebaiknya jangan menampilkan pesan generik saja. Gunakan `details.current_status` untuk pesan yang lebih presisi:

- `UPLOADED` / `VALIDATING` ? “File masih divalidasi, tunggu beberapa saat.”
- `VALIDATION_FAILED` ? “File gagal divalidasi, commit tidak bisa dilanjutkan.”
- `COMMITTING` ? “Import sedang diproses, tidak perlu klik commit lagi.”
- `COMMITTED` ? “Import ini sudah selesai.”


### 9.5 Panduan Frontend untuk Potensi Duplikasi Owner

Bagian ini penting karena backend belum memiliki endpoint khusus duplicate candidate detector untuk satu payload owner. Namun frontend sudah bisa membangun duplicate warning yang cukup baik dengan memanfaatkan endpoint owner yang ada sekarang.

Prinsipnya:

- backend saat ini punya hard duplicate untuk `code`,
- backend melakukan normalisasi nomor telepon,
- tetapi untuk kemiripan data lain seperti nama, brand, dan lokasi, keputusan "ini duplikat atau bukan" masih merupakan otoritas bisnis di frontend atau user, bukan hard block backend.

#### 9.5.1 Hard Duplicate vs Soft Duplicate

| Jenis | Contoh | Backend Saat Ini |
| --- | --- | --- |
| Hard duplicate | `code` owner sama persis dengan owner existing | ditolak dengan `409 CODE_ALREADY_USED` |
| Soft duplicate | telepon sama, nama sama, brand sama, atau kombinasi data sangat mirip | tidak otomatis ditolak; frontend disarankan memberi warning |

#### 9.5.2 Daftar Kandidat Data yang Patut Dianggap Mungkin Duplikat

Frontend disarankan menampilkan warning jika menemukan salah satu atau beberapa kondisi berikut terhadap data owner yang sudah ada:

1. `code` owner sama persis.
   - Ini adalah hard duplicate.
   - Contoh cek:

```http
GET /api/v1/owners?code=OWN-000123
```

2. `phone` owner sama setelah normalisasi.
   - Sangat kuat sebagai sinyal duplikasi.
   - Contoh: `0812-3456-7890`, `+62 812 3456 7890`, dan `6281234567890` pada praktik backend akan dianggap nomor yang sama setelah dinormalisasi.
   - Contoh cek:

```http
GET /api/v1/owners?phone=6281234567890
```

3. `name` owner sama atau sangat mirip.
   - Gunakan sebagai warning, bukan block otomatis.
   - Contoh cek:

```http
GET /api/v1/owners?name=Laundry%20Maju%20Jaya
```

4. `brand_name` sama.
   - Berguna jika owner memakai nama usaha yang khas.
   - Contoh cek:

```http
GET /api/v1/owners?brand_name=Maju%20Laundry
```

5. Kombinasi `brand_name + city + province` sama.
   - Cukup kuat untuk memunculkan warning bahwa owner ini mungkin sudah pernah dimasukkan.
   - Contoh cek:

```http
GET /api/v1/owners?brand_name=Maju%20Laundry&city=Surabaya&province=Jawa%20Timur
```

6. Pencarian bebas lintas field untuk screening cepat.
   - Cocok dipakai ketika frontend hanya punya satu string utama dari user.
   - Contoh cek:

```http
GET /api/v1/owners?q=Maju%20Laundry
```

#### 9.5.3 Strategi Frontend yang Direkomendasikan

Saat user mengisi owner baru atau saat frontend sedang menyiapkan commit import, frontend bisa menjalankan preflight duplicate screening seperti ini:

1. Jika ada `code`, cek `GET /owners?code=...`
2. Jika ada `phone`, normalisasi atau bersihkan input lalu cek `GET /owners?phone=...`
3. Jika ada `brand_name`, cek `GET /owners?brand_name=...`
4. Jika ada `name`, cek `GET /owners?name=...`
5. Jika ada banyak hasil mirip, tampilkan daftar kandidat owner existing kepada user

Daftar kandidat yang baik untuk ditampilkan frontend minimal memuat:

- `id`
- `code`
- `name`
- `phone`
- `brand_name`
- `city`
- `province`
- `status`

Semua data itu sudah tersedia dari response list owner.

#### 9.5.4 Otoritas Frontend untuk Tetap Menyimpan Data yang Mirip

Frontend boleh memberi user pilihan:

- Gunakan owner yang sudah ada
- Tetap simpan sebagai owner baru

Tetapi ada aturan penting:

- jika user memilih tetap simpan sebagai owner baru, frontend wajib mengirim `code` owner yang berbeda dan unik,
- backend saat ini tidak menyediakan flag seperti `force_duplicate=true`,
- jadi override duplikasi dilakukan bukan dengan melewati validasi backend, melainkan dengan mengirimkan kode owner baru sambil tetap mempertahankan data lain yang mirip.

Artinya, otoritas frontend di sini adalah:

- frontend boleh memperingatkan,
- frontend boleh meminta konfirmasi user,
- frontend boleh tetap melanjutkan create atau import,
- asalkan kode owner baru tidak bentrok.

#### 9.5.5 Contoh Flow UI yang Disarankan

1. User input owner:
   - code: `OWN-009999`
   - name: `Laundry Maju Jaya`
   - phone: `0812 3456 7890`
   - brand: `Maju Laundry`

2. Frontend menjalankan duplicate screening:
   - `GET /owners?code=OWN-009999`
   - `GET /owners?phone=6281234567890`
   - `GET /owners?brand_name=Maju%20Laundry`

3. Frontend menemukan owner existing:
   - `OWN-000123 / Laundry Maju Jaya / 6281234567890 / Maju Laundry`

4. Frontend menampilkan warning:
   - Data owner ini kemungkinan sudah pernah terdaftar.
   - tampilkan owner candidate yang mirip

5. User memilih salah satu:
   - pakai owner lama, atau
   - tetap buat owner baru

6. Jika user tetap buat owner baru:
   - frontend minta atau generate `code` baru yang unik
   - baru kirim request create atau import

#### 9.5.6 Contoh Error Jika Tetap Mengirim Kode yang Bentrok

Jika frontend tetap mengirim kode owner yang sudah dipakai, backend akan mengembalikan:

**Status:** `409 Conflict`

```json
{
  "error": {
    "code": "CODE_ALREADY_USED",
    "message": "kode sudah digunakan; kemungkinan data ini sudah pernah terdaftar atau duplikat",
    "details": {
      "possible_cause": "kode yang dikirim sudah dipakai oleh data lain atau data yang dimasukkan kemungkinan adalah duplikat dari owner atau outlet yang sudah ada",
      "hint": "jika ini memang data baru yang mirip owner lama, frontend boleh meminta user mengganti kode menjadi kode baru yang unik sebelum mengirim ulang"
    },
    "request_id": "req-owner-duplicate-001"
  }
}
```

Pesan ini sebaiknya diterjemahkan frontend menjadi tindakan yang jelas, misalnya:

- Kode owner sudah dipakai.
- Data ini mungkin sudah pernah terdaftar.
- Jika tetap ingin menyimpan sebagai owner baru, ubah kode owner terlebih dahulu.

## 10. Manual Smoke Test Summary

Pengujian manual yang sudah dilakukan dan lulus:

| Area | Skenario | Hasil |
| --- | --- | --- |
| Upload | Upload file `OWNER_OUTLET` tanpa profile eksplisit | PASS |
| Upload | Upload file `NON_REGISTER` dengan profile eksplisit | PASS |
| Validation | Deteksi profile otomatis dari header asli | PASS |
| Validation | Split valid/invalid rows dari file nyata | PASS |
| Security | Kolom OTP tidak pernah tersimpan ke `raw_payload` | PASS |
| Dedup | Re-upload file identik mengembalikan batch lama | PASS |
| Commit | Commit batch valid | PASS |
| Commit Retry | Retry commit saat `COMMITTING` / `COMMITTED` | PASS secara kontrak backend terbaru |
| Export | Download rejected rows CSV | PASS |
| RBAC | Sales mencoba endpoint import | PASS (ditolak) |

## 11. Kesimpulan

Modul importing Sprint 14 sekarang lebih siap dipakai frontend karena:

- alur upload ? validate ? preview ? commit sudah jelas,
- retry commit tidak lagi mudah memicu false error,
- response `INVALID_BATCH_STATUS` sekarang jauh lebih kaya konteks,
- progress commit konsisten kembali dari `0` sampai `100`,
- dan dokumentasi ini sudah memetakan kapan setiap tombol/action di frontend boleh dipanggil.

Jika frontend tetap menerima `INVALID_BATCH_STATUS`, hampir selalu artinya urutan pemanggilan endpoint belum sesuai state batch saat itu. Karena itu, state polling `GET /imports/{id}` harus dianggap sebagai source of truth utama dalam modul import.


## 12. Addendum 28 Juli 2026 — Histori Import, Viewer File, dan Resume Progress

### 12.1 Ringkasan

Mulai pembaruan 28 Juli 2026, histori import perlu dipahami sebagai data persisten berbasis database, bukan state session browser.

Artinya:

- ketika Admin upload file, batch import langsung tercatat di tabel import batch,
- ketika user keluar halaman lalu masuk lagi, histori batch itu tetap bisa dilihat lewat `GET /imports` dan `GET /imports/{id}`,
- row snapshot tetap statis karena disimpan di `import_rows.raw_payload`,
- file asli yang pernah diupload juga sekarang bisa dibuka lagi dan diunduh lagi.

### 12.2 Endpoint Baru untuk File Asli

| Endpoint | Method | Fungsi |
| --- | --- | --- |
| `/imports/{id}/file` | GET | Mengambil file Excel asli untuk ditampilkan oleh frontend pada halaman viewer |
| `/imports/{id}/file/download` | GET | Mengunduh file Excel asli sebagai attachment |

### 12.3 Data Histori yang Sudah Bisa Ditampilkan Frontend

Dari `GET /imports` dan `GET /imports/{id}`, frontend sudah bisa membangun halaman histori import dengan data berikut:

- siapa admin yang mengupload: `uploaded_by`
- kapan upload dilakukan: `uploaded_at`
- profile import: `profile`
- nama file asli: `original_filename`
- hash file: `file.sha256`
- path buka file: `file.view_path`
- path unduh file: `file.download_path`
- status terakhir batch: `status`
- ringkasan validasi: `total_rows`, `valid_rows`, `invalid_rows`
- ringkasan commit: `committed_rows`, `committed_by`, `committed_at`

Sedangkan detail data yang diimport pada waktu itu tetap bisa dilihat dari:

- `GET /imports/{id}/rows`

Karena `raw_payload` disimpan saat validasi, maka data tersebut bersifat snapshot statis pada saat import berlangsung.

### 12.4 Cara Frontend Membuat Halaman Histori Import

Struktur halaman yang direkomendasikan:

1. Tabel histori batch import
   - sumber data: `GET /imports`
   - kolom minimal:
     - `code`
     - `profile`
     - `original_filename`
     - `uploaded_by.name`
     - `uploaded_at`
     - `status`
     - `valid_rows`
     - `invalid_rows`
     - `committed_rows`

2. Halaman detail histori import
   - sumber data:
     - `GET /imports/{id}`
     - `GET /imports/{id}/rows`
   - bagian minimal:
     - informasi admin pengupload
     - waktu upload / validasi / commit
     - file asli
     - summary row
     - daftar row snapshot statis

3. Halaman viewer Excel
   - frontend mengambil file dari `GET /imports/{id}/file`
   - frontend merender workbook dengan library viewer sendiri
   - backend tidak merender sheet ke HTML; backend hanya menyediakan file sumbernya

4. Tombol download file
   - gunakan `GET /imports/{id}/file/download`

### 12.5 Resume Progress Setelah Keluar Halaman

Jika user keluar halaman ketika upload/validasi/commit masih berjalan:

- frontend tidak perlu mengandalkan state lokal atau session page,
- cukup buka lagi daftar batch melalui `GET /imports`,
- pilih batch yang sama,
- lalu poll `GET /imports/{id}` lagi.

Status batch akan tetap sama sesuai progress terakhir di backend:

- `UPLOADED`
- `VALIDATING`
- `VALIDATED`
- `COMMITTING`
- `COMMITTED`
- `VALIDATION_FAILED`

Ini berarti alur import sekarang sudah mendukung pola:

- upload hari ini,
- tutup halaman,
- buka lagi,
- lanjut lihat hasil validasi,
- lalu commit nanti.

### 12.6 Contoh Metadata File pada Response Batch

Contoh response `GET /imports/{id}` sekarang dapat memuat informasi file seperti ini:

```json
{
  "data": {
    "id": 15,
    "code": "IMPORT-20260728-8e51fa9a1c30",
    "original_filename": "01. Owner & Outlet 2026 (Copy).xlsx",
    "file": {
      "original_filename": "01. Owner & Outlet 2026 (Copy).xlsx",
      "sha256": "8e51fa9a1c30...",
      "view_path": "/api/v1/imports/15/file",
      "download_path": "/api/v1/imports/15/file/download"
    },
    "uploaded_by": {
      "id": 1,
      "name": "Admin User"
    },
    "uploaded_at": "2026-07-28T09:10:00Z",
    "status": "VALIDATED"
  }
}
```

### 12.7 Error Jika File Asli Sudah Tidak Tersedia

Jika file workbook asli di storage sudah tidak ada, backend akan mengembalikan:

**Status:** `404 Not Found`

```json
{
  "error": {
    "code": "FILE_UNAVAILABLE",
    "message": "importing: original file is no longer available",
    "request_id": "req-file-unavailable-001"
  }
}
```

Frontend sebaiknya menampilkan pesan seperti:

- “File Excel asli untuk histori ini sudah tidak tersedia.”
- “Data hasil import masih bisa dilihat dari snapshot baris, tetapi file sumber tidak bisa dibuka lagi.”
