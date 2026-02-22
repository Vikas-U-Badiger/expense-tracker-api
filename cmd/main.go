package main

import (
	"expense-tracker-api/internal/config"
	"expense-tracker-api/internal/handlers"
	"expense-tracker-api/internal/middleware"
	"expense-tracker-api/internal/repositories"
	"expense-tracker-api/internal/services"
	"log"
	"os"

	"github.com/gin-gonic/gin"
)

// @title Expense Tracker API
// @version 1.0
// @description A REST API for tracking shared expenses and splitting bills among friends
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.email support@expensetracker.com

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8080
// @BasePath /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

func main() {
	// Load configuration
	cfg := config.Load()

	// Set Gin mode
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	// Initialize database
	db, err := config.InitDatabase(&cfg.Database)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// Run migrations
	if err := config.AutoMigrate(db); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Initialize repositories
	userRepo := repositories.NewUserRepository(db)
	groupRepo := repositories.NewGroupRepository(db)
	expenseRepo := repositories.NewExpenseRepository(db)
	settlementRepo := repositories.NewSettlementRepository(db)

	// Initialize services
	userService := services.NewUserService(userRepo, &cfg.JWT)
	groupService := services.NewGroupService(groupRepo, userRepo)
	expenseService := services.NewExpenseService(expenseRepo, groupRepo, userRepo)
	settlementService := services.NewSettlementService(settlementRepo, groupRepo)

	// Initialize handlers
	userHandler := handlers.NewUserHandler(userService)
	groupHandler := handlers.NewGroupHandler(groupService)
	expenseHandler := handlers.NewExpenseHandler(expenseService)
	settlementHandler := handlers.NewSettlementHandler(settlementService)

	// Initialize middleware
	authMiddleware := middleware.AuthMiddleware(&cfg.JWT)

	// Setup router
	router := gin.New()
	router.Use(middleware.Recovery())
	router.Use(middleware.CORS(cfg.Server.AllowOrigins))
	router.Use(middleware.ErrorHandler())
	router.Use(gin.Logger())

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "healthy",
			"service": "expense-tracker-api",
			"version": "1.0.0",
		})
	})

	// API routes
	api := router.Group("/api/v1")
	{
		userHandler.RegisterRoutes(api, authMiddleware)
		groupHandler.RegisterRoutes(api, authMiddleware)
		expenseHandler.RegisterRoutes(api, authMiddleware)
		settlementHandler.RegisterRoutes(api, authMiddleware)
	}

	// Start server
	port := cfg.Server.Port
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s...", port)
	log.Printf("Environment: %s", cfg.Server.Environment)

	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
		os.Exit(1)
	}
}
