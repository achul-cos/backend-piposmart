package analytics

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"

	"backend_crm_piposmart/internal/identity"
)

func (r *Repository) TopupRevenueTrend(ctx context.Context, actor identity.User, req QueryRequest, tf ResolvedTimeFilter) (queryData, error) {
	where, args := walletPaymentScopedWhere(actor, req.Filters, tf, "wp.paid_at")
	where += " AND wp.status = 'PAID' AND wp.paid_at IS NOT NULL"
	query := `
		SELECT ` + groupExpr("wp.paid_at", tf.Granularity) + ` AS period, CAST(COALESCE(SUM(wp.amount), 0) AS DOUBLE) AS total
		FROM wallet_payments wp
		WHERE ` + where + `
		GROUP BY period
		ORDER BY period ASC`
	points, table, total, err := r.readTrendRows(ctx, query, args, "topup_revenue")
	if err != nil {
		return queryData{}, err
	}
	return queryData{
		Series: []ChartSeries{{Key: "topup_revenue", Label: "Topup Revenue", Points: points}},
		Table:  table,
		Value:  total,
	}, nil
}

func (r *Repository) TopupTransactionCount(ctx context.Context, actor identity.User, req QueryRequest, tf ResolvedTimeFilter) (queryData, error) {
	where, args := walletPaymentScopedWhere(actor, req.Filters, tf, "wp.paid_at")
	where += " AND wp.status = 'PAID' AND wp.paid_at IS NOT NULL"
	query := `
		SELECT ` + groupExpr("wp.paid_at", tf.Granularity) + ` AS period, COUNT(*) AS total
		FROM wallet_payments wp
		WHERE ` + where + `
		GROUP BY period
		ORDER BY period ASC`
	points, table, total, err := r.readTrendRows(ctx, query, args, "topup_count")
	if err != nil {
		return queryData{}, err
	}
	return queryData{
		Series: []ChartSeries{{Key: "topup_count", Label: "Topup Transaction Count", Points: points}},
		Table:  table,
		Value:  total,
	}, nil
}

func (r *Repository) OwnerBalanceDistribution(ctx context.Context, actor identity.User, req QueryRequest, tf ResolvedTimeFilter) (queryData, error) {
	where, args := walletAccountScopedWhere(actor, req.Filters)
	query := `
		SELECT CAST(COALESCE(wa.balance, 0) AS DOUBLE) AS balance_value
		FROM wallet_accounts wa
		WHERE ` + where
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return queryData{}, err
	}
	defer rows.Close()

	buckets := map[string]float64{
		"0":                   0,
		"1-99.999":            0,
		"100.000-499.999":     0,
		"500.000-999.999":     0,
		"1.000.000-4.999.999": 0,
		"5.000.000+":          0,
	}
	totalOwners := 0.0
	totalBalance := 0.0
	for rows.Next() {
		var balance float64
		if err := rows.Scan(&balance); err != nil {
			return queryData{}, err
		}
		totalOwners++
		totalBalance += balance
		switch {
		case balance <= 0:
			buckets["0"]++
		case balance < 100000:
			buckets["1-99.999"]++
		case balance < 500000:
			buckets["100.000-499.999"]++
		case balance < 1000000:
			buckets["500.000-999.999"]++
		case balance < 5000000:
			buckets["1.000.000-4.999.999"]++
		default:
			buckets["5.000.000+"]++
		}
	}
	if err := rows.Err(); err != nil {
		return queryData{}, err
	}

	labels := []string{"0", "1-99.999", "100.000-499.999", "500.000-999.999", "1.000.000-4.999.999", "5.000.000+"}
	series := ChartSeries{Key: "owner_count", Label: "Owner"}
	table := make([]map[string]any, 0, len(labels))
	for _, label := range labels {
		series.Points = append(series.Points, ChartPoint{X: label, Y: buckets[label]})
		table = append(table, map[string]any{"balance_bucket": label, "owner_count": buckets[label]})
	}
	avgBalance := 0.0
	if totalOwners > 0 {
		avgBalance = round2(totalBalance / totalOwners)
	}
	return queryData{
		Series: []ChartSeries{series},
		Table:  table,
		Extra:  map[string]any{"average_balance": avgBalance, "owner_count": totalOwners},
		Value:  avgBalance,
	}, nil
}

