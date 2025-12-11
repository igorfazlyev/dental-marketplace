package main

import (
	"errors"
	"sync"
)

// Database interface - can be swapped for real database
type Database interface {
	// Users
	CreateUser(user *User) error
	GetUserByUsername(username string) (*User, error)
	GetUserByID(id string) (*User, error)

	// Scans
	CreateScan(scan *Scan) error
	GetScanByID(id string) (*Scan, error)
	GetScansByPatient(patientID string) ([]*Scan, error)
	UpdateScan(scan *Scan) error
	GetAllAvailableScans() ([]*Scan, error)

	// Offers
	CreateOffer(offer *Offer) error
	GetOfferByID(id string) (*Offer, error) // ADD THIS LINE
	GetOffersByScan(scanID string) ([]*Offer, error)
	GetOffersByClinic(clinicID string) ([]*Offer, error)
	UpdateOffer(offer *Offer) error

	// Consultations
	CreateConsultation(consultation *Consultation) error
	GetConsultationsByPatient(patientID string) ([]*Consultation, error)
	UpdateConsultation(consultation *Consultation) error

	// Leads
	CreateLead(lead *Lead) error
	GetLeadsByClinic(clinicID string) ([]*Lead, error)

	// Reviews
	CreateReview(review *Review) error
	GetReviewsByClinic(clinicID string) ([]*Review, error)

	// Pricing
	SaveClinicPricing(pricing *ClinicPricing) error
	GetClinicPricing(clinicID string) (*ClinicPricing, error)

	// Analytics
	GetRegionStats(region string) (*RegionStats, error)
	GetClinicStats(clinicID string) map[string]interface{}
}

// InMemoryDatabase implementation
type InMemoryDatabase struct {
	users         map[string]*User
	scans         map[string]*Scan
	offers        map[string]*Offer
	consultations map[string]*Consultation
	leads         map[string]*Lead
	reviews       map[string]*Review
	pricing       map[string]*ClinicPricing
	mu            sync.RWMutex
}

func NewInMemoryDatabase() *InMemoryDatabase {
	return &InMemoryDatabase{
		users:         make(map[string]*User),
		scans:         make(map[string]*Scan),
		offers:        make(map[string]*Offer),
		consultations: make(map[string]*Consultation),
		leads:         make(map[string]*Lead),
		reviews:       make(map[string]*Review),
		pricing:       make(map[string]*ClinicPricing),
	}
}

func (db *InMemoryDatabase) CreateUser(user *User) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	db.users[user.ID] = user
	return nil
}

func (db *InMemoryDatabase) GetUserByUsername(username string) (*User, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	for _, user := range db.users {
		if user.Username == username {
			return user, nil
		}
	}
	return nil, errors.New("user not found")
}

func (db *InMemoryDatabase) GetUserByID(id string) (*User, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	user, ok := db.users[id]
	if !ok {
		return nil, errors.New("user not found")
	}
	return user, nil
}

func (db *InMemoryDatabase) CreateScan(scan *Scan) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	db.scans[scan.ID] = scan
	return nil
}

func (db *InMemoryDatabase) GetScanByID(id string) (*Scan, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	scan, ok := db.scans[id]
	if !ok {
		return nil, errors.New("scan not found")
	}
	return scan, nil
}

func (db *InMemoryDatabase) GetScansByPatient(patientID string) ([]*Scan, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	var scans []*Scan
	for _, scan := range db.scans {
		if scan.PatientID == patientID {
			scans = append(scans, scan)
		}
	}
	return scans, nil
}

func (db *InMemoryDatabase) UpdateScan(scan *Scan) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	db.scans[scan.ID] = scan
	return nil
}

func (db *InMemoryDatabase) GetAllAvailableScans() ([]*Scan, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	var scans []*Scan
	for _, scan := range db.scans {
		if scan.Status == "offers_pending" {
			scans = append(scans, scan)
		}
	}
	return scans, nil
}

func (db *InMemoryDatabase) CreateOffer(offer *Offer) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	db.offers[offer.ID] = offer
	return nil
}

