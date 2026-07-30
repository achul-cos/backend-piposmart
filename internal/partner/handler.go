package partner

import (
	"net/http"
	"strconv"
	"time"

	"backend_crm_piposmart/internal/identity"
	"backend_crm_piposmart/internal/platform/httpx"

	"github.com/gin-gonic/gin"
)

// Handler handles partner-related HTTP requests.
type Handler struct {
	service *Service
}

// NewHandler creates a new Partner handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers partner routes on the given router group.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	partnerTypes := rg.Group("/partner-types")
	{
		partnerTypes.POST("", h.CreatePartnerType)
		partnerTypes.GET("/:id", h.GetPartnerTypeByID)
		partnerTypes.GET("/:id/histories", h.ListPartnerTypeHistories)
		partnerTypes.GET("", h.ListPartnerTypes)
		partnerTypes.PUT("/:id", h.UpdatePartnerType)

		partnerTypeGroup := partnerTypes.Group("/:id")
		{
			partnerTypeGroup.POST("/commission-rules", h.CreateCommissionRule)
			partnerTypeGroup.GET("/commission-rules", h.ListCommissionRules)
			partnerTypeGroup.GET("/commission-rules/:ruleID", h.GetCommissionRule)
			partnerTypeGroup.PATCH("/commission-rules/:ruleID/deactivate", h.DeactivateCommissionRule)
		}
	}

	partners := rg.Group("/partners")
	{
		partners.POST("", h.CreatePartner)
		// static route before param routes
		partners.GET("/code/:code", h.GetPartnerByCode)
		partners.GET("", h.ListPartners)
		partners.GET("/all", h.ListAllPartners)
		partners.GET("/all-deleted", h.ListAllPartners)
		// Group routes that use partnerID
		partnerGroup := partners.Group("/:partnerID")
		{
			partnerGroup.GET("", h.GetPartnerByID)       // GET /partners/:partnerID
			partnerGroup.PUT("", h.UpdatePartner)        // PUT /partners/:partnerID
			partnerGroup.DELETE("", h.DeactivatePartner) // DELETE /partners/:partnerID

			partnerGroup.GET("/assignments/active", h.GetActiveAssignmentForPartner)
			partnerGroup.GET("/assignments", h.ListPartnerAssignments)
			partnerGroup.POST("/assignments", h.AssignPIC)
			partnerGroup.DELETE("/assignments/release", h.ReleasePartner)

			partnerGroup.GET("/interactions", h.ListInteractions)
			partnerGroup.GET("/interactions/all", h.ListAllInteractions)
			partnerGroup.GET("/interactions/all-deleted", h.ListAllInteractions)
			partnerGroup.POST("/interactions", h.RecordInteraction)

			partnerGroup.GET("/referrals", h.ListReferrals)
			partnerGroup.POST("/referrals", h.CreateReferral)
			partnerGroup.GET("/activity", h.GetMonthlyActivityStatus)

			partnerGroup.POST("/commissions/sync", h.SyncCommissions)
			partnerGroup.GET("/commissions", h.ListCommissions)
			partnerGroup.GET("/commissions/all", h.ListAllCommissions)
			partnerGroup.GET("/commissions/all-deleted", h.ListAllCommissions)
			partnerGroup.GET("/commissions/:commissionID", h.GetCommission)
			partnerGroup.PATCH("/commissions/:commissionID/approve", h.ApproveCommission)
			partnerGroup.PATCH("/commissions/:commissionID/pay", h.PayCommission)
			partnerGroup.PATCH("/commissions/:commissionID/cancel", h.CancelCommission)

			partnerGroup.POST("/payouts", h.CreatePayout)
			partnerGroup.GET("/payouts", h.ListPayouts)
			partnerGroup.GET("/payouts/all", h.ListAllPayouts)
			partnerGroup.GET("/payouts/all-deleted", h.ListAllPayouts)
			partnerGroup.GET("/payouts/:payoutID", h.GetPayout)
			partnerGroup.PATCH("/payouts/:payoutID/pay", h.PayPayout)
			partnerGroup.PATCH("/payouts/:payoutID/cancel", h.CancelPayout)
		}
	}
}

/* ---------- PartnerType ---------- */

// CreatePartnerType godoc
// @Summary Create partner type
// @Description Create a new partner type
// @Tags partner-types
// @Accept json
// @Produce json
// @Param request body CreatePartnerTypeRequest true "Partner type data"
// @Success 201 {object} PartnerTypeResponse
// @Failure 400 {object} httpx.ErrorEnvelope
// @Failure 409 {object} httpx.ErrorEnvelope
// @Router /partner-types [post]
func (h *Handler) CreatePartnerType(c *gin.Context) {
	var req CreatePartnerTypeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "Payload request tidak valid", gin.H{"error": err.Error()})
		return
	}
	actor, _ := identity.CurrentUser(c)
	resp, err := h.service.CreatePartnerTypeWithMeta(c.Request.Context(), actor, req, requestMeta(c))
	if err != nil {
		switch err {
		case ErrForbidden:
			httpx.Error(c, http.StatusForbidden, "FORBIDDEN", "not allowed to perform this action", nil)
		case ErrDuplicateType:
			httpx.Error(c, http.StatusConflict, "PARTNER_TYPE_CODE_EXISTS", "partner type code already exists", nil)
		case ErrInvalidCommissionRate, ErrInvalidMoney, ErrInvalidCommissionMode:
			httpx.Error(c, http.StatusBadRequest, "INVALID_COMMISSION_VALUE", err.Error(), nil)
		default:
			httpx.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create partner type", nil)
		}
		return
	}
	httpx.Success(c, http.StatusCreated, resp)
}

// GetPartnerTypeByID godoc
// @Summary Get partner type by ID
// @Description Retrieve a partner type by its ID
// @Tags partner-types
// @Accept json
// @Produce json
// @Param id path int64 true "Partner type ID"
// @Success 200 {object} PartnerTypeResponse
// @Failure 404 {object} httpx.ErrorEnvelope
// @Router /partner-types/{id} [get]
func (h *Handler) GetPartnerTypeByID(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "ID tidak valid", nil)
		return
	}
	resp, err := h.service.GetPartnerTypeByID(c.Request.Context(), id)
	if err != nil {
		if err == ErrNotFound {
			httpx.Error(c, http.StatusNotFound, "NOT_FOUND", "partner type not found", nil)
		} else {
			httpx.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get partner type", nil)
		}
		return
	}
	httpx.Success(c, http.StatusOK, resp)
}

