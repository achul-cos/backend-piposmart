package customer

import (
	"context"
	"strings"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) ListOwners(ctx context.Context, actor Actor, params ListParams) (OwnerListResponse, error) {
	params = normalizeListParams(params)
	if params.Phone != "" {
		phone, err := NormalizePhone(params.Phone)
		if err == nil {
			params.Phone = phone
		}
	}
	owners, total, err := s.repo.ListOwners(ctx, actor, params)
	if err != nil {
		return OwnerListResponse{}, err
	}
	items := make([]OwnerResponse, 0, len(owners))
	for _, owner := range owners {
		items = append(items, NewOwnerResponse(owner))
	}
	return OwnerListResponse{
		Items: items,
		Pagination: PaginationMeta{
			Page:  params.Page,
			Limit: resolveReturnedLimit(params.All, params.Limit, len(items), total),
			Total: total,
		},
	}, nil
}

func (s *Service) CreateOwner(ctx context.Context, actor Actor, req CreateOwnerRequest) (OwnerResponse, error) {
	if !actorCanManageOwners(actor) {
		return OwnerResponse{}, ErrForbidden
	}
	phone, err := NormalizePhone(req.Phone)
	if err != nil {
		return OwnerResponse{}, err
	}
	owner, err := s.repo.CreateOwner(ctx, actor, req, phone)
	if err != nil {
		return OwnerResponse{}, err
	}
	return NewOwnerResponse(owner), nil
}

func (s *Service) BulkCreateOwners(ctx context.Context, actor Actor, req BulkOwnerCreateRequest) (OwnerBulkResponse, error) {
	if !actorCanManageOwners(actor) {
		return OwnerBulkResponse{}, ErrForbidden
	}
	if len(req.Items) == 0 {
		return OwnerBulkResponse{}, ErrEmptyBulk
	}
	phones := make([]string, len(req.Items))
	for index, item := range req.Items {
		phone, err := NormalizePhone(item.Phone)
		if err != nil {
			return OwnerBulkResponse{}, err
		}
		phones[index] = phone
	}
	owners, err := s.repo.CreateOwners(ctx, actor, req.Items, phones)
	if err != nil {
		return OwnerBulkResponse{}, err
	}
	return ownerBulkResponse(owners), nil
}

func (s *Service) GetOwner(ctx context.Context, actor Actor, id int64) (OwnerResponse, error) {
	owner, err := s.repo.FindOwnerByID(ctx, actor, id)
	if err != nil {
		return OwnerResponse{}, err
	}
	return NewOwnerResponse(owner), nil
}

func (s *Service) GetOwnerOverview(ctx context.Context, actor Actor, id int64) (OwnerOverviewResponse, error) {
	overview, err := s.repo.GetOwnerOverview(ctx, actor, id)
	if err != nil {
		return OwnerOverviewResponse{}, err
	}
	return NewOwnerOverviewResponse(overview), nil
}

func (s *Service) RestoreOwner(ctx context.Context, actor Actor, id int64) (OwnerResponse, error) {
	if !actorCanManageOwners(actor) {
		return OwnerResponse{}, ErrForbidden
	}
	owner, err := s.repo.RestoreOwner(ctx, id)
	if err != nil {
		return OwnerResponse{}, err
	}
	return NewOwnerResponse(owner), nil
}

func (s *Service) UpdateOwner(ctx context.Context, actor Actor, id int64, req UpdateOwnerRequest) (OwnerResponse, error) {
	return s.UpdateOwnerWithMeta(ctx, actor, id, req, RequestMeta{})
}

// UpdateOwnerWithMeta is UpdateOwner plus request metadata for the audit_logs entry. Kept as a
// separate entry point (rather than changing UpdateOwner's signature) so existing callers that
// don't have request metadata (e.g. bulk update, scraper's normal PATCH call) keep working
// unchanged - both paths still get audited, just with meta zero-valued when unavailable.
func (s *Service) UpdateOwnerWithMeta(ctx context.Context, actor Actor, id int64, req UpdateOwnerRequest, meta RequestMeta) (OwnerResponse, error) {
	if !actorCanManageOwners(actor) {
		return OwnerResponse{}, ErrForbidden
	}
	before, err := s.repo.FindOwnerByID(ctx, actor, id)
	if err != nil {
		return OwnerResponse{}, err
	}
	var phone *string
	if req.Phone != nil {
		normalized, err := NormalizePhone(*req.Phone)
		if err != nil {
			return OwnerResponse{}, err
		}
		phone = &normalized
	}
	owner, err := s.repo.UpdateOwner(ctx, id, req, phone)
	if err != nil {
		return OwnerResponse{}, err
	}
	response := NewOwnerResponse(owner)
	_ = s.repo.Audit(ctx, actor.ID, "owner.update", entityTypeOwner, id, NewOwnerResponse(before), response, meta)
	return response, nil
}

