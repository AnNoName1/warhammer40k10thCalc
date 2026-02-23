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

// CalculateWoundProbability calculates the probability that a single successful hit
// will result in a wound on the target.
//
// Arguments:
// s (int): Weapon Strength.
// t (int): Target Toughness.
// rerollType (RerollType): Type of reroll (none, ones, fail).
// woundModifier (int): Modifier to the wound roll.
// devastatingWounds (bool): Presence of the [DEVASTATING WOUNDS] ability.
//
// Returns:
// (float64, float64): Probability of a normal wound and a devastating wound.

func CalculateWoundProbability(s int, t int, rerollType RerollType, woundModifier int, devastatingWounds bool,
	CriticalWoundThreshold int) (float64, float64) {
	// Constants
	const oneSixth = 1.0 / 6.0

	// 1. Determine the required Wound Roll (Target Wound Roll)
	var targetRoll int
	if s*2 <= t {
		targetRoll = 6
	} else if s < t {
		targetRoll = 5
	} else if s >= t*2 {
		targetRoll = 2
	} else if s > t {
		targetRoll = 3
	} else {
		targetRoll = 4
	}

	// 2. Apply the modifier and clamp between 2+ and 6+
	finalTargetRoll := float64(targetRoll) - float64(woundModifier)
	finalTargetRoll = math.Max(2.0, math.Min(6.0, finalTargetRoll))

	// 1. Calculate Base Success Chance (Normal math)
	woundChance := (7.0 - finalTargetRoll) / 6.0

	// 2. Calculate Critical Success Chance
	if CriticalWoundThreshold < 2 || CriticalWoundThreshold > 6 {
		CriticalWoundThreshold = 6
	}
	critChance := (7.0 - float64(CriticalWoundThreshold)) / 6.0

	// 3. APPLY RULE: Critical Wounds are always successful
	// If critChance is higher than woundChance (e.g., Anti-2+ vs T12),
	// the woundChance must be elevated to the critChance.
	if critChance > woundChance {
		woundChance = critChance
	}

	missChance := 1.0 - woundChance

	// 4. Process Rerolls
	switch rerollType {
	case RerollOnes:
		// Rerolling a 1 (1/6) gives another chance to hit wound or crit
		woundChance += oneSixth * woundChance
		critChance += oneSixth * critChance
	case RerollFail:
		// Rerolling all fails (missChance) gives another chance to hit wound or crit
		woundChance += missChance * woundChance
		critChance += missChance * critChance
	}

	// 5. Process [DEVASTATING WOUNDS]
	devastatingWoundChance := 0.0
	if devastatingWounds {
		devastatingWoundChance = critChance
		woundChance -= devastatingWoundChance
		woundChance = math.Max(0.0, woundChance)
	}

	return woundChance, devastatingWoundChance
}