// ListPartnerTypeHistories godoc
// @Summary List partner type histories
// @Description Retrieve audit history of partner type commission value and related master changes
// @Tags partner-types
// @Accept json
// @Produce json
// @Param id path int64 true "Partner type ID"
// @Success 200 {object} PartnerTypeHistoryListResponse
// @Failure 404 {object} httpx.ErrorEnvelope
// @Router /partner-types/{id}/histories [get]
func (h *Handler) ListPartnerTypeHistories(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "ID tidak valid", nil)
		return
	}
	resp, err := h.service.ListPartnerTypeHistories(c.Request.Context(), id)
	if err != nil {
		if err == ErrNotFound {
			httpx.Error(c, http.StatusNotFound, "NOT_FOUND", "partner type not found", nil)
		} else {
			httpx.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get partner type histories", nil)
		}
		return
	}
	httpx.Success(c, http.StatusOK, resp)
}

// ListPartnerTypes godoc
// @Summary List partner types
// @Description Get a list of all partner types
// @Tags partner-types
// @Accept json
// @Produce json
// @Success 200 {object} PartnerTypeListResponse
// @Router /partner-types [get]
func (h *Handler) ListPartnerTypes(c *gin.Context) {
	resp, err := h.service.ListPartnerTypes(c.Request.Context())
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list partner types", nil)
		return
	}
	httpx.Success(c, http.StatusOK, PartnerTypeListResponse{
		Items: resp,
		Pagination: PaginationMeta{
			Page:  1,
			Limit: len(resp),
			Total: int64(len(resp)),
		},
	})
}

// UpdatePartnerType godoc
// @Summary Update partner type
// @Description Update an existing partner type
// @Tags partner-types
// @Accept json
// @Produce json
// @Param id path int64 true "Partner type ID"
// @Param request body UpdatePartnerTypeRequest true "Partner type data"
// @Success 200 {object} PartnerTypeResponse
// @Failure 400 {object} httpx.ErrorEnvelope
// @Failure 404 {object} httpx.ErrorEnvelope
// @Router /partner-types/{id} [put]
func (h *Handler) UpdatePartnerType(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "ID tidak valid", nil)
		return
	}
	var req UpdatePartnerTypeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "Payload request tidak valid", gin.H{"error": err.Error()})
		return
	}
	actor, _ := identity.CurrentUser(c)
	resp, err := h.service.UpdatePartnerTypeWithMeta(c.Request.Context(), actor, id, req, requestMeta(c))
	if err != nil {
		switch err {
		case ErrNotFound:
			httpx.Error(c, http.StatusNotFound, "NOT_FOUND", "partner type not found", nil)
		case ErrForbidden:
			httpx.Error(c, http.StatusForbidden, "FORBIDDEN", "not allowed to perform this action", nil)
		case ErrInvalidCommissionRate, ErrInvalidMoney, ErrInvalidCommissionMode:
			httpx.Error(c, http.StatusBadRequest, "INVALID_COMMISSION_VALUE", err.Error(), nil)
		default:
			httpx.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to update partner type", nil)
		}
		return
	}
	httpx.Success(c, http.StatusOK, resp)
}

/* ---------- Partner ---------- */

// CreatePartner godoc
// @Summary Create partner
// @Description Create a new partner
// @Tags partners
// @Accept json
// @Produce json
// @Param request body CreatePartnerRequest true "Partner data"
// @Success 201 {object} PartnerResponse
// @Failure 400 {object} httpx.ErrorEnvelope
// @Failure 409 {object} httpx.ErrorEnvelope
// @Router /partners [post]
func (h *Handler) CreatePartner(c *gin.Context) {
	var req CreatePartnerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "Payload request tidak valid", gin.H{"error": err.Error()})
		return
	}
	actor, _ := identity.CurrentUser(c)
	resp, err := h.service.CreatePartner(c.Request.Context(), actor, req)
	if err != nil {
		switch err {
		case ErrDuplicatePartner:
			httpx.Error(c, http.StatusConflict, "PARTNER_CODE_EXISTS", "partner code already exists", nil)
		default:
			httpx.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create partner", nil)
		}
		return
	}
	httpx.Success(c, http.StatusCreated, resp)
}

// GetPartnerByID godoc
// @Summary Get partner by ID
// @Description Retrieve a partner by its ID
// @Tags partners
// @Accept json
// @Produce json
// @Param partnerID path int64 true "Partner ID"
// @Success 200 {object} PartnerResponse
// @Failure 404 {object} httpx.ErrorEnvelope
// @Router /partners/{partnerID} [get]
func (h *Handler) GetPartnerByID(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("partnerID"), 10, 64)
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "ID tidak valid", nil)
		return
	}
	resp, err := h.service.GetPartnerByID(c.Request.Context(), id)
	if err != nil {
		if err == ErrNotFound {
			httpx.Error(c, http.StatusNotFound, "NOT_FOUND", "partner not found", nil)
		} else {
			httpx.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get partner", nil)
		}
		return
	}
	httpx.Success(c, http.StatusOK, resp)
}

// GetPartnerByCode godoc
// @Summary Get partner by code
// @Description Retrieve a partner by its code
// @Tags partners
// @Accept json
// @Produce json
// @Param code path string true "Partner code"
// @Success 200 {object} PartnerResponse
// @Failure 404 {object} httpx.ErrorEnvelope
// @Router /partners/code/{code} [get]
func (h *Handler) GetPartnerByCode(c *gin.Context) {
	code := c.Param("code")
	resp, err := h.service.GetPartnerByCode(c.Request.Context(), code)
	if err != nil {
		if err == ErrNotFound {
			httpx.Error(c, http.StatusNotFound, "NOT_FOUND", "partner not found", nil)
		} else {
			httpx.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get partner", nil)
		}
		return
	}
	httpx.Success(c, http.StatusOK, resp)
}

