package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"dental-marketplace/internal/services"

	"github.com/joho/godotenv"
)

func main() {
	reportID := flag.String("id", "", "Report/Analysis ID (required)")
	outFile := flag.String("out", "", "Write JSON to file (optional)")
	pretty := flag.Bool("pretty", true, "Pretty-print JSON")
	flag.Parse()

	if *reportID == "" {
		fmt.Println("❌ Error: missing -id")
		fmt.Println("Example:")
		fmt.Println("  go run ./cmd/export_report -id=00000006-8d67-026e-b025-4c38d0576770 -out=report.json")
		os.Exit(1)
	}

	_ = godotenv.Load()

	svc := services.NewDiagnocatService()

	export, err := svc.ExportReport(*reportID)
	if err != nil {
		log.Fatalf("❌ Export failed: %v", err)
	}

	var b []byte
	if *pretty {
		b, err = json.MarshalIndent(export, "", "  ")
	} else {
		b, err = json.Marshal(export)
	}
	if err != nil {
		log.Fatalf("❌ JSON marshal failed: %v", err)
	}

	if *outFile != "" {
		if err := os.WriteFile(*outFile, b, 0644); err != nil {
			log.Fatalf("❌ write file failed: %v", err)
		}
		fmt.Printf("✅ Wrote export JSON to %s\n", *outFile)
		return
	}

	// stdout
	fmt.Println(string(b))
}
