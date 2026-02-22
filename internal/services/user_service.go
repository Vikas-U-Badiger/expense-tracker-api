package services

import (
	"expense-tracker-api/internal/config"
	"expense-tracker-api/internal/models"
	"expense-tracker-api/internal/repositories"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// UserService handles business logic for users
type UserService struct {
	userRepo  *repositories.UserRepository
	jwtConfig *config.JWTConfig
}

// NewUserService creates a new user service
func NewUserService(userRepo *repositories.UserRepository, jwtConfig *config.JWTConfig) *UserService {
	return &UserService{
		userRepo:  userRepo,
		jwtConfig: jwtConfig,
	}
}

// Register creates a new user account
func (s *UserService) Register(req *models.RegisterRequest) (*models.LoginResponse, error) {
	// Check if email already exists
	exists, err := s.userRepo.Exists(req.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to check email existence: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("email already registered")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user
	user := &models.User{
		Name:     req.Name,
		Email:    req.Email,
		Password: string(hashedPassword),
	}

	if err := s.userRepo.Create(user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Generate token
	token, err := s.generateToken(user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	return &models.LoginResponse{
		User:  user.ToResponse(),
		Token: token,
	}, nil
}

// Login authenticates a user and returns a token
func (s *UserService) Login(req *models.LoginRequest) (*models.LoginResponse, error) {
	// Find user by email
	user, err := s.userRepo.FindByEmailWithPassword(req.Email)
	if err != nil {
		return nil, fmt.Errorf("invalid email or password")
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, fmt.Errorf("invalid email or password")
	}

	// Generate token
	token, err := s.generateToken(user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	return &models.LoginResponse{
		User:  user.ToResponse(),
		Token: token,
	}, nil
}

// GetUserByID gets a user by ID
func (s *UserService) GetUserByID(id uuid.UUID) (*models.UserResponse, error) {
	user, err := s.userRepo.FindByID(id)
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}

	resp := user.ToResponse()
	return &resp, nil
}

// UpdateUser updates a user's information
func (s *UserService) UpdateUser(id uuid.UUID, name string) (*models.UserResponse, error) {
	user, err := s.userRepo.FindByID(id)
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}

	if name != "" {
		user.Name = name
	}

	if err := s.userRepo.Update(user); err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	resp := user.ToResponse()
	return &resp, nil
}

// ChangePassword changes a user's password
func (s *UserService) ChangePassword(id uuid.UUID, oldPassword, newPassword string) error {
	user, err := s.userRepo.FindByID(id)
	if err != nil {
		return fmt.Errorf("user not found")
	}

	// Verify old password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(oldPassword)); err != nil {
		return fmt.Errorf("incorrect old password")
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	user.Password = string(hashedPassword)
	return s.userRepo.Update(user)
}

// ListUsers returns a paginated list of users
func (s *UserService) ListUsers(page, pageSize int) ([]models.UserResponse, int64, error) {
	users, total, err := s.userRepo.List(page, pageSize)
	if err != nil {
		return nil, 0, err
	}

	responses := make([]models.UserResponse, len(users))
	for i, user := range users {
		responses[i] = user.ToResponse()
	}

	return responses, total, nil
}

// GetUserDashboard gets dashboard data for a user
func (s *UserService) GetUserDashboard(userID uuid.UUID) (map[string]interface{}, error) {
	// Get user info
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}

	// Get balances
	balances, err := s.userRepo.GetUserBalances(userID)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"user":     user.ToResponse(),
		"balances": balances,
	}, nil
}

// generateToken generates a JWT token for a user
func (s *UserService) generateToken(userID uuid.UUID) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID.String(),
		"exp":     time.Now().Add(time.Hour * time.Duration(s.jwtConfig.TokenExpiry)).Unix(),
		"iat":     time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtConfig.Secret))
}