// ListPartners godoc
// @Summary List partners
// @Description Get a paginated list of partners with optional search
// @Tags partners
// @Accept json
// @Produce json
// @Param limit query int false "Page size (default 10)"
// @Param offset query int false "Page offset (default 0)"
// @Param search query string false "Search term (name or code)"
// @Success 200 {object} PartnerListResponse
// @Router /partners [get]
func (h *Handler) ListPartners(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	search := c.DefaultQuery("search", "")
	resp, total, err := h.service.ListPartners(c.Request.Context(), limit, offset, search)
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list partners", nil)
		return
	}
	httpx.Success(c, http.StatusOK, PartnerListResponse{
		Items: resp,
		Pagination: PaginationMeta{
			Page:  pageFromOffsetLimit(offset, limit),
			Limit: limit,
			Total: total,
		},
	})
}

func (h *Handler) ListAllPartners(c *gin.Context) {
	search := c.DefaultQuery("search", "")
	resp, total, err := h.service.ListPartners(c.Request.Context(), 0, 0, search)
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list partners", nil)
		return
	}
	httpx.Success(c, http.StatusOK, PartnerListResponse{
		Items: resp,
		Pagination: PaginationMeta{
			Page:  1,
			Limit: len(resp),
			Total: total,
		},
	})
}

// UpdatePartner godoc
// @Summary Update partner
// @Description Update an existing partner
// @Tags partners
// @Accept json
// @Produce json
// @Param partnerID path int64 true "Partner ID"
// @Param request body UpdatePartnerRequest true "Partner data"
// @Success 200 {object} PartnerResponse
// @Failure 400 {object} httpx.ErrorEnvelope
// @Failure 404 {object} httpx.ErrorEnvelope
// @Router /partners/{partnerID} [put]
func (h *Handler) UpdatePartner(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("partnerID"), 10, 64)
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "ID tidak valid", nil)
		return
	}
	var req UpdatePartnerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "Payload request tidak valid", gin.H{"error": err.Error()})
		return
	}
	resp, err := h.service.UpdatePartner(c.Request.Context(), id, req)
	if err != nil {
		if err == ErrNotFound {
			httpx.Error(c, http.StatusNotFound, "NOT_FOUND", "partner not found", nil)
		} else {
			httpx.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to update partner", nil)
		}
		return
	}
	httpx.Success(c, http.StatusOK, resp)
}

// DeactivatePartner godoc
// @Summary Deactivate partner (soft delete)
// @Description Deactivate a partner by setting status to INACTIVE
// @Tags partners
// @Accept json
// @Produce json
// @Param partnerID path int64 true "Partner ID"
// @Success 204 {object} nil
// @Failure 404 {object} httpx.ErrorEnvelope
// @Router /partners/{partnerID} [delete]
func (h *Handler) DeactivatePartner(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("partnerID"), 10, 64)
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "ID tidak valid", nil)
		return
	}
	if err := h.service.DeactivatePartner(c.Request.Context(), id); err != nil {
		if err == ErrNotFound {
			httpx.Error(c, http.StatusNotFound, "NOT_FOUND", "partner not found", nil)
		} else {
			httpx.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to deactivate partner", nil)
		}
		return
	}
	httpx.Success(c, http.StatusNoContent, nil)
}

/* ---------- PartnerAssignment ---------- */

// AssignPIC godoc
// @Summary Assign PIC to partner
// @Description Assign a Person In Charge to a partner (deactivates any existing active assignment)
// @Tags partner-assignments
// @Accept json
// @Produce json
// @Param partnerID path int64 true "Partner ID"
// @Param request body CreatePartnerAssignmentRequest true "Assignment data"
// @Success 201 {object} PartnerAssignmentResponse
// @Failure 400 {object} httpx.ErrorEnvelope
// @Failure 404 {object} httpx.ErrorEnvelope
// @Router /partners/{partnerID}/assignments [post]
func (h *Handler) AssignPIC(c *gin.Context) {
	partnerID, err := strconv.ParseInt(c.Param("partnerID"), 10, 64)
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid partner ID", nil)
		return
	}
	var req CreatePartnerAssignmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "Payload request tidak valid", gin.H{"error": err.Error()})
		return
	}
	// AssignedByID can be taken from auth context or request body
	assignedByID := req.AssignedByID
	if assignedByID == nil || *assignedByID == 0 {
		if user, ok := identity.CurrentUser(c); ok && user.ID > 0 {
			assignedByID = &user.ID
		}
	}
	resp, err := h.service.AssignPIC(c.Request.Context(), partnerID, req.UserID, assignedByID)
	if err != nil {
		switch err {
		case ErrNotFound:
			httpx.Error(c, http.StatusNotFound, "NOT_FOUND", "partner or user not found", nil)
		case ErrInvalidAssignment:
			httpx.Error(c, http.StatusBadRequest, "INVALID_ASSIGNMENT", "invalid partner assignment", nil)
		default:
			httpx.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to assign PIC", nil)
		}
		return
	}
	httpx.Success(c, http.StatusCreated, resp)
}

// GetActiveAssignmentForPartner godoc
// @Summary Get active assignment for partner
// @Description Get the currently active PIC assignment for a partner
// @Tags partner-assignments
// @Accept json
// @Produce json
// @Param partnerID path int64 true "Partner ID"
// @Success 200 {object} PartnerAssignmentResponse
// @Failure 404 {object} httpx.ErrorEnvelope
// @Router /partners/{partnerID}/assignments/active [get]
func (h *Handler) GetActiveAssignmentForPartner(c *gin.Context) {
	partnerID, err := strconv.ParseInt(c.Param("partnerID"), 10, 64)
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid partner ID", nil)
		return
	}
	resp, err := h.service.GetActiveAssignmentForPartner(c.Request.Context(), partnerID)
	if err != nil {
		if err == ErrNotFound {
			httpx.Error(c, http.StatusNotFound, "NOT_FOUND", "no active assignment found", nil)
		} else {
			httpx.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get active assignment", nil)
		}
		return
	}
	httpx.Success(c, http.StatusOK, resp)
}

