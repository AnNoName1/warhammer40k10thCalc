package main

import (
	"log"

	"github.com/AnNoName1/warhammer40k10thCalc/internal/app"
)

//	@title			Warhammer 40k 10th Calc API
//	@version		1.0
//	@description	API for calculating damage statistics based on 10th Edition rules.
//	@host			localhost:8080
//	@BasePath		/api

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile) // Optional: Better logging format

	if err := app.Run(); err != nil {
		// Only handles the fatal error case, keeping main simple.
		log.Fatalf("Application shutdown: %v", err)
	}
}
