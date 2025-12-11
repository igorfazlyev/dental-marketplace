package main

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

func handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		renderTemplate(w, "login", nil)
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")

	user, err := db.GetUserByUsername(username)
	if err != nil || user.Password != password {
		renderTemplate(w, "login", map[string]interface{}{
			"Error": "Неверные учетные данные",
		})
		return
	}

	// Set session cookie
	http.SetCookie(w, &http.Cookie{
		Name:    "session",
		Value:   user.ID,
		Path:    "/",
		Expires: time.Now().Add(24 * time.Hour),
	})

	// Redirect based on user type
	switch user.Type {
	case UserTypePatient:
		http.Redirect(w, r, "/patient/dashboard", http.StatusSeeOther)
	case UserTypeClinic:
		http.Redirect(w, r, "/clinic/dashboard", http.StatusSeeOther)
	case UserTypeRegulator:
		http.Redirect(w, r, "/regulator/dashboard", http.StatusSeeOther)
	}
}

func handleLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:    "session",
		Value:   "",
		Path:    "/",
		Expires: time.Unix(0, 0),
	})
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// Patient handlers
func handlePatientDashboard(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value("user").(*User)
	scans, _ := db.GetScansByPatient(user.ID)
	consultations, _ := db.GetConsultationsByPatient(user.ID)

	data := map[string]interface{}{
		"User":          user,
		"Scans":         scans,
		"Consultations": consultations,
	}

	renderTemplate(w, "patient_dashboard", data)
}

func handlePatientScans(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value("user").(*User)
	scans, _ := db.GetScansByPatient(user.ID)

	data := map[string]interface{}{
		"User":  user,
		"Scans": scans,
	}

	renderTemplate(w, "patient_scans", data)
}

func handleScanUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user := r.Context().Value("user").(*User)

	scan := &Scan{
		ID:         fmt.Sprintf("scan-%d", time.Now().Unix()),
		PatientID:  user.ID,
		UploadDate: time.Now(),
		Status:     "analyzed",
		ImagePath:  "/uploads/scan.dcm",
		AIAnalysis: &AIAnalysis{
			ScanID:    fmt.Sprintf("scan-%d", time.Now().Unix()),
			Diagnoses: []string{"Требуется консультация врача", "Возможные проблемы обнаружены"},
			TreatmentPlan: []TreatmentItem{
				{
					ID:          fmt.Sprintf("item-%d-1", time.Now().Unix()),
					Description: "Зубной имплант",
					ToothNumber: "3.6",
					Category:    "implant",
				},
			},
			CreatedAt: time.Now(),
		},
	}

	db.CreateScan(scan)
	http.Redirect(w, r, "/patient/scans", http.StatusSeeOther)
}

func handleTreatmentPlan(w http.ResponseWriter, r *http.Request) {
	scanID := strings.TrimPrefix(r.URL.Path, "/patient/treatment-plan/")
	scan, err := db.GetScanByID(scanID)
	if err != nil {
		http.Error(w, "Scan not found", http.StatusNotFound)
		return
	}

	user := r.Context().Value("user").(*User)

	data := map[string]interface{}{
		"User": user,
		"Scan": scan,
	}

	renderTemplate(w, "patient_treatment_plan", data)
}

func handleCriteria(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value("user").(*User)

	if r.Method == "POST" {
		scanID := r.URL.Query().Get("scan")

		if scanID != "" {
			scan, err := db.GetScanByID(scanID)
			if err == nil {
				scan.Status = "offers_pending"
				db.UpdateScan(scan)
			}

			http.Redirect(w, r, "/patient/dashboard", http.StatusSeeOther)
			return
		}

		http.Redirect(w, r, "/patient/dashboard", http.StatusSeeOther)
		return
	}

	data := map[string]interface{}{
		"User": user,
	}

	renderTemplate(w, "patient_criteria", data)
}

func handleOffers(w http.ResponseWriter, r *http.Request) {
	scanID := strings.TrimPrefix(r.URL.Path, "/patient/offers/")
	offers, _ := db.GetOffersByScan(scanID)
	scan, _ := db.GetScanByID(scanID)
	user := r.Context().Value("user").(*User)

	data := map[string]interface{}{
		"User":   user,
		"Scan":   scan,
		"Offers": offers,
	}

	renderTemplate(w, "patient_offers", data)
}

func handleConsultations(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value("user").(*User)
	consultations, _ := db.GetConsultationsByPatient(user.ID)

	var consultationData []map[string]interface{}
	for _, consultation := range consultations {
		offer, _ := db.GetOfferByID(consultation.OfferID)
		consultationData = append(consultationData, map[string]interface{}{
			"Consultation": consultation,
			"Offer":        offer,
		})
	}

	data := map[string]interface{}{
		"User":          user,
		"Consultations": consultationData,
	}

	renderTemplate(w, "patient_consultations", data)
}