func (r *Repository) TopupUsedVsUnused(ctx context.Context, actor identity.User, req QueryRequest, tf ResolvedTimeFilter) (queryData, error) {
	wherePayments, argsPayments := walletPaymentScopedWhere(actor, req.Filters, tf, "wp.paid_at")
	wherePayments += " AND wp.status = 'PAID' AND wp.paid_at IS NOT NULL"
	var totalTopup float64
	if err := r.db.QueryRowContext(ctx, `SELECT CAST(COALESCE(SUM(wp.amount), 0) AS DOUBLE) FROM wallet_payments wp WHERE `+wherePayments, argsPayments...).Scan(&totalTopup); err != nil {
		return queryData{}, err
	}
	whereWallets, argsWallets := walletAccountScopedWhere(actor, req.Filters)
	var currentBalance float64
	if err := r.db.QueryRowContext(ctx, `SELECT CAST(COALESCE(SUM(wa.balance), 0) AS DOUBLE) FROM wallet_accounts wa WHERE `+whereWallets, argsWallets...).Scan(&currentBalance); err != nil {
		return queryData{}, err
	}
	used := totalTopup - currentBalance
	if used < 0 {
		used = 0
	}
	series := ChartSeries{
		Key:   "topup_usage",
		Label: "Topup Usage",
		Points: []ChartPoint{
			{X: "USED", Y: round2(used)},
			{X: "UNUSED", Y: round2(currentBalance)},
		},
	}
	table := []map[string]any{
		{"status": "USED", "amount": round2(used)},
		{"status": "UNUSED", "amount": round2(currentBalance)},
	}
	rate := 0.0
	if totalTopup > 0 {
		rate = round2((used / totalTopup) * 100)
	}
	return queryData{
		Series: []ChartSeries{series},
		Table:  table,
		Extra:  map[string]any{"total_topup": round2(totalTopup), "usage_rate": rate},
		Value:  round2(currentBalance),
	}, nil
}

func (r *Repository) TopupToSubscribeLag(ctx context.Context, actor identity.User, req QueryRequest, tf ResolvedTimeFilter) (queryData, error) {
	where, args := walletPaymentScopedWhere(actor, req.Filters, tf, "wp.paid_at")
	where += " AND wp.status = 'PAID' AND wp.paid_at IS NOT NULL"
	query := `
		SELECT TIMESTAMPDIFF(DAY, wp.paid_at, (
			SELECT MIN(so.purchased_at)
			FROM subscription_orders so
			WHERE so.deleted_at IS NULL
				AND so.owner_id = wp.owner_id
				AND so.purchased_at >= wp.paid_at
		)) AS lag_days
		FROM wallet_payments wp
		WHERE ` + where
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return queryData{}, err
	}
	defer rows.Close()

	buckets := map[string]float64{"0-1": 0, "2-7": 0, "8-30": 0, "31-60": 0, "61-90": 0, "90+": 0, "NO_SUBSCRIBE": 0}
	totalDays := 0.0
	totalCount := 0.0
	for rows.Next() {
		var lag sql.NullInt64
		if err := rows.Scan(&lag); err != nil {
			return queryData{}, err
		}
		if !lag.Valid {
			buckets["NO_SUBSCRIBE"]++
			continue
		}
		value := float64(lag.Int64)
		totalCount++
		totalDays += value
		switch {
		case value <= 1:
			buckets["0-1"]++
		case value <= 7:
			buckets["2-7"]++
		case value <= 30:
			buckets["8-30"]++
		case value <= 60:
			buckets["31-60"]++
		case value <= 90:
			buckets["61-90"]++
		default:
			buckets["90+"]++
		}
	}
	if err := rows.Err(); err != nil {
		return queryData{}, err
	}
	labels := []string{"0-1", "2-7", "8-30", "31-60", "61-90", "90+", "NO_SUBSCRIBE"}
	series := ChartSeries{Key: "payment_count", Label: "Payment"}
	table := make([]map[string]any, 0, len(labels))
	for _, label := range labels {
		series.Points = append(series.Points, ChartPoint{X: label, Y: buckets[label]})
		table = append(table, map[string]any{"lag_bucket": label, "payment_count": buckets[label]})
	}
	avgLag := 0.0
	if totalCount > 0 {
		avgLag = round2(totalDays / totalCount)
	}
	return queryData{
		Series: []ChartSeries{series},
		Table:  table,
		Extra:  map[string]any{"average_lag_days": avgLag},
		Value:  avgLag,
	}, nil
}

func (r *Repository) ZeroVsNonZeroBalance(ctx context.Context, actor identity.User, req QueryRequest, tf ResolvedTimeFilter) (queryData, error) {
	where, args := walletAccountScopedWhere(actor, req.Filters)
	query := `
		SELECT
			SUM(CASE WHEN COALESCE(wa.balance, 0) <= 0 THEN 1 ELSE 0 END) AS zero_count,
			SUM(CASE WHEN COALESCE(wa.balance, 0) > 0 THEN 1 ELSE 0 END) AS nonzero_count
		FROM wallet_accounts wa
		WHERE ` + where
	row := r.db.QueryRowContext(ctx, query, args...)
	var zeroCount, nonzeroCount float64
	if err := row.Scan(&zeroCount, &nonzeroCount); err != nil {
		return queryData{}, err
	}
	return queryData{
		Series: []ChartSeries{{
			Key:   "owner_count",
			Label: "Owner",
			Points: []ChartPoint{
				{X: "ZERO_BALANCE", Y: zeroCount},
				{X: "NONZERO_BALANCE", Y: nonzeroCount},
			},
		}},
		Table: []map[string]any{
			{"status": "ZERO_BALANCE", "owner_count": zeroCount},
			{"status": "NONZERO_BALANCE", "owner_count": nonzeroCount},
		},
		Value: nonzeroCount,
	}, nil
}

