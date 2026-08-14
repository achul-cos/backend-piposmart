package customer

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"backend_crm_piposmart/internal/platform/httpx"

	"github.com/gin-gonic/gin"
)

const (
	OutletSubscriptionStatusNotSubscribe = "NOT_SUBSCRIBE"
	OutletSubscriptionStatusTrial        = "TRIAL"
	OutletSubscriptionStatusInactive     = "UNSUBSCRIBE"
	OutletSubscriptionDueStatusNoPackage = "NO_PACKAGE"
	OutletSubscriptionStatusDueThisMonth = "JATUH_TEMPO"
	OutletSubscriptionStatusWillBeDue    = "AKAN_JATUH_TEMPO"
	OutletSubscriptionStatusDue          = "JATUH_TEMPO"
	OutletSubscriptionStatusPassedDue    = "TELAH_JATUH_TEMPO"
	OutletSubscriptionStatusSubscribed   = "SUBSCRIBE"
	OutletSubscriptionStatusNew          = "NEW"
	OutletSubscriptionLabelOneMonth      = "BERLANGGANAN 1 BULAN"
	NeverSubscribedDisplay               = "TIDAK PERNAH"
	TrialDurationDays                    = 14
)

type OutletSubscriptionSnapshot struct {
	OutletID                int64
	OutletCode              string
	OutletName              string
	OutletPhone             sql.NullString
	OutletProvince          sql.NullString
	OutletCity              sql.NullString
	OutletAddress           sql.NullString
	OutletStatus            string
	OwnerID                 sql.NullInt64
	OwnerCode               sql.NullString
	OwnerName               sql.NullString
	OwnerPhone              sql.NullString
	OwnerEmail              sql.NullString
	OwnerBrandName          sql.NullString
	OwnerCreatedAt          time.Time // untuk kalkulasi periode TRIAL (14 hari)
	OwnerHasAnySubscription bool      // true jika owner punya subscription di outlet manapun
	OutletSubscriptionCount int64     // riwayat jumlah subscription outlet

	PackageID               sql.NullInt64
	PackageCode             sql.NullString
	PackageName             sql.NullString
	PlanID                  sql.NullInt64
	PlanCode                sql.NullString
	PlanName                sql.NullString
	TenureMonths            sql.NullInt64
	SubscriptionStart       sql.NullTime
	SubscriptionEnd         sql.NullTime
	CreatedAt               time.Time
	UpdatedAt               time.Time
	LatestPIC               sql.NullString // No HP & Nama PIC terakhir dari customer_leads outlet ini
}

type OutletSubscriptionStatusParams struct {
	Query              string
	Code               string
	Name               string
	Phone              string
	BrandName          string
	Province           string
	City               string
	OwnerID            *int64
	SubscriptionStatus string
	CreationStatus     string
	StatusLangganan    string
	StatusJatuhTempo   string
	Month              string
	DueDate            string
	DueDateReference   string
	DueDateStart       string
	DueDateEnd         string
	StartDateStart     string
	StartDateEnd       string
	PackageName        string
	All                bool
	HasActivityOnly    bool
	Page               int
	Limit              int
	Sort               string
}

type PackagePlanBriefResponse struct {
	PackageID    *int64 `json:"package_id,omitempty"`
	PackageCode  string `json:"package_code,omitempty"`
	PackageName  string `json:"package_name,omitempty"`
	PlanID       *int64 `json:"plan_id,omitempty"`
	PlanCode     string `json:"plan_code,omitempty"`
	PlanName     string `json:"plan_name,omitempty"`
	TenureMonths *int64 `json:"tenure_months,omitempty"`
}

type OutletSubscriptionStatusResponse struct {
	OutletID                   int64                    `json:"outlet_id"`
	OutletCode                 string                   `json:"outlet_code"`
	OutletName                 string                   `json:"outlet_name"`
	OutletPhone                string                   `json:"outlet_phone,omitempty"`
	OutletProvince             string                   `json:"outlet_province,omitempty"`
	OutletCity                 string                   `json:"outlet_city,omitempty"`
	OutletAddress              string                   `json:"outlet_address,omitempty"`
	Owner                      OwnerBriefResponse       `json:"owner"`
	SubscriptionStatusCode     string                   `json:"subscription_status_code"`
	SubscriptionStatusLabel    string                   `json:"subscription_status_label"`
	DueStatusCode              string                   `json:"due_status_code"`
	DueStatusLabel             string                   `json:"due_status_label"`
	RemainingDays              *int64                   `json:"remaining_days,omitempty"`
	RemainingDaysDisplay       string                   `json:"remaining_days_display"`
	LastSubscriptionEnd        string                   `json:"last_subscription_end,omitempty"`
	LastSubscriptionEndDisplay string                   `json:"last_subscription_end_display"`
	SubscriptionStartDate      string                   `json:"subscription_start_date,omitempty"`
	SubscriptionEndDate        string                   `json:"subscription_end_date,omitempty"`
	PackagePlan                PackagePlanBriefResponse `json:"package_plan"`
	CreationStatus             string                   `json:"creation_status"`
	CreatedAt                  time.Time                `json:"created_at"`
	UpdatedAt                  time.Time                `json:"updated_at"`
}

type OutletSubscriptionStatusListResponse struct {
	ReferenceMonth      string                             `json:"reference_month"`
	ReferenceMonthStart string                             `json:"reference_month_start"`
	ReferenceMonthEnd   string                             `json:"reference_month_end"`
	Items               []OutletSubscriptionStatusResponse `json:"items"`
	Pagination          PaginationMeta                     `json:"pagination"`
}

