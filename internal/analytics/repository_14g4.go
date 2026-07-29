package analytics

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"backend_crm_piposmart/internal/identity"
)

func (r *Repository) PartnerGrowthTrend(ctx context.Context, actor identity.User, req QueryRequest, tf ResolvedTimeFilter) (queryData, error) {
	where, args := partnerScopedWhere(actor, req.Filters, tf, "p.created_at")
	query := `
		SELECT ` + groupExpr("p.created_at", tf.Granularity) + ` AS period, COUNT(*) AS total
		FROM partners p
		WHERE ` + where + `
		GROUP BY period
		ORDER BY period ASC`
	points, table, total, err := r.readTrendRows(ctx, query, args, "partner_count")
	if err != nil {
		return queryData{}, err
	}
	return queryData{
		Series: []ChartSeries{{Key: "partner_count", Label: "Partner Baru", Points: points}},
		Table:  table,
		Value:  total,
	}, nil
}

func (r *Repository) PartnerTypeDistribution(ctx context.Context, actor identity.User, req QueryRequest, tf ResolvedTimeFilter) (queryData, error) {
	where, args := partnerScopedWhere(actor, req.Filters, tf, "p.created_at")
	query := `
		SELECT pt.name AS label, COUNT(*) AS total
		FROM partners p
		JOIN partner_types pt ON pt.id = p.partner_type_id
		WHERE ` + where + `
		GROUP BY pt.id, pt.name
		ORDER BY total DESC, label ASC`
	return r.queryCategoryCount(ctx, query, args, "partner_type_name", "partner_count")
}

func (r *Repository) ReferralCountPerPartner(ctx context.Context, actor identity.User, req QueryRequest, tf ResolvedTimeFilter) (queryData, error) {
	where, args := partnerReferralScopedWhere(actor, req.Filters, tf, "pr.referral_date")
	query := `
		SELECT p.name AS label, COUNT(*) AS total
		FROM partner_referrals pr
		JOIN partners p ON p.id = pr.partner_id
		WHERE ` + where + `
		GROUP BY p.id, p.name
		ORDER BY total DESC, label ASC`
	return r.queryCategoryCount(ctx, query, args, "partner_name", "referral_count")
}

func (r *Repository) ReferralConversionPerPartner(ctx context.Context, actor identity.User, req QueryRequest, tf ResolvedTimeFilter) (queryData, error) {
	where, args := partnerReferralScopedWhere(actor, req.Filters, tf, "pr.referral_date")
	query := `
		SELECT
			p.id,
			p.name,
			COUNT(DISTINCT pr.id) AS referral_count,
			COUNT(DISTINCT pc.id) AS closing_count
		FROM partner_referrals pr
		JOIN partners p ON p.id = pr.partner_id
		LEFT JOIN partner_commissions pc ON pc.referral_id = pr.id AND pc.status <> 'CANCELLED'
		WHERE ` + where + `
		GROUP BY p.id, p.name
		ORDER BY closing_count DESC, referral_count DESC, p.name ASC`
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return queryData{}, err
	}
	defer rows.Close()

	series := ChartSeries{Key: "conversion_rate", Label: "Conversion Rate (%)"}
	table := make([]map[string]any, 0)
	total := 0.0
	for rows.Next() {
		var partnerID int64
		var name string
		var referralCount, closingCount float64
		if err := rows.Scan(&partnerID, &name, &referralCount, &closingCount); err != nil {
			return queryData{}, err
		}
		rate := 0.0
		if referralCount > 0 {
			rate = round2((closingCount / referralCount) * 100)
		}
		series.Points = append(series.Points, ChartPoint{X: name, Y: rate})
		table = append(table, map[string]any{
			"partner_id":      partnerID,
			"partner_name":    name,
			"referral_count":  referralCount,
			"closing_count":   closingCount,
			"conversion_rate": rate,
		})
		total += rate
	}
	value := 0.0
	if len(table) > 0 {
		value = round2(total / float64(len(table)))
	}
	return queryData{Series: []ChartSeries{series}, Table: table, Value: value}, rows.Err()
}

