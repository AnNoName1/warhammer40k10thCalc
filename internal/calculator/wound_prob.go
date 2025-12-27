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

	damagerequest "github.com/AnNoName1/warhammer40k10thCalc/pkg/models"
)

// _calculateWoundProbability calculates the probability that a single successful hit
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

func _calculateWoundProbability(s int, t int, rerollType damagerequest.RerollType, woundModifier int, devastatingWounds bool) (float64, float64) {
	// Constants
	const oneSixth = 1.0 / 6.0
	const fiveSixths = 5.0 / 6.0

	// 1. Determine the required Wound Roll (Target Wound Roll)
	// Compare S (Strength) and T (Toughness)
	var targetRoll int
	if s*2 <= t {
		targetRoll = 6 // S <= T/2: requires 6+
	} else if s < t {
		targetRoll = 5 // S < T: requires 5+
	} else if s >= t*2 {
		targetRoll = 2 // S >= 2T: requires 2+
	} else if s > t {
		targetRoll = 3 // S > T: requires 3+
	} else {
		targetRoll = 4 // S == T: requires 4+
	}

	// 2. Apply the modifier (-1/+1)
	finalTargetRoll := float64(targetRoll) - float64(woundModifier)

	// Limit the required roll: minimum 2+ (5/6 chance) and maximum 6+ (1/6 chance)
	finalTargetRoll = math.Max(2.0, math.Min(6.0, finalTargetRoll))

	// Chance to wound (base)
	// (7 - finalTargetRoll) / 6.0
	woundChance := (7.0 - finalTargetRoll) / 6.0
	missChance := 1.0 - woundChance

	// 3. Process Rerolls
	if rerollType == damagerequest.RerollOnes {
		// Reroll of 1 (1/6) on a wound with woundChance
		rerollChance := oneSixth
		woundChance += rerollChance * woundChance
	} else if rerollType == damagerequest.RerollFail {
		// Reroll of a miss (missChance) on a wound with woundChance
		woundChance += missChance * woundChance
	}

	// 4. Process [DEVASTATING WOUNDS]
	devastatingWoundChance := 0.0
	if devastatingWounds {
		// Base chance for Devastating Wound (a roll of 6)
		devastatingWoundChance = oneSixth

		if rerollType == damagerequest.RerollOnes {
			// Reroll of 1 into a 6: (1/6) * (1/6)
			devastatingWoundChance += oneSixth * oneSixth
		} else if rerollType == damagerequest.RerollFail {
			// Reroll of a miss (missChance) into a 6 (1/6)
			// Probability: (1 - woundChance) * (1/6)
			devastatingWoundChance += missChance * oneSixth
		}

		// Devastating Wounds are excluded from normal wounds
		woundChance -= devastatingWoundChance
		// Constraint: woundChance must not be negative
		woundChance = math.Max(0.0, woundChance)
	}

	return woundChance, devastatingWoundChance
}
