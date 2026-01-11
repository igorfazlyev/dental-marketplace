package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"dental-marketplace/internal/services"

	"github.com/joho/godotenv"
)

type CreatePatientRequest struct {
	NamePart1   string `json:"name_part1"` // First name
	NamePart2   string `json:"name_part2"` // Last name
	Gender      string `json:"gender,omitempty"`
	DateOfBirth string `json:"date_of_birth,omitempty"` // Format: YYYY-MM-DD
	PatientID   string `json:"patient_id,omitempty"`    // External Patient ID (optional)
}

type PatientResponse struct {
	UID         string `json:"uid"`
	NamePart1   string `json:"name_part1"`
	NamePart2   string `json:"name_part2"`
	Gender      string `json:"gender"`
	DateOfBirth string `json:"date_of_birth"`
	PatientID   string `json:"patient_id"`
	CreatedAt   string `json:"created_at"`
}

func main() {
	// Load .env
	if err := godotenv.Load(); err != nil {
		log.Println("⚠️  No .env file found")
	}

	fmt.Println("🧪 Creating Test Patient in Diagnocat\n")

	// Get auth headers
	service := services.NewDiagnocatService()
	headers, err := service.GetHeaders()
	if err != nil {
		log.Fatal("❌ Failed to get auth headers:", err)
	}

	// Create patient request with correct field names
	patientReq := CreatePatientRequest{
		NamePart1:   "TestPatient",
		NamePart2:   "DiagnocatTest",
		Gender:      "male",
		DateOfBirth: "1990-01-01",
		PatientID:   "test-patient-001", // External ID (optional)
	}

	body, _ := json.MarshalIndent(patientReq, "", "  ")

	fmt.Println("📤 Request body:")
	fmt.Println(string(body))
	fmt.Println()

	baseURL := os.Getenv("DIAGNOCAT_API_URL")
	url := baseURL + "/v2/patients"

	fmt.Printf("📍 POST %s\n\n", url)

	req, _ := http.NewRequest("POST", url, bytes.NewReader(body))
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal("❌ Failed to create patient:", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, _ := io.ReadAll(resp.Body)

	fmt.Printf("📥 Response Status: %d\n", resp.StatusCode)
	fmt.Println("📥 Response Body:")
	fmt.Println(string(respBody))
	fmt.Println()

	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		log.Fatalf("❌ Create patient failed with status %d\n%s", resp.StatusCode, string(respBody))
	}

	var patient PatientResponse
	if err := json.Unmarshal(respBody, &patient); err != nil {
		log.Fatal("❌ Failed to decode response:", err)
	}

	fmt.Printf("✅ Patient created successfully!\n")
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("👤 Patient UID: %s\n", patient.UID)
	fmt.Printf("📝 Name: %s %s\n", patient.NamePart1, patient.NamePart2)
	if patient.Gender != "" {
		fmt.Printf("⚧️  Gender: %s\n", patient.Gender)
	}
	if patient.DateOfBirth != "" {
		fmt.Printf("🎂 Date of Birth: %s\n", patient.DateOfBirth)
	}
	if patient.PatientID != "" {
		fmt.Printf("🔢 External ID: %s\n", patient.PatientID)
	}
	fmt.Printf("📅 Created: %s\n", patient.CreatedAt)
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Println("\n💡 Use this Patient UID to test file upload:")
	fmt.Printf("   go run cmd/test_upload/main.go -file=test_data/sample.dcm -patient=%s\n", patient.UID)
}
