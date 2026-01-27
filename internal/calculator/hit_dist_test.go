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

// Define the outcome struct locally for the test (or reference it if it's exported)
// Ensure this matches the struct in your implementation file.
/*
type HitOutcome struct {
    NormalHits int
    LethalHits int
}
*/

const epsilon = 0.00001

func TestCalculateSingleHitDistribution(t *testing.T) {
	tests := []struct {
		name              string
		bs                int
		rerollType        RerollType
		hitModifier       int
		lethalHits        bool
		sustainedHits     int
		criticalThreshold int
		expectedDist      map[HitOutcome]float64
	}{
		{
			name:              "BS 3+, No Rerolls, No Lethal",
			bs:                3,
			rerollType:        RerollNone,
			hitModifier:       0,
			lethalHits:        false,
			sustainedHits:     0,
			criticalThreshold: 6,
			expectedDist: map[HitOutcome]float64{
				{NormalHits: 0, LethalHits: 0}: 2.0 / 6.0, // Rolls 1,2
				{NormalHits: 1, LethalHits: 0}: 4.0 / 6.0, // Rolls 3,4,5,6
			},
		},
		{
			name:              "BS 4+, Reroll Ones, No Lethal",
			bs:                4,
			rerollType:        RerollOnes,
			hitModifier:       0,
			lethalHits:        false,
			sustainedHits:     0,
			criticalThreshold: 6,
			expectedDist: map[HitOutcome]float64{
				// Miss: Rolls 2,3 (2/6) + Rerolled 1s into 1,2,3 (1/6 * 3/6 = 3/36)
				// 12/36 + 3/36 = 15/36 = 5/12
				{NormalHits: 0, LethalHits: 0}: 5.0 / 12.0,
				// Hit: Rolls 4,5,6 (3/6) + Rerolled 1s into 4,5,6 (1/6 * 3/6 = 3/36)
				// 18/36 + 3/36 = 21/36 = 7/12
				{NormalHits: 1, LethalHits: 0}: 7.0 / 12.0,
			},
		},
		{
			name:              "BS 4+, Reroll Fails, No Lethal",
			bs:                4,
			rerollType:        RerollFail,
			hitModifier:       0,
			lethalHits:        false,
			sustainedHits:     0,
			criticalThreshold: 6,
			expectedDist: map[HitOutcome]float64{
				// Miss (0.5) -> Reroll Miss (0.5) = 0.25 total miss chance
				{NormalHits: 0, LethalHits: 0}: 0.25,
				// Hit (0.5) + MissThenHit (0.5 * 0.5) = 0.75
				{NormalHits: 1, LethalHits: 0}: 0.75,
			},
		},
		{
			name:              "BS 3+, Lethal Hits",
			bs:                3,
			rerollType:        RerollNone,
			hitModifier:       0,
			lethalHits:        true,
			sustainedHits:     0,
			criticalThreshold: 6,
			expectedDist: map[HitOutcome]float64{
				// Miss: 1,2
				{NormalHits: 0, LethalHits: 0}: 2.0 / 6.0,
				// Normal: 3,4,5 (Crit 6 is removed from normal pool)
				{NormalHits: 1, LethalHits: 0}: 3.0 / 6.0,
				// Lethal: 6
				{NormalHits: 0, LethalHits: 1}: 1.0 / 6.0,
			},
		},
		{
			name:              "BS 4+, Lethal Hits + Reroll Fails",
			bs:                4,
			rerollType:        RerollFail,
			hitModifier:       0,
			lethalHits:        true,
			sustainedHits:     0,
			criticalThreshold: 6,
			expectedDist: map[HitOutcome]float64{
				// Total Miss: 0.25
				{NormalHits: 0, LethalHits: 0}: 0.25,
				// Normal Hit (Roll 4,5): Base 2/6 + Rerolled 1/6 (half of misses) -> 2/6 = 1/3
				// Base (1/3) + Reroll (1/2 * 1/3 = 1/6) = 3/6 = 0.5
				{NormalHits: 1, LethalHits: 0}: 0.5,
				// Lethal Hit (Roll 6): Base 1/6 + Reroll (1/2 * 1/6 = 1/12) = 3/12 = 0.25
				{NormalHits: 0, LethalHits: 1}: 0.25,
			},
		},
		{
			name:              "Threshold 5+ and Lethal Hits",
			bs:                3,
			rerollType:        RerollNone,
			hitModifier:       0,
			lethalHits:        true,
			sustainedHits:     0,
			criticalThreshold: 5,
			expectedDist: map[HitOutcome]float64{
				// Miss: 1,2
				{NormalHits: 0, LethalHits: 0}: 2.0 / 6.0,
				// Normal: 3,4 (5,6 are lethal)
				{NormalHits: 1, LethalHits: 0}: 2.0 / 6.0,
				// Lethal: 5,6
				{NormalHits: 0, LethalHits: 1}: 2.0 / 6.0,
			},
		},
		{
			name:              "Threshold 5+ and Sustained Hits 2",
			bs:                3,
			rerollType:        RerollNone,
			hitModifier:       0,
			lethalHits:        false,
			sustainedHits:     2,
			criticalThreshold: 5,
			expectedDist: map[HitOutcome]float64{
				// Miss: 1,2
				{NormalHits: 0, LethalHits: 0}: 2.0 / 6.0,
				// Normal Hit: 3,4
				{NormalHits: 1, LethalHits: 0}: 2.0 / 6.0,
				// Critical Hit: 5,6 -> 1 Base + 2 Sustained = 3 Hits
				{NormalHits: 3, LethalHits: 0}: 2.0 / 6.0,
			},
		},
		{
			name:              "Sustained Hits 1 and Lethal Hits (Combined)",
			bs:                3,
			rerollType:        RerollNone,
			hitModifier:       0,
			lethalHits:        true,
			sustainedHits:     1,
			criticalThreshold: 6,
			expectedDist: map[HitOutcome]float64{
				// Miss: 1,2
				{NormalHits: 0, LethalHits: 0}: 2.0 / 6.0,
				// Normal Hit: 3,4,5
				{NormalHits: 1, LethalHits: 0}: 3.0 / 6.0,
				// Critical Hit (6):
				// - 1 Lethal (Auto Wound)
				// - 1 Sustained (Normal Hit)
				{NormalHits: 1, LethalHits: 1}: 1.0 / 6.0,
			},
		},
		{
			name:              "Negative Hit Modifier with Lethal Hits",
			bs:                3,
			rerollType:        RerollNone,
			hitModifier:       -1,
			lethalHits:        true,
			sustainedHits:     0,
			criticalThreshold: 6,
			expectedDist: map[HitOutcome]float64{
				// Modified roll must be >= 3. Roll 4,5,6 succeeds.
				// Miss: 1,2,3
				{NormalHits: 0, LethalHits: 0}: 3.0 / 6.0,
				// Normal Hit: 4,5
				{NormalHits: 1, LethalHits: 0}: 2.0 / 6.0,
				// Critical: Unmodified 6. (Lethal)
				{NormalHits: 0, LethalHits: 1}: 1.0 / 6.0,
			},
		},
		{
			name:              "BS 4+, Reroll Fails, Sustained Hits 1",
			bs:                4,
			rerollType:        RerollFail,
			hitModifier:       0,
			lethalHits:        false,
			sustainedHits:     1,
			criticalThreshold: 6,
			expectedDist: map[HitOutcome]float64{
				// Miss: 0.25
				{NormalHits: 0, LethalHits: 0}: 0.25,
				// Normal Hit (4,5): Base (1/3) + Reroll (1/6) = 0.5
				{NormalHits: 1, LethalHits: 0}: 0.5,
				// Critical (6) -> 1 Base + 1 Sustained = 2 Hits
				// Base (1/6) + Reroll (1/12) = 0.25
				{NormalHits: 2, LethalHits: 0}: 0.25,
			},
		},
		{
			name:              "BS 3+, Reroll Ones, Lethal Hits, Sustained Hits 2",
			bs:                3,
			rerollType:        RerollOnes,
			hitModifier:       0,
			lethalHits:        true,
			sustainedHits:     2,
			criticalThreshold: 6,
			expectedDist: map[HitOutcome]float64{
				// Miss (1,2):
				// Base 2 (1/6) stays 1/6 = 6/36
				// Base 1 (1/6) rerolls into 1,2 (2/6 chance) -> 1/6 * 2/6 = 2/36
				// Total Miss = 8/36
				{NormalHits: 0, LethalHits: 0}: 8.0 / 36.0,

				// Normal Hit (3,4,5):
				// Base 3,4,5 (3/6) stays 3/6 = 18/36
				// Base 1 rerolls into 3,4,5 (3/6 chance) -> 1/6 * 3/6 = 3/36
				// Total Normal = 21/36
				{NormalHits: 1, LethalHits: 0}: 21.0 / 36.0,

				// Critical Hit (6) -> Lethal + Sustained 2
				// Outcome: 1 Lethal + 2 Normal
				// Base 6 (1/6) stays 1/6 = 6/36
				// Base 1 rerolls into 6 (1/6 chance) -> 1/6 * 1/6 = 1/36
				// Total Crit = 7/36
				{NormalHits: 2, LethalHits: 1}: 7.0 / 36.0,
			},
		},
		{
			name:              "Positive Hit Modifier (+1), No Rerolls",
			bs:                4,
			rerollType:        RerollNone,
			hitModifier:       1,
			lethalHits:        false,
			sustainedHits:     0,
			criticalThreshold: 6,
			expectedDist: map[HitOutcome]float64{
				// BS 4 with +1 hits on 3+
				// Miss: 1,2
				{NormalHits: 0, LethalHits: 0}: 2.0 / 6.0,
				// Hit: 3,4,5,6
				{NormalHits: 1, LethalHits: 0}: 4.0 / 6.0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotDist := CalculateSingleHitDistribution(tt.bs, tt.rerollType, tt.hitModifier, tt.lethalHits, tt.sustainedHits, tt.criticalThreshold)

			// Verify distribution matches
			for outcome, expectedProb := range tt.expectedDist {
				gotProb := gotDist[outcome]
				if math.Abs(gotProb-expectedProb) > epsilon {
					t.Errorf("Outcome %+v: expected prob %.5f, got %.5f", outcome, expectedProb, gotProb)
				}
			}

			// Verify no extra unexpected outcomes in 'got' (keys that shouldn't exist/have 0 prob)
			for outcome, gotProb := range gotDist {
				if gotProb > epsilon {
					if _, ok := tt.expectedDist[outcome]; !ok {
						t.Errorf("Unexpected outcome %+v with prob %.5f", outcome, gotProb)
					}
				}
			}
		})
	}
}
