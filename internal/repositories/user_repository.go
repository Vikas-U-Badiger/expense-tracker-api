package repositories

import (
	"expense-tracker-api/internal/models"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// UserRepository handles database operations for users
type UserRepository struct {
	db *gorm.DB
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

// Create creates a new user
func (r *UserRepository) Create(user *models.User) error {
	return r.db.Create(user).Error
}

// FindByID finds a user by ID
func (r *UserRepository) FindByID(id uuid.UUID) (*models.User, error) {
	var user models.User
	err := r.db.First(&user, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// FindByEmail finds a user by email
func (r *UserRepository) FindByEmail(email string) (*models.User, error) {
	var user models.User
	err := r.db.First(&user, "email = ?", email).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// FindByEmailWithPassword finds a user by email including password
func (r *UserRepository) FindByEmailWithPassword(email string) (*models.User, error) {
	var user models.User
	err := r.db.First(&user, "email = ?", email).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// Update updates a user
func (r *UserRepository) Update(user *models.User) error {
	return r.db.Save(user).Error
}

// Delete soft deletes a user
func (r *UserRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&models.User{}, "id = ?", id).Error
}

// List returns a paginated list of users
func (r *UserRepository) List(page, pageSize int) ([]models.User, int64, error) {
	var users []models.User
	var total int64

	offset := (page - 1) * pageSize

	if err := r.db.Model(&models.User{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := r.db.Offset(offset).Limit(pageSize).Find(&users).Error; err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

// Exists checks if a user exists by email
func (r *UserRepository) Exists(email string) (bool, error) {
	var count int64
	err := r.db.Model(&models.User{}).Where("email = ?", email).Count(&count).Error
	return count > 0, err
}

// FindGroupMembers finds all members of a group
func (r *UserRepository) FindGroupMembers(groupID uuid.UUID) ([]models.User, error) {
	var users []models.User
	err := r.db.Joins("JOIN group_members ON group_members.user_id = users.id").
		Where("group_members.group_id = ?", groupID).
		Find(&users).Error
	return users, err
}

// GetUserBalances calculates the net balance for a user across all groups
func (r *UserRepository) GetUserBalances(userID uuid.UUID) (map[string]interface{}, error) {
	// This is a complex query that calculates:
	// 1. Total amount paid by user
	// 2. Total amount owed by user (shares)
	// 3. Net balance

	var result struct {
		TotalPaid decimal.Decimal `gorm:"column:total_paid"`
		TotalOwed decimal.Decimal `gorm:"column:total_owed"`
	}

	// Get total paid
	err := r.db.Raw(`
		SELECT COALESCE(SUM(amount), 0) as total_paid
		FROM expenses
		WHERE paid_by_id = ? AND deleted_at IS NULL
	`, userID).Scan(&result.TotalPaid).Error
	if err != nil {
		return nil, err
	}

	// Get total owed
	err = r.db.Raw(`
		SELECT COALESCE(SUM(amount), 0) as total_owed
		FROM expense_shares
		WHERE user_id = ?
	`, userID).Scan(&result.TotalOwed).Error
	if err != nil {
		return nil, err
	}

	netBalance := result.TotalPaid.Sub(result.TotalOwed)

	return map[string]interface{}{
		"total_paid":  result.TotalPaid,
		"total_owed":  result.TotalOwed,
		"net_balance": netBalance,
	}, nil
}