func (h *Handler) listOutletSubscriptionStatuses(c *gin.Context) {
	params := outletSubscriptionStatusParams(c)
	referenceMonth, isSpecificDate, err := resolveReferenceMonth(params.Month)
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "month harus format YYYY-MM atau YYYY-MM-DD", nil)
		return
	}
	response, err := h.service.ListOutletSubscriptionStatuses(c.Request.Context(), currentActor(c), params, referenceMonth, isSpecificDate)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, response)
}

func (h *Handler) listAllOutletSubscriptionStatuses(c *gin.Context) {
	params := outletSubscriptionStatusParams(c)
	params.All = true
	referenceMonth, isSpecificDate, err := resolveReferenceMonth(params.Month)
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "month harus format YYYY-MM atau YYYY-MM-DD", nil)
		return
	}
	response, err := h.service.ListOutletSubscriptionStatuses(c.Request.Context(), currentActor(c), params, referenceMonth, isSpecificDate)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, response)
}

func outletSubscriptionStatusParams(c *gin.Context) OutletSubscriptionStatusParams {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	params := OutletSubscriptionStatusParams{
		Query:              c.Query("q"),
		Code:               c.Query("code"),
		Name:               c.Query("name"),
		Phone:              c.Query("phone"),
		BrandName:          c.Query("brand_name"),
		Province:           c.Query("province"),
		City:               c.Query("city"),
		SubscriptionStatus: c.Query("subscription_status"),
		CreationStatus:     c.Query("creation_status"),
		StatusLangganan:    c.Query("status_langganan"),
		StatusJatuhTempo:   c.Query("status_jatuh_tempo"),
		Month: func() string {
			if m := c.Query("month"); m != "" {
				return m
			}
			if sm := c.Query("subscription_month"); sm != "" {
				return sm
			}
			return c.Query("due_date")
		}(),
		DueDate:          c.Query("due_date"),
		DueDateReference: c.Query("due_date_reference"),
		DueDateStart:     c.Query("due_date_start"),
		DueDateEnd:       c.Query("due_date_end"),
		StartDateStart:   c.Query("start_date_start"),
		StartDateEnd:     c.Query("start_date_end"),
		PackageName: func() string {
			if p := c.Query("package_name"); p != "" {
				return p
			}
			if p := c.Query("package"); p != "" {
				return p
			}
			return c.Query("plan")
		}(),
		All:              false,
		HasActivityOnly:  c.Query("has_activity_only") == "true",
		Page:             page,
		Limit:            limit,
		Sort:             c.Query("sort"),
	}
	if ownerID, err := strconv.ParseInt(c.Query("owner_id"), 10, 64); err == nil && ownerID > 0 {
		params.OwnerID = &ownerID
	}
	return params
}

func (s *Service) ListOutletSubscriptionStatuses(ctx context.Context, actor Actor, params OutletSubscriptionStatusParams, referenceMonth time.Time, isSpecificDate bool) (OutletSubscriptionStatusListResponse, error) {
	params = normalizeOutletSubscriptionStatusParams(params)
	if params.Phone != "" {
		phone, err := NormalizePhone(params.Phone)
		if err == nil {
			params.Phone = phone
		}
	}
	snapshots, total, err := s.repo.ListOutletSubscriptionSnapshots(ctx, actor, params, referenceMonth, isSpecificDate)
	if err != nil {
		return OutletSubscriptionStatusListResponse{}, err
	}
	items := make([]OutletSubscriptionStatusResponse, 0, len(snapshots))
	for _, snapshot := range snapshots {
		items = append(items, buildOutletSubscriptionStatusResponse(snapshot, referenceMonth, isSpecificDate, params.DueDateReference))
	}
	return OutletSubscriptionStatusListResponse{
		ReferenceMonth:      referenceMonth.Format("2006-01"),
		ReferenceMonthStart: monthStart(referenceMonth).Format("2006-01-02"),
		ReferenceMonthEnd:   monthEnd(referenceMonth).Format("2006-01-02"),
		Items:               items,
		Pagination: PaginationMeta{
			Page:  params.Page,
			Limit: resolveReturnedLimit(params.All, params.Limit, len(items), total),
			Total: total,
		},
	}, nil
}

