package calculator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	// Assuming 'damagerequest' is accessible or you define RerollType in this package for testing
	. "github.com/AnNoName1/warhammer40k10thCalc/pkg/models"
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
		expectedNormalWound      float64
		expectedDevastatingWound float64
	}{
		// --- Base Rolls (No Reroll, No Mod, No DW) ---
		{
			name:                     "2+ Wound (S >= 2T) - 5/6",
			s:                        10,
			t:                        4,
			rerollType:               RerollNone,
			expectedNormalWound:      5.0 / 6.0,
			expectedDevastatingWound: 0.0,
		},
		{
			name:                     "3+ Wound (S > T) - 4/6",
			s:                        6,
			t:                        4,
			rerollType:               RerollNone,
			expectedNormalWound:      4.0 / 6.0,
			expectedDevastatingWound: 0.0,
		},
		{
			name:                     "4+ Wound (S == T) - 3/6",
			s:                        4,
			t:                        4,
			rerollType:               RerollNone,
			expectedNormalWound:      3.0 / 6.0,
			expectedDevastatingWound: 0.0,
		},
		{
			name:                     "5+ Wound (S < T) - 2/6",
			s:                        3,
			t:                        4,
			rerollType:               RerollNone,
			expectedNormalWound:      2.0 / 6.0,
			expectedDevastatingWound: 0.0,
		},
		{
			name:                     "6+ Wound (S <= T/2) - 1/6",
			s:                        2,
			t:                        5,
			rerollType:               RerollNone,
			expectedNormalWound:      1.0 / 6.0,
			expectedDevastatingWound: 0.0,
		},

		// --- Modifier Tests ---
		{
			name:                     "4+ Wound with +1 Modifier -> 3+ (4/6)", // 4 - (-1) = 3
			s:                        4,
			t:                        4,
			rerollType:               RerollNone,
			woundModifier:            1, // 1 means 1 better for the roller
			expectedNormalWound:      4.0 / 6.0,
			expectedDevastatingWound: 0.0,
		},
		{
			name:                     "4+ Wound with -1 Modifier -> 5+ (2/6)", // 4 - (+1) = 5
			s:                        4,
			t:                        4,
			rerollType:               RerollNone,
			woundModifier:            -1,
			expectedNormalWound:      2.0 / 6.0,
			expectedDevastatingWound: 0.0,
		},
		{
			name:                     "2+ Wound limited by cap (2+ with -1 mod) - 5/6", // 2 - (+1) = 1 -> limited to 2
			s:                        10,
			t:                        4,
			rerollType:               RerollNone,
			woundModifier:            1,
			expectedNormalWound:      5.0 / 6.0,
			expectedDevastatingWound: 0.0,
		},

		// --- Reroll Tests ---
		// 4+ wound base: 3/6 success, 3/6 failure
		// Reroll ones: 1/6 rerolled. Success is (3/6) + (1/6)*(3/6) = 0.5 + 0.0833... = 0.5833...
		{
			name:                     "4+ Wound with RerollOnes",
			s:                        4,
			t:                        4,
			rerollType:               RerollOnes,
			expectedNormalWound:      (3.0 / 6.0) + (1.0 / 6.0 * 3.0 / 6.0),
			expectedDevastatingWound: 0.0,
		},
		// Reroll fails: 3/6 rerolled. Success is (3/6) + (3/6)*(3/6) = 0.5 + 0.25 = 0.75
		{
			name:                     "4+ Wound with RerollFail",
			s:                        4,
			t:                        4,
			rerollType:               RerollFail,
			expectedNormalWound:      (3.0 / 6.0) + (3.0 / 6.0 * 3.0 / 6.0),
			expectedDevastatingWound: 0.0,
		},

		// --- Devastating Wounds Tests (DW) ---
		// 4+ wound base: 3/6 success, 1/6 is a 6.
		{
			name:                     "4+ Wound with DevastatingWounds (No Reroll)",
			s:                        4,
			t:                        4,
			rerollType:               RerollNone,
			devastatingWounds:        true,
			expectedNormalWound:      (3.0 / 6.0) - (1.0 / 6.0), // 4+ success minus 6 (DW)
			expectedDevastatingWound: 1.0 / 6.0,
		},
		// DW with RerollOnes:
		// Normal wound: (3/6) - (1/6) + (1/6) * (3/6) - (1/6)*(1/6)
		// DW: 1/6 + (1/6)*(1/6)
		{
			name:                     "4+ Wound with DevastatingWounds and RerollOnes",
			s:                        4,
			t:                        4,
			rerollType:               RerollOnes,
			devastatingWounds:        true,
			expectedNormalWound:      ((3.0 / 6.0) + (1.0 / 6.0 * 3.0 / 6.0)) - (1.0/6.0 + 1.0/6.0*1.0/6.0),
			expectedDevastatingWound: 1.0/6.0 + 1.0/6.0*1.0/6.0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actualNormalWound, actualDevastatingWound := _calculateWoundProbability(
				tc.s,
				tc.t,
				tc.rerollType,
				tc.woundModifier,
				tc.devastatingWounds,
			)

			// Assert Normal Wound
			assert.InDelta(t, tc.expectedNormalWound, actualNormalWound, floatTolerance, "Normal Wound probability mismatch")

			// Assert Devastating Wound
			assert.InDelta(t, tc.expectedDevastatingWound, actualDevastatingWound, floatTolerance, "Devastating Wound probability mismatch")
		})
	}
}