func (r *Repository) ActiveSubscriptionTrend(ctx context.Context, actor identity.User, req QueryRequest, tf ResolvedTimeFilter) (queryData, error) {
	months := monthRange(tf)
	series := ChartSeries{Key: "active_subscription_count", Label: "Active Subscription"}
	table := make([]map[string]any, 0, len(months))
	total := 0.0
	for _, month := range months {
		monthStart := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, time.UTC)
		monthEnd := monthStart.AddDate(0, 1, 0).Add(-time.Nanosecond)
		count, err := r.countActiveSubscriptionsAtDate(ctx, actor, req.Filters, monthEnd)
		if err != nil {
			return queryData{}, err
		}
		period := monthStart.Format("2006-01")
		series.Points = append(series.Points, ChartPoint{X: period, Y: count})
		table = append(table, map[string]any{"period": period, "active_subscription_count": count})
		total += count
	}
	return queryData{Series: []ChartSeries{series}, Table: table, Value: total}, nil
}

func (r *Repository) ActivationVsExpiryTrend(ctx context.Context, actor identity.User, req QueryRequest, tf ResolvedTimeFilter) (queryData, error) {
	whereActivation, argsActivation := subscriptionScopedWhere(actor, req.Filters, tf, "s.active_from")
	queryActivation := `
		SELECT ` + groupExpr("s.active_from", tf.Granularity) + ` AS period, COUNT(*) AS total
		FROM subscriptions s
		WHERE ` + whereActivation + `
		GROUP BY period
		ORDER BY period ASC`
	activationPoints, activationTable, activationTotal, err := r.readTrendRows(ctx, queryActivation, argsActivation, "activation_count")
	if err != nil {
		return queryData{}, err
	}
	whereExpiry, argsExpiry := subscriptionScopedWhere(actor, req.Filters, tf, "s.active_until")
	queryExpiry := `
		SELECT ` + groupExpr("s.active_until", tf.Granularity) + ` AS period, COUNT(*) AS total
		FROM subscriptions s
		WHERE ` + whereExpiry + `
		GROUP BY period
		ORDER BY period ASC`
	expiryPoints, _, expiryTotal, err := r.readTrendRows(ctx, queryExpiry, argsExpiry, "expiry_count")
	if err != nil {
		return queryData{}, err
	}
	table := mergeTrendRowsByPeriod(activationTable, "activation_count", expiryPoints, "expiry_count")
	return queryData{
		Series: []ChartSeries{
			{Key: "activation_count", Label: "Activation", Points: activationPoints},
			{Key: "expiry_count", Label: "Expiry", Points: expiryPoints},
		},
		Table: table,
		Value: activationTotal - expiryTotal,
	}, nil
}

func (r *Repository) RenewalRate(ctx context.Context, actor identity.User, req QueryRequest, tf ResolvedTimeFilter) (queryData, error) {
	months := monthRange(tf)
	series := ChartSeries{Key: "renewal_rate", Label: "Renewal Rate (%)"}
	table := make([]map[string]any, 0, len(months))
	totalRate := 0.0
	for _, month := range months {
		start := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, time.UTC)
		end := start.AddDate(0, 1, 0)
		expired, renewed, err := r.renewalCountsForPeriod(ctx, actor, req.Filters, start, end)
		if err != nil {
			return queryData{}, err
		}
		rate := 0.0
		if expired > 0 {
			rate = round2((renewed / expired) * 100)
		}
		period := start.Format("2006-01")
		series.Points = append(series.Points, ChartPoint{X: period, Y: rate})
		table = append(table, map[string]any{
			"period":                  period,
			"expired_candidate_count": expired,
			"renewed_count":           renewed,
			"renewal_rate":            rate,
		})
		totalRate += rate
	}
	avgRate := 0.0
	if len(months) > 0 {
		avgRate = round2(totalRate / float64(len(months)))
	}
	return queryData{Series: []ChartSeries{series}, Table: table, Value: avgRate}, nil
}

func (r *Repository) ExpiryForecast(ctx context.Context, actor identity.User, req QueryRequest, tf ResolvedTimeFilter) (queryData, error) {
	baseDate := tf.End.Add(-time.Nanosecond)
	count30, count60, count90, err := r.expiryForecastCounts(ctx, actor, req.Filters, baseDate)
	if err != nil {
		return queryData{}, err
	}
	series := ChartSeries{
		Key:   "expiry_count",
		Label: "Expiry Forecast",
		Points: []ChartPoint{
			{X: "1-30 Hari", Y: count30},
			{X: "31-60 Hari", Y: count60},
			{X: "61-90 Hari", Y: count90},
		},
	}
	table := []map[string]any{
		{"bucket": "1-30 Hari", "subscription_count": count30},
		{"bucket": "31-60 Hari", "subscription_count": count60},
		{"bucket": "61-90 Hari", "subscription_count": count90},
	}
	return queryData{
		Series: []ChartSeries{series},
		Table:  table,
		Extra:  map[string]any{"base_date": baseDate.Format("2006-01-02")},
		Value:  count30,
	}, nil
}