// ListPartnerAssignments godoc
// @Summary List partner assignments
// @Description Get all assignments (history) for a partner
// @Tags partner-assignments
// @Accept json
// @Produce json
// @Param partnerID path int64 true "Partner ID"
// @Success 200 {object} PartnerAssignmentListResponse
// @Router /partners/{partnerID}/assignments [get]
func (h *Handler) ListPartnerAssignments(c *gin.Context) {
	partnerID, err := strconv.ParseInt(c.Param("partnerID"), 10, 64)
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid partner ID", nil)
		return
	}
	list, err := h.service.ListPartnerAssignments(c.Request.Context(), partnerID)
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list partner assignments", nil)
		return
	}
	httpx.Success(c, http.StatusOK, PartnerAssignmentListResponse{
		Items: list,
		Pagination: PaginationMeta{
			Page:  1,
			Limit: len(list),
			Total: int64(len(list)),
		},
	})
}

// ReleasePartner godoc
// @Summary Release partner from PIC
// @Description Remove the active PIC assignment from a partner
// @Tags partner-assignments
// @Accept json
// @Produce json
// @Param partnerID path int64 true "Partner ID"
// @Success 204 {object} nil
// @Failure 404 {object} httpx.ErrorEnvelope
// @Router /partners/{partnerID}/assignments/release [delete]
func (h *Handler) ReleasePartner(c *gin.Context) {
	partnerID, err := strconv.ParseInt(c.Param("partnerID"), 10, 64)
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid partner ID", nil)
		return
	}
	if err := h.service.ReleasePartner(c.Request.Context(), partnerID); err != nil {
		httpx.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to release partner", nil)
		return
	}
	httpx.Success(c, http.StatusNoContent, nil)
}

/* ---------- PartnerInteraction ---------- */

// RecordInteraction godoc
// @Summary Record partner interaction
// @Description Record a call or chat interaction with a partner
// @Tags partner-interactions
// @Accept json
// @Produce json
// @Param partnerID path int64 true "Partner ID"
// @Param request body CreatePartnerInteractionRequest true "Interaction data"
// @Success 201 {object} PartnerInteractionResponse
// @Failure 400 {object} httpx.ErrorEnvelope
// @Failure 404 {object} httpx.ErrorEnvelope
// @Router /partners/{partnerID}/interactions [post]
func (h *Handler) RecordInteraction(c *gin.Context) {
	partnerID, err := strconv.ParseInt(c.Param("partnerID"), 10, 64)
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid partner ID", nil)
		return
	}
	var req CreatePartnerInteractionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "Payload request tidak valid", gin.H{"error": err.Error()})
		return
	}
	var interactionAt *time.Time
	if req.InteractionAt != nil {
		t := *req.InteractionAt
		interactionAt = &t
	}
	resp, err := h.service.RecordInteraction(c.Request.Context(), partnerID, req.InteractionType, interactionAt, req.Note)
	if err != nil {
		if err == ErrNotFound {
			httpx.Error(c, http.StatusNotFound, "NOT_FOUND", "partner not found", nil)
		} else {
			httpx.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to record interaction", nil)
		}
		return
	}
	httpx.Success(c, http.StatusCreated, resp)
}

// ListInteractions godoc
// @Summary List partner interactions
// @Description Get paginated interaction history for a partner
// @Tags partner-interactions
// @Accept json
// @Produce json
// @Param partnerID path int64 true "Partner ID"
// @Param limit query int false "Page size (default 10)"
// @Param offset query int false "Page offset (default 0)"
// @Success 200 {object} PartnerInteractionListResponse
// @Router /partners/{partnerID}/interactions [get]
func (h *Handler) ListInteractions(c *gin.Context) {
	partnerID, err := strconv.ParseInt(c.Param("partnerID"), 10, 64)
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid partner ID", nil)
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	list, total, err := h.service.ListInteractions(c.Request.Context(), partnerID, limit, offset)
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list interactions", nil)
		return
	}
	httpx.Success(c, http.StatusOK, PartnerInteractionListResponse{
		Items: list,
		Pagination: PaginationMeta{
			Page:  pageFromOffsetLimit(offset, limit),
			Limit: limit,
			Total: total,
		},
	})
}

func (h *Handler) ListAllInteractions(c *gin.Context) {
	partnerID, err := strconv.ParseInt(c.Param("partnerID"), 10, 64)
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid partner ID", nil)
		return
	}
	list, total, err := h.service.ListInteractions(c.Request.Context(), partnerID, 0, 0)
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list interactions", nil)
		return
	}
	httpx.Success(c, http.StatusOK, PartnerInteractionListResponse{
		Items: list,
		Pagination: PaginationMeta{
			Page:  1,
			Limit: len(list),
			Total: total,
		},
	})
}

/* ---------- PartnerReferral ---------- */

// CreateReferral godoc
// @Summary Create partner referral
// @Description Create a referral from a partner to a lead (prevents duplicate partner-lead referrals)
// @Tags partner-referrals
// @Accept json
// @Produce json
// @Param partnerID path int64 true "Partner ID"
// @Param request body CreatePartnerReferralRequest true "Referral data"
// @Success 201 {object} PartnerReferralResponse
// @Failure 400 {object} httpx.ErrorEnvelope
// @Failure 404 {object} httpx.ErrorEnvelope
// @Failure 409 {object} httpx.ErrorEnvelope
// @Router /partners/{partnerID}/referrals [post]
func (h *Handler) CreateReferral(c *gin.Context) {
	partnerID, err := strconv.ParseInt(c.Param("partnerID"), 10, 64)
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid partner ID", nil)
		return
	}
	var req CreatePartnerReferralRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "Payload request tidak valid", gin.H{"error": err.Error()})
		return
	}
	var referralDate *time.Time
	if req.ReferralDate != nil {
		t := *req.ReferralDate
		referralDate = &t
	}
	resp, err := h.service.CreateReferral(c.Request.Context(), partnerID, req.LeadID, referralDate, req.Notes)
	if err != nil {
		switch err {
		case ErrNotFound:
			httpx.Error(c, http.StatusNotFound, "NOT_FOUND", "partner not found", nil)
		case ErrDuplicateReferral:
			httpx.Error(c, http.StatusConflict, "DUPLICATE_REFERRAL", "referral already exists for this partner-lead pair", nil)
		default:
			httpx.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create referral", nil)
		}
		return
	}
	httpx.Success(c, http.StatusCreated, resp)
}

