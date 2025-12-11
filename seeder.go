package main

import (
	"fmt"
	"math/rand"
	"time"
)

func SeedDatabase(db Database) {
	// Seed random number generator with current time
	rand.Seed(time.Now().UnixNano())

	// Create demo users
	users := []*User{
		{
			ID:       "patient-1",
			Username: "demo",
			Password: "demo",
			Type:     UserTypePatient,
			Name:     "Иван Петров",
			Email:    "ivan@example.com",
		},
		{
			ID:       "clinic-1",
			Username: "clinic",
			Password: "clinic",
			Type:     UserTypeClinic,
			Name:     "Клиника СмайлДент",
			Email:    "info@smiledent.com",
		},
		{
			ID:       "clinic-2",
			Username: "clinic2",
			Password: "clinic2",
			Type:     UserTypeClinic,
			Name:     "Медицинский центр ДентПро",
			Email:    "info@dentpro.com",
		},
		{
			ID:       "regulator-1",
			Username: "regulator",
			Password: "regulator",
			Type:     UserTypeRegulator,
			Name:     "Министерство здравоохранения",
			Email:    "ministry@health.gov",
		},
	}

	for _, user := range users {
		db.CreateUser(user)
	}

	// Create scans for the last 60 days (about 2 months)
	now := time.Now()
	diagnoses := [][]string{
		{"Кариес зуба 1.6", "Пульпит зуба 2.5"},
		{"Отсутствует зуб 3.6", "Кариес зуба 4.7"},
		{"Пародонтит", "Множественный кариес"},
	}

	for i := 0; i < 10; i++ {
		// Generate date within last 60 days
		daysAgo := rand.Intn(60)
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
						Description: "Зубной имплант",
						ToothNumber: "1.6",
						Category:    "implant",
					},
					{
						ID:          fmt.Sprintf("item-%d-2", i),
						Description: "Керамическая коронка",
						ToothNumber: "2.5",
						Category:    "crown",
					},
				},
				CreatedAt: uploadDate.Add(2 * time.Hour),
			},
		}
		db.CreateScan(scan)

		// Create offers for some scans (only for older scans to be realistic)
		if i%3 != 0 && daysAgo > 5 {
			for j := 1; j <= 2; j++ {
				clinicID := fmt.Sprintf("clinic-%d", j)
				offerDate := uploadDate.Add(time.Duration(24+rand.Intn(48)) * time.Hour)

				offer := &Offer{
					ID:           fmt.Sprintf("offer-%d-%d", i, j),
					ScanID:       scanID,
					ClinicID:     clinicID,
					ClinicName:   []string{"Клиника СмайлДент", "Медицинский центр ДентПро"}[j-1],
					Rating:       4.0 + float64(j)*0.5,
					TotalCost:    100000 + float64(j*25000) + float64(rand.Intn(50000)),
					Duration:     fmt.Sprintf("%d-%d месяца", 2+j, 3+j),
					Details:      "Комплексный план лечения с современными материалами",
					Guarantees:   fmt.Sprintf("Гарантия %d года", j+1),
					PaymentTerms: "Доступна рассрочка на 12 месяцев",
					Status:       []string{"pending", "selected", "rejected"}[rand.Intn(3)],
					CreatedAt:    offerDate,
				}
				db.CreateOffer(offer)

				// Create consultation if offer was selected
				if offer.Status == "selected" {
					consultationDate := offerDate.Add(time.Duration(2+rand.Intn(6)) * time.Hour)
					consultation := &Consultation{
						ID:        fmt.Sprintf("consultation-%d-%d", i, j),
						PatientID: "patient-1",
						OfferID:   offer.ID,
						ClinicID:  clinicID,
						Status:    []string{"awaiting_call", "scheduled", "completed"}[rand.Intn(3)],
						CreatedAt: consultationDate,
						UpdatedAt: now,
					}
					db.CreateConsultation(consultation)

					// Create lead for clinic
					lead := &Lead{
						ID:            fmt.Sprintf("lead-%d-%d", i, j),
						ClinicID:      clinicID,
						PatientID:     "patient-1",
						PatientName:   "Иван Петров",
						PatientPhone:  "+7-999-123-4567",
						PatientEmail:  "ivan@example.com",
						TreatmentPlan: scan.AIAnalysis.TreatmentPlan,
						Status:        []string{"new", "contacted", "scheduled", "treatment_started"}[rand.Intn(4)],
						CreatedAt:     consultationDate.Add(1 * time.Hour),
					}
					db.CreateLead(lead)
				}
			}
		}
	}

	// Create reviews for the last 30 days
	for i := 0; i < 5; i++ {
		reviewDate := now.AddDate(0, 0, -rand.Intn(30))
		review := &Review{
			ID:         fmt.Sprintf("review-%d", i+1),
			PatientID:  "patient-1",
			ClinicID:   fmt.Sprintf("clinic-%d", (i%2)+1),
			DoctorName: []string{"Доктор Иванов", "Доктор Петрова", "Доктор Сидоров"}[i%3],
			Rating:     4 + (i % 2),
			Text:       "Отличный сервис, профессиональный персонал, очень рекомендую!",
			CreatedAt:  reviewDate,
		}
		db.CreateReview(review)
	}

	// Create pricing for clinics
	pricing1 := &ClinicPricing{
		ClinicID: "clinic-1",
		Items: map[string]float64{
			"implant":  45000,
			"crown":    25000,
			"filling":  5000,
			"cleaning": 3000,
		},
	}
	db.SaveClinicPricing(pricing1)

	pricing2 := &ClinicPricing{
		ClinicID: "clinic-2",
		Items: map[string]float64{
			"implant":  55000,
			"crown":    30000,
			"filling":  6000,
			"cleaning": 3500,
		},
	}
	db.SaveClinicPricing(pricing2)
}