func (r *Repository) SubscriptionPackageMix(ctx context.Context, actor identity.User, req QueryRequest, tf ResolvedTimeFilter) (queryData, error) {
	baseDate := tf.End.Add(-time.Nanosecond)
	query := `
		SELECT COALESCE(sp.name, 'Tanpa Paket') AS label, COUNT(*) AS total
		FROM subscriptions s
		LEFT JOIN subscription_packages sp ON sp.id = s.package_id
		WHERE ` + subscriptionSnapshotWhere(actor, req.Filters, "s.owner_id", baseDate) + `
		GROUP BY label
		ORDER BY total DESC, label ASC`
	args := subscriptionSnapshotArgs(actor, req.Filters, baseDate)
	return r.queryCategoryCount(ctx, query, args, "package_name", "subscription_count")
}

func (r *Repository) SubscriptionTenureMix(ctx context.Context, actor identity.User, req QueryRequest, tf ResolvedTimeFilter) (queryData, error) {
	baseDate := tf.End.Add(-time.Nanosecond)
	query := `
		SELECT CONCAT(COALESCE(so.tenure_months, 0), ' bulan') AS label, COUNT(*) AS total
		FROM subscriptions s
		LEFT JOIN subscription_orders so ON so.id = s.order_id
		WHERE ` + subscriptionSnapshotWhere(actor, req.Filters, "s.owner_id", baseDate) + `
		GROUP BY COALESCE(so.tenure_months, 0), CONCAT(COALESCE(so.tenure_months, 0), ' bulan')
		ORDER BY COALESCE(so.tenure_months, 0) ASC`
	args := subscriptionSnapshotArgs(actor, req.Filters, baseDate)
	return r.queryCategoryCount(ctx, query, args, "tenure_label", "subscription_count")
}

func (r *Repository) DaysRemainingHistogram(ctx context.Context, actor identity.User, req QueryRequest, tf ResolvedTimeFilter) (queryData, error) {
	baseDate := tf.End.Add(-time.Nanosecond)
	query := `
		SELECT DATEDIFF(s.active_until, ?) AS remaining_days
		FROM subscriptions s
		WHERE ` + subscriptionSnapshotWhere(actor, req.Filters, "s.owner_id", baseDate)
	args := append([]any{baseDate.Format("2006-01-02")}, subscriptionSnapshotArgs(actor, req.Filters, baseDate)...)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return queryData{}, err
	}
	defer rows.Close()
	buckets := map[string]float64{"0-7": 0, "8-14": 0, "15-30": 0, "31-60": 0, "61-90": 0, "90+": 0}
	totalRemaining := 0.0
	totalCount := 0.0
	for rows.Next() {
		var remaining int
		if err := rows.Scan(&remaining); err != nil {
			return queryData{}, err
		}
		totalCount++
		totalRemaining += float64(remaining)
		switch {
		case remaining <= 7:
			buckets["0-7"]++
		case remaining <= 14:
			buckets["8-14"]++
		case remaining <= 30:
			buckets["15-30"]++
		case remaining <= 60:
			buckets["31-60"]++
		case remaining <= 90:
			buckets["61-90"]++
		default:
			buckets["90+"]++
		}
	}
	if err := rows.Err(); err != nil {
		return queryData{}, err
	}
	labels := []string{"0-7", "8-14", "15-30", "31-60", "61-90", "90+"}
	series := ChartSeries{Key: "subscription_count", Label: "Subscription"}
	table := make([]map[string]any, 0, len(labels))
	for _, label := range labels {
		series.Points = append(series.Points, ChartPoint{X: label, Y: buckets[label]})
		table = append(table, map[string]any{"days_remaining_bucket": label, "subscription_count": buckets[label]})
	}
	avgRemaining := 0.0
	if totalCount > 0 {
		avgRemaining = round2(totalRemaining / totalCount)
	}
	return queryData{Series: []ChartSeries{series}, Table: table, Extra: map[string]any{"average_days_remaining": avgRemaining}, Value: avgRemaining}, nil
}

func (r *Repository) ChurnBucketTrend(ctx context.Context, actor identity.User, req QueryRequest, tf ResolvedTimeFilter) (queryData, error) {
	months := monthRange(tf)
	expiredSeries := ChartSeries{Key: "expired_count", Label: "Expired"}
	notSubscribeSeries := ChartSeries{Key: "not_subscribe_count", Label: "Not Subscribe"}
	table := make([]map[string]any, 0, len(months))
	total := 0.0
	for _, month := range months {
		counts, _, err := r.outletSubscriptionStatusCounts(ctx, actor, req.Filters, month)
		if err != nil {
			return queryData{}, err
		}
		period := month.Format("2006-01")
		expiredSeries.Points = append(expiredSeries.Points, ChartPoint{X: period, Y: counts["EXPIRED"]})
		notSubscribeSeries.Points = append(notSubscribeSeries.Points, ChartPoint{X: period, Y: counts["NOT_SUBSCRIBE"]})
		table = append(table, map[string]any{
			"period":              period,
			"expired_count":       counts["EXPIRED"],
			"not_subscribe_count": counts["NOT_SUBSCRIBE"],
		})
		total += counts["EXPIRED"] + counts["NOT_SUBSCRIBE"]
	}
	return queryData{Series: []ChartSeries{expiredSeries, notSubscribeSeries}, Table: table, Value: total}, nil
}