func (r *Repository) PartnerPICWorkload(ctx context.Context, actor identity.User, req QueryRequest, tf ResolvedTimeFilter) (queryData, error) {
	baseDate := tf.End
	query := `
		SELECT COALESCE(u.name, 'Tanpa PIC') AS label,
			SUM(CASE WHEN p.status = 'ACTIVE' THEN 1 ELSE 0 END) AS active_partner_count,
			SUM(CASE WHEN p.status <> 'ACTIVE' THEN 1 ELSE 0 END) AS inactive_partner_count
		FROM partner_assignments pa
		JOIN partners p ON p.id = pa.partner_id
		LEFT JOIN users u ON u.id = pa.user_id
		WHERE ` + partnerAssignmentSnapshotWhere(actor, req.Filters, baseDate) + `
		GROUP BY pa.user_id, u.name
		ORDER BY active_partner_count DESC, label ASC`
	args := partnerAssignmentSnapshotArgs(actor, req.Filters, baseDate)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return queryData{}, err
	}
	defer rows.Close()
	activeSeries := ChartSeries{Key: "active_partner_count", Label: "Partner Active"}
	inactiveSeries := ChartSeries{Key: "inactive_partner_count", Label: "Partner Non-Active"}
	table := make([]map[string]any, 0)
	total := 0.0
	for rows.Next() {
		var label string
		var activeCount, inactiveCount float64
		if err := rows.Scan(&label, &activeCount, &inactiveCount); err != nil {
			return queryData{}, err
		}
		activeSeries.Points = append(activeSeries.Points, ChartPoint{X: label, Y: activeCount})
		inactiveSeries.Points = append(inactiveSeries.Points, ChartPoint{X: label, Y: inactiveCount})
		table = append(table, map[string]any{
			"pic_name":               label,
			"active_partner_count":   activeCount,
			"inactive_partner_count": inactiveCount,
		})
		total += activeCount
	}
	return queryData{Series: []ChartSeries{activeSeries, inactiveSeries}, Table: table, Value: total}, rows.Err()
}

func (r *Repository) CallMitraFrequency(ctx context.Context, actor identity.User, req QueryRequest, tf ResolvedTimeFilter) (queryData, error) {
	where, args := partnerInteractionScopedWhere(actor, req.Filters, tf, "pi.interaction_at")
	where += " AND UPPER(pi.interaction_type) = 'CALL'"
	query := `
		SELECT ` + groupExpr("pi.interaction_at", tf.Granularity) + ` AS period, COUNT(*) AS total
		FROM partner_interactions pi
		WHERE ` + where + `
		GROUP BY period
		ORDER BY period ASC`
	points, table, total, err := r.readTrendRows(ctx, query, args, "call_count")
	if err != nil {
		return queryData{}, err
	}
	return queryData{
		Series: []ChartSeries{{Key: "call_count", Label: "Call Mitra", Points: points}},
		Table:  table,
		Value:  total,
	}, nil
}

func (r *Repository) PartnerInactivityAging(ctx context.Context, actor identity.User, req QueryRequest, tf ResolvedTimeFilter) (queryData, error) {
	baseDate := tf.End
	query := `
		SELECT p.id,
			GREATEST(
				COALESCE(MAX(pi.interaction_at), '1970-01-01'),
				COALESCE(MAX(pr.referral_date), '1970-01-01'),
				COALESCE(MAX(pc.created_at), '1970-01-01')
			) AS last_activity_at
		FROM partners p
		LEFT JOIN partner_interactions pi ON pi.partner_id = p.id
		LEFT JOIN partner_referrals pr ON pr.partner_id = p.id
		LEFT JOIN partner_commissions pc ON pc.partner_id = p.id
		WHERE ` + partnerVisibilityCondition(actor, req.Filters, "p.id") + `
		GROUP BY p.id`
	args := partnerVisibilityArgs(actor, req.Filters)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return queryData{}, err
	}
	defer rows.Close()

	buckets := map[string]float64{"0-7": 0, "8-30": 0, "31-60": 0, "61-90": 0, "91-180": 0, "180+": 0}
	totalDays := 0.0
	totalCount := 0.0
	for rows.Next() {
		var partnerID int64
		var lastActivityRaw sql.NullString
		if err := rows.Scan(&partnerID, &lastActivityRaw); err != nil {
			return queryData{}, err
		}
		lastActivity := time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)
		if lastActivityRaw.Valid && strings.TrimSpace(lastActivityRaw.String) != "" {
			parsed, err := time.Parse("2006-01-02 15:04:05", lastActivityRaw.String)
			if err == nil {
				lastActivity = parsed
			}
		}
		days := baseDate.Sub(lastActivity).Hours() / 24
		if days < 0 {
			days = 0
		}
		totalCount++
		totalDays += days
		switch {
		case days <= 7:
			buckets["0-7"]++
		case days <= 30:
			buckets["8-30"]++
		case days <= 60:
			buckets["31-60"]++
		case days <= 90:
			buckets["61-90"]++
		case days <= 180:
			buckets["91-180"]++
		default:
			buckets["180+"]++
		}
	}
	if err := rows.Err(); err != nil {
		return queryData{}, err
	}
	labels := []string{"0-7", "8-30", "31-60", "61-90", "91-180", "180+"}
	series := ChartSeries{Key: "partner_count", Label: "Partner"}
	table := make([]map[string]any, 0, len(labels))
	for _, label := range labels {
		series.Points = append(series.Points, ChartPoint{X: label, Y: buckets[label]})
		table = append(table, map[string]any{"aging_bucket": label, "partner_count": buckets[label]})
	}
	avg := 0.0
	if totalCount > 0 {
		avg = round2(totalDays / totalCount)
	}
	return queryData{Series: []ChartSeries{series}, Table: table, Extra: map[string]any{"average_inactivity_days": avg}, Value: avg}, nil
}

