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

// GroupHandler handles HTTP requests for groups
type GroupHandler struct {
	groupService *services.GroupService
}

// NewGroupHandler creates a new group handler
func NewGroupHandler(groupService *services.GroupService) *GroupHandler {
	return &GroupHandler{groupService: groupService}
}

// RegisterRoutes registers group routes
func (h *GroupHandler) RegisterRoutes(router *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	groups := router.Group("/groups")
	groups.Use(authMiddleware)
	{
		groups.POST("", h.CreateGroup)
		groups.GET("", h.ListGroups)
		groups.GET("/:id", h.GetGroup)
		groups.PATCH("/:id", h.UpdateGroup)
		groups.DELETE("/:id", h.DeleteGroup)
		groups.POST("/:id/members", h.AddMember)
		groups.DELETE("/:id/members/:user_id", h.RemoveMember)
		groups.GET("/:id/balances", h.GetBalances)
		groups.GET("/:id/simplified-debts", h.GetSimplifiedDebts)
	}
}

// CreateGroup creates a new group
// @Summary Create a new group
// @Description Create a new expense sharing group
// @Tags groups
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.CreateGroupRequest true "Group details"
// @Success 201 {object} models.GroupResponse
// @Failure 400 {object} map[string]string
// @Router /groups [post]
func (h *GroupHandler) CreateGroup(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req models.CreateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	group, err := h.groupService.CreateGroup(userID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, group)
}

// GetGroup gets a group by ID
// @Summary Get group by ID
// @Description Get detailed information about a group including balances
// @Tags groups
// @Produce json
// @Security BearerAuth
// @Param id path string true "Group ID"
// @Success 200 {object} models.GroupDetailResponse
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /groups/{id} [get]
func (h *GroupHandler) GetGroup(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group ID"})
		return
	}

	group, err := h.groupService.GetGroup(groupID, userID)
	if err != nil {
		if err.Error() == "you are not a member of this group" {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, group)
}

// UpdateGroup updates a group
// @Summary Update group
// @Description Update group details (creator only)
// @Tags groups
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Group ID"
// @Param request body models.UpdateGroupRequest true "Update details"
// @Success 200 {object} models.GroupResponse
// @Failure 403 {object} map[string]string
// @Router /groups/{id} [patch]
func (h *GroupHandler) UpdateGroup(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group ID"})
		return
	}

	var req models.UpdateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	group, err := h.groupService.UpdateGroup(groupID, userID, &req)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, group)
}

// DeleteGroup deletes a group
// @Summary Delete group
// @Description Delete a group (creator only)
// @Tags groups
// @Security BearerAuth
// @Param id path string true "Group ID"
// @Success 204
// @Failure 403 {object} map[string]string
// @Router /groups/{id} [delete]
func (h *GroupHandler) DeleteGroup(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group ID"})
		return
	}

	if err := h.groupService.DeleteGroup(groupID, userID); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// ListGroups lists groups for the current user
// @Summary List groups
// @Description Get all groups the current user is a member of
// @Tags groups
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(10)
// @Success 200 {object} map[string]interface{}
// @Router /groups [get]
func (h *GroupHandler) ListGroups(c *gin.Context) {
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

	groups, total, err := h.groupService.ListGroups(userID, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"groups":    groups,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// AddMember adds a member to a group
// @Summary Add member to group
// @Description Add a new member to a group (creator only)
// @Tags groups
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Group ID"
// @Param request body models.AddMemberRequest true "Member to add"
// @Success 204
// @Failure 400 {object} map[string]string
// @Router /groups/{id}/members [post]
func (h *GroupHandler) AddMember(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group ID"})
		return
	}

	var req models.AddMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.groupService.AddMember(groupID, userID, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// RemoveMember removes a member from a group
// @Summary Remove member from group
// @Description Remove a member from a group (creator or self)
// @Tags groups
// @Security BearerAuth
// @Param id path string true "Group ID"
// @Param user_id path string true "User ID to remove"
// @Success 204
// @Failure 400 {object} map[string]string
// @Router /groups/{id}/members/{user_id} [delete]
func (h *GroupHandler) RemoveMember(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group ID"})
		return
	}

	memberID, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	if err := h.groupService.RemoveMember(groupID, userID, memberID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// GetBalances gets balances for a group
// @Summary Get group balances
// @Description Get the balance for each member in a group
// @Tags groups
// @Produce json
// @Security BearerAuth
// @Param id path string true "Group ID"
// @Success 200 {array} models.GroupBalance
// @Failure 403 {object} map[string]string
// @Router /groups/{id}/balances [get]
func (h *GroupHandler) GetBalances(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group ID"})
		return
	}

	balances, err := h.groupService.GetGroupBalances(groupID, userID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, balances)
}

// GetSimplifiedDebts gets simplified debts for a group
// @Summary Get simplified debts
// @Description Get the optimized list of debts to settle the group
// @Tags groups
// @Produce json
// @Security BearerAuth
// @Param id path string true "Group ID"
// @Success 200 {array} models.SimplifiedDebt
// @Failure 403 {object} map[string]string
// @Router /groups/{id}/simplified-debts [get]
func (h *GroupHandler) GetSimplifiedDebts(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group ID"})
		return
	}

	debts, err := h.groupService.GetSimplifiedDebts(groupID, userID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, debts)
}
