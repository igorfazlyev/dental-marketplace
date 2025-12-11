package main

import (
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"time"
)

var (
	db        Database
	templates map[string]*template.Template
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
	
	// Initialize template cache
	templates = make(map[string]*template.Template)
	
	// Define template files with their keys
	templateFiles := map[string]string{
		"login":                    "templates/login.html",
		"patient_dashboard":        "templates/patient/patient_dashboard.html",
		"patient_scans":            "templates/patient/scans.html",
		"patient_treatment_plan":   "templates/patient/treatment_plan.html",
		"patient_criteria":         "templates/patient/criteria.html",
		"patient_offers":           "templates/patient/offers.html",
		"patient_consultations":    "templates/patient/consultations.html",
		"patient_reviews":          "templates/patient/reviews.html",
		"clinic_dashboard":         "templates/clinic/clinic_dashboard.html",
		"clinic_incoming_plans":    "templates/clinic/incoming_plans.html",
		"clinic_leads":             "templates/clinic/leads.html",
		"clinic_analytics":         "templates/clinic/analytics.html",
		"clinic_pricing":           "templates/clinic/pricing.html",
		"regulator_dashboard":      "templates/regulator/regulator_dashboard.html",
		"regulator_analytics":      "templates/regulator/detailed_analytics.html",
	}
	
	// Parse and cache each template
	for key, path := range templateFiles {
		tmpl, err := template.New(filepath.Base(path)).Funcs(funcMap).ParseFiles(path)
		if err != nil {
			log.Fatalf("Error parsing template %s: %v", path, err)
		}
		templates[key] = tmpl
		log.Printf("Loaded template: %s -> %s", key, path)
	}
	
	log.Println("All templates loaded and cached!")
	
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
	
	// Wrap with logging
	handler := loggingMiddleware(mux)
	
	// Start server
	port := ":8080"
	log.Printf("Starting server on http://localhost%s", port)
	log.Printf("Demo credentials:")
	log.Printf("  Patient: demo / demo")
	log.Printf("  Clinic: clinic / clinic")
	log.Printf("  Regulator: regulator / regulator")
	log.Fatal(http.ListenAndServe(port, handler))
}

func handleHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// Logging middleware to track slow requests
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		duration := time.Since(start)
		if duration > 100*time.Millisecond {
			log.Printf("SLOW: %s %s took %v", r.Method, r.URL.Path, duration)
		}
	})
}

// Helper function to render templates
func renderTemplate(w http.ResponseWriter, name string, data interface{}) {
	tmpl, ok := templates[name]
	if !ok {
		log.Printf("Template not found: %s", name)
		http.Error(w, "Template not found", http.StatusInternalServerError)
		return
	}
	
	err := tmpl.Execute(w, data)
	if err != nil {
		log.Printf("Template execution error for %s: %v", name, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
