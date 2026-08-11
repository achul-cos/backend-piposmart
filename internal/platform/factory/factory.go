package factory

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"backend_crm_piposmart/internal/platform/password"
)

const DummyPassword = "Password123!"

type Factory struct {
	rng  *rand.Rand
	asOf time.Time
}

func New(seed int64, asOf time.Time) *Factory {
	return &Factory{
		rng:  rand.New(rand.NewSource(seed)),
		asOf: asOf,
	}
}

type User struct {
	Code         string
	RoleCode     string
	Name         string
	Email        string
	Phone        string
	PasswordHash string
}

type Owner struct {
	Code            string
	Name            string
	Phone           string
	Email           string
	BrandName       string
	Province        string
	City            string
	District        string
	SubDistrict     string
	Address         string
	CreatedAt       time.Time
	EnteredByUserID sql.NullInt64
	// IsTestingAccount marks an owner whose data only exists because a piposmart employee
	// installed the app to learn/demo it, not a real prospective customer — excluded from the
	// sales/lead pipeline. TestingMarkedBy/TestingMarkedAt are NULL when this was inferred from
	// import data (source "Kategori Akun" = "Akun Testing") rather than a specific admin action.
	IsTestingAccount      bool
	TestingMarkedByUserID sql.NullInt64
	TestingMarkedAt       sql.NullTime
}

type Outlet struct {
	Code            string
	Name            string
	Phone           string
	Province        string
	City            string
	District        string
	SubDistrict     string
	Address         string
	EnteredByUserID sql.NullInt64
	// RowCode is "Kode Baris" — the raw per-row identifier from the original source spreadsheet,
	// tracked per outlet row (one owner can have several outlets, each with its own Kode Baris).
	RowCode string
}

type Lead struct {
	Code             string
	SourceType       string
	SourceReference  string
	Stage            string
	Status           string
	NextFollowUpAt   time.Time
	ActiveSalesEmail string
	EnteredByUserID  sql.NullInt64
}

type Interaction struct {
	Type             string
	CallStatus       string
	ChatStatus       string
	InteractionAt    time.Time
	RemarkScore      int
	Note             string
	CustomerResponse string
	FollowUpAt       time.Time
	FollowUpNote     string
}

type TrainingReport struct {
	TrainingType string
	Status       string
	ScheduledAt  time.Time
	CompletedAt  sql.NullTime
	Location     string
	MeetingURL   string
	Note         string
	ResultNote   string
}

func (f *Factory) BuildUser(roleCode string, index int) User {
	rolePrefix := strings.ToLower(roleCode)
	codePrefix := map[string]string{
		"ADMIN":      "ADM",
		"SUPERVISOR": "SPV",
		"SALES":      "SLS",
	}[roleCode]
	if codePrefix == "" {
		codePrefix = "USR"
	}

	email := fmt.Sprintf("%s.%03d@demo.piposmart.id", rolePrefix, index)
	return User{
		Code:         fmt.Sprintf("%s-%03d", codePrefix, index),
		RoleCode:     roleCode,
		Name:         fmt.Sprintf("%s Demo %03d", title(roleCode), index),
		Email:        email,
		Phone:        fmt.Sprintf("62812%08d", 100000+index),
		PasswordHash: deterministicPasswordHash(email),
	}
}

// BuildRealUser builds a User from a real display name (e.g. "Willi", "CS 4 - Wati" already
// normalized to "Wati") extracted from real seed data, instead of the "<Role> Demo NNN" template
// BuildUser uses. Email/code stay deterministic so the account is reproducible and login-testable
// (same DummyPassword as every other seeded user).
func (f *Factory) BuildRealUser(roleCode, displayName string, seedIndex int) User {
	codePrefix := map[string]string{
		"ADMIN":      "ADM",
		"SUPERVISOR": "SPV",
		"SALES":      "SLS",
	}[roleCode]
	if codePrefix == "" {
		codePrefix = "USR"
	}

	slug := slugifyRealName(displayName)
	email := fmt.Sprintf("%s.%s@internal.piposmart.id", strings.ToLower(roleCode), slug)
	return User{
		Code:         fmt.Sprintf("%s-R%04d", codePrefix, seedIndex),
		RoleCode:     roleCode,
		Name:         displayName,
		Email:        email,
		Phone:        fmt.Sprintf("62813%08d", 900000+seedIndex),
		PasswordHash: deterministicPasswordHash(email),
	}
}