// GetMonthlyActivityStatus godoc
// @Summary Get partner monthly activity status
// @Description BELUM_MEMBERIKAN_REFERAL / TELAH_MEMBERIKAN_REFERAL for the given month (defaults to current month)
// @Tags partner-referrals
// @Accept json
// @Produce json
// @Param partnerID path int64 true "Partner ID"
// @Param month query string false "YYYY-MM, defaults to current month"
// @Success 200 {object} PartnerActivityStatusResponse
// @Router /partners/{partnerID}/activity [get]
func (h *Handler) GetMonthlyActivityStatus(c *gin.Context) {
	partnerID, err := strconv.ParseInt(c.Param("partnerID"), 10, 64)
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid partner ID", nil)
		return
	}
	year, month := time.Now().Year(), int(time.Now().Month())
	if raw := c.Query("month"); raw != "" {
		parsed, err := time.Parse("2006-01", raw)
		if err != nil {
			httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid month, expected YYYY-MM", nil)
			return
		}
		year, month = parsed.Year(), int(parsed.Month())
	}
	resp, err := h.service.GetMonthlyActivityStatus(c.Request.Context(), partnerID, year, month)
	if err != nil {
		writeCommissionError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, resp)
}

// ListReferrals godoc
// @Summary List partner referrals
// @Description Get all referrals made by a partner
// @Tags partner-referrals
// @Accept json
// @Produce json
// @Param partnerID path int64 true "Partner ID"
// @Success 200 {object} PartnerReferralListResponse
// @Router /partners/{partnerID}/referrals [get]
func (h *Handler) ListReferrals(c *gin.Context) {
	partnerID, err := strconv.ParseInt(c.Param("partnerID"), 10, 64)
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid partner ID", nil)
		return
	}
	list, err := h.service.ListReferrals(c.Request.Context(), partnerID)
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list referrals", nil)
		return
	}
	httpx.Success(c, http.StatusOK, PartnerReferralListResponse{
		Items: list,
		Pagination: PaginationMeta{
			Page:  1,
			Limit: len(list),
			Total: int64(len(list)),
		},
	})
}

/* ---------- PartnerCommission ---------- */

// SyncCommissions godoc
// @Summary Sync commissions from confirmed closings
// @Description Scan confirmed closings tied to this partner's referrals and create PENDING commission records for any not yet synced (ADMIN/SUPERVISOR only, idempotent)
// @Tags partner-commissions
// @Accept json
// @Produce json
// @Param partnerID path int64 true "Partner ID"
// @Success 200 {object} SyncCommissionsResponse
// @Failure 403 {object} httpx.ErrorEnvelope
// @Failure 404 {object} httpx.ErrorEnvelope
// @Router /partners/{partnerID}/commissions/sync [post]
func (h *Handler) SyncCommissions(c *gin.Context) {
	partnerID, err := strconv.ParseInt(c.Param("partnerID"), 10, 64)
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid partner ID", nil)
		return
	}
	actor, _ := identity.CurrentUser(c)
	resp, err := h.service.SyncCommissions(c.Request.Context(), actor, partnerID)
	if err != nil {
		writeCommissionError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, resp)
}

// ListCommissions godoc
// @Summary List partner commissions
// @Description Get a paginated list of commissions earned by a partner, optionally filtered by status
// @Tags partner-commissions
// @Accept json
// @Produce json
// @Param partnerID path int64 true "Partner ID"
// @Param status query string false "Filter by status (PENDING, APPROVED, PAID, CANCELLED)"
// @Param page query int false "Page number (default 1)"
// @Param limit query int false "Page size (default 10)"
// @Success 200 {object} PartnerCommissionListResponse
// @Router /partners/{partnerID}/commissions [get]
func (h *Handler) ListCommissions(c *gin.Context) {
	partnerID, err := strconv.ParseInt(c.Param("partnerID"), 10, 64)
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid partner ID", nil)
		return
	}
	status := c.DefaultQuery("status", "")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	resp, err := h.service.ListCommissions(c.Request.Context(), partnerID, status, page, limit)
	if err != nil {
		writeCommissionError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, resp)
}

func (h *Handler) ListAllCommissions(c *gin.Context) {
	partnerID, err := strconv.ParseInt(c.Param("partnerID"), 10, 64)
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid partner ID", nil)
		return
	}
	status := c.DefaultQuery("status", "")
	resp, err := h.service.ListCommissions(c.Request.Context(), partnerID, status, 1, 0)
	if err != nil {
		writeCommissionError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, resp)
}

// GetCommission godoc
// @Summary Get partner commission detail
// @Description Retrieve a single commission by ID, scoped to the given partner
// @Tags partner-commissions
// @Accept json
// @Produce json
// @Param partnerID path int64 true "Partner ID"
// @Param commissionID path int64 true "Commission ID"
// @Success 200 {object} PartnerCommissionResponse
// @Failure 404 {object} httpx.ErrorEnvelope
// @Router /partners/{partnerID}/commissions/{commissionID} [get]
func (h *Handler) GetCommission(c *gin.Context) {
	partnerID, commissionID, ok := h.parsePartnerCommissionIDs(c)
	if !ok {
		return
	}
	resp, err := h.service.GetCommission(c.Request.Context(), commissionID)
	if err != nil {
		writeCommissionError(c, err)
		return
	}
	if resp.PartnerID != partnerID {
		httpx.Error(c, http.StatusNotFound, "NOT_FOUND", "commission not found for this partner", nil)
		return
	}
	httpx.Success(c, http.StatusOK, resp)
}