func (r *Repository) ListOutletSubscriptionSnapshots(ctx context.Context, actor Actor, params OutletSubscriptionStatusParams, referenceMonth time.Time, isSpecificDate bool) ([]OutletSubscriptionSnapshot, int64, error) {
	where, args := outletSubscriptionStatusWhere(actor, params, referenceMonth, isSpecificDate)
	countArgs := append([]any{monthEnd(referenceMonth)}, args...)
	countQuery := outletSubscriptionLatestCTE() + `
		SELECT COUNT(*)
		FROM outlets ot
		LEFT JOIN owners o ON o.id = ot.owner_id
		LEFT JOIN latest_subscriptions ls ON ls.outlet_id = ot.id AND ls.rn = 1
		WHERE ` + where
	var total int64
	if err := r.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}
	orderBy, err := outletSubscriptionStatusOrderBy(params.Sort)
	if err != nil {
		return nil, 0, err
	}
	listArgs := append([]any{monthEnd(referenceMonth)}, args...)
	query := outletSubscriptionLatestCTE() + `
		SELECT
			ot.id, ot.code, ot.name, ot.phone, ot.province, ot.city, ot.address, ot.status,
			o.id, o.code, o.name, o.phone, o.email, o.brand_name,
			COALESCE(o.created_at, ot.created_at) AS owner_created_at,
			EXISTS(
				SELECT 1 FROM subscriptions s2
				JOIN outlets ot5 ON ot5.id = s2.outlet_id
				WHERE ot5.owner_id = ot.owner_id AND s2.deleted_at IS NULL
			) AS owner_has_any_subscription,
			(
				SELECT COUNT(*) FROM subscriptions s3
				WHERE s3.outlet_id = ot.id AND s3.deleted_at IS NULL
			) AS outlet_subscription_count,
			ls.package_id, ls.package_code, ls.package_name,
			ls.plan_id, ls.plan_code, ls.plan_name, ls.tenure_months,
			ls.active_from, ls.active_until,
			ot.created_at, ot.updated_at,
			(
				SELECT IF(
					TRIM(COALESCE(pic.phone, '')) != '',
					CONCAT(COALESCE(pic.name, ''), ' (', RIGHT(TRIM(pic.phone), 3), ')'),
					COALESCE(pic.name, '')
				)
				FROM customer_leads cl
				LEFT JOIN users pic ON pic.id = COALESCE(cl.active_sales_id, IF(cl.current_owner_role = 'SALES', cl.current_owner_user_id, NULL))
				WHERE cl.outlet_id = ot.id AND cl.deleted_at IS NULL
				ORDER BY cl.created_at DESC, cl.id DESC
				LIMIT 1
			) AS latest_pic
		FROM outlets ot
		LEFT JOIN owners o ON o.id = ot.owner_id
		LEFT JOIN latest_subscriptions ls ON ls.outlet_id = ot.id AND ls.rn = 1
		WHERE ` + where + `
		ORDER BY ` + orderBy
	if !params.All {
		offset := (params.Page - 1) * params.Limit
		listArgs = append(listArgs, params.Limit, offset)
		query += `
		LIMIT ? OFFSET ?`
	}
	rows, err := r.db.QueryContext(ctx, query, listArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := make([]OutletSubscriptionSnapshot, 0)
	for rows.Next() {
		var item OutletSubscriptionSnapshot
		if err := rows.Scan(
			&item.OutletID,
			&item.OutletCode,
			&item.OutletName,
			&item.OutletPhone,
			&item.OutletProvince,
			&item.OutletCity,
			&item.OutletAddress,
			&item.OutletStatus,
			&item.OwnerID,
			&item.OwnerCode,
			&item.OwnerName,
			&item.OwnerPhone,
			&item.OwnerEmail,
			&item.OwnerBrandName,
			&item.OwnerCreatedAt,
			&item.OwnerHasAnySubscription,
			&item.OutletSubscriptionCount,
			&item.PackageID,
			&item.PackageCode,
			&item.PackageName,
			&item.PlanID,
			&item.PlanCode,
			&item.PlanName,
			&item.TenureMonths,
			&item.SubscriptionStart,
			&item.SubscriptionEnd,
			&item.CreatedAt,
			&item.UpdatedAt,
			&item.LatestPIC,
		); err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func outletSubscriptionLatestCTE() string {
	return `
		WITH latest_subscriptions AS (
			SELECT
				s.outlet_id,
				s.package_id,
				sp.code AS package_code,
				sp.name AS package_name,
				s.plan_id,
				spl.code AS plan_code,
				spl.name AS plan_name,
				spl.tenure_months,
				s.active_from,
				s.active_until,
				ROW_NUMBER() OVER (PARTITION BY s.outlet_id ORDER BY s.active_from DESC, s.id DESC) AS rn
			FROM subscriptions s
			LEFT JOIN subscription_packages sp ON sp.id = s.package_id
			LEFT JOIN subscription_plans spl ON spl.id = s.plan_id
			WHERE s.deleted_at IS NULL
				AND s.active_from <= ?
		)
	`
}

func outletSubscriptionStatusWhere(actor Actor, params OutletSubscriptionStatusParams, referenceMonth time.Time, isSpecificDate bool) (string, []any) {
	where := []string{"ot.deleted_at IS NULL"}
	args := []any{}
	visibility, visibilityArgs := ownerVisibilityWhereByColumn(actor, "ot.owner_id")
	where = append(where, visibility)
	args = append(args, visibilityArgs...)
	if params.Query != "" {
		pattern := wordBoundaryRegexp(params.Query)
		where = append(where, "(ot.code REGEXP ? OR ot.name REGEXP ? OR ot.phone REGEXP ? OR ot.city REGEXP ? OR ot.province REGEXP ? OR o.code REGEXP ? OR o.name REGEXP ? OR o.brand_name REGEXP ?)")
		args = append(args, pattern, pattern, pattern, pattern, pattern, pattern, pattern, pattern)
	}
	if params.OwnerID != nil {
		where = append(where, "ot.owner_id = ?")
		args = append(args, *params.OwnerID)
	}
	if params.Code != "" {
		where = append(where, "ot.code LIKE ?")
		args = append(args, like(params.Code))
	}
	if params.Name != "" {
		where = append(where, "ot.name LIKE ?")
		args = append(args, like(params.Name))
	}
	if params.Phone != "" {
		where = append(where, "ot.phone LIKE ?")
		args = append(args, like(params.Phone))
	}
	if params.BrandName != "" {
		pattern := like(params.BrandName)
		where = append(where, "(o.brand_name LIKE ? OR o.name LIKE ? OR o.code LIKE ?)")
		args = append(args, pattern, pattern, pattern)
	}
	if params.Province != "" {
		where = append(where, "ot.province LIKE ?")
		args = append(args, like(params.Province))
	}
	if params.City != "" {
		where = append(where, "ot.city LIKE ?")
		args = append(args, like(params.City))
	}
	if params.PackageName != "" {
		pattern := like(params.PackageName)
		where = append(where, "(ls.package_name LIKE ? OR ls.package_code LIKE ? OR ls.plan_name LIKE ? OR ls.plan_code LIKE ?)")
		args = append(args, pattern, pattern, pattern, pattern)
	}
	if params.DueDate != "" {
		if dueDate, err := time.Parse("2006-01-02", params.DueDate); err == nil {
			where = append(where, "DATE(ls.active_until) = DATE(?)")
			args = append(args, dueDate)
		}
	}
	// Jika hanya salah satu tanggal diisi, gunakan batas bulan yang sama sebagai bound implisit
	// agar filter tidak menampilkan data dari seluruh waktu (beginning of time)
	effectiveDueDateStart := params.DueDateStart
	effectiveDueDateEnd := params.DueDateEnd
	if effectiveDueDateStart == "" && effectiveDueDateEnd != "" {
		// Hanya Sampai Tanggal diisi → batas bawah = awal bulan dari Sampai Tanggal
		if endDate, err := time.Parse("2006-01-02", effectiveDueDateEnd); err == nil {
			effectiveDueDateStart = time.Date(endDate.Year(), endDate.Month(), 1, 0, 0, 0, 0, endDate.Location()).Format("2006-01-02")
		}
	}
	if effectiveDueDateEnd == "" && effectiveDueDateStart != "" {
		// Hanya Dari Tanggal diisi → batas atas = akhir bulan dari Dari Tanggal
		if startDate, err := time.Parse("2006-01-02", effectiveDueDateStart); err == nil {
			lastDay := time.Date(startDate.Year(), startDate.Month()+1, 0, 23, 59, 59, 0, startDate.Location())
			effectiveDueDateEnd = lastDay.Format("2006-01-02")
		}
	}
	if effectiveDueDateStart != "" {
		if startDate, err := time.Parse("2006-01-02", effectiveDueDateStart); err == nil {
			where = append(where, "ls.active_until >= ?")
			args = append(args, startDate)
		}
	}
	if effectiveDueDateEnd != "" {
		if endDate, err := time.Parse("2006-01-02", effectiveDueDateEnd); err == nil {
			where = append(where, "ls.active_until <= ?")
			args = append(args, endDate.Add(23*time.Hour+59*time.Minute+59*time.Second))
		}
	}
	if params.CreationStatus != "" {
		st := strings.ToUpper(strings.TrimSpace(params.CreationStatus))
		if st == "NEW" {
			where = append(where, "YEAR(ot.created_at) = ? AND MONTH(ot.created_at) = ?")
			args = append(args, referenceMonth.Year(), int(referenceMonth.Month()))
		} else if st == "EXISTING" {
			where = append(where, "(YEAR(ot.created_at) < ? OR (YEAR(ot.created_at) = ? AND MONTH(ot.created_at) < ?))")
			args = append(args, referenceMonth.Year(), referenceMonth.Year(), int(referenceMonth.Month()))
		}
	}
	effectiveStartDateStart := params.StartDateStart
	effectiveStartDateEnd := params.StartDateEnd
	if effectiveStartDateStart == "" && effectiveStartDateEnd != "" {
		if endDate, err := time.Parse("2006-01-02", effectiveStartDateEnd); err == nil {
			effectiveStartDateStart = time.Date(endDate.Year(), endDate.Month(), 1, 0, 0, 0, 0, endDate.Location()).Format("2006-01-02")
		}
	}
	if effectiveStartDateEnd == "" && effectiveStartDateStart != "" {
		if startDate, err := time.Parse("2006-01-02", effectiveStartDateStart); err == nil {
			lastDay := time.Date(startDate.Year(), startDate.Month()+1, 0, 23, 59, 59, 0, startDate.Location())
			effectiveStartDateEnd = lastDay.Format("2006-01-02")
		}
	}
	if effectiveStartDateStart != "" {
		if startDate, err := time.Parse("2006-01-02", effectiveStartDateStart); err == nil {
			where = append(where, "ls.active_from >= ?")
			args = append(args, startDate)
		}
	}
	if effectiveStartDateEnd != "" {
		if endDate, err := time.Parse("2006-01-02", effectiveStartDateEnd); err == nil {
			where = append(where, "ls.active_from <= ?")
			args = append(args, endDate.Add(23*time.Hour+59*time.Minute+59*time.Second))
		}
	}
	if params.StatusLangganan != "" {
		condition, conditionArgs := outletSubscriptionStatusFilterCondition(params.StatusLangganan, referenceMonth, isSpecificDate, params.DueDateReference)
		if condition != "" {
			where = append(where, condition)
			args = append(args, conditionArgs...)
		}
	}
	if params.StatusJatuhTempo != "" {
		condition, conditionArgs := outletSubscriptionStatusFilterCondition(params.StatusJatuhTempo, referenceMonth, isSpecificDate, params.DueDateReference)
		if condition != "" {
			where = append(where, condition)
			args = append(args, conditionArgs...)
		}
	}
	return strings.Join(where, " AND "), args
}

func resolveRefDate(referenceMonth time.Time, isSpecificDate bool) time.Time {
	if isSpecificDate {
		return dateOnly(referenceMonth)
	}
	now := dateOnly(time.Now().UTC())
	if sameMonth(now, referenceMonth) {
		return now
	}
	return dateOnly(monthEnd(referenceMonth))
}

func outletSubscriptionStatusFilterCondition(status string, referenceMonth time.Time, isSpecificDate bool, dueDateReference string) (string, []any) {
	if strings.Contains(status, ",") {
		parts := strings.Split(status, ",")
		var conds []string
		var allArgs []any
		for _, part := range parts {
			part = strings.TrimSpace(part)
			c, args := singleOutletSubscriptionStatusFilterCondition(part, referenceMonth, isSpecificDate, dueDateReference)
			if c != "" {
				conds = append(conds, c)
				allArgs = append(allArgs, args...)
			}
		}
		if len(conds) == 0 {
			return "", nil
		}
		if len(conds) == 1 {
			return conds[0], allArgs
		}
		return "(" + strings.Join(conds, " OR ") + ")", allArgs
	}
	return singleOutletSubscriptionStatusFilterCondition(status, referenceMonth, isSpecificDate, dueDateReference)
}

func singleOutletSubscriptionStatusFilterCondition(status string, referenceMonth time.Time, isSpecificDate bool, dueDateReference string) (string, []any) {
	refDate := resolveRefDate(referenceMonth, isSpecificDate)
	if dueDateReference != "" {
		if parsedRef, err := time.Parse("2006-01-02", dueDateReference); err == nil {
			refDate = dateOnly(parsedRef)
		}
	}

	switch status {
	case OutletSubscriptionStatusTrial:
		// TRIAL = belum pernah berlangganan DAN masih dalam masa trial (created > trialCutoff14)
		trialCutoff14 := refDate.AddDate(0, 0, -TrialDurationDays)
		return `(
			ls.active_until IS NULL
			AND COALESCE(o.created_at, ot.created_at) > ?
			AND NOT EXISTS(
				SELECT 1 FROM subscriptions s2
				JOIN outlets ot5 ON ot5.id = s2.outlet_id
				WHERE ot5.owner_id = ot.owner_id AND s2.deleted_at IS NULL
			)
		)`, []any{trialCutoff14}
	case OutletSubscriptionDueStatusNoPackage:
		// NO_PACKAGE sebagai Kategori Nasabah: belum pernah berlangganan DAN trial sudah habis
		trialCutoff14 := refDate.AddDate(0, 0, -TrialDurationDays)
		return `(
			ls.active_until IS NULL
			AND COALESCE(o.created_at, ot.created_at) <= ?
			AND NOT EXISTS(
				SELECT 1 FROM subscriptions s2
				JOIN outlets ot5 ON ot5.id = s2.outlet_id
				WHERE ot5.owner_id = ot.owner_id AND s2.deleted_at IS NULL
			)
		)`, []any{trialCutoff14}
	case OutletSubscriptionStatusInactive:
		// UNSUBSCRIBE: pernah berlangganan tapi sekarang tidak aktif.
		// Dua kasus: (1) langganan sudah expired, (2) tidak ada langganan tapi owner pernah punya sub
		return `(
			ls.active_until < ? OR (
				(ls.active_until IS NULL OR ls.active_from IS NULL) AND EXISTS(
					SELECT 1 FROM subscriptions s2
					JOIN outlets ot5 ON ot5.id = s2.outlet_id
					WHERE ot5.owner_id = ot.owner_id AND s2.deleted_at IS NULL
				)
			)
		)`, []any{refDate}
	case OutletSubscriptionStatusNew:
		return `(
			ls.active_until >= ? 
			AND YEAR(ls.active_from) = YEAR(?) AND MONTH(ls.active_from) = MONTH(?)
			AND (SELECT COUNT(*) FROM subscriptions s3 WHERE s3.outlet_id = ot.id AND s3.deleted_at IS NULL) = 1
		)`, []any{refDate, refDate, refDate}
	case OutletSubscriptionStatusSubscribed:
		return `(
			ls.active_until >= ? 
			AND (
				(YEAR(ls.active_from) = YEAR(?) AND MONTH(ls.active_from) = MONTH(?) AND (SELECT COUNT(*) FROM subscriptions s3 WHERE s3.outlet_id = ot.id AND s3.deleted_at IS NULL) > 1)
				OR NOT (YEAR(ls.active_from) = YEAR(?) AND MONTH(ls.active_from) = MONTH(?))
			)
		)`, []any{refDate, refDate, refDate, refDate, refDate}
	case OutletSubscriptionStatusWillBeDue:
		// Cutoff dihitung dari refDate agar Acuan Jatuh Tempo berpengaruh
		// AKAN JATUH TEMPO TRIAL: owner dibuat antara (refDate-14) eksklusif dan (refDate-7) inklusif
		// yaitu sisa 1-7 hari trial
		trialCutoff14 := refDate.AddDate(0, 0, -TrialDurationDays)
		trialCutoff7 := refDate.AddDate(0, 0, -7)
		return `(
			(ls.active_until IS NOT NULL AND ls.active_until > ? AND ls.active_until <= ?)
			OR (
				ls.active_until IS NULL
				AND COALESCE(o.created_at, ot.created_at) > ? 
				AND COALESCE(o.created_at, ot.created_at) <= ?
				AND NOT EXISTS(
					SELECT 1 FROM subscriptions s2
					JOIN outlets ot5 ON ot5.id = s2.outlet_id
					WHERE ot5.owner_id = ot.owner_id AND s2.deleted_at IS NULL
				)
			)
		)`, []any{refDate, refDate.AddDate(0, 0, 14), trialCutoff14, trialCutoff7}
	case OutletSubscriptionStatusDue:
		trialCutoff14 := refDate.AddDate(0, 0, -TrialDurationDays)
		return `(
			(ls.active_until IS NOT NULL AND DATE(ls.active_until) = DATE(?))
			OR (
				ls.active_until IS NULL
				AND DATE(COALESCE(o.created_at, ot.created_at)) = DATE(?)
				AND NOT EXISTS(
					SELECT 1 FROM subscriptions s2
					JOIN outlets ot5 ON ot5.id = s2.outlet_id
					WHERE ot5.owner_id = ot.owner_id AND s2.deleted_at IS NULL
				)
			)
		)`, []any{refDate, trialCutoff14}
	case OutletSubscriptionStatusPassedDue:
		trialCutoff14 := refDate.AddDate(0, 0, -TrialDurationDays)
		return `(
			(ls.active_until IS NOT NULL AND ls.active_until < ?)
			OR (
				ls.active_until IS NULL
				AND DATE(COALESCE(o.created_at, ot.created_at)) < DATE(?)
				AND NOT EXISTS(
					SELECT 1 FROM subscriptions s2
					JOIN outlets ot5 ON ot5.id = s2.outlet_id
					WHERE ot5.owner_id = ot.owner_id AND s2.deleted_at IS NULL
				)
			)
			OR (
				(ls.active_until IS NULL OR ls.active_from IS NULL) AND EXISTS(
					SELECT 1 FROM subscriptions s2
					JOIN outlets ot5 ON ot5.id = s2.outlet_id
					WHERE ot5.owner_id = ot.owner_id AND s2.deleted_at IS NULL
				)
			)
		)`, []any{refDate, trialCutoff14}
	case "BELUM_JATUH_TEMPO":
		// Cutoff dihitung dari refDate agar Acuan Jatuh Tempo berpengaruh
		trialCutoff7 := refDate.AddDate(0, 0, -7)
		return `(
			(ls.active_until IS NOT NULL AND ls.active_until > ?)
			OR (
				ls.active_until IS NULL
				AND COALESCE(o.created_at, ot.created_at) >= ?
				AND NOT EXISTS(
					SELECT 1 FROM subscriptions s2
					JOIN outlets ot5 ON ot5.id = s2.outlet_id
					WHERE ot5.owner_id = ot.owner_id AND s2.deleted_at IS NULL
				)
			)
		)`, []any{refDate.AddDate(0, 0, 14), trialCutoff7}
	default:
		return "", nil
	}
}

func outletSubscriptionStatusOrderBy(sort string) (string, error) {
	return orderBy(sort, map[string]string{
		"created_at":              "ot.created_at",
		"updated_at":              "ot.updated_at",
		"code":                    "ot.code",
		"name":                    "ot.name",
		"city":                    "ot.city",
		"province":                "ot.province",
		"subscription_start_date": "ls.active_from",
		"subscription_end_date":   "ls.active_until",
	}, "ot.created_at DESC, ot.id DESC")
}

func normalizeOutletSubscriptionStatusParams(params OutletSubscriptionStatusParams) OutletSubscriptionStatusParams {
	if params.All {
		params.Page = 1
		params.Limit = 0
	} else {
		if params.Page < 1 {
			params.Page = 1
		}
		if params.Limit < 1 {
			params.Limit = 10
		}
		if params.Limit > 100 {
			params.Limit = 100
		}
	}
	params.Query = strings.TrimSpace(params.Query)
	params.Code = strings.TrimSpace(params.Code)
	params.Name = strings.TrimSpace(params.Name)
	params.Phone = strings.TrimSpace(params.Phone)
	params.BrandName = strings.TrimSpace(params.BrandName)
	params.Province = strings.TrimSpace(params.Province)
	params.City = strings.TrimSpace(params.City)
	params.Month = strings.TrimSpace(params.Month)
	params.DueDate = strings.TrimSpace(params.DueDate)
	params.SubscriptionStatus = normalizeOutletSubscriptionStatusCode(params.SubscriptionStatus)
	params.StatusLangganan = normalizeOutletSubscriptionStatusCode(params.StatusLangganan)
	if params.StatusLangganan == "" {
		params.StatusLangganan = params.SubscriptionStatus
	}
	params.StatusJatuhTempo = normalizeOutletSubscriptionStatusCode(params.StatusJatuhTempo)
	params.DueDateReference = strings.TrimSpace(params.DueDateReference)
	params.DueDateStart = strings.TrimSpace(params.DueDateStart)
	params.DueDateEnd = strings.TrimSpace(params.DueDateEnd)
	params.Sort = strings.TrimSpace(params.Sort)
	return params
}

func normalizeOutletSubscriptionStatusCode(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	parts := strings.Split(value, ",")
	var normalized []string
	for _, p := range parts {
		p = strings.ToUpper(strings.TrimSpace(p))
		switch p {
		case OutletSubscriptionStatusTrial, OutletSubscriptionStatusInactive, OutletSubscriptionStatusDue, OutletSubscriptionStatusWillBeDue, OutletSubscriptionStatusPassedDue, OutletSubscriptionStatusSubscribed, OutletSubscriptionStatusNew, "BELUM_JATUH_TEMPO", OutletSubscriptionDueStatusNoPackage:
			normalized = append(normalized, p)
		}
	}
	return strings.Join(normalized, ",")
}

func resolveReferenceMonth(value string) (time.Time, bool, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		now := time.Now().UTC()
		return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC), false, nil
	}
	if len(value) == 10 {
		refDate, err := time.Parse("2006-01-02", value)
		if err != nil {
			return time.Time{}, false, err
		}
		return refDate, true, nil
	}
	referenceMonth, err := time.Parse("2006-01", value)
	if err != nil {
		return time.Time{}, false, err
	}
	return time.Date(referenceMonth.Year(), referenceMonth.Month(), 1, 0, 0, 0, 0, time.UTC), false, nil
}

