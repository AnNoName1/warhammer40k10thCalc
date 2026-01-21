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

	"math"
)

const epsilon = 0.00001

func TestCalculateHitExpected(t *testing.T) {
	tests := []struct {
		name              string
		bs                int
		rerollType        RerollType
		hitModifier       int
		lethalHits        bool
		sustainedHits     int
		criticalThreshold int
		expectedNormalHit float64
		expectedLethalHit float64
	}{
		{
			name:              "BS 3+, No Rerolls, No Lethal",
			bs:                3,
			rerollType:        RerollNone,
			hitModifier:       0,
			lethalHits:        false,
			sustainedHits:     0,
			criticalThreshold: 6,
			// Hit on 3,4,5,6 → 4/6 total hits.
			// No crit effects, all are normal hits.
			expectedNormalHit: 4.0 / 6.0,
			expectedLethalHit: 0.0,
		},
		{
			name:              "BS 4+, Reroll Ones, No Lethal",
			bs:                4,
			rerollType:        RerollOnes,
			hitModifier:       0,
			lethalHits:        false,
			sustainedHits:     0,
			criticalThreshold: 6,
			// Initial hit chance: rolls 4,5,6 → 3/6 = 0.5.
			// Ones rerolled: 1/6 chance to reroll.
			// Reroll hits with same 0.5 probability.
			// Total hit = 0.5 + (1/6 * 0.5) = 0.58333...
			expectedNormalHit: 0.5 + (1.0/6.0)*0.5,
			expectedLethalHit: 0.0,
		},
		{
			name:              "BS 4+, Reroll Fails, No Lethal",
			bs:                4,
			rerollType:        RerollFail,
			hitModifier:       0,
			lethalHits:        false,
			sustainedHits:     0,
			criticalThreshold: 6,
			// Initial hit chance: 3/6 = 0.5.
			// Initial miss chance: 3/6 = 0.5.
			// Reroll misses with 0.5 hit chance.
			// Total hit = 0.5 + (0.5 * 0.5) = 0.75.
			expectedNormalHit: 0.75,
			expectedLethalHit: 0.0,
		},
		{
			name:              "BS 3+, Lethal Hits",
			bs:                3,
			rerollType:        RerollNone,
			hitModifier:       0,
			lethalHits:        true,
			sustainedHits:     0,
			criticalThreshold: 6,
			// Hits on 3,4,5,6 → 4/6 total.
			// Criticals on 6 → 1/6 lethal hits.
			// Remaining normal hits = 4/6 - 1/6 = 3/6.
			expectedNormalHit: 3.0 / 6.0,
			expectedLethalHit: 1.0 / 6.0,
		},
		{
			name:              "BS 4+, Lethal Hits + Reroll Fails",
			bs:                4,
			rerollType:        RerollFail,
			hitModifier:       0,
			lethalHits:        true,
			sustainedHits:     0,
			criticalThreshold: 6,
			// Base hit chance after reroll fails = 0.75.
			// Probability of rolling a 6 after rerolls = 0.25.
			// Lethal hits = 1/4.
			// Normal hits = 0.75 - 0.25 = 0.5.
			expectedNormalHit: 0.5,
			expectedLethalHit: 0.25,
		},
		{
			name:              "Threshold 5+ and Lethal Hits",
			bs:                3,
			rerollType:        RerollNone,
			hitModifier:       0,
			lethalHits:        true,
			sustainedHits:     0,
			criticalThreshold: 5,
			// Hits on 3,4,5,6 → 4/6.
			// Criticals on 5,6 → 2/6 lethal hits.
			// Normal hits = 4/6 - 2/6 = 2/6.
			expectedNormalHit: 2.0 / 6.0,
			expectedLethalHit: 2.0 / 6.0,
		},
		{
			name:              "Threshold 5+ and Sustained Hits 2",
			bs:                3,
			rerollType:        RerollNone,
			hitModifier:       0,
			lethalHits:        false,
			sustainedHits:     2,
			criticalThreshold: 5,
			// Base hits: 4/6.
			// Criticals on 5,6 → 2/6.
			// Each critical generates 2 additional hits.
			// Bonus hits = 2/6 * 2 = 4/6.
			// Total normal hits = 4/6 + 4/6 = 8/6.
			expectedNormalHit: 8.0 / 6.0,
			expectedLethalHit: 0.0,
		},
		{
			name:              "Sustained Hits 1 and Lethal Hits (Combined)",
			bs:                3,
			rerollType:        RerollNone,
			hitModifier:       0,
			lethalHits:        true,
			sustainedHits:     1,
			criticalThreshold: 6,
			// Hits on 3,4,5,6 → 4/6.
			// Criticals on 6 → 1/6.
			// Crit becomes lethal, not a normal hit.
			// Sustained Hits 1 adds one extra normal hit per crit.
			// Normal hits = (4/6 - 1/6) + 1/6 = 4/6.
			// Lethal hits = 1/6.
			expectedNormalHit: 4.0 / 6.0,
			expectedLethalHit: 1.0 / 6.0,
		},
		{
			name:              "Negative Hit Modifier with Lethal Hits",
			bs:                3,
			rerollType:        RerollNone,
			hitModifier:       -1,
			lethalHits:        true,
			sustainedHits:     0,
			criticalThreshold: 6,
			// Modifier worsens hit roll: hits now on 4,5,6 → 3/6.
			// Criticals on 6 → 1/6 lethal hits.
			// Normal hits = 3/6 - 1/6 = 2/6.
			expectedNormalHit: 2.0 / 6.0,
			expectedLethalHit: 1.0 / 6.0,
		},
		{
			name:              "BS 4+, Reroll Fails, Sustained Hits 1",
			bs:                4,
			rerollType:        RerollFail,
			hitModifier:       0,
			lethalHits:        false,
			sustainedHits:     1,
			criticalThreshold: 6,
			// Base hit chance with reroll fails:
			// Initial hit = 3/6 = 0.5
			// Miss = 3/6, reroll hits on 0.5
			// Total hit = 0.5 + (0.5 * 0.5) = 0.75
			//
			// Chance to roll a 6 after rerolls:
			// First roll 6: 1/6
			// Miss then reroll into 6: (3/6 * 1/6) = 1/12
			// Total crit = 1/6 + 1/12 = 1/4
			//
			// Sustained Hits 1:
			// Each crit adds 1 extra normal hit
			//
			// Normal hits:
			// Base hits (0.75) + bonus hits (0.25) = 1.0
			expectedNormalHit: 1.0,
			expectedLethalHit: 0.0,
		},
		{
			name:              "BS 3+, Reroll Ones, Lethal Hits, Sustained Hits 2",
			bs:                3,
			rerollType:        RerollOnes,
			hitModifier:       0,
			lethalHits:        true,
			sustainedHits:     2,
			criticalThreshold: 6,
			// Base hit chance:
			// Hits on 3,4,5,6 → 4/6
			//
			// Reroll ones:
			// Ones = 1/6, reroll hits with 4/6 chance
			// Added hits = 1/6 * 4/6 = 4/36
			//
			// Total hits = 4/6 + 4/36 = 28/36
			//
			// Criticals (6s):
			// First roll 6: 1/6
			// Rerolled 1 into 6: 1/6 * 1/6 = 1/36
			// Total crits = 7/36
			//
			// Lethal Hits:
			// All crits become lethal → 7/36 lethal
			//
			// Sustained Hits 2:
			// Each crit adds 2 normal hits
			// Bonus hits = 7/36 * 2 = 14/36
			//
			// Normal hits:
			// (Total hits - crits) + sustained
			// (28/36 - 7/36) + 14/36 = 35/36
			expectedNormalHit: 35.0 / 36.0,
			expectedLethalHit: 7.0 / 36.0,
		},
		{
			name:              "Positive Hit Modifier (+1), No Rerolls",
			bs:                4,
			rerollType:        RerollNone,
			hitModifier:       1, // Hits on 3+
			lethalHits:        false,
			sustainedHits:     0,
			criticalThreshold: 6,
			// Base BS 4+ → normally 3/6 hits
			// +1 modifier improves to 3+ → hits on 3,4,5,6 → 4/6
			// No crit effects
			expectedNormalHit: 4.0 / 6.0,
			expectedLethalHit: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotNormal, gotLethal := CalculateHitExpected(tt.bs, tt.rerollType, tt.hitModifier, tt.lethalHits, tt.sustainedHits, tt.criticalThreshold)

			if math.Abs(gotNormal-tt.expectedNormalHit) > epsilon {
				t.Errorf("%s - Normal Hit: expected %.5f, got %.5f", tt.name, tt.expectedNormalHit, gotNormal)
			}
			if math.Abs(gotLethal-tt.expectedLethalHit) > epsilon {
				t.Errorf("%s - Lethal Hit: expected %.5f, got %.5f", tt.name, tt.expectedLethalHit, gotLethal)
			}
		})
	}
}
