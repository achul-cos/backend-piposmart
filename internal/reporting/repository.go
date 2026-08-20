package reporting

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func hasPermission(actor Actor, permission string) bool {
	for _, item := range actor.Permissions {
		if item == permission {
			return true
		}
	}
	return false
}

func canReadAll(actor Actor) bool {
	return actor.RoleCode == RoleAdmin || actor.RoleCode == RoleSupervisor || hasPermission(actor, PermissionReadAll)
}

func canReadOwn(actor Actor) bool {
	return canReadAll(actor) || actor.RoleCode == RoleSales || hasPermission(actor, PermissionReadOwn)
}

func normalizeDateRange(dateFrom, dateTo string) (time.Time, time.Time, error) {
	now := time.Now().UTC()
	defaultFrom := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	defaultTo := now
	if strings.TrimSpace(dateFrom) == "" && strings.TrimSpace(dateTo) == "" {
		return defaultFrom, defaultTo, nil
	}
	if strings.TrimSpace(dateFrom) == "" || strings.TrimSpace(dateTo) == "" {
		return time.Time{}, time.Time{}, fmt.Errorf("%w: date_from dan date_to harus diisi berpasangan", ErrInvalidFilter)
	}
	from, err := time.ParseInLocation("2006-01-02", dateFrom, time.UTC)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("%w: date_from tidak valid", ErrInvalidFilter)
	}
	to, err := time.ParseInLocation("2006-01-02", dateTo, time.UTC)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("%w: date_to tidak valid", ErrInvalidFilter)
	}
	to = to.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
	if to.Before(from) {
		return time.Time{}, time.Time{}, fmt.Errorf("%w: date_to tidak boleh lebih kecil dari date_from", ErrInvalidFilter)
	}
	return from, to, nil
}

func normalizeOptionalDateRange(dateFrom, dateTo string) (*time.Time, *time.Time, error) {
	if strings.TrimSpace(dateFrom) == "" && strings.TrimSpace(dateTo) == "" {
		return nil, nil, nil
	}
	if strings.TrimSpace(dateFrom) == "" || strings.TrimSpace(dateTo) == "" {
		return nil, nil, fmt.Errorf("%w: created_from dan created_to harus diisi berpasangan", ErrInvalidFilter)
	}
	from, err := time.ParseInLocation("2006-01-02", dateFrom, time.UTC)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: created_from tidak valid", ErrInvalidFilter)
	}
	to, err := time.ParseInLocation("2006-01-02", dateTo, time.UTC)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: created_to tidak valid", ErrInvalidFilter)
	}
	nextDay := to.AddDate(0, 0, 1)
	if nextDay.Before(from) {
		return nil, nil, fmt.Errorf("%w: created_to tidak boleh lebih kecil dari created_from", ErrInvalidFilter)
	}
	return &from, &nextDay, nil
}

func buildScopedSalesCondition(actor Actor, salesColumn, supervisorColumn string) (string, []any) {
	switch actor.RoleCode {
	case RoleSales:
		return fmt.Sprintf(" AND %s = ?", salesColumn), []any{actor.ID}
	case RoleSupervisor:
		if supervisorColumn != "" {
			return fmt.Sprintf(" AND %s = ?", supervisorColumn), []any{actor.ID}
		}
	}
	return "", nil
}

func buildOwnerVisibilityCondition(actor Actor, ownerColumn string) (string, []any) {
	switch actor.RoleCode {
	case RoleAdmin:
		return "", nil
	case RoleSupervisor:
		return fmt.Sprintf(` AND EXISTS (
			SELECT 1 FROM customer_leads cl
			WHERE cl.owner_id = %s
			  AND cl.deleted_at IS NULL
			  AND (cl.current_owner_user_id = ? OR cl.supervisor_id = ?)
		)`, ownerColumn), []any{actor.ID, actor.ID}
	case RoleSales:
		return fmt.Sprintf(` AND EXISTS (
			SELECT 1 FROM customer_leads cl
			WHERE cl.owner_id = %s
			  AND cl.deleted_at IS NULL
			  AND cl.current_owner_role = 'SALES'
			  AND cl.current_owner_user_id = ?
		)`, ownerColumn), []any{actor.ID}
	default:
		return " AND 1 = 0", nil
	}
}

func monthNameIDSQL(column string) string {
	return `CASE
		WHEN ` + column + ` IS NULL THEN ''
		WHEN MONTH(` + column + `) = 1 THEN 'Januari'
		WHEN MONTH(` + column + `) = 2 THEN 'Februari'
		WHEN MONTH(` + column + `) = 3 THEN 'Maret'
		WHEN MONTH(` + column + `) = 4 THEN 'April'
		WHEN MONTH(` + column + `) = 5 THEN 'Mei'
		WHEN MONTH(` + column + `) = 6 THEN 'Juni'
		WHEN MONTH(` + column + `) = 7 THEN 'Juli'
		WHEN MONTH(` + column + `) = 8 THEN 'Agustus'
		WHEN MONTH(` + column + `) = 9 THEN 'September'
		WHEN MONTH(` + column + `) = 10 THEN 'Oktober'
		WHEN MONTH(` + column + `) = 11 THEN 'November'
		ELSE 'Desember'
	END`
}

func buildSalesActorCondition(actor Actor, salesColumn string) (string, []any) {
	switch actor.RoleCode {
	case RoleSales:
		return fmt.Sprintf(" AND %s = ?", salesColumn), []any{actor.ID}
	default:
		return "", nil
	}
}

func buildSupervisorSalesVisibilityCondition(actor Actor, salesColumn string) (string, []any) {
	switch actor.RoleCode {
	case RoleAdmin:
		return "", nil
	case RoleSupervisor:
		return fmt.Sprintf(` AND EXISTS (
			SELECT 1 FROM customer_leads cl
			WHERE cl.active_sales_id = %s
			  AND cl.supervisor_id = ?
			  AND cl.deleted_at IS NULL
		)`, salesColumn), []any{actor.ID}
	case RoleSales:
		return fmt.Sprintf(" AND %s = ?", salesColumn), []any{actor.ID}
	default:
		return " AND 1 = 0", nil
	}
}

func buildPagination(params ListReportsParams) (page, limit int) {
	page, limit = params.Page, params.Limit
	if params.All {
		return 1, 0
	}
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	if limit > 10000 {
		limit = 10000
	}
	return page, limit
}

func paginateQuery(base string, page, limit int) string {
	if limit <= 0 {
		return base
	}
	offset := (page - 1) * limit
	return fmt.Sprintf("%s LIMIT %d OFFSET %d", base, limit, offset)
}

func scanRows(rows *sql.Rows, columns []string) ([]map[string]any, error) {
	defer rows.Close()
	items := make([]map[string]any, 0)
	for rows.Next() {
		values := make([]any, len(columns))
		dest := make([]any, len(columns))
		for i := range values {
			dest[i] = &values[i]
		}
		if err := rows.Scan(dest...); err != nil {
			return nil, err
		}
		row := make(map[string]any, len(columns))
		for i, col := range columns {
			switch v := values[i].(type) {
			case []byte:
				row[col] = string(v)
			case nil: // ← TAMBAH INI
				row[col] = ""
			default:
				row[col] = v
			}
		}
		items = append(items, row)
	}
	return items, rows.Err()
}

