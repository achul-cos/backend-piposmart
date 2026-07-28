# Investigation: Import Excel Frontend Polling Issue

**Date:** 2026-07-28  
**Issue:** Frontend repeatedly polling `/api/v1/imports/{id}` every ~1 second during import processing  
**Root Cause:** Backend did not track progress — only status changes (VALIDATING → VALIDATED)  
**Fix:** Add `progress_percentage` field to track upload/validation progress in real-time

## Problem Analysis

Frontend was observed making HTTP GET requests to `/api/v1/imports/2` with ~1 second intervals:

```
Request: GET https://backend-piposmart-production.up.railway.app/api/v1/imports/2
Status: 200 OK
Interval: ~1 second
```

### Why This Happened

1. **No progress field in response** — `GET /imports/{id}` returned only:
   - Status (UPLOADED, VALIDATING, VALIDATED, VALIDATION_FAILED)
   - Counts (total_rows, valid_rows, invalid_rows, committed_rows)
   - No indication of processing progress

2. **Long processing times** — Validation job could take minutes for large files:
   - Frontend had no way to know if validation was stalled or just slow
   - Users saw no feedback on upload progress
   - Frontend needed to poll to detect completion

3. **No intermediate updates** — Backend only updated status at END of validation:
   - ValidateHandler read all rows, validated them all, then called SetValidationResult once
   - Progress was 0% until complete, then jumped to 100%
   - Workers processing could take seconds/minutes per batch

### Impact

- **Poor UX:** Users see spinning loader with no indication of progress
- **Network waste:** Unnecessary polling 60+ times per minute per user
- **Latency:** Frontend waiting 1 second between polls adds perceived delay
- **Uncertainty:** Users don't know if upload is working or stuck

## Root Cause

Database schema for `import_batches` was missing a progress tracking column:

```sql
-- Sprint 14 (existing schema)
CREATE TABLE import_batches (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    -- ... other fields ...
    total_rows INT UNSIGNED NOT NULL DEFAULT 0,
    valid_rows INT UNSIGNED NOT NULL DEFAULT 0,
    invalid_rows INT UNSIGNED NOT NULL DEFAULT 0,
    committed_rows INT UNSIGNED NOT NULL DEFAULT 0,
    -- MISSING: progress_percentage field
);
```

Worker ValidateHandler processed all rows in a single pass:

```go
// Sprint 14 (old implementation)
for i := headerRowIdx + 1; i < len(rows); i++ {
    // ... parse & validate each row ...
    // No progress updates here
}
// Only called AFTER all rows done:
return repo.SetValidationResult(ctx, tx, p.BatchID, total, valid, invalid)
```

Frontend had no choice but to poll:

```javascript
// Frontend strategy (forced by lack of progress)
const pollInterval = setInterval(async () => {
    const resp = await GET `/imports/{id}`
    if (resp.status === 'VALIDATED') {
        clearInterval(pollInterval)
        // show results
    }
    // else: keep polling, status still VALIDATING
}, 1000) // 1 second interval
```

## Solution Implemented

### 1. Database Schema
Added `progress_percentage INT UNSIGNED NOT NULL DEFAULT 0` column to track progress (0–100%).

### 2. Type System
Updated Go types to include progress field:
- `ImportBatch.ProgressPercentage`
- `ImportBatchResponse.progress_percentage` (JSON)

### 3. Repository Layer
Added `UpdateProgress()` method to periodically update progress during processing.

### 4. Worker Implementation
- **ValidateHandler:** Updates progress every 100 rows (or per-row for small files)
- **CommitHandler:** Updates progress every 100 rows during entity creation

### 5. Status Transitions
```
Upload  → status: UPLOADED,    progress: 0%
Validate → status: VALIDATING, progress: 0–100% (updates periodically)
        → status: VALIDATED,   progress: 100%
Commit   → status: COMMITTING, progress: 0–100% (updates periodically)
        → status: COMMITTED,   progress: 100%
```

## Verification

### API Response Before (Sprint 14)
```json
{
  "id": 1,
  "status": "VALIDATING",
  "total_rows": 0,
  "valid_rows": 0,
  "invalid_rows": 0,
  // No progress field
}
```

### API Response After (Sprint 14d)
```json
{
  "id": 1,
  "status": "VALIDATING",
  "total_rows": 0,
  "valid_rows": 0,
  "invalid_rows": 0,
  "progress_percentage": 45  // <-- NEW
}
```

### Polling Behavior

**Before (constant polling every 1s):**
```
GET /imports/1 → status=VALIDATING, no progress info
GET /imports/1 → status=VALIDATING, no progress info
GET /imports/1 → status=VALIDATING, no progress info
GET /imports/1 → status=VALIDATED, done
```

**After (can reduce to 5s polling with progress visibility):**
```
GET /imports/1 → status=VALIDATING, progress=25%
GET /imports/1 → status=VALIDATING, progress=50%
GET /imports/1 → status=VALIDATING, progress=75%
GET /imports/1 → status=VALIDATED, progress=100%, done
```

## Lessons Learned

1. **Progress tracking is essential for long-running operations** — polling without progress indication leads to excessive requests and poor UX.

2. **Intermediate feedback improves perception** — showing "45%" is psychologically better than a spinning loader, even with the same total time.

3. **Batch processing should track progress** — updating progress every 100 rows is a good balance between DB overhead and responsiveness.

4. **Backend should enable efficient polling** — providing progress reduces necessary polling frequency from 1s to 5s (5x reduction).

## Frontend Recommendations

1. **Read progress from API:**
   ```javascript
   const batch = await fetch(`/api/v1/imports/${id}`).then(r => r.json())
   console.log(`${batch.progress_percentage}% complete`)
   ```

2. **Display progress bar:**
   ```jsx
   <progress value={batch.progress_percentage} max={100} />
   ```

3. **Reduce polling interval:**
   ```javascript
   const interval = (batch.status === 'VALIDATING' || batch.status === 'COMMITTING')
     ? 5000  // 5 seconds (down from 1 second)
     : 10000 // 10 seconds (idle)
   ```

4. **Optional: Add ETA estimation** (future enhancement):
   - Track processing rate: rows/second
   - Estimate time remaining based on (total - processed) / rate

## Future Improvements

- [ ] WebSocket/SSE for real-time progress push (if polling becomes bottleneck)
- [ ] ETA calculation on client-side
- [ ] Abort support (cancel in-progress job)
- [ ] Progress granularity control (configurable update frequency)

## References

- Related: Sprint 14 Import Framework (`docs/sprint-14/`)
- Database: `migrations/20260728000100_add_import_progress.sql`
- Code: `internal/importing/` (types, repository, worker)
- Testing: `docs/sprint-14d/README.md`
