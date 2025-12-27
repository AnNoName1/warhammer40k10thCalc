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

// _calculateFailedSaveProbability calculates the probability of a defender failing their saving throw
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
func _calculateFailedSaveProbability(ap int, save int, invulnerable *int, saveModifier int) float64 {
	// 1. Calculate the Modified Armor Save
	// AP makes the save harder (adds to the target number).
	// Modifiers (like cover) make the save easier (subtract from the target number).
	armorSaveTarget := save + ap - saveModifier

	// 2. Determine the "Best" Save Target
	// Start with the armor save as the best option
	finalTarget := armorSaveTarget

	// If an invulnerable save exists, check if it is better (lower) than the armor save.
	// Note: Based on Test Case 8, Invulnerable saves are NOT affected by the saveModifier.
	if invulnerable != nil {
		invulnTarget := *invulnerable
		if invulnTarget < finalTarget {
			finalTarget = invulnTarget
		}
	}

	// 3. Apply the "Rule of 1" and Logic Caps
	// A roll of 1 always fails, so the effective target cannot be lower than 2.
	if finalTarget < 2 {
		finalTarget = 2
	}

	// 4. Calculate Fail Chance
	// If the target is greater than 6, it is impossible to pass on a D6.
	if finalTarget > 6 {
		return 1.0
	}

	// Probability of Passing = (Sides of Die - Target + 1) / Sides of Die
	// Simplified: (7 - Target) / 6
	passChance := (7.0 - float64(finalTarget)) / 6.0

	return 1.0 - passChance
}