func countFromSubquery(ctx context.Context, db *sql.DB, query string, args ...any) (int64, error) {
	row := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM ("+query+") AS q", args...)
	var total int64
	if err := row.Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}

func (r *Repository) Dashboard(ctx context.Context, actor Actor, params DashboardParams) (*DashboardResponse, error) {
	if !canReadOwn(actor) {
		return nil, ErrForbidden
	}
	from, to, err := normalizeDateRange(params.DateFrom, params.DateTo)
	if err != nil {
		return nil, err
	}

	cards := make([]DashboardCard, 0, 6)
	var ownerCount, outletCount, activeSubscriptions, paidTopups, confirmedClosings int64
	var topupRevenue, closingAmount sql.NullString

	ownerQuery := `SELECT COUNT(*) FROM owners o WHERE o.deleted_at IS NULL`
	ownerScope, ownerScopeArgs := buildOwnerVisibilityCondition(actor, "o.id")
	if err := r.db.QueryRowContext(ctx, ownerQuery+ownerScope, ownerScopeArgs...).Scan(&ownerCount); err != nil {
		return nil, err
	}
	outletQuery := `SELECT COUNT(*) FROM outlets ot WHERE ot.deleted_at IS NULL`
	outletScope, outletScopeArgs := buildOwnerVisibilityCondition(actor, "ot.owner_id")
	if err := r.db.QueryRowContext(ctx, outletQuery+outletScope, outletScopeArgs...).Scan(&outletCount); err != nil {
		return nil, err
	}
	subscriptionQuery := `SELECT COUNT(*) FROM subscriptions s WHERE s.deleted_at IS NULL AND s.status = 'ACTIVE'`
	subscriptionScope, subscriptionScopeArgs := buildOwnerVisibilityCondition(actor, "s.owner_id")
	if err := r.db.QueryRowContext(ctx, subscriptionQuery+subscriptionScope, subscriptionScopeArgs...).Scan(&activeSubscriptions); err != nil {
		return nil, err
	}

	topupScope, topupArgs := buildOwnerVisibilityCondition(actor, "wp.owner_id")
	topupQuery := `SELECT COUNT(*), COALESCE(SUM(wp.amount), 0)
		FROM wallet_payments wp
		WHERE wp.status = 'ACCEPTED' AND wp.deleted_at IS NULL
			AND COALESCE(wp.transfer_date_override, wp.paid_at) >= ? AND COALESCE(wp.transfer_date_override, wp.paid_at) <= ?` + topupScope
	args := []any{from, to}
	args = append(args, topupArgs...)
	if err := r.db.QueryRowContext(ctx, topupQuery, args...).Scan(&paidTopups, &topupRevenue); err != nil {
		return nil, err
	}

	closingScope, closingArgs := buildOwnerVisibilityCondition(actor, "sc.owner_id")
	closingQuery := `SELECT COUNT(*), COALESCE(SUM(sc.final_amount), 0)
		FROM sales_closings sc
		WHERE sc.deleted_at IS NULL
			AND sc.status = 'CONFIRMED'
			AND COALESCE(sc.confirmed_at, sc.closed_at) >= ? AND COALESCE(sc.confirmed_at, sc.closed_at) <= ?` + closingScope
	args = []any{from, to}
	args = append(args, closingArgs...)
	if err := r.db.QueryRowContext(ctx, closingQuery, args...).Scan(&confirmedClosings, &closingAmount); err != nil {
		return nil, err
	}

	cards = append(cards,
		DashboardCard{Key: "owners_total", Label: "Total Owner", Value: fmt.Sprintf("%d", ownerCount), Description: "Owner aktif di CRM"},
		DashboardCard{Key: "outlets_total", Label: "Total Outlet", Value: fmt.Sprintf("%d", outletCount), Description: "Outlet aktif di CRM"},
		DashboardCard{Key: "subscriptions_active", Label: "Langganan Aktif", Value: fmt.Sprintf("%d", activeSubscriptions), Description: "Subscription status ACTIVE"},
		DashboardCard{Key: "topups_paid", Label: "Topup Diterima", Value: fmt.Sprintf("%d", paidTopups), Description: "Jumlah topup ACCEPTED di periode"},
		DashboardCard{Key: "topup_revenue", Label: "Revenue Topup", Value: nullDecimalString(topupRevenue), Description: "Omset topup periode filter"},
		DashboardCard{Key: "confirmed_closing_amount", Label: "Omset Closing Confirmed", Value: nullDecimalString(closingAmount), Description: "Closing confirmed periode filter"},
		DashboardCard{Key: "confirmed_closing_count", Label: "Closing Confirmed", Value: fmt.Sprintf("%d", confirmedClosings), Description: "Jumlah closing confirmed periode filter"},
	)

	return &DashboardResponse{
		Role:     actor.RoleCode,
		DateFrom: from.Format("2006-01-02"),
		DateTo:   to.Format("2006-01-02"),
		Cards:    cards,
	}, nil
}

func nullDecimalString(v sql.NullString) string {
	if !v.Valid || strings.TrimSpace(v.String) == "" {
		return "0"
	}
	return v.String
}

func nullableString(value string) sql.NullString {
	value = strings.TrimSpace(value)
	if value == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: value, Valid: true}
}

