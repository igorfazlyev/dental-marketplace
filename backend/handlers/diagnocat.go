package handlers

import (
	"dental-marketplace/config"
	"dental-marketplace/models"
	"dental-marketplace/pkg/diagnocat"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

var diagnocatClient *diagnocat.Client

func init() {
	apiKey := os.Getenv("DIAGNOCAT_API_KEY")
	if apiKey != "" {
		diagnocatClient = diagnocat.NewClient(apiKey)
	}
}

// SendToDiagnocat sends a study to Diagnocat for analysis
func SendToDiagnocat(c *gin.Context) {
	userID := c.GetUint("user_id")

	var req struct {
		StudyID uint `json:"study_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get study from database
	var study models.Study
	if err := config.DB.Where("id = ? AND patient_id = ?", req.StudyID, userID).First(&study).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Study not found"})
		return
	}

	// Check if already sent to Diagnocat
	var existing models.DiagnocatAnalysis
	if err := config.DB.Where("study_id = ?", study.ID).First(&existing).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Study already sent to Diagnocat"})
		return
	}

	if diagnocatClient == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Diagnocat not configured"})
		return
	}

	// Step 1: Open session
	studyUID := fmt.Sprintf("study_%d", study.ID)
	patientUID := fmt.Sprintf("patient_%d", userID)

	sessionResp, err := diagnocatClient.OpenSession(studyUID, patientUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to open session: %v", err)})
		return
	}

	// Create analysis record
	analysis := models.DiagnocatAnalysis{
		StudyID:            study.ID,
		DiagnocatStudyUID:  studyUID,
		DiagnocatSessionID: sessionResp.SessionID,
		Status:             "uploading",
	}
	config.DB.Create(&analysis)

	// Step 2: Get DICOM files from Orthanc
	files, err := getOrthancDICOMFiles(study.OrthancStudyID)
	if err != nil {
		analysis.Status = "failed"
		analysis.Error = err.Error()
		config.DB.Save(&analysis)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get DICOM files: %v", err)})
		return
	}

	// Step 3: Request upload URLs
	fileKeys := make([]string, len(files))
	for i := range files {
		fileKeys[i] = fmt.Sprintf("file_%d.dcm", i)
	}

	urlsResp, err := diagnocatClient.RequestUploadURLs(sessionResp.SessionID, fileKeys)
	if err != nil {
		analysis.Status = "failed"
		analysis.Error = err.Error()
		config.DB.Save(&analysis)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get upload URLs: %v", err)})
		return
	}

	// Step 4: Upload files to S3
	for i, uploadURL := range urlsResp.UploadURLs {
		if err := diagnocatClient.UploadFile(uploadURL.URL, files[i]); err != nil {
			analysis.Status = "failed"
			analysis.Error = err.Error()
			config.DB.Save(&analysis)
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to upload file: %v", err)})
			return
		}
	}

	// Step 5: Close session (triggers analysis)
	if err := diagnocatClient.CloseSession(sessionResp.SessionID); err != nil {
		analysis.Status = "failed"
		analysis.Error = err.Error()
		config.DB.Save(&analysis)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to close session: %v", err)})
		return
	}

	analysis.Status = "processing"
	config.DB.Save(&analysis)

	c.JSON(http.StatusOK, gin.H{
		"message":  "Study sent to Diagnocat for analysis",
		"analysis": analysis,
	})
}

// GetDiagnocatAnalyses lists all Diagnocat analyses for current user
func GetDiagnocatAnalyses(c *gin.Context) {
	userID := c.GetUint("user_id")

	var analyses []models.DiagnocatAnalysis
	err := config.DB.
		Joins("JOIN studies ON studies.id = diagnocat_analyses.study_id").
		Where("studies.patient_id = ?", userID).
		Preload("Study").
		Order("diagnocat_analyses.created_at DESC").
		Find(&analyses).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch analyses"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"analyses": analyses,
		"count":    len(analyses),
	})
}

// RefreshDiagnocatAnalysis polls Diagnocat for updated analysis status
// RefreshDiagnocatAnalysis polls Diagnocat for updated analysis status
func RefreshDiagnocatAnalysis(c *gin.Context) {
	userID := c.GetUint("user_id")
	analysisID := c.Param("id")

	var analysis models.DiagnocatAnalysis
	err := config.DB.
		Joins("JOIN studies ON studies.id = diagnocat_analyses.study_id").
		Where("diagnocat_analyses.id = ? AND studies.patient_id = ?", analysisID, userID).
		First(&analysis).Error

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Analysis not found"})
		return
	}

	if analysis.Complete && analysis.Diagnoses.Diagnoses != nil {
		c.JSON(http.StatusOK, gin.H{"analysis": analysis, "message": "Analysis already complete with data"})
		return
	}

	if diagnocatClient == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Diagnocat not configured"})
		return
	}

	// Poll Diagnocat for results
	patientUID := fmt.Sprintf("patient_%d", userID)
	diagnocatAnalyses, err := diagnocatClient.GetAnalyses(patientUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch from Diagnocat"})
		return
	}

	// Find matching analysis
	for _, da := range diagnocatAnalyses {
		if da.StudyUID == analysis.DiagnocatStudyUID {
			// Update basic info
			analysis.AnalysisUID = da.UID
			analysis.AnalysisType = da.AnalysisType
			analysis.Complete = da.Complete
			analysis.Started = da.Started
			analysis.Error = da.Error
			analysis.PDFUrl = da.PDFUrl
			analysis.PreviewUrl = da.PreviewUrl
			analysis.WebpageUrl = da.WebpageUrl

			if da.Complete {
				analysis.Status = "complete"

				// Fetch detailed diagnoses
				diagnoses, err := diagnocatClient.GetDiagnoses(da.UID)
				if err == nil && diagnoses != nil {
					// Convert to our model format
					analysis.Diagnoses = models.DiagnosesData{
						Diagnoses: convertDiagnoses(diagnoses.Diagnoses),
					}
				}

				// Fetch ortho measurements (if applicable)
				orthoMeasurements, err := diagnocatClient.GetOrthoMeasurements(da.UID)
				if err == nil && orthoMeasurements != nil {
					analysis.OrthoMeasurements = models.OrthoMeasurements{
						CephalometricMeasurements: convertCephalometric(orthoMeasurements.CephalometricMeasurements),
						TeethAnalysis:             convertTeethAnalysis(orthoMeasurements.TeethAnalysis),
					}
				}
			} else if da.Started {
				analysis.Status = "processing"
			}

			config.DB.Save(&analysis)
			break
		}
	}

	c.JSON(http.StatusOK, gin.H{"analysis": analysis})
}

// Helper functions to convert API types to model types
func convertDiagnoses(apiDiagnoses []diagnocat.ToothDiagnosis) []models.ToothDiagnosis {
	result := make([]models.ToothDiagnosis, len(apiDiagnoses))
	for i, d := range apiDiagnoses {
		result[i] = models.ToothDiagnosis{
			ToothNumber:       d.ToothNumber,
			Attributes:        convertAttributes(d.Attributes),
			PeriodontalStatus: convertPeriodontalStatus(d.PeriodontalStatus),
			TextComment:       d.TextComment,
		}
	}
	return result
}

func convertAttributes(apiAttrs []diagnocat.AttributeData) []models.AttributeData {
	result := make([]models.AttributeData, len(apiAttrs))
	for i, a := range apiAttrs {
		result[i] = models.AttributeData{
			AttributeID:   a.AttributeID,
			ModelPositive: a.ModelPositive,
			UserDecision:  a.UserDecision,
			UserPositive:  a.UserPositive,
		}
	}
	return result
}

func convertPeriodontalStatus(apiStatus diagnocat.PeriodontalStatus) models.PeriodontalStatus {
	roots := make([]models.RootMeasurement, len(apiStatus.Roots))
	for i, r := range apiStatus.Roots {
		roots[i] = models.RootMeasurement{
			Root:         r.Root,
			Measurements: r.Measurements,
		}
	}

	sites := make([]models.SiteMeasurement, len(apiStatus.Sites))
	for i, s := range apiStatus.Sites {
		sites[i] = models.SiteMeasurement{
			Site:         s.Site,
			Measurements: s.Measurements,
		}
	}

	return models.PeriodontalStatus{
		Roots: roots,
		Sites: sites,
	}
}

func convertCephalometric(apiCeph []diagnocat.CephalometricMeasurement) []models.CephalometricMeasurement {
	result := make([]models.CephalometricMeasurement, len(apiCeph))
	for i, c := range apiCeph {
		result[i] = models.CephalometricMeasurement{
			Type:   c.Type,
			Values: c.Values,
			Unit:   c.Unit,
		}
	}
	return result
}

func convertTeethAnalysis(apiTeeth diagnocat.TeethAnalysis) models.TeethAnalysis {
	return models.TeethAnalysis{
		CurveOfSpee:   apiTeeth.CurveOfSpee,
		Ponts:         apiTeeth.Ponts,
		SpaceCrowding: apiTeeth.SpaceCrowding,
		TonnBolton:    apiTeeth.TonnBolton,
	}
}

// Helper function to get DICOM files from Orthanc
func getOrthancDICOMFiles(studyID string) ([][]byte, error) {
	// Get list of instances in study
	url := fmt.Sprintf("%s/studies/%s/instances", orthancURL, studyID)
	req, _ := http.NewRequest("GET", url, nil)
	req.SetBasicAuth(orthancUser, orthancPass)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var instances []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&instances); err != nil {
		return nil, err
	}

	// Download each instance as DICOM
	var files [][]byte
	for _, instance := range instances {
		instanceID := instance["ID"].(string)
		fileURL := fmt.Sprintf("%s/instances/%s/file", orthancURL, instanceID)

		req, _ := http.NewRequest("GET", fileURL, nil)
		req.SetBasicAuth(orthancUser, orthancPass)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			continue
		}

		fileData, err := io.ReadAll(resp.Body)
		resp.Body.Close()

		if err == nil {
			files = append(files, fileData)
		}
	}

	return files, nil
}
