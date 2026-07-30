# API Testing Report - Sprint 15

## 1. Informasi Dokumen

| Item | Nilai |
| --- | --- |
| Project | Backend CRM Piposmart |
| Sprint | 15 |
| Tanggal Testing | 30 Juli 2026 |
| Environment | Local Development |
| Base URL API | `http://localhost:8080/api/v1` |
| Port isolasi saat smoke test | `http://127.0.0.1:18083/api/v1` |
| Testing Tool | `curl.exe`, PowerShell, MySQL CLI |
| Fokus Testing | Import profile Sprint 15, histori batch, histori row, commit, file access, dan error handler |

Catatan:

- contoh response di bawah memakai data hasil smoke test nyata;
- nilai `id`, `request_id`, `code`, waktu, dan hash bisa berbeda di environment lain;
- untuk frontend, yang penting adalah shape response dan aturan status flow-nya.

## 2. Tujuan Pengujian

Dokumen ini dibuat agar:

- frontend punya referensi integrasi yang jelas;
- QA bisa melakukan smoke test tanpa menebak-nebak payload;
- CTO bisa membaca hasil validasi API tanpa membuka Postman;
- tim kantor bisa memahami alur status batch import.

## 3. Verifikasi Teknis yang Dijalankan

### 3.1 Command

```powershell
go build ./...
go test ./internal/importing/... ./internal/platform/httpserver/... ./internal/platform/migration/...
go run . migrate up
```

### 3.2 Hasil

| Command | Status | Catatan |
| --- | --- | --- |
| `go build ./...` | PASS | Seluruh package berhasil compile |
| `go test ./internal/importing/...` | PASS | Parser, worker logic, dan import service lulus |
| `go test ./internal/platform/httpserver/...` | PASS | Router dan contract endpoint aman |
| `go test ./internal/platform/migration/...` | PASS | Validasi migration lulus |
| `go run . migrate up` | PASS | Migration sprint 15 terbaru terpasang |

## 4. Header dan Envelope

### 4.1 Header Auth

```http
Authorization: Bearer {access_token}
Accept: application/json
```

### 4.2 Header Upload Multipart

```http
Authorization: Bearer {access_token}
Accept: application/json
Content-Type: multipart/form-data
```

### 4.3 Format Success

```json
{
  "data": {},
  "meta": {
    "request_id": "867130985c965f4641532bca08567150"
  }
}
```

### 4.4 Format Error

```json
{
  "error": {
    "code": "INVALID_FILE_TYPE",
    "message": "importing: only .xlsx files are accepted",
    "request_id": "0d7454490930948dd85d7b438cf969fe"
  }
}
```

### 4.5 Format Error dengan `details`

```json
{
  "error": {
    "code": "SHEET_NAME_REQUIRED",
    "message": "importing: this profile requires an explicit sheet_name (its workbook has multiple similar sheets)",
    "details": {
      "root_cause": "Profile import ini memakai workbook multi-sheet sehingga backend butuh nama sheet yang eksplisit.",
      "solution": "Kirim parameter sheet_name yang sama dengan nama sheet pada file Excel.",
      "frontend_prevent": "Wajibkan user memilih sheet ketika profile adalah SALES_CALL_CHAT atau SALES_TARGET.",
      "technical_error": "importing: this profile requires an explicit sheet_name (its workbook has multiple similar sheets)"
    },
    "request_id": "bc6d361fa073067f67a9934f39224dcc"
  }
}
```

## 5. Ringkasan Test Case