func monthStart(referenceMonth time.Time) time.Time {
	return time.Date(referenceMonth.Year(), referenceMonth.Month(), 1, 0, 0, 0, 0, time.UTC)
}

func monthEnd(referenceMonth time.Time) time.Time {
	start := monthStart(referenceMonth)
	return start.AddDate(0, 1, -1)
}

func computeDueStatus(statusCode string) (string, string) {
	switch statusCode {
	case OutletSubscriptionStatusTrial:
		return OutletSubscriptionDueStatusNoPackage, "NO PACKAGE"
	case OutletSubscriptionStatusWillBeDue:
		return OutletSubscriptionStatusWillBeDue, "AKAN JATUH TEMPO"
	case OutletSubscriptionStatusDue:
		return OutletSubscriptionStatusDue, "JATUH TEMPO"
	case OutletSubscriptionStatusPassedDue:
		return OutletSubscriptionStatusPassedDue, "TELAH JATUH TEMPO"
	default:
		return "BELUM_JATUH_TEMPO", "BELUM JATUH TEMPO"
	}
}

func buildOutletSubscriptionStatusResponse(snapshot OutletSubscriptionSnapshot, referenceMonth time.Time, isSpecificDate bool, dueDateReference string) OutletSubscriptionStatusResponse {
	statusCode, statusLabel, dueCode, dueLabel, remainingDays, remainingDaysDisplay, lastSubscriptionEndDisplay := classifyOutletSubscription(snapshot, referenceMonth, isSpecificDate, dueDateReference)
	creationStatus := "EXISTING"
	if snapshot.CreatedAt.Year() == referenceMonth.Year() && snapshot.CreatedAt.Month() == referenceMonth.Month() {
		creationStatus = "NEW"
	}

	response := OutletSubscriptionStatusResponse{
		OutletID:                   snapshot.OutletID,
		OutletCode:                 snapshot.OutletCode,
		OutletName:                 snapshot.OutletName,
		OutletPhone:                snapshot.OutletPhone.String,
		OutletProvince:             snapshot.OutletProvince.String,
		OutletCity:                 snapshot.OutletCity.String,
		OutletAddress:              snapshot.OutletAddress.String,
		Owner:                      ownerBriefFromSnapshot(snapshot),
		SubscriptionStatusCode:     statusCode,
		SubscriptionStatusLabel:    statusLabel,
		DueStatusCode:              dueCode,
		DueStatusLabel:             dueLabel,
		RemainingDays:              remainingDays,
		RemainingDaysDisplay:       remainingDaysDisplay,
		LastSubscriptionEndDisplay: lastSubscriptionEndDisplay,
		PackagePlan:                packagePlanBriefFromSnapshot(snapshot),
		CreationStatus:             creationStatus,
		CreatedAt:                  snapshot.CreatedAt,
		UpdatedAt:                  snapshot.UpdatedAt,
	}
	if snapshot.SubscriptionStart.Valid {
		response.SubscriptionStartDate = snapshot.SubscriptionStart.Time.Format("2006-01-02")
	}
	if snapshot.SubscriptionEnd.Valid {
		formatted := snapshot.SubscriptionEnd.Time.Format("2006-01-02")
		response.SubscriptionEndDate = formatted
		response.LastSubscriptionEnd = formatted
	}
	return response
}

