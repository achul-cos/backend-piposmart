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
	OutletSubscriptionStatusExpired      = "EXPIRED"
	OutletSubscriptionStatusDueThisMonth = "JATUH_TEMPO"
	OutletSubscriptionStatusSubscribed   = "BERLANGGANAN"
	OutletSubscriptionStatusNew          = "NEW"
	OutletSubscriptionLabelOneMonth      = "BERLANGGANAN 1 BULAN"
	NeverSubscribedDisplay               = "TIDAK PERNAH"
)

type OutletSubscriptionSnapshot struct {
	OutletID          int64
	OutletCode        string
	OutletName        string
	OutletPhone       sql.NullString
	OutletProvince    sql.NullString
	OutletCity        sql.NullString
	OutletAddress     sql.NullString
	OutletStatus      string
	OwnerID           sql.NullInt64
	OwnerCode         sql.NullString
	OwnerName         sql.NullString
	OwnerPhone        sql.NullString
	OwnerEmail        sql.NullString
	OwnerBrandName    sql.NullString
	PackageID         sql.NullInt64
	PackageCode       sql.NullString
	PackageName       sql.NullString
	PlanID            sql.NullInt64
	PlanCode          sql.NullString
	PlanName          sql.NullString
	TenureMonths      sql.NullInt64
	SubscriptionStart sql.NullTime
	SubscriptionEnd   sql.NullTime
	CreatedAt         time.Time
	UpdatedAt         time.Time
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
	Month              string
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
	referenceMonth, err := resolveReferenceMonth(params.Month)
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "month harus format YYYY-MM", nil)
		return
	}
	response, err := h.service.ListOutletSubscriptionStatuses(c.Request.Context(), currentActor(c), params, referenceMonth)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, response)
}

func (h *Handler) listAllOutletSubscriptionStatuses(c *gin.Context) {
	params := outletSubscriptionStatusParams(c)
	params.All = true
	referenceMonth, err := resolveReferenceMonth(params.Month)
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "month harus format YYYY-MM", nil)
		return
	}
	response, err := h.service.ListOutletSubscriptionStatuses(c.Request.Context(), currentActor(c), params, referenceMonth)
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
		Month:              c.Query("month"),
		All:                false,
		Page:               page,
		Limit:              limit,
		Sort:               c.Query("sort"),
	}
	if ownerID, err := strconv.ParseInt(c.Query("owner_id"), 10, 64); err == nil && ownerID > 0 {
		params.OwnerID = &ownerID
	}
	return params
}

