package handlers

import (
	"bytes"
	"dental-marketplace/config"
	"dental-marketplace/models"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

var orthancURL = os.Getenv("ORTHANC_URL")
var orthancUser = os.Getenv("ORTHANC_USER")
var orthancPass = os.Getenv("ORTHANC_PASS")

type OrthancUploadResponse struct {
	ID          string `json:"ID"`
	Path        string `json:"Path"`
	Status      string `json:"Status"`
	ParentStudy string `json:"ParentStudy"`
}

func UploadDICOM(c *gin.Context) {
	userID := c.GetUint("user_id")

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
		return
	}
	defer file.Close()

	// Read file content
	fileBytes, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read file"})
		return
	}

	// Upload to Orthanc
	req, err := http.NewRequest("POST", orthancURL+"/instances", bytes.NewReader(fileBytes))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create request"})
		return
	}

	req.SetBasicAuth(orthancUser, orthancPass)
	req.Header.Set("Content-Type", "application/dicom")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload to Orthanc"})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Orthanc upload failed", "details": string(body)})
		return
	}

	var orthancResp OrthancUploadResponse
	json.NewDecoder(resp.Body).Decode(&orthancResp)

	// Get study information from Orthanc
	studyInfo, err := getOrthancStudyInfo(orthancResp.ParentStudy)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get study info"})
		return
	}

	// Save to database
	study := models.Study{
		PatientID:      userID,
		OrthancStudyID: orthancResp.ParentStudy,
		Status:         models.StatusUploaded,
		FileSize:       header.Size,
		NumInstances:   1,
		Description:    fmt.Sprintf("Study uploaded: %s", header.Filename),
	}

	if err := config.DB.Create(&study).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save study"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "File uploaded successfully",
		"study_id":   study.ID,
		"orthanc_id": orthancResp.ParentStudy,
		"study_info": studyInfo,
	})
}

func getOrthancStudyInfo(studyID string) (map[string]interface{}, error) {
	req, _ := http.NewRequest("GET", orthancURL+"/studies/"+studyID, nil)
	req.SetBasicAuth(orthancUser, orthancPass)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var studyInfo map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&studyInfo)
	return studyInfo, nil
}