| No | Endpoint | Method | Fokus | Hasil |
| --- | --- | --- | --- | --- |
| 1 | `/imports` | POST | Upload `MONTHLY_ACTIVE` | PASS |
| 2 | `/imports` | POST | Upload `NEW_SUBSCRIBE` | PASS |
| 3 | `/imports` | POST | Upload `BONUS_MITRA` | PASS |
| 4 | `/imports` | POST | Upload `SALES_TARGET` | PASS |
| 5 | `/imports` | POST | Upload `SALES_CALL_CHAT` | PASS |
| 6 | `/imports` | GET | Histori import paginated | PASS |
| 7 | `/imports/all` | GET | Histori import full list | PASS |
| 8 | `/imports/{id}` | GET | Detail batch | PASS |
| 9 | `/imports/{id}/rows` | GET | Row hasil parsing | PASS |
| 10 | `/imports/{id}/rows/all` | GET | Semua row batch | PASS |
| 11 | `/imports/{id}/file` | GET | View file asli | PASS |
| 12 | `/imports/{id}/file/download` | GET | Download file asli | PASS |
| 13 | `/imports/{id}/commit` | POST | Commit batch valid | PASS |
| 14 | `/imports` | POST | Upload file duplikat | PASS / idempoten |
| 15 | Mixed | Mixed | Error handling | PASS |

## 6. Detail Testing API

### 6.1 POST `/imports` — Upload `MONTHLY_ACTIVE`

Request:

```http
POST /api/v1/imports
Authorization: Bearer {access_token}
Accept: application/json
Content-Type: multipart/form-data
```

Form-data:

- `file`: `monthly_active_smoke.xlsx`
- `profile`: `MONTHLY_ACTIVE`

Contoh response:

```json
{
  "data": {
    "id": 7,
    "code": "IMPORT-20260730-88dd707eb222",
    "profile": "MONTHLY_ACTIVE",
    "original_filename": "monthly_active_smoke.xlsx",
    "status": "VALIDATED",
    "total_rows": 1,
    "valid_rows": 1,
    "invalid_rows": 0,
    "committed_rows": 0,
    "progress_percentage": 100,
    "file": {
      "original_filename": "monthly_active_smoke.xlsx",
      "sha256": "88dd707eb222adade9b6303621bb5ea6d1d32d8540327df3a584860d4488facf",
      "view_path": "/api/v1/imports/7/file",
      "download_path": "/api/v1/imports/7/file/download"
    },
    "uploaded_by": {
      "id": 54,
      "name": "Sprint 15 Tester"
    }
  },
  "meta": {
    "request_id": "2ee85ad691e726af98d294ac503b5e18"
  }
}
```

Validasi:

- upload berhasil;
- worker memvalidasi file;
- batch berisi 1 row valid;
- file path untuk view/download tersedia.

### 6.2 GET `/imports/{id}/rows` — Hasil Parsing `MONTHLY_ACTIVE`

Request:

```http
GET /api/v1/imports/7/rows?limit=10
Authorization: Bearer {access_token}
Accept: application/json
```

Contoh response:

```json
{
  "data": {
    "items": [
      {
        "id": 610,
        "batch_id": 7,
        "row_index": 3,
        "status": "VALID",
        "raw_payload": {
          "owner_code": "OWN-00921",
          "owner_name": "Owner Laundry 921",
          "project_name": "Laundry Cerah 921",
          "outlet_name": "Laundry Cerah 921 Outlet 01",
          "city": "Medan",
          "region": "SUMUT",
          "activities": [
            {
              "category": "NEW",
              "raw_code": "N-BC",
              "period_year": 2023,
              "period_month": 12,
              "package_code": "BC"
            },
            {
              "category": "SUBSCRIBE",
              "raw_code": "S-BC",
              "period_year": 2024,
              "period_month": 1,
              "package_code": "BC"
            },
            {
              "category": "FOLLOWING",
              "raw_code": "F-BC(6)",
              "period_year": 2024,
              "period_month": 2,
              "package_code": "BC",
              "tenor_months": 6
            }
          ]
        }
      }
    ],
    "meta": {
      "page": 1,
      "limit": 10,
      "total": 1
    }
  },
  "meta": {
    "request_id": "cb251e9aed6fefbd07731fa4a7c959fe"
  }
}
```

Validasi:

- payload hasil parsing terlihat jelas;
- frontend bisa menampilkan preview sebelum commit;
- row status `VALID` bisa dipakai untuk indikator preview.

