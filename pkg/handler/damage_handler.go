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
	"net/http"

	"go.uber.org/zap"

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
func CalculateDamageHandler(calculator DamageCalculator, log *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			SendError(w, "", "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		reqID := middleware.GetRequestID(r.Context())

		var dto damagerequest.DamageRequestDTO

		if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
			msg := "Malformed JSON or invalid data types"
			if err == io.EOF {
				msg = "Request body cannot be empty"
			}
			log.Warn("JSON decode error",
				zap.String("request_id", reqID),
				zap.Error(err),
			)
			SendError(w, reqID, msg, http.StatusBadRequest)
			return
		}

		if err := dto.Validate(); err != nil {
			log.Warn("validation failed",
				zap.String("request_id", reqID),
				zap.Error(err),
			)
			SendError(w, reqID, err.Error(), http.StatusBadRequest)
			return
		}

		domainReq, err := dto.ToDomain()
		if err != nil {
			log.Warn("domain mapping failed",
				zap.String("request_id", reqID),
				zap.Error(err),
			)
			SendError(w, reqID, err.Error(), http.StatusUnprocessableEntity)
			return
		}

		result, err := calculator.CalculateDamageCore(domainReq)
		if err != nil {
			log.Error("calculation error",
				zap.String("request_id", reqID),
				zap.Error(err),
			)
			SendError(w, reqID, err.Error(), http.StatusBadRequest)
			return
		}

		resp := damagerequest.MapResultToResponse(result, reqID)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			log.Error("JSON encode error",
				zap.String("request_id", reqID),
				zap.Error(err),
			)
		}
	}
}