// ApproveCommission godoc
// @Summary Approve a pending commission
// @Description Move a PENDING commission to APPROVED, marking it ready for payout (ADMIN/SUPERVISOR only)
// @Tags partner-commissions
// @Accept json
// @Produce json
// @Param partnerID path int64 true "Partner ID"
// @Param commissionID path int64 true "Commission ID"
// @Success 200 {object} PartnerCommissionResponse
// @Failure 400 {object} httpx.ErrorEnvelope
// @Failure 403 {object} httpx.ErrorEnvelope
// @Failure 404 {object} httpx.ErrorEnvelope
// @Router /partners/{partnerID}/commissions/{commissionID}/approve [patch]
func (h *Handler) ApproveCommission(c *gin.Context) {
	partnerID, commissionID, ok := h.parsePartnerCommissionIDs(c)
	if !ok {
		return
	}
	actor, _ := identity.CurrentUser(c)
	resp, err := h.service.ApproveCommission(c.Request.Context(), actor, commissionID)
	if err != nil {
		writeCommissionError(c, err)
		return
	}
	if resp.PartnerID != partnerID {
		httpx.Error(c, http.StatusNotFound, "NOT_FOUND", "commission not found for this partner", nil)
		return
	}
	httpx.Success(c, http.StatusOK, resp)
}

// PayCommission godoc
// @Summary Mark a commission as paid
// @Description Move an APPROVED commission to PAID (ADMIN only)
// @Tags partner-commissions
// @Accept json
// @Produce json
// @Param partnerID path int64 true "Partner ID"
// @Param commissionID path int64 true "Commission ID"
// @Success 200 {object} PartnerCommissionResponse
// @Failure 400 {object} httpx.ErrorEnvelope
// @Failure 403 {object} httpx.ErrorEnvelope
// @Failure 404 {object} httpx.ErrorEnvelope
// @Router /partners/{partnerID}/commissions/{commissionID}/pay [patch]
func (h *Handler) PayCommission(c *gin.Context) {
	partnerID, commissionID, ok := h.parsePartnerCommissionIDs(c)
	if !ok {
		return
	}
	actor, _ := identity.CurrentUser(c)
	resp, err := h.service.PayCommission(c.Request.Context(), actor, commissionID)
	if err != nil {
		writeCommissionError(c, err)
		return
	}
	if resp.PartnerID != partnerID {
		httpx.Error(c, http.StatusNotFound, "NOT_FOUND", "commission not found for this partner", nil)
		return
	}
	httpx.Success(c, http.StatusOK, resp)
}

// CancelCommission godoc
// @Summary Cancel a commission
// @Description Void a PENDING or APPROVED commission, e.g. when the underlying closing is later reversed (ADMIN/SUPERVISOR only)
// @Tags partner-commissions
// @Accept json
// @Produce json
// @Param partnerID path int64 true "Partner ID"
// @Param commissionID path int64 true "Commission ID"
// @Param request body CommissionActionRequest false "Cancellation note"
// @Success 200 {object} PartnerCommissionResponse
// @Failure 400 {object} httpx.ErrorEnvelope
// @Failure 403 {object} httpx.ErrorEnvelope
// @Failure 404 {object} httpx.ErrorEnvelope
// @Router /partners/{partnerID}/commissions/{commissionID}/cancel [patch]
func (h *Handler) CancelCommission(c *gin.Context) {
	partnerID, commissionID, ok := h.parsePartnerCommissionIDs(c)
	if !ok {
		return
	}
	var req CommissionActionRequest
	_ = c.ShouldBindJSON(&req) // note is optional; ignore empty/absent body
	actor, _ := identity.CurrentUser(c)
	resp, err := h.service.CancelCommission(c.Request.Context(), actor, commissionID, req.Note)
	if err != nil {
		writeCommissionError(c, err)
		return
	}
	if resp.PartnerID != partnerID {
		httpx.Error(c, http.StatusNotFound, "NOT_FOUND", "commission not found for this partner", nil)
		return
	}
	httpx.Success(c, http.StatusOK, resp)
}

func (h *Handler) parsePartnerCommissionIDs(c *gin.Context) (partnerID int64, commissionID int64, ok bool) {
	partnerID, err := strconv.ParseInt(c.Param("partnerID"), 10, 64)
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid partner ID", nil)
		return 0, 0, false
	}
	commissionID, err = strconv.ParseInt(c.Param("commissionID"), 10, 64)
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid commission ID", nil)
		return 0, 0, false
	}
	return partnerID, commissionID, true
}

/* ---------- CommissionRule ---------- */

// CreateCommissionRule godoc
// @Summary Create a commission rule
// @Description Add an effective-dated, optionally package-scoped commission rate overlay for a partner type (PERCENTAGE/FIXED/TIER). Falls back to the partner type's flat rate if no rule matches a closing (ADMIN/SUPERVISOR only)
// @Tags partner-commission-rules
// @Accept json
// @Produce json
// @Param id path int64 true "Partner Type ID"
// @Param request body CreateCommissionRuleRequest true "Commission rule data"
// @Success 201 {object} CommissionRuleResponse
// @Failure 400 {object} httpx.ErrorEnvelope
// @Failure 403 {object} httpx.ErrorEnvelope
// @Failure 404 {object} httpx.ErrorEnvelope
// @Router /partner-types/{id}/commission-rules [post]
func (h *Handler) CreateCommissionRule(c *gin.Context) {
	partnerTypeID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid partner type ID", nil)
		return
	}
	var req CreateCommissionRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	actor, _ := identity.CurrentUser(c)
	resp, err := h.service.CreateCommissionRule(c.Request.Context(), actor, partnerTypeID, req)
	if err != nil {
		writeCommissionError(c, err)
		return
	}
	httpx.Success(c, http.StatusCreated, resp)
}

// ListCommissionRules godoc
// @Summary List commission rules for a partner type
// @Description Get commission rules for a partner type, optionally filtered by plan and active status
// @Tags partner-commission-rules
// @Accept json
// @Produce json
// @Param id path int64 true "Partner Type ID"
// @Param plan_id query int64 false "Filter by subscription plan ID"
// @Param active_only query bool false "Only return active rules (default false)"
// @Success 200 {array} CommissionRuleResponse
// @Router /partner-types/{id}/commission-rules [get]
func (h *Handler) ListCommissionRules(c *gin.Context) {
	partnerTypeID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid partner type ID", nil)
		return
	}
	var planID *int64
	if raw := c.Query("plan_id"); raw != "" {
		v, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid plan_id", nil)
			return
		}
		planID = &v
	}
	activeOnly := c.DefaultQuery("active_only", "false") == "true"
	resp, err := h.service.ListCommissionRules(c.Request.Context(), partnerTypeID, planID, activeOnly)
	if err != nil {
		writeCommissionError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, resp)
}

