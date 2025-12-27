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
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// compiled regex for performance.
// We use ^ and $ to ensure we match the full string (e.g., prevent "10 models" matching as "1").
var diceRegex = regexp.MustCompile(`(?i)^(\d*)d(\d+)\s*\+?\s*(\d*)$`)

// CalculateAttackDistribution parses a string like "2d6+1" or "4"
// and returns a probability map [attacks]probability.
func CalculateAttackDistribution(attackStr string) (map[int]float64, error) {
	attackStr = strings.TrimSpace(attackStr)

	// 1. Try to parse as a simple fixed number (e.g., "4")
	if val, err := strconv.Atoi(attackStr); err == nil {
		return map[int]float64{val: 1.0}, nil
	}

	// 2. Try to parse as Dice format (e.g., "2d6+1", "d6", "D3")
	matches := diceRegex.FindStringSubmatch(attackStr)
	if matches == nil {
		// If it's neither an Int nor a Dice string, return an error.
		// Note: The Python code printed a warning and returned 0.
		// In Go, returning an error is safer so the Handler can tell the user.
		return nil, fmt.Errorf("invalid attack format: '%s'", attackStr)
	}

	// Parse Groups from Regex
	// Group 1: Number of dice (optional, default 1)
	numDice := 1
	if matches[1] != "" {
		var err error
		numDice, err = strconv.Atoi(matches[1])
		if err != nil {
			return nil, fmt.Errorf("invalid dice count: %w", err)
		}
	}

	// Group 2: Die Type (required)
	dieType, err := strconv.Atoi(matches[2])
	if err != nil {
		return nil, fmt.Errorf("invalid die type: %w", err)
	}

	// Group 3: Modifier (optional, default 0)
	modifier := 0
	if matches[3] != "" {
		var err error
		modifier, err = strconv.Atoi(matches[3])
		if err != nil {
			return nil, fmt.Errorf("invalid modifier: %w", err)
		}
	}

	// 3. Calculate Distribution (Convolution Logic)
	// Start with a distribution of 0 attacks with probability 1.0
	currentDist := map[int]float64{0: 1.0}

	// For each die being rolled...
	for i := 0; i < numDice; i++ {
		newDist := make(map[int]float64)
		probPerFace := 1.0 / float64(dieType)

		// For every existing possible sum...
		for currentSum, currentProb := range currentDist {
			// Add the result of the new die roll (1 to dieType)
			for roll := 1; roll <= dieType; roll++ {
				newDist[currentSum+roll] += currentProb * probPerFace
			}
		}
		currentDist = newDist
	}

	// 4. Apply the modifier (e.g., +1)
	finalDist := make(map[int]float64)
	for sumVal, prob := range currentDist {
		finalDist[sumVal+modifier] = prob
	}

	return finalDist, nil
}