func reportColumns(reportKey string) ([]ReportColumn, error) {
	switch reportKey {
	case ReportOwnersOutlets:
		return []ReportColumn{
			{Key: "owner_code", Label: "Kode Owner", Type: "string"},
			{Key: "owner_name", Label: "Nama Owner", Type: "string"},
			{Key: "brand_name", Label: "Brand", Type: "string"},
			{Key: "province", Label: "Provinsi", Type: "string"},
			{Key: "city", Label: "Kota", Type: "string"},
			{Key: "outlet_count", Label: "Jumlah Outlet", Type: "number"},
			{Key: "wallet_balance", Label: "Saldo Wallet", Type: "currency"},
		}, nil
	case ReportActivities:
		return []ReportColumn{
			{Key: "interaction_at", Label: "Waktu Interaksi", Type: "datetime"},
			{Key: "lead_code", Label: "Kode Lead", Type: "string"},
			{Key: "owner_name", Label: "Owner", Type: "string"},
			{Key: "sales_name", Label: "Sales", Type: "string"},
			{Key: "supervisor_name", Label: "Supervisor", Type: "string"},
			{Key: "call_status", Label: "Status Call", Type: "string"},
			{Key: "chat_status", Label: "Status Chat", Type: "string"},
			{Key: "score", Label: "Score", Type: "number"},
			{Key: "next_follow_up_at", Label: "Follow Up Berikutnya", Type: "datetime"},
		}, nil
	case ReportTopups:
		return []ReportColumn{
			{Key: "payment_code", Label: "Kode Payment", Type: "string"},
			{Key: "owner_code", Label: "Kode Owner", Type: "string"},
			{Key: "owner_name", Label: "Owner", Type: "string"},
			{Key: "payment_channel", Label: "Channel", Type: "string"},
			{Key: "status", Label: "Status", Type: "string"},
			{Key: "amount", Label: "Nominal", Type: "currency"},
			{Key: "effective_transfer_at", Label: "Tanggal Transfer Efektif", Type: "datetime"},
		}, nil
	case ReportClosings:
		return []ReportColumn{
			{Key: "closing_code", Label: "Kode Closing", Type: "string"},
			{Key: "owner_code", Label: "Kode Owner", Type: "string"},
			{Key: "owner_name", Label: "Owner", Type: "string"},
			{Key: "sales_name", Label: "Sales", Type: "string"},
			{Key: "supervisor_name", Label: "Supervisor", Type: "string"},
			{Key: "package_name", Label: "Paket", Type: "string"},
			{Key: "tenure_months", Label: "Tenor Bulan", Type: "number"},
			{Key: "final_amount", Label: "Final Amount", Type: "currency"},
			{Key: "status", Label: "Status", Type: "string"},
			{Key: "closed_at", Label: "Closed At", Type: "datetime"},
			{Key: "confirmed_at", Label: "Confirmed At", Type: "datetime"},
		}, nil
	case ReportSubscriptions:
		return []ReportColumn{
			{Key: "subscription_code", Label: "Kode Subscription", Type: "string"},
			{Key: "owner_code", Label: "Kode Owner", Type: "string"},
			{Key: "owner_name", Label: "Owner", Type: "string"},
			{Key: "outlet_code", Label: "Kode Outlet", Type: "string"},
			{Key: "outlet_name", Label: "Outlet", Type: "string"},
			{Key: "status", Label: "Status", Type: "string"},
			{Key: "active_from", Label: "Active From", Type: "date"},
			{Key: "active_until", Label: "Active Until", Type: "date"},
			{Key: "total_duration_days", Label: "Durasi Hari", Type: "number"},
		}, nil
	case ReportPartners:
		return []ReportColumn{
			{Key: "partner_code", Label: "Kode Mitra", Type: "string"},
			{Key: "partner_name", Label: "Nama Mitra", Type: "string"},
			{Key: "partner_type", Label: "Jenis Mitra", Type: "string"},
			{Key: "closing_code", Label: "Kode Closing", Type: "string"},
			{Key: "commission_amount", Label: "Komisi", Type: "currency"},
			{Key: "commission_status", Label: "Status Komisi", Type: "string"},
			{Key: "payout_code", Label: "Kode Payout", Type: "string"},
			{Key: "payout_status", Label: "Status Payout", Type: "string"},
		}, nil
	case ReportTargetsKPI:
		return []ReportColumn{
			{Key: "sales_code", Label: "Kode Sales", Type: "string"},
			{Key: "sales_name", Label: "Sales", Type: "string"},
			{Key: "period_year", Label: "Tahun", Type: "number"},
			{Key: "period_month", Label: "Bulan", Type: "number"},
			{Key: "target_value", Label: "Target", Type: "number"},
			{Key: "actual_value", Label: "Aktual", Type: "number"},
			{Key: "total_score", Label: "Skor KPI", Type: "number"},
			{Key: "classification", Label: "Klasifikasi", Type: "string"},
			{Key: "rank_position", Label: "Peringkat", Type: "number"},
		}, nil
	case ReportAdminOwner:
		return []ReportColumn{
			{Key: "no", Label: "No", Type: "number"},
			{Key: "date_of_work", Label: "Date of Work", Type: "string"},
			{Key: "nama_penginput", Label: "Nama Penginput", Type: "string"},
			{Key: "kategori_akun", Label: "Kategori Akun", Type: "string"},
			{Key: "kode_baris", Label: "Kode Baris", Type: "string"},
			{Key: "owner_code", Label: "Kode Owner", Type: "string"},
			{Key: "owner_name", Label: "Nama Owner", Type: "string"},
			{Key: "owner_email", Label: "Email Owner", Type: "string"},
			{Key: "owner_phone", Label: "No Hp Owner", Type: "string"},
			{Key: "create_date_project", Label: "Create Date Project", Type: "string"},
			{Key: "bulan", Label: "Bulan", Type: "string"},
			{Key: "brand_name", Label: "Nama Project/BRAND", Type: "string"},
			{Key: "kelurahan", Label: "Kelurahan", Type: "string"},
			{Key: "kecamatan", Label: "Kecamatan", Type: "string"},
			{Key: "kota", Label: "Kota", Type: "string"},
			{Key: "provinsi", Label: "Provinsi", Type: "string"},
			{Key: "alamat_lengkap", Label: "Alamat Lengkap", Type: "string"},
			{Key: "jumlah_outlet", Label: "Jumlah Outlet", Type: "number"},
		}, nil
	case ReportAdminOwnerOutlet:
		return GetAdminOwnerOutletColumns(), nil
	case ReportAdminNewSubscribe:
		return []ReportColumn{
			{Key: "date_of_work", Label: "Date Of Work", Type: "date"},
			{Key: "owner_code", Label: "Kode Owner", Type: "string"},
			{Key: "owner_name", Label: "Nama Owner", Type: "string"},
			{Key: "owner_phone", Label: "No. Hp Owner", Type: "string"},
			{Key: "outlet_phone", Label: "No. Hp Outlet", Type: "string"},
			{Key: "project_name", Label: "Project/Outlet", Type: "string"},
			{Key: "city", Label: "Kota", Type: "string"},
			{Key: "province", Label: "Provinsi", Type: "string"},
			{Key: "created_date", Label: "Created Date", Type: "date"},
			{Key: "topup_date", Label: "Date Top UP System", Type: "datetime"},
			{Key: "activation_amount", Label: "Nominal Aktivasi", Type: "currency"},
			{Key: "activation_date", Label: "Tanggal Aktivasi", Type: "datetime"},
			{Key: "package_name", Label: "Paket Membership", Type: "string"},
			{Key: "status", Label: "Status", Type: "string"},
		}, nil
	case ReportAdminNasabahProvinsi:
		return []ReportColumn{
			{Key: "year_member", Label: "Tahun Member", Type: "number"},
			{Key: "month_member", Label: "Bulan", Type: "string"},
			{Key: "owner_code", Label: "Kode Owner", Type: "string"},
			{Key: "owner_name", Label: "Nama Owner", Type: "string"},
			{Key: "owner_phone", Label: "No. Hp Owner", Type: "string"},
			{Key: "owner_email", Label: "Email", Type: "string"},
			{Key: "project_outlet", Label: "Project/Outlet", Type: "string"},
			{Key: "city", Label: "Kota", Type: "string"},
			{Key: "address", Label: "Alamat", Type: "string"},
			{Key: "province", Label: "Provinsi", Type: "string"},
		}, nil
	default:
		return nil, ErrInvalidReportKey
	}
}

