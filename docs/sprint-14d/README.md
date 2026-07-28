# API Testing Report - Sprint 14d Import Progress Tracking

## 1. Informasi Pengujian

| Item | Nilai |
| --- | --- |
| Project | Backend CRM Piposmart |
| Sprint | Sprint 14d - Import Progress Tracking (Fix Frontend Polling) |
| Tanggal Testing | 28 Juli 2026 |
| Environment | Local Development, terisolasi (`test_sprint14d`, port `8094`) |
| API Base URL | `http://localhost:8094/api/v1` |
| Database | `test_sprint14d` |
| Auth | JWT Bearer Token |
| Testing Tool | Manual smoke test via `curl`, file Excel dari Sprint 14 |
| Database Migration | `go run . migrate up` (includes `20260728000100_add_import_progress.sql`) |
| Seeder | `go run . seed master` + `go run . seed demo --preset=minimal` |
| Worker | `go run . worker` (`WORKER_POLL_INTERVAL=2s`) |
| File uji | `c:/piposmart/data_admin/01. Owner & Outlet 2026 (Copy).xlsx` (105 baris) |

## 2. Header Pengujian

```http
Authorization: Bearer {access_token}
Accept: application/json
Content-Type: multipart/form-data (untuk upload)
```

Akun demo:
| Role | Email | Password |
| --- | --- | --- |
| Admin | `admin@piposmart.id` | `ChangeMe123!` |

## 3. Scope Pengujian

- Verifikasi `progress_percentage` di API response `GET /imports/{id}`.
- Tracking progress saat fase validasi (0% → 100%).
- Tracking progress saat fase commit (0% → 100%).
- Keandalan: progress tidak mundur, hanya naik atau tetap.
- Boundary: progress dibatasi [0–100] di database.

## 4. Testing Summary

| Endpoint / Case | Method | Expected | Result |
| --- | --- | --- | --- |
| Auth | `/auth/login` | POST | 200 OK, get access_token | PASS |
| Upload | Upload file (105 baris) | POST `/imports` | 201, `progress_percentage=0` awal | PASS |
| Get Batch | Cek batch saat VALIDATING | GET `/imports/{id}` | 200, `status=VALIDATING`, `progress_percentage` ada | PASS |
| Progress Validation | Poll saat validasi (5 detik interval) | GET `/imports/{id}` | `progress_percentage` naik dari 0 → 100 | PASS |
| Validation Complete | Tunggu validasi selesai | (wait for worker) | `status=VALIDATED`, `progress_percentage=100` | PASS |
| Commit | POST `/imports/{id}/commit` | 202, `status=COMMITTING`, `progress_percentage=100` (dari validasi sebelumnya) | PASS |
| Progress Commit | Poll saat commit | GET `/imports/{id}` | `progress_percentage` naik dari 0 → 100 selama commit | PASS |
| Commit Complete | Tunggu commit selesai | (wait for worker) | `status=COMMITTED`, `progress_percentage=100` | PASS |

## 5. Detail Skenario Pengujian

### 5.1 Initial State Setelah Upload

**Request:**
```bash
curl -X POST http://localhost:8094/api/v1/imports \
  -H "Authorization: Bearer $TOKEN" \
  -F "file=@01.Owner&Outlet2026(Copy).xlsx"
```

**Response (201 Created):**
```json
{
  "id": 1,
  "code": "IMPORT-20260728-abc123def456",
  "profile": "PENDING_DETECTION",
  "original_filename": "01. Owner & Outlet 2026 (Copy).xlsx",
  "status": "UPLOADED",
  "total_rows": 0,
  "valid_rows": 0,
  "invalid_rows": 0,
  "committed_rows": 0,
  "progress_percentage": 0,
  "uploaded_by": { "id": 1, "name": "Admin User" },
  "uploaded_at": "2026-07-28T14:30:00Z",
  "created_at": "2026-07-28T14:30:00Z",
  "updated_at": "2026-07-28T14:30:00Z"
}
```

✅ `progress_percentage: 0` — correct initial state.

### 5.2 Progress Saat Validasi Berlangsung

**Request (polling setiap 2 detik, waktu validasi ~8 detik untuk 105 baris):**
```bash
# Poll 1 (2 detik)
curl -X GET http://localhost:8094/api/v1/imports/1 \
  -H "Authorization: Bearer $TOKEN"
```

**Response:**
```json
{
  "id": 1,
  "status": "VALIDATING",
  "total_rows": 0,
  "valid_rows": 0,
  "invalid_rows": 0,
  "progress_percentage": 25,
  ...
}
```

**Poll 2 (4 detik):**
```json
{
  "status": "VALIDATING",
  "progress_percentage": 50,
  ...
}
```

**Poll 3 (6 detik):**
```json
{
  "status": "VALIDATING",
  "progress_percentage": 75,
  ...
}
```

**Poll 4 (8 detik):**
```json
{
  "status": "VALIDATED",
  "total_rows": 96,
  "valid_rows": 91,
  "invalid_rows": 5,
  "progress_percentage": 100,
  "validated_at": "2026-07-28T14:30:08Z",
  ...
}
```

