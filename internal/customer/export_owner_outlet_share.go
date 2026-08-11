package customer

import (
	"context"
	"fmt"
	"strings"
)

const adminOwnerOutletShareLimit = 10

type adminShareSnapshot struct {
	Date     string
	Name     string
	Category string
}

type adminOwnerExportSnapshot struct {
	CurrentPIC    string
	StatusTerbaru string
	Akuisisi      string
	Shares        [adminOwnerOutletShareLimit]adminShareSnapshot
}

func (r *Repository) enrichOwnerOutletExportRows(ctx context.Context, rows []map[string]any) error {
	if len(rows) == 0 {
		return nil
	}

	ownerIDs := make([]int64, 0, len(rows))
	seen := make(map[int64]bool, len(rows))
	for _, row := range rows {
		ownerID := int64Val(row["owner_id"])
		if ownerID == 0 || seen[ownerID] {
			continue
		}
		seen[ownerID] = true
		ownerIDs = append(ownerIDs, ownerID)
	}
	if len(ownerIDs) == 0 {
		return nil
	}

	snapshots, err := r.loadAdminOwnerExportSnapshots(ctx, ownerIDs)
	if err != nil {
		return err
	}
	for _, row := range rows {
		snapshot, ok := snapshots[int64Val(row["owner_id"])]
		if !ok {
			continue
		}
		row["status_terbaru"] = snapshot.StatusTerbaru
		row["akuisisi"] = snapshot.Akuisisi
		row["pic"] = snapshot.CurrentPIC
		for index, share := range snapshot.Shares {
			n := index + 1
			row[fmt.Sprintf("tanggal_dibagikan_%d", n)] = share.Date
			row[fmt.Sprintf("share_%d", n)] = share.Name
			row[fmt.Sprintf("kategori_nasabah_%d", n)] = share.Category
		}
	}
	return nil
}

