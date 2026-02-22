package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// Group represents an expense group (e.g., roommates, trip group)
type Group struct {
	ID          uuid.UUID      `json:"id" gorm:"type:text;primary_key"`
	Name        string         `json:"name" gorm:"not null"`
	Description string         `json:"description"`
	CreatedByID uuid.UUID      `json:"created_by_id" gorm:"not null"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`

	// Relationships
	Creator  User      `json:"creator,omitempty" gorm:"foreignKey:CreatedByID"`
	Members  []User    `json:"members,omitempty" gorm:"many2many:group_members;"`
	Expenses []Expense `json:"expenses,omitempty" gorm:"foreignKey:GroupID"`
}

// GroupMember represents the many-to-many relationship between groups and users
type GroupMember struct {
	GroupID    uuid.UUID `json:"group_id" gorm:"primaryKey"`
	UserID     uuid.UUID `json:"user_id" gorm:"primaryKey"`
	JoinedAt   time.Time `json:"joined_at" gorm:"autoCreateTime"`
	IsActive   bool      `json:"is_active" gorm:"default:true"`
}

// GroupBalance represents the simplified balance within a group
type GroupBalance struct {
	UserID   uuid.UUID       `json:"user_id"`
	UserName string          `json:"user_name"`
	Balance  decimal.Decimal `json:"balance"` // Positive = owed to user, Negative = user owes
}

// SimplifiedDebt represents a single debt transaction after optimization
type SimplifiedDebt struct {
	FromUserID   uuid.UUID       `json:"from_user_id"`
	FromUserName string          `json:"from_user_name"`
	ToUserID     uuid.UUID       `json:"to_user_id"`
	ToUserName   string          `json:"to_user_name"`
	Amount       decimal.Decimal `json:"amount"`
}

// CreateGroupRequest represents group creation input
type CreateGroupRequest struct {
	Name        string      `json:"name" binding:"required,min=2,max=100"`
	Description string      `json:"description"`
	MemberIDs   []uuid.UUID `json:"member_ids"`
}

// UpdateGroupRequest represents group update input
type UpdateGroupRequest struct {
	Name        string `json:"name" binding:"omitempty,min=2,max=100"`
	Description string `json:"description"`
}

// AddMemberRequest represents adding a member to a group
type AddMemberRequest struct {
	UserID uuid.UUID `json:"user_id" binding:"required"`
}

// GroupResponse is the group data returned in API responses
type GroupResponse struct {
	ID          uuid.UUID    `json:"id"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	CreatedByID uuid.UUID    `json:"created_by_id"`
	MemberCount int          `json:"member_count"`
	Members     []UserResponse `json:"members,omitempty"`
	CreatedAt   time.Time    `json:"created_at"`
}

// ToResponse converts Group to GroupResponse
func (g *Group) ToResponse() GroupResponse {
	resp := GroupResponse{
		ID:          g.ID,
		Name:        g.Name,
		Description: g.Description,
		CreatedByID: g.CreatedByID,
		MemberCount: len(g.Members),
		CreatedAt:   g.CreatedAt,
	}
	
	if len(g.Members) > 0 {
		resp.Members = make([]UserResponse, len(g.Members))
		for i, member := range g.Members {
			resp.Members[i] = member.ToResponse()
		}
	}
	
	return resp
}

// GroupDetailResponse includes balances and simplified debts
type GroupDetailResponse struct {
	GroupResponse
	Balances        []GroupBalance   `json:"balances"`
	SimplifiedDebts []SimplifiedDebt `json:"simplified_debts"`
	TotalExpenses   decimal.Decimal  `json:"total_expenses"`
}