func (r *Repository) PartnerRegionDistribution(ctx context.Context, actor identity.User, req QueryRequest, tf ResolvedTimeFilter) (queryData, error) {
	query := `
		SELECT COALESCE(NULLIF(TRIM(o.province), ''), 'Tanpa Wilayah') AS label, COUNT(DISTINCT p.id) AS total
		FROM partners p
		LEFT JOIN partner_referrals pr ON pr.partner_id = p.id
		LEFT JOIN customer_leads cl ON cl.id = pr.lead_id
		LEFT JOIN owners o ON o.id = cl.owner_id
		WHERE ` + partnerVisibilityCondition(actor, req.Filters, "p.id") + `
		GROUP BY label
		ORDER BY total DESC, label ASC`
	args := partnerVisibilityArgs(actor, req.Filters)
	return r.queryCategoryCount(ctx, query, args, "region", "partner_count")
}

func (r *Repository) CommissionEarnedTrend(ctx context.Context, actor identity.User, req QueryRequest, tf ResolvedTimeFilter) (queryData, error) {
	where, args := partnerCommissionScopedWhere(actor, req.Filters, tf, "pc.created_at")
	query := `
		SELECT ` + groupExpr("pc.created_at", tf.Granularity) + ` AS period, CAST(COALESCE(SUM(pc.commission_amount), 0) AS DOUBLE) AS total
		FROM partner_commissions pc
		WHERE ` + where + `
		GROUP BY period
		ORDER BY period ASC`
	points, table, total, err := r.readTrendRows(ctx, query, args, "commission_earned")
	if err != nil {
		return queryData{}, err
	}
	return queryData{Series: []ChartSeries{{Key: "commission_earned", Label: "Commission Earned", Points: points}}, Table: table, Value: total}, nil
}

func (r *Repository) PaidVsUnpaidCommission(ctx context.Context, actor identity.User, req QueryRequest, tf ResolvedTimeFilter) (queryData, error) {
	where, args := partnerCommissionScopedWhere(actor, req.Filters, tf, "pc.created_at")
	query := `
		SELECT
			CAST(COALESCE(SUM(CASE WHEN pc.status = 'PAID' THEN pc.commission_amount ELSE 0 END), 0) AS DOUBLE) AS paid_total,
			CAST(COALESCE(SUM(CASE WHEN pc.status IN ('PENDING','APPROVED') THEN pc.commission_amount ELSE 0 END), 0) AS DOUBLE) AS unpaid_total
		FROM partner_commissions pc
		WHERE ` + where
	var paidTotal, unpaidTotal float64
	if err := r.db.QueryRowContext(ctx, query, args...).Scan(&paidTotal, &unpaidTotal); err != nil {
		return queryData{}, err
	}
	return queryData{
		Series: []ChartSeries{{
			Key:   "commission_amount",
			Label: "Commission",
			Points: []ChartPoint{
				{X: "PAID", Y: paidTotal},
				{X: "UNPAID", Y: unpaidTotal},
			},
		}},
		Table: []map[string]any{
			{"status": "PAID", "commission_amount": paidTotal},
			{"status": "UNPAID", "commission_amount": unpaidTotal},
		},
		Value: unpaidTotal,
	}, nil
}

