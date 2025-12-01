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

	damagerequest "github.com/AnNoName1/warhammer40k10thCalc/pkg/models"

	calculator "github.com/AnNoName1/warhammer40k10thCalc/internal/calculator"
)

// CalculateDamageHandler calculates the expected damage.
//
//	@Summary		Calculate Damage
//	@Description	Calculates statistical damage based on input parameters
//	@Tags			damage
//	@Accept			json
//	@Produce		json
//	@Param			request	body		damagerequest.DamageRequest	true	"Calculation Parameters"
//	@Success		200		{object}	damagerequest.DamageResponse
//	@Router			/damage/calculate [post]
func CalculateDamageHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// 1. Decode the Request Body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusInternalServerError)
		return
	}

	var req damagerequest.DamageRequest
	if err := json.Unmarshal(body, &req); err != nil {
		log.Printf("Error decoding JSON: %v", err)
		http.Error(w, "Invalid JSON format: "+err.Error(), http.StatusBadRequest)
		return
	}

	// 2. Call Core Business Logic
	resp, err := calculator.CalculateDamageCore(req)
	if err != nil {
		log.Printf("Calculation error: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// 3. Encode the Response Body
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("Error encoding JSON response: %v", err)
	}
}
