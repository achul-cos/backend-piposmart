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
	OutletSubscriptionStatusInactive     = "TIDAK_AKTIF"
	OutletSubscriptionStatusDueThisMonth = "JATUH_TEMPO"
	OutletSubscriptionStatusWillBeDue    = "AKAN_JATUH_TEMPO"
	OutletSubscriptionStatusDue          = "JATUH_TEMPO"
	OutletSubscriptionStatusPassedDue    = "TELAH_JATUH_TEMPO"
	OutletSubscriptionStatusSubscribed   = "BERLANGGANAN"
	OutletSubscriptionStatusNew          = "NEW"
	OutletSubscriptionStatusRenewal      = "RENEWAL"
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
	StatusLangganan    string
	StatusJatuhTempo   string
	Month              string
	DueDate            string
	DueDateReference   string
	DueDateStart       string
	DueDateEnd         string
	All                bool
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
		All:              false,
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
			ot.created_at, ot.updated_at
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
		where = append(where, "o.brand_name LIKE ?")
		args = append(args, like(params.BrandName))
	}
	if params.Province != "" {
		where = append(where, "ot.province LIKE ?")
		args = append(args, like(params.Province))
	}
	if params.City != "" {
		where = append(where, "ot.city LIKE ?")
		args = append(args, like(params.City))
	}
	if params.DueDate != "" {
		if dueDate, err := time.Parse("2006-01-02", params.DueDate); err == nil {
			where = append(where, "DATE(ls.active_until) = DATE(?)")
			args = append(args, dueDate)
		}
	}
	if params.DueDateStart != "" {
		if startDate, err := time.Parse("2006-01-02", params.DueDateStart); err == nil {
			where = append(where, "ls.active_until >= ?")
			args = append(args, startDate)
		}
	}
	if params.DueDateEnd != "" {
		if endDate, err := time.Parse("2006-01-02", params.DueDateEnd); err == nil {
			where = append(where, "ls.active_until <= ?")
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
	refDate := resolveRefDate(referenceMonth, isSpecificDate)
	if dueDateReference != "" {
		if parsedRef, err := time.Parse("2006-01-02", dueDateReference); err == nil {
			refDate = dateOnly(parsedRef)
		}
	}

	switch status {
	case OutletSubscriptionStatusTrial:
		trialCutoff := time.Now().UTC().AddDate(0, 0, -TrialDurationDays)
		return `(
			ls.active_until IS NULL
			AND o.created_at >= ?
			AND NOT EXISTS(
				SELECT 1 FROM subscriptions s2
				JOIN outlets ot5 ON ot5.id = s2.outlet_id
				WHERE ot5.owner_id = ot.owner_id AND s2.deleted_at IS NULL
			)
		)`, []any{trialCutoff}
	case OutletSubscriptionStatusInactive:
		// Sisa hari < 0 (ls.active_until < refDate) ATAU belum pernah berlangganan (bukan Trial).
		trialCutoff := time.Now().UTC().AddDate(0, 0, -TrialDurationDays)
		return `(
			ls.active_until < ? OR (
				(ls.active_until IS NULL OR ls.active_from IS NULL) AND (
					o.created_at < ? OR EXISTS(
						SELECT 1 FROM subscriptions s2
						JOIN outlets ot5 ON ot5.id = s2.outlet_id
						WHERE ot5.owner_id = ot.owner_id AND s2.deleted_at IS NULL
					)
				)
			)
		)`, []any{refDate, trialCutoff}
	case OutletSubscriptionStatusNew:
		return `(
			ls.active_until >= ? 
			AND YEAR(ls.active_from) = YEAR(?) AND MONTH(ls.active_from) = MONTH(?)
			AND (SELECT COUNT(*) FROM subscriptions s3 WHERE s3.outlet_id = ot.id AND s3.deleted_at IS NULL) = 1
		)`, []any{refDate, refDate, refDate}
	case OutletSubscriptionStatusRenewal:
		return `(
			ls.active_until >= ? 
			AND YEAR(ls.active_from) = YEAR(?) AND MONTH(ls.active_from) = MONTH(?)
			AND (SELECT COUNT(*) FROM subscriptions s3 WHERE s3.outlet_id = ot.id AND s3.deleted_at IS NULL) > 1
		)`, []any{refDate, refDate, refDate}
	case OutletSubscriptionStatusSubscribed:
		return `(
			ls.active_until >= ? 
			AND NOT (YEAR(ls.active_from) = YEAR(?) AND MONTH(ls.active_from) = MONTH(?))
		)`, []any{refDate, refDate, refDate}
	case OutletSubscriptionStatusWillBeDue:
		return `(ls.active_until IS NOT NULL AND ls.active_until > ? AND ls.active_until <= ?)`, []any{refDate, refDate.AddDate(0, 0, 7)}
	case OutletSubscriptionStatusDue:
		return `(ls.active_until IS NOT NULL AND DATE(ls.active_until) = DATE(?))`, []any{refDate}
	case OutletSubscriptionStatusPassedDue:
		return `(ls.active_until IS NOT NULL AND ls.active_until < ?)`, []any{refDate}
	case "BELUM_JATUH_TEMPO":
		return `(ls.active_until IS NOT NULL AND ls.active_until > ?)`, []any{refDate.AddDate(0, 0, 7)}
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
	value = strings.ToUpper(strings.TrimSpace(value))
	switch value {
	case OutletSubscriptionStatusTrial, OutletSubscriptionStatusInactive, OutletSubscriptionStatusDue, OutletSubscriptionStatusWillBeDue, OutletSubscriptionStatusPassedDue, OutletSubscriptionStatusSubscribed, OutletSubscriptionStatusNew, OutletSubscriptionStatusRenewal, "BELUM_JATUH_TEMPO":
		return value
	default:
		return ""
	}
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
		return OutletSubscriptionStatusTrial, "TRIAL"
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
	if !snapshot.SubscriptionEnd.Valid || !snapshot.SubscriptionStart.Valid {
		now := time.Now().UTC()
		ownerAge := now.Sub(snapshot.OwnerCreatedAt)
		if ownerAge <= time.Duration(TrialDurationDays)*24*time.Hour && !snapshot.OwnerHasAnySubscription {
			trialEnd := snapshot.OwnerCreatedAt.AddDate(0, 0, TrialDurationDays)
			diffTrialDays := int64(trialEnd.Sub(now).Hours() / 24)
			if diffTrialDays < 0 {
				diffTrialDays = 0
			}
			trialEndDisplay := trialEnd.Format("2006-01-02")
			remainingDisplay := fmt.Sprintf("%d hari", diffTrialDays)
			return OutletSubscriptionStatusTrial, "TRIAL", OutletSubscriptionStatusTrial, "TRIAL", &diffTrialDays, remainingDisplay, trialEndDisplay
		}
		return OutletSubscriptionStatusInactive, "TIDAK AKTIF", "", "", nil, NeverSubscribedDisplay, NeverSubscribedDisplay
	}

	start := dateOnly(snapshot.SubscriptionStart.Time)
	end := dateOnly(snapshot.SubscriptionEnd.Time)
	refDate := resolveRefDate(referenceMonth, isSpecificDate)
	if dueDateReference != "" {
		if parsedRef, err := time.Parse("2006-01-02", dueDateReference); err == nil {
			refDate = dateOnly(parsedRef)
		}
	}
	lastSubscriptionEndDisplay := end.Format("2006-01-02")
	diffDays := int64(end.Sub(refDate).Hours() / 24)

	var remaining *int64 = &diffDays
	var remainingDisplay string = fmt.Sprintf("%d hari", diffDays)

	var dueCode, dueLabel string
	if diffDays < 0 {
		dueCode = OutletSubscriptionStatusPassedDue
		dueLabel = "TELAH JATUH TEMPO"
	} else if diffDays == 0 {
		dueCode = OutletSubscriptionStatusDue
		dueLabel = "JATUH TEMPO"
	} else if diffDays > 0 && diffDays <= 7 {
		dueCode = OutletSubscriptionStatusWillBeDue
		dueLabel = "AKAN JATUH TEMPO"
	} else if diffDays > 7 {
		dueCode = "BELUM_JATUH_TEMPO"
		dueLabel = "BELUM JATUH TEMPO"
	}

	if diffDays < 0 {
		return OutletSubscriptionStatusInactive, "TIDAK AKTIF", dueCode, dueLabel, remaining, remainingDisplay, lastSubscriptionEndDisplay
	}

	isStartSameMonth := start.Year() == refDate.Year() && start.Month() == refDate.Month()
	var statusCode, statusLabel string

	if isStartSameMonth {
		if snapshot.OutletSubscriptionCount > 1 {
			statusCode = OutletSubscriptionStatusRenewal
			statusLabel = "RENEWAL"
		} else {
			statusCode = OutletSubscriptionStatusNew
			statusLabel = "NEW"
		}
	} else {
		statusCode = OutletSubscriptionStatusSubscribed
		statusLabel = "BERLANGGANAN"
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