func (r *Repository) CommissionAging(ctx context.Context, actor identity.User, req QueryRequest, tf ResolvedTimeFilter) (queryData, error) {
	where, args := partnerCommissionScopedWhere(actor, req.Filters, tf, "pc.created_at")
	where += " AND pc.status IN ('PENDING','APPROVED')"
	query := `
		SELECT TIMESTAMPDIFF(DAY, pc.created_at, ?) AS aging_days
		FROM partner_commissions pc
		WHERE ` + where
	args = append([]any{tf.End}, args...)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return queryData{}, err
	}
	defer rows.Close()
	buckets := map[string]float64{"0-7": 0, "8-30": 0, "31-60": 0, "61-90": 0, "91-180": 0, "180+": 0}
	totalDays := 0.0
	totalCount := 0.0
	for rows.Next() {
		var days float64
		if err := rows.Scan(&days); err != nil {
			return queryData{}, err
		}
		totalCount++
		totalDays += days
		switch {
		case days <= 7:
			buckets["0-7"]++
		case days <= 30:
			buckets["8-30"]++
		case days <= 60:
			buckets["31-60"]++
		case days <= 90:
			buckets["61-90"]++
		case days <= 180:
			buckets["91-180"]++
		default:
			buckets["180+"]++
		}
	}
	if err := rows.Err(); err != nil {
		return queryData{}, err
	}
	labels := []string{"0-7", "8-30", "31-60", "61-90", "91-180", "180+"}
	series := ChartSeries{Key: "commission_count", Label: "Commission"}
	table := make([]map[string]any, 0, len(labels))
	for _, label := range labels {
		series.Points = append(series.Points, ChartPoint{X: label, Y: buckets[label]})
		table = append(table, map[string]any{"aging_bucket": label, "commission_count": buckets[label]})
	}
	avg := 0.0
	if totalCount > 0 {
		avg = round2(totalDays / totalCount)
	}
	return queryData{Series: []ChartSeries{series}, Table: table, Extra: map[string]any{"average_aging_days": avg}, Value: avg}, nil
}

func (r *Repository) CommissionByPartnerType(ctx context.Context, actor identity.User, req QueryRequest, tf ResolvedTimeFilter) (queryData, error) {
	where, args := partnerCommissionScopedWhere(actor, req.Filters, tf, "pc.created_at")
	query := `
		SELECT pt.name AS label, CAST(COALESCE(SUM(pc.commission_amount), 0) AS DOUBLE) AS total
		FROM partner_commissions pc
		JOIN partners p ON p.id = pc.partner_id
		JOIN partner_types pt ON pt.id = p.partner_type_id
		WHERE ` + where + `
		GROUP BY pt.id, pt.name
		ORDER BY total DESC, label ASC`
	return r.queryCategoryAggregate(ctx, query, args, "partner_type_name", "commission_amount")
}

func (r *Repository) CommissionByPackage(ctx context.Context, actor identity.User, req QueryRequest, tf ResolvedTimeFilter) (queryData, error) {
	where, args := partnerCommissionScopedWhere(actor, req.Filters, tf, "pc.created_at")
	query := `
		SELECT COALESCE(NULLIF(JSON_UNQUOTE(JSON_EXTRACT(sc.package_snapshot_json, '$.name')), ''), 'Tanpa Paket') AS label,
			CAST(COALESCE(SUM(pc.commission_amount), 0) AS DOUBLE) AS total
		FROM partner_commissions pc
		JOIN sales_closings sc ON sc.id = pc.closing_id
		WHERE ` + where + `
		GROUP BY label
		ORDER BY total DESC, label ASC`
	return r.queryCategoryAggregate(ctx, query, args, "package_name", "commission_amount")
}

