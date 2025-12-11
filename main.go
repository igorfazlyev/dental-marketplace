package main

import (
	"html/template"
	"log"
	"net/http"
)

var (
	db        Database
	templates *template.Template
)

func main() {
	// Initialize in-memory database
	db = NewInMemoryDatabase()

	// Seed database with demo data
	log.Println("Seeding database with demo data...")
	SeedDatabase(db)
	log.Println("Database seeded successfully!")

	// Create template with custom functions
	funcMap := template.FuncMap{
		"divf": func(a, b float64) float64 {
			if b == 0 {
				return 0
			}
			return a / b
		},
		"mulf": func(a, b float64) float64 {
			return a * b
		},
	}

	// Initialize templates
	templates = template.New("").Funcs(funcMap)

	// Parse each template file individually
	templateFiles := []string{
		"templates/login.html",
		"templates/patient/patient_dashboard.html",
		"templates/patient/scans.html",
		"templates/patient/treatment_plan.html",
		"templates/patient/criteria.html",
		"templates/patient/offers.html",
		"templates/patient/consultations.html",
		"templates/patient/reviews.html",
		"templates/clinic/clinic_dashboard.html",
		"templates/clinic/incoming_plans.html",
		"templates/clinic/leads.html",
		"templates/clinic/analytics.html",
		"templates/clinic/pricing.html",
		"templates/regulator/regulator_dashboard.html",
		"templates/regulator/detailed_analytics.html",
	}

	for _, tmpl := range templateFiles {
		_, err := templates.ParseFiles(tmpl)
		if err != nil {
			log.Fatalf("Error parsing template %s: %v", tmpl, err)
		}
		log.Printf("Loaded template: %s", tmpl)
	}

	// Setup routes
	mux := http.NewServeMux()

	// Public routes
	mux.HandleFunc("/", handleHome)
	mux.HandleFunc("/login", handleLogin)
	mux.HandleFunc("/logout", handleLogout)

	// Patient routes
	mux.HandleFunc("/patient/dashboard", authMiddleware("patient", handlePatientDashboard))
	mux.HandleFunc("/patient/scans", authMiddleware("patient", handlePatientScans))
	mux.HandleFunc("/patient/scans/upload", authMiddleware("patient", handleScanUpload))
	mux.HandleFunc("/patient/treatment-plan/", authMiddleware("patient", handleTreatmentPlan))
	mux.HandleFunc("/patient/criteria", authMiddleware("patient", handleCriteria))
	mux.HandleFunc("/patient/offers/", authMiddleware("patient", handleOffers))
	mux.HandleFunc("/patient/consultations", authMiddleware("patient", handleConsultations))
	mux.HandleFunc("/patient/reviews", authMiddleware("patient", handleReviews))
	mux.HandleFunc("/patient/reviews/submit", authMiddleware("patient", handleSubmitReview))
	mux.HandleFunc("/patient/select-offer", authMiddleware("patient", handleSelectOffer))

	// Clinic routes
	mux.HandleFunc("/clinic/dashboard", authMiddleware("clinic", handleClinicDashboard))
	mux.HandleFunc("/clinic/incoming-plans", authMiddleware("clinic", handleIncomingPlans))
	mux.HandleFunc("/clinic/calculate-offer", authMiddleware("clinic", handleCalculateOffer))
	mux.HandleFunc("/clinic/leads", authMiddleware("clinic", handleLeads))
	mux.HandleFunc("/clinic/analytics", authMiddleware("clinic", handleAnalytics))
	mux.HandleFunc("/clinic/pricing", authMiddleware("clinic", handlePricing))

	// Regulator routes
	mux.HandleFunc("/regulator/dashboard", authMiddleware("regulator", handleRegulatorDashboard))
	mux.HandleFunc("/regulator/analytics", authMiddleware("regulator", handleRegulatorAnalytics))

	// Start server
	port := ":8080"
	log.Printf("Starting server on http://localhost%s", port)
	log.Printf("Demo credentials:")
	log.Printf("  Patient: demo / demo")
	log.Printf("  Clinic: clinic / clinic")
	log.Printf("  Regulator: regulator / regulator")
	log.Fatal(http.ListenAndServe(port, mux))
}

func handleHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}
