package calculator

import (
	"math"
	"testing"
)

func TestCalculateDamageDistribution(t *testing.T) {
	intPtr := func(i int) *int { return &i }
	const epsilon = 1e-6

	tests := []struct {
		name         string
		damageString string
		fnp          *int
		// We verify specific points in the distribution to ensure correctness
		expectedCheck map[int]float64
	}{
		{
			name:         "Static Damage 3",
			damageString: "3",
			fnp:          nil,
			expectedCheck: map[int]float64{
				3: 1.0,
			},
		},
		{
			name:         "Basic d6",
			damageString: "d6",
			fnp:          nil,
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
			name:         "2d6 (Bell Curve Check)",
			damageString: "2d6",
			fnp:          nil,
			// The Python script would have failed this because it treated 2d6 as flat.
			// Correct math:
			// 2 (1+1) -> 1/36 (~0.0277)
			// 7 (Average) -> 6/36 (~0.1666)
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
			name:         "d3+1",
			damageString: "d3+1",
			fnp:          nil,
			expectedCheck: map[int]float64{
				2: 1.0 / 3.0, // rolled 1 + 1
				3: 1.0 / 3.0, // rolled 2 + 1
				4: 1.0 / 3.0, // rolled 3 + 1
			},
		},
		{
			name:         "Static 2 with FNP 5+",
			damageString: "2",
			fnp:          intPtr(5),
			// Incoming Damage: 2
			// FNP 5+ means P(Save)=2/6, P(Fail)=4/36
			// Binomial Outcomes (Damage Taken = Failures):
			// 0 Dmg (2 saves): (1/3)^2 = 1/9
			// 1 Dmg (1 save, 1 fail): 2 * (1/3)*(2/3) = 4/9
			// 2 Dmg (0 saves, 2 fail): (2/3)^2 = 4/9
			expectedCheck: map[int]float64{
				0: 1.0 / 9.0,
				1: 4.0 / 9.0,
				2: 4.0 / 9.0,
			},
		},
		{
			name:         "Single d1 (Fixed 1) with FNP 6+",
			damageString: "d1",
			fnp:          intPtr(6),
			// Incoming 1. P(Save)=1/6, P(Fail)=5/6.
			// 0 Dmg (Save): 1/6
			// 1 Dmg (Fail): 5/6
			expectedCheck: map[int]float64{
				0: 1.0 / 6.0,
				1: 5.0 / 6.0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotDist := _calculateDamageDistribution(tt.damageString, tt.fnp)

			for dmgVal, expectedProb := range tt.expectedCheck {
				gotProb, exists := gotDist[dmgVal]
				if !exists {
					// Treat missing as 0.0, but if we expected >0 that's an error
					if expectedProb > epsilon {
						t.Errorf("Damage value %d missing from distribution", dmgVal)
					}
					continue
				}

				if math.Abs(gotProb-expectedProb) > epsilon {
					t.Errorf("For Damage %d: expected prob %.4f, got %.4f", dmgVal, expectedProb, gotProb)
				}
			}

			// Optional: Check if sum is 1.0
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