func (r *Repository) ListReport(ctx context.Context, actor Actor, reportKey string, params ListReportsParams) (*ReportListResponse, error) {
	if !canReadOwn(actor) {
		return nil, ErrForbidden
	}
	columns, err := reportColumns(reportKey)
	if err != nil {
		return nil, err
	}
	page, limit := buildPagination(params)
	from, to, err := normalizeDateRange(params.DateFrom, params.DateTo)
	if err != nil {
		return nil, err
	}
	createdFrom, createdTo, err := normalizeOptionalDateRange(params.CreatedFrom, params.CreatedTo)
	if err != nil {
		return nil, err
	}
	baseQuery, args, err := r.buildReportQuery(actor, reportKey, params, from, to, createdFrom, createdTo)
	if err != nil {
		return nil, err
	}
	total, err := countFromSubquery(ctx, r.db, baseQuery, args...)
	if err != nil {
		return nil, err
	}
	query := paginateQuery(baseQuery, page, limit)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	keys := make([]string, 0, len(columns))
	for _, column := range columns {
		keys = append(keys, column.Key)
	}
	items, err := scanRows(rows, keys)
	if err != nil {
		return nil, err
	}
	insight := map[string]any{
		"date_from": from.Format("2006-01-02"),
		"date_to":   to.Format("2006-01-02"),
		"count":     total,
	}
	if createdFrom != nil && createdTo != nil {
		insight["created_from"] = createdFrom.Format("2006-01-02")
		insight["created_to"] = createdTo.AddDate(0, 0, -1).Format("2006-01-02")
	}
	return &ReportListResponse{
		ReportKey: reportKey,
		Columns:   columns,
		Items:     items,
		Pagination: PaginationMeta{
			Page: page,
			Limit: func() int {
				if limit <= 0 {
					return len(items)
				}
				return limit
			}(),
			Total: total,
		},
		Insight: insight,
	}, nil
}

