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
	"testing"

	"github.com/stretchr/testify/assert"
)

// Define a small utility to check float equality with tolerance
const floatTolerance = 0.0001

func TestCalculateWoundProbability(t *testing.T) {
	// A slice of test cases
	tests := []struct {
		name                     string
		s                        int
		t                        int
		rerollType               RerollType
		woundModifier            int
		devastatingWounds        bool
		criticalWoundThreshold   int // Added field
		expectedNormalWound      float64
		expectedDevastatingWound float64
	}{
		// --- Existing Tests (Threshold 6) ---
		{
			name:                     "2+ Wound (S >= 2T) - 5/6",
			s:                        10,
			t:                        4,
			rerollType:               RerollNone,
			criticalWoundThreshold:   6,
			expectedNormalWound:      5.0 / 6.0,
			expectedDevastatingWound: 0.0,
		},
		{
			name:                     "6+ Wound (S <= T/2) - 1/6",
			s:                        2,
			t:                        5,
			rerollType:               RerollNone,
			criticalWoundThreshold:   6,
			expectedNormalWound:      1.0 / 6.0,
			expectedDevastatingWound: 0.0,
		},
		{
			name:                     "4+ Wound with DevastatingWounds and RerollOnes",
			s:                        4,
			t:                        4,
			rerollType:               RerollOnes,
			devastatingWounds:        true,
			criticalWoundThreshold:   6,
			expectedNormalWound:      ((3.0 / 6.0) + (1.0 / 6.0 * 3.0 / 6.0)) - (1.0/6.0 + 1.0/6.0*1.0/6.0),
			expectedDevastatingWound: 1.0/6.0 + 1.0/6.0*1.0/6.0,
		},
		{
			// Anti-2+ means Crits on 2+. Even if S=1 and T=100, success must be at least 5/6.
			name:                     "Anti-2+ (Crit on 2+) without Devastating",
			s:                        1,
			t:                        10, // Normally requires 6+
			rerollType:               RerollNone,
			devastatingWounds:        false,
			criticalWoundThreshold:   2,
			expectedNormalWound:      5.0 / 6.0, // Crit success floor (2,3,4,5,6)
			expectedDevastatingWound: 0.0,
		},
		{
			// Anti-2+ with Devastating.
			// Total success is 5/6.
			// Because it's Anti-2+, all 5 of those successes are Devastating.
			name:                     "Anti-2+ (Crit on 2+) WITH Devastating",
			s:                        1,
			t:                        10,
			rerollType:               RerollNone,
			devastatingWounds:        true,
			criticalWoundThreshold:   2,
			expectedNormalWound:      0.0,       // All successes are diverted to Devastating
			expectedDevastatingWound: 5.0 / 6.0, // 2,3,4,5,6 are all Crits
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Pass tc.criticalWoundThreshold to the function
			actualNormalWound, actualDevastatingWound := CalculateWoundProbability(
				tc.s,
				tc.t,
				tc.rerollType,
				tc.woundModifier,
				tc.devastatingWounds,
				tc.criticalWoundThreshold,
			)

			// Assert Normal Wound
			assert.InDelta(t, tc.expectedNormalWound, actualNormalWound, floatTolerance, tc.name+": Normal Wound mismatch")

			// Assert Devastating Wound
			assert.InDelta(t, tc.expectedDevastatingWound, actualDevastatingWound, floatTolerance, tc.name+": Devastating Wound mismatch")
		})
	}
}
