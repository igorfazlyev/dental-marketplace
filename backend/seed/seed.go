package seed

import (
	"dental-marketplace/models"
	"log"

	"gorm.io/gorm"
)

func SeedData(db *gorm.DB) {
	log.Println("Starting database seeding...")

	// Create sample patient
	patient := models.User{
		Email:     "patient@example.com",
		FirstName: "John",
		LastName:  "Doe",
		Role:      models.RolePatient,
		Phone:     "+1234567890",
	}
	patient.SetPassword("password123")

	if err := db.FirstOrCreate(&patient, models.User{Email: patient.Email}).Error; err != nil {
		log.Printf("Failed to create patient: %v", err)
	} else {
		log.Printf("✓ Patient created: %s", patient.Email)
	}

	// Create sample clinic
	clinic := models.User{
		Email:     "clinic@example.com",
		FirstName: "Downtown",
		LastName:  "Dental Clinic",
		Role:      models.RoleClinic,
		Phone:     "+1987654321",
	}
	clinic.SetPassword("password123")

	if err := db.FirstOrCreate(&clinic, models.User{Email: clinic.Email}).Error; err != nil {
		log.Printf("Failed to create clinic: %v", err)
	} else {
		log.Printf("✓ Clinic created: %s", clinic.Email)
	}

	// Create sample government user
	gov := models.User{
		Email:     "gov@example.com",
		FirstName: "Health",
		LastName:  "Inspector",
		Role:      models.RoleGovernment,
		Phone:     "+1555555555",
	}
	gov.SetPassword("password123")

	if err := db.FirstOrCreate(&gov, models.User{Email: gov.Email}).Error; err != nil {
		log.Printf("Failed to create government user: %v", err)
	} else {
		log.Printf("✓ Government user created: %s", gov.Email)
	}

	log.Println("Database seeding completed!")
	log.Println("\n=== Sample Login Credentials ===")
	log.Println("Patient:     patient@example.com / password123")
	log.Println("Clinic:      clinic@example.com / password123")
	log.Println("Government:  gov@example.com / password123")
	log.Println("================================\n")
}