### 6.3 POST `/imports/{id}/commit` — Commit `MONTHLY_ACTIVE`

Request:

```http
POST /api/v1/imports/7/commit
Authorization: Bearer {access_token}
Accept: application/json
```

Respons awal:

```json
{
  "data": {
    "id": 7,
    "status": "COMMITTING",
    "progress_percentage": 0
  },
  "meta": {
    "request_id": "ef1fab2181e2d8c0073fcdc0da704afd"
  }
}
```

Poll final:

```http
GET /api/v1/imports/7
```

Contoh response final:

```json
{
  "data": {
    "id": 7,
    "status": "COMMITTED",
    "committed_rows": 1,
    "progress_percentage": 100,
    "committed_at": "2026-07-30T10:50:25Z",
    "committed_by": {
      "id": 54,
      "name": "Sprint 15 Tester"
    }
  },
  "meta": {
    "request_id": "2bd2f53023ac947d5b861b59992e9f8e"
  }
}
```

Validasi:

- commit berjalan async;
- frontend wajib poll setelah commit;
- `committed_rows` dan `committed_at` menjadi indikator selesai.

### 6.4 POST `/imports` — Upload `NEW_SUBSCRIBE`

Form-data:

- `file`: `new_subscribe_smoke.xlsx`
- `profile`: `NEW_SUBSCRIBE`

Contoh response ringkas:

```json
{
  "data": {
    "id": 13,
    "code": "IMPORT-20260730-0f00030966a4",
    "profile": "NEW_SUBSCRIBE",
    "status": "UPLOADED",
    "original_filename": "new_subscribe_smoke.xlsx"
  }
}
```

Contoh row preview:

```json
{
  "id": 716,
  "row_index": 2,
  "status": "VALID",
  "raw_payload": {
    "kode": "NS-SMOKE-001",
    "owner_code": "OWN-00921",
    "owner_name": "Owner Laundry 921",
    "project_name": "Laundry Cerah 921",
    "outlet_name": "Laundry Cerah 921 Outlet 01",
    "paket_membership": "Basic",
    "tenor_months": 1,
    "nominal_aktivasi": "99000",
    "tanggal_aktivasi": "2026-07-30",
    "expired_date": "2026-08-29",
    "status": "PAID",
    "metode_pembayaran": "MANUAL_TRANSFER"
  }
}
```

Contoh final setelah commit:

```json
{
  "data": {
    "id": 13,
    "status": "COMMITTED",
    "committed_rows": 1,
    "progress_percentage": 100
  }
}
```

Validasi:

- transaksi berhasil di-commit;
- verifikasi DB menunjukkan 1 `subscription_order` terbentuk.

### 6.5 POST `/imports` — Upload `SALES_TARGET`

Request khusus:

Profile ini wajib:

- `profile=SALES_TARGET`
- `sheet_name`
- `target_sales_user_id`

Form-data:

- `file`: `sales_target_smoke.xlsx`
- `profile`: `SALES_TARGET`
- `sheet_name`: `TARGET`
- `target_sales_user_id`: `7`

Contoh response ringkas:

```json
{
  "data": {
    "id": 8,
    "profile": "SALES_TARGET",
    "sheet_name": "TARGET",
    "target_sales_user_id": 7,
    "status": "UPLOADED"
  }
}
```

Contoh row preview:

```json
{
  "id": 711,
  "row_index": 3,
  "status": "VALID",
  "raw_payload": {
    "period_year": 2026,
    "period_month": 7,
    "target_user": "5",
    "target_omset": "495000"
  }
}
```

Contoh final:

```json
{
  "data": {
    "id": 8,
    "status": "COMMITTED",
    "committed_rows": 1
  }
}
```

Validasi:

- target count dan target omset berhasil masuk;
- verifikasi DB menunjukkan masing-masing 1 record target untuk sales ID `7`.

### 6.6 POST `/imports` — Upload `SALES_CALL_CHAT`

Form-data:

