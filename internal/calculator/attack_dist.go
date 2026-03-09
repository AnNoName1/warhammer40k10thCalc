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

func CalculateAttackDistribution(
	attacks DiceRoll,
	attackerCount int,
	blast bool,
	targetCount int,
) map[int]float64 {

	// 1. Parse attack string into a PER-MODEL distribution
	perModelDist := getDiceDistribution(attacks)

	// 2. Apply Blast (still PER-MODEL)
	if blast {
		perModelDist = applyBlastModifier(perModelDist, targetCount)
	}

	// 3. Scale per-model distribution by attacker count
	unitDist := scaleByAttackerCount(perModelDist, attackerCount)

	return unitDist
}

func applyDamageFloor(value int) int {
	if value < 1 {
		return 1
	}
	return value
}

func getDiceDistribution(attacks DiceRoll) map[int]float64 {

	// Roll dice
	dist := rollDiceDistribution(attacks.Count, attacks.Sides)
	// Apply flat modifier AND damage floor (Damage cannot be < 1)
	finalDist := make(map[int]float64)
	for val, p := range dist {
		floored := applyDamageFloor(val + attacks.Modifier)
		finalDist[floored] += p
	}

	return finalDist
}

func rollDiceDistribution(numDice, dieType int) map[int]float64 {
	current := map[int]float64{0: 1.0}
	probPerFace := 1.0 / float64(dieType)

	for i := 0; i < numDice; i++ {
		next := make(map[int]float64)
		for sum, p := range current {
			for roll := 1; roll <= dieType; roll++ {
				next[sum+roll] += p * probPerFace
			}
		}
		current = next
	}

	return current
}

func applyBlastModifier(
	perModelDist map[int]float64,
	targetCount int,
) map[int]float64 {

	// Rule: +1 attack per 5 target models
	if targetCount < 5 {
		return perModelDist
	}

	blastBonus := targetCount / 5
	out := make(map[int]float64)

	for val, p := range perModelDist {
		out[val+blastBonus] += p
	}

	return out
}

func scaleByAttackerCount(
	perModelDist map[int]float64,
	count int,
) map[int]float64 {

	// Start with zero total attacks
	unitDist := map[int]float64{0: 1.0}

	// Convolve the per-model distribution `count` times
	for i := 0; i < count; i++ {
		next := make(map[int]float64)
		for total, pTotal := range unitDist {
			for perModel, pModel := range perModelDist {
				next[total+perModel] += pTotal * pModel
			}
		}
		unitDist = next
	}

	return unitDist
}
