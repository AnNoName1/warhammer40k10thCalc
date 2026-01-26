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
)

// UnitState tracks the health of the defending unit during sequential damage allocation.
// This is the "State" in our Markov Chain, representing a snapshot of unit health.
type UnitState struct {
	Killed    int // Number of models completely removed from the unit
	CurrentHP int // Remaining HP of the specific model currently taking damage
}

type DamageCalculatorImpl struct{}

// CalculateDamageCore is the main entry point for the probability engine.
// It uses a "Transition Map" algorithm: instead of simulating dice rolls, it
// calculates the mathematical probability of every possible branch in the
// attack sequence (Hit -> Wound -> Save -> Damage).
// getBinomialVector calculates the binomial distribution for n trials with probability p.
// Returns a vector where index i is the probability of exactly i successes.

func (d *DamageCalculatorImpl) CalculateDamageCore(req CombatSimulationRequest) (SimulationResult, error) {
	if err := d.Sanitize(&req); err != nil {
		return SimulationResult{}, err
	}

	// ── 1. Attack count distribution ───────────────────────────────
	attackCountDist := CalculateAttackDistribution(
		req.Attacker.Attacks, req.Attacker.Count, req.Attacker.Blast, *req.Target.Count,
	)

	// ── 2. Expected hits per single attack roll (split into normal and lethal) ──
	expectedNormalHits, expectedLethalHits := CalculateHitExpected(
		req.Attacker.BS,
		req.Settings.HitReroll,
		req.Settings.HitModifier,
		req.Attacker.LethalHits,
		req.Attacker.SustainedHits,
		req.Settings.CriticalHitThreshold,
	)
	expectedTotalHits := expectedNormalHits + expectedLethalHits

	// ── 3. Wound probabilities (per single wound roll / hit) ───────
	probNormalWound, probDevastatingWound := CalculateWoundProbability(
		req.Attacker.Strength,
		req.Target.Toughness,
		req.Settings.WoundReroll,
		req.Settings.WoundModifier,
		req.Attacker.DevastatingWounds,
		req.Settings.CriticalWoundThreshold,
	)

	// ── 4. Save failure probability (after AP, invuln, cover, etc) ─
	// This is only for armor/invuln saves; FNP is applied later in damage distribution
	probSaveFailed := CalculateFailedSaveProbability(
		req.Attacker.AP,
		req.Target.Save,
		req.Target.Invulnerable,
		req.Settings.SaveModifier,
		req.Target.HasCover,
		req.Settings.SaveReroll,
	)

	// ── 5. Compute max possible attacks for sizing ──────────────────
	maxAttacks := 0
	for n := range attackCountDist {
		if n > maxAttacks {
			maxAttacks = n
		}
	}
	max := maxAttacks // shorthand

	// ── 6. Joint distribution: normal wounds before save × devastating wounds ──
	// Using binomial approximation with correlation via conditional
	jointNormalBeforeSave_Dev := make([][]float64, max+1)
	for i := range jointNormalBeforeSave_Dev {
		jointNormalBeforeSave_Dev[i] = make([]float64, max+1)
	}

	// ── 7. Hits distribution (total, for reporting) ────────────────
	finalHitsDist := make([]float64, max+1)

	for attacks, attackProb := range attackCountDist {
		if attackProb < 1e-15 {
			continue
		}

		// Hits (using total expected)
		hitDist := getBinomialVector(attacks, expectedTotalHits)
		for h, p := range hitDist {
			finalHitsDist[h] += p * attackProb
		}

		// ── Joint normal-before-save & devastating ──────────────────
		// First, expected normal wounds before save from normal hits + lethal (lethal become normal wounds)
		pNormalWoundBeforeSave := (expectedNormalHits * probNormalWound) + expectedLethalHits
		pDevWound := expectedNormalHits * probDevastatingWound // lethal usually not devastating

		pTotalSuccess := pNormalWoundBeforeSave + pDevWound
		pNeither := 1.0 - pTotalSuccess/float64(attacks) // approximate normalization

		if pTotalSuccess > float64(attacks) || pNeither < 0 {
			// Invalid - cap or log error
			pTotalSuccess = float64(attacks)
			pNeither = 0
		}

		// Binomial for normal before save
		normalDist := getBinomialVector(attacks, pNormalWoundBeforeSave/float64(attacks))
		for normal, pNormal := range normalDist {
			if pNormal < 1e-15 {
				continue
			}

			remaining := attacks - normal // approximate remaining trials
			pDevCond := 0.0
			if remaining > 0 && pNeither+pDevWound > 0 {
				pDevCond = pDevWound / (pNeither + pDevWound)
			}

			devDist := getBinomialVector(remaining, pDevCond)
			for dev, pDev := range devDist {
				if pDev < 1e-15 {
					continue
				}
				jointNormalBeforeSave_Dev[normal][dev] += attackProb * pNormal * pDev
			}
		}
	}

	// ── 8. Total wounds dist (for reporting, sum nw + dw) ──────────
	totalWoundsDist := make([]float64, max+1)
	for nw := 0; nw <= max; nw++ {
		for dw := 0; dw <= max-nw; dw++ {
			p := jointNormalBeforeSave_Dev[nw][dw]
			if p > 1e-15 {
				totalWoundsDist[nw+dw] += p
			}
		}
	}

	// ── 9. Marginal devastating dist ───────────────────────────────
	devastatingDist := make([]float64, max+1)
	for dw := 0; dw <= max; dw++ {
		colSum := 0.0
		for nw := 0; nw <= max; nw++ {
			colSum += jointNormalBeforeSave_Dev[nw][dw]
		}
		devastatingDist[dw] = colSum
	}

	// ── 10. Final unsaved dist (unsaved normal + devastating) ──────
	// Compute by looping over joint
	finalUnsavedDist := make([]float64, max+1)
	for nw := 0; nw <= max; nw++ {
		for dw := 0; dw <= max; dw++ {
			pJoint := jointNormalBeforeSave_Dev[nw][dw]
			if pJoint < 1e-15 {
				continue
			}
			unsavedNormal := getBinomialVector(nw, probSaveFailed)
			for u, pU := range unsavedNormal {
				if pU < 1e-15 {
					continue
				}
				totalUnsaved := u + dw
				finalUnsavedDist[totalUnsaved] += pJoint * pU
			}
		}
	}

	// ── 11. Damage allocation ──────────────────────────────────────
	finalDestroyedDist := make(map[int]float64)
	dmgWithFnp := _calculateDamageDistribution(req.Attacker.Damage, req.Target.FeelNoPain) // FNP applied here
	dmgNoFnp := _calculateDamageDistribution(req.Attacker.Damage, req.Target.FeelNoPain)   // FNP for devastating - latest rules

	for nw := 0; nw <= max; nw++ {
		for dw := 0; dw <= max; dw++ {
			pJoint := jointNormalBeforeSave_Dev[nw][dw]
			if pJoint < 1e-12 {
				continue
			}
			unsavedNormalDist := getBinomialVector(nw, probSaveFailed)
			for u, pU := range unsavedNormalDist {
				if pU < 1e-12 {
					continue
				}
				weight := pJoint * pU
				killedMap := resolveDamageSequentialSplit(
					u, dw,
					dmgWithFnp,
					dmgNoFnp,
					req.Target.WoundsPerModel,
					*req.Target.Count,
				)
				for k, pk := range killedMap {
					finalDestroyedDist[k] += weight * pk
				}
			}
		}
	}

	return formatResponse(
		vectorToMap(finalHitsDist),
		vectorToMap(totalWoundsDist),
		vectorToMap(finalUnsavedDist),
		finalDestroyedDist,
	), nil
}

