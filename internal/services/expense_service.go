package services

import (
	"expense-tracker-api/internal/models"
	"expense-tracker-api/internal/repositories"
	"expense-tracker-api/pkg/settlement"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ExpenseService handles business logic for expenses
type ExpenseService struct {
	expenseRepo *repositories.ExpenseRepository
	groupRepo   *repositories.GroupRepository
	userRepo    *repositories.UserRepository
	calc        *settlement.Calculator
}

// NewExpenseService creates a new expense service
func NewExpenseService(
	expenseRepo *repositories.ExpenseRepository,
	groupRepo *repositories.GroupRepository,
	userRepo *repositories.UserRepository,
) *ExpenseService {
	return &ExpenseService{
		expenseRepo: expenseRepo,
		groupRepo:   groupRepo,
		userRepo:    userRepo,
		calc:        settlement.NewCalculator(),
	}
}

// CreateExpense creates a new expense
func (s *ExpenseService) CreateExpense(userID uuid.UUID, req *models.CreateExpenseRequest) (*models.ExpenseResponse, error) {
	// Check if user is member of the group
	isMember, err := s.groupRepo.IsMember(req.GroupID, userID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, fmt.Errorf("you are not a member of this group")
	}

	// Parse amount
	amount, err := decimal.NewFromString(req.Amount)
	if err != nil {
		return nil, fmt.Errorf("invalid amount format")
	}

	if amount.LessThanOrEqual(decimal.Zero) {
		return nil, fmt.Errorf("amount must be greater than zero")
	}

	// Get group members
	members, err := s.userRepo.FindGroupMembers(req.GroupID)
	if err != nil {
		return nil, err
	}

	memberMap := make(map[uuid.UUID]bool)
	for _, m := range members {
		memberMap[m.ID] = true
	}

	// Validate shares
	if len(req.Shares) == 0 {
		return nil, fmt.Errorf("at least one share is required")
	}

	// Calculate shares based on split type
	shares := make([]models.ExpenseShare, 0, len(req.Shares))
	shareAmounts := make([]decimal.Decimal, 0, len(req.Shares))

	switch req.SplitType {
	case models.SplitTypeEqual:
		// Equal split among specified users
		equalAmounts := s.calc.CalculateEqualShares(amount, len(req.Shares))
		for i, shareReq := range req.Shares {
			if !memberMap[shareReq.UserID] {
				return nil, fmt.Errorf("user %s is not a member of this group", shareReq.UserID)
			}
			shares = append(shares, models.ExpenseShare{
				UserID: shareReq.UserID,
				Amount: equalAmounts[i],
			})
			shareAmounts = append(shareAmounts, equalAmounts[i])
		}

	case models.SplitTypeExact:
		// Exact amounts specified
		for _, shareReq := range req.Shares {
			if !memberMap[shareReq.UserID] {
				return nil, fmt.Errorf("user %s is not a member of this group", shareReq.UserID)
			}
			shareAmount, err := decimal.NewFromString(shareReq.Amount)
			if err != nil {
				return nil, fmt.Errorf("invalid share amount for user %s", shareReq.UserID)
			}
			shares = append(shares, models.ExpenseShare{
				UserID: shareReq.UserID,
				Amount: shareAmount,
			})
			shareAmounts = append(shareAmounts, shareAmount)
		}

		// Validate sum
		valid, diff := s.calc.ValidateSplit(amount, shareAmounts, models.SplitTypeExact)
		if !valid {
			return nil, fmt.Errorf("shares do not sum to total amount (difference: %s)", diff.String())
		}

	case models.SplitTypePercent:
		// Percentage split
		percentages := make([]decimal.Decimal, 0, len(req.Shares))
		for _, shareReq := range req.Shares {
			if !memberMap[shareReq.UserID] {
				return nil, fmt.Errorf("user %s is not a member of this group", shareReq.UserID)
			}
			percent, err := decimal.NewFromString(shareReq.Amount)
			if err != nil {
				return nil, fmt.Errorf("invalid percentage for user %s", shareReq.UserID)
			}
			percentages = append(percentages, percent)
		}

		// Validate percentages sum to 100
		valid, diff := s.calc.ValidateSplit(decimal.NewFromInt(100), percentages, models.SplitTypePercent)
		if !valid {
			return nil, fmt.Errorf("percentages do not sum to 100 (difference: %s)", diff.String())
		}

		// Calculate amounts
		shareAmounts = s.calc.CalculatePercentShares(amount, percentages)
		for i, shareReq := range req.Shares {
			shares = append(shares, models.ExpenseShare{
				UserID: shareReq.UserID,
				Amount: shareAmounts[i],
			})
		}

	default:
		return nil, fmt.Errorf("invalid split type")
	}

	// Set expense date
	expenseDate := time.Now()
	if req.ExpenseDate != nil {
		expenseDate = *req.ExpenseDate
	}

	// Set category
	category := req.Category
	if category == "" {
		category = models.CategoryOther
	}

	// Create expense
	expense := &models.Expense{
		Description: req.Description,
		Amount:      amount,
		Category:    category,
		GroupID:     req.GroupID,
		PaidByID:    userID,
		ExpenseDate: expenseDate,
	}

	if err := s.expenseRepo.Create(expense, shares); err != nil {
		return nil, fmt.Errorf("failed to create expense: %w", err)
	}

	// Fetch the created expense with relations
	createdExpense, err := s.expenseRepo.FindByID(expense.ID)
	if err != nil {
		return nil, err
	}

	resp := createdExpense.ToResponse()
	return &resp, nil
}

// GetExpense gets an expense by ID
func (s *ExpenseService) GetExpense(expenseID, userID uuid.UUID) (*models.ExpenseResponse, error) {
	expense, err := s.expenseRepo.FindByID(expenseID)
	if err != nil {
		return nil, fmt.Errorf("expense not found")
	}

	// Check if user is member of the group
	isMember, err := s.groupRepo.IsMember(expense.GroupID, userID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, fmt.Errorf("you do not have access to this expense")
	}

	resp := expense.ToResponse()
	return &resp, nil
}

// UpdateExpense updates an expense
func (s *ExpenseService) UpdateExpense(expenseID, userID uuid.UUID, req *models.UpdateExpenseRequest) (*models.ExpenseResponse, error) {
	expense, err := s.expenseRepo.FindByID(expenseID)
	if err != nil {
		return nil, fmt.Errorf("expense not found")
	}

	// Only the person who paid can update the expense
	if expense.PaidByID != userID {
		return nil, fmt.Errorf("only the person who paid can update this expense")
	}

	// Update fields
	if req.Description != "" {
		expense.Description = req.Description
	}

	if req.Amount != "" {
		newAmount, err := decimal.NewFromString(req.Amount)
		if err != nil {
			return nil, fmt.Errorf("invalid amount format")
		}
		if newAmount.LessThanOrEqual(decimal.Zero) {
			return nil, fmt.Errorf("amount must be greater than zero")
		}
		expense.Amount = newAmount
	}

	if req.Category != "" {
		expense.Category = req.Category
	}

	if req.ExpenseDate != nil {
		expense.ExpenseDate = *req.ExpenseDate
	}

	if err := s.expenseRepo.Update(expense); err != nil {
		return nil, fmt.Errorf("failed to update expense: %w", err)
	}

	// Fetch updated expense
	updatedExpense, err := s.expenseRepo.FindByID(expenseID)
	if err != nil {
		return nil, err
	}

	resp := updatedExpense.ToResponse()
	return &resp, nil
}

// DeleteExpense deletes an expense
func (s *ExpenseService) DeleteExpense(expenseID, userID uuid.UUID) error {
	expense, err := s.expenseRepo.FindByID(expenseID)
	if err != nil {
		return fmt.Errorf("expense not found")
	}

	// Only the person who paid can delete the expense
	if expense.PaidByID != userID {
		return fmt.Errorf("only the person who paid can delete this expense")
	}

	return s.expenseRepo.Delete(expenseID)
}

// ListGroupExpenses lists expenses for a group
func (s *ExpenseService) ListGroupExpenses(groupID, userID uuid.UUID, page, pageSize int) (*models.ExpenseListResponse, error) {
	// Check if user is member
	isMember, err := s.groupRepo.IsMember(groupID, userID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, fmt.Errorf("you are not a member of this group")
	}

	expenses, total, err := s.expenseRepo.List(groupID, page, pageSize)
	if err != nil {
		return nil, err
	}

	responses := make([]models.ExpenseResponse, len(expenses))
	for i, expense := range expenses {
		responses[i] = expense.ToResponse()
	}

	return &models.ExpenseListResponse{
		Expenses: responses,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// ListUserExpenses lists expenses for a user across all groups
func (s *ExpenseService) ListUserExpenses(userID uuid.UUID, page, pageSize int) (*models.ExpenseListResponse, error) {
	expenses, total, err := s.expenseRepo.ListByUser(userID, page, pageSize)
	if err != nil {
		return nil, err
	}

	responses := make([]models.ExpenseResponse, len(expenses))
	for i, expense := range expenses {
		responses[i] = expense.ToResponse()
	}

	return &models.ExpenseListResponse{
		Expenses: responses,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// GetUserSummary gets expense summary for a user
func (s *ExpenseService) GetUserSummary(userID uuid.UUID) (*models.ExpenseSummary, error) {
	return s.expenseRepo.GetSummary(userID)
}