func (r *Repository) buildReportQuery(actor Actor, reportKey string, params ListReportsParams, from, to time.Time, createdFrom, createdTo *time.Time) (string, []any, error) {
	switch reportKey {
	case ReportOwnersOutlets:
		query := `SELECT
			o.code AS owner_code,
			o.name AS owner_name,
			COALESCE(o.brand_name, '') AS brand_name,
			COALESCE(o.province, '') AS province,
			COALESCE(o.city, '') AS city,
			COUNT(DISTINCT ot.id) AS outlet_count,
			COALESCE(MAX(wa.balance), 0) AS wallet_balance
		FROM owners o
		LEFT JOIN outlets ot ON ot.owner_id = o.id AND ot.deleted_at IS NULL
		LEFT JOIN wallet_accounts wa ON wa.owner_id = o.id
		WHERE o.deleted_at IS NULL
		  AND o.created_at >= ? AND o.created_at <= ?`
		args := []any{from, to}
		if scope, scopeArgs := buildOwnerVisibilityCondition(actor, "o.id"); scope != "" {
			query += scope
			args = append(args, scopeArgs...)
		}
		if params.Query != "" {
			query += ` AND (o.code LIKE ? OR o.name LIKE ? OR o.brand_name LIKE ?)`
			like := "%" + params.Query + "%"
			args = append(args, like, like, like)
		}
		if params.Province != "" {
			query += ` AND o.province = ?`
			args = append(args, params.Province)
		}
		if params.City != "" {
			query += ` AND o.city = ?`
			args = append(args, params.City)
		}
		if createdFrom != nil && createdTo != nil {
			query += ` AND o.created_at >= ? AND o.created_at < ?`
			args = append(args, *createdFrom, *createdTo)
		}
		query += ` GROUP BY o.id, o.code, o.name, o.brand_name, o.province, o.city ORDER BY o.created_at DESC`
		return query, args, nil

	case ReportActivities:
		query := `SELECT
			ci.interaction_at AS interaction_at,
			cl.code AS lead_code,
			COALESCE(o.name, '') AS owner_name,
			COALESCE(sales.name, '') AS sales_name,
			COALESCE(sup.name, '') AS supervisor_name,
			COALESCE(ci.call_status, '') AS call_status,
			COALESCE(ci.chat_status, '') AS chat_status,
			COALESCE(ci.remark_score, 0) AS score,
			ci.next_follow_up_at AS next_follow_up_at
		FROM customer_interactions ci
		LEFT JOIN customer_leads cl ON cl.id = ci.lead_id
		LEFT JOIN owners o ON o.id = ci.owner_id
		LEFT JOIN users sales ON sales.id = ci.sales_id
		LEFT JOIN users sup ON sup.id = ci.supervisor_id
		WHERE ci.deleted_at IS NULL
		  AND ci.interaction_at >= ? AND ci.interaction_at <= ?`
		args := []any{from, to}
		if scope, scopeArgs := buildOwnerVisibilityCondition(actor, "ci.owner_id"); scope != "" {
			query += scope
			args = append(args, scopeArgs...)
		}
		if params.Status != "" {
			query += ` AND (ci.call_status = ? OR ci.chat_status = ?)`
			args = append(args, params.Status, params.Status)
		}
		if params.Query != "" {
			like := "%" + params.Query + "%"
			query += ` AND (cl.code LIKE ? OR o.name LIKE ? OR sales.name LIKE ? OR sup.name LIKE ?)`
			args = append(args, like, like, like, like)
		}
		if params.SalesID != nil {
			query += ` AND ci.sales_id = ?`
			args = append(args, *params.SalesID)
		}
		if params.SupervisorID != nil {
			query += ` AND ci.supervisor_id = ?`
			args = append(args, *params.SupervisorID)
		}
		if createdFrom != nil && createdTo != nil {
			query += ` AND ci.created_at >= ? AND ci.created_at < ?`
			args = append(args, *createdFrom, *createdTo)
		}
		query += ` ORDER BY ci.interaction_at DESC`
		return query, args, nil

	case ReportTopups:
		query := `SELECT
			wp.code AS payment_code,
			COALESCE(o.code, '') AS owner_code,
			COALESCE(o.name, '') AS owner_name,
			wp.payment_channel AS payment_channel,
			wp.status AS status,
			wp.amount AS amount,
			COALESCE(wp.transfer_date_override, wp.paid_at) AS effective_transfer_at
		FROM wallet_payments wp
		LEFT JOIN owners o ON o.id = wp.owner_id
		WHERE wp.deleted_at IS NULL
		  AND COALESCE(wp.transfer_date_override, wp.paid_at, wp.created_at) >= ?
		  AND COALESCE(wp.transfer_date_override, wp.paid_at, wp.created_at) <= ?`
		args := []any{from, to}
		if scope, scopeArgs := buildOwnerVisibilityCondition(actor, "wp.owner_id"); scope != "" {
			query += scope
			args = append(args, scopeArgs...)
		}
		if params.Status != "" {
			query += ` AND wp.status = ?`
			args = append(args, params.Status)
		}
		if params.Query != "" {
			like := "%" + params.Query + "%"
			query += ` AND (wp.code LIKE ? OR o.code LIKE ? OR o.name LIKE ? OR wp.external_reference LIKE ?)`
			args = append(args, like, like, like, like)
		}
		if params.Province != "" {
			query += ` AND o.province = ?`
			args = append(args, params.Province)
		}
		if params.City != "" {
			query += ` AND o.city = ?`
			args = append(args, params.City)
		}
		if createdFrom != nil && createdTo != nil {
			query += ` AND wp.created_at >= ? AND wp.created_at < ?`
			args = append(args, *createdFrom, *createdTo)
		}
		query += ` ORDER BY effective_transfer_at DESC, wp.created_at DESC`
		return query, args, nil

	case ReportClosings:
		query := `SELECT
			sc.code AS closing_code,
			COALESCE(o.code, '') AS owner_code,
			COALESCE(o.name, '') AS owner_name,
			COALESCE(sales.name, '') AS sales_name,
			COALESCE(sup.name, '') AS supervisor_name,
			JSON_UNQUOTE(JSON_EXTRACT(sc.package_snapshot_json, '$.name')) AS package_name,
			sc.tenure_months AS tenure_months,
			sc.final_amount AS final_amount,
			sc.status AS status,
			sc.closed_at AS closed_at,
			sc.confirmed_at AS confirmed_at
		FROM sales_closings sc
		LEFT JOIN owners o ON o.id = sc.owner_id
		LEFT JOIN users sales ON sales.id = sc.sales_id
		LEFT JOIN users sup ON sup.id = sc.supervisor_id
		WHERE sc.deleted_at IS NULL
		  AND COALESCE(sc.confirmed_at, sc.closed_at) >= ? AND COALESCE(sc.confirmed_at, sc.closed_at) <= ?`
		args := []any{from, to}
		if scope, scopeArgs := buildOwnerVisibilityCondition(actor, "sc.owner_id"); scope != "" {
			query += scope
			args = append(args, scopeArgs...)
		}
		if params.Status != "" {
			query += ` AND sc.status = ?`
			args = append(args, params.Status)
		}
		if params.Query != "" {
			like := "%" + params.Query + "%"
			query += ` AND (sc.code LIKE ? OR o.code LIKE ? OR o.name LIKE ? OR sales.name LIKE ? OR sup.name LIKE ?)`
			args = append(args, like, like, like, like, like)
		}
		if params.SalesID != nil {
			query += ` AND sc.sales_id = ?`
			args = append(args, *params.SalesID)
		}
		if params.SupervisorID != nil {
			query += ` AND sc.supervisor_id = ?`
			args = append(args, *params.SupervisorID)
		}
		if createdFrom != nil && createdTo != nil {
			query += ` AND sc.created_at >= ? AND sc.created_at < ?`
			args = append(args, *createdFrom, *createdTo)
		}
		query += ` ORDER BY sc.closed_at DESC`
		return query, args, nil

	case ReportSubscriptions:
		query := `SELECT
			s.code AS subscription_code,
			COALESCE(o.code, '') AS owner_code,
			COALESCE(o.name, '') AS owner_name,
			COALESCE(ot.code, '') AS outlet_code,
			COALESCE(ot.name, '') AS outlet_name,
			s.status AS status,
			s.active_from AS active_from,
			s.active_until AS active_until,
			s.total_duration_days AS total_duration_days
		FROM subscriptions s
		LEFT JOIN owners o ON o.id = s.owner_id
		LEFT JOIN outlets ot ON ot.id = s.outlet_id
		LEFT JOIN subscription_orders so ON so.id = s.order_id
		WHERE s.deleted_at IS NULL
		  AND s.active_from <= ? AND s.active_until >= ?`
		args := []any{to, from}
		if scope, scopeArgs := buildOwnerVisibilityCondition(actor, "s.owner_id"); scope != "" {
			query += scope
			args = append(args, scopeArgs...)
		}
		if params.Status != "" {
			query += ` AND s.status = ?`
			args = append(args, params.Status)
		}
		if params.Query != "" {
			like := "%" + params.Query + "%"
			query += ` AND (s.code LIKE ? OR o.code LIKE ? OR o.name LIKE ? OR ot.code LIKE ? OR ot.name LIKE ?)`
			args = append(args, like, like, like, like, like)
		}
		if params.SalesID != nil {
			query += ` AND so.sales_id = ?`
			args = append(args, *params.SalesID)
		}
		if params.SupervisorID != nil {
			query += ` AND so.supervisor_id = ?`
			args = append(args, *params.SupervisorID)
		}
		if createdFrom != nil && createdTo != nil {
			query += ` AND s.created_at >= ? AND s.created_at < ?`
			args = append(args, *createdFrom, *createdTo)
		}
		query += ` ORDER BY s.active_until ASC`
		return query, args, nil

	case ReportPartners:
		query := `SELECT
			COALESCE(p.code, '') AS partner_code,
			COALESCE(p.name, '') AS partner_name,
			COALESCE(pt.name, '') AS partner_type,
			COALESCE(pc.code, '') AS closing_code,
			COALESCE(c.commission_amount, 0) AS commission_amount,
			c.status AS commission_status,
			COALESCE(pp.code, '') AS payout_code,
			COALESCE(pp.status, '') AS payout_status
		FROM partner_commissions c
		LEFT JOIN partners p ON p.id = c.partner_id
		LEFT JOIN partner_types pt ON pt.id = p.partner_type_id
		LEFT JOIN sales_closings pc ON pc.id = c.closing_id
		LEFT JOIN partner_payout_items ppi ON ppi.commission_id = c.id
		LEFT JOIN partner_payouts pp ON pp.id = ppi.payout_id
		WHERE c.created_at >= ? AND c.created_at <= ?`
		args := []any{from, to}
		if scope, scopeArgs := buildSalesActorCondition(actor, "pa.user_id"); scope != "" {
			query += ` AND EXISTS (
				SELECT 1 FROM partner_assignments pa
				WHERE pa.partner_id = p.id
				  AND pa.active = TRUE` + scope + `)`
			args = append(args, scopeArgs...)
		}
		if params.Status != "" {
			query += ` AND c.status = ?`
			args = append(args, params.Status)
		}
		if params.Query != "" {
			like := "%" + params.Query + "%"
			query += ` AND (p.code LIKE ? OR p.name LIKE ? OR pc.code LIKE ? OR pp.code LIKE ?)`
			args = append(args, like, like, like, like)
		}
		if createdFrom != nil && createdTo != nil {
			query += ` AND c.created_at >= ? AND c.created_at < ?`
			args = append(args, *createdFrom, *createdTo)
		}
		query += ` ORDER BY c.created_at DESC`
		return query, args, nil

	case ReportTargetsKPI:
		query := `SELECT
			COALESCE(u.code, '') AS sales_code,
			u.name AS sales_name,
			skr.period_year AS period_year,
			skr.period_month AS period_month,
			COALESCE(st.target_value, 0) AS target_value,
			COALESCE(actual.metric_value, 0) AS actual_value,
			COALESCE(skr.total_score, 0) AS total_score,
			COALESCE(skr.classification, '') AS classification,
			COALESCE(skr.rank_position, 0) AS rank_position
		FROM sales_kpi_results skr
			INNER JOIN users u ON u.id = skr.sales_id
		LEFT JOIN sales_targets st ON st.sales_id = skr.sales_id
			AND st.period_year = skr.period_year AND st.period_month = skr.period_month
		LEFT JOIN metric_codes mc ON mc.id = st.metric_code_id AND mc.code = 'CONFIRMED_CLOSING_COUNT'
		LEFT JOIN (
			SELECT kpir.sales_id, kd.period_year, kd.period_month, kpir.metric_value
			FROM kpi_metric_results kpir
			INNER JOIN kpi_definitions kd ON kd.id = kpir.kpi_definition_id
			INNER JOIN metric_codes mc2 ON mc2.id = kd.metric_code_id
			WHERE mc2.code = 'CONFIRMED_CLOSING_COUNT'
		) actual ON actual.sales_id = skr.sales_id
			AND actual.period_year = skr.period_year AND actual.period_month = skr.period_month
		WHERE skr.computed_at >= ? AND skr.computed_at <= ?`
		args := []any{from, to}
		if scope, scopeArgs := buildSupervisorSalesVisibilityCondition(actor, "skr.sales_id"); scope != "" {
			query += scope
			args = append(args, scopeArgs...)
		}
		if params.SalesID != nil {
			query += ` AND skr.sales_id = ?`
			args = append(args, *params.SalesID)
		}
		if createdFrom != nil && createdTo != nil {
			query += ` AND skr.created_at >= ? AND skr.created_at < ?`
			args = append(args, *createdFrom, *createdTo)
		}
		query += ` ORDER BY skr.period_year DESC, skr.period_month DESC, skr.rank_position ASC`
		return query, args, nil

	case ReportAdminOwner:
		// ✅ PERBAIKAN: 1 row per OWNER (tidak per outlet)
		// Tidak ada LEFT JOIN outlets, hanya count outlets
		query := `SELECT
			0 AS no,
			DATE_FORMAT(o.created_at, '%d/%m/%Y') AS date_of_work,
			'-' AS nama_penginput,
			'OWNER' AS kategori_akun,
			CAST(o.id AS CHAR) AS kode_baris,
			o.code AS owner_code,
			o.name AS owner_name,
			COALESCE(NULLIF(o.email, ''), '') AS owner_email,
			CASE
				WHEN COALESCE(NULLIF(o.phone,''), '') REGEXP '^62[0-9]'
				THEN CONCAT('0', SUBSTRING(COALESCE(NULLIF(o.phone,''), ''), 3))
				ELSE COALESCE(NULLIF(o.phone,''), '')
			END AS owner_phone,
			DATE_FORMAT(o.created_at, '%d/%m/%Y') AS create_date_project,
			` + monthNameIDSQL("o.created_at") + ` AS bulan,
			CASE WHEN COALESCE(o.brand_name,'') LIKE '%#REF!%' OR COALESCE(o.brand_name,'') LIKE '%#VALUE!%' THEN '' ELSE COALESCE(o.brand_name, '') END AS brand_name,
			COALESCE(NULLIF(o.sub_district, ''), '') AS kelurahan,
			COALESCE(NULLIF(o.district, ''), '') AS kecamatan,
			COALESCE(NULLIF(o.city, ''), '') AS kota,
			COALESCE(NULLIF(o.province, ''), '') AS provinsi,
			COALESCE(NULLIF(o.address, ''), '') AS alamat_lengkap,
			COALESCE(outlets_count.total, 0) AS jumlah_outlet
		FROM owners o
		LEFT JOIN (SELECT owner_id, COUNT(id) AS total FROM outlets WHERE deleted_at IS NULL GROUP BY owner_id) outlets_count ON outlets_count.owner_id = o.id
		WHERE o.deleted_at IS NULL AND o.created_at >= ? AND o.created_at <= ?`
		args := []any{from, to}
		if scope, scopeArgs := buildOwnerVisibilityCondition(actor, "o.id"); scope != "" {
			query += scope
			args = append(args, scopeArgs...)
		}
		if params.Query != "" {
			like := "%" + params.Query + "%"
			query += ` AND (o.code LIKE ? OR COALESCE(o.email, '') LIKE ? OR COALESCE(o.brand_name, '') LIKE ?)`
			args = append(args, like, like, like)
		}
		if params.Province != "" {
			query += ` AND COALESCE(o.province, '') = ?`
			args = append(args, params.Province)
		}
		if params.City != "" {
			query += ` AND COALESCE(o.city, '') = ?`
			args = append(args, params.City)
		}
		if createdFrom != nil && createdTo != nil {
			query += ` AND o.created_at >= ? AND o.created_at < ?`
			args = append(args, *createdFrom, *createdTo)
		}
		query += ` ORDER BY o.created_at DESC`
		return query, args, nil

	case ReportAdminOwnerOutlet:
		// ✅ PERBAIKAN: 1 row per OUTLET (per outlet, owner bisa repeat)
		// Ada LEFT JOIN ke outlets untuk ambil outlet details
		query := `SELECT
			0 AS no,
			DATE_FORMAT(o.created_at, '%d/%m/%Y') AS date_of_work,
			'-' AS nama_penginput,
			'OWNER' AS kategori_akun,
			COALESCE(ot.row_code, '') AS kode_baris,
			o.code AS owner_code,
			o.name AS owner_name,
			COALESCE(NULLIF(o.email, ''), '') AS owner_email,
			CASE
				WHEN COALESCE(NULLIF(o.phone,''), NULLIF(ot.phone,''), '') REGEXP '^62[0-9]'
				THEN CONCAT('0', SUBSTRING(COALESCE(NULLIF(o.phone,''), NULLIF(ot.phone,''), ''), 3))
				ELSE COALESCE(NULLIF(o.phone,''), NULLIF(ot.phone,''), '')
			END AS owner_phone,
			CASE
				WHEN COALESCE(NULLIF(ot.phone,''), NULLIF(o.phone,''), '') REGEXP '^62[0-9]'
				THEN CONCAT('0', SUBSTRING(COALESCE(NULLIF(ot.phone,''), NULLIF(o.phone,''), ''), 3))
				ELSE COALESCE(NULLIF(ot.phone,''), NULLIF(o.phone,''), '')
			END AS outlet_phone,
			DATE_FORMAT(o.created_at, '%d/%m/%Y') AS create_date_project,
			` + monthNameIDSQL("o.created_at") + ` AS bulan,
			CASE WHEN COALESCE(o.brand_name,'') LIKE '%#REF!%' OR COALESCE(o.brand_name,'') LIKE '%#VALUE!%' THEN '' ELSE COALESCE(o.brand_name, '') END AS brand_name,
			CASE WHEN COALESCE(ot.name,'') LIKE '%#REF!%' OR COALESCE(ot.name,'') LIKE '%#VALUE!%' THEN '' ELSE COALESCE(ot.name, '') END AS outlet_name,
			COALESCE(NULLIF(ot.sub_district, ''), NULLIF(o.sub_district, ''), '') AS kelurahan,
			COALESCE(NULLIF(ot.district, ''), NULLIF(o.district, ''), '') AS kecamatan,
			COALESCE(NULLIF(ot.city, ''), NULLIF(o.city, ''), '') AS kota,
			COALESCE(NULLIF(ot.province, ''), NULLIF(o.province, ''), '') AS provinsi,
			COALESCE(NULLIF(ot.address, ''), NULLIF(o.address, ''), '') AS alamat_lengkap,
			COALESCE(outlets_count.total, 0) AS jumlah_outlet
		FROM owners o
		LEFT JOIN outlets ot ON ot.owner_id = o.id AND ot.deleted_at IS NULL
		LEFT JOIN (SELECT owner_id, COUNT(id) AS total FROM outlets WHERE deleted_at IS NULL GROUP BY owner_id) outlets_count ON outlets_count.owner_id = o.id
		WHERE o.deleted_at IS NULL AND o.created_at >= ? AND o.created_at <= ?`
		args := []any{from, to}
		if scope, scopeArgs := buildOwnerVisibilityCondition(actor, "o.id"); scope != "" {
			query += scope
			args = append(args, scopeArgs...)
		}
		if params.Query != "" {
			like := "%" + params.Query + "%"
			query += ` AND (o.code LIKE ? OR COALESCE(o.email, '') LIKE ? OR COALESCE(o.brand_name, '') LIKE ? OR COALESCE(ot.name, '') LIKE ?)`
			args = append(args, like, like, like, like)
		}
		if params.Province != "" {
			query += ` AND COALESCE(ot.province, o.province, '') = ?`
			args = append(args, params.Province)
		}
		if params.City != "" {
			query += ` AND COALESCE(ot.city, o.city, '') = ?`
			args = append(args, params.City)
		}
		if createdFrom != nil && createdTo != nil {
			query += ` AND o.created_at >= ? AND o.created_at < ?`
			args = append(args, *createdFrom, *createdTo)
		}
		query += ` ORDER BY o.created_at DESC, ot.created_at DESC, ot.id DESC`
		return query, args, nil

	case ReportAdminNewSubscribe:
		query := `SELECT
			DATE(COALESCE(sc.confirmed_at, sc.closed_at, so.purchased_at)) AS date_of_work,
			COALESCE(o.code, '') AS owner_code,
			COALESCE(o.name, '') AS owner_name,
			COALESCE(o.phone, '') AS owner_phone,
			COALESCE(ot.phone, '') AS outlet_phone,
			COALESCE(ot.name, JSON_UNQUOTE(JSON_EXTRACT(sc.package_snapshot_json, '$.name')), '') AS project_name,
			COALESCE(ot.city, o.city, '') AS city,
			COALESCE(ot.province, o.province, '') AS province,
			DATE(o.created_at) AS created_date,
			MIN(wp.paid_at) AS topup_date,
			COALESCE(sc.final_amount, so.final_amount, 0) AS activation_amount,
			COALESCE(sc.confirmed_at, sc.closed_at, so.purchased_at) AS activation_date,
			JSON_UNQUOTE(JSON_EXTRACT(sc.package_snapshot_json, '$.name')) AS package_name,
			COALESCE(sc.status, so.status, '') AS status
		FROM subscription_orders so
		LEFT JOIN owners o ON o.id = so.owner_id
		LEFT JOIN outlets ot ON ot.id = so.outlet_id
		LEFT JOIN sales_closings sc ON sc.id = so.closing_id
		LEFT JOIN wallet_payments wp ON wp.owner_id = so.owner_id AND wp.status = 'ACCEPTED'
		WHERE so.deleted_at IS NULL
		  AND COALESCE(sc.confirmed_at, sc.closed_at, so.purchased_at) >= ?
		  AND COALESCE(sc.confirmed_at, sc.closed_at, so.purchased_at) <= ?`
		args := []any{from, to}
		if scope, scopeArgs := buildOwnerVisibilityCondition(actor, "so.owner_id"); scope != "" {
			query += scope
			args = append(args, scopeArgs...)
		}
		if params.Status != "" {
			query += ` AND COALESCE(sc.status, so.status, '') = ?`
			args = append(args, params.Status)
		}
		if params.Query != "" {
			like := "%" + params.Query + "%"
			query += ` AND (so.code LIKE ? OR COALESCE(o.code, '') LIKE ? OR COALESCE(o.name, '') LIKE ? OR COALESCE(ot.name, '') LIKE ?)`
			args = append(args, like, like, like, like)
		}
		if params.Province != "" {
			query += ` AND COALESCE(ot.province, o.province, '') = ?`
			args = append(args, params.Province)
		}
		if params.City != "" {
			query += ` AND COALESCE(ot.city, o.city, '') = ?`
			args = append(args, params.City)
		}
		if params.SalesID != nil {
			query += ` AND so.sales_id = ?`
			args = append(args, *params.SalesID)
		}
		if params.SupervisorID != nil {
			query += ` AND so.supervisor_id = ?`
			args = append(args, *params.SupervisorID)
		}
		if createdFrom != nil && createdTo != nil {
			query += ` AND so.created_at >= ? AND so.created_at < ?`
			args = append(args, *createdFrom, *createdTo)
		}
		query += ` GROUP BY so.id ORDER BY activation_date DESC`
		return query, args, nil

	case ReportAdminNasabahProvinsi:
		query := `SELECT
			YEAR(o.created_at) AS year_member,
			DATE_FORMAT(o.created_at, '%M') AS month_member,
			o.code AS owner_code,
			o.name AS owner_name,
			COALESCE(o.phone, '') AS owner_phone,
			COALESCE(o.email, '') AS owner_email,
			COALESCE(ot.name, o.brand_name, '') AS project_outlet,
			COALESCE(ot.city, o.city, '') AS city,
			COALESCE(ot.address, o.address, '') AS address,
			COALESCE(ot.province, o.province, '') AS province
		FROM owners o
		LEFT JOIN outlets ot ON ot.owner_id = o.id AND ot.deleted_at IS NULL
		WHERE o.deleted_at IS NULL
		  AND o.created_at >= ? AND o.created_at <= ?`
		args := []any{from, to}
		if scope, scopeArgs := buildOwnerVisibilityCondition(actor, "o.id"); scope != "" {
			query += scope
			args = append(args, scopeArgs...)
		}
		if params.Query != "" {
			like := "%" + params.Query + "%"
			query += ` AND (o.code LIKE ? OR o.name LIKE ? OR COALESCE(ot.name, '') LIKE ? OR COALESCE(o.email, '') LIKE ?)`
			args = append(args, like, like, like, like)
		}
		if params.Province != "" {
			query += ` AND COALESCE(ot.province, o.province, '') = ?`
			args = append(args, params.Province)
		}
		if params.City != "" {
			query += ` AND COALESCE(ot.city, o.city, '') = ?`
			args = append(args, params.City)
		}
		if createdFrom != nil && createdTo != nil {
			query += ` AND o.created_at >= ? AND o.created_at < ?`
			args = append(args, *createdFrom, *createdTo)
		}
		query += ` ORDER BY province ASC, city ASC, o.created_at DESC`
		return query, args, nil

	default:
		return "", nil, ErrInvalidReportKey
	}
}