// --- HELPER FUNCTIONS ---
func getBinomialVector(n int, p float64) []float64 {
	dist := make([]float64, n+1)
	dist[0] = 1.0
	if p <= 0 {
		return dist
	}
	if p >= 1.0 {
		dist[0] = 0
		dist[n] = 1.0
		return dist
	}

	q := 1.0 - p
	for i := 1; i <= n; i++ {
		// Update in place from right to left to maintain O(n) space
		for j := i; j > 0; j-- {
			dist[j] = dist[j]*q + dist[j-1]*p
		}
		dist[0] *= q
	}
	return dist
}

// convolve performs a discrete convolution of two probability distributions.
func convolve(a, b []float64) []float64 {
	if len(a) == 0 {
		return b
	}
	if len(b) == 0 {
		return a
	}
	res := make([]float64, len(a)+len(b)-1)
	for i, va := range a {
		if va < 1e-18 {
			continue
		}
		for j, vb := range b {
			res[i+j] += va * vb
		}
	}
	return res
}

// vectorToMap translates the internal vector back to the map used in the output contract.
func vectorToMap(vec []float64) map[int]float64 {
	res := make(map[int]float64)
	for i, p := range vec {
		if p > 1e-15 {
			res[i] = p
		}
	}
	return res
}

// In resolveDamageSequentialSplit: ensure NO spill for BOTH (your original applyWoundsLinear with spills=false for both)
func resolveDamageSequentialSplit(
	nNorm, nMortal int,
	normDmgDist, mortalDmgDist map[int]float64,
	maxHP, totalModels int,
) map[int]float64 {
	maxPossible := totalModels * maxHP
	states := make([]float64, maxPossible+1)
	states[maxPossible] = 1.0

	// Apply normal unsaved (FNP already accounted in dist, no spill)
	for i := 0; i < nNorm; i++ {
		states = applyWoundsLinear(states, normDmgDist, maxHP, false) // no spill
	}
	// Apply devastating (ignore FNP, no spill)
	for i := 0; i < nMortal; i++ {
		states = applyWoundsLinear(states, mortalDmgDist, maxHP, false) // IMPORTANT: false = no spill
	}

	final := make(map[int]float64)
	for remaining, prob := range states {
		if prob < 1e-12 {
			continue
		}
		modelsLeft := (remaining + maxHP - 1) / maxHP
		killed := totalModels - modelsLeft
		final[killed] += prob
	}
	return final
}

