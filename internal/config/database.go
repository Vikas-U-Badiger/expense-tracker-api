package config

import (
	"expense-tracker-api/internal/models"
	"fmt"
	"log"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DB is the global database instance
var DB *gorm.DB

// InitDatabase initializes the database connection
func InitDatabase(cfg *DatabaseConfig) (*gorm.DB, error) {
	var dialector gorm.Dialector

	if cfg.Driver == "sqlite" {
		dialector = sqlite.Open(cfg.GetDatabaseURL())
		log.Println("Using SQLite database")
	} else {
		dialector = postgres.Open(cfg.GetDatabaseURL())
		log.Println("Using PostgreSQL database")
	}

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	DB = db
	return db, nil
}

// AutoMigrate runs database migrations
func AutoMigrate(db *gorm.DB) error {
	log.Println("Running database migrations...")

	err := db.AutoMigrate(
		&models.User{},
		&models.Group{},
		&models.GroupMember{},
		&models.Expense{},
		&models.ExpenseShare{},
		&models.Settlement{},
	)
	if err != nil {
		return fmt.Errorf("failed to migrate database: %w", err)
	}

	// Register UUID generation callbacks
	registerUUIDCallbacks(db)

	log.Println("Database migrations completed successfully")
	return nil
}

// registerUUIDCallbacks registers callbacks to generate UUIDs before creating records
func registerUUIDCallbacks(db *gorm.DB) {
	// BeforeCreate callbacks for UUID generation
	db.Callback().Create().Before("gorm:create").Register("generate_user_uuid", func(db *gorm.DB) {
		if user, ok := db.Statement.Dest.(*models.User); ok {
			if user.ID == uuid.Nil {
				user.ID = uuid.New()
			}
		}
	})

	db.Callback().Create().Before("gorm:create").Register("generate_group_uuid", func(db *gorm.DB) {
		if group, ok := db.Statement.Dest.(*models.Group); ok {
			if group.ID == uuid.Nil {
				group.ID = uuid.New()
			}
		}
	})

	db.Callback().Create().Before("gorm:create").Register("generate_expense_uuid", func(db *gorm.DB) {
		if expense, ok := db.Statement.Dest.(*models.Expense); ok {
			if expense.ID == uuid.Nil {
				expense.ID = uuid.New()
			}
		}
	})

	db.Callback().Create().Before("gorm:create").Register("generate_expense_share_uuid", func(db *gorm.DB) {
		if share, ok := db.Statement.Dest.(*models.ExpenseShare); ok {
			if share.ID == uuid.Nil {
				share.ID = uuid.New()
			}
		}
	})

	db.Callback().Create().Before("gorm:create").Register("generate_settlement_uuid", func(db *gorm.DB) {
		if settlement, ok := db.Statement.Dest.(*models.Settlement); ok {
			if settlement.ID == uuid.Nil {
				settlement.ID = uuid.New()
			}
		}
	})
}

// SeedData seeds the database with initial data for testing
func SeedData(db *gorm.DB) error {
	log.Println("Seeding database with test data...")

	// Check if we already have users
	var count int64
	if err := db.Model(&models.User{}).Count(&count).Error; err != nil {
		return err
	}

	if count > 0 {
		log.Println("Database already has data, skipping seed")
		return nil
	}

	// Create test users with UUIDs
	users := []models.User{
		{ID: uuid.New(), Name: "Alice Johnson", Email: "alice@example.com"},
		{ID: uuid.New(), Name: "Bob Smith", Email: "bob@example.com"},
		{ID: uuid.New(), Name: "Charlie Brown", Email: "charlie@example.com"},
		{ID: uuid.New(), Name: "Diana Prince", Email: "diana@example.com"},
	}

	for i := range users {
		users[i].Password = "$2a$10$YourHashedPasswordHere" // Placeholder, will be set properly
		if err := db.Create(&users[i]).Error; err != nil {
			return fmt.Errorf("failed to create user: %w", err)
		}
	}

	log.Printf("Created %d test users\n", len(users))
	return nil
}