func (r *Repository) CreateExport(ctx context.Context, code, reportKey, format string, filters map[string]string, actorID int64) (int64, error) {
	body, err := json.Marshal(filters)
	if err != nil {
		return 0, fmt.Errorf("reporting: marshal filters: %w", err)
	}
	result, err := r.db.ExecContext(ctx, `INSERT INTO report_exports
		(code, report_key, format, status, filters_json, requested_by_user_id)
		VALUES (?, ?, ?, ?, ?, ?)`,
		code, reportKey, format, ExportStatusPending, string(body), actorID,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (r *Repository) SetExportJobID(ctx context.Context, exportID, jobID int64) error {
	_, err := r.db.ExecContext(ctx, `UPDATE report_exports SET job_id = ?, updated_at = NOW() WHERE id = ?`, jobID, exportID)
	return err
}

func (r *Repository) GetExportByID(ctx context.Context, exportID int64) (*ReportExport, error) {
	row := r.db.QueryRowContext(ctx, `SELECT
		re.id, re.code, re.report_key, re.format, re.status, re.filters_json,
		re.requested_by_user_id, COALESCE(u.name, ''), COALESCE(r.code, ''), re.job_id, re.file_path, re.file_name,
		re.mime_type, re.storage_disk, re.row_count, re.last_error, re.completed_at, re.created_at, re.updated_at
		FROM report_exports re
		LEFT JOIN users u ON u.id = re.requested_by_user_id
		LEFT JOIN roles r ON r.id = u.role_id
		WHERE re.id = ?`, exportID)
	var item ReportExport
	if err := row.Scan(
		&item.ID, &item.Code, &item.ReportKey, &item.Format, &item.Status, &item.FiltersJSON,
		&item.RequestedByUserID, &item.RequestedByName, &item.RequestedByRole, &item.JobID, &item.FilePath, &item.FileName,
		&item.MimeType, &item.StorageDisk, &item.RowCount, &item.LastError, &item.CompletedAt, &item.CreatedAt, &item.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrExportNotFound
		}
		return nil, err
	}
	return &item, nil
}

func (r *Repository) GetExportFileSource(ctx context.Context, exportID int64) (*ReportExport, error) {
	row := r.db.QueryRowContext(ctx, `SELECT
		re.id, re.status, re.requested_by_user_id, re.file_path, re.file_name, re.mime_type, re.file_blob
		FROM report_exports re
		WHERE re.id = ?`, exportID)
	var item ReportExport
	if err := row.Scan(
		&item.ID,
		&item.Status,
		&item.RequestedByUserID,
		&item.FilePath,
		&item.FileName,
		&item.MimeType,
		&item.FileBlob,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrExportNotFound
		}
		return nil, err
	}
	return &item, nil
}

