package main

import (
	"dental-marketplace/config"
	"dental-marketplace/handlers"
	"dental-marketplace/middleware"
	"dental-marketplace/seed"
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	// Wait for database to be ready
	time.Sleep(5 * time.Second)

	// Connect to database
	config.ConnectDatabase()

	// Seed database if SEED_DB environment variable is set
	if os.Getenv("SEED_DB") == "true" {
		seed.SeedData(config.DB)
	}

	// Setup Gin router
	router := gin.Default()

	// CORS middleware
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Public routes
	public := router.Group("/api")
	{
		public.POST("/register", handlers.Register)
		public.POST("/login", handlers.Login)
	}

	// Protected routes
	protected := router.Group("/api")
	protected.Use(middleware.AuthMiddleware())
	{
		protected.GET("/me", handlers.GetMe)

		// Patient routes
		patient := protected.Group("/patient")
		patient.Use(middleware.RoleMiddleware("patient"))
		{
			patient.POST("/upload", handlers.UploadDICOM)
			patient.GET("/studies", handlers.GetMyStudies)
		}

		// Clinic routes
		clinic := protected.Group("/clinic")
		clinic.Use(middleware.RoleMiddleware("clinic"))
		{
			clinic.GET("/dashboard", func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "Clinic dashboard - Coming soon"})
			})
		}
	}

	log.Println("Server starting on :8080")
	router.Run(":8080")
}
