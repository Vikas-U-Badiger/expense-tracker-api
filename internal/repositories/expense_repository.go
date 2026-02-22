package repositories

import (
	"expense-tracker-api/internal/models"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// ExpenseRepository handles database operations for expenses
type ExpenseRepository struct {
	db *gorm.DB
}

// NewExpenseRepository creates a new expense repository
func NewExpenseRepository(db *gorm.DB) *ExpenseRepository {
	return &ExpenseRepository{db: db}
}

// Create creates a new expense with shares
func (r *ExpenseRepository) Create(expense *models.Expense, shares []models.ExpenseShare) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Create the expense
		if err := tx.Create(expense).Error; err != nil {
			return err
		}

		// Create shares
		for i := range shares {
			shares[i].ExpenseID = expense.ID
			if err := tx.Create(&shares[i]).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

// FindByID finds an expense by ID with shares
func (r *ExpenseRepository) FindByID(id uuid.UUID) (*models.Expense, error) {
	var expense models.Expense
	err := r.db.Preload("PaidBy").
		Preload("Shares").
		Preload("Shares.User").
		Preload("Group").
		First(&expense, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &expense, nil
}

// Update updates an expense
func (r *ExpenseRepository) Update(expense *models.Expense) error {
	return r.db.Save(expense).Error
}

// UpdateWithShares updates an expense and its shares
func (r *ExpenseRepository) UpdateWithShares(expense *models.Expense, shares []models.ExpenseShare) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Update the expense
		if err := tx.Save(expense).Error; err != nil {
			return err
		}

		// Delete old shares
		if err := tx.Where("expense_id = ?", expense.ID).Delete(&models.ExpenseShare{}).Error; err != nil {
			return err
		}

		// Create new shares
		for i := range shares {
			shares[i].ExpenseID = expense.ID
			if err := tx.Create(&shares[i]).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

// Delete soft deletes an expense and its shares
func (r *ExpenseRepository) Delete(id uuid.UUID) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Delete shares first
		if err := tx.Where("expense_id = ?", id).Delete(&models.ExpenseShare{}).Error; err != nil {
			return err
		}

		// Delete expense
		if err := tx.Delete(&models.Expense{}, "id = ?", id).Error; err != nil {
			return err
		}

		return nil
	})
}

// List returns a paginated list of expenses for a group
func (r *ExpenseRepository) List(groupID uuid.UUID, page, pageSize int) ([]models.Expense, int64, error) {
	var expenses []models.Expense
	var total int64

	offset := (page - 1) * pageSize

	if err := r.db.Model(&models.Expense{}).Where("group_id = ?", groupID).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := r.db.Preload("PaidBy").
		Preload("Shares").
		Preload("Shares.User").
		Where("group_id = ?", groupID).
		Offset(offset).Limit(pageSize).
		Order("expense_date DESC, created_at DESC").
		Find(&expenses).Error; err != nil {
		return nil, 0, err
	}

	return expenses, total, nil
}

// ListByUser returns expenses where user is involved (either paid or has share)
func (r *ExpenseRepository) ListByUser(userID uuid.UUID, page, pageSize int) ([]models.Expense, int64, error) {
	var expenses []models.Expense
	var total int64

	offset := (page - 1) * pageSize

	// Count total
	if err := r.db.Model(&models.Expense{}).
		Joins("LEFT JOIN expense_shares ON expense_shares.expense_id = expenses.id").
		Where("expenses.paid_by_id = ? OR expense_shares.user_id = ?", userID, userID).
		Group("expenses.id").
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get expenses
	if err := r.db.Preload("PaidBy").
		Preload("Shares").
		Preload("Shares.User").
		Preload("Group").
		Joins("LEFT JOIN expense_shares ON expense_shares.expense_id = expenses.id").
		Where("expenses.paid_by_id = ? OR expense_shares.user_id = ?", userID, userID).
		Group("expenses.id").
		Offset(offset).Limit(pageSize).
		Order("expense_date DESC, created_at DESC").
		Find(&expenses).Error; err != nil {
		return nil, 0, err
	}

	return expenses, total, nil
}

// GetSummary gets expense summary for a user
func (r *ExpenseRepository) GetSummary(userID uuid.UUID) (*models.ExpenseSummary, error) {
	summary := &models.ExpenseSummary{
		CategoryTotals: make(map[string]decimal.Decimal),
	}

	// Total expenses count and amount paid by user
	var totalPaid struct {
		Count  int64
		Amount decimal.Decimal
	}
	if err := r.db.Model(&models.Expense{}).
		Where("paid_by_id = ?", userID).
		Select("COUNT(*) as count, COALESCE(SUM(amount), 0) as amount").
		Scan(&totalPaid).Error; err != nil {
		return nil, err
	}

	// Total amount owed by user (shares)
	var totalOwed decimal.Decimal
	if err := r.db.Model(&models.ExpenseShare{}).
		Where("user_id = ?", userID).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&totalOwed).Error; err != nil {
		return nil, err
	}

	// Category totals
	var categoryTotals []struct {
		Category string
		Amount   decimal.Decimal
	}
	if err := r.db.Model(&models.Expense{}).
		Where("paid_by_id = ?", userID).
		Select("category, COALESCE(SUM(amount), 0) as amount").
		Group("category").
		Scan(&categoryTotals).Error; err != nil {
		return nil, err
	}

	// Calculate what user owes others (excluding what they've paid)
	var youOwe decimal.Decimal
	if err := r.db.Raw(`
		SELECT COALESCE(SUM(es.amount), 0)
		FROM expense_shares es
		JOIN expenses e ON e.id = es.expense_id
		WHERE es.user_id = ? AND e.paid_by_id != ?
	`, userID, userID).Scan(&youOwe).Error; err != nil {
		return nil, err
	}

	// Calculate what others owe user
	var youAreOwed decimal.Decimal
	if err := r.db.Raw(`
		SELECT COALESCE(SUM(es.amount), 0)
		FROM expense_shares es
		JOIN expenses e ON e.id = es.expense_id
		WHERE e.paid_by_id = ? AND es.user_id != ?
	`, userID, userID).Scan(&youAreOwed).Error; err != nil {
		return nil, err
	}

	summary.TotalExpenses = int(totalPaid.Count)
	summary.TotalAmount = totalPaid.Amount
	summary.YouOwe = youOwe
	summary.YouAreOwed = youAreOwed
	summary.NetBalance = youAreOwed.Sub(youOwe)

	for _, ct := range categoryTotals {
		summary.CategoryTotals[ct.Category] = ct.Amount
	}

	return summary, nil
}

// GetExpensesByDateRange gets expenses within a date range for a group
func (r *ExpenseRepository) GetExpensesByDateRange(groupID uuid.UUID, startDate, endDate time.Time) ([]models.Expense, error) {
	var expenses []models.Expense
	err := r.db.Preload("Shares").
		Where("group_id = ? AND expense_date >= ? AND expense_date <= ?", groupID, startDate, endDate).
		Order("expense_date DESC").
		Find(&expenses).Error
	return expenses, err
}

// GetTotalByGroup gets total expenses for a group
func (r *ExpenseRepository) GetTotalByGroup(groupID uuid.UUID) (decimal.Decimal, error) {
	var total decimal.Decimal
	err := r.db.Model(&models.Expense{}).
		Where("group_id = ?", groupID).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&total).Error
	return total, err
}