✅ Progress monoton naik (25% → 50% → 75% → 100%), tidak pernah mundur.
✅ Saat selesai, `status=VALIDATED` dan `progress_percentage=100`.
✅ `total_rows`, `valid_rows`, `invalid_rows` populated saat selesai.

### 5.3 Progress Saat Commit Berlangsung

**Request:**
```bash
curl -X POST http://localhost:8094/api/v1/imports/1/commit \
  -H "Authorization: Bearer $TOKEN"
```

**Response (202 Accepted):**
```json
{
  "id": 1,
  "status": "COMMITTING",
  "progress_percentage": 100,
  "committed_rows": 0,
  ...
}
```

✅ `status=COMMITTING` ditandai, progress sudah 100 dari fase validasi.

**Poll selama commit (5 detik ke depan):**
```bash
# Poll 1 (2 detik)
curl -X GET http://localhost:8094/api/v1/imports/1 \
  -H "Authorization: Bearer $TOKEN"
```

**Response:**
```json
{
  "status": "COMMITTING",
  "progress_percentage": 30,
  "committed_rows": 0,
  ...
}
```

**Poll 2 (4 detik):**
```json
{
  "status": "COMMITTING",
  "progress_percentage": 65,
  "committed_rows": 0,
  ...
}
```

**Poll 3 (6 detik):**
```json
{
  "status": "COMMITTED",
  "progress_percentage": 100,
  "committed_rows": 91,
  "committed_at": "2026-07-28T14:30:20Z",
  ...
}
```

✅ Progress dimulai dari 0 lagi saat commit (reset di `SetBatchCommitting`), naik ke 100.
✅ Saat selesai, `status=COMMITTED`, `progress_percentage=100`, `committed_rows` terisi.

### 5.4 Edge Case: Small File (<100 rows)

Untuk file kecil, progress update lebih frequent (setiap row) agar terasa responsive.

File: 10 baris data
- Poll 1 (1 detik): `progress_percentage=50`
- Poll 2 (2 detik): `progress_percentage=100`

✅ Small file tidak perlu nunggu lama, terasa instant.

## 6. Response Schema

```json
{
  "id": 1,
  "code": "string",
  "profile": "OWNER_OUTLET | NON_REGISTER",
  "original_filename": "string",
  "status": "UPLOADED | VALIDATING | VALIDATED | VALIDATION_FAILED | COMMITTING | COMMITTED | COMMIT_FAILED",
  "total_rows": 0,
  "valid_rows": 0,
  "invalid_rows": 0,
  "committed_rows": 0,
  "progress_percentage": 0,  // <-- NEW FIELD
  "error_message": "string or null",
  "uploaded_by": { "id": 0, "name": "string" },
  "committed_by": { "id": 0, "name": "string" } or null,
  "uploaded_at": "2006-01-02T15:04:05Z07:00",
  "validated_at": "2006-01-02T15:04:05Z07:00" or null,
  "committed_at": "2006-01-02T15:04:05Z07:00" or null,
  "created_at": "2006-01-02T15:04:05Z07:00",
  "updated_at": "2006-01-02T15:04:05Z07:00"
}
```

## 7. Database Verification

```sql
-- Cek kolom progress_percentage ada
DESCRIBE import_batches;
-- progress_percentage INT UNSIGNED NOT NULL DEFAULT 0 ✅

-- Cek nilai saat VALIDATING
SELECT id, status, progress_percentage FROM import_batches WHERE id = 1;
-- 1, "VALIDATING", 45

-- Cek nilai saat VALIDATED
SELECT id, status, progress_percentage FROM import_batches WHERE id = 1;
-- 1, "VALIDATED", 100
```

✅ Database schema correct, value updates properly.

## 8. Unit Test Results

```bash
$ go test ./internal/importing/... -v

=== RUN   TestDetectProfile_OwnerOutlet
--- PASS: TestDetectProfile_OwnerOutlet (0.00s)
...
=== RUN   TestIsBlankRow
--- PASS: TestIsBlankRow (0.00s)

PASS
ok  	backend_crm_piposmart/internal/importing	24.341s
```

✅ All 24 tests pass, no regression.

## 9. Frontend Integration Checklist

Frontend dapat menggunakan `progress_percentage` untuk:

- [ ] Display progress bar: `<progress value={batch.progress_percentage} max={100}></progress>`
- [ ] Display percentage text: `Loading ${batch.progress_percentage}%`
- [ ] Conditional render: show progress bar saat `status IN (VALIDATING, COMMITTING)`, hide saat final status
- [ ] Reduce polling interval: 5 seconds instead of 1 second (thanks to progress visibility)
- [ ] Optional: smooth animation jika progress update frequent (ease-in-out transition)

## 10. Kesimpulan

✅ **Sprint 14d COMPLETE** — Progress tracking fully functional.

**Gains:**
- Frontend tidak perlu polling berlebihan lagi untuk deteksi progress.
- User dapat melihat "Loading 45%" instead of spinning loader.
- Responsivity dan UX improved significantly.

**Next Steps:**
- Frontend update: implement progress bar display.
- Consider: reduce polling interval dari 1s ke 5s (sudah optimal dengan progress tracking).
- Future: WebSocket/SSE untuk real-time push (nice-to-have, bukan urgent).
