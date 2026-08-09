package partner

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"time"

	"backend_crm_piposmart/internal/identity"
	"backend_crm_piposmart/internal/platform/config"

	"database/sql"
)

// Service encapsulates business logic for partner management.
type Service struct {
	repo          *Repository
	cfg           config.Config
	encryptionKey []byte // AES key (must be 16, 24, or 32 bytes)
}

// NewService creates a new Partner service.
func NewService(repo *Repository, cfg config.Config) *Service {
	// Encryption key from environment variable PARTNER_ENC_KEY (base64)
	keyStr := getEnv("PARTNER_ENC_KEY", "")
	var key []byte
	if keyStr != "" {
		var err error
		key, err = base64.StdEncoding.DecodeString(keyStr)
		if err != nil || (len(key) != 16 && len(key) != 24 && len(key) != 32) {
			// fallback to a deterministic key for dev (not safe for production)
			// In production, must be set properly.
			key = []byte("0123456789abcdef") // 16 bytes
		}
	} else {
		key = []byte("0123456789abcdef") // 16 bytes
	}
	return &Service{
		repo:          repo,
		cfg:           cfg,
		encryptionKey: key,
	}
}

// getEnv retrieves an environment variable or returns a fallback value.
func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return strings.TrimSpace(value)
	}
	return fallback
}

