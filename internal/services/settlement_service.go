package services

import (
	"expense-tracker-api/internal/models"
	"expense-tracker-api/internal/repositories"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// SettlementService handles business logic for settlements
type SettlementService struct {
	settlementRepo *repositories.SettlementRepository
	groupRepo      *repositories.GroupRepository
}

// NewSettlementService creates a new settlement service
func NewSettlementService(
	settlementRepo *repositories.SettlementRepository,
	groupRepo *repositories.GroupRepository,
) *SettlementService {
	return &SettlementService{
		settlementRepo: settlementRepo,
		groupRepo:      groupRepo,
	}
}

// CreateSettlement creates a new settlement
func (s *SettlementService) CreateSettlement(fromUserID uuid.UUID, req *models.CreateSettlementRequest) (*models.SettlementResponse, error) {
	// Parse amount
	amount, err := decimal.NewFromString(req.Amount)
	if err != nil {
		return nil, fmt.Errorf("invalid amount format")
	}

	if amount.LessThanOrEqual(decimal.Zero) {
		return nil, fmt.Errorf("amount must be greater than zero")
	}

	// Validate group if provided
	if req.GroupID != nil {
		isMember, err := s.groupRepo.IsMember(*req.GroupID, fromUserID)
		if err != nil {
			return nil, err
		}
		if !isMember {
			return nil, fmt.Errorf("you are not a member of this group")
		}

		// Check if recipient is also a member
		isRecipientMember, err := s.groupRepo.IsMember(*req.GroupID, req.ToUserID)
		if err != nil {
			return nil, err
		}
		if !isRecipientMember {
			return nil, fmt.Errorf("recipient is not a member of this group")
		}
	}

	// Cannot settle with yourself
	if fromUserID == req.ToUserID {
		return nil, fmt.Errorf("cannot create a settlement with yourself")
	}

	// Set settled at time
	settledAt := time.Now()
	if req.SettledAt != nil {
		settledAt = *req.SettledAt
	}

	// Create settlement
	settlement := &models.Settlement{
		FromUserID: fromUserID,
		ToUserID:   req.ToUserID,
		GroupID:    req.GroupID,
		Amount:     amount,
		Status:     models.SettlementStatusCompleted,
		Notes:      req.Notes,
		SettledAt:  settledAt,
	}

	if err := s.settlementRepo.Create(settlement); err != nil {
		return nil, fmt.Errorf("failed to create settlement: %w", err)
	}

	// Fetch the created settlement with relations
	createdSettlement, err := s.settlementRepo.FindByID(settlement.ID)
	if err != nil {
		return nil, err
	}

	resp := createdSettlement.ToResponse()
	return &resp, nil
}

// GetSettlement gets a settlement by ID
func (s *SettlementService) GetSettlement(settlementID, userID uuid.UUID) (*models.SettlementResponse, error) {
	settlement, err := s.settlementRepo.FindByID(settlementID)
	if err != nil {
		return nil, fmt.Errorf("settlement not found")
	}

	// Check if user is involved in the settlement
	if settlement.FromUserID != userID && settlement.ToUserID != userID {
		return nil, fmt.Errorf("you do not have access to this settlement")
	}

	resp := settlement.ToResponse()
	return &resp, nil
}

// UpdateSettlement updates a settlement
func (s *SettlementService) UpdateSettlement(settlementID, userID uuid.UUID, notes string) (*models.SettlementResponse, error) {
	settlement, err := s.settlementRepo.FindByID(settlementID)
	if err != nil {
		return nil, fmt.Errorf("settlement not found")
	}

	// Only the payer can update the settlement
	if settlement.FromUserID != userID {
		return nil, fmt.Errorf("only the payer can update this settlement")
	}

	// Can only update pending or completed settlements
	if settlement.Status == models.SettlementStatusCancelled {
		return nil, fmt.Errorf("cannot update a cancelled settlement")
	}

	settlement.Notes = notes

	if err := s.settlementRepo.Update(settlement); err != nil {
		return nil, fmt.Errorf("failed to update settlement: %w", err)
	}

	resp := settlement.ToResponse()
	return &resp, nil
}

// DeleteSettlement cancels a settlement
func (s *SettlementService) DeleteSettlement(settlementID, userID uuid.UUID) error {
	settlement, err := s.settlementRepo.FindByID(settlementID)
	if err != nil {
		return fmt.Errorf("settlement not found")
	}

	// Only the payer can cancel the settlement
	if settlement.FromUserID != userID {
		return fmt.Errorf("only the payer can cancel this settlement")
	}

	// Soft delete by marking as cancelled
	settlement.Status = models.SettlementStatusCancelled
	return s.settlementRepo.Update(settlement)
}

// ListUserSettlements lists settlements for a user
func (s *SettlementService) ListUserSettlements(userID uuid.UUID, page, pageSize int) (*models.SettlementListResponse, error) {
	settlements, total, err := s.settlementRepo.List(userID, page, pageSize)
	if err != nil {
		return nil, err
	}

	responses := make([]models.SettlementResponse, len(settlements))
	for i, settlement := range settlements {
		responses[i] = settlement.ToResponse()
	}

	return &models.SettlementListResponse{
		Settlements: responses,
		Total:       total,
		Page:        page,
		PageSize:    pageSize,
	}, nil
}

// ListGroupSettlements lists settlements for a group
func (s *SettlementService) ListGroupSettlements(groupID, userID uuid.UUID, page, pageSize int) (*models.SettlementListResponse, error) {
	// Check if user is member
	isMember, err := s.groupRepo.IsMember(groupID, userID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, fmt.Errorf("you are not a member of this group")
	}

	settlements, total, err := s.settlementRepo.ListByGroup(groupID, page, pageSize)
	if err != nil {
		return nil, err
	}

	responses := make([]models.SettlementResponse, len(settlements))
	for i, settlement := range settlements {
		responses[i] = settlement.ToResponse()
	}

	return &models.SettlementListResponse{
		Settlements: responses,
		Total:       total,
		Page:        page,
		PageSize:    pageSize,
	}, nil
}

// GetSettlementSummary gets settlement summary for a user
func (s *SettlementService) GetSettlementSummary(userID uuid.UUID) (*models.SettlementSummary, error) {
	return s.settlementRepo.GetSettlementSummary(userID)
}

// GetBalanceWithUser gets the net balance between two users
func (s *SettlementService) GetBalanceWithUser(userID, otherUserID uuid.UUID) (map[string]interface{}, error) {
	if userID == otherUserID {
		return nil, fmt.Errorf("cannot get balance with yourself")
	}

	// Get settlements between users
	settlements, err := s.settlementRepo.ListBetweenUsers(userID, otherUserID)
	if err != nil {
		return nil, err
	}

	// Calculate net balance
	var paidToOther decimal.Decimal
	var receivedFromOther decimal.Decimal

	for _, settlement := range settlements {
		if settlement.FromUserID == userID {
			paidToOther = paidToOther.Add(settlement.Amount)
		} else {
			receivedFromOther = receivedFromOther.Add(settlement.Amount)
		}
	}

	netBalance := receivedFromOther.Sub(paidToOther)

	return map[string]interface{}{
		"user_id":             userID,
		"other_user_id":       otherUserID,
		"paid_to_other":       paidToOther,
		"received_from_other": receivedFromOther,
		"net_balance":         netBalance,
		"net_balance_text":    s.formatBalanceText(netBalance),
	}, nil
}

// formatBalanceText formats the balance for display
func (s *SettlementService) formatBalanceText(balance decimal.Decimal) string {
	if balance.GreaterThan(decimal.Zero) {
		return fmt.Sprintf("They owe you $%s", balance.Abs().StringFixed(2))
	} else if balance.LessThan(decimal.Zero) {
		return fmt.Sprintf("You owe them $%s", balance.Abs().StringFixed(2))
	}
	return "All settled up"
}