// applyWounds is the core state-transition engine.
// It iterates through every possible HP state and applies a single attack's damage distribution.
func applyWoundsLinear(states []float64, dmgDist map[int]float64, maxHP int, spills bool) []float64 {
	next := make([]float64, len(states))

	for currentWounds, stateProb := range states {
		if stateProb <= 0 {
			continue
		}
		if currentWounds == 0 { // Unit already dead
			next[0] += stateProb
			continue
		}

		for dVal, dProb := range dmgDist {
			p := stateProb * dProb

			var newWounds int
			if spills {
				// Mortal Wounds: Simple subtraction, floor at 0
				newWounds = currentWounds - dVal
				if newWounds < 0 {
					newWounds = 0
				}
			} else {
				// Normal Damage: Cannot lose more than the current model's remaining HP
				// Current model HP is: ((currentWounds-1) % maxHP) + 1
				currentModelHP := ((currentWounds - 1) % maxHP) + 1
				damageDealt := dVal
				if damageDealt > currentModelHP {
					damageDealt = currentModelHP
				}
				newWounds = currentWounds - damageDealt
			}
			next[newWounds] += p
		}
	}
	return next
}

// formatResponse calculates final averages and builds the structured response for the client.
func formatResponse(hits, wounds, pens, killed map[int]float64) SimulationResult {
	avgK := 0.0
	for k, v := range killed {
		avgK += float64(k) * v
	}
	avgH := 0.0
	for k, v := range hits {
		avgH += float64(k) * v
	}

	return SimulationResult{
		//mapping here
		AverageHits:      avgH,
		AverageDestroyed: avgK,
		HitDist:          hits,
		WoundDist:        wounds,
		PenDist:          pens,
		DestroyedDist:    killed,
	}
}

