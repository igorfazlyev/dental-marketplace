package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"dental-marketplace/internal/services"

	"github.com/joho/godotenv"
)

func main() {
	reportID := flag.String("id", "", "Analysis/Report ID (required)")
	flag.Parse()

	if *reportID == "" {
		fmt.Println("❌ Error: Missing report ID")
		flag.Usage()
		fmt.Println("\nExample:")
		fmt.Println("  go run cmd/check_status/main.go -id=report-abc123")
		os.Exit(1)
	}

	godotenv.Load()

	service := services.NewDiagnocatService()

	fmt.Printf("🔍 Checking analysis: %s\n\n", *reportID)

	report, err := service.GetAnalysisStatus(*reportID)
	if err != nil {
		log.Fatalf("❌ Failed: %v", err)
	}

	fmt.Println("📊 Analysis Status:")
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("  ID: %s\n", report.ID)
	fmt.Printf("  Status: %s\n", report.Status)
	fmt.Printf("  Complete: %v\n", report.Complete)

	if len(report.Error) > 0 && string(report.Error) != "null" {
		fmt.Printf("  ❌ Error: %s\n", string(report.Error))
	}

	if report.Complete {
		fmt.Println("\n🎉 Analysis Complete!")
		if report.WebpageUrl != "" {
			fmt.Printf("  🌐 View Report: %s\n", report.WebpageUrl)
		}
		if report.PDFUrl != "" {
			fmt.Printf("  📄 Download PDF: %s\n", report.PDFUrl)
		}
	} else {
		fmt.Println("\n⏳ Still processing...")
	}
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
}