- `file`: `sales_call_chat_smoke.xlsx`
- `profile`: `SALES_CALL_CHAT`
- `sheet_name`: `Call & Chat-Lidya`
- `target_sales_user_id`: `7`

Contoh row preview:

```json
{
  "id": 712,
  "row_index": 3,
  "status": "VALID",
  "raw_payload": {
    "owner_code": "OWN-00921",
    "owner_name": "Owner Laundry 921",
    "owner_phone": "6281300200921",
    "scor": 1,
    "validitas": "Masih pertimbangan",
    "remaks": "Follow up minggu depan",
    "tanggal_fu": "2026-07-29",
    "call_status": "TERHUBUNG",
    "finalisasi_paket": "Not Yet",
    "is_closing": false
  }
}
```

Contoh final:

```json
{
  "data": {
    "id": 9,
    "status": "COMMITTED",
    "committed_rows": 1
  }
}
```

Validasi:

- interaction historis berhasil masuk;
- verifikasi DB menunjukkan 1 row `customer_interactions` tercatat untuk lead terkait.

### 6.7 POST `/imports` — Upload `BONUS_MITRA`

Form-data:

- `file`: `bonus_mitra_smoke_v3.xlsx`
- `profile`: `BONUS_MITRA`

Contoh row preview:

```json
{
  "id": 715,
  "row_index": 2,
  "status": "VALID",
  "raw_payload": {
    "partner_name": "Mitra Demo Rev 3",
    "partner_owner_code": "MTR-001",
    "partner_owner_name": "Mitra Demo",
    "partner_category": "Referral",
    "sales_pic_name": "Sales Demo 093",
    "referred_owner_code": "OWN-00921",
    "referred_owner_name": "Owner Laundry 921",
    "referred_project_name": "Laundry Cerah 921",
    "referred_outlet_name": "Laundry Cerah 921 Outlet 01",
    "package_name": "Basic 1 Bulan",
    "unpaid_amount": "9900",
    "total_amount": "9900",
    "payout_status": "Belum Dicairkan"
  }
}
```

Contoh final:

```json
{
  "data": {
    "id": 12,
    "status": "COMMITTED",
    "committed_rows": 1
  }
}
```

Contoh row setelah commit:

```json
{
  "id": 715,
  "status": "COMMITTED",
  "owner_id": 14,
  "outlet_id": 16,
  "lead_id": 14
}
```

Validasi:

- snapshot referral berhasil terbentuk;
- owner, outlet, dan lead berhasil dipetakan.

### 6.8 GET `/imports`

Request:

```http
GET /api/v1/imports?limit=5&sort=-created_at
Authorization: Bearer {access_token}
Accept: application/json
```

Contoh response ringkas:

```json
{
  "data": {
    "items": [
      {
        "id": 16,
        "profile": "SALES_TARGET",
        "sheet_name": "SHEET-SALAH",
        "status": "VALIDATION_FAILED",
        "error_message": "importing: declared sheet_name was not found in the workbook"
      },
      {
        "id": 13,
        "profile": "NEW_SUBSCRIBE",
        "status": "COMMITTED",
        "committed_rows": 1
      }
    ],
    "meta": {
      "page": 1,
      "limit": 5,
      "total": 9
    }
  }
}
```

Validasi:

- histori batch bisa ditampilkan di tabel frontend;
- status gagal dan berhasil bisa tampil bersamaan;
- `error_message` tersedia untuk batch gagal validasi.

### 6.9 GET `/imports/all`

Request:

```http
GET /api/v1/imports/all?sort=-created_at&profile=NEW_SUBSCRIBE
Authorization: Bearer {access_token}
Accept: application/json
```

Contoh response ringkas:

```json
{
  "data": {
    "items": [
      {
        "id": 13,
        "profile": "NEW_SUBSCRIBE",
        "status": "COMMITTED",
        "committed_rows": 1
      }
    ],
    "meta": {
      "page": 1,
      "limit": 1,
      "total": 1
    }
  }
}
```