func classifyOutletSubscription(snapshot OutletSubscriptionSnapshot, referenceMonth time.Time, isSpecificDate bool, dueDateReference string) (string, string, string, string, *int64, string, string) {
	// Tentukan refDate terlebih dahulu, agar trial classification juga menggunakan Acuan Jatuh Tempo
	refDate := resolveRefDate(referenceMonth, isSpecificDate)
	if dueDateReference != "" {
		if parsedRef, err := time.Parse("2006-01-02", dueDateReference); err == nil {
			refDate = dateOnly(parsedRef)
		}
	}

	if !snapshot.SubscriptionEnd.Valid || !snapshot.SubscriptionStart.Valid {
		ownerCreated := dateOnly(snapshot.OwnerCreatedAt)
		trialEnd := ownerCreated.AddDate(0, 0, TrialDurationDays)
		diffTrialDays := int64(trialEnd.Sub(refDate).Hours() / 24)
		trialEndDisplay := trialEnd.Format("2006-01-02")

		if !snapshot.OwnerHasAnySubscription {
			// Belum pernah berlangganan:
			// - Hari 1-13 (sisa > 7): TRIAL / BELUM JATUH TEMPO
			// - Hari 8-13 (sisa 1-7): TRIAL / AKAN JATUH TEMPO
			// - Hari 14+ (sisa <= 0): NO_PACKAGE / JATUH TEMPO atau TELAH JATUH TEMPO
			if diffTrialDays > 7 {
				remainingDisplay := fmt.Sprintf("%d hari", diffTrialDays)
				return OutletSubscriptionStatusTrial, "TRIAL", "BELUM_JATUH_TEMPO", "BELUM JATUH TEMPO", &diffTrialDays, remainingDisplay, trialEndDisplay
			} else if diffTrialDays > 0 && diffTrialDays <= 7 {
				remainingDisplay := fmt.Sprintf("%d hari", diffTrialDays)
				return OutletSubscriptionStatusTrial, "TRIAL", OutletSubscriptionStatusWillBeDue, "AKAN JATUH TEMPO", &diffTrialDays, remainingDisplay, trialEndDisplay
			} else if diffTrialDays == 0 {
				// Hari jatuh tempo trial → NO PACKAGE / JATUH TEMPO
				zero := int64(0)
				return OutletSubscriptionDueStatusNoPackage, "NO PACKAGE", OutletSubscriptionStatusDue, "JATUH TEMPO", &zero, "0 hari", trialEndDisplay
			}
			// diffTrialDays < 0: trial sudah lewat → NO PACKAGE / TELAH JATUH TEMPO
			return OutletSubscriptionDueStatusNoPackage, "NO PACKAGE", OutletSubscriptionStatusPassedDue, "TELAH JATUH TEMPO", &diffTrialDays, fmt.Sprintf("%d hari", diffTrialDays), trialEndDisplay
		}
		// Pernah berlangganan di outlet lain, sehingga outlet baru tidak dapat trial.
		// Status yang tepat adalah NO PACKAGE (karena outlet ini sendiri belum pernah dibelikan paket).
		return OutletSubscriptionDueStatusNoPackage, "NO PACKAGE", OutletSubscriptionStatusPassedDue, "TELAH JATUH TEMPO", nil, NeverSubscribedDisplay, NeverSubscribedDisplay
	}

	start := dateOnly(snapshot.SubscriptionStart.Time)
	end := dateOnly(snapshot.SubscriptionEnd.Time)
	lastSubscriptionEndDisplay := end.Format("2006-01-02")
	diffDays := int64(end.Sub(refDate).Hours() / 24)

	var remaining *int64 = &diffDays
	var remainingDisplay string = fmt.Sprintf("%d hari", diffDays)

	var dueCode, dueLabel string
	if diffDays < 0 {
		dueCode = OutletSubscriptionStatusPassedDue
		dueLabel = "TELAH JATUH TEMPO"
	} else if diffDays == 0 {
		// 0 hari = JATUH TEMPO (jatuh tempo hari ini)
		dueCode = OutletSubscriptionStatusDue
		dueLabel = "JATUH TEMPO"
	} else if diffDays > 0 && diffDays <= 14 {
		// 1-14 hari = AKAN JATUH TEMPO
		dueCode = OutletSubscriptionStatusWillBeDue
		dueLabel = "AKAN JATUH TEMPO"
	} else if diffDays > 14 {
		dueCode = "BELUM_JATUH_TEMPO"
		dueLabel = "BELUM JATUH TEMPO"
	}

	if diffDays < 0 {
		return OutletSubscriptionStatusInactive, "UNSUBSCRIBE", dueCode, dueLabel, remaining, remainingDisplay, lastSubscriptionEndDisplay
	}

	isStartSameMonth := start.Year() == refDate.Year() && start.Month() == refDate.Month()
	var statusCode, statusLabel string

	if isStartSameMonth {
		if snapshot.OutletSubscriptionCount <= 1 {
			statusCode = OutletSubscriptionStatusNew
			statusLabel = "NEW"
		} else {
			statusCode = OutletSubscriptionStatusSubscribed
			statusLabel = "SUBSCRIBE"
		}
	} else {
		statusCode = OutletSubscriptionStatusSubscribed
		statusLabel = "SUBSCRIBE"
	}

	return statusCode, statusLabel, dueCode, dueLabel, remaining, remainingDisplay, lastSubscriptionEndDisplay
}

