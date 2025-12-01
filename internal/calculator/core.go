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
