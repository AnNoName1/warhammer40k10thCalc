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
)

func TestCalculateAttackDistribution(t *testing.T) {
	const epsilon = 1e-6

	tests := []struct {
		name          string
		attacks       DiceRoll
		attackerCount int
		blast         bool
		targetCount   int
		expectedCheck map[int]float64
	}{
		{
			name:          "Static Attacks 4 (No Blast, 1 model)",
			attacks:       DiceRoll{Count: 0, Sides: 0, Modifier: 4},
			attackerCount: 1,
			blast:         false,
			targetCount:   10,
			expectedCheck: map[int]float64{4: 1.0},
		},
		{
			name:          "Static Attacks 4 (No Blast, 2 models)",
			attacks:       DiceRoll{Count: 0, Sides: 0, Modifier: 4},
			attackerCount: 2,
			blast:         false,
			targetCount:   10,
			expectedCheck: map[int]float64{8: 1.0},
		},
		{
			name:          "Static 4 with Blast (11 targets -> +2 hits)",
			attacks:       DiceRoll{Count: 0, Sides: 0, Modifier: 4},
			attackerCount: 1,
			blast:         true,
			targetCount:   11, // floor(11/5) = 2
			expectedCheck: map[int]float64{
				6: 1.0, // 4 + 2
			},
		},
		{
			name:          "Static 4 with Blast (11 targets -> +2 hits), blast used separately for each attack",
			attacks:       DiceRoll{Count: 0, Sides: 0, Modifier: 4},
			attackerCount: 2,
			blast:         true,
			targetCount:   11, // floor(11/5) = 2
			expectedCheck: map[int]float64{
				12: 1.0, // 4 + 2
			},
		},
		{
			name:          "d6 with Blast (5 targets -> +1 hit)",
			attacks:       DiceRoll{Count: 1, Sides: 6, Modifier: 0},
			attackerCount: 1,
			blast:         true,
			targetCount:   5,
			expectedCheck: map[int]float64{
				2: 1.0 / 6.0,
				3: 1.0 / 6.0,
				4: 1.0 / 6.0,
				5: 1.0 / 6.0,
				6: 1.0 / 6.0,
				7: 1.0 / 6.0,
			},
		},
		{
			name:          "d3 with Blast (Under 5 targets -> +0 hits)",
			attacks:       DiceRoll{Count: 1, Sides: 3, Modifier: 0},
			attackerCount: 1,
			blast:         true,
			targetCount:   4,
			expectedCheck: map[int]float64{
				1: 1.0 / 3.0,
				2: 1.0 / 3.0,
				3: 1.0 / 3.0,
			},
		},
		{
			name:          "2d3+1 (Complex, No Blast)",
			attacks:       DiceRoll{Count: 2, Sides: 3, Modifier: 1},
			attackerCount: 1,
			blast:         false,
			targetCount:   10,
			expectedCheck: map[int]float64{
				3: 1.0 / 9.0,
				4: 2.0 / 9.0,
				5: 3.0 / 9.0,
				6: 2.0 / 9.0,
				7: 1.0 / 9.0,
			},
		},
		{
			name:          "Two models with d6 attacks (2 × 1d6 = 2d6)",
			attacks:       DiceRoll{Count: 1, Sides: 6, Modifier: 0},
			attackerCount: 2,
			blast:         false,
			targetCount:   1,
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
			name:          "Damage Floor (d3-2 rounds to 1)",
			attacks:       DiceRoll{Count: 1, Sides: 3, Modifier: -2},
			attackerCount: 1,
			blast:         false,
			targetCount:   1,
			expectedCheck: map[int]float64{
				1: 1.0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Logic now returns map directly since parsing errors are moved to DTO layer
			gotDist := CalculateAttackDistribution(tt.attacks, tt.attackerCount, tt.blast, tt.targetCount)

			for val, expectedProb := range tt.expectedCheck {
				gotProb, exists := gotDist[val]
				if !exists {
					if expectedProb > epsilon {
						t.Errorf("Value %d missing. Expected prob %.4f", val, expectedProb)
					}
					continue
				}

				if math.Abs(gotProb-expectedProb) > epsilon {
					t.Errorf("For %d: expected %.4f, got %.4f", val, expectedProb, gotProb)
				}
			}

			sum := 0.0
			for _, p := range gotDist {
				sum += p
			}
			if math.Abs(sum-1.0) > epsilon {
				t.Errorf("Total sum %.4f, expected 1.0", sum)
			}
		})
	}
}
