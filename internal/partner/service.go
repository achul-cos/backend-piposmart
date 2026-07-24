package partner

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"os"
	"strings"
	"time"

	"backend_crm_piposmart/internal/platform/config"

	"database/sql"
)

// Service encapsulates business logic for partner management.
type Service struct {
	repo      *Repository
	cfg       config.Config
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
		repo:           repo,
		cfg:            cfg,
		encryptionKey:  key,
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

/* ---------- PartnerType ---------- */

func (s *Service) CreatePartnerType(ctx context.Context, req CreatePartnerTypeRequest) (*PartnerTypeResponse, error) {
	pt := PartnerType{
		Code:        req.Code,
		Name:        req.Name,
		Description: req.Description,
	}
	id, err := s.repo.CreatePartnerType(ctx, pt)
	if err != nil {
		return nil, err
	}
	pt.ID = id
	pt.CreatedAt = time.Now()
	pt.UpdatedAt = time.Now()
	return &PartnerTypeResponse{
		ID:          pt.ID,
		Code:        pt.Code,
		Name:        pt.Name,
		Description: pt.Description,
		CreatedAt:   pt.CreatedAt,
		UpdatedAt:   pt.UpdatedAt,
	}, nil
}

func (s *Service) GetPartnerTypeByID(ctx context.Context, id int64) (*PartnerTypeResponse, error) {
	pt, err := s.repo.GetPartnerTypeByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return &PartnerTypeResponse{
		ID:          pt.ID,
		Code:        pt.Code,
		Name:        pt.Name,
		Description: pt.Description,
		CreatedAt:   pt.CreatedAt,
		UpdatedAt:   pt.UpdatedAt,
	}, nil
}

func (s *Service) ListPartnerTypes(ctx context.Context) ([]PartnerTypeResponse, error) {
	pts, err := s.repo.ListPartnerTypes(ctx)
	if err != nil {
		return nil, err
	}
	resp := make([]PartnerTypeResponse, len(pts))
	for i, pt := range pts {
		resp[i] = PartnerTypeResponse{
			ID:          pt.ID,
			Code:        pt.Code,
			Name:        pt.Name,
			Description: pt.Description,
			CreatedAt:   pt.CreatedAt,
			UpdatedAt:   pt.UpdatedAt,
		}
	}
	return resp, nil
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
	if err := s.repo.UpdatePartnerType(ctx, id, *pt); err != nil {
		return nil, err
	}
	pt.UpdatedAt = time.Now()
	return &PartnerTypeResponse{
		ID:          pt.ID,
		Code:        pt.Code,
		Name:        pt.Name,
		Description: pt.Description,
		CreatedAt:   pt.CreatedAt,
		UpdatedAt:   pt.UpdatedAt,
	}, nil
}

/* ---------- Partner ---------- */

func (s *Service) CreatePartner(ctx context.Context, req CreatePartnerRequest) (*PartnerResponse, error) {
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

	p := Partner{
		PartnerTypeID:      req.PartnerTypeID,
		Code:               req.Code,
		Name:               req.Name,
		Phone:              ptrToSqlNullString(req.Phone),
		Email:              ptrToSqlNullString(req.Email),
		Address:            ptrToSqlNullString(req.Address),
		BankAccountEncrypted: encrypted,
		BankAccountLast4:   last4,
		Status:             req.Status,
	}
	if p.Status == "" {
		p.Status = StatusActive
	}

	id, err := s.repo.CreatePartner(ctx, p)
	if err != nil {
		return nil, err
	}
	p.ID = id
	p.CreatedAt = time.Now()
	p.UpdatedAt = time.Now()

	pt, _ := s.repo.GetPartnerTypeByID(ctx, p.PartnerTypeID)
	if pt != nil {
		p.PartnerTypeCode = sql.NullString{String: pt.Code, Valid: true}
		p.PartnerTypeName = sql.NullString{String: pt.Name, Valid: true}
	}
	resp := NewPartnerResponse(p)
	return &resp, nil
}

func (s *Service) GetPartnerByID(ctx context.Context, id int64) (*PartnerResponse, error) {
	p, err := s.repo.GetPartnerByID(ctx, id)
	if err != nil {
		return nil, err
	}
	pt, _ := s.repo.GetPartnerTypeByID(ctx, p.PartnerTypeID)
	if pt != nil {
		p.PartnerTypeCode = sql.NullString{String: pt.Code, Valid: true}
		p.PartnerTypeName = sql.NullString{String: pt.Name, Valid: true}
	}
	resp := NewPartnerResponse(*p)
	return &resp, nil
}

func (s *Service) GetPartnerByCode(ctx context.Context, code string) (*PartnerResponse, error) {
	p, err := s.repo.GetPartnerByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	pt, _ := s.repo.GetPartnerTypeByID(ctx, p.PartnerTypeID)
	if pt != nil {
		p.PartnerTypeCode = sql.NullString{String: pt.Code, Valid: true}
		p.PartnerTypeName = sql.NullString{String: pt.Name, Valid: true}
	}
	resp := NewPartnerResponse(*p)
	return &resp, nil
}

func (s *Service) ListPartners(ctx context.Context, limit int, offset int, search string) ([]PartnerResponse, int64, error) {
	parts, total, err := s.repo.ListPartners(ctx, limit, offset, search)
	if err != nil {
		return nil, 0, err
	}
	resp := make([]PartnerResponse, len(parts))
	for i, p := range parts {
		pt, _ := s.repo.GetPartnerTypeByID(ctx, p.PartnerTypeID)
		if pt != nil {
			p.PartnerTypeCode = sql.NullString{String: pt.Code, Valid: true}
			p.PartnerTypeName = sql.NullString{String: pt.Name, Valid: true}
		}
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
	if pt != nil {
		updatedP.PartnerTypeCode = sql.NullString{String: pt.Code, Valid: true}
		updatedP.PartnerTypeName = sql.NullString{String: pt.Name, Valid: true}
	}
	resp := NewPartnerResponse(*updatedP)
	return &resp, nil
}

func (s *Service) DeactivatePartner(ctx context.Context, id int64) error {
	return s.repo.DeactivatePartner(ctx, id)
}

/* ---------- PartnerAssignment ---------- */

func (s *Service) AssignPIC(ctx context.Context, partnerID int64, userID int64, assignedByID *int64) (*PartnerAssignmentResponse, error) {
	// Ensure partner exists
	if _, err := s.repo.GetPartnerByID(ctx, partnerID); err != nil {
		return nil, err
	}
	// Ensure user exists (could check via identity service, but we trust ID)
	// Ensure only one active assignment per partner
	active, err := s.repo.GetActiveAssignmentForPartner(ctx, partnerID)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return nil, err
	}
	if active != nil {
		// Deactivate existing active assignment
		if err := s.repo.DeactivatePartnerAssignment(ctx, active.ID); err != nil {
			return nil, err
		}
	}
	a := PartnerAssignment{
		PartnerID:   partnerID,
		UserID:      userID,
		AssignedByID: ptrToInt64(assignedByID),
		AssignedAt:  time.Now(),
		Active:      true,
	}
	id, err := s.repo.CreatePartnerAssignment(ctx, a)
	if err != nil {
		return nil, err
	}
	a.ID = id
	a.CreatedAt = time.Now()
	a.UpdatedAt = time.Now()
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

func (s *Service) ListPartnerAssignments(ctx context.Context, partnerID int64) ([]PartnerAssignmentResponse, error) {
	list, err := s.repo.ListPartnerAssignments(ctx, partnerID)
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

func (s *Service) RecordInteraction(ctx context.Context, partnerID int64, itype string, iTime *time.Time, note *string) (*PartnerInteractionResponse, error) {
	if itype != "CALL" && itype != "CHAT" {
		return nil, errors.New("invalid interaction type")
	}
	if _, err := s.repo.GetPartnerByID(ctx, partnerID); err != nil {
		return nil, err
	}
	i := PartnerInteraction{
		PartnerID:      partnerID,
		InteractionType: itype,
		InteractionAt:  time.Now(),
		Note:           ptrToSqlNullString(note),
	}
	if iTime != nil {
		i.InteractionAt = *iTime
	}
	id, err := s.repo.CreatePartnerInteraction(ctx, i)
	if err != nil {
		return nil, err
	}
	i.ID = id
	i.CreatedAt = time.Now()
	resp := NewPartnerInteractionResponse(i)
	return &resp, nil
}

func (s *Service) ListInteractions(ctx context.Context, partnerID int64, limit int, offset int) ([]PartnerInteractionResponse, int64, error) {
	list, total, err := s.repo.ListPartnerInteractions(ctx, partnerID, limit, offset)
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

func (s *Service) CreateReferral(ctx context.Context, partnerID int64, leadID int64, refTime *time.Time, notes *string) (*PartnerReferralResponse, error) {
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
	r := PartnerReferral{
		PartnerID:    partnerID,
		LeadID:       leadID,
		ReferralDate: time.Now(),
		Notes:        ptrToSqlNullString(notes),
	}
	if refTime != nil {
		r.ReferralDate = *refTime
	}
	id, err := s.repo.CreatePartnerReferral(ctx, r)
	if err != nil {
		return nil, err
	}
	r.ID = id
	r.CreatedAt = time.Now()
	resp := NewPartnerReferralResponse(r)
	return &resp, nil
}

func (s *Service) ListReferrals(ctx context.Context, partnerID int64) ([]PartnerReferralResponse, error) {
	list, err := s.repo.ListPartnerReferrals(ctx, partnerID)
	if err != nil {
		return nil, err
	}
	resp := make([]PartnerReferralResponse, len(list))
	for i, r := range list {
		resp[i] = NewPartnerReferralResponse(r)
	}
	return resp, nil
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