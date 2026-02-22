package repositories

import (
	"expense-tracker-api/internal/models"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// GroupRepository handles database operations for groups
type GroupRepository struct {
	db *gorm.DB
}

// NewGroupRepository creates a new group repository
func NewGroupRepository(db *gorm.DB) *GroupRepository {
	return &GroupRepository{db: db}
}

// Create creates a new group with members
func (r *GroupRepository) Create(group *models.Group, memberIDs []uuid.UUID) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Create the group
		if err := tx.Create(group).Error; err != nil {
			return err
		}

		// Add creator as member
		if err := tx.Create(&models.GroupMember{
			GroupID:  group.ID,
			UserID:   group.CreatedByID,
			IsActive: true,
		}).Error; err != nil {
			return err
		}

		// Add other members
		for _, memberID := range memberIDs {
			if memberID == group.CreatedByID {
				continue // Skip if creator is in the list
			}
			if err := tx.Create(&models.GroupMember{
				GroupID:  group.ID,
				UserID:   memberID,
				IsActive: true,
			}).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

// FindByID finds a group by ID with members
func (r *GroupRepository) FindByID(id uuid.UUID) (*models.Group, error) {
	var group models.Group
	err := r.db.Preload("Members").Preload("Creator").First(&group, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &group, nil
}

// FindByIDWithExpenses finds a group by ID with members and expenses
func (r *GroupRepository) FindByIDWithExpenses(id uuid.UUID) (*models.Group, error) {
	var group models.Group
	err := r.db.Preload("Members").
		Preload("Creator").
		Preload("Expenses", func(db *gorm.DB) *gorm.DB {
			return db.Order("expense_date DESC")
		}).
		Preload("Expenses.PaidBy").
		Preload("Expenses.Shares").
		Preload("Expenses.Shares.User").
		First(&group, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &group, nil
}

// Update updates a group
func (r *GroupRepository) Update(group *models.Group) error {
	return r.db.Save(group).Error
}

// Delete soft deletes a group
func (r *GroupRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&models.Group{}, "id = ?", id).Error
}

// List returns a paginated list of groups for a user
func (r *GroupRepository) List(userID uuid.UUID, page, pageSize int) ([]models.Group, int64, error) {
	var groups []models.Group
	var total int64

	offset := (page - 1) * pageSize

	// Count total groups for user
	if err := r.db.Model(&models.Group{}).
		Joins("JOIN group_members ON group_members.group_id = groups.id").
		Where("group_members.user_id = ?", userID).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get groups with members
	if err := r.db.Preload("Members").Preload("Creator").
		Joins("JOIN group_members ON group_members.group_id = groups.id").
		Where("group_members.user_id = ?", userID).
		Offset(offset).Limit(pageSize).
		Order("groups.created_at DESC").
		Find(&groups).Error; err != nil {
		return nil, 0, err
	}

	return groups, total, nil
}

// AddMember adds a member to a group
func (r *GroupRepository) AddMember(groupID, userID uuid.UUID) error {
	return r.db.Create(&models.GroupMember{
		GroupID:  groupID,
		UserID:   userID,
		IsActive: true,
	}).Error
}

// RemoveMember removes a member from a group
func (r *GroupRepository) RemoveMember(groupID, userID uuid.UUID) error {
	return r.db.Where("group_id = ? AND user_id = ?", groupID, userID).
		Delete(&models.GroupMember{}).Error
}

// IsMember checks if a user is a member of a group
func (r *GroupRepository) IsMember(groupID, userID uuid.UUID) (bool, error) {
	var count int64
	err := r.db.Model(&models.GroupMember{}).
		Where("group_id = ? AND user_id = ? AND is_active = ?", groupID, userID, true).
		Count(&count).Error
	return count > 0, err
}

// GetGroupBalances calculates balances for all members in a group
func (r *GroupRepository) GetGroupBalances(groupID uuid.UUID) ([]models.GroupBalance, error) {
	// Get all members
	var members []models.User
	if err := r.db.Joins("JOIN group_members ON group_members.user_id = users.id").
		Where("group_members.group_id = ? AND group_members.is_active = ?", groupID, true).
		Find(&members).Error; err != nil {
		return nil, err
	}

	// Initialize balances
	balances := make(map[uuid.UUID]decimal.Decimal)
	userNames := make(map[uuid.UUID]string)
	for _, member := range members {
		balances[member.ID] = decimal.Zero
		userNames[member.ID] = member.Name
	}

	// Get all expenses for the group
	var expenses []models.Expense
	if err := r.db.Preload("Shares").Where("group_id = ?", groupID).Find(&expenses).Error; err != nil {
		return nil, err
	}

	// Calculate from expenses
	for _, expense := range expenses {
		// Credit to payer
		if _, exists := balances[expense.PaidByID]; exists {
			balances[expense.PaidByID] = balances[expense.PaidByID].Add(expense.Amount)
		}

		// Debit to share holders
		for _, share := range expense.Shares {
			if _, exists := balances[share.UserID]; exists {
				balances[share.UserID] = balances[share.UserID].Sub(share.Amount)
			}
		}
	}

	// Get all settlements for the group
	var settlements []models.Settlement
	if err := r.db.Where("group_id = ? AND status = ?", groupID, models.SettlementStatusCompleted).Find(&settlements).Error; err != nil {
		return nil, err
	}

	// Apply settlements
	for _, settlement := range settlements {
		if _, exists := balances[settlement.FromUserID]; exists {
			balances[settlement.FromUserID] = balances[settlement.FromUserID].Add(settlement.Amount)
		}
		if _, exists := balances[settlement.ToUserID]; exists {
			balances[settlement.ToUserID] = balances[settlement.ToUserID].Sub(settlement.Amount)
		}
	}

	// Convert to response
	result := make([]models.GroupBalance, 0, len(balances))
	for userID, balance := range balances {
		result = append(result, models.GroupBalance{
			UserID:   userID,
			UserName: userNames[userID],
			Balance:  balance.Round(2),
		})
	}

	return result, nil
}

// GetTotalExpenses calculates the total expenses for a group
func (r *GroupRepository) GetTotalExpenses(groupID uuid.UUID) (decimal.Decimal, error) {
	var total decimal.Decimal
	err := r.db.Model(&models.Expense{}).
		Where("group_id = ?", groupID).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&total).Error
	return total, err
}
