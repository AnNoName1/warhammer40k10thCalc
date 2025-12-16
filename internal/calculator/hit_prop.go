package calculator

import (
	"math"

	damagerequest "github.com/AnNoName1/warhammer40k10thCalc/pkg/models"
)

// _calculateHitProbability calculates the hit probability for a single attack.
// Returns (normal_hit_probability, lethal_hit_probability).
func _calculateHitProbability(bs int, rerollType damagerequest.RerollType, hitModifier int, lethalHits bool) (float64, float64) {
	const oneSixth = 1.0 / 6.0
	const fiveSixths = 5.0 / 6.0

	bsFloat := float64(bs)
	hitModifierFloat := float64(hitModifier)

	// 1. Base Hit Chance
	// Formula: (7 - BS + Modifier) / 6
	targetRollChance := (7.0 - bsFloat + hitModifierFloat) / 6.0

	// Clamp: the chance cannot be less than 1/6 (natural 1 is always a miss)
	// and no more than 5/6 (natural 6 is always a hit, unless modified, but 1 is always a miss).
	// In Go, math.Max/Min work with float64.
	hitChance := math.Max(oneSixth, math.Min(targetRollChance, fiveSixths))

	// Store the miss chance BEFORE modifying hitChance with rerolls
	missChance := 1.0 - hitChance

	// 2. Rerolls
	if rerollType == damagerequest.RerollOnes {
		// Reroll ones (1/6)
		hitChance += oneSixth * hitChance
	} else if rerollType == damagerequest.RerollFail {
		// Reroll all misses
		hitChance += missChance * hitChance
	}

	// 3. Lethal Hits
	lethalHitChance := 0.0
	if lethalHits {
		// Base 6
		lethalHitChance = oneSixth

		if rerollType == damagerequest.RerollOnes {
			// Additional chance from rerolling ones: (1/6 chance to roll 1) * (1/6 chance to roll 6)
			lethalHitChance += oneSixth * oneSixth
		} else if rerollType == damagerequest.RerollFail {
			// Additional chance from rerolling misses.
			// Logic: We take the ORIGINAL miss chance (missChance) and multiply by the chance to roll a 6 (1/6).
			// Note: The Python code used (1 - hitChance) * 1/6 inside the block,
			// but hitChance was already modified. Mathematically, using missChance is more correct.
			lethalHitChance += missChance * oneSixth
		}

		// Lethal hits are subtracted from normal hits, as they automatically wound
		hitChance -= lethalHitChance
		// Guard against negative values
		hitChance = math.Max(0.0, hitChance)
	}

	return hitChance, lethalHitChance
}