func (r *Repository) loadAdminOwnerExportSnapshots(ctx context.Context, ownerIDs []int64) (map[int64]adminOwnerExportSnapshot, error) {
	snapshots := make(map[int64]adminOwnerExportSnapshot, len(ownerIDs))
	if len(ownerIDs) == 0 {
		return snapshots, nil
	}

	args := make([]any, 0, len(ownerIDs))
	for _, ownerID := range ownerIDs {
		args = append(args, ownerID)
	}
	inClause := placeholders(len(ownerIDs))

	leadRows, err := r.db.QueryContext(ctx, `
		SELECT
			cl.owner_id,
			COALESCE(pic.name, '') AS pic_name,
			COALESCE(cl.stage, '') AS lead_stage,
			COALESCE(cl.status, '') AS lead_status,
			EXISTS(
				SELECT 1
				FROM subscriptions s
				WHERE s.owner_id = cl.owner_id
				  AND s.deleted_at IS NULL
			) AS has_subscription,
			EXISTS(
				SELECT 1
				FROM sales_closings sc
				WHERE sc.owner_id = cl.owner_id
				  AND sc.deleted_at IS NULL
				  AND sc.status = 'CONFIRMED'
			) AS has_confirmed_closing
		FROM customer_leads cl
		LEFT JOIN users pic
			ON pic.id = COALESCE(cl.active_sales_id, IF(cl.current_owner_role = 'SALES', cl.current_owner_user_id, NULL))
		WHERE cl.deleted_at IS NULL
		  AND cl.owner_id IN (`+inClause+`)`,
		args...,
	)
	if err != nil {
		return nil, err
	}
	defer leadRows.Close()

	for leadRows.Next() {
		var (
			ownerID             int64
			currentPIC          string
			leadStage           string
			leadStatus          string
			hasSubscription     bool
			hasConfirmedClosing bool
		)
		if err := leadRows.Scan(&ownerID, &currentPIC, &leadStage, &leadStatus, &hasSubscription, &hasConfirmedClosing); err != nil {
			return nil, err
		}
		snapshot := snapshots[ownerID]
		snapshot.CurrentPIC = currentPIC
		snapshot.Akuisisi = deriveAdminAkuisisi(hasSubscription, hasConfirmedClosing, leadStatus)
		snapshot.StatusTerbaru = deriveAdminStatusTerbaru("", hasSubscription, hasConfirmedClosing, leadStage, leadStatus)
		snapshots[ownerID] = snapshot
	}
	if err := leadRows.Err(); err != nil {
		return nil, err
	}

	assignmentRows, err := r.db.QueryContext(ctx, `
		SELECT owner_id, rn, assigned_at, share_name
		FROM (
			SELECT
				la.owner_id,
				ROW_NUMBER() OVER (PARTITION BY la.owner_id ORDER BY la.started_at ASC, la.id ASC) AS rn,
				DATE_FORMAT(la.started_at, '%Y-%m-%dT%H:%i:%sZ') AS assigned_at,
				COALESCE(u.name, '') AS share_name
			FROM lead_assignments la
			LEFT JOIN users u ON u.id = la.to_user_id
			WHERE la.deleted_at IS NULL
			  AND la.to_role = 'SALES'
			  AND la.owner_id IN (`+inClause+`)
		) ranked
		WHERE rn <= ?
		ORDER BY owner_id, rn`,
		append(args, adminOwnerOutletShareLimit)...,
	)
	if err != nil {
		return nil, err
	}
	defer assignmentRows.Close()

	for assignmentRows.Next() {
		var (
			ownerID    int64
			rank       int
			assignedAt string
			shareName  string
		)
		if err := assignmentRows.Scan(&ownerID, &rank, &assignedAt, &shareName); err != nil {
			return nil, err
		}
		if rank < 1 || rank > adminOwnerOutletShareLimit {
			continue
		}
		snapshot := snapshots[ownerID]
		snapshot.Shares[rank-1].Date = formatAdminDate(assignedAt)
		snapshot.Shares[rank-1].Name = shareName
		snapshots[ownerID] = snapshot
	}
	if err := assignmentRows.Err(); err != nil {
		return nil, err
	}

	noteRows, err := r.db.QueryContext(ctx, `
		SELECT owner_id, rn, interaction_at, note
		FROM (
			SELECT
				ci.owner_id,
				ROW_NUMBER() OVER (PARTITION BY ci.owner_id ORDER BY ci.interaction_at ASC, ci.id ASC) AS rn,
				DATE_FORMAT(ci.interaction_at, '%Y-%m-%dT%H:%i:%sZ') AS interaction_at,
				COALESCE(ci.note, '') AS note
			FROM customer_interactions ci
			WHERE ci.deleted_at IS NULL
			  AND ci.interaction_type = 'NOTE'
			  AND ci.note LIKE 'Share ke %'
			  AND ci.owner_id IN (`+inClause+`)
		) ranked
		WHERE rn <= ?
		ORDER BY owner_id, rn`,
		append(args, adminOwnerOutletShareLimit)...,
	)
	if err != nil {
		return nil, err
	}
	defer noteRows.Close()

	for noteRows.Next() {
		var (
			ownerID       int64
			rank          int
			interactionAt string
			note          string
		)
		if err := noteRows.Scan(&ownerID, &rank, &interactionAt, &note); err != nil {
			return nil, err
		}
		if rank < 1 || rank > adminOwnerOutletShareLimit {
			continue
		}
		shareName, category := parseAdminShareNote(note)
		snapshot := snapshots[ownerID]
		share := snapshot.Shares[rank-1]
		if share.Date == "" {
			share.Date = formatAdminDate(interactionAt)
		}
		if share.Name == "" {
			share.Name = shareName
		}
		share.Category = category
		snapshot.Shares[rank-1] = share
		if category != "" {
			snapshot.StatusTerbaru = deriveAdminStatusTerbaru(category, snapshot.Akuisisi == "Berlangganan", snapshot.Akuisisi == "Berlangganan", "", "")
		}
		snapshots[ownerID] = snapshot
	}
	if err := noteRows.Err(); err != nil {
		return nil, err
	}

	return snapshots, nil
}

func deriveAdminAkuisisi(hasSubscription, hasConfirmedClosing bool, leadStatus string) string {
	switch {
	case hasSubscription || hasConfirmedClosing:
		return "Berlangganan"
	case strings.EqualFold(strings.TrimSpace(leadStatus), "INVALID"):
		return "Gagal"
	default:
		return ""
	}
}

func deriveAdminStatusTerbaru(lastShareCategory string, hasSubscription, hasConfirmedClosing bool, leadStage, leadStatus string) string {
	lastShareCategory = strings.TrimSpace(lastShareCategory)
	if lastShareCategory != "" {
		return lastShareCategory
	}
	switch {
	case hasSubscription || hasConfirmedClosing:
		return "Berlangganan"
	case strings.EqualFold(strings.TrimSpace(leadStatus), "INVALID"):
		return "INVALID"
	case strings.TrimSpace(leadStage) != "":
		return strings.TrimSpace(leadStage)
	default:
		return ""
	}
}

func parseAdminShareNote(note string) (shareName, category string) {
	text := strings.TrimSpace(note)
	text = strings.TrimPrefix(text, "Share ke ")
	if idx := strings.LastIndex(text, " ("); idx >= 0 && strings.HasSuffix(text, ")") {
		text = strings.TrimSpace(text[:idx])
	}
	for _, separator := range []string{" — ", " – ", " - "} {
		if idx := strings.Index(text, separator); idx >= 0 {
			return strings.TrimSpace(text[:idx]), strings.TrimSpace(text[idx+len(separator):])
		}
	}
	return strings.TrimSpace(text), ""
}
