package calculator

import (
	"math"
	"testing"
)

func TestCalculateFailedSaveProbability(t *testing.T) {
	// Helper function for optional integer pointer
	intPtr := func(val int) *int { return &val }

	tests := []struct {
		name               string
		ap                 int
		save               int
		invulnerable       *int
		saveModifier       int
		expectedFailChance float64
	}{
		{
			name:         "Basic 3+ Save, AP 0",
			ap:           0,
			save:         3,
			invulnerable: nil,
			saveModifier: 0,
			// Pass: 4/6 (roll 3+). Fail: 2/6 (~0.33)
			expectedFailChance: 2.0 / 6.0,
		},
		{
			name:         "Save 4+, AP -1",
			ap:           1,
			save:         4,
			invulnerable: nil,
			saveModifier: 0,
			// Modified Save: 4 + 1 = 5+. Pass: 2/6 (roll 3+). Fail: 4/6 (~0.66)
			expectedFailChance: 4.0 / 6.0,
		},
		{
			name:         "Save 3+, AP -3 (4+ needed, max AP applied)",
			ap:           3,
			save:         3,
			invulnerable: nil,
			saveModifier: 0,
			// Modified Save: 3 + 3 = 0. Capped at 6+. Pass: 1/6. Fail: 5/6.
			expectedFailChance: 5.0 / 6.0,
		},
		{
			name:         "Invulnerable vs Normal (3+ vs 4++)",
			ap:           2,
			save:         3,
			invulnerable: intPtr(4),
			saveModifier: 0,
			// Modified Normal: 3 + 2 = 5. Used: Invuln 4+. Pass: 3/6. Fail: 3/6.
			expectedFailChance: 3.0 / 6.0,
		},
		{
			name:         "Invulnerable Not Used (3+ vs 5++)",
			ap:           0,
			save:         3,
			invulnerable: intPtr(5),
			saveModifier: 0,
			// Modified Normal: 3. Used: 3+. Pass: 4/6. Fail: 2/6.
			expectedFailChance: 2.0 / 6.0,
		},
		{
			name:         "Failed Save (7+ needed)",
			ap:           0,
			save:         7,
			invulnerable: nil,
			saveModifier: 0,
			// Required 7+. Chance to pass = 0. Fail: 1.0.
			expectedFailChance: 1.0,
		},
		{
			name:         "Save 4+, AP -1, +1 Modifier (Cover)",
			ap:           1,
			save:         4,
			invulnerable: nil,
			saveModifier: 1,
			// Modified Save: (4 - 1) + 1 = 4+. Pass: 3/6. Fail: 3/6.
			expectedFailChance: 3.0 / 6.0,
		},
		{
			name:         "Invulnerable 4++, AP -3, +1 Modifier (Cover)",
			ap:           3,
			save:         3,
			invulnerable: intPtr(4),
			saveModifier: 1,
			// Modified Normal: 3 - 3 + 1 = 1. Effective Save (max(1, 4)) = 4.
			// Used: 4+. Pass: 3/6. Fail: 3/6.
			expectedFailChance: 3.0 / 6.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotFailChance := _calculateFailedSaveProbability(tt.ap, tt.save, tt.invulnerable, tt.saveModifier)

			// Use math.Abs for comparison due to floating point arithmetic
			if math.Abs(gotFailChance-tt.expectedFailChance) > epsilon {
				t.Errorf("expected Failed Save Chance %.5f, got %.5f", tt.expectedFailChance, gotFailChance)
			}
		})
	}
}