Validasi:

- frontend bisa mengambil seluruh histori import tanpa limit pagination default.

### 6.10 GET `/imports/{id}/file`

Request:

```http
GET /api/v1/imports/13/file
Authorization: Bearer {access_token}
```

Hasil verifikasi:

| Item | Nilai |
| --- | --- |
| HTTP Status | `200 OK` |
| Content-Type | `application/vnd.openxmlformats-officedocument.spreadsheetml.sheet` |
| Content-Disposition | `inline; filename="new_subscribe_smoke.xlsx"` |

Validasi:

- frontend bisa buka file asli dalam viewer/tab baru.

### 6.11 GET `/imports/{id}/file/download`

Request:

```http
GET /api/v1/imports/13/file/download
Authorization: Bearer {access_token}
```

Hasil verifikasi:

| Item | Nilai |
| --- | --- |
| HTTP Status | `200 OK` |
| Content-Type | `application/vnd.openxmlformats-officedocument.spreadsheetml.sheet` |
| Content-Disposition | `attachment; filename="new_subscribe_smoke.xlsx"` |

Validasi:

- frontend bisa memicu download file asli.

### 6.12 Upload File Duplikat Bersifat Idempoten

Request:

```http
POST /api/v1/imports
Authorization: Bearer {access_token}
Accept: application/json
Content-Type: multipart/form-data
```

Form-data:

- `file`: `new_subscribe_smoke.xlsx`
- `profile`: `NEW_SUBSCRIBE`

Contoh response saat file yang sama di-upload ulang:

```json
{
  "data": {
    "id": 13,
    "code": "IMPORT-20260730-0f00030966a4",
    "profile": "NEW_SUBSCRIBE",
    "status": "COMMITTED",
    "committed_rows": 1
  }
}
```

Catatan:

- backend mengembalikan batch existing;
- ini mencegah duplikasi batch import.

## 7. Error Handler, Contoh Error, dan Solusinya

### 7.1 Token Tidak Dikirim

Request:

```http
POST /api/v1/imports
Accept: application/json
```

Response aktual:

```json
{
  "error": {
    "code": "UNAUTHENTICATED",
    "message": "Token akses wajib dikirim",
    "request_id": "175b14a4e3454b881f2edcd14423ddcc"
  }
}
```

HTTP status:

```text
401 Unauthorized
```

Solusi:

- selalu kirim `Authorization: Bearer {access_token}`.

Pencegahan frontend:

- pakai auth interceptor global;
- redirect ke login jika token kosong.

### 7.2 File Bukan `.xlsx`

Request:

```http
POST /api/v1/imports
Authorization: Bearer {access_token}
Accept: application/json
Content-Type: multipart/form-data
```

Form-data:

- `file`: `README.md`
- `profile`: `MONTHLY_ACTIVE`

Response aktual:

```json
{
  "error": {
    "code": "INVALID_FILE_TYPE",
    "message": "importing: only .xlsx files are accepted",
    "details": {
      "root_cause": "File yang diunggah bukan format Excel .xlsx.",
      "solution": "Unggah file .xlsx yang valid.",
      "frontend_prevent": "Validasi ekstensi file di frontend sebelum upload.",
      "technical_error": "importing: only .xlsx files are accepted"
    },
    "request_id": "0d7454490930948dd85d7b438cf969fe"
  }
}
```

HTTP status:

```text
400 Bad Request
```

### 7.3 `sheet_name` Dikirim Tanpa `profile`

Request:

```http
POST /api/v1/imports
Authorization: Bearer {access_token}
Accept: application/json
Content-Type: multipart/form-data
```

Form-data:

- `file`: `sales_target_smoke.xlsx`
- `sheet_name`: `TARGET`

Response aktual:

```json
{
  "error": {
    "code": "SHEET_NAME_NEEDS_PROFILE",
    "message": "importing: sheet_name requires an explicit profile to verify it against",
    "details": {
      "root_cause": "Frontend mengirim sheet_name tanpa profile eksplisit.",
      "solution": "Kirim profile yang sesuai bersama sheet_name.",
      "frontend_prevent": "Jangan tampilkan input sheet_name bila profile belum dipilih.",
      "technical_error": "importing: sheet_name requires an explicit profile to verify it against"
    },
    "request_id": "33e7829a56f4a5cc4e4a49fde74a9ceb"
  }
}
```

HTTP status:

```text
400 Bad Request
```

### 7.4 `SALES_TARGET` Tanpa `sheet_name`

Request:

```http
POST /api/v1/imports
Authorization: Bearer {access_token}
Accept: application/json
Content-Type: multipart/form-data
```

Form-data:

- `file`: `sales_target_smoke.xlsx`
- `profile`: `SALES_TARGET`
- `target_sales_user_id`: `7`

Response aktual:

```json
{
  "error": {
    "code": "SHEET_NAME_REQUIRED",
    "message": "importing: this profile requires an explicit sheet_name (its workbook has multiple similar sheets)",
    "details": {
      "root_cause": "Profile import ini memakai workbook multi-sheet sehingga backend butuh nama sheet yang eksplisit.",
      "solution": "Kirim parameter sheet_name yang sama dengan nama sheet pada file Excel.",
      "frontend_prevent": "Wajibkan user memilih sheet ketika profile adalah SALES_CALL_CHAT atau SALES_TARGET.",
      "technical_error": "importing: this profile requires an explicit sheet_name (its workbook has multiple similar sheets)"
    },
    "request_id": "bc6d361fa073067f67a9934f39224dcc"
  }
}
```

HTTP status:

```text
400 Bad Request
```

### 7.5 `SALES_TARGET` / `SALES_CALL_CHAT` Tanpa `target_sales_user_id`

Request:

```http
POST /api/v1/imports
Authorization: Bearer {access_token}
Accept: application/json
Content-Type: multipart/form-data
```

Form-data:

- `file`: `sales_target_smoke.xlsx`
- `profile`: `SALES_TARGET`
- `sheet_name`: `TARGET`

Response aktual:

```json
{
  "error": {
    "code": "TARGET_SALES_USER_REQUIRED",
    "message": "importing: this profile requires an explicit target_sales_user_id (the sales rep is only encoded in the sheet name)",
    "details": {
      "root_cause": "Profile import ini membutuhkan target_sales_user_id karena sales hanya diketahui dari konteks sheet.",
      "solution": "Kirim target_sales_user_id dari akun Sales yang sesuai.",
      "frontend_prevent": "Wajibkan pemilihan Sales saat profile adalah SALES_CALL_CHAT atau SALES_TARGET.",
      "technical_error": "importing: this profile requires an explicit target_sales_user_id (the sales rep is only encoded in the sheet name)"
    },
    "request_id": "2ec11a318968075e15cb07cc29ad61fc"
  }
}
```

HTTP status:

```text
400 Bad Request
```

### 7.6 `sheet_name` Salah / Tidak Ada di Workbook

Request:

```http
POST /api/v1/imports
Authorization: Bearer {access_token}
Accept: application/json
Content-Type: multipart/form-data
```

Form-data:

- `file`: `sales_target_smoke.xlsx`
- `profile`: `SALES_TARGET`
- `sheet_name`: `SHEET-SALAH`
- `target_sales_user_id`: `7`

Response upload awal:

```json
{
  "data": {
    "id": 16,
    "code": "IMPORT-20260730-97d07c4e-fc5d5f",
    "profile": "SALES_TARGET",
    "sheet_name": "SHEET-SALAH",
    "status": "UPLOADED"
  }
}
```

Setelah dipoll:

```http
GET /api/v1/imports/16
```

Response aktual:

```json
{
  "data": {
    "id": 16,
    "status": "VALIDATION_FAILED",
    "error_message": "importing: declared sheet_name was not found in the workbook",
    "total_rows": 0,
    "valid_rows": 0,
    "invalid_rows": 0
  },
  "meta": {
    "request_id": "867130985c965f4641532bca08567150"
  }
}
```

