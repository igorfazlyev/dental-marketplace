package main

import "time"

type UserType string

const (
	UserTypePatient   UserType = "patient"
	UserTypeClinic    UserType = "clinic"
	UserTypeRegulator UserType = "regulator"
)

type User struct {
	ID       string
	Username string
	Password string
	Type     UserType
	Name     string
	Email    string
}

type Scan struct {
	ID          string
	PatientID   string
	UploadDate  time.Time
	Status      string // "processing", "analyzed", "offers_pending", "offers_received"
	ImagePath   string
	AIAnalysis  *AIAnalysis
}

type AIAnalysis struct {
	ScanID       string
	Diagnoses    []string
	TreatmentPlan []TreatmentItem
	CreatedAt    time.Time
}

type TreatmentItem struct {
	ID          string
	Description string
	ToothNumber string
	Category    string // "implant", "crown", "filling", etc.
}

type SearchCriteria struct {
	PatientID     string
	City          string
	PriceSegment  string
	Specialties   []string
	StartDate     time.Time
	NeedInstallment bool
}

type Offer struct {
	ID             string
	ScanID         string
	ClinicID       string
	ClinicName     string
	Rating         float64
	TotalCost      float64
	Duration       string
	Details        string
	Guarantees     string
	PaymentTerms   string
	Status         string // "pending", "selected", "rejected"
	CreatedAt      time.Time
}

type Consultation struct {
	ID         string
	PatientID  string
	OfferID    string
	ClinicID   string
	Status     string // "awaiting_call", "scheduled", "completed", "rejected"
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type Review struct {
	ID         string
	PatientID  string
	ClinicID   string
	DoctorName string
	Rating     int
	Text       string
	CreatedAt  time.Time
}

type Lead struct {
	ID            string
	ClinicID      string
	PatientID     string
	PatientName   string
	PatientPhone  string
	PatientEmail  string
	TreatmentPlan []TreatmentItem
	Status        string // "new", "contacted", "scheduled", "treatment_started", "rejected"
	CreatedAt     time.Time
}

type ClinicPricing struct {
	ClinicID string
	Items    map[string]float64 // category -> price
}

type RegionStats struct {
	Region              string
	TotalPlans          int
	PlannedRevenue      float64
	ActualRevenue       float64
	AverageImplantPrice float64
	AverageCrownPrice   float64
}