func slugifyRealName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	var b strings.Builder
	lastDash := true // avoid leading dash
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		default:
			if !lastDash {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	slug := strings.TrimRight(b.String(), "-")
	if slug == "" {
		slug = "user"
	}
	return slug
}

func (f *Factory) BuildOwner(index int) Owner {
	provinces := []string{"DKI Jakarta", "Jawa Barat", "Jawa Timur", "Sumatera Utara", "Bali"}
	cities := map[string][]string{
		"DKI Jakarta":    {"Jakarta Selatan", "Jakarta Barat", "Jakarta Timur"},
		"Jawa Barat":     {"Bandung", "Bekasi", "Depok"},
		"Jawa Timur":     {"Surabaya", "Malang", "Sidoarjo"},
		"Sumatera Utara": {"Medan", "Binjai", "Deli Serdang"},
		"Bali":           {"Denpasar", "Badung", "Gianyar"},
	}
	province := provinces[f.rng.Intn(len(provinces))]
	cityOptions := cities[province]
	city := cityOptions[f.rng.Intn(len(cityOptions))]

	return Owner{
		Code:        fmt.Sprintf("OWN-%05d", index),
		Name:        fmt.Sprintf("Owner %03d", index),
		Phone:       fmt.Sprintf("62813%08d", 200000+index),
		Email:       fmt.Sprintf("owner%03d@example.test", index),
		BrandName:   fmt.Sprintf("Laundry Cerah %03d", index),
		Province:    province,
		City:        city,
		District:    fmt.Sprintf("Kecamatan Demo %02d", index),
		SubDistrict: fmt.Sprintf("Kelurahan Demo %02d", index),
		Address:     fmt.Sprintf("Jl. Demo CRM No. %d, %s", index, city),
	}
}

func (f *Factory) BuildOutlet(ownerCode string, index int, owner Owner) Outlet {
	return Outlet{
		Code:        fmt.Sprintf("%s-OUT-%02d", ownerCode, index),
		Name:        fmt.Sprintf("%s Outlet %02d", owner.BrandName, index),
		Phone:       fmt.Sprintf("62821%08d", 300000+index),
		Province:    owner.Province,
		City:        owner.City,
		District:    owner.District,
		SubDistrict: owner.SubDistrict,
		Address:     fmt.Sprintf("Jl. Outlet Demo No. %d, %s", index, owner.City),
	}
}

func (f *Factory) BuildLead(ownerCode string, index int, activeSalesEmail string) Lead {
	stage := []string{"NEW", "POSSIBLE", "POTENTIAL"}[f.rng.Intn(3)]
	return Lead{
		Code:             fmt.Sprintf("%s-LEAD-%02d", ownerCode, index),
		SourceType:       "DEMO_SEED",
		SourceReference:  fmt.Sprintf("minimal-%s", ownerCode),
		Stage:            stage,
		Status:           "OPEN",
		NextFollowUpAt:   f.asOf.AddDate(0, 0, 3+index),
		ActiveSalesEmail: activeSalesEmail,
	}
}

func (f *Factory) BuildInteraction(index int, score int) Interaction {
	callStatus := "TERHUBUNG"
	chatStatus := ""
	interactionType := "CALL"
	if index%2 == 0 {
		interactionType = "CALL_CHAT"
		chatStatus = "TERBALAS"
	}
	return Interaction{
		Type:             interactionType,
		CallStatus:       callStatus,
		ChatStatus:       chatStatus,
		InteractionAt:    f.asOf.Add(time.Duration(index) * time.Hour),
		RemarkScore:      score,
		Note:             fmt.Sprintf("Demo Sprint 06 remark %d-%d", score, index),
		CustomerResponse: fmt.Sprintf("Response customer demo untuk remark %d", score),
		FollowUpAt:       f.asOf.AddDate(0, 0, 3+index),
		FollowUpNote:     "Follow-up demo dari factory",
	}
}