// SetTestingAccount lets an admin flag an owner as a testing account (a piposmart employee's
// demo/learning account, not a real prospective customer) so it's excluded from the sales/lead
// pipeline, or clear that flag.
func (s *Service) SetTestingAccount(ctx context.Context, actor Actor, id int64, isTesting bool) (OwnerResponse, error) {
	if !actorCanManageOwners(actor) {
		return OwnerResponse{}, ErrForbidden
	}
	owner, err := s.repo.SetTestingAccount(ctx, id, isTesting, actor.ID)
	if err != nil {
		return OwnerResponse{}, err
	}
	return NewOwnerResponse(owner), nil
}

func (s *Service) BulkUpdateOwners(ctx context.Context, actor Actor, req BulkOwnerUpdateRequest) (OwnerBulkResponse, error) {
	if !actorCanManageOwners(actor) {
		return OwnerBulkResponse{}, ErrForbidden
	}
	if len(req.Items) == 0 {
		return OwnerBulkResponse{}, ErrEmptyBulk
	}
	updates := make([]OwnerUpdateInput, 0, len(req.Items))
	for _, item := range req.Items {
		var phone *string
		if item.Phone != nil {
			normalized, err := NormalizePhone(*item.Phone)
			if err != nil {
				return OwnerBulkResponse{}, err
			}
			phone = &normalized
		}
		updates = append(updates, OwnerUpdateInput{
			ID: item.ID,
			Request: UpdateOwnerRequest{
				Code:      item.Code,
				Name:      item.Name,
				Phone:     item.Phone,
				Email:     item.Email,
				BrandName: item.BrandName,
				Province:  item.Province,
				City:      item.City,
				Address:   item.Address,
			},
			NormalizedPhone: phone,
		})
	}
	owners, err := s.repo.UpdateOwners(ctx, updates)
	if err != nil {
		return OwnerBulkResponse{}, err
	}
	return ownerBulkResponse(owners), nil
}

func (s *Service) DeleteOwner(ctx context.Context, actor Actor, id int64) error {
	if !actorCanManageOwners(actor) {
		return ErrForbidden
	}
	return s.repo.SoftDeleteOwner(ctx, id)
}

func (s *Service) ForceDeleteOwner(ctx context.Context, actor Actor, id int64) error {
	if !actorCanManageOwners(actor) {
		return ErrForbidden
	}
	return s.repo.ForceDeleteOwner(ctx, id)
}

func (s *Service) BulkDeleteOwners(ctx context.Context, actor Actor, ids []int64) (BulkActionResponse, error) {
	if !actorCanManageOwners(actor) {
		return BulkActionResponse{}, ErrForbidden
	}
	ids, err := normalizeIDs(ids)
	if err != nil {
		return BulkActionResponse{}, err
	}
	affected, err := s.repo.SoftDeleteOwners(ctx, ids)
	if err != nil {
		return BulkActionResponse{}, err
	}
	return BulkActionResponse{IDs: ids, Affected: affected}, nil
}

func (s *Service) BulkForceDeleteOwners(ctx context.Context, actor Actor, ids []int64) (BulkActionResponse, error) {
	if !actorCanManageOwners(actor) {
		return BulkActionResponse{}, ErrForbidden
	}
	ids, err := normalizeIDs(ids)
	if err != nil {
		return BulkActionResponse{}, err
	}
	affected, err := s.repo.ForceDeleteOwners(ctx, ids)
	if err != nil {
		return BulkActionResponse{}, err
	}
	return BulkActionResponse{IDs: ids, Affected: affected}, nil
}

