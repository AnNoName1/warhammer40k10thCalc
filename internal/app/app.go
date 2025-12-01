package app

import (
	"log"
	"net/http"
	"os"

	httpSwagger "github.com/swaggo/http-swagger"

	_ "github.com/AnNoName1/warhammer40k10thCalc/docs"

	handler "github.com/AnNoName1/warhammer40k10thCalc/pkg/handler"
)

// Run initializes the application and starts the HTTP server.
// It returns an error if the server fails to start or encounters an issue.
func Run() error {
	http.HandleFunc("/api/damage/calculate", handler.CalculateDamageHandler)

	// This serves the documentation at /swagger/index.html
	http.HandleFunc("/swagger/", httpSwagger.WrapHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on http://localhost:%s\n", port)
	log.Printf("Swagger UI available at http://localhost:%s/swagger/index.html\n", port)
	// 3. Start the server
	return http.ListenAndServe(":"+port, nil)
}