func (r *Repository) PayoutWaterfall(ctx context.Context, actor identity.User, req QueryRequest, tf ResolvedTimeFilter) (queryData, error) {
	where, args := partnerCommissionScopedWhere(actor, req.Filters, tf, "pc.created_at")
	query := `
		SELECT
			CAST(COALESCE(SUM(pc.commission_amount), 0) AS DOUBLE) AS earned_total,
			CAST(COALESCE(SUM(CASE WHEN pc.status IN ('APPROVED','PAID') THEN pc.commission_amount ELSE 0 END), 0) AS DOUBLE) AS approved_total,
			CAST(COALESCE(SUM(CASE WHEN pc.status = 'PAID' THEN pc.commission_amount ELSE 0 END), 0) AS DOUBLE) AS paid_total
		FROM partner_commissions pc
		WHERE ` + where
	var earnedTotal, approvedTotal, paidTotal float64
	if err := r.db.QueryRowContext(ctx, query, args...).Scan(&earnedTotal, &approvedTotal, &paidTotal); err != nil {
		return queryData{}, err
	}
	points := []ChartPoint{
		{X: "Earned", Y: earnedTotal},
		{X: "Approved", Y: approvedTotal},
		{X: "Paid", Y: paidTotal},
	}
	table := []map[string]any{
		{"step": "EARNED", "amount": earnedTotal},
		{"step": "APPROVED", "amount": approvedTotal},
		{"step": "PAID", "amount": paidTotal},
	}
	return queryData{Series: []ChartSeries{{Key: "payout_waterfall", Label: "Payout Waterfall", Points: points}}, Table: table, Value: paidTotal}, nil
}

func (r *Repository) CommissionRuleHistoryTimeline(ctx context.Context, actor identity.User, req QueryRequest, tf ResolvedTimeFilter) (queryData, error) {
	if actor.RoleCode == identity.RoleSales {
		return queryData{Series: []ChartSeries{}, Table: []map[string]any{}, Value: 0}, nil
	}
	query := `
		SELECT
			DATE_FORMAT(a.created_at, '%Y-%m-%d') AS period,
			COALESCE(JSON_UNQUOTE(JSON_EXTRACT(a.after_json, '$.code')), JSON_UNQUOTE(JSON_EXTRACT(a.before_json, '$.code')), CONCAT('PARTNER-TYPE-', a.entity_id)) AS partner_type_code,
			COALESCE(JSON_UNQUOTE(JSON_EXTRACT(a.after_json, '$.name')), JSON_UNQUOTE(JSON_EXTRACT(a.before_json, '$.name')), 'Tanpa Nama') AS partner_type_name,
			CAST(COALESCE(JSON_UNQUOTE(JSON_EXTRACT(a.after_json, '$.commission_value')), JSON_UNQUOTE(JSON_EXTRACT(a.before_json, '$.commission_value')), '0') AS DOUBLE) AS commission_value,
			COALESCE(JSON_UNQUOTE(JSON_EXTRACT(a.after_json, '$.commission_mode')), JSON_UNQUOTE(JSON_EXTRACT(a.before_json, '$.commission_mode')), '') AS commission_mode,
			a.action,
			COALESCE(u.name, 'System') AS actor_name,
			a.request_id,
			a.created_at
		FROM audit_logs a
		LEFT JOIN users u ON u.id = a.actor_user_id
		WHERE a.entity_type = 'partner.type'
			AND a.created_at >= ? AND a.created_at < ?
		ORDER BY a.created_at ASC, a.id ASC`
	rows, err := r.db.QueryContext(ctx, query, tf.Start, tf.End)
	if err != nil {
		return queryData{}, err
	}
	defer rows.Close()
	seriesMap := map[string]*ChartSeries{}
	table := make([]map[string]any, 0)
	total := 0.0
	for rows.Next() {
		var period, code, name, mode, action, actorName, requestID string
		var value float64
		var changedAt time.Time
		if err := rows.Scan(&period, &code, &name, &value, &mode, &action, &actorName, &requestID, &changedAt); err != nil {
			return queryData{}, err
		}
		if _, ok := seriesMap[code]; !ok {
			seriesMap[code] = &ChartSeries{Key: slugifyKey(code), Label: name}
		}
		seriesMap[code].Points = append(seriesMap[code].Points, ChartPoint{X: period, Y: value})
		table = append(table, map[string]any{
			"period":            period,
			"partner_type_code": code,
			"partner_type_name": name,
			"commission_mode":   mode,
			"commission_value":  value,
			"action":            action,
			"actor_name":        actorName,
			"request_id":        requestID,
			"changed_at":        changedAt,
		})
		total++
	}
	if err := rows.Err(); err != nil {
		return queryData{}, err
	}
	return queryData{Series: sortSeriesMap(seriesMap), Table: table, Extra: map[string]any{"change_count": total}, Value: total}, nil
}

