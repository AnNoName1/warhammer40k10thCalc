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
	middleware "github.com/AnNoName1/warhammer40k10thCalc/internal/middleware"
)

// CalculateDamageHandler calculates the expected damage.
//
//	@Summary		Calculate Damage
//	@Description	Calculates statistical damage based on input parameters
//	@Tags			damage
//	@Accept			json
//	@Produce		json
//	@Param			X-Request-ID	header		string						false	"Request UUID"
//	@Param			request			body		damagerequest.DamageRequest	true	"Calculation Parameters"
//	@Success		200				{object}	damagerequest.DamageResponse
//	@Router			/damage/calculate [post]
func CalculateDamageHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		// Use the helper for Method Not Allowed
		SendError(w, "", "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	reqID := middleware.GetRequestID(r.Context())

	var req damagerequest.DamageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[%s] JSON decode error: %v", reqID, err)
		msg := "Malformed JSON or invalid data types"
		if err == io.EOF {
			msg = "Request body cannot be empty"
		}
		// Use the helper for 400 errors
		SendError(w, reqID, msg, http.StatusBadRequest)
		return
	}

	resp, err := calculator.CalculateDamageCore(req)
	if err != nil {
		log.Printf("[%s] Calculation error: %v", reqID, err)
		// Use the helper for business logic errors
		SendError(w, reqID, err.Error(), http.StatusBadRequest)
		return
	}

	resp.RequestUUID = reqID

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("[%s] Error encoding JSON response: %v", reqID, err)
	}
}
