package analytics

import (
	"errors"
	"net/http"

	"backend_crm_piposmart/internal/identity"
	"backend_crm_piposmart/internal/platform/httpx"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	analytics := rg.Group("/analytics")
	analytics.GET("/catalog", h.catalog)
	analytics.GET("/catalog/:module", h.catalogByModule)
	analytics.GET("/catalog/:module/:diagram", h.catalogDiagram)
	analytics.POST("/:module/:diagram/query", h.query)
}

func (h *Handler) catalog(c *gin.Context) {
	httpx.Success(c, http.StatusOK, gin.H{"items": h.service.Catalog()})
}

func (h *Handler) catalogByModule(c *gin.Context) {
	httpx.Success(c, http.StatusOK, gin.H{"items": h.service.CatalogByModule(c.Param("module"))})
}

func (h *Handler) catalogDiagram(c *gin.Context) {
	item, ok := h.service.Diagram(c.Param("module"), c.Param("diagram"))
	if !ok {
		httpx.Error(c, http.StatusNotFound, "NOT_FOUND", "diagram tidak ditemukan", nil)
		return
	}
	httpx.Success(c, http.StatusOK, item)
}

func (h *Handler) query(c *gin.Context) {
	user, ok := identity.CurrentUser(c)
	if !ok {
		httpx.Error(c, http.StatusUnauthorized, "UNAUTHENTICATED", "Token akses tidak valid", nil)
		return
	}
	var req QueryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "Payload request tidak valid", gin.H{"error": err.Error()})
		return
	}
	result, err := h.service.Query(c.Request.Context(), user, c.Param("module"), c.Param("diagram"), req)
	if err != nil {
		if errors.Is(err, ErrDiagramNotFound) {
			httpx.Error(c, http.StatusNotFound, "NOT_FOUND", err.Error(), nil)
			return
		}
		httpx.Error(c, http.StatusBadRequest, "ANALYTICS_QUERY_ERROR", err.Error(), nil)
		return
	}
	httpx.Success(c, http.StatusOK, result)
}