func (r *Repository) ReconciliationSuccessRate(ctx context.Context, actor identity.User, req QueryRequest, tf ResolvedTimeFilter) (queryData, error) {
	where, args := reconciliationScopedWhere(actor, req.Filters, tf, "sr.created_at")
	query := `
		SELECT ` + groupExpr("sr.created_at", tf.Granularity) + ` AS period,
			COUNT(*) AS total_count,
			SUM(CASE WHEN sr.status = 'CONFIRMED' THEN 1 ELSE 0 END) AS confirmed_count
		FROM subscription_reconciliations sr
		WHERE ` + where + `
		GROUP BY period
		ORDER BY period ASC`
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return queryData{}, err
	}
	defer rows.Close()
	series := ChartSeries{Key: "success_rate", Label: "Success Rate (%)"}
	table := make([]map[string]any, 0)
	totalRate := 0.0
	totalPeriods := 0.0
	for rows.Next() {
		var period string
		var totalCount, confirmedCount float64
		if err := rows.Scan(&period, &totalCount, &confirmedCount); err != nil {
			return queryData{}, err
		}
		rate := 0.0
		if totalCount > 0 {
			rate = round2((confirmedCount / totalCount) * 100)
		}
		series.Points = append(series.Points, ChartPoint{X: period, Y: rate})
		table = append(table, map[string]any{"period": period, "total_count": totalCount, "confirmed_count": confirmedCount, "success_rate": rate})
		totalRate += rate
		totalPeriods++
	}
	value := 0.0
	if totalPeriods > 0 {
		value = round2(totalRate / totalPeriods)
	}
	return queryData{Series: []ChartSeries{series}, Table: table, Value: value}, rows.Err()
}

func (r *Repository) ReconciliationIssueByType(ctx context.Context, actor identity.User, req QueryRequest, tf ResolvedTimeFilter) (queryData, error) {
	where, args := reconciliationIssueScopedWhere(actor, req.Filters, tf, "ri.detected_at")
	query := `
		SELECT ri.issue_type AS label, COUNT(*) AS total
		FROM reconciliation_issues ri
		WHERE ` + where + `
		GROUP BY ri.issue_type
		ORDER BY total DESC, label ASC`
	return r.queryCategoryCount(ctx, query, args, "issue_type", "issue_count")
}

func (r *Repository) ReconciliationIssueAging(ctx context.Context, actor identity.User, req QueryRequest, tf ResolvedTimeFilter) (queryData, error) {
	where, args := reconciliationIssueScopedWhere(actor, req.Filters, tf, "ri.detected_at")
	query := `
		SELECT TIMESTAMPDIFF(DAY, ri.detected_at, COALESCE(ri.resolved_at, ?)) AS aging_days
		FROM reconciliation_issues ri
		WHERE ` + where
	args = append([]any{tf.End}, args...)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return queryData{}, err
	}
	defer rows.Close()
	buckets := map[string]float64{"0-3": 0, "4-7": 0, "8-14": 0, "15-30": 0, "31-60": 0, "60+": 0}
	totalAging := 0.0
	totalCount := 0.0
	for rows.Next() {
		var aging float64
		if err := rows.Scan(&aging); err != nil {
			return queryData{}, err
		}
		totalCount++
		totalAging += aging
		switch {
		case aging <= 3:
			buckets["0-3"]++
		case aging <= 7:
			buckets["4-7"]++
		case aging <= 14:
			buckets["8-14"]++
		case aging <= 30:
			buckets["15-30"]++
		case aging <= 60:
			buckets["31-60"]++
		default:
			buckets["60+"]++
		}
	}
	if err := rows.Err(); err != nil {
		return queryData{}, err
	}
	labels := []string{"0-3", "4-7", "8-14", "15-30", "31-60", "60+"}
	series := ChartSeries{Key: "issue_count", Label: "Issue"}
	table := make([]map[string]any, 0, len(labels))
	for _, label := range labels {
		series.Points = append(series.Points, ChartPoint{X: label, Y: buckets[label]})
		table = append(table, map[string]any{"aging_bucket": label, "issue_count": buckets[label]})
	}
	avgAging := 0.0
	if totalCount > 0 {
		avgAging = round2(totalAging / totalCount)
	}
	return queryData{Series: []ChartSeries{series}, Table: table, Extra: map[string]any{"average_aging_days": avgAging}, Value: avgAging}, nil
}

func (r *Repository) ReconciliationAutoVsManual(ctx context.Context, actor identity.User, req QueryRequest, tf ResolvedTimeFilter) (queryData, error) {
	where, args := reconciliationScopedWhere(actor, req.Filters, tf, "sr.created_at")
	query := `
		SELECT sr.match_type AS label, COUNT(*) AS total
		FROM subscription_reconciliations sr
		WHERE ` + where + `
		GROUP BY sr.match_type
		ORDER BY total DESC, label ASC`
	return r.queryCategoryCount(ctx, query, args, "match_type", "reconciliation_count")
}

