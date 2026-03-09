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

// CalculateDamageDistribution parses a damage string and returns a probability map of outcomes.
//
// Arguments:
// damageString: e.g., "d6", "2d6", "d3+1", "3".
// feelNoPain: Optional pointer to FNP value (e.g., 5 for 5+). nil if none.
//
// Returns:
// map[int]float64: Mapping of DamageAmount -> Probability (0.0 to 1.0).
func _calculateDamageDistribution(damage DiceRoll, feelNoPain *int) map[int]float64 {
	baseDist := generateDiceDistribution(damage)

	if feelNoPain == nil {
		return baseDist
	}

	return applyFeelNoPain(baseDist, *feelNoPain)
}

// This replaces the string-parsing version with a direct mathematical approach.
func generateDiceDistribution(d DiceRoll) map[int]float64 {
	// Base case: If there are no dice to roll (e.g., flat damage "3"),
	// start with 100% probability at 0, then add the modifier.
	if d.Count <= 0 || d.Sides <= 0 {
		return map[int]float64{applyDamageFloor(d.Modifier): 1.0}
	}

	// 1. Initialize distribution with the first die (1 to Sides)
	currentDist := make(map[int]float64)
	prob := 1.0 / float64(d.Sides)
	for i := 1; i <= d.Sides; i++ {
		currentDist[i] = prob
	}

	// 2. Convolve for additional dice (O(Count * Sides * Outcomes))
	// Example: turning 1d6 distribution into 2d6, then 3d6...
	for i := 1; i < d.Count; i++ {
		newDist := make(map[int]float64)
		for valA, probA := range currentDist {
			for valDie := 1; valDie <= d.Sides; valDie++ {
				newDist[valA+valDie] += probA * prob
			}
		}
		currentDist = newDist
	}

	// 3. Apply Modifier and Damage Floor
	// In 40k, damage/attacks generally cannot be modified below 1.
	finalDist := make(map[int]float64)
	for val, p := range currentDist {
		result := applyDamageFloor(val + d.Modifier)
		finalDist[result] += p
	}

	return finalDist
}

// applyFeelNoPain applies the Binomial Distribution logic.
func applyFeelNoPain(baseDist map[int]float64, fnpVal int) map[int]float64 {
	fnpDist := make(map[int]float64)

	// Chance to SAVE a point of damage
	// FNP 5+ -> succeeds on 5, 6 (2/6)
	pSave := 0.0
	if fnpVal <= 6 && fnpVal >= 2 {
		pSave = (7.0 - float64(fnpVal)) / 6.0
	} else if fnpVal <= 1 {
		pSave = 1.0 // Auto pass
	}
	// Note: If fnpVal >= 7, pSave is 0.

	pFail := 1.0 - pSave

	// Iterate over every possible incoming damage amount
	for incomingDmg, incomingProb := range baseDist {
		// For a specific amount of damage 'n', the actual damage taken 'k'
		// follows a Binomial Distribution B(n, pFail).
		// k = number of failed saves.
		for k := 0; k <= incomingDmg; k++ {
			// Binomial Probability: P(X=k) = nCk * p^k * (1-p)^(n-k)
			// p = pFail (chance to take damage)
			combinatorics := float64(nCr(incomingDmg, k))
			probKFailed := combinatorics * math.Pow(pFail, float64(k)) * math.Pow(pSave, float64(incomingDmg-k))

			// Add weighted probability to the final map
			fnpDist[k] += incomingProb * probKFailed
		}
	}

	return fnpDist
}

// nCr calculates combinations (n choose k).
// Uses basic multiplicative formula to avoid huge factorials.
func nCr(n, k int) int {
	if k < 0 || k > n {
		return 0
	}
	if k == 0 || k == n {
		return 1
	}
	if k > n/2 {
		k = n - k
	}
	res := 1
	for i := 1; i <= k; i++ {
		res = res * (n - i + 1) / i
	}
	return res
}
