package importing

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"backend_crm_piposmart/internal/activity"
	"backend_crm_piposmart/internal/catalog"
	"backend_crm_piposmart/internal/closing"
	"backend_crm_piposmart/internal/customer"
	"backend_crm_piposmart/internal/identity"
	"backend_crm_piposmart/internal/lead"
	"backend_crm_piposmart/internal/platform/jobqueue"
	"backend_crm_piposmart/internal/subscription"
	"backend_crm_piposmart/internal/target"
	"backend_crm_piposmart/internal/wallet"
)

// unmatchedError marks a row as a reconciliation candidate rather than a hard commit failure —
// the row is structurally valid but references an owner/outlet/partner/package/sales-user code
// that doesn't exist yet (Sprint 15's transactional profiles assume it should already exist from
// prior imports, unlike Sprint 14's OWNER_OUTLET/NON_REGISTER which auto-create).
type unmatchedError struct{ reason string }

func (e *unmatchedError) Error() string { return e.reason }

func unmatchedf(format string, args ...any) error {
	return &unmatchedError{reason: fmt.Sprintf(format, args...)}
}

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

		sheets, err := readAllSheets(batch.FilePath)
		if err != nil {
			return repo.SetValidationFailed(ctx, tx, p.BatchID, err.Error())
		}
		if len(sheets) == 0 {
			return repo.SetValidationFailed(ctx, tx, p.BatchID, "workbook memiliki 0 sheet")
		}

		profile := batch.Profile
		var sheetIdx, headerRowIdx int
		var headers headerIndex
		switch {
		case batch.SheetName != "":
			// Declared sheet_name (SALES_CALL_CHAT/SALES_TARGET) — never auto-detect, only verify
			// that the named sheet's headers match the declared profile. Service.Upload already
			// guarantees profile is non-empty whenever sheet_name is set. Trimmed comparison:
			// the real PBGC workbook's own sheet names carry stray trailing spaces (e.g. "TARGET "),
			// and form-encoded values passing through HTTP/curl/browsers routinely lose trailing
			// whitespace — an exact match would make this required field fail unpredictably.
			wantSheet := strings.TrimSpace(batch.SheetName)
			found := -1
			for i, sheet := range sheets {
				if strings.TrimSpace(sheet.Name) == wantSheet {
					found = i
					break
				}
			}
			if found == -1 {
				return repo.SetValidationFailed(ctx, tx, p.BatchID, ErrSheetNotFound.Error())
			}
			hRow, hIdx, verr := verifyProfile(sheets[found].Rows, profile)
			if verr != nil {
				return repo.SetValidationFailed(ctx, tx, p.BatchID, verr.Error())
			}
			sheetIdx, headerRowIdx, headers = found, hRow, hIdx
		case profile == "" || profile == ProfilePendingDetection:
			detected, sIdx, hRow, hIdx, derr := detectProfileAcrossSheets(sheets)
			if derr != nil {
				return repo.SetValidationFailed(ctx, tx, p.BatchID, derr.Error())
			}
			profile, sheetIdx, headerRowIdx, headers = detected, sIdx, hRow, hIdx
			if err := repo.SetBatchProfile(ctx, tx, p.BatchID, profile); err != nil {
				return err
			}
		default:
			sIdx, hRow, hIdx, verr := verifyProfileAcrossSheets(sheets, profile)
			if verr != nil {
				return repo.SetValidationFailed(ctx, tx, p.BatchID, verr.Error())
			}
			sheetIdx, headerRowIdx, headers = sIdx, hRow, hIdx
		}
		rows := sheets[sheetIdx].Rows
		if len(rows) == 0 {
			return repo.SetValidationFailed(ctx, tx, p.BatchID, "workbook memiliki 0 baris")
		}

		// SALES_TARGET's "TARGET" sheet only carries the year in its merged title rows, never per
		// data row — resolved once here rather than re-scanned per row.
		var salesTargetYear int
		if profile == ProfileSalesTarget {
			salesTargetYear, _ = yearInTitleRows(rows, headerRowIdx)
		}
		var monthlyActiveMonths []monthColumn
		if profile == ProfileMonthlyActive {
			monthlyActiveMonths, err = inferMonthlyActiveColumns(rows, headerRowIdx, headers)
			if err != nil {
				return repo.SetValidationFailed(ctx, tx, p.BatchID, err.Error())
			}
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
			case ProfileNewSubscribe:
				payloadData, errs = parseNewSubscribeRow(row, headers)
			case ProfileMonthlyActive:
				payloadData, errs = parseMonthlyActiveRow(row, headers, monthlyActiveMonths)
			case ProfileBonusMitra:
				payloadData, errs = parseBonusMitraRow(row, headers)
			case ProfileSalesTarget:
				payloadData, errs = parseSalesTargetRow(row, headers, salesTargetYear)
			case ProfileSalesCallChat:
				payloadData, errs = parseSalesCallChatRow(row, headers)
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
				_ = repo.UpdateProgressWithExec(ctx, tx, p.BatchID, percentage)
			}
		}

		return repo.SetValidationResult(ctx, tx, p.BatchID, total, valid, invalid)
	}
}