func (r *Repository) SnapshotVsCurrentCommission(ctx context.Context, actor identity.User, req QueryRequest, tf ResolvedTimeFilter) (queryData, error) {
	where, args := partnerCommissionScopedWhere(actor, req.Filters, tf, "pc.created_at")
	query := `
		SELECT
			CAST(COALESCE(SUM(pc.commission_amount), 0) AS DOUBLE) AS snapshot_total,
			CAST(COALESCE(SUM(
				CASE
					WHEN cr.id IS NOT NULL AND cr.mode = 'PERCENTAGE' THEN pc.base_amount * (CAST(cr.value AS DOUBLE) / 100)
					WHEN cr.id IS NOT NULL AND cr.mode = 'FIXED' THEN CAST(cr.value AS DOUBLE)
					WHEN cr.id IS NULL AND pt.commission_mode = 'PERCENTAGE' THEN pc.base_amount * (CAST(pt.commission_value AS DOUBLE) / 100)
					WHEN cr.id IS NULL AND pt.commission_mode = 'FIXED' THEN CAST(pt.commission_value AS DOUBLE)
					ELSE pc.commission_amount
				END
			), 0) AS DOUBLE) AS current_total
		FROM partner_commissions pc
		JOIN partners p ON p.id = pc.partner_id
		JOIN partner_types pt ON pt.id = p.partner_type_id
		LEFT JOIN commission_rules cr ON cr.id = pc.commission_rule_id
		WHERE ` + where
	var snapshotTotal, currentTotal float64
	if err := r.db.QueryRowContext(ctx, query, args...).Scan(&snapshotTotal, &currentTotal); err != nil {
		return queryData{}, err
	}
	return queryData{
		Series: []ChartSeries{{
			Key:   "commission_amount",
			Label: "Commission Amount",
			Points: []ChartPoint{
				{X: "SNAPSHOT", Y: snapshotTotal},
				{X: "CURRENT_RULE", Y: currentTotal},
			},
		}},
		Table: []map[string]any{
			{"mode": "SNAPSHOT", "commission_amount": snapshotTotal},
			{"mode": "CURRENT_RULE", "commission_amount": currentTotal},
			{"mode": "DELTA", "commission_amount": round2(currentTotal - snapshotTotal)},
		},
		Value: round2(currentTotal - snapshotTotal),
	}, nil
}

func (r *Repository) AuditLogVolumeByModule(ctx context.Context, actor identity.User, req QueryRequest, tf ResolvedTimeFilter) (queryData, error) {
	if actor.RoleCode == identity.RoleSales {
		return queryData{Series: []ChartSeries{}, Table: []map[string]any{}, Value: 0}, nil
	}
	query := `
		SELECT
			CASE
				WHEN LOCATE('.', a.entity_type) > 0 THEN SUBSTRING_INDEX(a.entity_type, '.', 1)
				ELSE a.entity_type
			END AS label,
			COUNT(*) AS total
		FROM audit_logs a
		WHERE a.created_at >= ? AND a.created_at < ?
		GROUP BY label
		ORDER BY total DESC, label ASC`
	return r.queryCategoryCount(ctx, query, []any{tf.Start, tf.End}, "module", "audit_log_count")
}

func (r *Repository) ActorActivityChart(ctx context.Context, actor identity.User, req QueryRequest, tf ResolvedTimeFilter) (queryData, error) {
	if actor.RoleCode == identity.RoleSales {
		return queryData{Series: []ChartSeries{}, Table: []map[string]any{}, Value: 0}, nil
	}
	query := `
		SELECT COALESCE(u.name, 'System') AS label, COUNT(*) AS total
		FROM audit_logs a
		LEFT JOIN users u ON u.id = a.actor_user_id
		WHERE a.created_at >= ? AND a.created_at < ?
		GROUP BY label
		ORDER BY total DESC, label ASC`
	return r.queryCategoryCount(ctx, query, []any{tf.Start, tf.End}, "actor_name", "activity_count")
}