func (r *Repository) HangingTransactionTrend(ctx context.Context, actor identity.User, req QueryRequest, tf ResolvedTimeFilter) (queryData, error) {
	where, args := reconciliationIssueScopedWhere(actor, req.Filters, tf, "ri.detected_at")
	where += " AND ri.status = 'OPEN'"
	query := `
		SELECT ` + groupExpr("ri.detected_at", tf.Granularity) + ` AS period, COUNT(*) AS total
		FROM reconciliation_issues ri
		WHERE ` + where + `
		GROUP BY period
		ORDER BY period ASC`
	points, table, total, err := r.readTrendRows(ctx, query, args, "hanging_issue_count")
	if err != nil {
		return queryData{}, err
	}
	return queryData{
		Series: []ChartSeries{{Key: "hanging_issue_count", Label: "Hanging Transaction", Points: points}},
		Table:  table,
		Value:  total,
	}, nil
}

func (r *Repository) RevenueVsClosingPeriodCompare(ctx context.Context, actor identity.User, req QueryRequest, tf ResolvedTimeFilter) (queryData, error) {
	whereTopup, argsTopup := walletPaymentScopedWhere(actor, req.Filters, tf, "wp.paid_at")
	whereTopup += " AND wp.status = 'PAID' AND wp.paid_at IS NOT NULL"
	queryTopup := `
		SELECT ` + groupExpr("wp.paid_at", tf.Granularity) + ` AS period, CAST(COALESCE(SUM(wp.amount), 0) AS DOUBLE) AS total
		FROM wallet_payments wp
		WHERE ` + whereTopup + `
		GROUP BY period
		ORDER BY period ASC`
	topupPoints, topupTable, _, err := r.readTrendRows(ctx, queryTopup, argsTopup, "topup_revenue")
	if err != nil {
		return queryData{}, err
	}

	whereClosing, argsClosing := closingScopedWhere(actor, req.Filters, tf, "sc.confirmed_at")
	whereClosing += " AND sc.status = 'CONFIRMED' AND sc.confirmed_at IS NOT NULL"
	queryClosing := `
		SELECT ` + groupExpr("sc.confirmed_at", tf.Granularity) + ` AS period,
			CAST(COALESCE(SUM(` + closingRevenueExpr + `), 0) AS DOUBLE) AS total
		FROM sales_closings sc
		WHERE ` + whereClosing + `
		GROUP BY period
		ORDER BY period ASC`
	closingPoints, _, _, err := r.readTrendRows(ctx, queryClosing, argsClosing, "closing_revenue_snapshot")
	if err != nil {
		return queryData{}, err
	}
	table := mergeTrendRowsByPeriod(topupTable, "topup_revenue", closingPoints, "closing_revenue_snapshot")
	value := 0.0
	for _, row := range table {
		value += parseFloatFromAny(row["topup_revenue"])
	}
	return queryData{
		Series: []ChartSeries{
			{Key: "topup_revenue", Label: "Topup Revenue", Points: topupPoints},
			{Key: "closing_revenue_snapshot", Label: "Closing Revenue Snapshot", Points: closingPoints},
		},
		Table: table,
		Value: value,
	}, nil
}

func walletAccountScopedWhere(actor identity.User, filters FilterRequest) (string, []any) {
	where := []string{"wa.deleted_at IS NULL"}
	args := []any{}
	visibility, visibilityArgs := ownerVisibilityWhere(actor, "wa.owner_id")
	where = append(where, visibility)
	args = append(args, visibilityArgs...)
	appendIntFilter(&where, &args, "wa.owner_id", filters.OwnerIDs)
	appendOwnerLeadScopedFilters(&where, &args, filters, "wa.owner_id")
	return strings.Join(where, " AND "), args
}

func walletPaymentScopedWhere(actor identity.User, filters FilterRequest, tf ResolvedTimeFilter, timeColumn string) (string, []any) {
	where := []string{"wp.deleted_at IS NULL"}
	args := []any{}
	visibility, visibilityArgs := ownerVisibilityWhere(actor, "wp.owner_id")
	where = append(where, visibility)
	args = append(args, visibilityArgs...)
	if timeColumn != "" {
		where = append(where, timeColumn+" >= ?", timeColumn+" < ?")
		args = append(args, tf.Start, tf.End)
	}
	appendIntFilter(&where, &args, "wp.owner_id", filters.OwnerIDs)
	appendOwnerLeadScopedFilters(&where, &args, filters, "wp.owner_id")
	return strings.Join(where, " AND "), args
}

