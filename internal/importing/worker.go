package importing

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"backend_crm_piposmart/internal/customer"
	"backend_crm_piposmart/internal/lead"
	"backend_crm_piposmart/internal/platform/jobqueue"
)

func isBlankRow(row []string) bool {
	for _, cell := range row {
		if cell != "" {
			for _, r := range cell {
				if r != ' ' && r != '\t' {
					return false
				}
			}
		}
	}
	return true
}

// ValidateHandler adapts the parse/validate pipeline to jobqueue.Handler. A malformed workbook
// or undetectable profile is a data problem, not a transient failure — it records
// VALIDATION_FAILED on the batch and returns nil (job COMPLETED), rather than retrying five times
// against a file that will never become valid.
func ValidateHandler(repo *Repository) jobqueue.Handler {
	return func(ctx context.Context, tx *sql.Tx, jobID int64, payload json.RawMessage) error {
		var p ValidateJobPayload
		if err := json.Unmarshal(payload, &p); err != nil {
			return fmt.Errorf("importing: unmarshal validate payload: %w", err)
		}

		batch, err := repo.GetBatchByID(ctx, p.BatchID)
		if err != nil {
			return err
		}

		if err := repo.SetBatchValidating(ctx, tx, p.BatchID); err != nil {
			return err
		}

		rows, err := readRows(batch.FilePath)
		if err != nil {
			return repo.SetValidationFailed(ctx, tx, p.BatchID, err.Error())
		}
		if len(rows) == 0 {
			return repo.SetValidationFailed(ctx, tx, p.BatchID, "workbook memiliki 0 baris")
		}

		profile := batch.Profile
		var headerRowIdx int
		var headers headerIndex
		if profile == "" || profile == ProfilePendingDetection {
			detected, hRow, hIdx, derr := detectProfile(rows)
			if derr != nil {
				return repo.SetValidationFailed(ctx, tx, p.BatchID, derr.Error())
			}
			profile, headerRowIdx, headers = detected, hRow, hIdx
			if err := repo.SetBatchProfile(ctx, tx, p.BatchID, profile); err != nil {
				return err
			}
		} else {
			hRow, hIdx, verr := verifyProfile(rows, profile)
			if verr != nil {
				return repo.SetValidationFailed(ctx, tx, p.BatchID, verr.Error())
			}
			headerRowIdx, headers = hRow, hIdx
		}

		total, valid, invalid := 0, 0, 0
		dataRows := [][]string{}
		// Pre-collect non-blank rows to know total upfront
		for i := headerRowIdx + 1; i < len(rows); i++ {
			row := rows[i]
			if !isBlankRow(row) {
				dataRows = append(dataRows, row)
			}
		}
		totalData := len(dataRows)

		for idx, row := range dataRows {
			total++
			rowIndex := headerRowIdx + 2 + idx // 1-based Excel row number, for admin-facing reference

			var payloadData any
			var errs []string
			switch profile {
			case ProfileOwnerOutlet:
				payloadData, errs = parseOwnerOutletRow(row, headers)
			case ProfileNonRegister:
				payloadData, errs = parseNonRegisterRow(row, headers)
			}

			status := RowStatusValid
			if len(errs) > 0 {
				status = RowStatusInvalid
				invalid++
			} else {
				valid++
			}
			if err := repo.InsertRow(ctx, tx, p.BatchID, rowIndex, status, payloadData, errs); err != nil {
				return err
			}

			// Update progress every 100 rows or 10% of data
			if totalData > 0 && (total%100 == 0 || (totalData <= 100 && total == totalData)) {
				percentage := int(float64(total) / float64(totalData) * 100)
				if percentage > 100 {
					percentage = 100
				}
				_ = repo.UpdateProgress(ctx, p.BatchID, percentage)
			}
		}

		return repo.SetValidationResult(ctx, tx, p.BatchID, total, valid, invalid)
	}
}

// CommitHandler adapts the commit pipeline to jobqueue.Handler. Each row's entity creation goes
// through the existing, already-tested customer/lead services — reused as-is, not reimplemented
// here — and is isolated to its own internal transaction. One row's failure (e.g. a race on a
// unique code) is recorded on that row and does not abort the rest of the batch.
func CommitHandler(repo *Repository, customerService *customer.Service, leadService *lead.Service) jobqueue.Handler {
	return func(ctx context.Context, tx *sql.Tx, jobID int64, payload json.RawMessage) error {
		var p CommitJobPayload
		if err := json.Unmarshal(payload, &p); err != nil {
			return fmt.Errorf("importing: unmarshal commit payload: %w", err)
		}

		batch, err := repo.GetBatchByID(ctx, p.BatchID)
		if err != nil {
			return err
		}
		if batch.Status != BatchStatusCommitting {
			return fmt.Errorf("importing: batch %d is not COMMITTING (status=%s)", p.BatchID, batch.Status)
		}

		validRows, err := repo.ListRowsByStatus(ctx, p.BatchID, RowStatusValid)
		if err != nil {
			return err
		}

		customerActor := customer.Actor{ID: p.TriggeredByUserID, RoleCode: RoleAdmin}
		totalRows := len(validRows)

		committed := 0
		for idx, row := range validRows {
			var commitErr error
			switch batch.Profile {
			case ProfileOwnerOutlet:
				commitErr = commitOwnerOutletRow(ctx, repo, customerService, customerActor, row)
			case ProfileNonRegister:
				commitErr = commitNonRegisterRow(ctx, repo, customerService, customerActor, row)
			default:
				commitErr = fmt.Errorf("unknown profile %q", batch.Profile)
			}
			if commitErr != nil {
				if err := repo.MarkRowCommitFailed(ctx, row.ID, commitErr.Error()); err != nil {
					return err
				}
				continue
			}
			committed++

			// Update progress every 100 rows or 10% of data
			if totalRows > 0 && ((idx+1)%100 == 0 || (totalRows <= 100 && idx+1 == totalRows)) {
				percentage := int(float64(idx+1) / float64(totalRows) * 100)
				if percentage > 100 {
					percentage = 100
				}
				_ = repo.UpdateProgress(ctx, p.BatchID, percentage)
			}
		}

		return repo.SetCommitResult(ctx, tx, p.BatchID, committed, p.TriggeredByUserID)
	}
}

