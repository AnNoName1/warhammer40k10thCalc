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
		attackString  string
		shouldError   bool
		// We verify specific points in the distribution to ensure correctness
		expectedCheck map[int]float64
	}{
		{
			name:         "Static Attacks 4",
			attackString: "4",
			shouldError:  false,
			expectedCheck: map[int]float64{
				4: 1.0,
			},
		},
		{
			name:         "Basic d6",
			attackString: "d6",
			shouldError:  false,
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
			name:         "2d6 (Bell Curve Convolution)",
			attackString: "2d6",
			shouldError:  false,
			// Logic:
			// 2 (1+1) -> 1/36
			// 7 (Avg) -> 6/36
			// 12 (6+6) -> 1/36
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
			name:         "d3+1 (Modifier)",
			attackString: "d3+1",
			shouldError:  false,
			expectedCheck: map[int]float64{
				2: 1.0 / 3.0, // rolled 1 + 1
				3: 1.0 / 3.0, // rolled 2 + 1
				4: 1.0 / 3.0, // rolled 3 + 1
			},
		},
		{
			name:         "2d3+1 (Complex)",
			attackString: "2d3+1",
			shouldError:  false,
			// 2d3 distribution:
			// 2 (1+1): 1/9
			// 3 (1+2, 2+1): 2/9
			// 4 (1+3, 2+2, 3+1): 3/9
			// 5 (2+3, 3+2): 2/9
			// 6 (3+3): 1/9
			// Apply +1 modifier -> Shift keys by 1
			expectedCheck: map[int]float64{
				3: 1.0 / 9.0,
				4: 2.0 / 9.0,
				5: 3.0 / 9.0,
				6: 2.0 / 9.0,
				7: 1.0 / 9.0,
			},
		},
		{
			name:         "Garbage Input",
			attackString: "invalid_dice",
			shouldError:  true,
			expectedCheck: nil,
		},
		{
			name:         "Zero Dice Count (0d6)",
			attackString: "0d6",
			shouldError:  false,
			expectedCheck: map[int]float64{
				0: 1.0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotDist, err := CalculateAttackDistribution(tt.attackString)

			// 1. Check Error State
			if tt.shouldError {
				if err == nil {
					t.Errorf("expected error for input '%s', but got nil", tt.attackString)
				}
				return // Stop checking distribution if we expected an error
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// 2. Check Distribution Values
			for val, expectedProb := range tt.expectedCheck {
				gotProb, exists := gotDist[val]
				if !exists {
					// Treat missing as 0.0, but if we expected >0 that's an error
					if expectedProb > epsilon {
						t.Errorf("Attack value %d missing from distribution", val)
					}
					continue
				}

				if math.Abs(gotProb-expectedProb) > epsilon {
					t.Errorf("For Attacks %d: expected prob %.4f, got %.4f", val, expectedProb, gotProb)
				}
			}

			// 3. Verify Total Probability Sums to 1.0
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