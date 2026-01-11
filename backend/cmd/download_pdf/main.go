package main

import (
	"flag"
	"fmt"
	"log"

	"dental-marketplace/internal/services"

	"github.com/joho/godotenv"
)

func main() {
	reportID := flag.String("id", "", "Report ID (uid or id_v3) (required)")
	out := flag.String("out", "report.pdf", "Output PDF path")
	flag.Parse()

	if *reportID == "" {
		log.Fatal("missing -id")
	}

	_ = godotenv.Load()
	svc := services.NewDiagnocatService()

	fmt.Printf("⬇️ Downloading PDF for report %s -> %s\n", *reportID, *out)

	if err := svc.DownloadReportPDF(*reportID, *out); err != nil {
		log.Fatalf("download failed: %v", err)
	}

	fmt.Println("✅ PDF downloaded")
}
