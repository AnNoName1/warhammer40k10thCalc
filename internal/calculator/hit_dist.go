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

// HitOutcome represents the discrete result of a single die roll.
// Example: A Critical Hit with Sustained 2 and Lethal Hits would be:
// {Lethal: 1, Normal: 2} (1 auto-wound, 2 extra hits to roll for)
type HitOutcome struct {
	NormalHits int
	LethalHits int
}

// CalculateSingleHitDistribution returns the PMF for a single attack die.
func CalculateSingleHitDistribution(
	bs int,
	rerollType RerollType,
	hitModifier int,
	lethalHits bool,
	sustainedHits int,
	criticalThreshold int,
) map[HitOutcome]float64 {

	dist := make(map[HitOutcome]float64)

	// 1. Determine Probability of each Face (1-6) considering Rerolls
	// We calculate the weight of each face out of 36 (for clean math) or float.
	// Basic P(x) = 1/6.
	faceProbs := resolveRerolls(bs, hitModifier, rerollType)

	for face := 1; face <= 6; face++ {
		prob := faceProbs[face]
		if prob == 0 {
			continue
		}

		outcome := resolveDieOutcome(face, bs, hitModifier, criticalThreshold, lethalHits, sustainedHits)
		dist[outcome] += prob
	}

	return dist
}

// Determine what happens on a specific physical die roll.
func resolveDieOutcome(face int, bs int, mod int, critThreshold int, lethal bool, sustained int) HitOutcome {
	// Critical Hits are usually based on Unmodified rolls in 10th
	isCrit := face >= critThreshold // e.g., 6 >= 6

	modRoll := clampToD6Range(face + mod)

	// A natural 1 always fails and a natural 6 is a Critical Hit (auto-success);
	// Criticals override the modified-roll check entirely.
	isHit := isCrit || (modRoll >= bs && face != 1)

	if !isHit {
		return HitOutcome{NormalHits: 0, LethalHits: 0}
	}

	outcome := HitOutcome{}

	if isCrit && lethal {
		outcome.LethalHits = 1
	} else {
		outcome.NormalHits = 1
	}

	// Sustained hits are treated as Normal Hits; they don't trigger Lethals recursively.
	if isCrit && sustained > 0 {
		outcome.NormalHits += sustained
	}

	return outcome
}

// clampToD6Range clamps a modified roll into the valid [1, 6] die range.
func clampToD6Range(v int) int {
	if v > 6 {
		return 6
	}
	if v < 1 {
		return 1
	}
	return v
}

// Reroll logic: computes P(Face) for each die face under rerollType.
func resolveRerolls(bs, mod int, reroll RerollType) map[int]float64 {
	probs := make(map[int]float64)
	base := 1.0 / 6.0

	for i := 1; i <= 6; i++ {
		probs[i] = base
	}

	if reroll == RerollNone {
		return probs
	}

	rerollPool := 0.0
	shouldReroll := func(face int) bool {
		if reroll == RerollOnes {
			return face == 1
		}
		if reroll == RerollFail {
			// Fail check: Nat 1 or Modified < BS
			modRoll := clampToD6Range(face + mod)
			return face == 1 || modRoll < bs
		}
		return false
	}

	// Harvest probability from rerolled faces
	for i := 1; i <= 6; i++ {
		if shouldReroll(i) {
			rerollPool += probs[i]
			probs[i] = 0
		}
	}

	// Distribute Reroll Pool evenly across all faces (1/6 per face per reroll mass)
	perFaceAdd := rerollPool / 6.0
	for i := 1; i <= 6; i++ {
		probs[i] += perFaceAdd
	}

	return probs
}