func (s *Service) ListOutlets(ctx context.Context, actor Actor, ownerID int64, params ListParams) (OutletListResponse, error) {
	params = normalizeListParams(params)
	if params.Phone != "" {
		phone, err := NormalizePhone(params.Phone)
		if err == nil {
			params.Phone = phone
		}
	}
	outlets, total, err := s.repo.ListOutlets(ctx, actor, ownerID, params)
	if err != nil {
		return OutletListResponse{}, err
	}
	items := make([]OutletResponse, 0, len(outlets))
	for _, outlet := range outlets {
		items = append(items, NewOutletResponse(outlet))
	}
	return OutletListResponse{
		Items: items,
		Pagination: PaginationMeta{
			Page:  params.Page,
			Limit: resolveReturnedLimit(params.All, params.Limit, len(items), total),
			Total: total,
		},
	}, nil
}

func (s *Service) ListGlobalOutlets(ctx context.Context, actor Actor, params ListParams) (OutletOverviewListResponse, error) {
	params = normalizeListParams(params)
	if params.Phone != "" {
		phone, err := NormalizePhone(params.Phone)
		if err == nil {
			params.Phone = phone
		}
	}
	items, total, err := s.repo.ListGlobalOutlets(ctx, actor, params)
	if err != nil {
		return OutletOverviewListResponse{}, err
	}
	responses := make([]OutletOverviewResponse, 0, len(items))
	for _, item := range items {
		responses = append(responses, NewOutletOverviewResponse(item))
	}
	return OutletOverviewListResponse{
		Items: responses,
		Pagination: PaginationMeta{
			Page:  params.Page,
			Limit: resolveReturnedLimit(params.All, params.Limit, len(responses), total),
			Total: total,
		},
	}, nil
}

func (s *Service) CreateOutlet(ctx context.Context, actor Actor, ownerID int64, req CreateOutletRequest) (OutletResponse, error) {
	if !actorCanManageOwners(actor) {
		return OutletResponse{}, ErrForbidden
	}

	phone := req.Phone
	if phone == "" {
		owner, err := s.repo.FindOwnerByID(ctx, actor, ownerID)
		if err == nil && owner.Phone.Valid {
			phone = owner.Phone.String
		}
	}

	normalizedPhone, err := NormalizePhone(phone)
	if err != nil {
		return OutletResponse{}, err
	}
	outlet, err := s.repo.CreateOutlet(ctx, actor, ownerID, req, normalizedPhone)
	if err != nil {
		return OutletResponse{}, err
	}
	return NewOutletResponse(outlet), nil
}

func (s *Service) BulkCreateOutlets(ctx context.Context, actor Actor, ownerID int64, req BulkOutletCreateRequest) (OutletBulkResponse, error) {
	if !actorCanManageOwners(actor) {
		return OutletBulkResponse{}, ErrForbidden
	}
	if len(req.Items) == 0 {
		return OutletBulkResponse{}, ErrEmptyBulk
	}

	var fallbackPhone string
	owner, err := s.repo.FindOwnerByID(ctx, actor, ownerID)
	if err == nil && owner.Phone.Valid {
		fallbackPhone = owner.Phone.String
	}

	phones := make([]string, len(req.Items))
	for index, item := range req.Items {
		phone := item.Phone
		if phone == "" {
			phone = fallbackPhone
		}
		normalizedPhone, err := NormalizePhone(phone)
		if err != nil {
			return OutletBulkResponse{}, err
		}
		phones[index] = normalizedPhone
	}
	outlets, err := s.repo.CreateOutlets(ctx, actor, ownerID, req.Items, phones)
	if err != nil {
		return OutletBulkResponse{}, err
	}
	return outletBulkResponse(outlets), nil
}

func (s *Service) GetOutlet(ctx context.Context, actor Actor, ownerID, outletID int64) (OutletResponse, error) {
	outlet, err := s.repo.FindOutletByID(ctx, actor, ownerID, outletID)
	if err != nil {
		return OutletResponse{}, err
	}
	return NewOutletResponse(outlet), nil
}

func (s *Service) GetGlobalOutlet(ctx context.Context, actor Actor, outletID int64) (OutletDetailResponse, error) {
	item, err := s.repo.FindGlobalOutletByID(ctx, actor, outletID)
	if err != nil {
		return OutletDetailResponse{}, err
	}
	return NewOutletDetailResponse(item), nil
}

