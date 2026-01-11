package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

type DiagnocatAnalysis struct {
	ID                 uint              `gorm:"primaryKey" json:"id"`
	StudyID            uint              `json:"study_id"`
	Study              Study             `gorm:"foreignKey:StudyID" json:"-"`
	DiagnocatStudyUID  string            `json:"diagnocat_study_uid"`
	DiagnocatSessionID string            `json:"diagnocat_session_id"`
	AnalysisUID        string            `json:"analysis_uid"`
	AnalysisType       string            `json:"analysis_type"` // GP, CBCT, etc
	Status             string            `json:"status"`        // uploading, processing, complete, failed
	Complete           bool              `json:"complete"`
	Started            bool              `json:"started"`
	Error              string            `json:"error,omitempty"`
	PDFUrl             string            `json:"pdf_url,omitempty"`
	PreviewUrl         string            `json:"preview_url,omitempty"`
	WebpageUrl         string            `json:"webpage_url,omitempty"`
	Diagnoses          DiagnosesData     `gorm:"type:jsonb" json:"diagnoses,omitempty"`
	OrthoMeasurements  OrthoMeasurements `gorm:"type:jsonb" json:"ortho_measurements,omitempty"`
	CreatedAt          time.Time         `json:"created_at"`
	UpdatedAt          time.Time         `json:"updated_at"`
}

// DiagnosesData stores tooth-by-tooth pathology findings
type DiagnosesData struct {
	Diagnoses []ToothDiagnosis `json:"diagnoses"`
}

type ToothDiagnosis struct {
	ToothNumber       int               `json:"tooth_number"`
	Attributes        []AttributeData   `json:"attributes"`
	PeriodontalStatus PeriodontalStatus `json:"periodontal_status,omitempty"`
	TextComment       string            `json:"text_comment,omitempty"`
}

type AttributeData struct {
	AttributeID   int  `json:"attribute_id"`
	ModelPositive bool `json:"model_positive"`
	UserDecision  bool `json:"user_decision"`
	UserPositive  bool `json:"user_positive"`
}

type PeriodontalStatus struct {
	Roots []RootMeasurement `json:"roots"`
	Sites []SiteMeasurement `json:"sites"`
}

type RootMeasurement struct {
	Root         string                 `json:"root"`
	Measurements map[string]interface{} `json:"measurements"`
}

type SiteMeasurement struct {
	Site         string                 `json:"site"`
	Measurements map[string]interface{} `json:"measurements"`
}

// OrthoMeasurements stores orthodontic analysis data
type OrthoMeasurements struct {
	CephalometricMeasurements []CephalometricMeasurement `json:"cephalometric_measurements"`
	TeethAnalysis             TeethAnalysis              `json:"teeth_analysis"`
}

type CephalometricMeasurement struct {
	Type   string                 `json:"type"`
	Values map[string]interface{} `json:"values"`
	Unit   string                 `json:"unit"`
}

type TeethAnalysis struct {
	CurveOfSpee   map[string]interface{} `json:"curve_of_spee"`
	Ponts         map[string]interface{} `json:"ponts"`
	SpaceCrowding map[string]interface{} `json:"space_crowding"`
	TonnBolton    map[string]interface{} `json:"tonn_bolton"`
}

// JSON serialization for custom types
func (d DiagnosesData) Value() (driver.Value, error) {
	return json.Marshal(d)
}

func (d *DiagnosesData) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, d)
}

func (o OrthoMeasurements) Value() (driver.Value, error) {
	return json.Marshal(o)
}

func (o *OrthoMeasurements) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, o)
}
