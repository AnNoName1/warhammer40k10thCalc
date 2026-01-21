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
)

// _calculateHitProbability calculates the hit probability for a single attack.
// Returns (normal_hit_probability, lethal_hit_probability).
func CalculateHitExpected(bs int, rerollType RerollType, hitModifier int,
	lethalHits bool, sustainedHits int, criticalThreshold int) (float64, float64) {
	const oneSixth = 1.0 / 6.0

	// Default threshold to 6 if not provided or set to 0
	if criticalThreshold <= 0 {
		criticalThreshold = 6
	}

	bsFloat := float64(bs)
	hitModifierFloat := float64(hitModifier)

	// 1. Base Hit Chance
	targetRollChance := (7.0 - bsFloat + hitModifierFloat) / 6.0
	hitChance := math.Max(oneSixth, math.Min(targetRollChance, 5.0/6.0))
	missChance := 1.0 - hitChance

	// 2. Critical Hit Chance (The trigger for Lethal/Sustained)
	// Formula: (7 - threshold) / 6. Example: 6+ is 1/6, 5+ is 2/6
	critChance := float64(7-criticalThreshold) / 6.0

	// Apply Rerolls to both standard hits and critical hits
	if rerollType == RerollOnes {
		hitChance += oneSixth * hitChance
		critChance += oneSixth * critChance
	} else if rerollType == RerollFail {
		hitChance += missChance * hitChance
		critChance += missChance * critChance
	}

	// 3. Handle Sustained Hits (Additional Normal Hits)
	// These are added to hitChance regardless of Lethal Hits.
	if sustainedHits > 0 {
		hitChance += critChance * float64(sustainedHits)
	}

	// 4. Handle Lethal Hits (Converted from Normal Hits to Lethal)
	lethalHitChance := 0.0
	if lethalHits {
		lethalHitChance = critChance

		// Subtract the crits from the hit pool because they moved to the lethal pool
		hitChance -= lethalHitChance
		hitChance = math.Max(0.0, hitChance)
	}

	return hitChance, lethalHitChance
}