func ownerBriefFromSnapshot(snapshot OutletSubscriptionSnapshot) OwnerBriefResponse {
	if !snapshot.OwnerID.Valid {
		return OwnerBriefResponse{Message: "Data owner tidak tersedia"}
	}
	id := snapshot.OwnerID.Int64
	return OwnerBriefResponse{
		ID:        &id,
		Code:      snapshot.OwnerCode.String,
		Name:      snapshot.OwnerName.String,
		Phone:     snapshot.OwnerPhone.String,
		Email:     snapshot.OwnerEmail.String,
		BrandName: snapshot.OwnerBrandName.String,
	}
}

func packagePlanBriefFromSnapshot(snapshot OutletSubscriptionSnapshot) PackagePlanBriefResponse {
	return PackagePlanBriefResponse{
		PackageID:    nullableInt64Ptr(snapshot.PackageID),
		PackageCode:  snapshot.PackageCode.String,
		PackageName:  snapshot.PackageName.String,
		PlanID:       nullableInt64Ptr(snapshot.PlanID),
		PlanCode:     snapshot.PlanCode.String,
		PlanName:     snapshot.PlanName.String,
		TenureMonths: nullableInt64Ptr(snapshot.TenureMonths),
	}
}

func sameMonth(value time.Time, referenceMonth time.Time) bool {
	return value.Year() == referenceMonth.Year() && value.Month() == referenceMonth.Month()
}

