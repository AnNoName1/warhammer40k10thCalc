package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	middleware "github.com/AnNoName1/warhammer40k10thCalc/internal/middleware"
	damagerequest "github.com/AnNoName1/warhammer40k10thCalc/pkg/models"
)

func TestCalculateDamageHandler_IncludesRequestUUID(t *testing.T) {
	// create a minimal valid request body
	reqBody := damagerequest.DamageRequest{
		NumModels:     1,
		AttacksString: "1",
		BS:            4,
		S:             4,
		AP:            0,
		D:             "1",
		T:             4,
		Save:          7,
	}
	b, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/damage/calculate", bytes.NewReader(b))
	rr := httptest.NewRecorder()

	// wrap the handler with middleware to inject/generate request id
	h := middleware.LoggingMiddleware(http.HandlerFunc(CalculateDamageHandler))

	h.ServeHTTP(rr, req)

	res := rr.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status: %d", res.StatusCode)
	}

	// response must contain X-Request-ID header and body must include the same id
	id := res.Header.Get("X-Request-ID")
	if id == "" {
		t.Fatalf("expected X-Request-ID in response")
	}

	var resp damagerequest.DamageResponse
	if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.RequestUUID == "" {
		t.Fatalf("response missing RequestUUID")
	}
	if resp.RequestUUID != id {
		t.Fatalf("response RequestUUID (%s) != header id (%s)", resp.RequestUUID, id)
	}
}
