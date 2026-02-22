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

// ExpenseHandler handles HTTP requests for expenses
type ExpenseHandler struct {
	expenseService *services.ExpenseService
}

// NewExpenseHandler creates a new expense handler
func NewExpenseHandler(expenseService *services.ExpenseService) *ExpenseHandler {
	return &ExpenseHandler{expenseService: expenseService}
}

// RegisterRoutes registers expense routes
func (h *ExpenseHandler) RegisterRoutes(router *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	expenses := router.Group("/expenses")
	expenses.Use(authMiddleware)
	{
		expenses.POST("", h.CreateExpense)
		expenses.GET("", h.ListUserExpenses)
		expenses.GET("/summary", h.GetUserSummary)
		expenses.GET("/group/:group_id", h.ListGroupExpenses)
		expenses.GET("/:id", h.GetExpense)
		expenses.PATCH("/:id", h.UpdateExpense)
		expenses.DELETE("/:id", h.DeleteExpense)
	}
}

// CreateExpense creates a new expense
// @Summary Create a new expense
// @Description Create a new expense in a group with specified split
// @Tags expenses
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.CreateExpenseRequest true "Expense details"
// @Success 201 {object} models.ExpenseResponse
// @Failure 400 {object} map[string]string
// @Router /expenses [post]
func (h *ExpenseHandler) CreateExpense(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req models.CreateExpenseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	expense, err := h.expenseService.CreateExpense(userID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, expense)
}

// GetExpense gets an expense by ID
// @Summary Get expense by ID
// @Description Get detailed information about an expense
// @Tags expenses
// @Produce json
// @Security BearerAuth
// @Param id path string true "Expense ID"
// @Success 200 {object} models.ExpenseResponse
// @Failure 403 {object} map[string]string
// @Router /expenses/{id} [get]
func (h *ExpenseHandler) GetExpense(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	expenseID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid expense ID"})
		return
	}

	expense, err := h.expenseService.GetExpense(expenseID, userID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, expense)
}

// UpdateExpense updates an expense
// @Summary Update expense
// @Description Update expense details (payer only)
// @Tags expenses
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Expense ID"
// @Param request body models.UpdateExpenseRequest true "Update details"
// @Success 200 {object} models.ExpenseResponse
// @Failure 403 {object} map[string]string
// @Router /expenses/{id} [patch]
func (h *ExpenseHandler) UpdateExpense(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	expenseID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid expense ID"})
		return
	}

	var req models.UpdateExpenseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	expense, err := h.expenseService.UpdateExpense(expenseID, userID, &req)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, expense)
}

// DeleteExpense deletes an expense
// @Summary Delete expense
// @Description Delete an expense (payer only)
// @Tags expenses
// @Security BearerAuth
// @Param id path string true "Expense ID"
// @Success 204
// @Failure 403 {object} map[string]string
// @Router /expenses/{id} [delete]
func (h *ExpenseHandler) DeleteExpense(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	expenseID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid expense ID"})
		return
	}

	if err := h.expenseService.DeleteExpense(expenseID, userID); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// ListGroupExpenses lists expenses for a group
// @Summary List group expenses
// @Description Get all expenses in a group
// @Tags expenses
// @Produce json
// @Security BearerAuth
// @Param group_id path string true "Group ID"
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(10)
// @Success 200 {object} models.ExpenseListResponse
// @Router /expenses/group/{group_id} [get]
func (h *ExpenseHandler) ListGroupExpenses(c *gin.Context) {
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

	expenses, err := h.expenseService.ListGroupExpenses(groupID, userID, page, pageSize)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, expenses)
}

// ListUserExpenses lists expenses for the current user
// @Summary List user expenses
// @Description Get all expenses the user is involved in
// @Tags expenses
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(10)
// @Success 200 {object} models.ExpenseListResponse
// @Router /expenses [get]
func (h *ExpenseHandler) ListUserExpenses(c *gin.Context) {
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

	expenses, err := h.expenseService.ListUserExpenses(userID, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, expenses)
}

// GetUserSummary gets expense summary for the current user
// @Summary Get expense summary
// @Description Get summary statistics for the current user's expenses
// @Tags expenses
// @Produce json
// @Security BearerAuth
// @Success 200 {object} models.ExpenseSummary
// @Router /expenses/summary [get]
func (h *ExpenseHandler) GetUserSummary(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	summary, err := h.expenseService.GetUserSummary(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, summary)
}
