package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// ExpenseCategory represents the category of an expense
type ExpenseCategory string

const (
	CategoryFood        ExpenseCategory = "food"
	CategoryTransport   ExpenseCategory = "transport"
	CategoryHousing     ExpenseCategory = "housing"
	CategoryUtilities   ExpenseCategory = "utilities"
	CategoryEntertainment ExpenseCategory = "entertainment"
	CategoryShopping    ExpenseCategory = "shopping"
	CategoryHealth      ExpenseCategory = "health"
	CategoryEducation   ExpenseCategory = "education"
	CategoryTravel      ExpenseCategory = "travel"
	CategoryOther       ExpenseCategory = "other"
)

// Expense represents a single expense transaction
type Expense struct {
	ID          uuid.UUID       `json:"id" gorm:"type:text;primary_key"`
	Description string          `json:"description" gorm:"not null"`
	Amount      decimal.Decimal `json:"amount" gorm:"type:decimal(19,4);not null"`
	Category    ExpenseCategory `json:"category" gorm:"not null;default:'other'"`
	GroupID     uuid.UUID       `json:"group_id" gorm:"not null"`
	PaidByID    uuid.UUID       `json:"paid_by_id" gorm:"not null"`
	ExpenseDate time.Time       `json:"expense_date" gorm:"not null"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
	DeletedAt   gorm.DeletedAt  `json:"-" gorm:"index"`

	// Relationships
	Group   Group          `json:"group,omitempty" gorm:"foreignKey:GroupID"`
	PaidBy  User           `json:"paid_by,omitempty" gorm:"foreignKey:PaidByID"`
	Shares  []ExpenseShare `json:"shares,omitempty" gorm:"foreignKey:ExpenseID;constraint:OnDelete:CASCADE;"`
}

// ExpenseShare represents how an expense is split among users
type ExpenseShare struct {
	ID        uuid.UUID       `json:"id" gorm:"type:text;primary_key"`
	ExpenseID uuid.UUID       `json:"expense_id" gorm:"not null"`
	UserID    uuid.UUID       `json:"user_id" gorm:"not null"`
	Amount    decimal.Decimal `json:"amount" gorm:"type:decimal(19,4);not null"`
	CreatedAt time.Time       `json:"created_at"`

	// Relationships
	User User `json:"user,omitempty" gorm:"foreignKey:UserID"`
}

// SplitType represents how an expense is split among users
type SplitType string

const (
	SplitTypeEqual   SplitType = "equal"
	SplitTypeExact   SplitType = "exact"
	SplitTypePercent SplitType = "percent"
)

// CreateExpenseRequest represents expense creation input
type CreateExpenseRequest struct {
	Description string                `json:"description" binding:"required,min=1,max=255"`
	Amount      string                `json:"amount" binding:"required"`
	Category    ExpenseCategory       `json:"category" binding:"omitempty,oneof=food transport housing utilities entertainment shopping health education travel other"`
	GroupID     uuid.UUID             `json:"group_id" binding:"required"`
	ExpenseDate *time.Time            `json:"expense_date"`
	SplitType   SplitType             `json:"split_type" binding:"required,oneof=equal exact percent"`
	Shares      []ExpenseShareRequest `json:"shares" binding:"required,min=1"`
}

// ExpenseShareRequest represents a share in an expense creation request
type ExpenseShareRequest struct {
	UserID uuid.UUID `json:"user_id" binding:"required"`
	Amount string    `json:"amount" binding:"required"` // For exact: actual amount, for percent: percentage value
}

// UpdateExpenseRequest represents expense update input
type UpdateExpenseRequest struct {
	Description string          `json:"description" binding:"omitempty,min=1,max=255"`
	Amount      string          `json:"amount"`
	Category    ExpenseCategory `json:"category" binding:"omitempty,oneof=food transport housing utilities entertainment shopping health education travel other"`
	ExpenseDate *time.Time      `json:"expense_date"`
}

// ExpenseResponse is the expense data returned in API responses
type ExpenseResponse struct {
	ID          uuid.UUID            `json:"id"`
	Description string               `json:"description"`
	Amount      decimal.Decimal      `json:"amount"`
	Category    ExpenseCategory      `json:"category"`
	GroupID     uuid.UUID            `json:"group_id"`
	PaidByID    uuid.UUID            `json:"paid_by_id"`
	PaidBy      *UserResponse        `json:"paid_by,omitempty"`
	ExpenseDate time.Time            `json:"expense_date"`
	Shares      []ExpenseShareResponse `json:"shares,omitempty"`
	CreatedAt   time.Time            `json:"created_at"`
}

// ExpenseShareResponse is the share data returned in API responses
type ExpenseShareResponse struct {
	ID     uuid.UUID       `json:"id"`
	UserID uuid.UUID       `json:"user_id"`
	User   *UserResponse   `json:"user,omitempty"`
	Amount decimal.Decimal `json:"amount"`
}

// ToResponse converts Expense to ExpenseResponse
func (e *Expense) ToResponse() ExpenseResponse {
	resp := ExpenseResponse{
		ID:          e.ID,
		Description: e.Description,
		Amount:      e.Amount,
		Category:    e.Category,
		GroupID:     e.GroupID,
		PaidByID:    e.PaidByID,
		ExpenseDate: e.ExpenseDate,
		CreatedAt:   e.CreatedAt,
	}

	if e.PaidBy.ID != uuid.Nil {
		userResp := e.PaidBy.ToResponse()
		resp.PaidBy = &userResp
	}

	if len(e.Shares) > 0 {
		resp.Shares = make([]ExpenseShareResponse, len(e.Shares))
		for i, share := range e.Shares {
			resp.Shares[i] = ExpenseShareResponse{
				ID:     share.ID,
				UserID: share.UserID,
				Amount: share.Amount,
			}
			if share.User.ID != uuid.Nil {
				userResp := share.User.ToResponse()
				resp.Shares[i].User = &userResp
			}
		}
	}

	return resp
}

// ExpenseListResponse represents a paginated list of expenses
type ExpenseListResponse struct {
	Expenses []ExpenseResponse `json:"expenses"`
	Total    int64             `json:"total"`
	Page     int               `json:"page"`
	PageSize int               `json:"page_size"`
}

// ExpenseSummary represents summary statistics for expenses
type ExpenseSummary struct {
	TotalExpenses   int             `json:"total_expenses"`
	TotalAmount     decimal.Decimal `json:"total_amount"`
	CategoryTotals  map[string]decimal.Decimal `json:"category_totals"`
	YouOwe          decimal.Decimal `json:"you_owe"`
	YouAreOwed      decimal.Decimal `json:"you_are_owed"`
	NetBalance      decimal.Decimal `json:"net_balance"` // Positive = you're owed, Negative = you owe
}
