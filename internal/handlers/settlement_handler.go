package handlers

import (
	"expense-tracker-api/internal/middleware"
	"expense-tracker-api/internal/models"
	"expense-tracker-api/internal/services"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// SettlementHandler handles HTTP requests for settlements
type SettlementHandler struct {
	settlementService *services.SettlementService
}

// NewSettlementHandler creates a new settlement handler
func NewSettlementHandler(settlementService *services.SettlementService) *SettlementHandler {
	return &SettlementHandler{settlementService: settlementService}
}

// RegisterRoutes registers settlement routes
func (h *SettlementHandler) RegisterRoutes(router *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	settlements := router.Group("/settlements")
	settlements.Use(authMiddleware)
	{
		settlements.POST("", h.CreateSettlement)
		settlements.GET("", h.ListUserSettlements)
		settlements.GET("/summary", h.GetSettlementSummary)
		settlements.GET("/group/:group_id", h.ListGroupSettlements)
		settlements.GET("/balance/:user_id", h.GetBalanceWithUser)
		settlements.GET("/:id", h.GetSettlement)
		settlements.PATCH("/:id", h.UpdateSettlement)
		settlements.DELETE("/:id", h.DeleteSettlement)
	}
}

// CreateSettlement creates a new settlement
// @Summary Create a new settlement
// @Description Record a payment settlement between users
// @Tags settlements
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.CreateSettlementRequest true "Settlement details"
// @Success 201 {object} models.SettlementResponse
// @Failure 400 {object} map[string]string
// @Router /settlements [post]
func (h *SettlementHandler) CreateSettlement(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req models.CreateSettlementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	settlement, err := h.settlementService.CreateSettlement(userID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, settlement)
}

// GetSettlement gets a settlement by ID
// @Summary Get settlement by ID
// @Description Get detailed information about a settlement
// @Tags settlements
// @Produce json
// @Security BearerAuth
// @Param id path string true "Settlement ID"
// @Success 200 {object} models.SettlementResponse
// @Failure 403 {object} map[string]string
// @Router /settlements/{id} [get]
func (h *SettlementHandler) GetSettlement(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	settlementID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid settlement ID"})
		return
	}

	settlement, err := h.settlementService.GetSettlement(settlementID, userID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, settlement)
}

// UpdateSettlement updates a settlement
// @Summary Update settlement
// @Description Update settlement notes (payer only)
// @Tags settlements
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Settlement ID"
// @Param request body map[string]string true "Update details"
// @Success 200 {object} models.SettlementResponse
// @Failure 403 {object} map[string]string
// @Router /settlements/{id} [patch]
func (h *SettlementHandler) UpdateSettlement(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	settlementID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid settlement ID"})
		return
	}

	var req struct {
		Notes string `json:"notes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	settlement, err := h.settlementService.UpdateSettlement(settlementID, userID, req.Notes)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, settlement)
}

// DeleteSettlement cancels a settlement
// @Summary Cancel settlement
// @Description Cancel a settlement (payer only)
// @Tags settlements
// @Security BearerAuth
// @Param id path string true "Settlement ID"
// @Success 204
// @Failure 403 {object} map[string]string
// @Router /settlements/{id} [delete]
func (h *SettlementHandler) DeleteSettlement(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	settlementID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid settlement ID"})
		return
	}

	if err := h.settlementService.DeleteSettlement(settlementID, userID); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// ListUserSettlements lists settlements for the current user
// @Summary List user settlements
// @Description Get all settlements the user is involved in
// @Tags settlements
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(10)
// @Success 200 {object} models.SettlementListResponse
// @Router /settlements [get]
func (h *SettlementHandler) ListUserSettlements(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	settlements, err := h.settlementService.ListUserSettlements(userID, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, settlements)
}

// ListGroupSettlements lists settlements for a group
// @Summary List group settlements
// @Description Get all settlements in a group
// @Tags settlements
// @Produce json
// @Security BearerAuth
// @Param group_id path string true "Group ID"
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(10)
// @Success 200 {object} models.SettlementListResponse
// @Router /settlements/group/{group_id} [get]
func (h *SettlementHandler) ListGroupSettlements(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	groupID, err := uuid.Parse(c.Param("group_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group ID"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	settlements, err := h.settlementService.ListGroupSettlements(groupID, userID, page, pageSize)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, settlements)
}

// GetSettlementSummary gets settlement summary for the current user
// @Summary Get settlement summary
// @Description Get summary statistics for the current user's settlements
// @Tags settlements
// @Produce json
// @Security BearerAuth
// @Success 200 {object} models.SettlementSummary
// @Router /settlements/summary [get]
func (h *SettlementHandler) GetSettlementSummary(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	summary, err := h.settlementService.GetSettlementSummary(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, summary)
}

// GetBalanceWithUser gets the balance between current user and another user
// @Summary Get balance with user
// @Description Get the net balance between current user and another user
// @Tags settlements
// @Produce json
// @Security BearerAuth
// @Param user_id path string true "Other user ID"
// @Success 200 {object} map[string]interface{}
// @Router /settlements/balance/{user_id} [get]
func (h *SettlementHandler) GetBalanceWithUser(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	otherUserID, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	balance, err := h.settlementService.GetBalanceWithUser(userID, otherUserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, balance)
}