func (s *Service) ListOutletSubscriptionStatuses(ctx context.Context, actor Actor, params OutletSubscriptionStatusParams, referenceMonth time.Time) (OutletSubscriptionStatusListResponse, error) {
	params = normalizeOutletSubscriptionStatusParams(params)
	if params.Phone != "" {
		phone, err := NormalizePhone(params.Phone)
		if err == nil {
			params.Phone = phone
		}
	}
	snapshots, total, err := s.repo.ListOutletSubscriptionSnapshots(ctx, actor, params, referenceMonth)
	if err != nil {
		return OutletSubscriptionStatusListResponse{}, err
	}
	items := make([]OutletSubscriptionStatusResponse, 0, len(snapshots))
	for _, snapshot := range snapshots {
		items = append(items, buildOutletSubscriptionStatusResponse(snapshot, referenceMonth))
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

func (r *Repository) ListOutletSubscriptionSnapshots(ctx context.Context, actor Actor, params OutletSubscriptionStatusParams, referenceMonth time.Time) ([]OutletSubscriptionSnapshot, int64, error) {
	where, args := outletSubscriptionStatusWhere(actor, params, referenceMonth)
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

func outletSubscriptionStatusWhere(actor Actor, params OutletSubscriptionStatusParams, referenceMonth time.Time) (string, []any) {
	where := []string{"ot.deleted_at IS NULL"}
	args := []any{}
	visibility, visibilityArgs := ownerVisibilityWhereByColumn(actor, "ot.owner_id")
	where = append(where, visibility)
	args = append(args, visibilityArgs...)
	if params.Query != "" {
		pattern := like(params.Query)
		where = append(where, "(ot.code LIKE ? OR ot.name LIKE ? OR ot.phone LIKE ? OR ot.city LIKE ? OR ot.province LIKE ? OR o.code LIKE ? OR o.name LIKE ? OR o.brand_name LIKE ?)")
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
	if params.SubscriptionStatus != "" {
		condition, conditionArgs := outletSubscriptionStatusFilterCondition(params.SubscriptionStatus, referenceMonth)
		if condition != "" {
			where = append(where, condition)
			args = append(args, conditionArgs...)
		}
	}
	return strings.Join(where, " AND "), args
}

func outletSubscriptionStatusFilterCondition(status string, referenceMonth time.Time) (string, []any) {
	monthStartValue := monthStart(referenceMonth)
	monthEndValue := monthEnd(referenceMonth)
	threshold := monthEndValue.AddDate(0, 0, -60)
	recentStart := monthEndValue.AddDate(0, 0, -30)
	year := referenceMonth.Year()
	month := int(referenceMonth.Month())
	switch status {
	case OutletSubscriptionStatusNotSubscribe:
		return `(ls.active_until IS NULL OR ls.active_from IS NULL OR ls.active_until < ?)`, []any{threshold}
	case OutletSubscriptionStatusExpired:
		return `(ls.active_until >= ? AND ls.active_until < ?)`, []any{threshold, monthStartValue}
	case OutletSubscriptionStatusNew:
		return `(ls.active_from IS NOT NULL AND ls.active_until IS NOT NULL AND ls.active_until >= ? AND ls.active_from >= ? AND NOT (COALESCE(ls.tenure_months, 0) = 1 AND YEAR(ls.active_until) = ? AND MONTH(ls.active_until) = ?))`, []any{monthStartValue, recentStart, year, month}
	case OutletSubscriptionStatusDueThisMonth:
		return `(ls.active_until IS NOT NULL AND YEAR(ls.active_until) = ? AND MONTH(ls.active_until) = ? AND COALESCE(ls.tenure_months, 0) <> 1)`, []any{year, month}
	case OutletSubscriptionStatusSubscribed:
		return `((ls.active_until IS NOT NULL AND YEAR(ls.active_until) = ? AND MONTH(ls.active_until) = ? AND COALESCE(ls.tenure_months, 0) = 1) OR (ls.active_until IS NOT NULL AND ls.active_until >= ? AND ls.active_from < ? AND NOT (YEAR(ls.active_until) = ? AND MONTH(ls.active_until) = ?)))`, []any{year, month, monthStartValue, recentStart, year, month}
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
	params.SubscriptionStatus = normalizeOutletSubscriptionStatusCode(params.SubscriptionStatus)
	params.Sort = strings.TrimSpace(params.Sort)
	return params
}

func normalizeOutletSubscriptionStatusCode(value string) string {
	value = strings.ToUpper(strings.TrimSpace(value))
	switch value {
	case OutletSubscriptionStatusNotSubscribe, OutletSubscriptionStatusExpired, OutletSubscriptionStatusDueThisMonth, OutletSubscriptionStatusSubscribed, OutletSubscriptionStatusNew:
		return value
	default:
		return ""
	}
}

func resolveReferenceMonth(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		now := time.Now().UTC()
		return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC), nil
	}
	referenceMonth, err := time.Parse("2006-01", value)
	if err != nil {
		return time.Time{}, err
	}
	return time.Date(referenceMonth.Year(), referenceMonth.Month(), 1, 0, 0, 0, 0, time.UTC), nil
}

func monthStart(referenceMonth time.Time) time.Time {
	return time.Date(referenceMonth.Year(), referenceMonth.Month(), 1, 0, 0, 0, 0, time.UTC)
}

func monthEnd(referenceMonth time.Time) time.Time {
	start := monthStart(referenceMonth)
	return start.AddDate(0, 1, -1)
}

func buildOutletSubscriptionStatusResponse(snapshot OutletSubscriptionSnapshot, referenceMonth time.Time) OutletSubscriptionStatusResponse {
	statusCode, statusLabel, remainingDays, remainingDaysDisplay, lastSubscriptionEndDisplay := classifyOutletSubscription(snapshot, referenceMonth)
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

func classifyOutletSubscription(snapshot OutletSubscriptionSnapshot, referenceMonth time.Time) (string, string, *int64, string, string) {
	if !snapshot.SubscriptionEnd.Valid || !snapshot.SubscriptionStart.Valid {
		return OutletSubscriptionStatusNotSubscribe, OutletSubscriptionStatusNotSubscribe, nil, NeverSubscribedDisplay, NeverSubscribedDisplay
	}
	start := dateOnly(snapshot.SubscriptionStart.Time)
	end := dateOnly(snapshot.SubscriptionEnd.Time)
	monthStartValue := monthStart(referenceMonth)
	monthEndValue := monthEnd(referenceMonth)
	remaining := int64(end.Sub(monthEndValue).Hours() / 24)
	remainingDisplay := fmt.Sprintf("%d", remaining)
	lastSubscriptionEndDisplay := end.Format("2006-01-02")

	if end.Before(monthStartValue) {
		threshold := monthEndValue.AddDate(0, 0, -60)
		if end.Before(threshold) {
			return OutletSubscriptionStatusNotSubscribe, OutletSubscriptionStatusNotSubscribe, &remaining, remainingDisplay, lastSubscriptionEndDisplay
		}
		return OutletSubscriptionStatusExpired, OutletSubscriptionStatusExpired, &remaining, remainingDisplay, lastSubscriptionEndDisplay
	}

	if snapshot.TenureMonths.Valid && snapshot.TenureMonths.Int64 == 1 && sameMonth(end, referenceMonth) {
		return OutletSubscriptionStatusSubscribed, OutletSubscriptionLabelOneMonth, &remaining, remainingDisplay, lastSubscriptionEndDisplay
	}

	if !start.Before(monthEndValue.AddDate(0, 0, -30)) {
		return OutletSubscriptionStatusNew, OutletSubscriptionStatusNew, &remaining, remainingDisplay, lastSubscriptionEndDisplay
	}

	if sameMonth(end, referenceMonth) {
		return OutletSubscriptionStatusDueThisMonth, OutletSubscriptionStatusDueThisMonth, &remaining, remainingDisplay, lastSubscriptionEndDisplay
	}

	return OutletSubscriptionStatusSubscribed, OutletSubscriptionStatusSubscribed, &remaining, remainingDisplay, lastSubscriptionEndDisplay
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
