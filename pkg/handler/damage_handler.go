// Copyright (c) 2025 Olbutov Aleksandr
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package handler

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

	calculator "github.com/AnNoName1/warhammer40k10thCalc/internal/calculator"
	middleware "github.com/AnNoName1/warhammer40k10thCalc/internal/middleware"
	damagerequest "github.com/AnNoName1/warhammer40k10thCalc/pkg/models"
)

type DamageCalculator interface {
	CalculateDamageCore(calculator.CombatSimulationRequest) (calculator.SimulationResult, error)
}

// CalculateDamageHandler is the HTTP handler for calculating damage.
//
//	@Summary		Calculate Damage
//	@Description	Calculates statistical damage based on input parameters like attack rolls, modifiers, and defense stats.
//	@Tags			damage
//	@Accept			json
//	@Produce		json
//	@Param			X-Request-ID	header		string							false	"Request UUID"
//	@Param			request			body		damagerequest.DamageRequestDTO	true	"Calculation Parameters"
//	@Success		200				{object}	damagerequest.DamageResponseDTO
//	@Failure		400				{object}	map[string]string	"Invalid input payload"
//	@Router			/damage/calculate [post]
func CalculateDamageHandler(calculator DamageCalculator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Check if the HTTP method is POST. If not, return a 405 (Method Not Allowed).
		if r.Method != http.MethodPost {
			// Use a helper function to send an error response for method mismatch
			SendError(w, "", "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		// Get the Request ID from middleware (useful for tracing/logging)
		reqID := middleware.GetRequestID(r.Context())

		// Initialize a variable to store the decoded request data
		var dto damagerequest.DamageRequestDTO

		// Attempt to decode the incoming JSON request body into the DamageRequest struct
		if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
			// Log the error for debugging purposes
			log.Printf("[%s] JSON decode error: %v", reqID, err)

			// Define a default error message in case of malformed JSON
			msg := "Malformed JSON or invalid data types"

			// If the error is just an empty body (EOF), provide a specific message
			if err == io.EOF {
				msg = "Request body cannot be empty"
			}

			// Send an error response with the appropriate status code (400 Bad Request)
			SendError(w, reqID, msg, http.StatusBadRequest)
			return
		}

		if err := dto.Validate(); err != nil {
			SendError(w, reqID, err.Error(), http.StatusBadRequest)
			return
		}

		domainReq, err := dto.ToDomain()
		if err != nil {
			// If parsing a dice string fails,
			// it will be caught here.
			SendError(w, reqID, err.Error(), http.StatusUnprocessableEntity)
			return
		}

		// Call the business logic layer (calculator) to calculate damage using the decoded request data
		result, err := calculator.CalculateDamageCore(domainReq)
		if err != nil {
			// Log calculation errors for debugging
			log.Printf("[%s] Calculation error: %v", reqID, err)

			// Send a business logic error response (400 Bad Request)
			SendError(w, reqID, err.Error(), http.StatusBadRequest)
			return
		}

		// Assign the Request UUID to the response for tracking purposes
		resp := damagerequest.MapResultToResponse(result, reqID)

		// Set the response content type to JSON
		w.Header().Set("Content-Type", "application/json")
		// Send a 200 OK status with the calculated damage results encoded as JSON
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			// Log any errors in encoding the response to JSON
			log.Printf("[%s] Error encoding JSON response: %v", reqID, err)
		}
	}
}
