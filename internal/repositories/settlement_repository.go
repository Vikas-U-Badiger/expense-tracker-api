package repositories

import (
	"expense-tracker-api/internal/models"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// SettlementRepository handles database operations for settlements
type SettlementRepository struct {
	db *gorm.DB
}

// NewSettlementRepository creates a new settlement repository
func NewSettlementRepository(db *gorm.DB) *SettlementRepository {
	return &SettlementRepository{db: db}
}

// Create creates a new settlement
func (r *SettlementRepository) Create(settlement *models.Settlement) error {
	return r.db.Create(settlement).Error
}

// FindByID finds a settlement by ID
func (r *SettlementRepository) FindByID(id uuid.UUID) (*models.Settlement, error) {
	var settlement models.Settlement
	err := r.db.Preload("FromUser").Preload("ToUser").Preload("Group").
		First(&settlement, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &settlement, nil
}

// Update updates a settlement
func (r *SettlementRepository) Update(settlement *models.Settlement) error {
	return r.db.Save(settlement).Error
}

// Delete soft deletes a settlement
func (r *SettlementRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&models.Settlement{}, "id = ?", id).Error
}

// List returns a paginated list of settlements for a user
func (r *SettlementRepository) List(userID uuid.UUID, page, pageSize int) ([]models.Settlement, int64, error) {
	var settlements []models.Settlement
	var total int64

	offset := (page - 1) * pageSize

	if err := r.db.Model(&models.Settlement{}).
		Where("from_user_id = ? OR to_user_id = ?", userID, userID).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := r.db.Preload("FromUser").Preload("ToUser").Preload("Group").
		Where("from_user_id = ? OR to_user_id = ?", userID, userID).
		Offset(offset).Limit(pageSize).
		Order("settled_at DESC, created_at DESC").
		Find(&settlements).Error; err != nil {
		return nil, 0, err
	}

	return settlements, total, nil
}

// ListByGroup returns settlements for a group
func (r *SettlementRepository) ListByGroup(groupID uuid.UUID, page, pageSize int) ([]models.Settlement, int64, error) {
	var settlements []models.Settlement
	var total int64

	offset := (page - 1) * pageSize

	if err := r.db.Model(&models.Settlement{}).
		Where("group_id = ?", groupID).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := r.db.Preload("FromUser").Preload("ToUser").
		Where("group_id = ?", groupID).
		Offset(offset).Limit(pageSize).
		Order("settled_at DESC, created_at DESC").
		Find(&settlements).Error; err != nil {
		return nil, 0, err
	}

	return settlements, total, nil
}

// ListBetweenUsers returns settlements between two users
func (r *SettlementRepository) ListBetweenUsers(user1ID, user2ID uuid.UUID) ([]models.Settlement, error) {
	var settlements []models.Settlement
	err := r.db.Preload("FromUser").Preload("ToUser").Preload("Group").
		Where("((from_user_id = ? AND to_user_id = ?) OR (from_user_id = ? AND to_user_id = ?))",
			user1ID, user2ID, user2ID, user1ID).
		Where("status = ?", models.SettlementStatusCompleted).
		Order("settled_at DESC").
		Find(&settlements).Error
	return settlements, err
}

// GetTotalSettled gets total amount settled between users
func (r *SettlementRepository) GetTotalSettled(fromUserID, toUserID uuid.UUID) (decimal.Decimal, error) {
	var total decimal.Decimal
	err := r.db.Model(&models.Settlement{}).
		Where("from_user_id = ? AND to_user_id = ? AND status = ?",
			fromUserID, toUserID, models.SettlementStatusCompleted).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&total).Error
	return total, err
}

// GetSettlementSummary gets settlement summary for a user
func (r *SettlementRepository) GetSettlementSummary(userID uuid.UUID) (*models.SettlementSummary, error) {
	summary := &models.SettlementSummary{}

	// Total settlements made (user paid others)
	var settlementsMade struct {
		Count  int64
		Amount decimal.Decimal
	}
	if err := r.db.Model(&models.Settlement{}).
		Where("from_user_id = ? AND status = ?", userID, models.SettlementStatusCompleted).
		Select("COUNT(*) as count, COALESCE(SUM(amount), 0) as amount").
		Scan(&settlementsMade).Error; err != nil {
		return nil, err
	}

	// Total settlements received (others paid user)
	var settlementsRecv struct {
		Count  int64
		Amount decimal.Decimal
	}
	if err := r.db.Model(&models.Settlement{}).
		Where("to_user_id = ? AND status = ?", userID, models.SettlementStatusCompleted).
		Select("COUNT(*) as count, COALESCE(SUM(amount), 0) as amount").
		Scan(&settlementsRecv).Error; err != nil {
		return nil, err
	}

	summary.TotalSettlementsMade = int(settlementsMade.Count)
	summary.TotalSettlementsRecv = int(settlementsRecv.Count)
	summary.TotalAmountPaid = settlementsMade.Amount
	summary.TotalAmountReceived = settlementsRecv.Amount

	return summary, nil
}

// GetPendingSettlements gets pending settlements for a user
func (r *SettlementRepository) GetPendingSettlements(userID uuid.UUID) ([]models.Settlement, error) {
	var settlements []models.Settlement
	err := r.db.Preload("FromUser").Preload("ToUser").Preload("Group").
		Where("(from_user_id = ? OR to_user_id = ?) AND status = ?",
			userID, userID, models.SettlementStatusPending).
		Find(&settlements).Error
	return settlements, err
}

// CancelPendingSettlements cancels all pending settlements for a user in a group
func (r *SettlementRepository) CancelPendingSettlements(userID, groupID uuid.UUID) error {
	return r.db.Model(&models.Settlement{}).
		Where("(from_user_id = ? OR to_user_id = ?) AND group_id = ? AND status = ?",
			userID, userID, groupID, models.SettlementStatusPending).
		Update("status", models.SettlementStatusCancelled).Error
}
