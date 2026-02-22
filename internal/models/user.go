package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// User represents a user in the expense tracking system
type User struct {
	ID        uuid.UUID      `json:"id" gorm:"type:text;primary_key"`
	Name      string         `json:"name" gorm:"not null"`
	Email     string         `json:"email" gorm:"uniqueIndex;not null"`
	Password  string         `json:"-" gorm:"not null"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	// Relationships
	Groups          []Group          `json:"groups,omitempty" gorm:"many2many:group_members;"`
	ExpensesPaid    []Expense        `json:"expenses_paid,omitempty" gorm:"foreignKey:PaidByID"`
	ExpenseShares   []ExpenseShare   `json:"expense_shares,omitempty" gorm:"foreignKey:UserID"`
	SettlementsMade []Settlement     `json:"settlements_made,omitempty" gorm:"foreignKey:FromUserID"`
	SettlementsRecv []Settlement     `json:"settlements_received,omitempty" gorm:"foreignKey:ToUserID"`
}

// UserBalance represents the net balance between two users
type UserBalance struct {
	UserID        uuid.UUID       `json:"user_id"`
	CounterpartID uuid.UUID       `json:"counterpart_id"`
	NetAmount     decimal.Decimal `json:"net_amount"`
	User          User            `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Counterpart   User            `json:"counterpart,omitempty" gorm:"foreignKey:CounterpartID"`
}

// UserResponse is the safe user data to return in API responses
type UserResponse struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

// ToResponse converts User to UserResponse
func (u *User) ToResponse() UserResponse {
	return UserResponse{
		ID:        u.ID,
		Name:      u.Name,
		Email:     u.Email,
		CreatedAt: u.CreatedAt,
	}
}

// RegisterRequest represents user registration input
type RegisterRequest struct {
	Name     string `json:"name" binding:"required,min=2,max=100"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

// LoginRequest represents user login input
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse represents the login response with token
type LoginResponse struct {
	User  UserResponse `json:"user"`
	Token string       `json:"token"`
}
