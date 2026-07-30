package discussion

import (
	"errors"
	"net/http"
	"strconv"

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
	discussions := rg.Group("/discussions")
	{
		discussions.GET("/threads", h.listThreads)
		discussions.POST("/threads", h.createThread)
		discussions.GET("/threads/:id", h.getThread)
		discussions.POST("/threads/:id/like", h.toggleLike)
		discussions.DELETE("/threads/:id", h.deleteThread)
		discussions.POST("/threads/:id/replies", h.addReply)
		discussions.DELETE("/replies/:id", h.deleteReply)
	}
}

func (h *Handler) listThreads(c *gin.Context) {
	currentUser, ok := identity.CurrentUser(c)
	if !ok {
		httpx.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "Sesi tidak valid", nil)
		return
	}

	channel := c.Query("channel")
	query := c.Query("query")

	threads, err := h.service.ListThreads(c.Request.Context(), currentUser, channel, query)
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
		return
	}

	httpx.Success(c, http.StatusOK, threads)
}

func (h *Handler) createThread(c *gin.Context) {
	currentUser, ok := identity.CurrentUser(c)
	if !ok {
		httpx.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "Sesi tidak valid", nil)
		return
	}

	var req CreateThreadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "BAD_REQUEST", "Payload request tidak valid", nil)
		return
	}

	thread, err := h.service.CreateThread(c.Request.Context(), currentUser, req)
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
		return
	}

	httpx.Success(c, http.StatusCreated, thread)
}

func (h *Handler) getThread(c *gin.Context) {
	currentUser, ok := identity.CurrentUser(c)
	if !ok {
		httpx.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "Sesi tidak valid", nil)
		return
	}

	threadID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "BAD_REQUEST", "ID thread tidak valid", nil)
		return
	}

	thread, err := h.service.GetThread(c.Request.Context(), currentUser, threadID)
	if err != nil {
		if errors.Is(err, ErrThreadNotFound) {
			httpx.Error(c, http.StatusNotFound, "NOT_FOUND", err.Error(), nil)
			return
		}
		httpx.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
		return
	}

	httpx.Success(c, http.StatusOK, thread)
}

func (h *Handler) toggleLike(c *gin.Context) {
	currentUser, ok := identity.CurrentUser(c)
	if !ok {
		httpx.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "Sesi tidak valid", nil)
		return
	}

	threadID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "BAD_REQUEST", "ID thread tidak valid", nil)
		return
	}

	isLiked, likesCount, err := h.service.ToggleLike(c.Request.Context(), currentUser, threadID)
	if err != nil {
		if errors.Is(err, ErrThreadNotFound) {
			httpx.Error(c, http.StatusNotFound, "NOT_FOUND", err.Error(), nil)
			return
		}
		httpx.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
		return
	}

	httpx.Success(c, http.StatusOK, gin.H{
		"isLiked": isLiked,
		"likes":   likesCount,
	})
}

func (h *Handler) deleteThread(c *gin.Context) {
	currentUser, ok := identity.CurrentUser(c)
	if !ok {
		httpx.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "Sesi tidak valid", nil)
		return
	}

	threadID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "BAD_REQUEST", "ID thread tidak valid", nil)
		return
	}

	if err := h.service.DeleteThread(c.Request.Context(), currentUser, threadID); err != nil {
		if errors.Is(err, ErrThreadNotFound) {
			httpx.Error(c, http.StatusNotFound, "NOT_FOUND", err.Error(), nil)
			return
		}
		if errors.Is(err, ErrUnauthorizedDelete) {
			httpx.Error(c, http.StatusForbidden, "FORBIDDEN", err.Error(), nil)
			return
		}
		httpx.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
		return
	}

	httpx.Success(c, http.StatusOK, gin.H{"message": "Thread berhasil dihapus"})
}

func (h *Handler) addReply(c *gin.Context) {
	currentUser, ok := identity.CurrentUser(c)
	if !ok {
		httpx.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "Sesi tidak valid", nil)
		return
	}

	threadID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "BAD_REQUEST", "ID thread tidak valid", nil)
		return
	}

	var req CreateReplyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "BAD_REQUEST", "Payload balasan tidak valid", nil)
		return
	}

	reply, err := h.service.AddReply(c.Request.Context(), currentUser, threadID, req)
	if err != nil {
		if errors.Is(err, ErrThreadNotFound) {
			httpx.Error(c, http.StatusNotFound, "NOT_FOUND", err.Error(), nil)
			return
		}
		httpx.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
		return
	}

	httpx.Success(c, http.StatusCreated, reply)
}

func (h *Handler) deleteReply(c *gin.Context) {
	currentUser, ok := identity.CurrentUser(c)
	if !ok {
		httpx.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "Sesi tidak valid", nil)
		return
	}

	replyID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "BAD_REQUEST", "ID balasan tidak valid", nil)
		return
	}

	if err := h.service.DeleteReply(c.Request.Context(), currentUser, replyID); err != nil {
		if errors.Is(err, ErrReplyNotFound) {
			httpx.Error(c, http.StatusNotFound, "NOT_FOUND", err.Error(), nil)
			return
		}
		if errors.Is(err, ErrUnauthorizedDelete) {
			httpx.Error(c, http.StatusForbidden, "FORBIDDEN", err.Error(), nil)
			return
		}
		httpx.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
		return
	}

	httpx.Success(c, http.StatusOK, gin.H{"message": "Balasan berhasil dihapus"})
}