func subscriptionScopedWhere(actor identity.User, filters FilterRequest, tf ResolvedTimeFilter, timeColumn string) (string, []any) {
	where := []string{"s.deleted_at IS NULL"}
	args := []any{}
	visibility, visibilityArgs := ownerVisibilityWhere(actor, "s.owner_id")
	where = append(where, visibility)
	args = append(args, visibilityArgs...)
	where = append(where, timeColumn+" >= ?", timeColumn+" < ?")
	args = append(args, tf.Start.Format("2006-01-02"), tf.End.Format("2006-01-02"))
	appendStringFilter(&where, &args, "s.status", filters.Status)
	appendIntFilter(&where, &args, "s.owner_id", filters.OwnerIDs)
	appendIntFilter(&where, &args, "s.outlet_id", filters.OutletIDs)
	appendOwnerLeadScopedFilters(&where, &args, filters, "s.owner_id")
	return strings.Join(where, " AND "), args
}

func subscriptionSnapshotWhere(actor identity.User, filters FilterRequest, ownerColumn string, baseDate time.Time) string {
	where := []string{"s.deleted_at IS NULL", "s.active_from <= ?", "s.active_until >= ?"}
	visibility, _ := ownerVisibilityWhere(actor, ownerColumn)
	where = append(where, visibility)
	if len(filters.OwnerIDs) > 0 {
		where = append(where, "s.owner_id IN ("+placeholders(len(filters.OwnerIDs))+")")
	}
	if len(filters.OutletIDs) > 0 {
		where = append(where, "s.outlet_id IN ("+placeholders(len(filters.OutletIDs))+")")
	}
	_ = baseDate
	return strings.Join(where, " AND ")
}

func subscriptionSnapshotArgs(actor identity.User, filters FilterRequest, baseDate time.Time) []any {
	args := []any{}
	baseArgs := []any{}
	_, visibilityArgs := ownerVisibilityWhere(actor, "s.owner_id")
	baseArgs = append(baseArgs, visibilityArgs...)
	args = append(args, baseDate.Format("2006-01-02"), baseDate.Format("2006-01-02"))
	if len(baseArgs) > 0 {
		args = append(args, baseArgs...)
	}
	for _, id := range filters.OwnerIDs {
		args = append(args, id)
	}
	for _, id := range filters.OutletIDs {
		args = append(args, id)
	}
	return args
}

func reconciliationScopedWhere(actor identity.User, filters FilterRequest, tf ResolvedTimeFilter, timeColumn string) (string, []any) {
	where := []string{"sr.deleted_at IS NULL"}
	args := []any{}
	visibility, visibilityArgs := ownerVisibilityWhere(actor, "sr.owner_id")
	where = append(where, visibility)
	args = append(args, visibilityArgs...)
	where = append(where, timeColumn+" >= ?", timeColumn+" < ?")
	args = append(args, tf.Start, tf.End)
	appendStringFilter(&where, &args, "sr.status", filters.Status)
	appendIntFilter(&where, &args, "sr.owner_id", filters.OwnerIDs)
	appendOwnerLeadScopedFilters(&where, &args, filters, "sr.owner_id")
	return strings.Join(where, " AND "), args
}

func reconciliationIssueScopedWhere(actor identity.User, filters FilterRequest, tf ResolvedTimeFilter, timeColumn string) (string, []any) {
	where := []string{"ri.deleted_at IS NULL"}
	args := []any{}
	visibility, visibilityArgs := ownerVisibilityWhere(actor, "ri.owner_id")
	where = append(where, visibility)
	args = append(args, visibilityArgs...)
	where = append(where, timeColumn+" >= ?", timeColumn+" < ?")
	args = append(args, tf.Start, tf.End)
	appendStringFilter(&where, &args, "ri.status", filters.Status)
	appendIntFilter(&where, &args, "ri.owner_id", filters.OwnerIDs)
	appendOwnerLeadScopedFilters(&where, &args, filters, "ri.owner_id")
	return strings.Join(where, " AND "), args
}

