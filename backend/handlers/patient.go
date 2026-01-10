package handlers

import (
	"dental-marketplace/config"
	"dental-marketplace/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

func GetMyStudies(c *gin.Context) {
	userID := c.GetUint("user_id")

	var studies []models.Study
	if err := config.DB.Where("patient_id = ?", userID).Order("created_at DESC").Find(&studies).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch studies"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"studies": studies,
		"count":   len(studies),
	})
}
