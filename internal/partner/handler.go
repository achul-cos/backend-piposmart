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
		partnerTypes.GET("", h.ListPartnerTypes)
		partnerTypes.PUT("/:id", h.UpdatePartnerType)
	}

	partners := rg.Group("/partners")
	{
		partners.POST("", h.CreatePartner)
		// static route before param routes
		partners.GET("/code/:code", h.GetPartnerByCode)
		partners.GET("", h.ListPartners)
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
			partnerGroup.POST("/interactions", h.RecordInteraction)

			partnerGroup.GET("/referrals", h.ListReferrals)
			partnerGroup.POST("/referrals", h.CreateReferral)
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
	resp, err := h.service.CreatePartnerType(c.Request.Context(), req)
	if err != nil {
		switch err {
		case ErrDuplicateType:
			httpx.Error(c, http.StatusConflict, "PARTNER_TYPE_CODE_EXISTS", "partner type code already exists", nil)
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
	resp, err := h.service.UpdatePartnerType(c.Request.Context(), id, req)
	if err != nil {
		if err == ErrNotFound {
			httpx.Error(c, http.StatusNotFound, "NOT_FOUND", "partner type not found", nil)
		} else {
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
	resp, err := h.service.CreatePartner(c.Request.Context(), req)
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
			Page:  offset/limit + 1,
			Limit: limit,
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
			Page:  offset/limit + 1,
			Limit: limit,
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
