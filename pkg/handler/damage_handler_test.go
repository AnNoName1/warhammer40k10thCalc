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
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	calculator "github.com/AnNoName1/warhammer40k10thCalc/internal/calculator"
	damagerequest "github.com/AnNoName1/warhammer40k10thCalc/pkg/models"
)

type MockCalculator struct {
	ShouldFail bool
	LastReq    calculator.CombatSimulationRequest
}

// CalculateDamageCore implements [DamageCalculator].
func (m *MockCalculator) CalculateDamageCore(
	req calculator.CombatSimulationRequest,
) (calculator.SimulationResult, error) {

	m.LastReq = req

	if m.ShouldFail {
		return calculator.SimulationResult{}, errors.New("core failure")
	}

	return calculator.SimulationResult{
		AverageHits:      5.0,
		AverageDestroyed: 2.0,
		HitDist:          map[int]float64{5: 1.0},
		WoundDist:        map[int]float64{3: 1.0},
		PenDist:          map[int]float64{2: 1.0},
		DestroyedDist:    map[int]float64{2: 1.0},
	}, nil
}

func validRequestJSON() string {
	return `{
		"attacker": {
			"num_models": 1,
			"attacks_string": "1",
			"bs": 4,
			"s": 4,
			"ap": 0,
			"d": "1"
		},
		"target": {
			"t": 4,
			"save": 3,
			"wounds_per_model": 2,
			"model_count": 5
		},
		"rules": {}
	}`
}

func TestCalculateDamageHandler_Success(t *testing.T) {
	mock := &MockCalculator{}
	h := CalculateDamageHandler(mock)

	req := httptest.NewRequest(
		http.MethodPost,
		"/damage/calculate",
		bytes.NewBufferString(validRequestJSON()),
	)
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp damagerequest.DamageResponseDTO
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.Summary.AverageHits != 5.0 {
		t.Fatalf("unexpected hits: %f", resp.Summary.AverageHits)
	}
}
func TestCalculateDamageHandler_CoreError(t *testing.T) {
	mock := &MockCalculator{ShouldFail: true}
	h := CalculateDamageHandler(mock)

	req := httptest.NewRequest(
		http.MethodPost,
		"/damage/calculate",
		bytes.NewBufferString(validRequestJSON()),
	)
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestCalculateDamageHandler_MethodNotAllowed(t *testing.T) {
	mock := &MockCalculator{}
	h := CalculateDamageHandler(mock)

	req := httptest.NewRequest(http.MethodGet, "/damage/calculate", nil)
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rr.Code)
	}
}

func TestCalculateDamageHandler_MalformedJSON(t *testing.T) {
	mock := &MockCalculator{}
	h := CalculateDamageHandler(mock)

	req := httptest.NewRequest(
		http.MethodPost,
		"/damage/calculate",
		bytes.NewBufferString(`{ "attacker": {`),
	)
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestCalculateDamageHandler_ValidationDeepDive(t *testing.T) {
	mock := &MockCalculator{}
	h := CalculateDamageHandler(mock)

	// Sub-tests for specific validation rules
	t.Run("ValueRangeFail_BS", func(t *testing.T) {
		// BS must be between 2 and 6. 7 is impossible in 40k.
		body := `{
			"attacker": { "num_models": 1, "attacks_string": "1", "bs": 7, "s": 4, "ap": 0, "d": "1" },
			"target": { "t": 4, "save": 3, "wounds_per_model": 1, "model_count": 1 }
		}`
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(body))
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for BS range fail, got %d", rr.Code)
		}
	})

	t.Run("ValueRangeFail_Save", func(t *testing.T) {
		// Save must be 2 or higher (1+ saves do not exist naturally).
		body := `{
			"attacker": { "num_models": 1, "attacks_string": "1", "bs": 3, "s": 4, "ap": 0, "d": "1" },
			"target": { "t": 4, "save": 1, "wounds_per_model": 1, "model_count": 1 }
		}`
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(body))
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for Save range fail, got %d", rr.Code)
		}
	})

	t.Run("BlastMissingTargetModelCount", func(t *testing.T) {
		// Blast is true, but model_count is 0 or missing.
		body := `{
			"attacker": { 
				"num_models": 1, 
				"attacks_string": "D6", 
				"bs": 3, "s": 4, "ap": 0, "d": "1",
				"blast": true 
			},
			"target": { 
				"t": 4, "save": 3, "wounds_per_model": 1, 
				"model_count": 0 
			}
		}`
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(body))
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for Blast without model_count, got %d", rr.Code)
		}
	})

	t.Run("InvalidAttacksString", func(t *testing.T) {
		// "banana" is not a valid dice expression
		body := `{
            "attacker": { "num_models": 1, "attacks_string": "banana", "bs": 3, "s": 4, "ap": 0, "d": "1" },
            "target": { "t": 4, "save": 3, "wounds_per_model": 1, "model_count": 1 }
        }`
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(body))
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnprocessableEntity {
			t.Errorf("expected 422 for garbage attacks string, got %d", rr.Code)
		}
	})

	t.Run("MalformedDamageDice", func(t *testing.T) {
		// "2g6" is a typo for "2d6"
		body := `{
            "attacker": { "num_models": 1, "attacks_string": "1", "bs": 3, "s": 4, "ap": 0, "d": "2g6" },
            "target": { "t": 4, "save": 3, "wounds_per_model": 1, "model_count": 1 }
        }`
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(body))
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnprocessableEntity {
			t.Errorf("expected 422 for malformed damage string, got %d", rr.Code)
		}
	})

	t.Run("NegativeValueFail", func(t *testing.T) {
		// Strength (s) cannot be negative.
		body := `{
			"attacker": { "num_models": 1, "attacks_string": "1", "bs": 3, "s": -5, "ap": 0, "d": "1" },
			"target": { "t": 4, "save": 3, "wounds_per_model": 1, "model_count": 1 }
		}`
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(body))
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for negative strength, got %d", rr.Code)
		}
	})
}
