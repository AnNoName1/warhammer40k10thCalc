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

func TestCalculateFailedSaveProbability(t *testing.T) {
	const epsilon = 0.0001

	tests := []struct {
		name               string
		ap                 int
		save               int
		invulnerable       *int
		saveModifier       int
		hasCover           bool       // New field
		saveReroll         RerollType // New field
		expectedFailChance float64
	}{
		// --- Existing Tests (Updated with defaults) ---
		{
			name:               "Basic 3+ Save, AP 0",
			ap:                 0,
			save:               3,
			invulnerable:       nil,
			saveModifier:       0,
			hasCover:           false,
			saveReroll:         RerollNone,
			expectedFailChance: 2.0 / 6.0,
		},
		{
			name:               "Save 4+, AP -1, +1 Modifier (General Mod)",
			ap:                 1,
			save:               4,
			invulnerable:       nil,
			saveModifier:       1,
			hasCover:           false,
			saveReroll:         RerollNone,
			expectedFailChance: 3.0 / 6.0,
		},
		{
			// Rule: 3+ save vs AP 0 ignores Cover.
			name:               "BoC: 3+ Save vs AP 0 (Should ignore Cover)",
			ap:                 0,
			save:               3,
			invulnerable:       nil,
			saveModifier:       0,
			hasCover:           true,
			saveReroll:         RerollNone,
			expectedFailChance: 2.0 / 6.0, // Still 3+ (fail on 1,2)
		},
		{
			// Rule: 3+ save vs AP 1 DOES get Cover.
			name:         "BoC: 3+ Save vs AP 1 (Should apply Cover)",
			ap:           1,
			save:         3,
			invulnerable: nil,
			saveModifier: 0,
			hasCover:     true,
			saveReroll:   RerollNone,
			// Math: 3 (Save) + 1 (AP) - 1 (Cover) = 3+.
			expectedFailChance: 2.0 / 6.0,
		},
		{
			// Rule: 4+ save vs AP 0 DOES get Cover.
			name:         "BoC: 4+ Save vs AP 0 (Should apply Cover)",
			ap:           0,
			save:         4,
			invulnerable: nil,
			saveModifier: 0,
			hasCover:     true,
			saveReroll:   RerollNone,
			// Math: 4 (Save) + 0 (AP) - 1 (Cover) = 3+.
			expectedFailChance: 2.0 / 6.0,
		},
		{
			// Reroll Test
			name:         "Save 3+ with Reroll Ones",
			ap:           0,
			save:         3,
			invulnerable: nil,
			saveModifier: 0,
			hasCover:     false,
			saveReroll:   RerollOnes,
			// Base fail: 2/6.
			// Reroll: (1/6 chance to roll a 1) * (2/6 chance to fail again)
			// Total fail: (1/6) + (1/6 * 2/6) = 6/36 + 2/36 = 8/36 = 2/9
			expectedFailChance: 2.0 / 9.0,
		},
		{
			// Invulnerable (4+) is better than the armor save (6+), so it wins.
			name:               "Invulnerable Save Beats Worse Armor Save",
			ap:                 0,
			save:               6,
			invulnerable:       intPtr(4),
			saveModifier:       0,
			hasCover:           false,
			saveReroll:         RerollNone,
			expectedFailChance: 3.0 / 6.0,
		},
		{
			// Save 6+ vs AP 2 needs 8+, beyond the rollable 6+: auto-fail.
			name:               "Target Beyond 6+ Auto-Fails",
			ap:                 2,
			save:               6,
			invulnerable:       nil,
			saveModifier:       0,
			hasCover:           false,
			saveReroll:         RerollNone,
			expectedFailChance: 1.0,
		},
		{
			// Reroll Fail Test: base fail 2/6, both the original roll and its
			// reroll must fail, so total fail = (2/6)^2 = 1/9.
			name:               "Save 3+ with Reroll Fails",
			ap:                 0,
			save:               3,
			invulnerable:       nil,
			saveModifier:       0,
			hasCover:           false,
			saveReroll:         RerollFail,
			expectedFailChance: 1.0 / 9.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotFailChance := CalculateFailedSaveProbability(
				tt.ap,
				tt.save,
				tt.invulnerable,
				tt.saveModifier,
				tt.hasCover,
				tt.saveReroll,
			)

			if math.Abs(gotFailChance-tt.expectedFailChance) > epsilon {
				t.Errorf("%s: expected Failed Save Chance %.5f, got %.5f", tt.name, tt.expectedFailChance, gotFailChance)
			}
		})
	}
}
