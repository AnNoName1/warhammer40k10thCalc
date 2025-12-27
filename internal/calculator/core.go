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

	damagerequest "github.com/AnNoName1/warhammer40k10thCalc/pkg/models"
)

// UnitState tracks the health of the defending unit during sequential damage allocation.
// This is the "State" in our Markov Chain, representing a snapshot of unit health.
type UnitState struct {
	Killed    int // Number of models completely removed from the unit
	CurrentHP int // Remaining HP of the specific model currently taking damage
}

// CalculateDamageCore is the main entry point for the probability engine.
// It uses a "Transition Map" algorithm: instead of simulating dice rolls, it
// calculates the mathematical probability of every possible branch in the
// attack sequence (Hit -> Wound -> Save -> Damage).
func CalculateDamageCore(req damagerequest.DamageRequest) (damagerequest.DamageResponse, error) {
	// 1. SETUP: Convert raw request strings (like "2D6+2") into probability maps.
	// -------------------------------------------------------------------------
	attacksDist, err := CalculateAttackDistribution(req.AttacksString)
	if err != nil {
		return damagerequest.DamageResponse{}, err
	}

	// damageDist accounts for both the weapon damage and Feel No Pain (FNP) reduction.
	damageDist := _calculateDamageDistribution(req.D, req.FeelNoPain)

	// Pre-calculate fixed probabilities based on modifiers (Lethal Hits, Devastating Wounds, etc.)
	hitP, lethalP := _calculateHitProbability(req.BS, req.HitReroll, req.HitModifier, req.LethalHits)
	woundP, devP := _calculateWoundProbability(req.S, req.T, req.WoundReroll, req.WoundModifier, req.DevastatingWounds)
	saveFailP := _calculateFailedSaveProbability(req.AP, req.Save, req.Invulnerable, req.SaveModifier)

	// Accumulators for the final statistical distributions.
	hitsDist := make(map[int]float64)
	woundsDist := make(map[int]float64)
	pensDist := make(map[int]float64)
	killedDist := make(map[int]float64)

	// 2. THE PIPELINE: Nested loops representing the sequence of play.
	// Each loop handles one stage of the Warhammer 40k attack resolution.
	// -------------------------------------------------------------------------
	for numAttacks, pAtk := range attacksDist {

		// STAGE A: HIT ROLLS
		// Generates probabilities for (Normal Hits, Lethal Hits)
		hitOutcomes := getHitOutcomes(numAttacks, hitP, lethalP)
		for ho, pHO := range hitOutcomes {
			hitsDist[ho.normal+ho.lethal] += pAtk * pHO

			// STAGE B: WOUND ROLLS
			// Normal hits roll to wound; Lethal hits skip this and become automatic wounds.
			woundOutcomes := getWoundOutcomes(ho.normal, ho.lethal, woundP, devP)
			for wo, pWO := range woundOutcomes {
				woundsDist[wo.normal+wo.devastating] += pAtk * pHO * pWO

				// STAGE C: SAVE ROLLS
				// Devastating wounds skip saves; Normal wounds check against the Save/AP.
				unsavedOutcomes := getUnsavedOutcomes(wo.normal, wo.devastating, saveFailP)
				for uo, pUO := range unsavedOutcomes {
					pensDist[uo.normal+uo.mortal] += pAtk * pHO * pWO * pUO

					// STAGE D: DAMAGE RESOLUTION (The Markov Chain)
					// Applies unsaved damage sequentially to models, handling "Wasted" vs "Spillover" damage.
					killedMap := resolveDamageSequential(uo.normal, uo.mortal, damageDist, req.WoundsPerModel, req.NumModels)

					// Calculate the total probability weight for this specific branch of the tree.
					weight := pAtk * pHO * pWO * pUO
					for numKilled, pKilled := range killedMap {
						killedDist[numKilled] += weight * pKilled
					}
				}
			}
		}
	}

	return formatResponse(hitsDist, woundsDist, pensDist, killedDist), nil
}

// --- HELPER FUNCTIONS ---
type hitResult struct{ normal, lethal int }

// getHitOutcomes calculates the binomial distribution for hits.
// It tracks 'Lethal Hits' (6s) separately from 'Normal Hits' because they interact differently with wounds.
func getHitOutcomes(n int, pHit, pLethal float64) map[hitResult]float64 {
	res := map[hitResult]float64{{0, 0}: 1.0}
	for i := 0; i < n; i++ {
		next := make(map[hitResult]float64)
		for st, p := range res {
			next[st] += p * (1.0 - (pHit + pLethal))                 // Outcome: Miss
			next[hitResult{st.normal + 1, st.lethal}] += p * pHit    // Outcome: Normal Hit
			next[hitResult{st.normal, st.lethal + 1}] += p * pLethal // Outcome: Lethal Hit
		}
		res = next
	}
	return res
}

