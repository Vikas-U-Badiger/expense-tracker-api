package services

import (
	"expense-tracker-api/internal/models"
	"expense-tracker-api/internal/repositories"
	"expense-tracker-api/pkg/settlement"
	"fmt"

	"github.com/google/uuid"
)

// GroupService handles business logic for groups
type GroupService struct {
	groupRepo *repositories.GroupRepository
	userRepo  *repositories.UserRepository
	calc      *settlement.Calculator
}

// NewGroupService creates a new group service
func NewGroupService(groupRepo *repositories.GroupRepository, userRepo *repositories.UserRepository) *GroupService {
	return &GroupService{
		groupRepo: groupRepo,
		userRepo:  userRepo,
		calc:      settlement.NewCalculator(),
	}
}

// CreateGroup creates a new group
func (s *GroupService) CreateGroup(userID uuid.UUID, req *models.CreateGroupRequest) (*models.GroupResponse, error) {
	// Validate member IDs exist
	for _, memberID := range req.MemberIDs {
		if _, err := s.userRepo.FindByID(memberID); err != nil {
			return nil, fmt.Errorf("member not found: %s", memberID)
		}
	}

	// Create group
	group := &models.Group{
		Name:        req.Name,
		Description: req.Description,
		CreatedByID: userID,
	}

	if err := s.groupRepo.Create(group, req.MemberIDs); err != nil {
		return nil, fmt.Errorf("failed to create group: %w", err)
	}

	// Fetch the created group with members
	createdGroup, err := s.groupRepo.FindByID(group.ID)
	if err != nil {
		return nil, err
	}

	resp := createdGroup.ToResponse()
	return &resp, nil
}

// GetGroup gets a group by ID
func (s *GroupService) GetGroup(groupID, userID uuid.UUID) (*models.GroupDetailResponse, error) {
	// Check if user is member
	isMember, err := s.groupRepo.IsMember(groupID, userID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, fmt.Errorf("you are not a member of this group")
	}

	// Get group with expenses
	group, err := s.groupRepo.FindByIDWithExpenses(groupID)
	if err != nil {
		return nil, fmt.Errorf("group not found")
	}

	// Get balances
	balances, err := s.groupRepo.GetGroupBalances(groupID)
	if err != nil {
		return nil, err
	}

	// Calculate simplified debts
	settlementBalances := make([]settlement.Balance, len(balances))
	userNames := make(map[uuid.UUID]string)
	for i, b := range balances {
		settlementBalances[i] = settlement.Balance{
			UserID:   b.UserID,
			UserName: b.UserName,
			Amount:   b.Balance,
		}
		userNames[b.UserID] = b.UserName
	}

	simplifiedTransactions := s.calc.CalculateSimplifiedDebts(settlementBalances)

	// Convert to response format
	simplifiedDebts := make([]models.SimplifiedDebt, len(simplifiedTransactions))
	for i, t := range simplifiedTransactions {
		simplifiedDebts[i] = models.SimplifiedDebt{
			FromUserID:   t.FromUserID,
			FromUserName: t.FromUserName,
			ToUserID:     t.ToUserID,
			ToUserName:   t.ToUserName,
			Amount:       t.Amount,
		}
	}

	// Get total expenses
	totalExpenses, err := s.groupRepo.GetTotalExpenses(groupID)
	if err != nil {
		return nil, err
	}

	return &models.GroupDetailResponse{
		GroupResponse:   group.ToResponse(),
		Balances:        balances,
		SimplifiedDebts: simplifiedDebts,
		TotalExpenses:   totalExpenses,
	}, nil
}

// UpdateGroup updates a group
func (s *GroupService) UpdateGroup(groupID, userID uuid.UUID, req *models.UpdateGroupRequest) (*models.GroupResponse, error) {
	// Get group
	group, err := s.groupRepo.FindByID(groupID)
	if err != nil {
		return nil, fmt.Errorf("group not found")
	}

	// Check if user is creator
	if group.CreatedByID != userID {
		return nil, fmt.Errorf("only the group creator can update the group")
	}

	// Update fields
	if req.Name != "" {
		group.Name = req.Name
	}
	group.Description = req.Description

	if err := s.groupRepo.Update(group); err != nil {
		return nil, fmt.Errorf("failed to update group: %w", err)
	}

	resp := group.ToResponse()
	return &resp, nil
}

