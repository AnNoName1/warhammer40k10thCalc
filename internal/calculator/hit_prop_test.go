package calculator

import (
	"testing"

	"math"

	. "github.com/AnNoName1/warhammer40k10thCalc/pkg/models"
)

const epsilon = 0.00001

func TestCalculateHitProbability(t *testing.T) {
	tests := []struct {
		name              string
		bs                int
		rerollType        RerollType
		hitModifier       int
		lethalHits        bool
		expectedNormalHit float64
		expectedLethalHit float64
	}{
		{
			name:              "BS 3+, No Rerolls, No Lethal",
			bs:                3,
			rerollType:        RerollNone,
			hitModifier:       0,
			lethalHits:        false,
			expectedNormalHit: 4.0 / 6.0, // ~0.666
			expectedLethalHit: 0.0,
		},
		{
			name:        "BS 4+, Reroll Ones, No Lethal",
			bs:          4,
			rerollType:  RerollOnes,
			hitModifier: 0,
			lethalHits:  false,
			// Hit: 3/6. Reroll 1s: Hit + (1/6 * Hit) -> 0.5 + 0.166*0.5 = 0.5833
			expectedNormalHit: 0.5 + (1.0/6.0)*0.5,
			expectedLethalHit: 0.0,
		},
		{
			name:        "BS 4+, Reroll Fails, No Lethal",
			bs:          4,
			rerollType:  RerollFail,
			hitModifier: 0,
			lethalHits:  false,
			// Hit: 0.5. Miss: 0.5. Total: 0.5 + (0.5 * 0.5) = 0.75
			expectedNormalHit: 0.75,
			expectedLethalHit: 0.0,
		},
		{
			name:        "BS 3+, Lethal Hits",
			bs:          3,
			rerollType:  RerollNone,
			hitModifier: 0,
			lethalHits:  true,
			// Total Hit: 4/6. Lethal: 1/6. Normal: 3/6.
			expectedNormalHit: 3.0 / 6.0,
			expectedLethalHit: 1.0 / 6.0,
		},
		{
			name:        "BS 4+, Lethal Hits + Reroll Fails",
			bs:          4,
			rerollType:  RerollFail,
			hitModifier: 0,
			lethalHits:  true,
			// Base Hit: 3/6 (0.5). Miss: 0.5.
			// Total Hit after Reroll: 0.5 + (0.5 * 0.5) = 0.75
			// Lethal Chance: Base(1/6) + Reroll(Miss * 1/6) -> 0.166 + (0.5 * 0.166) = 0.25
			// Normal Hit: Total(0.75) - Lethal(0.25) = 0.5
			expectedNormalHit: 0.5,
			expectedLethalHit: 0.25,
		},
		{
			name:        "Modifier Cap Check (BS 2+ with +1 mod)",
			bs:          2,
			rerollType:  RerollNone,
			hitModifier: 1,
			lethalHits:  false,
			// (7 - 2 + 1)/6 = 6/6 = 1.0. Cap at 5/6.
			expectedNormalHit: 5.0 / 6.0,
			expectedLethalHit: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotNormal, gotLethal := _calculateHitProbability(tt.bs, tt.rerollType, tt.hitModifier, tt.lethalHits)

			if math.Abs(gotNormal-tt.expectedNormalHit) > epsilon {
				t.Errorf("Normal Hit: expected %.5f, got %.5f", tt.expectedNormalHit, gotNormal)
			}
			if math.Abs(gotLethal-tt.expectedLethalHit) > epsilon {
				t.Errorf("Lethal Hit: expected %.5f, got %.5f", tt.expectedLethalHit, gotLethal)
			}
		})
	}
}