// GetCommissionRule godoc
// @Summary Get commission rule detail
// @Description Retrieve a single commission rule (with its tiers, if TIER mode), scoped to the given partner type
// @Tags partner-commission-rules
// @Accept json
// @Produce json
// @Param id path int64 true "Partner Type ID"
// @Param ruleID path int64 true "Commission Rule ID"
// @Success 200 {object} CommissionRuleResponse
// @Failure 404 {object} httpx.ErrorEnvelope
// @Router /partner-types/{id}/commission-rules/{ruleID} [get]
func (h *Handler) GetCommissionRule(c *gin.Context) {
	partnerTypeID, ruleID, ok := h.parseCommissionRuleIDs(c)
	if !ok {
		return
	}
	resp, err := h.service.GetCommissionRule(c.Request.Context(), ruleID)
	if err != nil {
		writeCommissionError(c, err)
		return
	}
	if resp.PartnerTypeID != partnerTypeID {
		httpx.Error(c, http.StatusNotFound, "NOT_FOUND", "commission rule not found for this partner type", nil)
		return
	}
	httpx.Success(c, http.StatusOK, resp)
}

// DeactivateCommissionRule godoc
// @Summary Deactivate a commission rule
// @Description Retire a commission rule (rules are superseded by creating a new one, never edited in place) (ADMIN/SUPERVISOR only)
// @Tags partner-commission-rules
// @Accept json
// @Produce json
// @Param id path int64 true "Partner Type ID"
// @Param ruleID path int64 true "Commission Rule ID"
// @Success 200 {object} CommissionRuleResponse
// @Failure 403 {object} httpx.ErrorEnvelope
// @Failure 404 {object} httpx.ErrorEnvelope
// @Router /partner-types/{id}/commission-rules/{ruleID}/deactivate [patch]
func (h *Handler) DeactivateCommissionRule(c *gin.Context) {
	partnerTypeID, ruleID, ok := h.parseCommissionRuleIDs(c)
	if !ok {
		return
	}
	actor, _ := identity.CurrentUser(c)
	resp, err := h.service.DeactivateCommissionRule(c.Request.Context(), actor, ruleID)
	if err != nil {
		writeCommissionError(c, err)
		return
	}
	if resp.PartnerTypeID != partnerTypeID {
		httpx.Error(c, http.StatusNotFound, "NOT_FOUND", "commission rule not found for this partner type", nil)
		return
	}
	httpx.Success(c, http.StatusOK, resp)
}

func (h *Handler) parseCommissionRuleIDs(c *gin.Context) (partnerTypeID int64, ruleID int64, ok bool) {
	partnerTypeID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid partner type ID", nil)
		return 0, 0, false
	}
	ruleID, err = strconv.ParseInt(c.Param("ruleID"), 10, 64)
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid commission rule ID", nil)
		return 0, 0, false
	}
	return partnerTypeID, ruleID, true
}

/* ---------- PartnerPayout ---------- */

// CreatePayout godoc
// @Summary Create a payout batching approved commissions
// @Description Batch every APPROVED, not-yet-batched commission for this partner into one new PENDING payout (ADMIN/SUPERVISOR only)
// @Tags partner-payouts
// @Accept json
// @Produce json
// @Param partnerID path int64 true "Partner ID"
// @Success 201 {object} PartnerPayoutResponse
// @Failure 400 {object} httpx.ErrorEnvelope
// @Failure 403 {object} httpx.ErrorEnvelope
// @Failure 404 {object} httpx.ErrorEnvelope
// @Router /partners/{partnerID}/payouts [post]
func (h *Handler) CreatePayout(c *gin.Context) {
	partnerID, err := strconv.ParseInt(c.Param("partnerID"), 10, 64)
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid partner ID", nil)
		return
	}
	actor, _ := identity.CurrentUser(c)
	resp, err := h.service.CreatePayout(c.Request.Context(), actor, partnerID)
	if err != nil {
		writeCommissionError(c, err)
		return
	}
	httpx.Success(c, http.StatusCreated, resp)
}

// ListPayouts godoc
// @Summary List partner payouts
// @Description Get a paginated list of payouts for a partner, optionally filtered by status
// @Tags partner-payouts
// @Accept json
// @Produce json
// @Param partnerID path int64 true "Partner ID"
// @Param status query string false "Filter by status (PENDING, PAID, CANCELLED)"
// @Param page query int false "Page number (default 1)"
// @Param limit query int false "Page size (default 10)"
// @Success 200 {object} PartnerPayoutListResponse
// @Router /partners/{partnerID}/payouts [get]
func (h *Handler) ListPayouts(c *gin.Context) {
	partnerID, err := strconv.ParseInt(c.Param("partnerID"), 10, 64)
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid partner ID", nil)
		return
	}
	status := c.DefaultQuery("status", "")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	resp, err := h.service.ListPayouts(c.Request.Context(), partnerID, status, page, limit)
	if err != nil {
		writeCommissionError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, resp)
}

func (h *Handler) ListAllPayouts(c *gin.Context) {
	partnerID, err := strconv.ParseInt(c.Param("partnerID"), 10, 64)
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid partner ID", nil)
		return
	}
	status := c.DefaultQuery("status", "")
	resp, err := h.service.ListPayouts(c.Request.Context(), partnerID, status, 1, 0)
	if err != nil {
		writeCommissionError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, resp)
}

func pageFromOffsetLimit(offset int, limit int) int {
	if limit <= 0 {
		return 1
	}
	return offset/limit + 1
}