// DeleteGroup deletes a group
func (s *GroupService) DeleteGroup(groupID, userID uuid.UUID) error {
	// Get group
	group, err := s.groupRepo.FindByID(groupID)
	if err != nil {
		return fmt.Errorf("group not found")
	}

	// Check if user is creator
	if group.CreatedByID != userID {
		return fmt.Errorf("only the group creator can delete the group")
	}

	return s.groupRepo.Delete(groupID)
}

// ListGroups lists groups for a user
func (s *GroupService) ListGroups(userID uuid.UUID, page, pageSize int) ([]models.GroupResponse, int64, error) {
	groups, total, err := s.groupRepo.List(userID, page, pageSize)
	if err != nil {
		return nil, 0, err
	}

	responses := make([]models.GroupResponse, len(groups))
	for i, group := range groups {
		responses[i] = group.ToResponse()
	}

	return responses, total, nil
}

// AddMember adds a member to a group
func (s *GroupService) AddMember(groupID, userID uuid.UUID, req *models.AddMemberRequest) error {
	// Get group
	group, err := s.groupRepo.FindByID(groupID)
	if err != nil {
		return fmt.Errorf("group not found")
	}

	// Check if user is creator
	if group.CreatedByID != userID {
		return fmt.Errorf("only the group creator can add members")
	}

	// Check if member exists
	if _, err := s.userRepo.FindByID(req.UserID); err != nil {
		return fmt.Errorf("user not found")
	}

	// Check if already member
	isMember, err := s.groupRepo.IsMember(groupID, req.UserID)
	if err != nil {
		return err
	}
	if isMember {
		return fmt.Errorf("user is already a member of this group")
	}

	return s.groupRepo.AddMember(groupID, req.UserID)
}

// RemoveMember removes a member from a group
func (s *GroupService) RemoveMember(groupID, userID, memberID uuid.UUID) error {
	// Get group
	group, err := s.groupRepo.FindByID(groupID)
	if err != nil {
		return fmt.Errorf("group not found")
	}

	// Check if user is creator or removing themselves
	if group.CreatedByID != userID && userID != memberID {
		return fmt.Errorf("only the group creator can remove other members")
	}

	// Cannot remove creator
	if memberID == group.CreatedByID {
		return fmt.Errorf("cannot remove the group creator")
	}

	// Check if is member
	isMember, err := s.groupRepo.IsMember(groupID, memberID)
	if err != nil {
		return err
	}
	if !isMember {
		return fmt.Errorf("user is not a member of this group")
	}

	return s.groupRepo.RemoveMember(groupID, memberID)
}

// GetGroupBalances gets balances for a group
func (s *GroupService) GetGroupBalances(groupID, userID uuid.UUID) ([]models.GroupBalance, error) {
	// Check if user is member
	isMember, err := s.groupRepo.IsMember(groupID, userID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, fmt.Errorf("you are not a member of this group")
	}

	return s.groupRepo.GetGroupBalances(groupID)
}

// GetSimplifiedDebts gets simplified debts for a group
func (s *GroupService) GetSimplifiedDebts(groupID, userID uuid.UUID) ([]models.SimplifiedDebt, error) {
	// Check if user is member
	isMember, err := s.groupRepo.IsMember(groupID, userID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, fmt.Errorf("you are not a member of this group")
	}

	// Get balances
	balances, err := s.groupRepo.GetGroupBalances(groupID)
	if err != nil {
		return nil, err
	}

	// Convert to settlement balances
	settlementBalances := make([]settlement.Balance, len(balances))
	for i, b := range balances {
		settlementBalances[i] = settlement.Balance{
			UserID:   b.UserID,
			UserName: b.UserName,
			Amount:   b.Balance,
		}
	}

	// Calculate simplified debts
	transactions := s.calc.CalculateSimplifiedDebts(settlementBalances)

	// Convert to response
	simplifiedDebts := make([]models.SimplifiedDebt, len(transactions))
	for i, t := range transactions {
		simplifiedDebts[i] = models.SimplifiedDebt{
			FromUserID:   t.FromUserID,
			FromUserName: t.FromUserName,
			ToUserID:     t.ToUserID,
			ToUserName:   t.ToUserName,
			Amount:       t.Amount,
		}
	}

	return simplifiedDebts, nil
}