func (s *Service) RestoreOutlet(ctx context.Context, actor Actor, ownerID, outletID int64) (OutletResponse, error) {
	if !actorCanManageOwners(actor) {
		return OutletResponse{}, ErrForbidden
	}
	outlet, err := s.repo.RestoreOutlet(ctx, ownerID, outletID)
	if err != nil {
		return OutletResponse{}, err
	}
	return NewOutletResponse(outlet), nil
}

func (s *Service) UpdateOutlet(ctx context.Context, actor Actor, ownerID, outletID int64, req UpdateOutletRequest) (OutletResponse, error) {
	return s.UpdateOutletWithMeta(ctx, actor, ownerID, outletID, req, RequestMeta{})
}

// UpdateOutletWithMeta is UpdateOutlet plus request metadata for the audit_logs entry - same
// rationale as UpdateOwnerWithMeta above.
func (s *Service) UpdateOutletWithMeta(ctx context.Context, actor Actor, ownerID, outletID int64, req UpdateOutletRequest, meta RequestMeta) (OutletResponse, error) {
	if !actorCanManageOwners(actor) {
		return OutletResponse{}, ErrForbidden
	}
	before, err := s.repo.FindOutletByID(ctx, actor, ownerID, outletID)
	if err != nil {
		return OutletResponse{}, err
	}
	var normalizedPhone *string
	if req.Phone != nil {
		phone := *req.Phone
		if phone == "" {
			owner, err := s.repo.FindOwnerByID(ctx, actor, ownerID)
			if err == nil && owner.Phone.Valid {
				phone = owner.Phone.String
			}
		}
		normalized, err := NormalizePhone(phone)
		if err != nil {
			return OutletResponse{}, err
		}
		normalizedPhone = &normalized
	}
	outlet, err := s.repo.UpdateOutlet(ctx, ownerID, outletID, req, normalizedPhone)
	if err != nil {
		return OutletResponse{}, err
	}
	response := NewOutletResponse(outlet)
	_ = s.repo.Audit(ctx, actor.ID, "outlet.update", entityTypeOutlet, outletID, NewOutletResponse(before), response, meta)
	return response, nil
}

func (s *Service) BulkUpdateOutlets(ctx context.Context, actor Actor, ownerID int64, req BulkOutletUpdateRequest) (OutletBulkResponse, error) {
	if !actorCanManageOwners(actor) {
		return OutletBulkResponse{}, ErrForbidden
	}
	if len(req.Items) == 0 {
		return OutletBulkResponse{}, ErrEmptyBulk
	}

	var fallbackPhone string
	owner, err := s.repo.FindOwnerByID(ctx, actor, ownerID)
	if err == nil && owner.Phone.Valid {
		fallbackPhone = owner.Phone.String
	}

	updates := make([]OutletUpdateInput, 0, len(req.Items))
	for _, item := range req.Items {
		var normalizedPhone *string
		if item.Phone != nil {
			phone := *item.Phone
			if phone == "" {
				phone = fallbackPhone
			}
			normalized, err := NormalizePhone(phone)
			if err != nil {
				return OutletBulkResponse{}, err
			}
			normalizedPhone = &normalized
		}
		updates = append(updates, OutletUpdateInput{
			ID: item.ID,
			Request: UpdateOutletRequest{
				Code:        item.Code,
				Name:        item.Name,
				Phone:       item.Phone,
				Province:    item.Province,
				City:        item.City,
				District:    item.District,
				SubDistrict: item.SubDistrict,
				Address:     item.Address,
			},
			NormalizedPhone: normalizedPhone,
		})
	}
	outlets, err := s.repo.UpdateOutlets(ctx, ownerID, updates)
	if err != nil {
		return OutletBulkResponse{}, err
	}
	return outletBulkResponse(outlets), nil
}

func (s *Service) DeleteOutlet(ctx context.Context, actor Actor, ownerID, outletID int64) error {
	if !actorCanManageOwners(actor) {
		return ErrForbidden
	}
	return s.repo.SoftDeleteOutlet(ctx, ownerID, outletID)
}

func (s *Service) ForceDeleteOutlet(ctx context.Context, actor Actor, ownerID, outletID int64) error {
	if !actorCanManageOwners(actor) {
		return ErrForbidden
	}
	return s.repo.ForceDeleteOutlet(ctx, ownerID, outletID)
}

