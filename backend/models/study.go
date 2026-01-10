package models

import (
	"time"

	"gorm.io/gorm"
)

type StudyStatus string

const (
	StatusUploaded   StudyStatus = "uploaded"
	StatusProcessing StudyStatus = "processing"
	StatusAnalyzed   StudyStatus = "analyzed"
	StatusFailed     StudyStatus = "failed"
)

type Study struct {
	ID              uint           `gorm:"primaryKey" json:"id"`
	PatientID       uint           `gorm:"not null;index" json:"patient_id"`
	OrthancStudyID  string         `gorm:"uniqueIndex" json:"orthanc_study_id"`
	StudyDate       *time.Time     `json:"study_date"`
	Description     string         `json:"description"`
	Status          StudyStatus    `gorm:"type:varchar(20);default:'uploaded'" json:"status"`
	FileSize        int64          `json:"file_size"`
	NumInstances    int            `json:"num_instances"`
	DiagnocatResult string         `gorm:"type:text" json:"diagnocat_result,omitempty"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`

	// Relations
	Patient User `gorm:"foreignKey:PatientID" json:"patient,omitempty"`
}
