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

// Validator defines the contract for complexity/safety checks.
type Validator func(*CombatSimulationRequest) error

type DamageCalculatorImpl struct {
	Validator Validator
}

// CalculateDamageCore is the main entry point for the probability engine.
// It uses a "Transition Map" algorithm: instead of simulating dice rolls, it
// calculates the mathematical probability of every possible branch in the
// attack sequence (Hit -> Wound -> Save -> Damage).
// getBinomialVector calculates the binomial distribution for n trials with probability p.
// Returns a vector where index i is the probability of exactly i successes.

func (d *DamageCalculatorImpl) CalculateDamageCore(req CombatSimulationRequest) (SimulationResult, error) {
	// 1. Hydrate (Mutate/Normalize State) - ALWAYS runs
	d.Hydrate(&req)

	// 2. Validate (Check Constraints) - Runs default unless overridden
	validate := d.Validator
	if validate == nil {
		validate = DefaultComplexityValidator
	}

	if err := validate(&req); err != nil {
		return SimulationResult{}, err
	}

	// ── 1. Attack count distribution ───────────────────────────────
	attackCountDist := CalculateAttackDistribution(
		req.Attacker.Attacks,
		req.Attacker.Count,
		req.Attacker.Blast,
		*req.Target.Count,
	)

	// ── 2. Single-attack hit outcome distribution (DISCRETE) ───────
	hitOutcomeDist := computeHitOutcomeDist(req)

	// ── 3. Wound probabilities (per wound roll) ────────────────────
	probNormalWound, probDevWound := CalculateWoundProbability(
		req.Attacker.Strength,
		req.Target.Toughness,
		req.Settings.WoundReroll,
		req.Settings.WoundModifier,
		req.Attacker.DevastatingWounds,
		req.Settings.CriticalWoundThreshold,
	)

	// ── 4. Save failure probability ────────────────────────────────
	probSaveFailed := CalculateFailedSaveProbability(
		req.Attacker.AP,
		req.Target.Save,
		req.Target.Invulnerable,
		req.Settings.SaveModifier,
		req.Target.HasCover,
		req.Settings.SaveReroll,
	)

	// ── 5. Max attacks for sizing ──────────────────────────────────
	bounds := computeHitBounds(attackCountDist, hitOutcomeDist)

	// ── 7. Hits distribution (exact, discrete) ─────────────────────
	finalAutoWoundNormalHitDist := computeAutoWoundNormalHitDist(hitOutcomeDist, attackCountDist, bounds)

	// ── 6. Joint distribution: normal-before-save × devastating ────
	jointWoundDist := computeJointWoundDist(finalAutoWoundNormalHitDist, bounds, probNormalWound, probDevWound)

	finalHitsDist := computeFinalHitsDist(finalAutoWoundNormalHitDist, bounds)

	totalWoundsDist := computeTotalWoundsDist(jointWoundDist, bounds.maxHits)

	// ── 10. Final unsaved dist (unsaved normal + devastating) ──────
	// Compute by looping over joint
	finalUnsavedDist := make([]float64, bounds.maxHits+1)
	for nw := 0; nw <= bounds.maxHits; nw++ {
		for dw := 0; dw <= bounds.maxHits; dw++ {
			pJoint := jointWoundDist[nw][dw]
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
	// TODO: Some units grant Feel No Pain that explicitly excludes devastating
	// wound damage. Not modeled — FNP is currently applied uniformly to both
	// normal and devastating damage via dmgDist.
	dmgDist := _calculateDamageDistribution(req.Attacker.Damage, req.Target.FeelNoPain)
	finalKilledSlice := make([]float64, *req.Target.Count+1)

	// NEW: Total Damage Accumulators
	maxD := GetMaxFromDice(req.Attacker.Damage)
	totalDamageVec := make([]float64, bounds.maxHits*maxD+1)
	// Pre-calculate convolutions for each possible total hit count [0...maxHits]
	// damageConvs[hits][damage]
	damageConvs := make([][]float64, bounds.maxHits+1)
	damageConvs[0] = []float64{1.0}

	for nw := 0; nw <= bounds.maxHits; nw++ {
		row := jointWoundDist[nw]
		for dw := 0; dw <= bounds.maxHits; dw++ {
			pJoint := row[dw]
			if pJoint < 1e-12 {
				continue
			}

			unsavedNormalDist := getBinomialVector(nw, probSaveFailed)
			for u, pU := range unsavedNormalDist {
				weight := pJoint * pU
				if weight < 1e-15 {
					continue
				}
				resolveDamageToSlice(
					u, dw,
					dmgDist, dmgDist,
					req.Target.WoundsPerModel,
					*req.Target.Count,
					finalKilledSlice,
					weight,
				)
				// 2. New Logic: Total Damage Distribution
				totalUnsaved := u + dw
				// Lazily compute convolution only when needed
				if damageConvs[totalUnsaved] == nil {
					prev := damageConvs[totalUnsaved-1]
					curr := make([]float64, len(prev)+maxD)
					for i, pPrev := range prev {
						for dVal, pD := range dmgDist {
							curr[i+dVal] += pPrev * pD
						}
					}
					damageConvs[totalUnsaved] = curr
				}

				// Accumulate directly into result
				for d, pD := range damageConvs[totalUnsaved] {
					totalDamageVec[d] += pD * weight
				}
			}
		}
	}

	return formatResponse(
		vectorToMap(finalHitsDist),
		vectorToMap(totalWoundsDist),
		vectorToMap(finalUnsavedDist),
		vectorToMap(totalDamageVec),
		vectorToMap(finalKilledSlice),
	), nil
}

// hitBounds carries the truncation bounds used to size every dense
// matrix/slice downstream. maxHits is tighter than maxN+maxL because a
// single attack can't simultaneously produce its worst-case normal AND
// worst-case lethal outcome.
type hitBounds struct {
	maxAttacks         int
	maxNormalPerAttack int
	maxLethalPerAttack int
	maxN               int // maxAttacks * maxNormalPerAttack
	maxL               int // maxAttacks * maxLethalPerAttack
	maxHits            int
}

// computeHitBounds derives the sizing bounds for the hit/wound/damage
// matrices from the attack-count and single-attack hit-outcome
// distributions: the largest attack count that can occur, and the largest
// normal/lethal/total hit outcome a single attack can produce.
func computeHitBounds(attackCountDist map[int]float64, hitOutcomeDist map[HitOutcome]float64) hitBounds {
	maxAttacks := 0
	for n := range attackCountDist {
		if n > maxAttacks {
			maxAttacks = n
		}
	}

	maxNper, maxLper, maxPerTotal := 0, 0, 0
	for o := range hitOutcomeDist {
		if o.NormalHits > maxNper {
			maxNper = o.NormalHits
		}
		if o.LethalHits > maxLper {
			maxLper = o.LethalHits
		}
		if o.NormalHits+o.LethalHits > maxPerTotal {
			maxPerTotal = o.NormalHits + o.LethalHits
		}
	}

	return hitBounds{
		maxAttacks:         maxAttacks,
		maxNormalPerAttack: maxNper,
		maxLethalPerAttack: maxLper,
		maxN:               maxAttacks * maxNper,
		maxL:               maxAttacks * maxLper,
		maxHits:            maxAttacks * maxPerTotal,
	}
}

// computeAutoWoundNormalHitDist returns the final collapsed hit distribution
// (auto wounds × normal hits), summed across every possible attack count.
func computeAutoWoundNormalHitDist(hitOutcomeDist map[HitOutcome]float64, attackCountDist map[int]float64, bounds hitBounds) AutoWoundNormalHitMatrix {
	// Build dense single-attack hit matrix
	singleAttackHitMatrix := BuildSingleAttackHitMatrix(
		hitOutcomeDist,
		bounds.maxNormalPerAttack,
		bounds.maxLethalPerAttack,
	)

	finalAutoWoundNormalHitDist := make(AutoWoundNormalHitMatrix, bounds.maxL+1)
	for i := range finalAutoWoundNormalHitDist {
		finalAutoWoundNormalHitDist[i] = make([]float64, bounds.maxN+1)
	}

	// Loop over attack count distribution
	for attackCount, attackProbability := range attackCountDist {
		if attackProbability < 1e-15 {
			continue
		}

		// Exact joint hit distribution for this attack count
		jointHitMatrix := ComputeMultiAttackHitDistribution(
			singleAttackHitMatrix,
			attackCount,
			bounds.maxNormalPerAttack,
			bounds.maxLethalPerAttack,
			bounds.maxN,
			bounds.maxL,
		)

		// Collapse lethal → auto wounds
		autoWoundNormalHitMatrix :=
			CollapseLethalHitsIntoAutoWounds(jointHitMatrix, bounds.maxN, bounds.maxL)

		// Accumulate weighted result
		for auto := 0; auto <= bounds.maxL; auto++ {
			for normal := 0; normal <= bounds.maxN; normal++ {
				p := autoWoundNormalHitMatrix[auto][normal]
				if p > 0 {
					finalAutoWoundNormalHitDist[auto][normal] +=
						attackProbability * p
				}
			}
		}
	}

	return finalAutoWoundNormalHitDist
}

// NormalDevastatingWoundMatrix is the joint probability mass of
// (normal wounds before save, devastating wounds), indexed
// [normalWounds][devastatingWounds].
type NormalDevastatingWoundMatrix [][]float64

// computeJointWoundDist resolves each (autoWounds, normalHits) hit-count
// state into wounds: normal hits roll to wound independently (binomAny),
// and each of those wounds is further split into normal vs. devastating
// (binomDev, conditioned on having already wounded). Auto-wounds (from
// Lethal Hits) always wound, so they pass straight through.
func computeJointWoundDist(autoWoundNormalHitDist AutoWoundNormalHitMatrix, bounds hitBounds, probNormalWound, probDevWound float64) NormalDevastatingWoundMatrix {
	jointWoundDist := make(NormalDevastatingWoundMatrix, bounds.maxHits+1)
	for i := range jointWoundDist {
		jointWoundDist[i] = make([]float64, bounds.maxHits+1)
	}

	// Precompute binomials (caching)
	pAnyWound := probNormalWound + probDevWound
	pDevCond := 0.0
	if pAnyWound > 0 {
		pDevCond = probDevWound / pAnyWound
	}
	binomAny := precomputeBinomials(bounds.maxN, pAnyWound)
	binomDev := precomputeBinomials(bounds.maxN, pDevCond) // max wounds <= maxN

	for autoWounds := 0; autoWounds <= bounds.maxL; autoWounds++ {
		for normalHits := 0; normalHits <= bounds.maxN; normalHits++ {
			pState := autoWoundNormalHitDist[autoWounds][normalHits]
			if pState < 1e-15 {
				continue
			}

			// If no wounds possible or no normal hits, only auto-wounds contribute.
			if pAnyWound <= 0 || normalHits == 0 {
				jointWoundDist[autoWounds][0] += pState
				continue
			}

			// Roll to wound for normal hits only.
			for totalWounds, pWound := range binomAny[normalHits] {
				if pWound < 1e-15 {
					continue
				}

				// Split wounds into normal vs devastating.
				for devWounds, pDev := range binomDev[totalWounds] {
					pFinal := pState * pWound * pDev
					if pFinal < 1e-15 {
						continue
					}

					normWounds := autoWounds + (totalWounds - devWounds)
					jointWoundDist[normWounds][devWounds] += pFinal
				}
			}
		}
	}

	return jointWoundDist
}

// computeFinalHitsDist returns the total hit distribution (Normal + Auto),
// summing autoWounds (from Lethal Hits) and normalHits per state.
func computeFinalHitsDist(autoWoundNormalHitDist AutoWoundNormalHitMatrix, bounds hitBounds) []float64 {
	finalHitsDist := make([]float64, bounds.maxHits+1)

	for autoWounds := 0; autoWounds <= bounds.maxL; autoWounds++ {
		for normalHits := 0; normalHits <= bounds.maxN; normalHits++ {
			probability := autoWoundNormalHitDist[autoWounds][normalHits]
			if probability < 1e-15 {
				continue
			}

			totalHits := autoWounds + normalHits
			if totalHits <= bounds.maxHits {
				finalHitsDist[totalHits] += probability
			}
		}
	}

	return finalHitsDist
}

// computeTotalWoundsDist returns the total potential wounds distribution
// (nw + dw) — normal wounds before save, plus devastating wounds.
func computeTotalWoundsDist(jointWoundDist NormalDevastatingWoundMatrix, maxHits int) []float64 {
	totalWoundsDist := make([]float64, maxHits+1)
	for nw := 0; nw <= maxHits; nw++ {
		row := jointWoundDist[nw]
		for dw := 0; dw <= maxHits; dw++ {
			p := row[dw]
			if p < 1e-15 {
				continue
			}

			total := nw + dw
			if total <= maxHits {
				totalWoundsDist[total] += p
			}
		}
	}
	return totalWoundsDist
}

// computeHitOutcomeDist returns the PMF of hit outcomes for a single attack.
func computeHitOutcomeDist(req CombatSimulationRequest) map[HitOutcome]float64 {
	if req.Attacker.Torrent {
		// Torrent: Auto-hits bypass the hit roll logic entirely.
		// They cannot trigger Critical Hit effects (Lethal/Sustained).
		return map[HitOutcome]float64{
			{NormalHits: 1, LethalHits: 0}: 1.0,
		}
	}

	// Standard Hit Roll logic
	return CalculateSingleHitDistribution(
		req.Attacker.BS,
		req.Settings.HitReroll,
		req.Settings.HitModifier,
		req.Attacker.LethalHits,
		req.Attacker.SustainedHits,
		req.Settings.CriticalHitThreshold,
	)
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

// applyWoundsLinear is the core state-transition engine.
func applyWoundsLinear(next, states []float64, dmgDist map[int]float64, maxHP int, spills bool) {
	// 'next' must be zeroed by the caller before passing in.

	for currentWounds, stateProb := range states {
		if stateProb <= 1e-15 { // Efficiency: Skip negligible probabilities
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
}

// formatResponse calculates final averages and builds the structured response for the client.
func formatResponse(hits, wounds, pens, damage, killed map[int]float64) SimulationResult {
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
		DamageDist:       damage,
		DestroyedDist:    killed,
	}
}

// Precompute all binomial distributions for n=0 to maxN with probability p
func precomputeBinomials(maxN int, p float64) [][]float64 {
	res := make([][]float64, maxN+1)
	if p <= 0 {
		for n := 0; n <= maxN; n++ {
			dist := make([]float64, n+1)
			dist[0] = 1.0
			res[n] = dist
		}
		return res
	}
	if p >= 1 {
		for n := 0; n <= maxN; n++ {
			dist := make([]float64, n+1)
			dist[n] = 1.0
			res[n] = dist
		}
		return res
	}

	q := 1 - p
	current := []float64{1.0}
	res[0] = []float64{1.0}
	for n := 1; n <= maxN; n++ {
		newCurrent := make([]float64, n+1)
		newCurrent[0] = current[0] * q
		for j := 1; j < n; j++ {
			newCurrent[j] = current[j]*q + current[j-1]*p
		}
		newCurrent[n] = current[n-1] * p
		res[n] = newCurrent
		current = newCurrent
	}
	return res
}

func resolveDamageToSlice(
	nNorm, nMortal int,
	normDmgDist, mortalDmgDist map[int]float64,
	maxHP, totalModels int,
	dest []float64,
	weight float64,
) {
	maxPossible := totalModels * maxHP

	// Buffers for ping-ponging.
	// To reach Zero-Alloc, these should be moved to a sync.Pool.
	buf1 := make([]float64, maxPossible+1)
	buf2 := make([]float64, maxPossible+1)

	states := buf1
	states[maxPossible] = 1.0
	next := buf2

	// 1. Normal Hits Loop
	for i := 0; i < nNorm; i++ {
		// Zero the scratchpad
		for j := range next {
			next[j] = 0
		}
		// In-place mutation
		applyWoundsLinear(next, states, normDmgDist, maxHP, false)
		// Swap
		states, next = next, states
	}

	// 2. Devastating Wounds Loop (spills = false by new rules)
	for i := 0; i < nMortal; i++ {
		for j := range next {
			next[j] = 0
		}
		applyWoundsLinear(next, states, mortalDmgDist, maxHP, false)
		states, next = next, states
	}

	// 3. Accumulate weighted results directly into the destination
	for remaining, prob := range states {
		if prob < 1e-15 {
			continue
		}
		modelsLeft := (remaining + maxHP - 1) / maxHP
		killed := totalModels - modelsLeft
		dest[killed] += prob * weight
	}
}

// Dense 2D probability mass for joint hit outcomes
// Indexing: [normalHits][lethalHits]
type JointHitProbabilityMatrix [][]float64

// Dense 2D probability after collapsing lethals
// Indexing: [autoWounds][normalHits]
type AutoWoundNormalHitMatrix [][]float64

func BuildSingleAttackHitMatrix(
	hitOutcomePMF map[HitOutcome]float64,
	maxNormalHitsPerAttack int,
	maxLethalHitsPerAttack int,
) JointHitProbabilityMatrix {

	matrix := make(JointHitProbabilityMatrix, maxNormalHitsPerAttack+1)
	for i := range matrix {
		matrix[i] = make([]float64, maxLethalHitsPerAttack+1)
	}

	for outcome, probability := range hitOutcomePMF {
		n := outcome.NormalHits
		l := outcome.LethalHits
		if n <= maxNormalHitsPerAttack && l <= maxLethalHitsPerAttack {
			matrix[n][l] += probability
		}
	}

	return matrix
}

func ConvolveJointHitMatricesBounded(
	left JointHitProbabilityMatrix,
	right JointHitProbabilityMatrix,
	leftMaxNormal int,
	leftMaxLethal int,
	rightMaxNormal int,
	rightMaxLethal int,
	globalMaxNormal int,
	globalMaxLethal int,
) (JointHitProbabilityMatrix, int, int) {

	result := make(JointHitProbabilityMatrix, globalMaxNormal+1)
	for i := range result {
		result[i] = make([]float64, globalMaxLethal+1)
	}

	for ln := 0; ln <= leftMaxNormal; ln++ {
		for ll := 0; ll <= leftMaxLethal; ll++ {
			leftProb := left[ln][ll]
			if leftProb < 1e-18 {
				continue
			}
			for rn := 0; rn <= rightMaxNormal && ln+rn <= globalMaxNormal; rn++ {
				for rl := 0; rl <= rightMaxLethal && ll+rl <= globalMaxLethal; rl++ {
					rightProb := right[rn][rl]
					if rightProb > 0 {
						result[ln+rn][ll+rl] += leftProb * rightProb
					}
				}
			}
		}
	}

	newMaxNormal := min(globalMaxNormal, leftMaxNormal+rightMaxNormal)
	newMaxLethal := min(globalMaxLethal, leftMaxLethal+rightMaxLethal)

	return result, newMaxNormal, newMaxLethal
}

func ComputeMultiAttackHitDistribution(
	singleAttackMatrix JointHitProbabilityMatrix,
	attacks int,
	maxNormalPerAttack int,
	maxLethalPerAttack int,
	globalMaxNormal int,
	globalMaxLethal int,
) JointHitProbabilityMatrix {

	// Identity distribution: zero attacks
	result := make(JointHitProbabilityMatrix, globalMaxNormal+1)
	for i := range result {
		result[i] = make([]float64, globalMaxLethal+1)
	}
	result[0][0] = 1.0
	resultMaxNormal := 0
	resultMaxLethal := 0

	base := singleAttackMatrix
	baseMaxNormal := maxNormalPerAttack
	baseMaxLethal := maxLethalPerAttack

	remaining := attacks

	for remaining > 0 {
		if remaining&1 == 1 {
			result, resultMaxNormal, resultMaxLethal =
				ConvolveJointHitMatricesBounded(
					result, base,
					resultMaxNormal, resultMaxLethal,
					baseMaxNormal, baseMaxLethal,
					globalMaxNormal, globalMaxLethal,
				)
		}

		remaining >>= 1
		if remaining > 0 {
			base, baseMaxNormal, baseMaxLethal =
				ConvolveJointHitMatricesBounded(
					base, base,
					baseMaxNormal, baseMaxLethal,
					baseMaxNormal, baseMaxLethal,
					globalMaxNormal, globalMaxLethal,
				)
		}
	}

	return result
}

func CollapseLethalHitsIntoAutoWounds(
	hitMatrix JointHitProbabilityMatrix,
	maxNormalHits int,
	maxLethalHits int,
) AutoWoundNormalHitMatrix {

	result := make(AutoWoundNormalHitMatrix, maxLethalHits+1)
	for auto := range result {
		result[auto] = make([]float64, maxNormalHits+1)
	}

	for normal := 0; normal <= maxNormalHits; normal++ {
		for lethal := 0; lethal <= maxLethalHits; lethal++ {
			prob := hitMatrix[normal][lethal]
			if prob > 0 {
				result[lethal][normal] += prob
			}
		}
	}

	return result
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

// Hydrate enforces data integrity and defaults.
// It guarantees the Core logic receives valid ranges (2-6) and initialized pointers.
func (d *DamageCalculatorImpl) Hydrate(req *CombatSimulationRequest) {
	// 1. Critical Thresholds
	// If 0 (uninitialized), default to 6. Otherwise, clamp to valid D6 range [2, 6].
	req.Settings.CriticalHitThreshold = fixThreshold(req.Settings.CriticalHitThreshold)
	req.Settings.CriticalWoundThreshold = fixThreshold(req.Settings.CriticalWoundThreshold)

	// 2. Ballistic Skill (Clamping [2, 6])
	if !req.Attacker.Torrent {
		req.Attacker.BS = clamp(req.Attacker.BS, 2, 6)
	}

	// 3. Save Floor (Min 2+)
	if req.Target.Save < 2 {
		req.Target.Save = 2
	}

	// 4. Invulnerable Save (Pointer logic)
	if req.Target.Invulnerable != nil && *req.Target.Invulnerable < 2 {
		val := 2
		req.Target.Invulnerable = &val
	}

	// 5. Infinite Logic / Target Count Resolution
	if req.Target.Count == nil {
		// Calculate the ceiling for target resolution
		maxAttacks := GetMaxFromDice(req.Attacker.Attacks) * req.Attacker.Count
		count := maxAttacks

		// Enforce DOS cap
		if count > 200 {
			count = 200
		}
		req.Target.Count = &count
	}
}

// DefaultComplexityValidator implements the stress-test logic.
// It is read-only and calculates the computational cost.
func DefaultComplexityValidator(req *CombatSimulationRequest) error {
	const Threshold = 8_000_000

	// 1. maxAttacks
	baseAttacksPerModel := GetMaxFromDice(req.Attacker.Attacks)

	blastBonusPerModel := 0
	if req.Attacker.Blast {
		blastBonusPerModel = *req.Target.Count / 5
	}

	maxAttacks :=
		req.Attacker.Count *
			(baseAttacksPerModel + blastBonusPerModel)
	targetCount := 0
	if req.Target.Count != nil {
		targetCount = *req.Target.Count
	}
	if req.Attacker.Blast {
		maxAttacks += targetCount / 5
	}

	// 2. stateSpace
	stateSpace := targetCount * req.Target.WoundsPerModel

	// 3. hit upper bound
	maxHitsPerAttack := 1 + req.Attacker.SustainedHits
	maxHits := maxAttacks * maxHitsPerAttack

	// 4. score
	logA := 0
	for a := maxAttacks; a > 1; a >>= 1 {
		logA++
	}

	score :=
		logA*maxHits*maxHits +
			maxHits*maxHits +
			4*maxHits*stateSpace

	if score > Threshold {
		return fmt.Errorf(
			"complexity overflow: score=%d > %d (A=%d H=%d S=%d)",
			score, Threshold, maxAttacks, maxHits, stateSpace,
		)
	}
	return nil
}

// clamp helper for hydration efficiency
func clamp(val, min, max int) int {
	if val < min {
		return min
	}
	if val > max {
		return max
	}
	return val
}

// fixThreshold handles the 0 -> 6 default before clamping
func fixThreshold(val int) int {
	if val == 0 {
		return 6
	}
	return clamp(val, 2, 6)
}
