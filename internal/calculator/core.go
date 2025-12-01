package calculator

import (
	"fmt"

	damagerequest "github.com/AnNoName1/warhammer40k10thCalc/pkg/models"
)

// CalculateDamageCore is the exported function containing the business logic.
// It accepts the DamageRequest and returns a calculated response and an error.
func CalculateDamageCore(req damagerequest.DamageRequest) (damagerequest.DamageResponse, error) {
	// Check for a required input
	if req.NumModels <= 0 {
		return damagerequest.DamageResponse{}, fmt.Errorf("attacks must be greater than zero")
	}

	//
	// damage calculation logic here
	//

	// For demonstration, return a placeholder response:
	result := damagerequest.DamageResponse{
		AverageHits: float64(req.NumModels) * 0.5, // Example calculation
		Message:     "Unfinished calculation performed",
	}

	return result, nil
}
