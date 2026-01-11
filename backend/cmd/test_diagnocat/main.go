package main

import (
	"fmt"

	"dental-marketplace/internal/services"
)

func main() {
	// Load .env file

	fmt.Println("🧪 Testing Diagnocat Authentication\n")

	services.NewDiagnocatService()

	fmt.Println("\n✅ Test complete!")
}
