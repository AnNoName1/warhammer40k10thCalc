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

	bocModifier := _getBenefitOfCoverModifier(save, ap, hasCover)
	armorSaveTarget := modifiedArmorSaveTarget(save, ap, saveModifier, bocModifier)
	finalTarget := betterSaveTarget(armorSaveTarget, invulnerable)

	finalTarget, autoFail := clampSaveTarget(finalTarget)
	if autoFail {
		return 1.0
	}

	passChance := chanceOfRollingAtLeast(float64(finalTarget))
	failChance := 1.0 - passChance

	switch saveReroll {
	case RerollOnes:
		// Only a natural 1 can be rerolled.
		failChance = applyRerollReduction(failChance, passChance, oneSixth)
	case RerollFail:
		// Every failed roll can be rerolled.
		failChance = applyRerollReduction(failChance, passChance, failChance)
	}

	return math.Max(0.0, failChance)
}

// modifiedArmorSaveTarget applies the general save modifier and the Benefit
// of Cover bonus to the defender's armor save.
func modifiedArmorSaveTarget(save, ap, saveModifier, bocModifier int) int {
	totalModifier := saveModifier + bocModifier
	return save + ap - totalModifier
}

// betterSaveTarget returns the lower (easier) of the armor save target and
// the invulnerable save, if one is present; a lower target is always at
// least as good, since invulnerable saves ignore AP and modifiers.
func betterSaveTarget(armorTarget int, invulnerable *int) int {
	if invulnerable != nil && *invulnerable < armorTarget {
		return *invulnerable
	}
	return armorTarget
}

// clampSaveTarget applies the core rules floor and ceiling: a natural 1
// always fails, so no save can need better than 2+; nothing beyond 6+ is
// rollable, so any higher requirement auto-fails the save entirely.
func clampSaveTarget(target int) (clamped int, autoFail bool) {
	if target < 2 {
		target = 2
	}
	if target > 6 {
		return 0, true
	}
	return target, false
}

// applyRerollReduction subtracts the chance that a reroll turns a fail into
// a pass: a second attempt happens with probability retryChance, so
// retryChance*passChance of the original failures are saved.
func applyRerollReduction(failChance, passChance, retryChance float64) float64 {
	return failChance - retryChance*passChance
}
