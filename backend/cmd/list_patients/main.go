package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"dental-marketplace/internal/services"

	"github.com/joho/godotenv"
)

type Patient struct {
	UID  string `json:"uid"`
	Name struct {
		Given  string `json:"given"`
		Family string `json:"family"`
	} `json:"name"`
	Gender      string `json:"gender"`
	DateOfBirth string `json:"date_of_birth"`
	ExternalID  string `json:"external_id"`
}

func main() {
	// Load .env
	if err := godotenv.Load(); err != nil {
		log.Println("⚠️  No .env file found")
	}

	fmt.Println("🧪 Listing Diagnocat Patients\n")

	// Get auth headers
	service := services.NewDiagnocatService()
	headers, err := service.GetHeaders()
	if err != nil {
		log.Fatal("❌ Failed to get auth headers:", err)
	}

	baseURL := os.Getenv("DIAGNOCAT_API_URL")
	url := baseURL + "/v2/patients?limit=50"

	fmt.Printf("📍 GET %s\n\n", url)

	req, _ := http.NewRequest("GET", url, nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal("❌ Failed to list patients:", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		fmt.Printf("❌ Failed with status %d\n", resp.StatusCode)
		fmt.Println(string(respBody))
		os.Exit(1)
	}

	// The API returns an array directly, not an object
	var patients []Patient
	if err := json.Unmarshal(respBody, &patients); err != nil {
		log.Fatal("❌ Failed to decode response:", err)
	}

	fmt.Printf("📊 Found %d patient(s)\n", len(patients))
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	if len(patients) == 0 {
		fmt.Println("⚠️  No patients found in your account")
		fmt.Println("\n💡 You can create a patient through:")
		fmt.Println("   1. The Diagnocat web interface at https://app2.diagnocat.ru")
		fmt.Println("   2. Or try: go run cmd/create_patient/main.go")
	} else {
		for i, patient := range patients {
			fmt.Printf("%d. 👤 %s %s\n", i+1, patient.Name.Given, patient.Name.Family)
			fmt.Printf("   UID: %s\n", patient.UID)
			if patient.ExternalID != "" {
				fmt.Printf("   External ID: %s\n", patient.ExternalID)
			}
			if patient.Gender != "" {
				fmt.Printf("   Gender: %s\n", patient.Gender)
			}
			if patient.DateOfBirth != "" {
				fmt.Printf("   DOB: %s\n", patient.DateOfBirth)
			}
			fmt.Println()
		}

		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println("\n💡 To test file upload with the first patient:")
		fmt.Printf("   go run cmd/test_upload/main.go -file=test_data/sample.dcm -patient=%s\n",
			patients[0].UID)
	}
}