Solusi:

- kirim `sheet_name` yang benar-benar ada di workbook.

Pencegahan frontend:

- tampilkan sheet list hasil pembacaan file;
- jangan biarkan user mengetik bebas bila bisa dipilih dari dropdown.

### 7.7 Commit Saat Status Belum `VALIDATED`

Request:

```http
POST /api/v1/imports/16/commit
Authorization: Bearer {access_token}
Accept: application/json
```

Kondisi pemicu:

- batch `16` berada pada `VALIDATION_FAILED`.

Response aktual:

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
      "root_cause": "Frontend memanggil aksi import pada status batch yang belum sesuai alur backend.",
      "solution": "Poll GET /imports/{id} dan hanya aktifkan aksi yang sesuai dengan status batch saat ini.",
      "frontend_prevent": "Gunakan status batch sebagai source of truth utama. Tombol commit hanya aktif saat status VALIDATED.",
      "hint": "poll GET /imports/{id} until status VALIDATED before first commit; retry commit safely if status COMMITTING or COMMITTED",
      "technical_error": "importing: batch is not in a state that allows this action (action=commit, current_status=VALIDATION_FAILED, allowed_statuses=VALIDATED, COMMITTING, COMMITTED)"
    },
    "request_id": "65ef1ae2f1ce8d838b97f4cc101c9339"
  }
}
```

HTTP status:

```text
400 Bad Request
```

### 7.8 Catatan Tentang Duplikat Upload

Perilaku sekarang:

- bila dedup key sama, backend mengembalikan batch lama;
- ini **bukan error**, melainkan perilaku idempoten.

Implikasi frontend:

- aman untuk retry upload;
- aman untuk reload halaman histori import;
- tetapi frontend sebaiknya menjelaskan ke user bahwa file sama tidak membuat batch baru.

## 8. Verifikasi Data Setelah Commit

Pengecekan ke database setelah smoke test menunjukkan:

| Profil | Verifikasi | Hasil |
| --- | --- | --- |
| `MONTHLY_ACTIVE` | `outlet_monthly_activity_snapshot` by `import_batch_id=7` | `3 row` |
| `BONUS_MITRA` | `partner_bonus_referral_snapshots` by `import_batch_id=12` | `1 row` |
| `NEW_SUBSCRIBE` | `subscription_orders` by external reference smoke | `1 row` |
| `SALES_TARGET` | `sales_targets` untuk count metric | `1 row` |
| `SALES_TARGET` | `sales_targets` untuk amount metric | `1 row` |
| `SALES_CALL_CHAT` | `customer_interactions` untuk note smoke | `1 row` |

## 9. Checklist Integrasi Frontend

- selalu kirim Bearer token;
- batasi uploader hanya untuk role yang berhak;
- profile `SALES_TARGET` dan `SALES_CALL_CHAT` wajib:
  - pilih `sheet_name`
  - pilih `target_sales_user_id`
- tampilkan status batch sebagai state utama UI;
- tombol commit hanya aktif saat `status=VALIDATED`;
- tampilkan `error_message` pada batch `VALIDATION_FAILED`;
- simpan `batch_id` setelah upload untuk polling lanjutan;
- dukung preview row sebelum commit;
- tampilkan `request_id` ketika error agar mudah tracing;
- validasi ekstensi file `.xlsx` sebelum upload;
- beri pesan ke user bila upload ulang file yang sama mengembalikan batch lama.

## 10. Kesimpulan

Sprint 15 backend untuk modul importing sudah tervalidasi secara manual dengan data nyata dan fixture kecil. Dokumentasi ini mencakup:

- contoh request sukses;
- contoh response sukses;
- contoh request yang memicu error;
- response error aktual;
- solusi dan pencegahan di frontend;
- verifikasi side-effect ke database;
- hasil audit runtime yang ditemukan selama testing.