type woundResult struct{ normal, devastating int }

// getWoundOutcomes processes normal hits through a wound roll.
// It integrates Lethal Hits as guaranteed "normal" wounds.
func getWoundOutcomes(nHit, lHit int, pWound, pDev float64) map[woundResult]float64 {
	res := map[woundResult]float64{{lHit, 0}: 1.0}
	for i := 0; i < nHit; i++ {
		next := make(map[woundResult]float64)
		for st, p := range res {
			next[st] += p * (1.0 - (pWound + pDev))                        // Outcome: Fail to wound
			next[woundResult{st.normal + 1, st.devastating}] += p * pWound // Outcome: Normal Wound
			next[woundResult{st.normal, st.devastating + 1}] += p * pDev   // Outcome: Devastating Wound
		}
		res = next
	}
	return res
}

type unsavedResult struct{ normal, mortal int }

// getUnsavedOutcomes determines how many wounds get past the Armor/Invulnerable save.
// It labels Devastating Wounds as "mortal" because they ignore saves entirely in 10th Ed.
func getUnsavedOutcomes(nWnd, dWnd int, pFail float64) map[unsavedResult]float64 {
	res := map[unsavedResult]float64{{0, 0}: 1.0}
	for i := 0; i < nWnd; i++ {
		next := make(map[unsavedResult]float64)
		for st, p := range res {
			next[st] += p * (1.0 - pFail)                              // Outcome: Save passed
			next[unsavedResult{st.normal + 1, st.mortal}] += p * pFail // Outcome: Save failed
		}
		res = next
	}
	for i := 0; i < dWnd; i++ {
		next := make(map[unsavedResult]float64)
		for st, p := range res {
			next[unsavedResult{st.normal, st.mortal + 1}] += p // Outcome: Guaranteed unsaved
		}
		res = next
	}
	return res
}

// resolveDamageSequential manages the transition from "Unsaved Wound" to "Model Removed".
// It processes Normal Wounds first (damage wasted if it exceeds remaining HP),
// then processes Mortal Wounds (damage spills over to the next model).
func resolveDamageSequential(nNorm, nMortal int, dmgDist map[int]float64, maxHP, totalModels int) map[int]float64 {
	states := map[UnitState]float64{{Killed: 0, CurrentHP: maxHP}: 1.0}

	// Normal wounds do not spill over.
	for i := 0; i < nNorm; i++ {
		states = applyWounds(states, dmgDist, maxHP, totalModels, false)
	}
	// Mortal wounds (Devastating) do spill over.
	for i := 0; i < nMortal; i++ {
		states = applyWounds(states, dmgDist, maxHP, totalModels, true)
	}

	// Collapse the detailed UnitState (Killed+HP) back into a simple 'Models Killed' map.
	final := make(map[int]float64)
	for st, p := range states {
		final[st.Killed] += p
	}
	return final
}

// applyWounds is the core state-transition engine.
// It iterates through every possible HP state and applies a single attack's damage distribution.
func applyWounds(states map[UnitState]float64, dmgDist map[int]float64, maxHP, totalModels int, spills bool) map[UnitState]float64 {
	next := make(map[UnitState]float64)
	for st, p := range states {
		// Stop calculating if the whole unit is already dead.
		if st.Killed >= totalModels {
			next[st] += p
			continue
		}
		for dVal, dP := range dmgDist {
			prob := p * dP
			hp, killed := st.CurrentHP, st.Killed

			if spills {
				// MORTAL WOUND LOGIC: Damage carries over.
				rem := dVal
				for rem > 0 && killed < totalModels {
					if rem >= hp {
						rem -= hp
						killed++
						hp = maxHP
					} else {
						hp -= rem
						rem = 0
					}
				}
			} else {
				// NORMAL DAMAGE LOGIC: Excess damage is lost.
				if dVal >= hp {
					killed++
					hp = maxHP
				} else {
					hp -= dVal
				}
			}
			next[UnitState{killed, hp}] += prob
		}
	}
	return next
}

// formatResponse calculates final averages and builds the structured response for the client.
func formatResponse(hits, wounds, pens, killed map[int]float64) damagerequest.DamageResponse {
	avgK := 0.0
	for k, v := range killed {
		avgK += float64(k) * v
	}
	avgH := 0.0
	for k, v := range hits {
		avgH += float64(k) * v
	}

	return damagerequest.DamageResponse{
		AverageHits:           avgH,
		AverageDestroyed:      avgK,
		HitsDistribution:      hits,
		WoundsDistribution:    wounds,
		PensDistribution:      pens,
		DestroyedDistribution: killed,
		Message:               fmt.Sprintf("Calculated probability for %d potential unit health states.", len(killed)),
	}
}
