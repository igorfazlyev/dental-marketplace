package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

func handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		err := templates.ExecuteTemplate(w, "login.html", nil)
		if err != nil {
			log.Printf("Template error: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")

	user, err := db.GetUserByUsername(username)
	if err != nil || user.Password != password {
		templates.ExecuteTemplate(w, "login.html", map[string]interface{}{
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

	err := templates.ExecuteTemplate(w, "patient_dashboard.html", data)
	if err != nil {
		log.Printf("Template error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func handlePatientScans(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value("user").(*User)
	scans, _ := db.GetScansByPatient(user.ID)

	data := map[string]interface{}{
		"User":  user,
		"Scans": scans,
	}

	err := templates.ExecuteTemplate(w, "scans.html", data)
	if err != nil {
		log.Printf("Template error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
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
		Status:     "processing",
		ImagePath:  "/uploads/scan.dcm",
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

	err = templates.ExecuteTemplate(w, "treatment_plan.html", data)
	if err != nil {
		log.Printf("Template error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// func handleCriteria(w http.ResponseWriter, r *http.Request) {
// 	user := r.Context().Value("user").(*User)

// 	if r.Method == "POST" {
// 		// Process the form submission
// 		scanID := r.URL.Query().Get("scan")

// 		if scanID != "" {
// 			// Update scan status to show offers are being requested
// 			scan, err := db.GetScanByID(scanID)
// 			if err == nil {
// 				scan.Status = "offers_received"
// 				db.UpdateScan(scan)
// 			}

// 			// Redirect to the offers page for this scan
// 			http.Redirect(w, r, "/patient/offers/"+scanID, http.StatusSeeOther)
// 			return
// 		}

// 		// If no scan ID, redirect to dashboard
// 		http.Redirect(w, r, "/patient/dashboard", http.StatusSeeOther)
// 		return
// 	}

// 	// GET request - show the form
// 	data := map[string]interface{}{
// 		"User": user,
// 	}

// 	err := templates.ExecuteTemplate(w, "criteria.html", data)
// 	if err != nil {
// 		log.Printf("Template error: %v", err)
// 		http.Error(w, err.Error(), http.StatusInternalServerError)
// 	}
// }

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

	err := templates.ExecuteTemplate(w, "offers.html", data)
	if err != nil {
		log.Printf("Template error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func handleConsultations(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value("user").(*User)
	consultations, _ := db.GetConsultationsByPatient(user.ID)

	// Get offer details for each consultation
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

	err := templates.ExecuteTemplate(w, "consultations.html", data)
	if err != nil {
		log.Printf("Template error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func handleReviews(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value("user").(*User)

	data := map[string]interface{}{
		"User": user,
	}

	err := templates.ExecuteTemplate(w, "reviews.html", data)
	if err != nil {
		log.Printf("Template error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
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

// Clinic handlers
func handleClinicDashboard(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value("user").(*User)
	stats := db.GetClinicStats(user.ID)

	data := map[string]interface{}{
		"User":  user,
		"Stats": stats,
	}

	err := templates.ExecuteTemplate(w, "clinic_dashboard.html", data)
	if err != nil {
		log.Printf("Template error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func handleIncomingPlans(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value("user").(*User)
	scans, _ := db.GetAllAvailableScans()

	data := map[string]interface{}{
		"User":  user,
		"Scans": scans,
	}

	err := templates.ExecuteTemplate(w, "incoming_plans.html", data)
	if err != nil {
		log.Printf("Template error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// func handleCalculateOffer(w http.ResponseWriter, r *http.Request) {
// 	if r.Method != "POST" {
// 		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
// 		return
// 	}

// 	user := r.Context().Value("user").(*User)
// 	scanID := r.FormValue("scan_id")

// 	// Create offer
// 	offer := &Offer{
// 		ID:           fmt.Sprintf("offer-%d", time.Now().Unix()),
// 		ScanID:       scanID,
// 		ClinicID:     user.ID,
// 		ClinicName:   user.Name,
// 		Rating:       4.5,
// 		TotalCost:    125000,
// 		Duration:     "2-3 месяца",
// 		Details:      "Комплексный план лечения",
// 		Guarantees:   "Гарантия 2 года",
// 		PaymentTerms: "Доступна рассрочка",
// 		Status:       "pending",
// 		CreatedAt:    time.Now(),
// 	}

// 	db.CreateOffer(offer)
// 	http.Redirect(w, r, "/clinic/incoming-plans", http.StatusSeeOther)
// }

func handleLeads(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value("user").(*User)
	leads, _ := db.GetLeadsByClinic(user.ID)

	data := map[string]interface{}{
		"User":  user,
		"Leads": leads,
	}

	err := templates.ExecuteTemplate(w, "leads.html", data)
	if err != nil {
		log.Printf("Template error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func handleAnalytics(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value("user").(*User)
	stats := db.GetClinicStats(user.ID)

	data := map[string]interface{}{
		"User":  user,
		"Stats": stats,
	}

	err := templates.ExecuteTemplate(w, "analytics.html", data)
	if err != nil {
		log.Printf("Template error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func handlePricing(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value("user").(*User)
	pricing, _ := db.GetClinicPricing(user.ID)

	data := map[string]interface{}{
		"User":    user,
		"Pricing": pricing,
	}

	err := templates.ExecuteTemplate(w, "pricing.html", data)
	if err != nil {
		log.Printf("Template error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// Regulator handlers
func handleRegulatorDashboard(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value("user").(*User)
	stats, _ := db.GetRegionStats("Москва")

	data := map[string]interface{}{
		"User":  user,
		"Stats": stats,
	}

	err := templates.ExecuteTemplate(w, "regulator_dashboard.html", data)
	if err != nil {
		log.Printf("Template error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func handleRegulatorAnalytics(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value("user").(*User)
	stats, _ := db.GetRegionStats("Москва")

	data := map[string]interface{}{
		"User":  user,
		"Stats": stats,
	}

	err := templates.ExecuteTemplate(w, "detailed_analytics.html", data)
	if err != nil {
		log.Printf("Template error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func handleCriteria(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value("user").(*User)

	if r.Method == "POST" {
		// Process the form submission
		scanID := r.URL.Query().Get("scan")

		if scanID != "" {
			// Update scan status so clinics can see it
			scan, err := db.GetScanByID(scanID)
			if err == nil {
				scan.Status = "offers_pending" // Changed from "offers_received"
				db.UpdateScan(scan)
			}

			// Redirect to dashboard with a message
			http.Redirect(w, r, "/patient/dashboard", http.StatusSeeOther)
			return
		}

		// If no scan ID, redirect to dashboard
		http.Redirect(w, r, "/patient/dashboard", http.StatusSeeOther)
		return
	}

	// GET request - show the form
	data := map[string]interface{}{
		"User": user,
	}

	err := templates.ExecuteTemplate(w, "criteria.html", data)
	if err != nil {
		log.Printf("Template error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func handleCalculateOffer(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user := r.Context().Value("user").(*User)
	scanID := r.FormValue("scan_id")

	// Get the scan to access patient info
	scan, err := db.GetScanByID(scanID)
	if err != nil {
		http.Error(w, "Scan not found", http.StatusNotFound)
		return
	}

	// Create offer
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

	// Update scan status to show offers have been received
	scan.Status = "offers_received"
	db.UpdateScan(scan)

	http.Redirect(w, r, "/clinic/incoming-plans", http.StatusSeeOther)
}

func handleSelectOffer(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	user := r.Context().Value("user").(*User)
	offerID := r.FormValue("offer_id")
	
	// Get the offer
	offer, err := db.GetOfferByID(offerID)
	if err != nil {
		http.Error(w, "Offer not found", http.StatusNotFound)
		return
	}
	
	// Update offer status
	offer.Status = "selected"
	db.UpdateOffer(offer)
	
	// Create consultation for patient
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
	
	// Get scan to access treatment plan
	scan, _ := db.GetScanByID(offer.ScanID)
	
	// Create lead for clinic
	lead := &Lead{
		ID:            fmt.Sprintf("lead-%d", time.Now().Unix()),
		ClinicID:      offer.ClinicID,
		PatientID:     user.ID,
		PatientName:   user.Name,
		PatientPhone:  "+7-999-123-4567", // In real app, get from user profile
		PatientEmail:  user.Email,
		TreatmentPlan: scan.AIAnalysis.TreatmentPlan,
		Status:        "new",
		CreatedAt:     time.Now(),
	}
	db.CreateLead(lead)
	
	http.Redirect(w, r, "/patient/consultations", http.StatusSeeOther)
}
