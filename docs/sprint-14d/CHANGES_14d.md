# Code Changes Summary - Sprint 14d Import Progress Tracking

## Files Modified

### 1. Database Migration
**File:** `migrations/20260728000100_add_import_progress.sql`
- **Change:** Add `progress_percentage INT UNSIGNED NOT NULL DEFAULT 0` column to `import_batches` table
- **Purpose:** Store progress (0–100%) for frontend display
- **Reversible:** Yes (has `DROP COLUMN` in rollback)

### 2. Type Definitions
**File:** `internal/importing/types.go`

#### ImportBatch struct
- **Added:** `ProgressPercentage int` field
- **Position:** After `CommittedRows`, before `ValidateJobID`

#### ImportBatchResponse struct
- **Added:** `ProgressPercentage int` field with JSON tag `json:"progress_percentage"`
- **Position:** After `CommittedRows`, before `ErrorMessage`

#### NewImportBatchResponse function
- **Updated:** Map `b.ProgressPercentage` to `resp.ProgressPercentage`
- **Purpose:** Ensure progress included in API response

### 3. Repository Layer
**File:** `internal/importing/repository.go`

#### batchSelectColumns constant
- **Added:** `ib.progress_percentage` to SELECT columns

#### scanBatch function
- **Updated:** Scan `&b.ProgressPercentage` from database query
- **Position:** After `&b.CommittedRows`

#### New method: UpdateProgress
```go
func (r *Repository) UpdateProgress(ctx context.Context, batchID int64, percentage int) error
```
- **Purpose:** Update progress_percentage with bounds checking [0–100]
- **Behavior:** Clamps percentage to valid range, never exceeds 100 or below 0

#### SetBatchValidating method
- **Updated:** Reset `progress_percentage = 0` when marking batch as VALIDATING
- **Query:** `UPDATE import_batches SET status = ?, progress_percentage = 0 WHERE id = ?`

#### SetValidationResult method
- **Updated:** Set `progress_percentage = 100` when validation completes
- **Query:** Added `progress_percentage = 100` to UPDATE clause

#### SetCommitResult method
- **Updated:** Set `progress_percentage = 100` when commit completes
- **Query:** Added `progress_percentage = 100` to UPDATE clause

### 4. Worker Implementation
**File:** `internal/importing/worker.go`

#### ValidateHandler function
- **Change:** Pre-collect non-blank data rows to know total count upfront
- **Logic:**
  ```
  1. Scan all rows, collect non-blank ones
  2. For each data row:
     a. Parse & validate
     b. Insert row
     c. Update progress every 100 rows (or more frequently for small files)
  3. Call SetValidationResult (sets progress to 100%)
  ```
- **Progress Calculation:** `(total_processed / total_data) * 100`
- **Update Frequency:** Every 100 rows, or every row if total <= 100

#### CommitHandler function
- **Change:** Track progress during entity creation (Owner/Outlet/Lead)
- **Logic:**
  ```
  1. Get valid rows from database
  2. For each row:
     a. Create/reuse Owner/Outlet/Lead
     b. Mark row as committed
     c. Update progress every 100 rows (or more frequently for small files)
  3. Call SetCommitResult (sets progress to 100%)
  ```
- **Progress Calculation:** `(rows_processed / total_rows) * 100`
- **Update Frequency:** Every 100 rows, or every row if total <= 100

## Code Flow: Progress Tracking Lifecycle

### Upload Phase
```
POST /imports
├─ Upload file, compute SHA-256
├─ CreateBatch → status = UPLOADED, progress_percentage = 0
├─ Enqueue IMPORT_VALIDATE job
└─ Return 201 response (progress_percentage = 0)
```

### Validation Phase
```
Worker processes IMPORT_VALIDATE job
├─ SetBatchValidating → status = VALIDATING, progress_percentage = 0 (reset)
├─ Read all rows from Excel
├─ Collect non-blank rows (know total upfront)
├─ Process rows in loop:
│  ├─ Parse & validate each row
│  ├─ UpdateProgress() every 100 rows
│  │  └─ progress_percentage = (processed / total) * 100
│  └─ Repeat until all rows done
└─ SetValidationResult → status = VALIDATED, progress_percentage = 100
```

### Commit Phase
```
POST /imports/{id}/commit
├─ TriggerCommit → Enqueue IMPORT_COMMIT job, status = COMMITTING
└─ Return 202 response

Worker processes IMPORT_COMMIT job
├─ Get valid rows from database (know total)
├─ Process each valid row in loop:
│  ├─ Create/reuse Owner/Outlet/Lead via service
│  ├─ Mark row as committed
│  ├─ UpdateProgress() every 100 rows
│  │  └─ progress_percentage = (processed / total) * 100
│  └─ Repeat until all rows done
└─ SetCommitResult → status = COMMITTED, progress_percentage = 100
```

## Frontend Integration Guide

### Simple Progress Bar
```javascript
const response = await fetch(`/api/v1/imports/${batchId}`, {
  headers: { Authorization: `Bearer ${token}` }
});
const batch = await response.json();

// Display progress
console.log(`Progress: ${batch.progress_percentage}%`);

// React example:
<progress value={batch.progress_percentage} max={100} />
<span>{batch.progress_percentage}%</span>
```

### Polling Strategy (Recommended)
```javascript
// OLD: Poll every 1 second during processing
// NEW: Poll every 5 seconds (progress now visible)
const interval = batch.status === 'VALIDATING' || batch.status === 'COMMITTING' 
  ? 5000  // 5 seconds when processing
  : 10000; // 10 seconds when idle
```

## Testing Checklist

- [x] Build: `go build ./...` — SUCCESS
- [x] Unit tests: `go test ./internal/importing/... -v` — 24 PASS
- [x] Migration up/down/up — reversible, no errors
- [x] Database schema — column exists with correct type
- [x] API response — `progress_percentage` field present in `GET /imports/{id}`
- [x] Validation tracking — progress updates 0→100 during validation
- [x] Commit tracking — progress updates 0→100 during commit
- [x] Boundary checking — progress clamped [0–100], never exceeds
- [x] No regression — existing tests still pass, no breaking changes

## Backwards Compatibility

✅ **Fully backwards compatible:**
- Existing clients that don't read `progress_percentage` are unaffected
- Field is pure addition, no existing fields removed or renamed
- JSON response includes new field, but old clients can ignore it
- Default value (0) safe for old batches (pre-migration)

## Migration Impact

- **Uptime:** None — ADD COLUMN is non-blocking in MySQL
- **Data loss:** None — rollback includes DROP COLUMN
- **Performance:** None — default value (0) set, no index added
- **Storage:** +4 bytes per batch row (INT UNSIGNED)

## Known Limitations

1. **Progress updates only during active processing** — idle batches show last progress value
2. **No WebSocket/real-time push** — polling-based approach, sufficient for current use case
3. **Large files (10k+ rows)** — progress update every 100 rows = ~100 updates, acceptable overhead
4. **Concurrent validation/commit** — updates not atomic (acceptable trade-off for simplicity)

## Future Enhancements (Not in Scope)

- [ ] WebSocket for real-time progress push
- [ ] Server-sent events (SSE) for unidirectional stream
- [ ] ETA calculation based on historical processing rate
- [ ] Progress granularity control (e.g., update every N rows, not hardcoded 100)
