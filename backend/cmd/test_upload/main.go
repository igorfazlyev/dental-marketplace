package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"dental-marketplace/internal/services"

	"github.com/joho/godotenv"
)

func main() {
	// Define command-line flags
	filePath := flag.String("file", "", "Path to DICOM file (required)")
	patientUID := flag.String("patient", "", "Patient UID in Diagnocat (required)")
	studyType := flag.String("type", "CBCT", "Study type (CBCT, PANORAMA, FMX, STL)")
	waitTime := flag.Int("wait", 5, "Seconds to wait before checking status")
	checkStatus := flag.Bool("status", true, "Check analysis status after upload")

	// Parse flags
	flag.Parse()

	// Validate required flags
	if *filePath == "" || *patientUID == "" {
		fmt.Println("❌ Error: Missing required parameters\n")
		flag.Usage()
		fmt.Println("\nExample usage:")
		fmt.Println("  go run cmd/test_upload/main.go -file=test_data/sample.dcm -patient=dc1234567890abcdef")
		fmt.Println("  go run cmd/test_upload/main.go -file=./scan.dcm -patient=patient-123 -type=PANORAMA")
		os.Exit(1)
	}

	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("⚠️  No .env file found, using system environment variables")
	}

	fmt.Println("🧪 Testing Diagnocat File Upload\n")
	fmt.Println("Parameters:")
	fmt.Printf("  📁 File: %s\n", *filePath)
	fmt.Printf("  👤 Patient UID: %s\n", *patientUID)
	fmt.Printf("  📋 Study Type: %s\n", *studyType)
	fmt.Println()

	// Check if file exists
	fileInfo, err := os.Stat(*filePath)
	if os.IsNotExist(err) {
		log.Fatalf("❌ File not found: %s", *filePath)
	}
	fmt.Printf("✅ File found: %s (%.2f MB)\n\n", fileInfo.Name(), float64(fileInfo.Size())/1024/1024)

	// Initialize service
	service := services.NewDiagnocatService()

	// Upload the study
	fmt.Println("🚀 Starting upload...\n")
	analysis, err := service.UploadStudy(*patientUID, *filePath)
	if err != nil {
		log.Fatalf("❌ Upload failed: %v", err)
	}

	fmt.Printf("\n🎉 Upload successful!\n")
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	reportID := analysis.UID
	if reportID == "" {
		reportID = analysis.IDV3
	}

	fmt.Printf("📊 Report UID: %s\n", analysis.UID)
	fmt.Printf("📊 Report IDV3: %s\n", analysis.IDV3)
	fmt.Printf("📊 Status: %s\n", analysis.Status)
	fmt.Printf("📌 Use for status checks (-id): %s\n", reportID)

	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

	// Optionally check status
	if *checkStatus {
		fmt.Printf("⏳ Waiting %d seconds before checking status...\n", *waitTime)
		time.Sleep(time.Duration(*waitTime) * time.Second)

		report, err := service.GetAnalysisStatus(reportID)
		if err != nil {
			log.Printf("⚠️  Failed to get status: %v\n", err)
		} else {
			fmt.Println("\n📊 Analysis Status:")
			fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
			fmt.Printf("  Status: %s\n", report.Status)
			fmt.Printf("  Complete: %v\n", report.Complete)

			if len(report.Error) > 0 && string(report.Error) != "null" {
				fmt.Printf("  ❌ Error: %s\n", string(report.Error))
			}

			if report.Complete {
				if report.WebpageUrl != "" {
					fmt.Printf("  🌐 Webpage: %s\n", report.WebpageUrl)
				}
				if report.PDFUrl != "" {
					fmt.Printf("  📄 PDF: %s\n", report.PDFUrl)
				}
				if report.PreviewUrl != "" {
					fmt.Printf("  👁️  Preview: %s\n", report.PreviewUrl)
				}
			} else {
				fmt.Printf("  ⏳ Analysis is still processing...\n")
				fmt.Printf("  💡 Check status later with: -patient=%s\n", *patientUID)
			}
			fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
		}
	}

	fmt.Println("\n✅ Test complete!")
}