func commitOwnerOutletRow(ctx context.Context, repo *Repository, customerService *customer.Service, actor customer.Actor, row ImportRow) error {
	var r ownerOutletRow
	if err := json.Unmarshal(row.RawPayload, &r); err != nil {
		return fmt.Errorf("unmarshal row payload: %w", err)
	}

	ownerID, exists, err := repo.FindOwnerIDByCode(ctx, r.OwnerCode)
	if err != nil {
		return err
	}
	if !exists {
		ownerResp, err := customerService.CreateOwner(ctx, actor, customer.CreateOwnerRequest{
			Code:      r.OwnerCode,
			Name:      r.OwnerName,
			Phone:     r.OwnerPhone,
			Email:     r.OwnerEmail,
			BrandName: r.BrandName,
			Province:  r.Province,
			City:      r.City,
			Address:   r.Address,
		})
		if err != nil {
			if err == customer.ErrCodeAlreadyUsed {
				// Race with a concurrent commit/creation using the same owner_code — recover by
				// reusing whichever owner won, instead of failing this row outright.
				recoveredID, found, findErr := repo.FindOwnerIDByCode(ctx, r.OwnerCode)
				if findErr != nil || !found {
					return fmt.Errorf("create owner %s: %w", r.OwnerCode, err)
				}
				ownerID = recoveredID
			} else {
				return fmt.Errorf("create owner %s: %w", r.OwnerCode, err)
			}
		} else {
			ownerID = ownerResp.ID
		}
	}

	outletSeq, err := repo.CountOutletsForOwner(ctx, ownerID)
	if err != nil {
		return err
	}
	outletCode := fmt.Sprintf("%s-OUT-%02d", r.OwnerCode, outletSeq+1)
	outletResp, err := customerService.CreateOutlet(ctx, actor, ownerID, customer.CreateOutletRequest{
		Code:     outletCode,
		Name:     r.OutletName,
		Phone:    r.OutletPhone,
		Province: r.Province,
		City:     r.City,
		Address:  r.Address,
	})
	if err != nil {
		return fmt.Errorf("create outlet %s: %w", outletCode, err)
	}

	outletID := outletResp.ID
	return repo.MarkRowCommitted(ctx, row.ID, ownerID, &outletID, nil)
}

// commitNonRegisterRow creates (or reuses) a minimal Owner for the row's phone number.
//
// It deliberately never calls lead.Service.CreateLead: customer.Service.CreateOwner already
// creates exactly one lead as part of owner creation (internal/customer/repository.go, existing
// Sprint 4 behavior — confirmed by testing against the real "Data Belum Registrasi" workbook,
// where every committed owner already had a lead by the time this function reached the lead
// step). Since lead codes are deterministically derived from owner_id ("LEAD-%06d"), which
// structurally allows only one lead per owner system-wide, attempting a second CreateLead call
// here would always collide with the one CreateOwner already made — it did on every single row
// before this was caught. The row's job is only to find and record that lead's ID, whether the
// owner was just created above or reused (e.g. two rows sharing the same phone number).
func commitNonRegisterRow(ctx context.Context, repo *Repository, customerService *customer.Service, customerActor customer.Actor, row ImportRow) error {
	var r nonRegisterRow
	if err := json.Unmarshal(row.RawPayload, &r); err != nil {
		return fmt.Errorf("unmarshal row payload: %w", err)
	}

	ownerCode := "NONREG-" + r.Phone
	ownerID, exists, err := repo.FindOwnerIDByCode(ctx, ownerCode)
	if err != nil {
		return err
	}
	if !exists {
		ownerResp, err := customerService.CreateOwner(ctx, customerActor, customer.CreateOwnerRequest{
			Code:  ownerCode,
			Name:  "Prospek " + r.Phone,
			Phone: r.Phone,
		})
		if err != nil {
			if err == customer.ErrCodeAlreadyUsed {
				recoveredID, found, findErr := repo.FindOwnerIDByCode(ctx, ownerCode)
				if findErr != nil || !found {
					return fmt.Errorf("create minimal owner %s: %w", ownerCode, err)
				}
				ownerID = recoveredID
			} else {
				return fmt.Errorf("create minimal owner %s: %w", ownerCode, err)
			}
		} else {
			ownerID = ownerResp.ID
		}
	}

	leadID, found, err := repo.FindLeadIDByOwnerID(ctx, ownerID)
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("owner %d has no lead (unexpected: CreateOwner should always create one)", ownerID)
	}
	return repo.MarkRowCommitted(ctx, row.ID, ownerID, nil, &leadID)
}