func (r *Repository) ListExports(ctx context.Context, actor Actor, page, limit int) (*ReportExportListResponse, error) {
	if !canReadOwn(actor) {
		return nil, ErrForbidden
	}
	args := []any{}
	where := `WHERE 1=1`
	if !canReadAll(actor) {
		where += ` AND re.requested_by_user_id = ?`
		args = append(args, actor.ID)
	}
	baseQuery := `SELECT
		re.id, re.code, re.report_key, re.format, re.status, re.filters_json,
		re.requested_by_user_id, COALESCE(u.name, ''), COALESCE(r.code, ''), re.job_id, re.file_path, re.file_name,
		re.mime_type, re.storage_disk, re.row_count, re.last_error, re.completed_at, re.created_at, re.updated_at
		FROM report_exports re
		LEFT JOIN users u ON u.id = re.requested_by_user_id
		LEFT JOIN roles r ON r.id = u.role_id
		` + where + ` ORDER BY re.created_at DESC`
	total, err := countFromSubquery(ctx, r.db, baseQuery, args...)
	if err != nil {
		return nil, err
	}
	query := paginateQuery(baseQuery, page, limit)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]ReportExportResponse, 0)
	for rows.Next() {
		var item ReportExport
		if err := rows.Scan(
			&item.ID, &item.Code, &item.ReportKey, &item.Format, &item.Status, &item.FiltersJSON,
			&item.RequestedByUserID, &item.RequestedByName, &item.RequestedByRole, &item.JobID, &item.FilePath, &item.FileName,
			&item.MimeType, &item.StorageDisk, &item.RowCount, &item.LastError, &item.CompletedAt, &item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, toExportResponse(item))
	}
	return &ReportExportListResponse{
		Items: items,
		Pagination: PaginationMeta{
			Page:  page,
			Limit: limit,
			Total: total,
		},
	}, nil
}

