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
	"regexp"
	"strconv"
	"strings"
)

// CalculateDamageDistribution parses a damage string and returns a probability map of outcomes.
//
// Arguments:
// damageString: e.g., "d6", "2d6", "d3+1", "3".
// feelNoPain: Optional pointer to FNP value (e.g., 5 for 5+). nil if none.
//
// Returns:
// map[int]float64: Mapping of DamageAmount -> Probability (0.0 to 1.0).
func _calculateDamageDistribution(damageString string, feelNoPain *int) map[int]float64 {
	baseDist := parseAndCalculateBaseDamage(damageString)

	if feelNoPain == nil {
		return baseDist
	}

	return applyFeelNoPain(baseDist, *feelNoPain)
}

// parseAndCalculateBaseDamage handles the regex parsing and dice math.
// Unlike the Python version, this correctly calculates bell curves for multi-dice (e.g. 2d6).
func parseAndCalculateBaseDamage(damageString string) map[int]float64 {
	dist := make(map[int]float64)
	normalized := strings.ToLower(strings.TrimSpace(damageString))

	// Regex to capture: (Count)d(Faces)+(Modifier)
	// Examples: "d6", "2d6", "2d6+1", "d3+2"
	re := regexp.MustCompile(`^(\d*)d(\d+)\s*\+?\s*(\d*)$`)
	matches := re.FindStringSubmatch(normalized)

	// Case 1: It's a static number (e.g., "3")
	if matches == nil {
		val, err := strconv.Atoi(normalized)
		if err != nil {
			// Fallback for bad input, similar to Python version
			dist[0] = 1.0
			return dist
		}
		dist[val] = 1.0
		return dist
	}

	// Case 2: It's a dice string
	countStr, facesStr, modStr := matches[1], matches[2], matches[3]

	// Parse Count (default to 1 if empty, e.g. "d6")
	count := 1
	if countStr != "" {
		count, _ = strconv.Atoi(countStr)
	}

	// Parse Faces (required)
	faces, _ := strconv.Atoi(facesStr)

	// Parse Modifier (default to 0)
	modifier := 0
	if modStr != "" {
		modifier, _ = strconv.Atoi(modStr)
	}

	// Logic for Probability Distribution
	if count == 0 {
		dist[modifier] = 1.0
		return dist
	}

	// Start with one die
	// Probability of rolling x on 1dFaces is 1/Faces
	currentDist := make(map[int]float64)
	prob := 1.0 / float64(faces)
	for i := 1; i <= faces; i++ {
		currentDist[i] = prob
	}

	// Convolve for multiple dice (e.g., combining distributions for 2d6)
	// We repeat the convolution 'count - 1' times.
	for i := 1; i < count; i++ {
		newDist := make(map[int]float64)
		for valA, probA := range currentDist {
			// Convolve with a single fresh die (1 to Faces)
			for valB := 1; valB <= faces; valB++ {
				// Probability of this combination is ProbA * ProbB
				// Resulting damage is ValA + ValB
				newDist[valA+valB] += probA * prob
			}
		}
		currentDist = newDist
	}

	// Apply Modifier to the final distribution
	finalDist := make(map[int]float64)
	for k, v := range currentDist {
		finalDist[k+modifier] = v
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