func dateOnly(value time.Time) time.Time {
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC)
}

func nullableInt64Ptr(value sql.NullInt64) *int64 {
	if !value.Valid {
		return nil
	}
	out := value.Int64
	return &out
}

type SubscriptionDetail struct {
	ID           int64
	OutletID     int64
	PackageID    sql.NullInt64
	PackageCode  sql.NullString
	PackageName  sql.NullString
	PlanID       sql.NullInt64
	PlanCode     sql.NullString
	PlanName     sql.NullString
	TenureMonths sql.NullInt64
	ActiveFrom   sql.NullTime
	ActiveUntil  sql.NullTime
}

func (r *Repository) GetSubscriptionsForOutlets(ctx context.Context, outletIDs []int64) (map[int64][]SubscriptionDetail, error) {
	if len(outletIDs) == 0 {
		return make(map[int64][]SubscriptionDetail), nil
	}

	args := make([]any, len(outletIDs))
	placeholders := make([]string, len(outletIDs))
	for i, id := range outletIDs {
		args[i] = id
		placeholders[i] = "?"
	}

	query := `
		SELECT 
			s.id, s.outlet_id, s.package_id, s.plan_id, s.active_from, s.active_until, 
			p.code AS package_code, p.name AS package_name,
			pl.code AS plan_code, pl.name AS plan_name, pl.tenure_months
		FROM subscriptions s
		LEFT JOIN subscription_packages p ON p.id = s.package_id
		LEFT JOIN subscription_plans pl ON pl.id = s.plan_id
		WHERE s.deleted_at IS NULL AND s.outlet_id IN (` + strings.Join(placeholders, ",") + `)
	`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[int64][]SubscriptionDetail)
	for rows.Next() {
		var d SubscriptionDetail
		err := rows.Scan(
			&d.ID, &d.OutletID, &d.PackageID, &d.PlanID, &d.ActiveFrom, &d.ActiveUntil,
			&d.PackageCode, &d.PackageName, &d.PlanCode, &d.PlanName, &d.TenureMonths,
		)
		if err != nil {
			return nil, err
		}
		result[d.OutletID] = append(result[d.OutletID], d)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}