func toExportResponse(item ReportExport) ReportExportResponse {
	resp := ReportExportResponse{
		ID:        item.ID,
		Code:      item.Code,
		ReportKey: item.ReportKey,
		Format:    item.Format,
		Status:    item.Status,
		FileName:  item.FileName.String,
		MimeType:  item.MimeType.String,
		RowCount:  item.RowCount,
		LastError: item.LastError.String,
		CreatedAt: item.CreatedAt,
		UpdatedAt: item.UpdatedAt,
	}
	if item.FiltersJSON.Valid && item.FiltersJSON.String != "" {
		_ = json.Unmarshal([]byte(item.FiltersJSON.String), &resp.Filters)
	}
	if item.RequestedByUserID.Valid {
		resp.RequestedBy = &UserBrief{ID: item.RequestedByUserID.Int64, Name: item.RequestedByName.String}
	}
	if item.JobID.Valid {
		value := item.JobID.Int64
		resp.JobID = &value
	}
	if item.CompletedAt.Valid {
		value := item.CompletedAt.Time
		resp.CompletedAt = &value
	}
	if item.Status == ExportStatusCompleted {
		resp.DownloadURL = fmt.Sprintf("/api/v1/reports/exports/%d/download", item.ID)
	}
	return resp
}

func (r *Repository) MarkExportProcessing(ctx context.Context, exec sqlExecutor, exportID int64) error {
	_, err := exec.ExecContext(ctx, `UPDATE report_exports SET status = ?, updated_at = NOW() WHERE id = ?`, ExportStatusProcessing, exportID)
	return err
}

func (r *Repository) MarkExportCompleted(ctx context.Context, exec sqlExecutor, exportID int64, filePath, fileName, mimeType string, fileBlob []byte, rowCount int64) error {
	_, err := exec.ExecContext(ctx, `UPDATE report_exports
		SET status = ?, file_path = ?, file_name = ?, mime_type = ?, file_blob = ?, row_count = ?, completed_at = NOW(), updated_at = NOW()
		WHERE id = ?`,
		ExportStatusCompleted, nullableString(filePath), fileName, mimeType, fileBlob, rowCount, exportID,
	)
	return err
}

func (r *Repository) MarkExportFailed(ctx context.Context, exec sqlExecutor, exportID int64, jobErr error) error {
	_, err := exec.ExecContext(ctx, `UPDATE report_exports
		SET status = ?, last_error = ?, updated_at = NOW()
		WHERE id = ?`, ExportStatusFailed, jobErr.Error(), exportID)
	return err
}

type sqlExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}