func (r *Repository) RestoreVsDeleteTrend(ctx context.Context, actor identity.User, req QueryRequest, tf ResolvedTimeFilter) (queryData, error) {
	if actor.RoleCode == identity.RoleSales {
		return queryData{Series: []ChartSeries{}, Table: []map[string]any{}, Value: 0}, nil
	}
	query := `
		SELECT ` + groupExpr("a.created_at", tf.Granularity) + ` AS period,
			SUM(CASE WHEN a.action LIKE '%restore%' THEN 1 ELSE 0 END) AS restore_count,
			SUM(CASE WHEN a.action LIKE '%soft_delete%' OR a.action LIKE '%force_delete%' OR a.action LIKE '%delete%' THEN 1 ELSE 0 END) AS delete_count
		FROM audit_logs a
		WHERE a.created_at >= ? AND a.created_at < ?
		GROUP BY period
		ORDER BY period ASC`
	rows, err := r.db.QueryContext(ctx, query, tf.Start, tf.End)
	if err != nil {
		return queryData{}, err
	}
	defer rows.Close()
	restoreSeries := ChartSeries{Key: "restore_count", Label: "Restore"}
	deleteSeries := ChartSeries{Key: "delete_count", Label: "Delete"}
	table := make([]map[string]any, 0)
	total := 0.0
	for rows.Next() {
		var period string
		var restoreCount, deleteCount float64
		if err := rows.Scan(&period, &restoreCount, &deleteCount); err != nil {
			return queryData{}, err
		}
		restoreSeries.Points = append(restoreSeries.Points, ChartPoint{X: period, Y: restoreCount})
		deleteSeries.Points = append(deleteSeries.Points, ChartPoint{X: period, Y: deleteCount})
		table = append(table, map[string]any{"period": period, "restore_count": restoreCount, "delete_count": deleteCount})
		total += restoreCount
	}
	return queryData{Series: []ChartSeries{restoreSeries, deleteSeries}, Table: table, Value: total}, rows.Err()
}

