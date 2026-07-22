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
	const oneSixth = 1.0 / 6.0

	finalTargetRoll := clampWoundTarget(woundRollTarget(s, t), woundModifier)
	woundChance := chanceOfRollingAtLeast(finalTargetRoll)

	critChance := chanceOfRollingAtLeast(float64(sanitizeCriticalThreshold(CriticalWoundThreshold)))

	// Critical Wounds are always successful: if critChance is higher than
	// woundChance (e.g., Anti-2+ vs T12), woundChance is elevated to match.
	if critChance > woundChance {
		woundChance = critChance
	}

	missChance := 1.0 - woundChance

	switch rerollType {
	case RerollOnes:
		// Only a natural 1 can be rerolled.
		woundChance = applyRerollBonus(woundChance, oneSixth)
		critChance = applyRerollBonus(critChance, oneSixth)
	case RerollFail:
		// Every failed roll can be rerolled.
		woundChance = applyRerollBonus(woundChance, missChance)
		critChance = applyRerollBonus(critChance, missChance)
	}

	woundChance, devastatingWoundChance := splitDevastatingWounds(woundChance, critChance, devastatingWounds)

	return woundChance, devastatingWoundChance
}

// woundRollTarget returns the unmodified D6 roll needed to wound. Per the
// core rules Strength-vs-Toughness table: S >= 2T wounds on 2+, S > T on
// 3+, S == T on 4+, S < T on 5+, S <= T/2 on 6+.
func woundRollTarget(s, t int) int {
	switch {
	case s*2 <= t:
		return 6
	case s < t:
		return 5
	case s >= t*2:
		return 2
	case s > t:
		return 3
	default:
		return 4
	}
}

// clampWoundTarget applies the wound-roll modifier and clamps to the core
// rules' legal target range: no roll can need better than 2+ or worse than
// 6+ to succeed.
func clampWoundTarget(targetRoll, modifier int) float64 {
	v := float64(targetRoll) - float64(modifier)
	return math.Max(2.0, math.Min(6.0, v))
}

// chanceOfRollingAtLeast returns P(roll >= target) on a fair D6.
func chanceOfRollingAtLeast(target float64) float64 {
	return (7.0 - target) / 6.0
}

// sanitizeCriticalThreshold resets an out-of-range threshold to the core
// rules default: critical only on a natural 6.
func sanitizeCriticalThreshold(v int) int {
	if v < 2 || v > 6 {
		return 6
	}
	return v
}

// applyRerollBonus adds the probability of succeeding on a reroll: a second
// attempt happens with probability retryChance, and succeeds at the same
// rate as the original roll, so the bonus is retryChance*chance.
func applyRerollBonus(chance, retryChance float64) float64 {
	return chance + retryChance*chance
}

// splitDevastatingWounds separates the devastating-wound share out of the
// total wound chance. Per the core rules, [DEVASTATING WOUNDS] wounds
// bypass the save roll entirely, so they must be resolved separately from
// ordinary wounds rather than folded into woundChance.
func splitDevastatingWounds(woundChance, critChance float64, devastatingWounds bool) (normal, devastating float64) {
	if !devastatingWounds {
		return woundChance, 0.0
	}
	return math.Max(0.0, woundChance-critChance), critChance
}