func handleReviews(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value("user").(*User)

	data := map[string]interface{}{
		"User": user,
	}

	renderTemplate(w, "patient_reviews", data)
}

func handleSubmitReview(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user := r.Context().Value("user").(*User)

	review := &Review{
		ID:         fmt.Sprintf("review-%d", time.Now().Unix()),
		PatientID:  user.ID,
		ClinicID:   r.FormValue("clinic_id"),
		DoctorName: r.FormValue("doctor_name"),
		Rating:     5,
		Text:       r.FormValue("text"),
		CreatedAt:  time.Now(),
	}

	db.CreateReview(review)
	http.Redirect(w, r, "/patient/reviews", http.StatusSeeOther)
}

func handleSelectOffer(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user := r.Context().Value("user").(*User)
	offerID := r.FormValue("offer_id")

	offer, err := db.GetOfferByID(offerID)
	if err != nil {
		http.Error(w, "Offer not found", http.StatusNotFound)
		return
	}

	offer.Status = "selected"
	db.UpdateOffer(offer)

	consultation := &Consultation{
		ID:        fmt.Sprintf("consultation-%d", time.Now().Unix()),
		PatientID: user.ID,
		OfferID:   offerID,
		ClinicID:  offer.ClinicID,
		Status:    "awaiting_call",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	db.CreateConsultation(consultation)

	scan, _ := db.GetScanByID(offer.ScanID)

	lead := &Lead{
		ID:            fmt.Sprintf("lead-%d", time.Now().Unix()),
		ClinicID:      offer.ClinicID,
		PatientID:     user.ID,
		PatientName:   user.Name,
		PatientPhone:  "+7-999-123-4567",
		PatientEmail:  user.Email,
		TreatmentPlan: scan.AIAnalysis.TreatmentPlan,
		Status:        "new",
		CreatedAt:     time.Now(),
	}
	db.CreateLead(lead)

	http.Redirect(w, r, "/patient/consultations", http.StatusSeeOther)
}

// Clinic handlers
func handleClinicDashboard(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value("user").(*User)
	stats := db.GetClinicStats(user.ID)

	data := map[string]interface{}{
		"User":  user,
		"Stats": stats,
	}

	renderTemplate(w, "clinic_dashboard", data)
}

func handleIncomingPlans(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value("user").(*User)
	scans, _ := db.GetAllAvailableScans()

	data := map[string]interface{}{
		"User":  user,
		"Scans": scans,
	}

	renderTemplate(w, "clinic_incoming_plans", data)
}

func handleCalculateOffer(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user := r.Context().Value("user").(*User)
	scanID := r.FormValue("scan_id")

	scan, err := db.GetScanByID(scanID)
	if err != nil {
		http.Error(w, "Scan not found", http.StatusNotFound)
		return
	}

	offer := &Offer{
		ID:           fmt.Sprintf("offer-%d", time.Now().Unix()),
		ScanID:       scanID,
		ClinicID:     user.ID,
		ClinicName:   user.Name,
		Rating:       4.5,
		TotalCost:    125000,
		Duration:     "2-3 месяца",
		Details:      "Комплексный план лечения",
		Guarantees:   "Гарантия 2 года",
		PaymentTerms: "Доступна рассрочка",
		Status:       "pending",
		CreatedAt:    time.Now(),
	}

	db.CreateOffer(offer)

	scan.Status = "offers_received"
	db.UpdateScan(scan)

	http.Redirect(w, r, "/clinic/incoming-plans", http.StatusSeeOther)
}

func handleLeads(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value("user").(*User)
	leads, _ := db.GetLeadsByClinic(user.ID)

	data := map[string]interface{}{
		"User":  user,
		"Leads": leads,
	}

	renderTemplate(w, "clinic_leads", data)
}

func handleAnalytics(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value("user").(*User)
	stats := db.GetClinicStats(user.ID)

	data := map[string]interface{}{
		"User":  user,
		"Stats": stats,
	}

	renderTemplate(w, "clinic_analytics", data)
}

func handlePricing(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value("user").(*User)
	pricing, _ := db.GetClinicPricing(user.ID)

	data := map[string]interface{}{
		"User":    user,
		"Pricing": pricing,
	}

	renderTemplate(w, "clinic_pricing", data)
}

// Regulator handlers
func handleRegulatorDashboard(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value("user").(*User)
	stats, _ := db.GetRegionStats("Москва")

	data := map[string]interface{}{
		"User":  user,
		"Stats": stats,
	}

	renderTemplate(w, "regulator_dashboard", data)
}

func handleRegulatorAnalytics(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value("user").(*User)
	stats, _ := db.GetRegionStats("Москва")

	data := map[string]interface{}{
		"User":  user,
		"Stats": stats,
	}

	renderTemplate(w, "regulator_analytics", data)
}
