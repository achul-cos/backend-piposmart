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
	Code      string
	Name      string
	Phone     string
	Email     string
	BrandName string
	Province  string
	City      string
	Address   string
}

type Outlet struct {
	Code     string
	Name     string
	Phone    string
	Province string
	City     string
	Address  string
}

type Lead struct {
	Code             string
	SourceType       string
	SourceReference  string
	Stage            string
	Status           string
	NextFollowUpAt   time.Time
	ActiveSalesEmail string
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
		Code:      fmt.Sprintf("OWN-%05d", index),
		Name:      fmt.Sprintf("Owner Laundry %03d", index),
		Phone:     fmt.Sprintf("62813%08d", 200000+index),
		Email:     fmt.Sprintf("owner%03d@example.test", index),
		BrandName: fmt.Sprintf("Laundry Cerah %03d", index),
		Province:  province,
		City:      city,
		Address:   fmt.Sprintf("Jl. Demo CRM No. %d, %s", index, city),
	}
}

func (f *Factory) BuildOutlet(ownerCode string, index int, owner Owner) Outlet {
	return Outlet{
		Code:     fmt.Sprintf("%s-OUT-%02d", ownerCode, index),
		Name:     fmt.Sprintf("%s Outlet %02d", owner.BrandName, index),
		Phone:    fmt.Sprintf("62821%08d", 300000+index),
		Province: owner.Province,
		City:     owner.City,
		Address:  fmt.Sprintf("Jl. Outlet Demo No. %d, %s", index, owner.City),
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

func (f *Factory) CreateUser(ctx context.Context, tx *sql.Tx, user User) (int64, error) {
	roleID, err := lookupID(ctx, tx, "roles", "code", user.RoleCode)
	if err != nil {
		return 0, err
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO users (role_id, code, name, email, phone, password_hash, status, must_change_password)
		VALUES (?, ?, ?, ?, ?, ?, 'ACTIVE', TRUE)
		ON DUPLICATE KEY UPDATE
			role_id = VALUES(role_id),
			name = VALUES(name),
			phone = VALUES(phone),
			password_hash = VALUES(password_hash),
			status = 'ACTIVE',
			must_change_password = TRUE`,
		roleID, user.Code, user.Name, user.Email, user.Phone, user.PasswordHash,
	)
	if err != nil {
		return 0, fmt.Errorf("seed user %s: %w", user.Email, err)
	}
	return lookupID(ctx, tx, "users", "email", user.Email)
}

func (f *Factory) CreateOwner(ctx context.Context, tx *sql.Tx, owner Owner) (int64, error) {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO owners (code, name, phone, email, brand_name, province, city, address, status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, 'ACTIVE')
		ON DUPLICATE KEY UPDATE
			name = VALUES(name),
			phone = VALUES(phone),
			email = VALUES(email),
			brand_name = VALUES(brand_name),
			province = VALUES(province),
			city = VALUES(city),
			address = VALUES(address),
			status = 'ACTIVE',
			deleted_at = NULL`,
		owner.Code, owner.Name, owner.Phone, owner.Email, owner.BrandName, owner.Province, owner.City, owner.Address,
	)
	if err != nil {
		return 0, fmt.Errorf("seed owner %s: %w", owner.Code, err)
	}
	return lookupID(ctx, tx, "owners", "code", owner.Code)
}

func (f *Factory) CreateOutlet(ctx context.Context, tx *sql.Tx, ownerID int64, outlet Outlet) (int64, error) {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO outlets (owner_id, code, name, phone, province, city, address, status)
		VALUES (?, ?, ?, ?, ?, ?, ?, 'ACTIVE')
		ON DUPLICATE KEY UPDATE
			owner_id = VALUES(owner_id),
			name = VALUES(name),
			phone = VALUES(phone),
			province = VALUES(province),
			city = VALUES(city),
			address = VALUES(address),
			status = 'ACTIVE',
			deleted_at = NULL`,
		ownerID, outlet.Code, outlet.Name, outlet.Phone, outlet.Province, outlet.City, outlet.Address,
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

	_, err = tx.ExecContext(ctx, `
		INSERT INTO customer_leads
			(code, owner_id, outlet_id, active_sales_id, current_owner_user_id, current_owner_role, supervisor_id, source_type, source_reference, stage, status, current_score, next_follow_up_at)
		VALUES (?, ?, ?, ?, ?, 'SALES', ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			owner_id = VALUES(owner_id),
			outlet_id = VALUES(outlet_id),
			active_sales_id = VALUES(active_sales_id),
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
		lead.Code, ownerID, outletID, salesID, salesID, supervisorID, lead.SourceType, lead.SourceReference, lead.Stage, lead.Status, score, lead.NextFollowUpAt,
	)
	if err != nil {
		return 0, fmt.Errorf("seed lead %s: %w", lead.Code, err)
	}
	leadID, err := lookupID(ctx, tx, "customer_leads", "code", lead.Code)
	if err != nil {
		return 0, err
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

func deterministicPasswordHash(email string) string {
	sum := sha256.Sum256([]byte("crm-piposmart-demo-password:" + email))
	return password.HashArgon2idWithSalt(DummyPassword, sum[:16])
}

func title(value string) string {
	value = strings.ToLower(value)
	return strings.ToUpper(value[:1]) + value[1:]
}
