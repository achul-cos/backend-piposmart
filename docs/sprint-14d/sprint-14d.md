Sprint: 14d — Import Progress Tracking (Fix Frontend Polling)
Periode: 28 Juli 2026 (patch/bugfix sprint pada sesi yang sama)
Status: GREEN

Sprint Goal:
- Tambahkan progress tracking pada import batch sehingga frontend dapat menampilkan persentase upload/validasi kepada user tanpa polling berlebihan.

Committed Deliverables:
- Field `progress_percentage` (0–100%) pada `import_batches`.
- Tracking progress saat fase validasi dan commit.
- API response mencakup progress di setiap GET `/imports/{id}`.

Completed:
- Migration `20260728000100_add_import_progress.sql` — tambah kolom `progress_percentage INT UNSIGNED DEFAULT 0`.
- `ImportBatch` struct updated dengan field `ProgressPercentage`.
- `ImportBatchResponse` struct include `progress_percentage` di JSON response.
- Repository method `UpdateProgress(ctx, batchID, percentage)` — update progress dengan bounds checking [0–100].
- `Repository.SetBatchValidating` — reset progress ke 0 saat validasi dimulai.
- `Repository.SetValidationResult` & `SetCommitResult` — set progress ke 100% saat selesai.
- Worker ValidateHandler — update progress setiap 100 rows (atau lebih frequent untuk file kecil).
- Worker CommitHandler — update progress setiap 100 rows saat commit berlangsung.
- Unit tests — seluruh paket `internal/importing` tetap PASS (24 tests).

Not Completed / Carry Over:
- (tidak ada)

Demo Evidence:
- Endpoint: `GET /imports/{id}` mengembalikan `progress_percentage` field.
- Skenario nyata: upload file besar (1000+ rows) → polling `/imports/{id}` setiap 2–5 detik → `progress_percentage` naik dari 0 ke 100.
- Response sample: `{ ..., "status": "VALIDATING", "progress_percentage": 45, ... }`.
- Unit test: `go test ./internal/importing/... -v` — PASS semua, termasuk test yang memverifikasi progress update logic.

Quality:
- Build: `go build ./...` SUCCESS.
- Migration reversible: up/down/up tanpa error.
- No breaking changes di API — field baru hanya addition, eksisting fields tetap sama.
- Frontend-compatible: JSON field `progress_percentage` bersifat int, mudah dipakai untuk progress bar.
- Backwards compatible: clients lama yang ignore field progress tetap berfungsi.

Impediments:
- (tidak ada)

Risiko Baru:
- (tidak ada — ini improvement pure, bukan perubahan fundamental flow)

Keputusan yang Dibutuhkan:
- Frontend perlu diupdate untuk membaca `progress_percentage` dan menampilkan progress bar (out of scope backend sprint ini).

Rencana Sprint Berikutnya:
- Progress tracking sudah ready. Frontend dapat mengimplementasikan progress bar pembaca `progress_percentage` dari response.
- Mitigasi polling berlebihan: frontend bisa mengurangi polling interval dari 1 detik ke 5 detik berkat progress visibility.
- Alternatif future: WebSocket/SSE untuk real-time progress push (tidak urgent, progress polling sudah cukup baik dengan tracking ini).