func (r *Repository) countActiveSubscriptionsAtDate(ctx context.Context, actor identity.User, filters FilterRequest, at time.Time) (float64, error) {
	query := `
		SELECT COUNT(*)
		FROM subscriptions s
		WHERE s.deleted_at IS NULL
			AND s.active_from <= ?
			AND s.active_until >= ?
			AND ` + ownerVisibilityCondition(actor, "s.owner_id", filters)
	args := []any{at.Format("2006-01-02"), at.Format("2006-01-02")}
	args = append(args, ownerVisibilityArgs(actor, filters)...)
	var count float64
	if err := r.db.QueryRowContext(ctx, query, args...).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func (r *Repository) renewalCountsForPeriod(ctx context.Context, actor identity.User, filters FilterRequest, start, end time.Time) (float64, float64, error) {
	query := `
		SELECT
			COUNT(*) AS expired_count,
			COALESCE(SUM(CASE WHEN EXISTS (
				SELECT 1 FROM subscriptions s2
				WHERE s2.deleted_at IS NULL
					AND s2.owner_id = s.owner_id
					AND COALESCE(s2.outlet_id, 0) = COALESCE(s.outlet_id, 0)
					AND s2.active_from > s.active_until
					AND s2.active_from <= DATE_ADD(s.active_until, INTERVAL 30 DAY)
			) THEN 1 ELSE 0 END), 0) AS renewed_count
		FROM subscriptions s
		WHERE s.deleted_at IS NULL
			AND s.active_until >= ? AND s.active_until < ?
			AND ` + ownerVisibilityCondition(actor, "s.owner_id", filters)
	args := []any{start.Format("2006-01-02"), end.Format("2006-01-02")}
	args = append(args, ownerVisibilityArgs(actor, filters)...)
	var expiredCount, renewedCount float64
	if err := r.db.QueryRowContext(ctx, query, args...).Scan(&expiredCount, &renewedCount); err != nil {
		return 0, 0, err
	}
	return expiredCount, renewedCount, nil
}

func (r *Repository) expiryForecastCounts(ctx context.Context, actor identity.User, filters FilterRequest, baseDate time.Time) (float64, float64, float64, error) {
	query := `
		SELECT
			SUM(CASE WHEN DATEDIFF(s.active_until, ?) BETWEEN 1 AND 30 THEN 1 ELSE 0 END) AS bucket_30,
			SUM(CASE WHEN DATEDIFF(s.active_until, ?) BETWEEN 31 AND 60 THEN 1 ELSE 0 END) AS bucket_60,
			SUM(CASE WHEN DATEDIFF(s.active_until, ?) BETWEEN 61 AND 90 THEN 1 ELSE 0 END) AS bucket_90
		FROM subscriptions s
		WHERE s.deleted_at IS NULL
			AND s.active_until > ?
			AND ` + ownerVisibilityCondition(actor, "s.owner_id", filters)
	args := []any{
		baseDate.Format("2006-01-02"),
		baseDate.Format("2006-01-02"),
		baseDate.Format("2006-01-02"),
		baseDate.Format("2006-01-02"),
	}
	args = append(args, ownerVisibilityArgs(actor, filters)...)
	var count30, count60, count90 float64
	if err := r.db.QueryRowContext(ctx, query, args...).Scan(&count30, &count60, &count90); err != nil {
		return 0, 0, 0, err
	}
	return count30, count60, count90, nil
}

func ownerVisibilityCondition(actor identity.User, ownerColumn string, filters FilterRequest) string {
	where := []string{}
	visibility, _ := ownerVisibilityWhere(actor, ownerColumn)
	where = append(where, visibility)
	if len(filters.OwnerIDs) > 0 {
		where = append(where, ownerColumn+" IN ("+placeholders(len(filters.OwnerIDs))+")")
	}
	if len(filters.SupervisorIDs) > 0 {
		where = append(where, `EXISTS (
			SELECT 1 FROM customer_leads cl
			WHERE cl.owner_id = `+ownerColumn+`
				AND cl.deleted_at IS NULL
				AND cl.supervisor_id IN (`+placeholders(len(filters.SupervisorIDs))+`)
		)`)
	}
	if len(filters.SalesIDs) > 0 {
		where = append(where, `EXISTS (
			SELECT 1 FROM customer_leads cl
			WHERE cl.owner_id = `+ownerColumn+`
				AND cl.deleted_at IS NULL
				AND cl.active_sales_id IN (`+placeholders(len(filters.SalesIDs))+`)
		)`)
	}
	if len(filters.OutletIDs) > 0 {
		where = append(where, `EXISTS (
			SELECT 1 FROM outlets ot
			WHERE ot.id IN (`+placeholders(len(filters.OutletIDs))+`)
				AND ot.deleted_at IS NULL
				AND ot.owner_id = `+ownerColumn+`
		)`)
	}
	return strings.Join(where, " AND ")
}

func ownerVisibilityArgs(actor identity.User, filters FilterRequest) []any {
	args := []any{}
	_, visibilityArgs := ownerVisibilityWhere(actor, "ignored")
	args = append(args, visibilityArgs...)
	for _, id := range filters.OwnerIDs {
		args = append(args, id)
	}
	for _, id := range filters.SupervisorIDs {
		args = append(args, id)
	}
	for _, id := range filters.SalesIDs {
		args = append(args, id)
	}
	for _, id := range filters.OutletIDs {
		args = append(args, id)
	}
	return args
}

func mergeTrendRowsByPeriod(primary []map[string]any, primaryKey string, secondary []ChartPoint, secondaryKey string) []map[string]any {
	lookup := map[string]float64{}
	for _, point := range secondary {
		lookup[fmt.Sprint(point.X)] = point.Y
	}
	for _, row := range primary {
		key := fmt.Sprint(row["period"])
		row[secondaryKey] = lookup[key]
		if _, ok := row[primaryKey]; !ok {
			row[primaryKey] = 0.0
		}
	}
	primaryPeriods := map[string]struct{}{}
	for _, row := range primary {
		primaryPeriods[fmt.Sprint(row["period"])] = struct{}{}
	}
	for _, point := range secondary {
		key := fmt.Sprint(point.X)
		if _, ok := primaryPeriods[key]; ok {
			continue
		}
		primary = append(primary, map[string]any{
			"period":     key,
			primaryKey:   0.0,
			secondaryKey: point.Y,
		})
	}
	sort.Slice(primary, func(i, j int) bool {
		return fmt.Sprint(primary[i]["period"]) < fmt.Sprint(primary[j]["period"])
	})
	return primary
}