// CommitHandler adapts the commit pipeline to jobqueue.Handler. Each row's entity creation goes
// through the existing, already-tested customer/lead services — reused as-is, not reimplemented
// here — and is isolated to its own internal transaction. One row's failure (e.g. a race on a
// unique code) is recorded on that row and does not abort the rest of the batch.
func CommitHandler(
	repo *Repository,
	customerService *customer.Service,
	leadService *lead.Service,
	walletService *wallet.Service,
	subscriptionService *subscription.Service,
	catalogService *catalog.Service,
	targetService *target.Service,
	activityService *activity.Service,
	closingService *closing.Service,
) jobqueue.Handler {
	return func(ctx context.Context, tx *sql.Tx, jobID int64, payload json.RawMessage) error {
		var p CommitJobPayload
		if err := json.Unmarshal(payload, &p); err != nil {
			return fmt.Errorf("importing: unmarshal commit payload: %w", err)
		}

		batch, err := repo.GetBatchByID(ctx, p.BatchID)
		if err != nil {
			return err
		}
		if !canWorkerProcessCommit(batch.Status) {
			return fmt.Errorf("importing: batch %d is not commitable by worker (status=%s)", p.BatchID, batch.Status)
		}
		if batch.Status == BatchStatusValidated {
			if err := repo.SetBatchCommitting(ctx, p.BatchID); err != nil {
				return err
			}
		}

		validRows, err := repo.ListRowsByStatus(ctx, p.BatchID, RowStatusValid)
		if err != nil {
			return err
		}

		customerActor := customer.Actor{ID: p.TriggeredByUserID, RoleCode: RoleAdmin}
		identityActor := identity.User{ID: p.TriggeredByUserID, RoleCode: RoleAdmin}
		totalRows := len(validRows)

		committed := 0
		for idx, row := range validRows {
			var commitErr error
			switch batch.Profile {
			case ProfileOwnerOutlet:
				commitErr = commitOwnerOutletRow(ctx, repo, customerService, customerActor, row)
			case ProfileNonRegister:
				commitErr = commitNonRegisterRow(ctx, repo, customerService, customerActor, row)
			case ProfileNewSubscribe:
				commitErr = commitNewSubscribeRow(ctx, repo, identityActor, walletService, subscriptionService, catalogService, row)
			case ProfileMonthlyActive:
				commitErr = commitMonthlyActiveRow(ctx, repo, batch.ID, row)
			case ProfileBonusMitra:
				commitErr = commitBonusMitraRow(ctx, repo, batch.ID, row)
			case ProfileSalesTarget:
				commitErr = commitSalesTargetRow(ctx, repo, identityActor, targetService, batch.TargetSalesUserID, row)
			case ProfileSalesCallChat:
				commitErr = commitSalesCallChatRow(ctx, repo, activityService, closingService, catalogService, batch.TargetSalesUserID, row)
			default:
				commitErr = fmt.Errorf("unknown profile %q", batch.Profile)
			}
			if commitErr != nil {
				var ue *unmatchedError
				if errors.As(commitErr, &ue) {
					if err := repo.MarkRowUnmatched(ctx, row.ID, ue.reason); err != nil {
						return err
					}
				} else if err := repo.MarkRowCommitFailed(ctx, row.ID, commitErr.Error()); err != nil {
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

// resolvePlanID finds the catalog plan matching a package display name + tenure in months (the
// only identifiers the source spreadsheets carry — no plan_id/plan_code column exists in any of
// them). Ambiguous or missing matches are the caller's job to turn into an UNMATCHED row, not an
// error here, since "which plan" is a data reconciliation question, not a parse failure.
func resolvePlanID(ctx context.Context, catalogService *catalog.Service, packageName string, tenorMonths int) (int64, bool, error) {
	plans, err := catalogService.ListPlans(ctx, catalog.ListParams{All: true})
	if err != nil {
		return 0, false, fmt.Errorf("list catalog plans: %w", err)
	}
	for _, plan := range plans.Items {
		if strings.EqualFold(strings.TrimSpace(plan.Package.Name), strings.TrimSpace(packageName)) && plan.TenureMonths == tenorMonths {
			return plan.ID, true, nil
		}
	}
	return 0, false, nil
}

// commitNewSubscribeRow turns one "02. New & Subscribe" row into a wallet top-up (Sprint 9) plus
// a subscription order (Sprint 10) against the referenced, already-existing owner/outlet. Unlike
// OWNER_OUTLET/NON_REGISTER, nothing here is auto-created: a missing owner, outlet, or catalog
// plan all become UNMATCHED reconciliation candidates. wallet.CreateTopup and
// subscription.CreateOrder are both natively idempotent on IdempotencyKey/ExternalReference, so
// re-committing an already-committed row (e.g. a retried batch) is safe by construction.
func commitNewSubscribeRow(
	ctx context.Context,
	repo *Repository,
	actor identity.User,
	walletService *wallet.Service,
	subscriptionService *subscription.Service,
	catalogService *catalog.Service,
	row ImportRow,
) error {
	var r newSubscribeRow
	if err := json.Unmarshal(row.RawPayload, &r); err != nil {
		return fmt.Errorf("unmarshal row payload: %w", err)
	}

	ownerID, found, err := repo.FindOwnerIDByCode(ctx, r.OwnerCode)
	if err != nil {
		return err
	}
	if !found {
		return unmatchedf("owner dengan kode %s tidak ditemukan", r.OwnerCode)
	}

	var outletIDPtr *int64
	if r.OutletName != "" {
		outletID, outletFound, err := repo.FindOutletIDByOwnerAndName(ctx, ownerID, r.OutletName)
		if err != nil {
			return err
		}
		if !outletFound {
			return unmatchedf("outlet %q milik owner %s tidak ditemukan", r.OutletName, r.OwnerCode)
		}
		outletIDPtr = &outletID
	}

	planID, planFound, err := resolvePlanID(ctx, catalogService, r.PaketMembership, r.TenorMonths)
	if err != nil {
		return err
	}
	if !planFound {
		return unmatchedf("paket %q tenor %d bulan tidak cocok dengan katalog manapun", r.PaketMembership, r.TenorMonths)
	}

	var paidAt *time.Time
	if r.TanggalAktivasi != "" {
		if t, err := time.Parse("2006-01-02", r.TanggalAktivasi); err == nil {
			paidAt = &t
		}
	}

	if _, err := walletService.CreateTopup(ctx, actor, ownerID, wallet.CreateTopupRequest{
		Amount:            r.NominalAktivasi,
		PaymentChannel:    r.MetodePembayaran,
		ExternalReference: r.Kode,
		IdempotencyKey:    r.Kode,
		PaidAt:            paidAt,
		Note:              "Import NEW_SUBSCRIBE " + r.Kode,
	}); err != nil {
		return fmt.Errorf("create topup for %s: %w", r.Kode, err)
	}

	if _, err := subscriptionService.CreateOrder(ctx, actor, ownerID, subscription.CreateOrderRequest{
		PlanID:                planID,
		OutletID:              outletIDPtr,
		ExternalReference:     r.Kode,
		IdempotencyKey:        r.Kode,
		PurchasedAt:           paidAt,
		SubscriptionStartDate: r.TanggalAktivasi,
		Note:                  "Import NEW_SUBSCRIBE " + r.Kode,
	}); err != nil {
		return fmt.Errorf("create subscription order for %s: %w", r.Kode, err)
	}

	return repo.MarkRowCommitted(ctx, row.ID, ownerID, outletIDPtr, nil)
}

// commitMonthlyActiveRow stores one outlet's month-by-month historical activity timeline into the
// append-only snapshot table added in Sprint 15. The source workbook is explicitly historical and
// coarse-grained ("active this month" codes only), so it does NOT attempt to backfill precise
// subscription periods/orders â€” that would invent dates the file never actually contains.
func commitMonthlyActiveRow(ctx context.Context, repo *Repository, batchID int64, row ImportRow) error {
	var r monthlyActiveRow
	if err := json.Unmarshal(row.RawPayload, &r); err != nil {
		return fmt.Errorf("unmarshal row payload: %w", err)
	}

	ownerID, found, err := repo.FindOwnerIDByCode(ctx, r.OwnerCode)
	if err != nil {
		return err
	}
	if !found {
		return unmatchedf("owner dengan kode %s tidak ditemukan", r.OwnerCode)
	}

	var outletID int64
	outletNames := []string{r.OutletName, r.ProjectName}
	for _, name := range outletNames {
		if name == "" {
			continue
		}
		id, ok, err := repo.FindOutletIDByOwnerAndName(ctx, ownerID, name)
		if err != nil {
			return err
		}
		if ok {
			outletID = id
			break
		}
	}
	if outletID == 0 {
		return unmatchedf("outlet %q milik owner %s tidak ditemukan", r.OutletName, r.OwnerCode)
	}

	for _, activity := range r.Activities {
		if err := repo.UpsertOutletMonthlyActivitySnapshot(ctx, batchID, outletID, activity); err != nil {
			return err
		}
	}

	return repo.MarkRowCommitted(ctx, row.ID, ownerID, &outletID, nil)
}

// commitBonusMitraRow snapshots one historical referral/bonus row from the "Mitra - Referral"
// sheet. We intentionally persist it as imported fact data first, linked to the referenced owner/
// outlet/lead when those already exist, instead of force-fitting it into the live commission
// ledger: the workbook contains payout-history columns and partial states that may predate the
// CRM's own partner-referral lifecycle.
func commitBonusMitraRow(ctx context.Context, repo *Repository, batchID int64, row ImportRow) error {
	var r bonusMitraRow
	if err := json.Unmarshal(row.RawPayload, &r); err != nil {
		return fmt.Errorf("unmarshal row payload: %w", err)
	}

	ownerID, found, err := repo.FindOwnerIDByCode(ctx, r.ReferredOwnerCode)
	if err != nil {
		return err
	}
	if !found {
		return unmatchedf("owner referral dengan kode %s tidak ditemukan", r.ReferredOwnerCode)
	}

	var outletIDPtr *int64
	for _, name := range []string{r.ReferredOutletName, r.ReferredProject} {
		if name == "" {
			continue
		}
		outletID, ok, err := repo.FindOutletIDByOwnerAndName(ctx, ownerID, name)
		if err != nil {
			return err
		}
		if ok {
			outletIDPtr = &outletID
			break
		}
	}

	var leadIDPtr *int64
	if leadID, ok, err := repo.FindLeadIDByOwnerID(ctx, ownerID); err != nil {
		return err
	} else if ok {
		leadIDPtr = &leadID
	}

	if err := repo.UpsertPartnerBonusReferralSnapshot(ctx, batchID, row.RowIndex, ownerID, outletIDPtr, leadIDPtr, r); err != nil {
		return err
	}

	return repo.MarkRowCommitted(ctx, row.ID, ownerID, outletIDPtr, leadIDPtr)
}

// metricCodeTargetUser/metricCodeTargetOmset map the "TARGET" sheet's two target columns onto
// Sprint 13's existing metric_codes registry. Neither the sheet nor the registry uses identical
// names — CONFIRMED_CLOSING_COUNT/CONFIRMED_CLOSING_AMOUNT are the closest existing metrics
// ("Target User" = new memberships, i.e. confirmed closings; "Target Omset" = revenue from them),
// and no other seeded metric_code fits either column at all.
const (
	metricCodeTargetUser  = "CONFIRMED_CLOSING_COUNT"
	metricCodeTargetOmset = "CONFIRMED_CLOSING_AMOUNT"
)

// commitSalesTargetRow upserts one month's target for the batch's declared Sales user (target
// overrides are idempotent by construction — target.Service.OverrideTarget always wins over any
// bulk value and simply overwrites on re-commit, never duplicates).
func commitSalesTargetRow(ctx context.Context, repo *Repository, actor identity.User, targetService *target.Service, targetSalesUserID sql.NullInt64, row ImportRow) error {
	if !targetSalesUserID.Valid {
		return unmatchedf("batch tidak memiliki target_sales_user_id")
	}
	var r salesTargetRow
	if err := json.Unmarshal(row.RawPayload, &r); err != nil {
		return fmt.Errorf("unmarshal row payload: %w", err)
	}

	if _, err := targetService.OverrideTarget(ctx, actor, targetSalesUserID.Int64, target.OverrideTargetRequest{
		PeriodYear:  r.PeriodYear,
		PeriodMonth: r.PeriodMonth,
		MetricCode:  metricCodeTargetUser,
		TargetValue: r.TargetUser,
	}); err != nil {
		if errors.Is(err, target.ErrSalesNotEligible) || errors.Is(err, target.ErrNotFound) {
			return unmatchedf("sales user %d tidak ditemukan/tidak aktif", targetSalesUserID.Int64)
		}
		return fmt.Errorf("override target user %d-%02d: %w", r.PeriodYear, r.PeriodMonth, err)
	}

	if _, err := targetService.OverrideTarget(ctx, actor, targetSalesUserID.Int64, target.OverrideTargetRequest{
		PeriodYear:  r.PeriodYear,
		PeriodMonth: r.PeriodMonth,
		MetricCode:  metricCodeTargetOmset,
		TargetValue: r.TargetOmset,
	}); err != nil {
		return fmt.Errorf("override target omset %d-%02d: %w", r.PeriodYear, r.PeriodMonth, err)
	}

	return repo.MarkRowCommittedNoEntity(ctx, row.ID)
}

// resolvePlanByPackageAndPrice finds the catalog plan matching a package name + exact price — used
// when the source data has a package name but no tenure ("Call & Chat-Lidya" has neither a Tenor
// nor a Plan Code column, only "Finalisasi (Closing)" naming the package and "Nominal" as what was
// actually paid). An exact price match is the only safe way to recover which tenure was meant;
// guessing a default tenure would silently misstate subscription length and revenue.
func resolvePlanByPackageAndPrice(ctx context.Context, catalogService *catalog.Service, packageName, price string) (int64, bool, error) {
	wantPrice, err := strconv.ParseFloat(price, 64)
	if err != nil {
		return 0, false, nil
	}
	plans, err := catalogService.ListPlans(ctx, catalog.ListParams{All: true})
	if err != nil {
		return 0, false, fmt.Errorf("list catalog plans: %w", err)
	}
	for _, plan := range plans.Items {
		// Compare numerically, not as strings: the source data has no decimal places ("1128600")
		// while catalog prices always carry two ("1128600.00") — an exact string match would never
		// hit, silently sending every real closing to reconciliation.
		planPrice, err := strconv.ParseFloat(plan.Price, 64)
		if err != nil {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(plan.Package.Name), strings.TrimSpace(packageName)) && planPrice == wantPrice {
			return plan.ID, true, nil
		}
	}
	return 0, false, nil
}

// commitSalesCallChatRow records one lead interaction from "Call & Chat-Lidya", and — when Scor is
// 3 with a real package name in Finalisasi — an accompanying Closing. Closing creation reuses
// closing.Service.CreateClosing as-is: its repository already inserts the Scor-3 interaction and
// the closing in one DB transaction (Sprint 8's "remark 3 dan closing dalam satu transaction"
// rule), so this function must NOT also call activity.Service.CreateInteraction for a closing row
// — that would double-record the interaction.
//
// closing.Service.CreateClosing requires the actor to BE the lead's current active Sales PIC
// (RoleCode == SALES, and repository-side salesOwnsLead check) — unlike every other Sprint 15
// profile's commit, which acts as the importing Admin. The batch's declared target_sales_user_id
// is used as that actor, and a mismatch (lead not currently assigned to that Sales user) is a
// legitimate reconciliation case, not a hard failure — it usually means live PIC assignment has
// moved on since this historical row was recorded.
func commitSalesCallChatRow(
	ctx context.Context,
	repo *Repository,
	activityService *activity.Service,
	closingService *closing.Service,
	catalogService *catalog.Service,
	targetSalesUserID sql.NullInt64,
	row ImportRow,
) error {
	if !targetSalesUserID.Valid {
		return unmatchedf("batch tidak memiliki target_sales_user_id")
	}
	var r salesCallChatRow
	if err := json.Unmarshal(row.RawPayload, &r); err != nil {
		return fmt.Errorf("unmarshal row payload: %w", err)
	}

	ownerID, found, err := repo.FindOwnerIDByCode(ctx, r.OwnerCode)
	if err != nil {
		return err
	}
	if !found {
		return unmatchedf("owner dengan kode %s tidak ditemukan", r.OwnerCode)
	}
	leadID, found, err := repo.FindLeadIDByOwnerID(ctx, ownerID)
	if err != nil {
		return err
	}
	if !found {
		return unmatchedf("owner %s tidak memiliki lead", r.OwnerCode)
	}

	salesActor := identity.User{ID: targetSalesUserID.Int64, RoleCode: activity.RoleSales}
	if r.IsClosing {
		existing, err := closingService.ListClosings(ctx, salesActor, closing.ListParams{LeadID: &leadID, All: true})
		if err != nil && !errors.Is(err, closing.ErrForbidden) {
			return fmt.Errorf("check existing closings for lead %d: %w", leadID, err)
		}
		if existing.Pagination.Total > 0 {
			// Already satisfied by a prior commit (or the live system) — re-committing must not
			// create a second Closing for the same historical event. closing.Service has no
			// idempotency key of its own, unlike wallet/subscription, so this check is the only
			// guard against duplicating revenue-affecting records on a retried commit.
			return repo.MarkRowCommitted(ctx, row.ID, ownerID, nil, &leadID)
		}

		planID, planFound, err := resolvePlanByPackageAndPrice(ctx, catalogService, r.FinalisasiPaket, r.Nominal)
		if err != nil {
			return err
		}
		if !planFound {
			return unmatchedf("paket %q seharga %s tidak cocok dengan plan katalog manapun", r.FinalisasiPaket, r.Nominal)
		}

		var closedAt *time.Time
		if r.TanggalFU != "" {
			if t, err := time.Parse("2006-01-02", r.TanggalFU); err == nil {
				closedAt = &t
			}
		}

		if _, err := closingService.CreateClosing(ctx, salesActor, leadID, closing.CreateClosingRequest{
			PlanID:           planID,
			ClosedAt:         closedAt,
			CallStatus:       r.CallStatus,
			ChatStatus:       r.ChatStatus,
			ContactName:      r.OwnerName,
			ContactPhone:     r.OwnerPhone,
			CustomerResponse: r.Validitas,
			Note:             r.Remaks,
		}); err != nil {
			if errors.Is(err, closing.ErrForbidden) || errors.Is(err, closing.ErrLeadHasNoPIC) {
				return unmatchedf("lead %d bukan milik sales user %d saat ini (PIC sudah berubah)", leadID, targetSalesUserID.Int64)
			}
			return fmt.Errorf("create closing for lead %d: %w", leadID, err)
		}
		return repo.MarkRowCommitted(ctx, row.ID, ownerID, nil, &leadID)
	}

	var remarkScore *int64
	if r.Scor >= 0 {
		score := int64(r.Scor)
		remarkScore = &score
	}
	if _, err := activityService.CreateInteraction(ctx, salesActor, leadID, activity.CreateInteractionRequest{
		CallStatus:       r.CallStatus,
		ChatStatus:       r.ChatStatus,
		ContactName:      r.OwnerName,
		ContactPhone:     r.OwnerPhone,
		RemarkScore:      remarkScore,
		Note:             r.Remaks,
		CustomerResponse: r.Validitas,
	}); err != nil {
		return fmt.Errorf("create interaction for lead %d: %w", leadID, err)
	}

	return repo.MarkRowCommitted(ctx, row.ID, ownerID, nil, &leadID)
}