func (s *Service) BulkDeleteOutlets(ctx context.Context, actor Actor, ownerID int64, ids []int64) (BulkActionResponse, error) {
	if !actorCanManageOwners(actor) {
		return BulkActionResponse{}, ErrForbidden
	}
	ids, err := normalizeIDs(ids)
	if err != nil {
		return BulkActionResponse{}, err
	}
	affected, err := s.repo.SoftDeleteOutlets(ctx, ownerID, ids)
	if err != nil {
		return BulkActionResponse{}, err
	}
	return BulkActionResponse{IDs: ids, Affected: affected}, nil
}

func (s *Service) BulkForceDeleteOutlets(ctx context.Context, actor Actor, ownerID int64, ids []int64) (BulkActionResponse, error) {
	if !actorCanManageOwners(actor) {
		return BulkActionResponse{}, ErrForbidden
	}
	ids, err := normalizeIDs(ids)
	if err != nil {
		return BulkActionResponse{}, err
	}
	affected, err := s.repo.ForceDeleteOutlets(ctx, ownerID, ids)
	if err != nil {
		return BulkActionResponse{}, err
	}
	return BulkActionResponse{IDs: ids, Affected: affected}, nil
}

func normalizeListParams(params ListParams) ListParams {
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
	params.SubscriptionStatus = strings.ToUpper(strings.TrimSpace(params.SubscriptionStatus))
	params.SubscriptionMonth = strings.TrimSpace(params.SubscriptionMonth)
	params.OwnerKind = normalizeOwnerKind(params.OwnerKind)
	params.Scope = normalizeScope(params.Scope)
	params.Sort = strings.TrimSpace(params.Sort)
	return params
}

func resolveReturnedLimit(all bool, limit int, itemCount int, total int64) int {
	if !all {
		return limit
	}
	if total == 0 {
		return 0
	}
	return itemCount
}

func normalizeScope(scope string) string {
	switch strings.ToUpper(strings.TrimSpace(scope)) {
	case ScopeDeleted:
		return ScopeDeleted
	case ScopeAll:
		return ScopeAll
	default:
		return ScopeActive
	}
}

func normalizeOwnerKind(kind string) string {
	switch strings.ToUpper(strings.TrimSpace(kind)) {
	case OwnerKindNonRegister, "NONREG", "NON_REGISTERED", "NON-REGISTER", "NON REGISTER":
		return OwnerKindNonRegister
	case OwnerKindAll:
		return OwnerKindAll
	default:
		return OwnerKindRegistered
	}
}

func normalizeIDs(ids []int64) ([]int64, error) {
	if len(ids) == 0 {
		return nil, ErrEmptyBulk
	}
	seen := make(map[int64]struct{}, len(ids))
	normalized := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id < 1 {
			return nil, ErrEmptyBulk
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		normalized = append(normalized, id)
	}
	if len(normalized) == 0 {
		return nil, ErrEmptyBulk
	}
	return normalized, nil
}

func ownerBulkResponse(owners []Owner) OwnerBulkResponse {
	items := make([]OwnerResponse, 0, len(owners))
	for _, owner := range owners {
		items = append(items, NewOwnerResponse(owner))
	}
	return OwnerBulkResponse{Items: items, Total: len(items)}
}

func outletBulkResponse(outlets []Outlet) OutletBulkResponse {
	items := make([]OutletResponse, 0, len(outlets))
	for _, outlet := range outlets {
		items = append(items, NewOutletResponse(outlet))
	}
	return OutletBulkResponse{Items: items, Total: len(items)}
}

func actorCanManageOwners(actor Actor) bool {
	return actor.RoleCode == RoleAdmin
}

func (s *Service) ExportOwnerOutlets(ctx context.Context, actor Actor, params ListParams) ([]map[string]any, error) {
	params = normalizeListParams(params)
	return s.repo.ExportOwnerOutlets(ctx, actor, params)
}

func (s *Service) ExportGlobalOutlets(ctx context.Context, actor Actor, params ListParams) ([]map[string]any, error) {
	params = normalizeListParams(params)
	return s.repo.ExportGlobalOutlets(ctx, actor, params)
}
