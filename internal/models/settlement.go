package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// SettlementStatus represents the status of a settlement
type SettlementStatus string

const (
	SettlementStatusPending   SettlementStatus = "pending"
	SettlementStatusCompleted SettlementStatus = "completed"
	SettlementStatusCancelled SettlementStatus = "cancelled"
)

// Settlement represents a recorded settlement between two users
type Settlement struct {
	ID          uuid.UUID        `json:"id" gorm:"type:text;primary_key"`
	FromUserID  uuid.UUID        `json:"from_user_id" gorm:"not null"`
	ToUserID    uuid.UUID        `json:"to_user_id" gorm:"not null"`
	GroupID     *uuid.UUID       `json:"group_id"` // Optional: if settling within a specific group
	Amount      decimal.Decimal  `json:"amount" gorm:"type:decimal(19,4);not null"`
	Status      SettlementStatus `json:"status" gorm:"not null;default:'completed'"`
	Notes       string           `json:"notes"`
	SettledAt   time.Time        `json:"settled_at" gorm:"not null"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
	DeletedAt   gorm.DeletedAt   `json:"-" gorm:"index"`

	// Relationships
	FromUser User   `json:"from_user,omitempty" gorm:"foreignKey:FromUserID"`
	ToUser   User   `json:"to_user,omitempty" gorm:"foreignKey:ToUserID"`
	Group    *Group `json:"group,omitempty" gorm:"foreignKey:GroupID"`
}

// CreateSettlementRequest represents settlement creation input
type CreateSettlementRequest struct {
	ToUserID  uuid.UUID `json:"to_user_id" binding:"required"`
	GroupID   *uuid.UUID `json:"group_id"`
	Amount    string    `json:"amount" binding:"required"`
	Notes     string    `json:"notes" binding:"max=500"`
	SettledAt *time.Time `json:"settled_at"`
}

// SettlementResponse is the settlement data returned in API responses
type SettlementResponse struct {
	ID           uuid.UUID        `json:"id"`
	FromUserID   uuid.UUID        `json:"from_user_id"`
	FromUser     *UserResponse    `json:"from_user,omitempty"`
	ToUserID     uuid.UUID        `json:"to_user_id"`
	ToUser       *UserResponse    `json:"to_user,omitempty"`
	GroupID      *uuid.UUID       `json:"group_id,omitempty"`
	GroupName    string           `json:"group_name,omitempty"`
	Amount       decimal.Decimal  `json:"amount"`
	Status       SettlementStatus `json:"status"`
	Notes        string           `json:"notes"`
	SettledAt    time.Time        `json:"settled_at"`
	CreatedAt    time.Time        `json:"created_at"`
}

// ToResponse converts Settlement to SettlementResponse
func (s *Settlement) ToResponse() SettlementResponse {
	resp := SettlementResponse{
		ID:         s.ID,
		FromUserID: s.FromUserID,
		ToUserID:   s.ToUserID,
		GroupID:    s.GroupID,
		Amount:     s.Amount,
		Status:     s.Status,
		Notes:      s.Notes,
		SettledAt:  s.SettledAt,
		CreatedAt:  s.CreatedAt,
	}

	if s.FromUser.ID != uuid.Nil {
		userResp := s.FromUser.ToResponse()
		resp.FromUser = &userResp
	}

	if s.ToUser.ID != uuid.Nil {
		userResp := s.ToUser.ToResponse()
		resp.ToUser = &userResp
	}

	if s.Group != nil {
		resp.GroupName = s.Group.Name
	}

	return resp
}

// SettlementListResponse represents a paginated list of settlements
type SettlementListResponse struct {
	Settlements []SettlementResponse `json:"settlements"`
	Total       int64                `json:"total"`
	Page        int                  `json:"page"`
	PageSize    int                  `json:"page_size"`
}

// SettlementSummary represents summary of settlements for a user
type SettlementSummary struct {
	TotalSettlementsMade int             `json:"total_settlements_made"`
	TotalSettlementsRecv int             `json:"total_settlements_received"`
	TotalAmountPaid      decimal.Decimal `json:"total_amount_paid"`
	TotalAmountReceived  decimal.Decimal `json:"total_amount_received"`
}
