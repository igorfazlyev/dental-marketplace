package handlers

import (
	"bytes"
	"dental-marketplace/config"
	"dental-marketplace/models"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"time"

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

// UploadDICOM handles DICOM file upload - sends directly to Diagnocat by default
func UploadDICOM(c *gin.Context) {
	userID := c.GetUint("user_id")

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
		return
	}

	// Validate DICOM file
	if !strings.HasSuffix(strings.ToLower(file.Filename), ".dcm") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Only DICOM (.dcm) files are allowed"})
		return
	}

	// Check upload destination preference (default: diagnocat)
	destination := c.DefaultPostForm("destination", "diagnocat") // "diagnocat" or "orthanc"

	// Open the uploaded file
	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to open file"})
		return
	}
	defer src.Close()

	// Read file content
	fileContent, err := io.ReadAll(src)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read file"})
		return
	}

	if destination == "diagnocat" {
		// Direct upload to Diagnocat
		handleDiagnocatDirectUpload(c, userID, file, fileContent)
	} else {
		// Legacy Orthanc upload
		handleOrthancUpload(c, userID, file, fileContent)
	}
}

// handleDiagnocatDirectUpload uploads directly to Diagnocat
// handleDiagnocatDirectUpload uploads directly to Diagnocat
func handleDiagnocatDirectUpload(c *gin.Context, userID uint, file *multipart.FileHeader, fileContent []byte) {
	if diagnocatClient == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Diagnocat service not configured"})
		return
	}

	// Generate unique study UID
	studyUID := fmt.Sprintf("study_%d_%d", userID, time.Now().Unix())
	patientUID := fmt.Sprintf("patient_%d", userID)

	// Step 1: Open Diagnocat session
	sessionResp, err := diagnocatClient.OpenSession(studyUID, patientUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to open Diagnocat session: " + err.Error()})
		return
	}

	// Step 2: Request upload URL
	fileKey := file.Filename
	uploadURLResp, err := diagnocatClient.RequestUploadURLs(sessionResp.SessionID, []string{fileKey})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get upload URL: " + err.Error()})
		return
	}

	// FIX: UploadURLs is an array, not a map
	var uploadURL string
	for _, urlObj := range uploadURLResp.UploadURLs {
		if urlObj.Key == fileKey {
			uploadURL = urlObj.URL
			break
		}
	}

	if uploadURL == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Upload URL not found"})
		return
	}

	// Step 3: Upload file to S3
	err = diagnocatClient.UploadFileToS3(uploadURL, fileContent)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload to Diagnocat: " + err.Error()})
		return
	}

	// Step 4: Close session to trigger analysis
	err = diagnocatClient.CloseSession(sessionResp.SessionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to close session: " + err.Error()})
		return
	}

	// Step 5: Create study record in our database
	study := models.Study{
		PatientID:   userID,
		Description: file.Filename,
		FileSize:    int64(len(fileContent)),
		Status:      "uploaded",
	}

	if err := config.DB.Create(&study).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save study"})
		return
	}

	// Step 6: Create Diagnocat analysis record
	analysis := models.DiagnocatAnalysis{
		StudyID:            study.ID,
		DiagnocatStudyUID:  studyUID,
		DiagnocatSessionID: sessionResp.SessionID,
		Status:             "processing",
		Complete:           false,
		Started:            true,
	}

	if err := config.DB.Create(&analysis).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save analysis record"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "File uploaded to Diagnocat successfully",
		"study":    study,
		"analysis": analysis,
	})
}

// handleOrthancUpload - legacy Orthanc-first upload (optional)
func handleOrthancUpload(c *gin.Context, userID uint, file *multipart.FileHeader, fileContent []byte) {
	// Your existing Orthanc upload logic here
	orthancURL := os.Getenv("ORTHANC_URL")
	if orthancURL == "" {
		orthancURL = "http://localhost:8042"
	}

	req, err := http.NewRequest("POST", orthancURL+"/instances", bytes.NewReader(fileContent))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create Orthanc request"})
		return
	}

	req.Header.Set("Content-Type", "application/dicom")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload to Orthanc"})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Orthanc upload failed"})
		return
	}

	var orthancResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&orthancResp); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse Orthanc response"})
		return
	}

	studyID, _ := orthancResp["ParentStudy"].(string)

	study := models.Study{
		PatientID:      userID,
		OrthancStudyID: studyID,
		Description:    file.Filename,
		FileSize:       int64(len(fileContent)),
		Status:         "uploaded",
	}

	if err := config.DB.Create(&study).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save study"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "File uploaded to Orthanc successfully",
		"study":   study,
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