// encrypt encrypts plaintext using AES-GCM and returns base64 ciphertext.
func (s *Service) encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	block, err := aes.NewCipher(s.encryptionKey)
	if err != nil {
		return "", err
	}
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, aesgcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := aesgcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// decrypt decrypts base64 ciphertext using AES-GCM and returns plaintext.
func (s *Service) decrypt(ciphertextB64 string) (string, error) {
	if ciphertextB64 == "" {
		return "", nil
	}
	ciphertext, err := base64.StdEncoding.DecodeString(ciphertextB64)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(s.encryptionKey)
	if err != nil {
		return "", err
	}
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(ciphertext) < aesgcm.NonceSize() {
		return "", errors.New("ciphertext too short")
	}
	nonce := ciphertext[:aesgcm.NonceSize()]
	plaintext, err := aesgcm.Open(nil, nonce, ciphertext[aesgcm.NonceSize():], nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

// maskAccount returns a masked version of the account number (e.g., ****1234)
// assuming last4 contains the last 4 digits.
func maskAccount(last4 string) string {
	if len(last4) == 0 {
		return ""
	}
	return "****" + last4
}

// nullStringToPtr converts sql.NullString to *string
func nullStringToPtr(ns sql.NullString) *string {
	if !ns.Valid || ns.String == "" {
		return nil
	}
	s := ns.String
	return &s
}

// attachPartnerType copies the fetched partner_type's denormalized fields onto p, so
// NewPartnerResponse can build a complete nested partner_type object.
func attachPartnerType(p *Partner, pt *PartnerType) {
	if pt == nil {
		return
	}
	p.PartnerTypeCode = sql.NullString{String: pt.Code, Valid: true}
	p.PartnerTypeName = sql.NullString{String: pt.Name, Valid: true}
	p.PartnerTypeCommissionMode = sql.NullString{String: pt.CommissionMode, Valid: true}
	p.PartnerTypeCommissionValue = sql.NullString{String: pt.CommissionValue, Valid: true}
}

func generatePartnerTypeCode(name string) string {
	cleanName := strings.ToUpper(strings.TrimSpace(name))
	cleanName = regexp.MustCompile(`[^A-Z0-9]`).ReplaceAllString(cleanName, "_")
	cleanName = regexp.MustCompile(`_+`).ReplaceAllString(cleanName, "_")
	cleanName = strings.Trim(cleanName, "_")
	if cleanName == "" {
		cleanName = "MITRA"
	}
	if len(cleanName) > 20 {
		cleanName = cleanName[:20]
	}
	return fmt.Sprintf("%s_%d", cleanName, time.Now().Unix()%10000)
}

/* ---------- PartnerType ---------- */

func (s *Service) CreatePartnerType(ctx context.Context, req CreatePartnerTypeRequest) (*PartnerTypeResponse, error) {
	if err := validateCommissionValue(req.CommissionMode, req.CommissionValue); err != nil {
		return nil, err
	}
	createdAt := time.Now()
	if req.CreatedAt != nil {
		createdAt = req.CreatedAt.UTC()
	}
	code := strings.ToUpper(strings.TrimSpace(req.Code))
	if code == "" {
		count, err := s.repo.CountPartnerTypes(ctx)
		if err != nil {
			count = 0
		}
		code = fmt.Sprintf("%03d", count+1)
	}
	pt := PartnerType{
		Code:            code,
		Name:            req.Name,
		CommissionMode:  req.CommissionMode,
		CommissionValue: req.CommissionValue,
		Description:     req.Description,
		CreatedAt:       createdAt,
		UpdatedAt:       createdAt,
	}
	id, err := s.repo.CreatePartnerType(ctx, pt)
	if err != nil {
		return nil, err
	}
	pt.ID = id
	resp := NewPartnerTypeResponse(pt)
	return &resp, nil
}

func (s *Service) GetPartnerTypeByID(ctx context.Context, id int64) (*PartnerTypeResponse, error) {
	pt, err := s.repo.GetPartnerTypeByID(ctx, id)
	if err != nil {
		return nil, err
	}
	resp := NewPartnerTypeResponse(*pt)
	return &resp, nil
}

func (s *Service) ListPartnerTypes(ctx context.Context, params PartnerTypeListParams) ([]PartnerTypeResponse, error) {
	pts, err := s.repo.ListPartnerTypes(ctx, params)
	if err != nil {
		return nil, err
	}
	resp := make([]PartnerTypeResponse, len(pts))
	for i, pt := range pts {
		resp[i] = NewPartnerTypeResponse(pt)
	}
	return resp, nil
}

func (s *Service) DeletePartnerType(ctx context.Context, id int64) error {
	return s.repo.DeletePartnerType(ctx, id)
}

func (s *Service) UpdatePartnerType(ctx context.Context, id int64, req UpdatePartnerTypeRequest) (*PartnerTypeResponse, error) {
	pt, err := s.repo.GetPartnerTypeByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if req.Name != "" {
		pt.Name = req.Name
	}
	if req.Description != "" {
		pt.Description = req.Description
	}
	mode := pt.CommissionMode
	if req.CommissionMode != "" {
		mode = req.CommissionMode
	}
	value := pt.CommissionValue
	if req.CommissionValue != "" {
		value = req.CommissionValue
	}
	if mode != pt.CommissionMode || value != pt.CommissionValue {
		if err := validateCommissionValue(mode, value); err != nil {
			return nil, err
		}
		pt.CommissionMode = mode
		pt.CommissionValue = value
	}
	if err := s.repo.UpdatePartnerType(ctx, id, *pt); err != nil {
		return nil, err
	}
	pt.UpdatedAt = time.Now()
	resp := NewPartnerTypeResponse(*pt)
	return &resp, nil
}

/* ---------- Partner ---------- */

func (s *Service) CreatePartner(ctx context.Context, actor identity.User, req CreatePartnerRequest) (*PartnerResponse, error) {
	// Validate partner type exists
	if _, err := s.repo.GetPartnerTypeByID(ctx, req.PartnerTypeID); err != nil {
		return nil, err
	}

	// Encrypt bank account if provided
	var encrypted []byte
	var last4 sql.NullString
	if req.BankAccount != nil && *req.BankAccount != "" {
		enc, err := s.encrypt(*req.BankAccount)
		if err != nil {
			return nil, err
		}
		encrypted = []byte(enc)
		// extract last 4 digits (assuming numeric)
		if len(*req.BankAccount) >= 4 {
			last4.String = (*req.BankAccount)[len(*req.BankAccount)-4:]
			last4.Valid = true
		}
	}

	createdAt := time.Now()
	if req.CreatedAt != nil {
		createdAt = req.CreatedAt.UTC()
	}
	p := Partner{
		PartnerTypeID:        req.PartnerTypeID,
		Code:                 req.Code,
		Name:                 req.Name,
		Phone:                ptrToSqlNullString(req.Phone),
		Email:                ptrToSqlNullString(req.Email),
		Province:             ptrToSqlNullString(req.Province),
		City:                 ptrToSqlNullString(req.City),
		District:             ptrToSqlNullString(req.District),
		SubDistrict:          ptrToSqlNullString(req.SubDistrict),
		Address:              ptrToSqlNullString(req.Address),
		BankAccountEncrypted: encrypted,
		BankAccountLast4:     last4,
		Status:               req.Status,
		CreatedAt:            createdAt,
		UpdatedAt:            createdAt,
	}
	if p.Status == "" {
		p.Status = StatusActive
	}

	id, err := s.repo.CreatePartner(ctx, p)
	if err != nil {
		return nil, err
	}
	p.ID = id

	if req.SelfAssignPIC && actor.ID > 0 {
		if _, err := s.repo.AssignPIC(ctx, PartnerAssignment{
			PartnerID:    id,
			UserID:       actor.ID,
			AssignedByID: sql.NullInt64{Int64: actor.ID, Valid: true},
			AssignedAt:   createdAt,
			Active:       true,
			CreatedAt:    createdAt,
			UpdatedAt:    createdAt,
		}); err != nil {
			return nil, fmt.Errorf("partner: self-assign PIC: %w", err)
		}
	}

	pt, _ := s.repo.GetPartnerTypeByID(ctx, p.PartnerTypeID)
	attachPartnerType(&p, pt)
	resp := NewPartnerResponse(p)
	return &resp, nil
}

func (s *Service) GetPartnerByID(ctx context.Context, id int64) (*PartnerResponse, error) {
	p, err := s.repo.GetPartnerByID(ctx, id)
	if err != nil {
		return nil, err
	}
	pt, _ := s.repo.GetPartnerTypeByID(ctx, p.PartnerTypeID)
	attachPartnerType(p, pt)
	resp := NewPartnerResponse(*p)
	return &resp, nil
}

func (s *Service) GetPartnerByCode(ctx context.Context, code string) (*PartnerResponse, error) {
	p, err := s.repo.GetPartnerByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	pt, _ := s.repo.GetPartnerTypeByID(ctx, p.PartnerTypeID)
	attachPartnerType(p, pt)
	resp := NewPartnerResponse(*p)
	return &resp, nil
}

func (s *Service) ListPartners(ctx context.Context, params PartnerListParams) ([]PartnerResponse, int64, error) {
	if params.Limit < 0 {
		params.Limit = 0
	}
	if params.Offset < 0 {
		params.Offset = 0
	}
	parts, total, err := s.repo.ListPartners(ctx, params)
	if err != nil {
		return nil, 0, err
	}
	resp := make([]PartnerResponse, len(parts))
	for i, p := range parts {
		pt, _ := s.repo.GetPartnerTypeByID(ctx, p.PartnerTypeID)
		attachPartnerType(&p, pt)
		resp[i] = NewPartnerResponse(p)
	}
	return resp, total, nil
}

func (s *Service) UpdatePartner(ctx context.Context, id int64, req UpdatePartnerRequest) (*PartnerResponse, error) {
	p, err := s.repo.GetPartnerByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if req.Name != nil {
		p.Name = *req.Name
	}
	if req.Phone != nil {
		p.Phone = ptrToSqlNullString(req.Phone)
	}
	if req.Email != nil {
		p.Email = ptrToSqlNullString(req.Email)
	}
	if req.Province != nil {
		p.Province = ptrToSqlNullString(req.Province)
	}
	if req.City != nil {
		p.City = ptrToSqlNullString(req.City)
	}
	if req.District != nil {
		p.District = ptrToSqlNullString(req.District)
	}
	if req.SubDistrict != nil {
		p.SubDistrict = ptrToSqlNullString(req.SubDistrict)
	}
	if req.Address != nil {
		p.Address = ptrToSqlNullString(req.Address)
	}
	if req.BankAccount != nil {
		if *req.BankAccount != "" {
			enc, err := s.encrypt(*req.BankAccount)
			if err != nil {
				return nil, err
			}
			p.BankAccountEncrypted = []byte(enc)
			if len(*req.BankAccount) >= 4 {
				p.BankAccountLast4.String = (*req.BankAccount)[len(*req.BankAccount)-4:]
				p.BankAccountLast4.Valid = true
			} else {
				p.BankAccountLast4 = sql.NullString{}
			}
		} else {
			p.BankAccountEncrypted = nil
			p.BankAccountLast4 = sql.NullString{}
		}
	}
	if req.Status != nil {
		p.Status = *req.Status
	}
	p.UpdatedAt = time.Now()
	if err := s.repo.UpdatePartner(ctx, id, *p); err != nil {
		return nil, err
	}
	// Fetch updated
	updatedP, err := s.repo.GetPartnerByID(ctx, id)
	if err != nil {
		return nil, err
	}
	pt, _ := s.repo.GetPartnerTypeByID(ctx, updatedP.PartnerTypeID)
	attachPartnerType(updatedP, pt)
	resp := NewPartnerResponse(*updatedP)
	return &resp, nil
}

func (s *Service) DeactivatePartner(ctx context.Context, id int64) error {
	return s.repo.DeactivatePartner(ctx, id)
}

/* ---------- PartnerAssignment ---------- */

func (s *Service) AssignPIC(ctx context.Context, partnerID int64, userID int64, assignedByID *int64, createdAt *time.Time) (*PartnerAssignmentResponse, error) {
	// Ensure partner exists
	if _, err := s.repo.GetPartnerByID(ctx, partnerID); err != nil {
		return nil, err
	}
	// Ensure user exists (could check via identity service, but we trust ID)
	now := time.Now()
	if createdAt != nil {
		now = createdAt.UTC()
	}
	a := PartnerAssignment{
		PartnerID:    partnerID,
		UserID:       userID,
		AssignedByID: ptrToInt64(assignedByID),
		AssignedAt:   now,
		Active:       true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	// Deactivate-then-insert happens atomically under a partner row lock
	// (see Repository.AssignPIC), backstopped by the DB-level
	// uq_partner_assignments_one_active constraint — this prevents two
	// concurrent AssignPIC calls for the same partner from both succeeding.
	id, err := s.repo.AssignPIC(ctx, a)
	if err != nil {
		return nil, err
	}
	a.ID = id
	resp := NewPartnerAssignmentResponse(a)
	return &resp, nil
}

func (s *Service) GetActiveAssignmentForPartner(ctx context.Context, partnerID int64) (*PartnerAssignmentResponse, error) {
	a, err := s.repo.GetActiveAssignmentForPartner(ctx, partnerID)
	if err != nil {
		return nil, err
	}
	resp := NewPartnerAssignmentResponse(*a)
	return &resp, nil
}

func (s *Service) ListPartnerAssignments(ctx context.Context, partnerID int64, params PartnerHistoryListParams) ([]PartnerAssignmentResponse, error) {
	list, err := s.repo.ListPartnerAssignments(ctx, partnerID, params)
	if err != nil {
		return nil, err
	}
	resp := make([]PartnerAssignmentResponse, len(list))
	for i, a := range list {
		resp[i] = NewPartnerAssignmentResponse(a)
	}
	return resp, nil
}

func (s *Service) ReleasePartner(ctx context.Context, partnerID int64) error {
	active, err := s.repo.GetActiveAssignmentForPartner(ctx, partnerID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil // already not assigned
		}
		return err
	}
	return s.repo.DeactivatePartnerAssignment(ctx, active.ID)
}

/* ---------- PartnerInteraction ---------- */

func (s *Service) RecordInteraction(ctx context.Context, partnerID int64, itype string, iTime *time.Time, note *string, createdAt *time.Time) (*PartnerInteractionResponse, error) {
	if itype != "CALL" && itype != "CHAT" {
		return nil, errors.New("invalid interaction type")
	}
	if _, err := s.repo.GetPartnerByID(ctx, partnerID); err != nil {
		return nil, err
	}
	now := time.Now()
	if createdAt != nil {
		now = createdAt.UTC()
	}
	i := PartnerInteraction{
		PartnerID:       partnerID,
		InteractionType: itype,
		InteractionAt:   now,
		Note:            ptrToSqlNullString(note),
		CreatedAt:       now,
	}
	if iTime != nil {
		i.InteractionAt = *iTime
	}
	id, err := s.repo.CreatePartnerInteraction(ctx, i)
	if err != nil {
		return nil, err
	}
	i.ID = id
	resp := NewPartnerInteractionResponse(i)
	return &resp, nil
}

func (s *Service) ListInteractions(ctx context.Context, partnerID int64, params PartnerHistoryListParams) ([]PartnerInteractionResponse, int64, error) {
	if params.Limit < 0 {
		params.Limit = 0
	}
	if params.Offset < 0 {
		params.Offset = 0
	}
	list, total, err := s.repo.ListPartnerInteractions(ctx, partnerID, params)
	if err != nil {
		return nil, 0, err
	}
	resp := make([]PartnerInteractionResponse, len(list))
	for i, iv := range list {
		resp[i] = NewPartnerInteractionResponse(iv)
	}
	return resp, total, nil
}

/* ---------- PartnerReferral ---------- */

func (s *Service) CreateReferral(ctx context.Context, partnerID int64, leadID int64, refTime *time.Time, notes *string, createdAt *time.Time) (*PartnerReferralResponse, error) {
	// Ensure partner exists
	if _, err := s.repo.GetPartnerByID(ctx, partnerID); err != nil {
		return nil, err
	}
	// Ensure lead exists (could call lead service, but we trust ID)
	// Check for duplicate referral
	existing, err := s.repo.GetReferralByPartnerLead(ctx, partnerID, leadID)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return nil, err
	}
	if existing != nil {
		return nil, ErrDuplicateReferral
	}
	now := time.Now()
	if createdAt != nil {
		now = createdAt.UTC()
	}
	r := PartnerReferral{
		PartnerID:    partnerID,
		LeadID:       leadID,
		ReferralDate: now,
		Notes:        ptrToSqlNullString(notes),
		CreatedAt:    now,
	}
	if refTime != nil {
		r.ReferralDate = *refTime
	}
	id, err := s.repo.CreatePartnerReferral(ctx, r)
	if err != nil {
		return nil, err
	}
	r.ID = id
	resp := NewPartnerReferralResponse(r)
	return &resp, nil
}

// GetMonthlyActivityStatus classifies a partner for the given month: TELAH_MEMBERIKAN_REFERAL if
// the PIC sales rep logged at least one referral that month, BELUM_MEMBERIKAN_REFERAL otherwise.
func (s *Service) GetMonthlyActivityStatus(ctx context.Context, partnerID int64, year int, month int) (*PartnerActivityStatusResponse, error) {
	if _, err := s.repo.GetPartnerByID(ctx, partnerID); err != nil {
		return nil, err
	}
	hasReferral, err := s.repo.HasReferralInMonth(ctx, partnerID, year, month)
	if err != nil {
		return nil, err
	}
	status := PartnerActivityNotYetReferred
	if hasReferral {
		status = PartnerActivityReferred
	}
	return &PartnerActivityStatusResponse{
		PartnerID: partnerID,
		Month:     fmt.Sprintf("%04d-%02d", year, month),
		Status:    status,
	}, nil
}

func (s *Service) ListReferrals(ctx context.Context, partnerID int64, params PartnerHistoryListParams) ([]PartnerReferralResponse, error) {
	if params.Limit < 0 {
		params.Limit = 0
	}
	if params.Offset < 0 {
		params.Offset = 0
	}
	list, err := s.repo.ListPartnerReferrals(ctx, partnerID, params)
	if err != nil {
		return nil, err
	}
	resp := make([]PartnerReferralResponse, len(list))
	for i, r := range list {
		resp[i] = NewPartnerReferralResponse(r)
	}
	return resp, nil
}

/* ---------- PartnerCommission ---------- */

// SyncCommissions scans confirmed closings tied to the partner's referrals and creates
// PENDING commission records for any that don't have one yet. Safe to call repeatedly.
func (s *Service) SyncCommissions(ctx context.Context, actor identity.User, partnerID int64) (*SyncCommissionsResponse, error) {
	if actor.RoleCode != RoleAdmin && actor.RoleCode != RoleSupervisor {
		return nil, ErrForbidden
	}
	if _, err := s.repo.GetPartnerByID(ctx, partnerID); err != nil {
		return nil, err
	}
	created, err := s.repo.SyncCommissions(ctx, partnerID)
	if err != nil {
		return nil, err
	}
	items := make([]PartnerCommissionResponse, len(created))
	for i, c := range created {
		items[i] = NewPartnerCommissionResponse(c)
	}
	return &SyncCommissionsResponse{Created: int64(len(created)), Items: items}, nil
}

func (s *Service) ListCommissions(ctx context.Context, partnerID int64, status string, page int, limit int) (PartnerCommissionListResponse, error) {
	if page < 1 {
		page = 1
	}
	offset := 0
	if limit <= 0 {
		limit = 0
	} else {
		if limit > 100 {
			limit = 100
		}
		offset = (page - 1) * limit
	}
	list, total, err := s.repo.ListPartnerCommissions(ctx, partnerID, status, limit, offset)
	if err != nil {
		return PartnerCommissionListResponse{}, err
	}
	items := make([]PartnerCommissionResponse, len(list))
	for i, c := range list {
		items[i] = NewPartnerCommissionResponse(c)
	}
	return PartnerCommissionListResponse{
		Items:      items,
		Pagination: PaginationMeta{Page: page, Limit: resolvePartnerLimit(limit, len(items), total), Total: total},
	}, nil
}

func (s *Service) GetCommission(ctx context.Context, commissionID int64) (*PartnerCommissionResponse, error) {
	c, err := s.repo.GetPartnerCommissionByID(ctx, commissionID)
	if err != nil {
		return nil, err
	}
	resp := NewPartnerCommissionResponse(*c)
	return &resp, nil
}

// ApproveCommission moves a PENDING commission to APPROVED, confirming it is ready to
// be paid out. ADMIN and SUPERVISOR may approve.
func (s *Service) ApproveCommission(ctx context.Context, actor identity.User, commissionID int64) (*PartnerCommissionResponse, error) {
	if actor.RoleCode != RoleAdmin && actor.RoleCode != RoleSupervisor {
		return nil, ErrForbidden
	}
	c, err := s.repo.ApproveCommission(ctx, commissionID, actor.ID)
	if err != nil {
		return nil, err
	}
	resp := NewPartnerCommissionResponse(*c)
	return &resp, nil
}

// PayCommission moves an APPROVED commission to PAID. Restricted to ADMIN since it
// represents money actually leaving the business.
func (s *Service) PayCommission(ctx context.Context, actor identity.User, commissionID int64) (*PartnerCommissionResponse, error) {
	if actor.RoleCode != RoleAdmin {
		return nil, ErrForbidden
	}
	c, err := s.repo.MarkCommissionPaid(ctx, commissionID, actor.ID)
	if err != nil {
		return nil, err
	}
	resp := NewPartnerCommissionResponse(*c)
	return &resp, nil
}

// CancelCommission voids a PENDING or APPROVED commission (e.g. the closing was later
// rejected/reversed). PAID and already-CANCELLED commissions cannot be cancelled again.
func (s *Service) CancelCommission(ctx context.Context, actor identity.User, commissionID int64, note string) (*PartnerCommissionResponse, error) {
	if actor.RoleCode != RoleAdmin && actor.RoleCode != RoleSupervisor {
		return nil, ErrForbidden
	}
	c, err := s.repo.CancelCommission(ctx, commissionID, note)
	if err != nil {
		return nil, err
	}
	resp := NewPartnerCommissionResponse(*c)
	return &resp, nil
}

/* ---------- CommissionRule ---------- */

// CreateCommissionRule adds an effective-dated, optionally package-scoped commission rate
// overlay for a partner_type. When mode is TIER, req.Tiers defines the volume brackets
// (validated by validateCommissionTiers); when PERCENTAGE/FIXED, req.Value is the flat rate
// (validated by validateRuleCommissionValue). If no rule matches a closing at sync time,
// calculation falls back to the legacy partner_types.commission_mode/value unchanged.
// ADMIN/SUPERVISOR only — rule configuration directly controls partner payouts.
func (s *Service) CreateCommissionRule(ctx context.Context, actor identity.User, partnerTypeID int64, req CreateCommissionRuleRequest) (*CommissionRuleResponse, error) {
	if actor.RoleCode != RoleAdmin && actor.RoleCode != RoleSupervisor {
		return nil, ErrForbidden
	}
	if _, err := s.repo.GetPartnerTypeByID(ctx, partnerTypeID); err != nil {
		return nil, err
	}
	if err := validateRuleCommissionValue(req.Mode, req.Value); err != nil {
		return nil, err
	}

	var tiers []CommissionTier
	if req.Mode == CommissionModeTier {
		if err := validateCommissionTiers(req.Tiers); err != nil {
			return nil, err
		}
		tiers = make([]CommissionTier, len(req.Tiers))
		for i, t := range req.Tiers {
			var maxClosings sql.NullInt64
			if t.MaxClosings != nil {
				maxClosings = sql.NullInt64{Int64: int64(*t.MaxClosings), Valid: true}
			}
			tiers[i] = CommissionTier{
				TierOrder:   t.TierOrder,
				MinClosings: t.MinClosings,
				MaxClosings: maxClosings,
				Mode:        t.Mode,
				Value:       t.Value,
			}
		}
	} else if len(req.Tiers) > 0 {
		return nil, ErrInvalidCommissionTier
	}

	var planID sql.NullInt64
	if req.PlanID != nil {
		planID = sql.NullInt64{Int64: *req.PlanID, Valid: true}
	}
	var value sql.NullString
	if req.Value != nil {
		value = sql.NullString{String: *req.Value, Valid: true}
	}
	var effectiveTo sql.NullTime
	if req.EffectiveTo != nil {
		effectiveTo = sql.NullTime{Time: *req.EffectiveTo, Valid: true}
	}

	rule := CommissionRule{
		PartnerTypeID:   partnerTypeID,
		PlanID:          planID,
		Mode:            req.Mode,
		Value:           value,
		EffectiveFrom:   req.EffectiveFrom,
		EffectiveTo:     effectiveTo,
		CreatedByUserID: sql.NullInt64{Int64: actor.ID, Valid: true},
	}
	id, err := s.repo.CreateCommissionRule(ctx, rule, tiers)
	if err != nil {
		return nil, err
	}
	created, err := s.repo.GetCommissionRuleByID(ctx, id)
	if err != nil {
		return nil, err
	}
	resp := NewCommissionRuleResponse(*created)
	return &resp, nil
}

func (s *Service) ListCommissionRules(ctx context.Context, partnerTypeID int64, planID *int64, activeOnly bool) ([]CommissionRuleResponse, error) {
	if _, err := s.repo.GetPartnerTypeByID(ctx, partnerTypeID); err != nil {
		return nil, err
	}
	list, err := s.repo.ListCommissionRules(ctx, partnerTypeID, planID, activeOnly)
	if err != nil {
		return nil, err
	}
	items := make([]CommissionRuleResponse, len(list))
	for i, r := range list {
		items[i] = NewCommissionRuleResponse(r)
	}
	return items, nil
}

func (s *Service) GetCommissionRule(ctx context.Context, ruleID int64) (*CommissionRuleResponse, error) {
	r, err := s.repo.GetCommissionRuleByID(ctx, ruleID)
	if err != nil {
		return nil, err
	}
	resp := NewCommissionRuleResponse(*r)
	return &resp, nil
}

// DeactivateCommissionRule retires a rule (rules are superseded by creating a new one with
// a later effective_from, never edited in place). ADMIN/SUPERVISOR only.
func (s *Service) DeactivateCommissionRule(ctx context.Context, actor identity.User, ruleID int64) (*CommissionRuleResponse, error) {
	if actor.RoleCode != RoleAdmin {
		return nil, ErrForbidden
	}
	if err := s.repo.DeactivateCommissionRule(ctx, ruleID); err != nil {
		return nil, err
	}
	r, err := s.repo.GetCommissionRuleByID(ctx, ruleID)
	if err != nil {
		return nil, err
	}
	resp := NewCommissionRuleResponse(*r)
	return &resp, nil
}

/* ---------- PartnerPayout ---------- */

// CreatePayout batches every APPROVED, not-yet-batched commission for partnerID into one
// new PENDING payout. ADMIN/SUPERVISOR only — mirrors SyncCommissions/ApproveCommission:
// supervisors may prepare a payout, only ADMIN executes the actual payment via PayPayout.
func (s *Service) CreatePayout(ctx context.Context, actor identity.User, partnerID int64) (*PartnerPayoutResponse, error) {
	if actor.RoleCode != RoleAdmin && actor.RoleCode != RoleSupervisor {
		return nil, ErrForbidden
	}
	if _, err := s.repo.GetPartnerByID(ctx, partnerID); err != nil {
		return nil, err
	}
	id, err := s.repo.CreatePayout(ctx, partnerID, actor.ID)
	if err != nil {
		return nil, err
	}
	p, err := s.repo.GetPayoutByID(ctx, id)
	if err != nil {
		return nil, err
	}
	resp := NewPartnerPayoutResponse(*p)
	return &resp, nil
}

func (s *Service) ListPayouts(ctx context.Context, partnerID int64, status string, page int, limit int) (PartnerPayoutListResponse, error) {
	if page < 1 {
		page = 1
	}
	offset := 0
	if limit <= 0 {
		limit = 0
	} else {
		if limit > 100 {
			limit = 100
		}
		offset = (page - 1) * limit
	}
	list, total, err := s.repo.ListPartnerPayouts(ctx, partnerID, status, limit, offset)
	if err != nil {
		return PartnerPayoutListResponse{}, err
	}
	items := make([]PartnerPayoutResponse, len(list))
	for i, p := range list {
		items[i] = NewPartnerPayoutResponse(p)
	}
	return PartnerPayoutListResponse{
		Items:      items,
		Pagination: PaginationMeta{Page: page, Limit: resolvePartnerLimit(limit, len(items), total), Total: total},
	}, nil
}

func resolvePartnerLimit(limit int, itemCount int, total int64) int {
	if limit > 0 {
		return limit
	}
	if total == 0 {
		return 0
	}
	return itemCount
}

func (s *Service) GetPayout(ctx context.Context, payoutID int64) (*PartnerPayoutResponse, error) {
	p, err := s.repo.GetPayoutByID(ctx, payoutID)
	if err != nil {
		return nil, err
	}
	resp := NewPartnerPayoutResponse(*p)
	return &resp, nil
}

// PayPayout moves a PENDING payout (and every commission still batched in it) to PAID.
// Restricted to ADMIN since it represents money actually leaving the business — same
// reasoning as PayCommission.
func (s *Service) PayPayout(ctx context.Context, actor identity.User, payoutID int64) (*PartnerPayoutResponse, error) {
	if actor.RoleCode != RoleAdmin {
		return nil, ErrForbidden
	}
	p, err := s.repo.MarkPayoutPaid(ctx, payoutID, actor.ID)
	if err != nil {
		return nil, err
	}
	resp := NewPartnerPayoutResponse(*p)
	return &resp, nil
}

// CancelPayout voids a PENDING payout, releasing its batched commissions back to APPROVED
// (payable individually or batchable into a future payout again).
func (s *Service) CancelPayout(ctx context.Context, actor identity.User, payoutID int64, note string) (*PartnerPayoutResponse, error) {
	if actor.RoleCode != RoleAdmin && actor.RoleCode != RoleSupervisor {
		return nil, ErrForbidden
	}
	p, err := s.repo.CancelPayout(ctx, payoutID, note)
	if err != nil {
		return nil, err
	}
	resp := NewPartnerPayoutResponse(*p)
	return &resp, nil
}

/* Helper functions */

func ptrToInt64(p *int64) sql.NullInt64 {
	if p == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: *p, Valid: true}
}

func ptrToSqlNullString(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *s, Valid: true}
}
