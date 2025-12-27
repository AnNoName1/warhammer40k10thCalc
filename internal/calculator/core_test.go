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
	"math"
	"testing"

	damagerequest "github.com/AnNoName1/warhammer40k10thCalc/pkg/models"
)

const epsilonCore = 0.00001

func TestCalculateDamageCore_Distributions(t *testing.T) {
	tests := []struct {
		name                 string
		req                  damagerequest.DamageRequest
		expectedAvgHits      float64
		expectedAvgDestroyed float64
		// Maps for you to fill with exact probability distributions
		expectedHitsDist   map[int]float64
		expectedWoundsDist map[int]float64
		expectedPensDist   map[int]float64
		expectedKilledDist map[int]float64
		expectError        bool
	}{
		{
			name: "Verification Case: 2 Attacks, BS3+, D1 vs 1W",
			req: damagerequest.DamageRequest{
				NumModels: 1, WoundsPerModel: 1, AttacksString: "1",
				BS: 4, S: 5, T: 3, AP: 0, Save: 6, D: "1",
				HitReroll: damagerequest.RerollNone, WoundReroll: damagerequest.RerollNone,
			},
			expectedAvgHits:      0.5,
			expectedAvgDestroyed: 0.28,
			expectedHitsDist: map[int]float64{
				0: 0.5,
				1: 0.5,
				2: 0.0,
			},
			expectedWoundsDist: map[int]float64{
				0: 0,
				1: 0,
			},
			expectedPensDist: map[int]float64{
				0: 0,
				1: 0,
			},
			expectedKilledDist: map[int]float64{
				0: 0.72,
				1: 0.28,
			},
		},
		{
			name: "Mortal Wound Spillover Case",
			req: damagerequest.DamageRequest{
				NumModels: 3, WoundsPerModel: 2, AttacksString: "1",
				BS: 1, S: 4, T: 4, AP: 0, Save: 7, D: "3",
				DevastatingWounds: true,
				HitReroll:         damagerequest.RerollNone, WoundReroll: damagerequest.RerollNone,
			},
			// 1 Attack, Auto Hit, Auto Wound, 3 Mortals vs 2W models = 1.5 kills average
			expectedAvgHits:      1.0,
			expectedAvgDestroyed: 1.5,
			expectedKilledDist: map[int]float64{
				1: 0.5, // 50% chance to kill 1 (if D3 rolls 1 or 2) -> Wait, D is string "3"
				2: 0.5, // If D is fixed 3, it should kill 1 and wound 1.
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := CalculateDamageCore(tc.req)
			if tc.expectError {
				if err == nil {
					t.Fatal("expected error but got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// 1. Verify Averages
			verifyValue(t, "AverageHits", resp.AverageHits, tc.expectedAvgHits)
			verifyValue(t, "AverageDestroyed", resp.AverageDestroyed, tc.expectedAvgDestroyed)

			// 2. Verify Distributions (if provided in test case)
			if len(tc.expectedHitsDist) > 0 {
				verifyDist(t, "HitsDistribution", resp.HitsDistribution, tc.expectedHitsDist)
			}
			if len(tc.expectedWoundsDist) > 0 {
				verifyDist(t, "WoundsDistribution", resp.WoundsDistribution, tc.expectedWoundsDist)
			}
			if len(tc.expectedPensDist) > 0 {
				verifyDist(t, "PensDistribution", resp.PensDistribution, tc.expectedPensDist)
			}
			if len(tc.expectedKilledDist) > 0 {
				verifyDist(t, "DestroyedDistribution", resp.DestroyedDistribution, tc.expectedKilledDist)
			}
		})
	}
}

// Helper: Checks float equality within epsilon
func verifyValue(t *testing.T, label string, got, want float64) {
	if math.Abs(got-want) > epsilonCore {
		t.Errorf("%s: expected %.6f got %.6f", label, want, got)
	}
}

// Helper: Compares two probability maps
func verifyDist(t *testing.T, label string, got, want map[int]float64) {
	for k, wantP := range want {
		gotP, ok := got[k]
		if !ok {
			t.Errorf("%s: missing key %d in result", label, k)
			continue
		}
		if math.Abs(gotP-wantP) > epsilonCore {
			t.Errorf("%s key %d: expected probability %.6f got %.6f", label, k, wantP, gotP)
		}
	}
}