// ADD THIS METHOD
func (db *InMemoryDatabase) GetOfferByID(id string) (*Offer, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	offer, ok := db.offers[id]
	if !ok {
		return nil, errors.New("offer not found")
	}
	return offer, nil
}

func (db *InMemoryDatabase) GetOffersByScan(scanID string) ([]*Offer, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	var offers []*Offer
	for _, offer := range db.offers {
		if offer.ScanID == scanID {
			offers = append(offers, offer)
		}
	}
	return offers, nil
}

func (db *InMemoryDatabase) GetOffersByClinic(clinicID string) ([]*Offer, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	var offers []*Offer
	for _, offer := range db.offers {
		if offer.ClinicID == clinicID {
			offers = append(offers, offer)
		}
	}
	return offers, nil
}

func (db *InMemoryDatabase) UpdateOffer(offer *Offer) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	db.offers[offer.ID] = offer
	return nil
}

func (db *InMemoryDatabase) CreateConsultation(consultation *Consultation) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	db.consultations[consultation.ID] = consultation
	return nil
}

func (db *InMemoryDatabase) GetConsultationsByPatient(patientID string) ([]*Consultation, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	var consultations []*Consultation
	for _, consultation := range db.consultations {
		if consultation.PatientID == patientID {
			consultations = append(consultations, consultation)
		}
	}
	return consultations, nil
}

func (db *InMemoryDatabase) UpdateConsultation(consultation *Consultation) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	db.consultations[consultation.ID] = consultation
	return nil
}

func (db *InMemoryDatabase) CreateLead(lead *Lead) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	db.leads[lead.ID] = lead
	return nil
}

func (db *InMemoryDatabase) GetLeadsByClinic(clinicID string) ([]*Lead, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	var leads []*Lead
	for _, lead := range db.leads {
		if lead.ClinicID == clinicID {
			leads = append(leads, lead)
		}
	}
	return leads, nil
}

func (db *InMemoryDatabase) CreateReview(review *Review) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	db.reviews[review.ID] = review
	return nil
}

func (db *InMemoryDatabase) GetReviewsByClinic(clinicID string) ([]*Review, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	var reviews []*Review
	for _, review := range db.reviews {
		if review.ClinicID == clinicID {
			reviews = append(reviews, review)
		}
	}
	return reviews, nil
}

func (db *InMemoryDatabase) SaveClinicPricing(pricing *ClinicPricing) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	db.pricing[pricing.ClinicID] = pricing
	return nil
}

func (db *InMemoryDatabase) GetClinicPricing(clinicID string) (*ClinicPricing, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	pricing, ok := db.pricing[clinicID]
	if !ok {
		return nil, errors.New("pricing not found")
	}
	return pricing, nil
}

func (db *InMemoryDatabase) GetRegionStats(region string) (*RegionStats, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	// Calculate stats from all offers
	var totalPlans int
	var plannedRevenue, actualRevenue float64

	for range db.scans {
		totalPlans++
	}

	for _, offer := range db.offers {
		plannedRevenue += offer.TotalCost
		if offer.Status == "selected" {
			actualRevenue += offer.TotalCost
		}
	}

	// Mock average prices
	return &RegionStats{
		Region:              region,
		TotalPlans:          totalPlans,
		PlannedRevenue:      plannedRevenue,
		ActualRevenue:       actualRevenue,
		AverageImplantPrice: 45000,
		AverageCrownPrice:   25000,
	}, nil
}

func (db *InMemoryDatabase) GetClinicStats(clinicID string) map[string]interface{} {
	db.mu.RLock()
	defer db.mu.RUnlock()

	stats := make(map[string]interface{})

	// Count offers and leads
	offerCount := 0
	leadCount := 0
	totalRevenue := 0.0

	for _, offer := range db.offers {
		if offer.ClinicID == clinicID {
			offerCount++
			if offer.Status == "selected" {
				totalRevenue += offer.TotalCost
			}
		}
	}

	for _, lead := range db.leads {
		if lead.ClinicID == clinicID {
			leadCount++
		}
	}

	stats["total_offers"] = offerCount
	stats["total_leads"] = leadCount
	stats["potential_revenue"] = totalRevenue
	stats["conversion_rate"] = 23.5

	return stats
}