func (r *Repository) BackendErrorCodeFrequency(ctx context.Context, actor identity.User, req QueryRequest, tf ResolvedTimeFilter) (queryData, error) {
	if actor.RoleCode == identity.RoleSales {
		return queryData{Series: []ChartSeries{}, Table: []map[string]any{}, Value: 0}, nil
	}
	type row struct {
		Label string
		Count float64
	}
	items := make([]row, 0)

	var validationFailed float64
	if err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM import_batches
		WHERE status = 'VALIDATION_FAILED'
			AND updated_at >= ? AND updated_at < ?`, tf.Start, tf.End).Scan(&validationFailed); err != nil {
		return queryData{}, err
	}
	items = append(items, row{Label: "VALIDATION_FAILED", Count: validationFailed})

	var commitFailed float64
	if err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM import_batches
		WHERE status = 'COMMIT_FAILED'
			AND updated_at >= ? AND updated_at < ?`, tf.Start, tf.End).Scan(&commitFailed); err != nil {
		return queryData{}, err
	}
	items = append(items, row{Label: "COMMIT_FAILED", Count: commitFailed})

	var jobFailed float64
	if err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM job_queue
		WHERE status = 'FAILED'
			AND updated_at >= ? AND updated_at < ?`, tf.Start, tf.End).Scan(&jobFailed); err != nil {
		return queryData{}, err
	}
	items = append(items, row{Label: "JOB_FAILED", Count: jobFailed})

	series := ChartSeries{Key: "error_count", Label: "Persisted Backend Failure"}
	table := make([]map[string]any, 0, len(items))
	total := 0.0
	for _, item := range items {
		series.Points = append(series.Points, ChartPoint{X: item.Label, Y: item.Count})
		table = append(table, map[string]any{"error_code": item.Label, "error_count": item.Count})
		total += item.Count
	}
	return queryData{
		Series: []ChartSeries{series},
		Table:  table,
		Extra: map[string]any{
			"source_note": "Chart ini saat ini hanya menghitung failure backend yang memang tersimpan di database (import_batches dan job_queue), belum seluruh http error response global.",
		},
		Value: total,
	}, nil
}

func partnerScopedWhere(actor identity.User, filters FilterRequest, tf ResolvedTimeFilter, timeColumn string) (string, []any) {
	where := []string{timeColumn + " >= ?", timeColumn + " < ?"}
	args := []any{tf.Start, tf.End}
	where = append(where, partnerVisibilityCondition(actor, filters, "p.id"))
	args = append(args, partnerVisibilityArgs(actor, filters)...)
	return strings.Join(where, " AND "), args
}

func partnerReferralScopedWhere(actor identity.User, filters FilterRequest, tf ResolvedTimeFilter, timeColumn string) (string, []any) {
	where := []string{timeColumn + " >= ?", timeColumn + " < ?"}
	args := []any{tf.Start, tf.End}
	where = append(where, partnerVisibilityCondition(actor, filters, "pr.partner_id"))
	args = append(args, partnerVisibilityArgs(actor, filters)...)
	return strings.Join(where, " AND "), args
}

func partnerInteractionScopedWhere(actor identity.User, filters FilterRequest, tf ResolvedTimeFilter, timeColumn string) (string, []any) {
	where := []string{timeColumn + " >= ?", timeColumn + " < ?"}
	args := []any{tf.Start, tf.End}
	where = append(where, partnerVisibilityCondition(actor, filters, "pi.partner_id"))
	args = append(args, partnerVisibilityArgs(actor, filters)...)
	return strings.Join(where, " AND "), args
}

func partnerCommissionScopedWhere(actor identity.User, filters FilterRequest, tf ResolvedTimeFilter, timeColumn string) (string, []any) {
	where := []string{timeColumn + " >= ?", timeColumn + " < ?"}
	args := []any{tf.Start, tf.End}
	where = append(where, partnerVisibilityCondition(actor, filters, "pc.partner_id"))
	args = append(args, partnerVisibilityArgs(actor, filters)...)
	return strings.Join(where, " AND "), args
}

func partnerAssignmentSnapshotWhere(actor identity.User, filters FilterRequest, baseDate time.Time) string {
	where := []string{
		"pa.assigned_at <= ?",
		"(pa.unassigned_at IS NULL OR pa.unassigned_at > ?)",
		partnerVisibilityCondition(actor, filters, "pa.partner_id"),
	}
	if len(filters.SalesIDs) > 0 {
		where = append(where, "pa.user_id IN ("+placeholders(len(filters.SalesIDs))+")")
	}
	return strings.Join(where, " AND ")
}

func partnerAssignmentSnapshotArgs(actor identity.User, filters FilterRequest, baseDate time.Time) []any {
	args := []any{baseDate, baseDate}
	args = append(args, partnerVisibilityArgs(actor, filters)...)
	for _, id := range filters.SalesIDs {
		args = append(args, id)
	}
	return args
}

func partnerVisibilityCondition(actor identity.User, filters FilterRequest, partnerColumn string) string {
	where := []string{}
	switch actor.RoleCode {
	case identity.RoleAdmin, identity.RoleSupervisor:
		where = append(where, "1 = 1")
	case identity.RoleSales:
		where = append(where, `EXISTS (
			SELECT 1 FROM partner_assignments pa
			WHERE pa.partner_id = `+partnerColumn+`
				AND pa.user_id = ?
				AND pa.active = TRUE
		)`)
	default:
		where = append(where, "1 = 0")
	}
	if len(filters.SalesIDs) > 0 {
		where = append(where, `EXISTS (
			SELECT 1 FROM partner_assignments pa
			WHERE pa.partner_id = `+partnerColumn+`
				AND pa.user_id IN (`+placeholders(len(filters.SalesIDs))+`)
				AND pa.active = TRUE
		)`)
	}
	if len(filters.Province) > 0 {
		where = append(where, `EXISTS (
			SELECT 1
			FROM partner_referrals pr
			JOIN customer_leads cl ON cl.id = pr.lead_id
			JOIN owners o ON o.id = cl.owner_id
			WHERE pr.partner_id = `+partnerColumn+`
				AND o.province IN (`+placeholders(len(filters.Province))+`)
		)`)
	}
	return strings.Join(where, " AND ")
}

func partnerVisibilityArgs(actor identity.User, filters FilterRequest) []any {
	args := []any{}
	if actor.RoleCode == identity.RoleSales {
		args = append(args, actor.ID)
	}
	for _, id := range filters.SalesIDs {
		args = append(args, id)
	}
	for _, province := range filters.Province {
		args = append(args, strings.TrimSpace(province))
	}
	return args
}
