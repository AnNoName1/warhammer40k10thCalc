// Copyright (c) 2026 Olbutov Aleksandr
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
)

func TestCalculateDamageDistribution(t *testing.T) {
	intPtr := func(i int) *int { return &i }
	const epsilon = 1e-6

	tests := []struct {
		name          string
		damage        DiceRoll
		fnp           *int
		expectedCheck map[int]float64
	}{
		{
			name:   "Static Damage 3",
			damage: DiceRoll{Count: 0, Sides: 0, Modifier: 3},
			fnp:    nil,
			expectedCheck: map[int]float64{
				3: 1.0,
			},
		},
		{
			name:   "Basic d6",
			damage: DiceRoll{Count: 1, Sides: 6, Modifier: 0},
			fnp:    nil,
			expectedCheck: map[int]float64{
				1: 1.0 / 6.0,
				2: 1.0 / 6.0,
				3: 1.0 / 6.0,
				4: 1.0 / 6.0,
				5: 1.0 / 6.0,
				6: 1.0 / 6.0,
			},
		},
		{
			name:   "2d6 (Bell Curve Check)",
			damage: DiceRoll{Count: 2, Sides: 6, Modifier: 0},
			fnp:    nil,
			expectedCheck: map[int]float64{
				2:  1.0 / 36.0,
				3:  2.0 / 36.0,
				4:  3.0 / 36.0,
				5:  4.0 / 36.0,
				6:  5.0 / 36.0,
				7:  6.0 / 36.0,
				8:  5.0 / 36.0,
				9:  4.0 / 36.0,
				10: 3.0 / 36.0,
				11: 2.0 / 36.0,
				12: 1.0 / 36.0,
			},
		},
		{
			name:   "d3+1",
			damage: DiceRoll{Count: 1, Sides: 3, Modifier: 1},
			fnp:    nil,
			expectedCheck: map[int]float64{
				2: 1.0 / 3.0, // rolled 1 + 1
				3: 1.0 / 3.0, // rolled 2 + 1
				4: 1.0 / 3.0, // rolled 3 + 1
			},
		},
		{
			name:   "Static 2 with FNP 5+",
			damage: DiceRoll{Count: 0, Sides: 0, Modifier: 2},
			fnp:    intPtr(5),
			expectedCheck: map[int]float64{
				0: 1.0 / 9.0,
				1: 4.0 / 9.0,
				2: 4.0 / 9.0,
			},
		},
		{
			name:   "Single d1 (Fixed 1) with FNP 6+",
			damage: DiceRoll{Count: 1, Sides: 1, Modifier: 0},
			fnp:    intPtr(6),
			expectedCheck: map[int]float64{
				0: 1.0 / 6.0,
				1: 5.0 / 6.0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Calling the internal distribution logic
			gotDist := _calculateDamageDistribution(tt.damage, tt.fnp)

			for dmgVal, expectedProb := range tt.expectedCheck {
				gotProb, exists := gotDist[dmgVal]
				if !exists {
					if expectedProb > epsilon {
						t.Errorf("Damage value %d missing from distribution", dmgVal)
					}
					continue
				}

				if math.Abs(gotProb-expectedProb) > epsilon {
					t.Errorf("For Damage %d: expected prob %.4f, got %.4f", dmgVal, expectedProb, gotProb)
				}
			}

			sum := 0.0
			for _, p := range gotDist {
				sum += p
			}
			if math.Abs(sum-1.0) > epsilon {
				t.Errorf("Total probability sum is %.4f, expected 1.0", sum)
			}
		})
	}
}