// GetPayout godoc
// @Summary Get partner payout detail
// @Description Retrieve a single payout (with its batched commission items), scoped to the given partner
// @Tags partner-payouts
// @Accept json
// @Produce json
// @Param partnerID path int64 true "Partner ID"
// @Param payoutID path int64 true "Payout ID"
// @Success 200 {object} PartnerPayoutResponse
// @Failure 404 {object} httpx.ErrorEnvelope
// @Router /partners/{partnerID}/payouts/{payoutID} [get]
func (h *Handler) GetPayout(c *gin.Context) {
	partnerID, payoutID, ok := h.parsePartnerPayoutIDs(c)
	if !ok {
		return
	}
	resp, err := h.service.GetPayout(c.Request.Context(), payoutID)
	if err != nil {
		writeCommissionError(c, err)
		return
	}
	if resp.PartnerID != partnerID {
		httpx.Error(c, http.StatusNotFound, "NOT_FOUND", "payout not found for this partner", nil)
		return
	}
	httpx.Success(c, http.StatusOK, resp)
}

// PayPayout godoc
// @Summary Pay a payout
// @Description Move a PENDING payout, and every commission still batched in it, to PAID (ADMIN only)
// @Tags partner-payouts
// @Accept json
// @Produce json
// @Param partnerID path int64 true "Partner ID"
// @Param payoutID path int64 true "Payout ID"
// @Success 200 {object} PartnerPayoutResponse
// @Failure 400 {object} httpx.ErrorEnvelope
// @Failure 403 {object} httpx.ErrorEnvelope
// @Failure 404 {object} httpx.ErrorEnvelope
// @Router /partners/{partnerID}/payouts/{payoutID}/pay [patch]
func (h *Handler) PayPayout(c *gin.Context) {
	partnerID, payoutID, ok := h.parsePartnerPayoutIDs(c)
	if !ok {
		return
	}
	actor, _ := identity.CurrentUser(c)
	resp, err := h.service.PayPayout(c.Request.Context(), actor, payoutID)
	if err != nil {
		writeCommissionError(c, err)
		return
	}
	if resp.PartnerID != partnerID {
		httpx.Error(c, http.StatusNotFound, "NOT_FOUND", "payout not found for this partner", nil)
		return
	}
	httpx.Success(c, http.StatusOK, resp)
}

// CancelPayout godoc
// @Summary Cancel a payout
// @Description Void a PENDING payout, releasing its batched commissions back to APPROVED (ADMIN/SUPERVISOR only)
// @Tags partner-payouts
// @Accept json
// @Produce json
// @Param partnerID path int64 true "Partner ID"
// @Param payoutID path int64 true "Payout ID"
// @Param request body PayoutActionRequest false "Cancellation note"
// @Success 200 {object} PartnerPayoutResponse
// @Failure 400 {object} httpx.ErrorEnvelope
// @Failure 403 {object} httpx.ErrorEnvelope
// @Failure 404 {object} httpx.ErrorEnvelope
// @Router /partners/{partnerID}/payouts/{payoutID}/cancel [patch]
func (h *Handler) CancelPayout(c *gin.Context) {
	partnerID, payoutID, ok := h.parsePartnerPayoutIDs(c)
	if !ok {
		return
	}
	var req PayoutActionRequest
	_ = c.ShouldBindJSON(&req) // note is optional; ignore empty/absent body
	actor, _ := identity.CurrentUser(c)
	resp, err := h.service.CancelPayout(c.Request.Context(), actor, payoutID, req.Note)
	if err != nil {
		writeCommissionError(c, err)
		return
	}
	if resp.PartnerID != partnerID {
		httpx.Error(c, http.StatusNotFound, "NOT_FOUND", "payout not found for this partner", nil)
		return
	}
	httpx.Success(c, http.StatusOK, resp)
}

func (h *Handler) parsePartnerPayoutIDs(c *gin.Context) (partnerID int64, payoutID int64, ok bool) {
	partnerID, err := strconv.ParseInt(c.Param("partnerID"), 10, 64)
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid partner ID", nil)
		return 0, 0, false
	}
	payoutID, err = strconv.ParseInt(c.Param("payoutID"), 10, 64)
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid payout ID", nil)
		return 0, 0, false
	}
	return partnerID, payoutID, true
}

func writeCommissionError(c *gin.Context, err error) {
	switch err {
	case ErrNotFound:
		httpx.Error(c, http.StatusNotFound, "NOT_FOUND", "partner, commission, rule, or payout not found", nil)
	case ErrForbidden:
		httpx.Error(c, http.StatusForbidden, "FORBIDDEN", "not allowed to perform this action", nil)
	case ErrInvalidCommissionRate:
		httpx.Error(c, http.StatusBadRequest, "INVALID_COMMISSION_RATE", err.Error(), nil)
	case ErrInvalidCommissionMode:
		httpx.Error(c, http.StatusBadRequest, "INVALID_COMMISSION_MODE", err.Error(), nil)
	case ErrInvalidCommissionStatus:
		httpx.Error(c, http.StatusBadRequest, "INVALID_STATUS", err.Error(), nil)
	case ErrInvalidMoney:
		httpx.Error(c, http.StatusBadRequest, "INVALID_DECIMAL", err.Error(), nil)
	case ErrCommissionAlreadyExists:
		httpx.Error(c, http.StatusConflict, "COMMISSION_ALREADY_EXISTS", err.Error(), nil)
	case ErrInvalidCommissionTier:
		httpx.Error(c, http.StatusBadRequest, "INVALID_COMMISSION_TIER", err.Error(), nil)
	case ErrNoMatchingTier:
		httpx.Error(c, http.StatusBadRequest, "NO_MATCHING_TIER", err.Error(), nil)
	case ErrCommissionInPayout:
		httpx.Error(c, http.StatusConflict, "COMMISSION_IN_PAYOUT", err.Error(), nil)
	case ErrNoPayableCommissions:
		httpx.Error(c, http.StatusBadRequest, "NO_PAYABLE_COMMISSIONS", err.Error(), nil)
	case ErrMixedCurrency:
		httpx.Error(c, http.StatusBadRequest, "MIXED_CURRENCY", err.Error(), nil)
	case ErrInvalidPayoutStatus:
		httpx.Error(c, http.StatusBadRequest, "INVALID_PAYOUT_STATUS", err.Error(), nil)
	default:
		httpx.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to process commission request", nil)
	}
}

func requestMeta(c *gin.Context) RequestMeta {
	return RequestMeta{
		IPAddress: c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
		RequestID: httpx.RequestID(c),
	}
}
