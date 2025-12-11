package main

import (
	"fmt"
	"math/rand"
	"time"
)

func SeedDatabase(db Database) {
	// Create demo users
	users := []*User{
		{
			ID:       "patient-1",
			Username: "demo",
			Password: "demo",
			Type:     UserTypePatient,
			Name:     "Ivan Petrov",
			Email:    "ivan@example.com",
		},
		{
			ID:       "clinic-1",
			Username: "clinic",
			Password: "clinic",
			Type:     UserTypeClinic,
			Name:     "SmileDent Clinic",
			Email:    "info@smiledent.com",
		},
		{
			ID:       "clinic-2",
			Username: "clinic2",
			Password: "clinic2",
			Type:     UserTypeClinic,
			Name:     "DentPro Medical Center",
			Email:    "info@dentpro.com",
		},
		{
			ID:       "regulator-1",
			Username: "regulator",
			Password: "regulator",
			Type:     UserTypeRegulator,
			Name:     "Health Ministry",
			Email:    "ministry@health.gov",
		},
	}
	
	for _, user := range users {
		db.CreateUser(user)
	}
	
	// Create scans for the last 3 months
	now := time.Now()
	diagnoses := [][]string{
		{"Caries on tooth 1.6", "Pulpitis on tooth 2.5"},
		{"Missing tooth 3.6", "Caries on tooth 4.7"},
		{"Periodontitis", "Multiple caries"},
	}
	
	for i := 0; i < 10; i++ {
		daysAgo := rand.Intn(90)
		uploadDate := now.AddDate(0, 0, -daysAgo)
		
		scanID := fmt.Sprintf("scan-%d", i+1)
		scan := &Scan{
			ID:         scanID,
			PatientID:  "patient-1",
			UploadDate: uploadDate,
			Status:     []string{"analyzed", "offers_pending", "offers_received"}[i%3],
			ImagePath:  fmt.Sprintf("/uploads/scan-%d.dcm", i+1),
			AIAnalysis: &AIAnalysis{
				ScanID:    scanID,
				Diagnoses: diagnoses[i%3],
				TreatmentPlan: []TreatmentItem{
					{
						ID:          fmt.Sprintf("item-%d-1", i),
						Description: "Dental implant",
						ToothNumber: "1.6",
						Category:    "implant",
					},
					{
						ID:          fmt.Sprintf("item-%d-2", i),
						Description: "Ceramic crown",
						ToothNumber: "2.5",
						Category:    "crown",
					},
				},
				CreatedAt: uploadDate.Add(2 * time.Hour),
			},
		}
		db.CreateScan(scan)
		
		// Create offers for some scans
		if i%3 != 0 {
			for j := 1; j <= 2; j++ {
				clinicID := fmt.Sprintf("clinic-%d", j)
				offer := &Offer{
					ID:           fmt.Sprintf("offer-%d-%d", i, j),
					ScanID:       scanID,
					ClinicID:     clinicID,
					ClinicName:   []string{"SmileDent Clinic", "DentPro Medical Center"}[j-1],
					Rating:       4.0 + float64(j)*0.5,
					TotalCost:    100000 + float64(j*25000) + float64(rand.Intn(50000)),
					Duration:     fmt.Sprintf("%d-%d months", 2+j, 3+j),
					Details:      "Comprehensive treatment plan with modern materials",
					Guarantees:   fmt.Sprintf("%d years warranty", j+1),
					PaymentTerms: "Installment available for 12 months",
					Status:       []string{"pending", "selected", "rejected"}[rand.Intn(3)],
					CreatedAt:    uploadDate.Add(24 * time.Hour),
				}
				db.CreateOffer(offer)
				
				// Create consultation if offer was selected
				if offer.Status == "selected" {
					consultation := &Consultation{
						ID:        fmt.Sprintf("consultation-%d-%d", i, j),
						PatientID: "patient-1",
						OfferID:   offer.ID,
						ClinicID:  clinicID,
						Status:    []string{"awaiting_call", "scheduled", "completed"}[rand.Intn(3)],
						CreatedAt: offer.CreatedAt.Add(2 * time.Hour),
						UpdatedAt: time.Now(),
					}
					db.CreateConsultation(consultation)
					
					// Create lead for clinic
					lead := &Lead{
						ID:           fmt.Sprintf("lead-%d-%d", i, j),
						ClinicID:     clinicID,
						PatientID:    "patient-1",
						PatientName:  "Ivan Petrov",
						PatientPhone: "+7-999-123-4567",
						PatientEmail: "ivan@example.com",
						TreatmentPlan: scan.AIAnalysis.TreatmentPlan,
						Status:       []string{"new", "contacted", "scheduled", "treatment_started"}[rand.Intn(4)],
						CreatedAt:    offer.CreatedAt.Add(3 * time.Hour),
					}
					db.CreateLead(lead)
				}
			}
		}
	}
	
	// Create reviews
	for i := 0; i < 5; i++ {
		review := &Review{
			ID:         fmt.Sprintf("review-%d", i+1),
			PatientID:  "patient-1",
			ClinicID:   fmt.Sprintf("clinic-%d", (i%2)+1),
			DoctorName: []string{"Dr. Ivanov", "Dr. Petrova", "Dr. Sidorov"}[i%3],
			Rating:     4 + (i % 2),
			Text:       "Great service, professional staff, highly recommended!",
			CreatedAt:  now.AddDate(0, 0, -rand.Intn(60)),
		}
		db.CreateReview(review)
	}
	
	// Create pricing for clinics
	pricing1 := &ClinicPricing{
		ClinicID: "clinic-1",
		Items: map[string]float64{
			"implant": 45000,
			"crown":   25000,
			"filling": 5000,
			"cleaning": 3000,
		},
	}
	db.SaveClinicPricing(pricing1)
	
	pricing2 := &ClinicPricing{
		ClinicID: "clinic-2",
		Items: map[string]float64{
			"implant": 55000,
			"crown":   30000,
			"filling": 6000,
			"cleaning": 3500,
		},
	}
	db.SaveClinicPricing(pricing2)
}