// Used solely for Sanity Checking and allocation limits.
// GetMaxFromDice returns the maximum possible value a DiceRoll can produce.
func GetMaxFromDice(d DiceRoll) int {
	// If it's a fixed value, d.Count and d.Sides are 0, so it returns d.Modifier.
	// If it's 2d6+3, it returns (2*6) + 3 = 15.
	max := (d.Count * d.Sides) + d.Modifier

	// We apply the same floor logic used in math to ensure our "worst case"
	// matches the engine's behavior.
	if max < 1 {
		return 1
	}
	return max
}

func (d *DamageCalculatorImpl) Sanitize(req *CombatSimulationRequest) error {
	// 1. Critical Thresholds (0 is uninitialized/impossible, treat as 6)
	if req.Settings.CriticalHitThreshold < 2 || req.Settings.CriticalHitThreshold > 6 {
		req.Settings.CriticalHitThreshold = 6
	}
	if req.Settings.CriticalWoundThreshold < 2 || req.Settings.CriticalWoundThreshold > 6 {
		req.Settings.CriticalWoundThreshold = 6
	}

	// 2. Ballistic Skill (Minimum 2+ per core rules, rolls of 1 always fail)
	if !req.Attacker.Torrent {
		if req.Attacker.BS < 2 {
			req.Attacker.BS = 2
		} else if req.Attacker.BS > 6 {
			req.Attacker.BS = 6
		}
	}

	// 3. Save Floor (1+ saves are treated as 2+; natural 1 always fails)
	if req.Target.Save < 2 {
		req.Target.Save = 2
	}

	// 4. Pointer-based defaults (if present)
	if req.Target.Invulnerable != nil && (*req.Target.Invulnerable < 2) {
		*req.Target.Invulnerable = 2
	}

	// 5. Get the "Ceiling" of the attack sequence
	maxAttacks := GetMaxFromDice(req.Attacker.Attacks) * req.Attacker.Count
	maxDamage := GetMaxFromDice(req.Attacker.Damage)

	devastatingDoSpillover := false

	// 6. Resolve the Target Count (Business Logic for "Infinite")
	var effectiveTargetCount int

	if req.Target.Count == nil {
		// "Infinite" logic: Determine how many models can actually be affected.
		if req.Attacker.DevastatingWounds && devastatingDoSpillover {
			// Spillover: Total pool of damage matters.
			totalPossibleDamage := maxAttacks * maxDamage
			effectiveTargetCount = (totalPossibleDamage / req.Target.WoundsPerModel) + 1
		} else {
			// No spillover: Cannot kill more models than there are attacks.
			effectiveTargetCount = maxAttacks
		}

		// Safety cap to prevent DOS (Business constraint).
		if effectiveTargetCount > 200 {
			effectiveTargetCount = 200
		}
		// Go's escape analysis will move effectiveTargetCount to the heap.
		req.Target.Count = &effectiveTargetCount
	} else {
		effectiveTargetCount = *req.Target.Count
	}

	// 7. Blast Correction (Feedback Loop)
	// If Blast is on, the target count actually INCREASES the number of attacks.
	if req.Attacker.Blast {
		bonus := effectiveTargetCount / 5
		// Update the actual domain object so the calculator sees it
		req.Attacker.Attacks.Modifier += bonus
		// Recalculate maxAttacks for the complexity score
		maxAttacks = GetMaxFromDice(req.Attacker.Attacks) * req.Attacker.Count
	}

	// 8. Calculate Final Complexity Score
	// State Space = Width (Models * HP). MaxAttacks = Depth.
	stateSpace := effectiveTargetCount * req.Target.WoundsPerModel
	// If each attack requires iterating over the state space:
	score := maxAttacks * (stateSpace * stateSpace) // Or a lower multiplier like * 10

	// 9. The Threshold Check
	const Threshold = 25_000_000
	if score > Threshold {
		return fmt.Errorf("simulation complexity %d exceeds limit %d. (Targeting ~%d models with ~%d max attacks)",
			score, Threshold, effectiveTargetCount, maxAttacks)
	}

	return nil
}
