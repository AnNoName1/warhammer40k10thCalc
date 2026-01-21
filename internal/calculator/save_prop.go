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

// CalculateFailedSaveProbability calculates the probability of a defender failing their saving throw
// against a single successful wound, incorporating 10th edition capping rules.
//
// The rule states that the final required save roll cannot be modified to be better than 2+
// or worse than 6+ (i.e., a roll of 7+ is always a fail).
//
// Args:
// ap (int): Armor Penetration of the attack.
// save (int): Defender's normal Save characteristic (e.g., 3 for 3+).
// invulnerable (*int): Defender's Invulnerable Save characteristic (e.g., 4 for 4+). Optional (nil if not present).
// saveModifier (int): Modifier applied to the Save roll (e.g., Cover is +1).
//
// Returns:
// float64: The probability of failing the save.

// _applyBenefitOfCover determines if the +1 to save applies.
// "Models with a Save characteristic of 3+ or better cannot have the Benefit of Cover
// against attacks with an Armour Penetration characteristic of 0."

import "math"

func _getBenefitOfCoverModifier(save int, ap int, hasCover bool) int {
	if !hasCover {
		return 0
	}

	// Rule: If Save is 3+ or better (3, 2) and AP is 0, no bonus.
	if save <= 3 && ap == 0 {
		return 0
	}

	return 1
}

func CalculateFailedSaveProbability(ap int, save int, invulnerable *int, saveModifier int,
	hasCover bool, saveReroll RerollType) float64 {

	const oneSixth = 1.0 / 6.0

	// 1. Calculate the Benefit of Cover specifically
	bocModifier := _getBenefitOfCoverModifier(save, ap, hasCover)

	// 2. Calculate the Modified Armor Save
	// Total modifier is the general saveModifier + the BoC bonus
	totalModifier := saveModifier + bocModifier
	armorSaveTarget := save + ap - totalModifier

	// 3. Determine the "Best" Save Target
	finalTarget := armorSaveTarget
	if invulnerable != nil {
		invulnTarget := *invulnerable
		if invulnTarget < finalTarget {
			finalTarget = invulnTarget
		}
	}

	// 4. Apply Caps (1 always fails, cannot pass if > 6)
	if finalTarget < 2 {
		finalTarget = 2
	}

	if finalTarget > 6 {
		return 1.0 // 100% fail rate
	}

	// 5. Calculate Base Probabilities
	passChance := (7.0 - float64(finalTarget)) / 6.0
	failChance := 1.0 - passChance

	// 6. Handle Rerolls
	if saveReroll == RerollOnes {
		// If we roll a 1 (1/6), we retry and get another passChance
		failChance -= oneSixth * passChance
	} else if saveReroll == RerollFail {
		// If we fail (failChance), we retry and get another passChance
		failChance -= failChance * passChance
	}

	return math.Max(0.0, failChance)
}