func (f *Factory) BuildTrainingReport(index int, completed bool) TrainingReport {
	trainingType := "ONLINE"
	if index%2 == 0 {
		trainingType = "OFFLINE"
	}
	item := TrainingReport{
		TrainingType: trainingType,
		Status:       "SCHEDULED",
		ScheduledAt:  f.asOf.AddDate(0, 0, 2+index).Add(10 * time.Hour),
		Location:     fmt.Sprintf("Kantor customer demo %02d", index),
		MeetingURL:   fmt.Sprintf("https://meet.example.test/demo-%02d", index),
		Note:         "Training/demo aplikasi dari factory",
	}
	if completed {
		item.Status = "COMPLETED"
		item.CompletedAt = sql.NullTime{Time: item.ScheduledAt.Add(90 * time.Minute), Valid: true}
		item.ResultNote = "Customer memahami flow kasir dan outlet."
	}
	return item
}

func nullableSeedStringCompat(value string) sql.NullString {
	value = strings.TrimSpace(value)
	if value == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: value, Valid: true}
}

func (f *Factory) CreateUser(ctx context.Context, tx *sql.Tx, user User) (int64, error) {
	roleID, err := lookupID(ctx, tx, "roles", "code", user.RoleCode)
	if err != nil {
		return 0, err
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO users (role_id, code, name, email, phone, password_hash, status, must_change_password)
		VALUES (?, ?, ?, ?, ?, ?, 'ACTIVE', FALSE)
		ON DUPLICATE KEY UPDATE
			role_id = VALUES(role_id),
			name = VALUES(name),
			phone = VALUES(phone),
			password_hash = VALUES(password_hash),
			status = 'ACTIVE',
			must_change_password = FALSE`,
		roleID, user.Code, user.Name, user.Email, user.Phone, user.PasswordHash,
	)
	if err != nil {
		return 0, fmt.Errorf("seed user %s: %w", user.Email, err)
	}
	return lookupID(ctx, tx, "users", "email", user.Email)
}

func (f *Factory) CreateOwner(ctx context.Context, tx *sql.Tx, owner Owner) (int64, error) {
	createdAt := owner.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now()
	}
	_, err := tx.ExecContext(ctx, `
		INSERT INTO owners (code, name, phone, email, brand_name, province, city, district, sub_district, address, status, entered_by_user_id, created_at, is_testing_account, testing_marked_by_user_id, testing_marked_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'ACTIVE', ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			name = VALUES(name),
			phone = VALUES(phone),
			email = VALUES(email),
			brand_name = VALUES(brand_name),
			province = VALUES(province),
			city = VALUES(city),
			district = VALUES(district),
			sub_district = VALUES(sub_district),
			address = VALUES(address),
			status = 'ACTIVE',
			entered_by_user_id = VALUES(entered_by_user_id),
			-- GREATEST only ever upgrades FALSE->TRUE on re-seed/upsert, never un-flags a testing
			-- account that was already set (by import or by an admin) back to FALSE.
			is_testing_account = GREATEST(is_testing_account, VALUES(is_testing_account)),
			testing_marked_by_user_id = COALESCE(testing_marked_by_user_id, VALUES(testing_marked_by_user_id)),
			testing_marked_at = COALESCE(testing_marked_at, VALUES(testing_marked_at)),
			deleted_at = NULL`,
		owner.Code, owner.Name, owner.Phone, owner.Email, owner.BrandName, owner.Province, owner.City, owner.District, owner.SubDistrict, owner.Address, owner.EnteredByUserID, createdAt,
		owner.IsTestingAccount, owner.TestingMarkedByUserID, owner.TestingMarkedAt,
	)
	if err != nil {
		return 0, fmt.Errorf("seed owner %s: %w", owner.Code, err)
	}
	return lookupID(ctx, tx, "owners", "code", owner.Code)
}

func (f *Factory) CreateOutlet(ctx context.Context, tx *sql.Tx, ownerID int64, outlet Outlet) (int64, error) {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO outlets (owner_id, code, row_code, name, phone, province, city, district, sub_district, address, status, entered_by_user_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'ACTIVE', ?)
		ON DUPLICATE KEY UPDATE
			owner_id = VALUES(owner_id),
			row_code = VALUES(row_code),
			name = VALUES(name),
			phone = VALUES(phone),
			province = VALUES(province),
			city = VALUES(city),
			district = VALUES(district),
			sub_district = VALUES(sub_district),
			address = VALUES(address),
			status = 'ACTIVE',
			entered_by_user_id = VALUES(entered_by_user_id),
			deleted_at = NULL`,
		ownerID, outlet.Code, outlet.RowCode, outlet.Name, outlet.Phone, outlet.Province, outlet.City, outlet.District, outlet.SubDistrict, outlet.Address, outlet.EnteredByUserID,
	)
	if err != nil {
		return 0, fmt.Errorf("seed outlet %s: %w", outlet.Code, err)
	}
	return lookupID(ctx, tx, "outlets", "code", outlet.Code)
}

func (f *Factory) CreateLead(ctx context.Context, tx *sql.Tx, ownerID, outletID int64, lead Lead) (int64, error) {
	salesID, err := lookupID(ctx, tx, "users", "email", lead.ActiveSalesEmail)
	if err != nil {
		return 0, err
	}
	supervisorID, err := lookupFirstUserIDByRole(ctx, tx, "SUPERVISOR")
	if err != nil {
		return 0, err
	}
	score := scoreForStage(lead.Stage)

	res, err := tx.ExecContext(ctx, `
		INSERT INTO customer_leads
			(code, owner_id, outlet_id, active_sales_id, entered_by_user_id, current_owner_user_id, current_owner_role, supervisor_id, source_type, source_reference, stage, status, current_score, next_follow_up_at)
		VALUES (?, ?, ?, ?, ?, ?, 'SALES', ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			owner_id = VALUES(owner_id),
			outlet_id = VALUES(outlet_id),
			active_sales_id = VALUES(active_sales_id),
			entered_by_user_id = VALUES(entered_by_user_id),
			current_owner_user_id = VALUES(current_owner_user_id),
			current_owner_role = VALUES(current_owner_role),
			supervisor_id = VALUES(supervisor_id),
			source_type = VALUES(source_type),
			source_reference = VALUES(source_reference),
			stage = VALUES(stage),
			status = VALUES(status),
			current_score = VALUES(current_score),
			next_follow_up_at = VALUES(next_follow_up_at),
			deleted_at = NULL`,
		lead.Code, ownerID, outletID, salesID, lead.EnteredByUserID, salesID, supervisorID, lead.SourceType, lead.SourceReference, lead.Stage, lead.Status, score, lead.NextFollowUpAt,
	)
	if err != nil {
		return 0, fmt.Errorf("seed lead %s: %w", lead.Code, err)
	}

	leadID, err := res.LastInsertId()
	if err != nil || leadID == 0 {
		// ON DUPLICATE KEY UPDATE → LastInsertId = 0, fallback ke SELECT
		if err2 := tx.QueryRowContext(ctx,
			`SELECT id FROM customer_leads WHERE code = ? AND deleted_at IS NULL`,
			lead.Code,
		).Scan(&leadID); err2 != nil {
			// juga coba tanpa filter deleted_at
			if err3 := tx.QueryRowContext(ctx,
				`SELECT id FROM customer_leads WHERE code = ?`,
				lead.Code,
			).Scan(&leadID); err3 != nil {
				return 0, fmt.Errorf("lookup customer_leads.code=%s: %w", lead.Code, err3)
			}
		}
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE lead_assignments
		SET active = FALSE, ended_at = ?
		WHERE lead_id = ? AND active = TRUE`,
		f.asOf, leadID,
	); err != nil {
		return 0, fmt.Errorf("close active lead assignment %s: %w", lead.Code, err)
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO lead_assignments
			(lead_id, owner_id, to_user_id, to_role, supervisor_id, assigned_by_user_id, action, score, active, started_at)
		VALUES (?, ?, ?, 'SALES', ?, ?, 'DEMO_ASSIGNED_TO_SALES', ?, TRUE, ?)`,
		leadID, ownerID, salesID, supervisorID, supervisorID, score, f.asOf,
	); err != nil {
		return 0, fmt.Errorf("seed lead assignment %s: %w", lead.Code, err)
	}
	return leadID, nil
}

// ReassignLead replays a historical hand-off of a lead from one sales user to another at a real
// point in time (e.g. a "Share N" reassignment event from real seed data), closing the previously
// active lead_assignments row and inserting a new one with the given historical timestamp — unlike
// CreateLead's own bootstrap assignment row, which always uses the seed run's asOf date.
func (f *Factory) ReassignLead(ctx context.Context, tx *sql.Tx, leadID, ownerID, fromSalesID, toSalesID, supervisorID int64, action, reason string, score int, at time.Time) error {
	if _, err := tx.ExecContext(ctx, `
		UPDATE lead_assignments
		SET active = FALSE, ended_at = ?
		WHERE lead_id = ? AND active = TRUE`,
		at, leadID,
	); err != nil {
		return fmt.Errorf("close active lead assignment lead=%d: %w", leadID, err)
	}

	var fromUserID sql.NullInt64
	if fromSalesID > 0 {
		fromUserID = sql.NullInt64{Int64: fromSalesID, Valid: true}
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO lead_assignments
			(lead_id, owner_id, from_user_id, from_role, to_user_id, to_role, supervisor_id, assigned_by_user_id, action, reason, score, active, started_at)
		VALUES (?, ?, ?, 'SALES', ?, 'SALES', ?, ?, ?, ?, ?, TRUE, ?)`,
		leadID, ownerID, fromUserID, toSalesID, supervisorID, supervisorID, action, reason, score, at,
	); err != nil {
		return fmt.Errorf("insert lead assignment lead=%d: %w", leadID, err)
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE customer_leads
		SET current_score = ?, active_sales_id = ?, current_owner_user_id = ?
		WHERE id = ?`,
		score, toSalesID, toSalesID, leadID,
	); err != nil {
		return fmt.Errorf("update lead assignment lead=%d: %w", leadID, err)
	}
	return nil
}

func (f *Factory) CreateInteraction(ctx context.Context, tx *sql.Tx, leadID int64, interaction Interaction) (int64, error) {
	state, err := lookupLeadSeedState(ctx, tx, leadID)
	if err != nil {
		return 0, err
	}
	reasonID, reasonCode, reasonLabel, err := lookupRemarkReasonByScore(ctx, tx, interaction.RemarkScore)
	if err != nil {
		return 0, err
	}
	if _, err := tx.ExecContext(ctx, `
		DELETE FROM customer_interactions
		WHERE lead_id = ? AND note = ?`,
		leadID, interaction.Note,
	); err != nil {
		return 0, fmt.Errorf("reset demo interaction lead=%d: %w", leadID, err)
	}

	result, err := tx.ExecContext(ctx, `
		INSERT INTO customer_interactions
			(lead_id, owner_id, outlet_id, sales_id, supervisor_id, interaction_type, call_status, chat_status, interaction_at,
			 remark_reason_id, remark_score, remark_code, remark_label, note, customer_response,
			 follow_up_at, follow_up_note, stage_before, stage_after, status_before, status_after,
			 score_before, score_after, created_by_user_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		leadID,
		state.ownerID,
		state.outletID,
		state.salesID,
		state.supervisorID,
		interaction.Type,
		nullableSeedStringCompat(interaction.CallStatus),
		nullableSeedStringCompat(interaction.ChatStatus),
		interaction.InteractionAt,
		reasonID,
		interaction.RemarkScore,
		reasonCode,
		reasonLabel,
		interaction.Note,
		interaction.CustomerResponse,
		interaction.FollowUpAt,
		interaction.FollowUpNote,
		state.stage,
		stageForScore(interaction.RemarkScore, state.stage),
		state.status,
		statusForScore(interaction.RemarkScore),
		state.score,
		scoreForRemark(interaction.RemarkScore),
		state.salesID,
	)
	if err != nil {
		return 0, fmt.Errorf("seed interaction lead=%d: %w", leadID, err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE customer_leads
		SET last_interaction_at = ?, next_follow_up_at = ?
		WHERE id = ?`,
		interaction.InteractionAt, interaction.FollowUpAt, leadID,
	); err != nil {
		return 0, fmt.Errorf("update lead interaction seed lead=%d: %w", leadID, err)
	}
	return id, nil
}

func (f *Factory) CreateTrainingReport(ctx context.Context, tx *sql.Tx, leadID int64, training TrainingReport) (int64, error) {
	state, err := lookupLeadSeedState(ctx, tx, leadID)
	if err != nil {
		return 0, err
	}
	if _, err := tx.ExecContext(ctx, `
		DELETE FROM training_reports
		WHERE lead_id = ? AND note = ?`,
		leadID, training.Note,
	); err != nil {
		return 0, fmt.Errorf("reset demo training lead=%d: %w", leadID, err)
	}
	result, err := tx.ExecContext(ctx, `
		INSERT INTO training_reports
			(lead_id, owner_id, outlet_id, sales_id, supervisor_id, training_type, status,
			 scheduled_at, completed_at, location, meeting_url,
			 note, result_note, created_by_user_id, updated_by_user_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		leadID,
		state.ownerID,
		state.outletID,
		state.salesID,
		state.supervisorID,
		training.TrainingType,
		training.Status,
		training.ScheduledAt,
		training.CompletedAt,
		training.Location,
		training.MeetingURL,
		training.Note,
		training.ResultNote,
		state.salesID,
		state.salesID,
	)
	if err != nil {
		return 0, fmt.Errorf("seed training lead=%d: %w", leadID, err)
	}
	return result.LastInsertId()
}

type leadSeedState struct {
	ownerID      sql.NullInt64
	outletID     sql.NullInt64
	salesID      sql.NullInt64
	supervisorID sql.NullInt64
	stage        string
	status       string
	score        sql.NullInt64
}

func lookupLeadSeedState(ctx context.Context, tx *sql.Tx, leadID int64) (leadSeedState, error) {
	var state leadSeedState
	err := tx.QueryRowContext(ctx, `
		SELECT owner_id, outlet_id, active_sales_id, supervisor_id, stage, status, current_score
		FROM customer_leads
		WHERE id = ?`, leadID).
		Scan(&state.ownerID, &state.outletID, &state.salesID, &state.supervisorID, &state.stage, &state.status, &state.score)
	if err != nil {
		return leadSeedState{}, fmt.Errorf("lookup lead seed state id=%d: %w", leadID, err)
	}
	return state, nil
}

func lookupRemarkReasonByScore(ctx context.Context, tx *sql.Tx, score int) (int64, string, string, error) {
	var id int64
	var code, label string
	err := tx.QueryRowContext(ctx, `
		SELECT id, code, label
		FROM remark_reasons
		WHERE score = ? AND active = TRUE
		ORDER BY id
		LIMIT 1`, score).Scan(&id, &code, &label)
	if err != nil {
		return 0, "", "", fmt.Errorf("lookup remark score=%d: %w", score, err)
	}
	return id, code, label, nil
}

func lookupID(ctx context.Context, tx *sql.Tx, table, column, value string) (int64, error) {
	query := fmt.Sprintf("SELECT id FROM %s WHERE %s = ?", table, column)
	var id int64
	if err := tx.QueryRowContext(ctx, query, value).Scan(&id); err != nil {
		return 0, fmt.Errorf("lookup %s.%s=%s: %w", table, column, value, err)
	}
	return id, nil
}

func lookupFirstUserIDByRole(ctx context.Context, tx *sql.Tx, roleCode string) (int64, error) {
	var id int64
	err := tx.QueryRowContext(ctx, `
		SELECT u.id
		FROM users u
		JOIN roles r ON r.id = u.role_id
		WHERE r.code = ? AND u.deleted_at IS NULL
		ORDER BY u.id
		LIMIT 1`, roleCode).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("lookup first user role=%s: %w", roleCode, err)
	}
	return id, nil
}

func scoreForStage(stage string) int {
	switch stage {
	case "INVALID":
		return 0
	case "POTENTIAL":
		return 2
	case "CLOSING":
		return 3
	default:
		return 1
	}
}

func stageForScore(score int, fallback string) string {
	switch score {
	case 0:
		return "INVALID"
	case 1:
		if fallback == "POTENTIAL" || fallback == "CLOSING" {
			return fallback
		}
		return "POSSIBLE"
	case 2:
		return "POTENTIAL"
	case 3:
		return "CLOSING"
	default:
		return fallback
	}
}

func statusForScore(score int) string {
	if score == 0 {
		return "INVALID"
	}
	return "OPEN"
}

func scoreForRemark(score int) sql.NullInt64 {
	if score < 0 || score > 3 {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: int64(score), Valid: true}
}

func deterministicPasswordHash(email string) string {
	sum := sha256.Sum256([]byte("crm-piposmart-demo-password:" + email))
	return password.HashArgon2idWithSalt(DummyPassword, sum[:16])
}

func title(value string) string {
	value = strings.ToLower(value)
	return strings.ToUpper(value[:1]) + value[1:]
}
