package identity

import (
	"errors"
	"net/http"
	"strconv"

	"backend_crm_piposmart/internal/platform/httpx"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(api *gin.RouterGroup) {
	auth := api.Group("/auth")
	auth.POST("/login", h.login)
	auth.POST("/refresh", h.refresh)

	protected := auth.Group("")
	protected.Use(AuthMiddleware(h.service))
	protected.POST("/logout", h.logout)
	protected.GET("/me", h.me)
	protected.POST("/change-password", h.changePassword)

	sales := api.Group("/sales")
	sales.Use(AuthMiddleware(h.service))
	sales.GET("", h.listSales)
	sales.POST("", h.createSales)
	sales.GET("/:id", h.getSales)
	sales.PATCH("/:id", h.updateSales)
	sales.POST("/:id/activate", h.activateSales)
	sales.POST("/:id/deactivate", h.deactivateSales)
	sales.POST("/:id/reset-password", h.resetSalesPassword)
}

func (h *Handler) login(c *gin.Context) {
	var req LoginRequest
	if !bindJSON(c, &req) {
		return
	}
	response, err := h.service.Login(c.Request.Context(), req, requestMeta(c, nil))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, response)
}

func (h *Handler) refresh(c *gin.Context) {
	var req RefreshRequest
	if !bindJSON(c, &req) {
		return
	}
	response, err := h.service.Refresh(c.Request.Context(), req.RefreshToken, requestMeta(c, nil))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, response)
}

func (h *Handler) logout(c *gin.Context) {
	user, _ := CurrentUser(c)
	var req LogoutRequest
	_ = c.ShouldBindJSON(&req)
	if err := h.service.Logout(c.Request.Context(), req.RefreshToken, user, requestMeta(c, &user)); err != nil {
		writeServiceError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, gin.H{"status": "logged_out"})
}

func (h *Handler) me(c *gin.Context) {
	user, _ := CurrentUser(c)
	httpx.Success(c, http.StatusOK, NewUserResponse(user))
}

func (h *Handler) changePassword(c *gin.Context) {
	user, _ := CurrentUser(c)
	var req ChangePasswordRequest
	if !bindJSON(c, &req) {
		return
	}
	if err := h.service.ChangePassword(c.Request.Context(), user, req, requestMeta(c, &user)); err != nil {
		writeServiceError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, gin.H{"status": "password_changed"})
}

func (h *Handler) listSales(c *gin.Context) {
	user, _ := CurrentUser(c)
	response, err := h.service.ListSales(c.Request.Context(), user, c.Query("status"))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, response)
}

func (h *Handler) createSales(c *gin.Context) {
	user, _ := CurrentUser(c)
	var req CreateSalesRequest
	if !bindJSON(c, &req) {
		return
	}
	response, err := h.service.CreateSales(c.Request.Context(), user, req, requestMeta(c, &user))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	httpx.Success(c, http.StatusCreated, response)
}

func (h *Handler) getSales(c *gin.Context) {
	user, _ := CurrentUser(c)
	id, ok := parseID(c)
	if !ok {
		return
	}
	response, err := h.service.GetSales(c.Request.Context(), user, id)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, response)
}

func (h *Handler) updateSales(c *gin.Context) {
	user, _ := CurrentUser(c)
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req UpdateSalesRequest
	if !bindJSON(c, &req) {
		return
	}
	response, err := h.service.UpdateSales(c.Request.Context(), user, id, req, requestMeta(c, &user))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, response)
}

func (h *Handler) activateSales(c *gin.Context) {
	h.setSalesStatus(c, true)
}

func (h *Handler) deactivateSales(c *gin.Context) {
	h.setSalesStatus(c, false)
}

func (h *Handler) setSalesStatus(c *gin.Context, active bool) {
	user, _ := CurrentUser(c)
	id, ok := parseID(c)
	if !ok {
		return
	}
	response, err := h.service.SetSalesStatus(c.Request.Context(), user, id, active, requestMeta(c, &user))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, response)
}

func (h *Handler) resetSalesPassword(c *gin.Context) {
	user, _ := CurrentUser(c)
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req ResetPasswordRequest
	_ = c.ShouldBindJSON(&req)
	response, err := h.service.ResetSalesPassword(c.Request.Context(), user, id, req, requestMeta(c, &user))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, response)
}

func bindJSON(c *gin.Context, target any) bool {
	if err := c.ShouldBindJSON(target); err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "Payload request tidak valid", gin.H{"error": err.Error()})
		return false
	}
	return true
}

func parseID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id < 1 {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "ID tidak valid", nil)
		return 0, false
	}
	return id, true
}

func requestMeta(c *gin.Context, actor *User) RequestMeta {
	return RequestMeta{
		Actor:     actor,
		IPAddress: c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
		RequestID: httpx.RequestID(c),
	}
}

func writeServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrInvalidCredentials):
		httpx.Error(c, http.StatusUnauthorized, "INVALID_CREDENTIALS", err.Error(), nil)
	case errors.Is(err, ErrInactiveUser):
		httpx.Error(c, http.StatusForbidden, "USER_INACTIVE", err.Error(), nil)
	case errors.Is(err, ErrInvalidToken):
		httpx.Error(c, http.StatusUnauthorized, "INVALID_TOKEN", err.Error(), nil)
	case errors.Is(err, ErrForbidden):
		httpx.Error(c, http.StatusForbidden, "FORBIDDEN", err.Error(), nil)
	case errors.Is(err, ErrNotFound):
		httpx.Error(c, http.StatusNotFound, "NOT_FOUND", err.Error(), nil)
	case errors.Is(err, ErrEmailAlreadyUsed):
		httpx.Error(c, http.StatusConflict, "EMAIL_ALREADY_USED", err.Error(), nil)
	case errors.Is(err, ErrCodeAlreadyUsed):
		httpx.Error(c, http.StatusConflict, "CODE_ALREADY_USED", err.Error(), nil)
	case errors.Is(err, ErrWeakPassword):
		httpx.Error(c, http.StatusBadRequest, "WEAK_PASSWORD", err.Error(), nil)
	default:
		httpx.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Terjadi kesalahan pada server", nil)
	}
}
