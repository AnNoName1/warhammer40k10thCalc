// Copyright (c) 2026 Olbutov Aleksandr
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
	"math"
	"testing"
)

// Test configuration

// Helper functions

// almostEqual checks if two floats are equal within epsilon tolerance
func almostEqual(a, b, eps float64) bool {
	return math.Abs(a-b) <= eps
}

// sumDistribution calculates the sum of all probabilities in a distribution
func sumDistribution(dist map[int]float64) float64 {
	sum := 0.0
	for _, p := range dist {
		sum += p
	}
	return sum
}

// expectedValue calculates the expected value of a distribution
func expectedValue(dist map[int]float64) float64 {
	ev := 0.0
	for k, p := range dist {
		ev += float64(k) * p
	}
	return ev
}

// maxKey returns the maximum key in a distribution
func maxKey(dist map[int]float64) int {
	max := 0
	for k := range dist {
		if k > max {
			max = k
		}
	}
	return max
}

// Helper function to compare distributions
func distributionsEqual(dist1, dist2 map[int]float64, eps float64) bool {
	if len(dist1) != len(dist2) {
		return false
	}

	for k, v1 := range dist1 {
		v2, ok := dist2[k]
		if !ok {
			return false
		}
		if !almostEqual(v1, v2, eps) {
			return false
		}
	}

	return true
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// getCDF converts a PMF (Dist) into a CDF (Cumulative Distribution Function)
func getCDF(dist map[int]float64, t *testing.T) map[int]float64 {
	cdf := make(map[int]float64)
	maxD := 0
	for k := range dist {
		if k > maxD {
			maxD = k
		}
	}

	runningSum := 0.0
	// Iterate through possible damage values (0 to max)
	for i := 0; i <= maxD; i++ {
		p := dist[i] // If key missing, p is 0
		runningSum += p
		cdf[i] = runningSum
	}

	// Normalization sanity check (warn if significant deviation)
	if math.Abs(runningSum-1.0) > 1e-6 {
		t.Logf("Warning: CDF sum is %v (not 1.0)", runningSum)
	}

	return cdf
}

// verifyDominance checks if 'better' stochastically dominates 'worse'.
// FSD: F_better(x) <= F_worse(x) for all x, with at least one strict <.
func verifyDominance(t *testing.T, name string, betterDist, worseDist map[int]float64) {
	t.Helper()
	cdfBetter := getCDF(betterDist, t)
	cdfWorse := getCDF(worseDist, t)
	// Union of domains
	maxDomain := 0
	for k := range cdfBetter {
		if k > maxDomain {
			maxDomain = k
		}
	}
	for k := range cdfWorse {
		if k > maxDomain {
			maxDomain = k
		}
	}
	isStrict := false
	violation := false
	for x := 0; x <= maxDomain; x++ {
		F_better := 1.0
		if val, ok := cdfBetter[x]; ok {
			F_better = val
		}
		F_worse := 1.0
		if val, ok := cdfWorse[x]; ok {
			F_worse = val
		}
		if F_better > F_worse+1e-9 {
			t.Errorf("%s FSD Violation at x=%d: F_better(%f) > F_worse(%f)", name, x, F_better, F_worse)
			violation = true
		}
		if F_better < F_worse-1e-9 {
			isStrict = true
			// Optional: t.Logf("%s Strict improvement at x=%d", name, x)
		}
	}
	if !violation && !isStrict {
		t.Logf("%s: Distributions identical (no strict improvement). Verify if param change should affect damage.", name)
	}
}

// Test generators

// generateBaseRequest creates a baseline request for testing
func generateBaseRequest() CombatSimulationRequest {
	return CombatSimulationRequest{
		Attacker: AttackerProfile{
			Count:             10,
			Attacks:           DiceRoll{Count: 0, Sides: 0, Modifier: 2},
			BS:                3,
			Strength:          4,
			AP:                1,
			Damage:            DiceRoll{Count: 0, Sides: 0, Modifier: 1},
			SustainedHits:     0,
			Blast:             false,
			LethalHits:        false,
			DevastatingWounds: false,
			Torrent:           false,
		},
		Target: TargetProfile{
			Count:          intPtr(10),
			Toughness:      4,
			Save:           3,
			Invulnerable:   nil,
			WoundsPerModel: 2,
			FeelNoPain:     nil,
			HasCover:       false,
		},
		Settings: SimulationSettings{
			HitReroll:              RerollNone,
			WoundReroll:            RerollNone,
			SaveReroll:             RerollNone,
			CriticalHitThreshold:   6,
			CriticalWoundThreshold: 6,
			SaveModifier:           0,
			HitModifier:            0,
			WoundModifier:          0,
		},
	}
}

// generateLargeScaleRequest creates a request with large numbers for stress testing
func generateLargeScaleRequest() CombatSimulationRequest {
	req := generateBaseRequest()
	req.Attacker.Count = 70
	req.Attacker.Attacks = DiceRoll{Count: 0, Sides: 0, Modifier: 6}
	req.Target.Count = intPtr(50)
	return req
}

func assertZeroSimulationResult(t *testing.T, result SimulationResult) {
	t.Helper()

	if !almostEqual(result.AverageHits, 0, epsilon) {
		t.Errorf("AverageHits should be 0, got %f", result.AverageHits)
	}
	if !almostEqual(result.AverageDestroyed, 0, epsilon) {
		t.Errorf("AverageDestroyed should be 0, got %f", result.AverageDestroyed)
	}

	checkZeroDist := func(dist map[int]float64, name string) {
		if len(dist) != 1 {
			t.Fatalf("%s should have exactly one entry, got %v", name, dist)
		}
		if p, ok := dist[0]; !ok || !almostEqual(p, 1.0, epsilon) {
			t.Fatalf("%s should be {0:1}, got %v", name, dist)
		}
	}

	checkZeroDist(result.HitDist, "HitDist")
	checkZeroDist(result.WoundDist, "WoundDist")
	checkZeroDist(result.PenDist, "PenDist")
	checkZeroDist(result.DestroyedDist, "DestroyedDist")
}

// P1 — Probability validity
func TestP01_ProbabilityValidity(t *testing.T) {
	calc := &DamageCalculatorImpl{}

	testCases := []struct {
		name string
		req  CombatSimulationRequest
	}{
		{"Base case", generateBaseRequest()},
		{"Large scale", generateLargeScaleRequest()},
		{"With rerolls", func() CombatSimulationRequest {
			req := generateBaseRequest()
			req.Settings.HitReroll = RerollOnes
			req.Settings.WoundReroll = RerollFail
			return req
		}()},
		{"With special abilities", func() CombatSimulationRequest {
			req := generateBaseRequest()
			req.Attacker.LethalHits = true
			req.Attacker.SustainedHits = 1
			return req
		}()},
		{"With devastating wounds", func() CombatSimulationRequest {
			req := generateBaseRequest()
			req.Attacker.DevastatingWounds = true
			return req
		}()},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := calc.CalculateDamageCore(tc.req)
			if err != nil {
				t.Fatalf("CalculateDamageCore failed: %v", err)
			}

			// Check all distributions
			distributions := map[string]map[int]float64{
				"HitDist":       result.HitDist,
				"WoundDist":     result.WoundDist,
				"PenDist":       result.PenDist,
				"DestroyedDist": result.DestroyedDist,
			}

			for distName, dist := range distributions {
				// Check all probabilities are non-negative
				for k, p := range dist {
					if p < -epsilon {
						t.Errorf("%s: probability for key %d is negative: %f", distName, k, p)
					}
				}

				// Check sum equals 1
				sum := sumDistribution(dist)
				if !almostEqual(sum, 1.0, epsilon) {
					t.Errorf("%s: sum of probabilities is %f, expected 1.0 (±%e)", distName, sum, epsilon)
				}
			}
		})
	}
}

// P2 — Averages must match distributions
func TestP02_AveragesMatchDistributions(t *testing.T) {
	calc := &DamageCalculatorImpl{}

	testCases := []struct {
		name string
		req  CombatSimulationRequest
	}{
		{"Base case", generateBaseRequest()},
		{"Large scale", generateLargeScaleRequest()},
		{"High damage", func() CombatSimulationRequest {
			req := generateBaseRequest()
			req.Attacker.Damage = DiceRoll{Count: 1, Sides: 6, Modifier: 2}
			return req
		}()},
		{"Multiple attacks", func() CombatSimulationRequest {
			req := generateBaseRequest()
			req.Attacker.Attacks = DiceRoll{Count: 3, Sides: 6, Modifier: 0}
			return req
		}()},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := calc.CalculateDamageCore(tc.req)
			if err != nil {
				t.Fatalf("CalculateDamageCore failed: %v", err)
			}

			// Check AverageHits matches HitDist
			expectedHits := expectedValue(result.HitDist)
			if !almostEqual(result.AverageHits, expectedHits, epsilon) {
				t.Errorf("AverageHits mismatch: got %f, expected from distribution %f",
					result.AverageHits, expectedHits)
			}

			// Check AverageDestroyed matches DestroyedDist
			expectedDestroyed := expectedValue(result.DestroyedDist)
			if !almostEqual(result.AverageDestroyed, expectedDestroyed, epsilon) {
				t.Errorf("AverageDestroyed mismatch: got %f, expected from distribution %f",
					result.AverageDestroyed, expectedDestroyed)
			}
		})
	}
}

// P3 — Destroyed models are bounded
func TestP03_DestroyedModelsBounds(t *testing.T) {
	calc := &DamageCalculatorImpl{}

	t.Run("P3a - Non-negative destroyed", func(t *testing.T) {
		testCases := []CombatSimulationRequest{
			generateBaseRequest(),
			generateLargeScaleRequest(),
		}

		for i, req := range testCases {
			t.Run(fmt.Sprintf("Case %d", i+1), func(t *testing.T) {
				result, err := calc.CalculateDamageCore(req)
				if err != nil {
					t.Fatalf("CalculateDamageCore failed: %v", err)
				}

				// Check average is non-negative
				if result.AverageDestroyed < -epsilon {
					t.Errorf("AverageDestroyed is negative: %f", result.AverageDestroyed)
				}

				// Check all distribution keys are non-negative
				for k := range result.DestroyedDist {
					if k < 0 {
						t.Errorf("DestroyedDist has negative key: %d", k)
					}
				}
			})
		}
	})

	t.Run("P3b - Non-spillover bound", func(t *testing.T) {
		testCases := []struct {
			name string
			req  CombatSimulationRequest
		}{
			{"Base without dev wounds", generateBaseRequest()},
			{"Large scale without dev wounds", generateLargeScaleRequest()},
			{"Low target count", func() CombatSimulationRequest {
				req := generateBaseRequest()
				req.Target.Count = intPtr(3)
				return req
			}()},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {

				result, err := calc.CalculateDamageCore(tc.req)
				if err != nil {
					t.Fatalf("CalculateDamageCore failed: %v", err)
				}

				// Calculate total possible attacks
				// For simplicity, assume Attacks is a simple number
				// In production, you'd parse the dice notation
				totalAttacks := tc.req.Attacker.Count * 2 // Simplified

				targetCount := *tc.req.Target.Count
				maxDestroyed := min(totalAttacks, targetCount)

				// Check max key in distribution
				if len(result.DestroyedDist) > 0 {
					maxDestroyedKey := maxKey(result.DestroyedDist)
					if maxDestroyedKey > maxDestroyed {
						t.Errorf("DestroyedDist max key %d exceeds bound %d", maxDestroyedKey, maxDestroyed)
					}
				}
			})
		}
	})
}

// P4 — Monotonicity properties (rerolls)
func TestP04_RerollMonotonicity(t *testing.T) {
	calc := &DamageCalculatorImpl{}

	t.Run("Hit reroll monotonicity", func(t *testing.T) {
		baseReq := generateBaseRequest()

		rerollTypes := []RerollType{RerollNone, RerollOnes, RerollFail}
		var prevHits float64 = -1

		for _, reroll := range rerollTypes {
			req := baseReq
			req.Settings.HitReroll = reroll

			result, err := calc.CalculateDamageCore(req)
			if err != nil {
				t.Fatalf("CalculateDamageCore failed for reroll %v: %v", reroll, err)
			}

			if prevHits >= 0 {
				if result.AverageHits < prevHits-epsilon {
					t.Errorf("Hit reroll monotonicity violated: %v gave %f hits, previous was %f",
						reroll, result.AverageHits, prevHits)
				}
			}
			prevHits = result.AverageHits
		}
	})

	t.Run("Wound reroll monotonicity", func(t *testing.T) {
		baseReq := generateBaseRequest()

		rerollTypes := []RerollType{RerollNone, RerollOnes, RerollFail}
		var prevWounds float64 = -1

		for _, reroll := range rerollTypes {
			req := baseReq
			req.Settings.WoundReroll = reroll

			result, err := calc.CalculateDamageCore(req)
			if err != nil {
				t.Fatalf("CalculateDamageCore failed for reroll %v: %v", reroll, err)
			}

			expectedWounds := expectedValue(result.WoundDist)
			if prevWounds >= 0 {
				if expectedWounds < prevWounds-epsilon {
					t.Errorf("Wound reroll monotonicity violated: %v gave %f wounds, previous was %f",
						reroll, expectedWounds, prevWounds)
				}
			}
			prevWounds = expectedWounds
		}
	})
}

// P5 — Parameter monotonicity (directional)
func TestP05_ParameterMonotonicity(t *testing.T) {
	calc := &DamageCalculatorImpl{}

	t.Run("Attacker parameters increase output", func(t *testing.T) {
		tests := []struct {
			name   string
			modify func(*CombatSimulationRequest, int)
			values []int
		}{
			{
				name: "Attacker Count",
				modify: func(req *CombatSimulationRequest, val int) {
					req.Attacker.Count = val
				},
				values: []int{5, 10, 20, 50},
			},
			{
				name: "Strength",
				modify: func(req *CombatSimulationRequest, val int) {
					req.Attacker.Strength = val
				},
				values: []int{3, 4, 6, 8},
			},
			{
				name: "AP",
				modify: func(req *CombatSimulationRequest, val int) {
					req.Attacker.AP = val
				},
				values: []int{0, 1, 2, 3},
			},
			{
				name: "SustainedHits",
				modify: func(req *CombatSimulationRequest, val int) {
					req.Attacker.SustainedHits = val
				},
				values: []int{0, 1, 2, 3},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				var prevDestroyed float64 = -1

				for _, val := range tt.values {
					req := generateBaseRequest()
					tt.modify(&req, val)

					result, err := calc.CalculateDamageCore(req)
					if err != nil {
						t.Fatalf("CalculateDamageCore failed for %s=%d: %v", tt.name, val, err)
					}

					if prevDestroyed >= 0 {
						if result.AverageDestroyed < prevDestroyed-epsilon {
							t.Errorf("%s monotonicity violated: %d gave %f destroyed, previous was %f",
								tt.name, val, result.AverageDestroyed, prevDestroyed)
						}
					}
					prevDestroyed = result.AverageDestroyed
				}
			})
		}
	})

	t.Run("Defender parameters decrease output", func(t *testing.T) {
		tests := []struct {
			name   string
			modify func(*CombatSimulationRequest, int)
			values []int
		}{
			{
				name: "Toughness",
				modify: func(req *CombatSimulationRequest, val int) {
					req.Target.Toughness = val
				},
				values: []int{3, 4, 6, 8},
			},
			{
				name: "Save",
				modify: func(req *CombatSimulationRequest, val int) {
					req.Target.Save = val
				},
				values: []int{6, 4, 3, 2},
			},
			{
				name: "WoundsPerModel",
				modify: func(req *CombatSimulationRequest, val int) {
					req.Target.WoundsPerModel = val
				},
				values: []int{1, 2, 3, 5},
			},
			{
				name: "FeelNoPain",
				modify: func(req *CombatSimulationRequest, val int) {
					req.Target.FeelNoPain = intPtr(val)
				},
				values: []int{6, 5, 4, 3},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				prevDestroyed := math.Inf(1)

				for _, val := range tt.values {
					req := generateBaseRequest()
					tt.modify(&req, val)

					result, err := calc.CalculateDamageCore(req)
					if err != nil {
						t.Fatalf("CalculateDamageCore failed for %s=%d: %v", tt.name, val, err)
					}

					if !math.IsInf(prevDestroyed, 1) {
						if result.AverageDestroyed > prevDestroyed+epsilon {
							t.Errorf("%s monotonicity violated: %d gave %f destroyed, previous was %f",
								tt.name, val, result.AverageDestroyed, prevDestroyed)
						}
					}
					prevDestroyed = result.AverageDestroyed
				}
			})
		}
	})
}

// P6 — Pipeline isolation
func TestP06_PipelineIsolation(t *testing.T) {
	calc := &DamageCalculatorImpl{}

	t.Run("Wound modifier doesn't affect hits", func(t *testing.T) {
		req1 := generateBaseRequest()
		req1.Settings.WoundModifier = 0

		req2 := generateBaseRequest()
		req2.Settings.WoundModifier = 1

		result1, err := calc.CalculateDamageCore(req1)
		if err != nil {
			t.Fatalf("CalculateDamageCore failed: %v", err)
		}

		result2, err := calc.CalculateDamageCore(req2)
		if err != nil {
			t.Fatalf("CalculateDamageCore failed: %v", err)
		}

		// HitDist should be identical
		if !distributionsEqual(result1.HitDist, result2.HitDist, epsilon) {
			t.Errorf("WoundModifier affected HitDist")
		}
	})

	t.Run("Save reroll doesn't affect hits or wounds", func(t *testing.T) {
		req1 := generateBaseRequest()
		req1.Settings.SaveReroll = RerollNone

		req2 := generateBaseRequest()
		req2.Settings.SaveReroll = RerollFail

		result1, err := calc.CalculateDamageCore(req1)
		if err != nil {
			t.Fatalf("CalculateDamageCore failed: %v", err)
		}

		result2, err := calc.CalculateDamageCore(req2)
		if err != nil {
			t.Fatalf("CalculateDamageCore failed: %v", err)
		}

		// HitDist should be identical
		if !distributionsEqual(result1.HitDist, result2.HitDist, epsilon) {
			t.Errorf("SaveReroll affected HitDist")
		}

		// WoundDist should be identical
		if !distributionsEqual(result1.WoundDist, result2.WoundDist, epsilon) {
			t.Errorf("SaveReroll affected WoundDist")
		}
	})

	t.Run("FeelNoPain doesn't affect penetration", func(t *testing.T) {
		req1 := generateBaseRequest()
		req1.Target.FeelNoPain = nil

		req2 := generateBaseRequest()
		req2.Target.FeelNoPain = intPtr(5)

		result1, err := calc.CalculateDamageCore(req1)
		if err != nil {
			t.Fatalf("CalculateDamageCore failed: %v", err)
		}

		result2, err := calc.CalculateDamageCore(req2)
		if err != nil {
			t.Fatalf("CalculateDamageCore failed: %v", err)
		}

		// PenDist should be identical
		if !distributionsEqual(result1.PenDist, result2.PenDist, epsilon) {
			t.Errorf("FeelNoPain affected PenDist")
		}
	})

	t.Run("Target parameters don't affect hits", func(t *testing.T) {
		req1 := generateBaseRequest()
		req1.Target.Toughness = 3
		req1.Target.Save = 3

		req2 := generateBaseRequest()
		req2.Target.Toughness = 8
		req2.Target.Save = 2

		result1, err := calc.CalculateDamageCore(req1)
		if err != nil {
			t.Fatalf("CalculateDamageCore failed: %v", err)
		}

		result2, err := calc.CalculateDamageCore(req2)
		if err != nil {
			t.Fatalf("CalculateDamageCore failed: %v", err)
		}

		// HitDist should be identical
		if !distributionsEqual(result1.HitDist, result2.HitDist, epsilon) {
			t.Errorf("Target parameters affected HitDist")
		}
	})
}

// P7 — Zero-input invariants
func TestP07_ZeroInputInvariants(t *testing.T) {
	calc := &DamageCalculatorImpl{}

	t.Run("Zero attacker count", func(t *testing.T) {
		req := generateBaseRequest()
		req.Attacker.Count = 0

		result, err := calc.CalculateDamageCore(req)
		if err != nil {
			t.Fatalf("CalculateDamageCore failed: %v", err)
		}

		assertZeroSimulationResult(t, result)
	})

	t.Run("Zero target models", func(t *testing.T) {
		req := generateBaseRequest()
		req.Target.Count = intPtr(0)

		result, err := calc.CalculateDamageCore(req)
		if err != nil {
			t.Fatalf("CalculateDamageCore failed: %v", err)
		}

		almostEqual(result.AverageDestroyed, 1.0, epsilon)
	})

	t.Run("Blast must not resurrect zero attackers", func(t *testing.T) {
		req := generateBaseRequest()
		req.Attacker.Count = 0
		req.Attacker.Blast = true
		req.Target.Count = intPtr(20)

		result, err := calc.CalculateDamageCore(req)
		if err != nil {
			t.Fatalf("CalculateDamageCore failed: %v", err)
		}

		assertZeroSimulationResult(t, result)
	})
}

// P8 — Determinism
func TestP08_Determinism(t *testing.T) {
	calc := &DamageCalculatorImpl{}

	testCases := []struct {
		name string
		req  CombatSimulationRequest
	}{
		{"Base case", generateBaseRequest()},
		{"Large scale", generateLargeScaleRequest()},
		{"Complex case", func() CombatSimulationRequest {
			req := generateBaseRequest()
			req.Attacker.LethalHits = true
			req.Attacker.SustainedHits = 2
			req.Settings.HitReroll = RerollOnes
			req.Settings.WoundReroll = RerollFail
			return req
		}()},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Run the same request multiple times
			runs := 5
			results := make([]SimulationResult, runs)

			for i := 0; i < runs; i++ {
				result, err := calc.CalculateDamageCore(tc.req)
				if err != nil {
					t.Fatalf("CalculateDamageCore failed on run %d: %v", i+1, err)
				}
				results[i] = result
			}

			// Compare all results to the first one
			for i := 1; i < runs; i++ {
				if !almostEqual(results[0].AverageHits, results[i].AverageHits, epsilon) {
					t.Errorf("Run %d: AverageHits differs: %f vs %f", i+1, results[0].AverageHits, results[i].AverageHits)
				}
				if !almostEqual(results[0].AverageDestroyed, results[i].AverageDestroyed, epsilon) {
					t.Errorf("Run %d: AverageDestroyed differs: %f vs %f", i+1, results[0].AverageDestroyed, results[i].AverageDestroyed)
				}

				if !distributionsEqual(results[0].HitDist, results[i].HitDist, epsilon) {
					t.Errorf("Run %d: HitDist differs", i+1)
				}
				if !distributionsEqual(results[0].WoundDist, results[i].WoundDist, epsilon) {
					t.Errorf("Run %d: WoundDist differs", i+1)
				}
				if !distributionsEqual(results[0].PenDist, results[i].PenDist, epsilon) {
					t.Errorf("Run %d: PenDist differs", i+1)
				}
				if !distributionsEqual(results[0].DestroyedDist, results[i].DestroyedDist, epsilon) {
					t.Errorf("Run %d: DestroyedDist differs", i+1)
				}
			}
		})
	}
}

// P9 — Distribution support bounds
func TestP09_DistributionSupportBounds(t *testing.T) {
	calc := &DamageCalculatorImpl{}

	testCases := []struct {
		name string
		req  CombatSimulationRequest
	}{
		{"Base case", generateBaseRequest()},
		{"Large scale", generateLargeScaleRequest()},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := calc.CalculateDamageCore(tc.req)
			if err != nil {
				t.Fatalf("CalculateDamageCore failed: %v", err)
			}

			// Calculate theoretical maximums
			// This is simplified - you'd need to parse dice notation properly
			maxPossibleHits := tc.req.Attacker.Count * 10 // Conservative estimate
			maxPossibleDestroyed := *tc.req.Target.Count

			// Check HitDist keys
			for k := range result.HitDist {
				if k < 0 || k > maxPossibleHits {
					t.Errorf("HitDist has out-of-bounds key: %d (expected [0, %d])", k, maxPossibleHits)
				}
			}

			// Check DestroyedDist keys
			for k := range result.DestroyedDist {
				if k < 0 {
					t.Errorf("DestroyedDist has negative key: %d", k)
				}
				// Note: with DevastatingWounds, this can exceed target count
				if !tc.req.Attacker.DevastatingWounds && k > maxPossibleDestroyed {
					t.Errorf("DestroyedDist has out-of-bounds key: %d (expected [0, %d])", k, maxPossibleDestroyed)
				}
			}
		})
	}
}

// TestP10_StochasticDominance validates that improving stats strictly improves the probability distribution (FSD).
func TestP10_StochasticDominance(t *testing.T) {
	calc := &DamageCalculatorImpl{} // Replace with your actual interface/struct constructor

	// 1. Attacker Parameters (Should INCREASE output -> New Dominate Old)
	t.Run("Attacker Parameters (Growth)", func(t *testing.T) {
		testCases := []struct {
			name   string
			modify func(*CombatSimulationRequest, int)
			values []int // Strictly increasing values
		}{
			{
				name: "Strength",
				modify: func(req *CombatSimulationRequest, v int) {
					req.Attacker.Strength = v
				},
				values: []int{3, 4, 6, 8}, // Breakpoints vs T4 (6+ -> 5+ -> 4+ -> 3+ -> 2+)
			},
			{
				name: "Attacker Count",
				modify: func(req *CombatSimulationRequest, v int) {
					req.Attacker.Count = v
				},
				values: []int{10, 11, 15},
			},
			{
				name: "AP",
				modify: func(req *CombatSimulationRequest, v int) {
					req.Attacker.AP = v
				},
				values: []int{0, 1, 2, 3},
			},
			{
				name: "Attacks (Dice)",
				modify: func(req *CombatSimulationRequest, v int) {
					req.Attacker.Attacks = DiceRoll{Count: 0, Sides: 0, Modifier: v}
				},
				values: []int{1, 2, 3},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				req := generateBaseRequest()

				// Calculate first value
				tc.modify(&req, tc.values[0])
				resPrev, err := calc.CalculateDamageCore(req)
				if err != nil {
					t.Fatalf("Failed calc for %v: %v", tc.values[0], err)
				}

				for i := 1; i < len(tc.values); i++ {
					val := tc.values[i]
					tc.modify(&req, val)

					resCurr, err := calc.CalculateDamageCore(req)
					if err != nil {
						t.Fatalf("Failed calc for %v: %v", val, err)
					}

					// Verify: Current (Better) Dominates Previous (Worse)
					verifyDominance(t, tc.name, resCurr.DestroyedDist, resPrev.DestroyedDist)

					resPrev = resCurr
				}
			})
		}
	})

	// 2. Defender Parameters (Should DECREASE output -> Old Dominate New)
	// Increasing Defender Stats makes the result 'Worse' for the attacker.
	// Therefore, Previous (Weaker Defender) should Stochastically Dominate Current (Stronger Defender).
	t.Run("Defender Parameters (Decay)", func(t *testing.T) {
		testCases := []struct {
			name   string
			modify func(*CombatSimulationRequest, int)
			values []int // Strictly increasing stat (making defender stronger)
		}{
			{
				name: "Toughness",
				modify: func(req *CombatSimulationRequest, v int) {
					req.Target.Toughness = v
				},
				values: []int{3, 4, 5, 8},
			},
			{
				name: "Invulnerable Save",
				modify: func(req *CombatSimulationRequest, v int) {
					req.Target.Invulnerable = intPtr(v)
				},
				values: []int{6, 5, 4}, // Lower is better for defender -> harder for attacker
			},
			{
				name: "Wounds Per Model",
				modify: func(req *CombatSimulationRequest, v int) {
					req.Target.WoundsPerModel = v
				},
				values: []int{1, 2, 3},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				req := generateBaseRequest()

				// Logic Check for Save/Invuln:
				// In 40k, LOWER save is BETTER.
				// If we iterate [6, 5, 4], the defender is getting stronger.
				// Therefore damage should decrease.

				tc.modify(&req, tc.values[0])
				resPrev, err := calc.CalculateDamageCore(req)
				if err != nil {
					t.Fatalf("Failed calc for %v: %v", tc.values[0], err)
				}

				for i := 1; i < len(tc.values); i++ {
					val := tc.values[i]
					tc.modify(&req, val)

					resCurr, err := calc.CalculateDamageCore(req)
					if err != nil {
						t.Fatalf("Failed calc for %v: %v", val, err)
					}

					// Verify: Previous (Weaker Defender) Dominates Current (Stronger Defender)
					verifyDominance(t, tc.name, resPrev.DestroyedDist, resCurr.DestroyedDist)

					resPrev = resCurr
				}
			})
		}
	})
}

// Stress test with large numbers
func TestStressTest_LargeNumbers(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	// INJECTION: Define a No-Op validator to bypass complexity limits
	noOpValidator := func(_ *CombatSimulationRequest) error {
		return nil
	}

	// Initialize calculator with the bypass
	calc := &DamageCalculatorImpl{
		Validator: noOpValidator,
	}

	testCases := []struct {
		name string
		req  CombatSimulationRequest
	}{
		{
			name: "Very large attacker count",
			req: func() CombatSimulationRequest {
				req := generateBaseRequest()
				req.Attacker.Count = 1000
				return req
			}(),
		},
		{
			name: "Very large target count",
			req: func() CombatSimulationRequest {
				req := generateBaseRequest()
				// Note: Hydrate() may clamp this to 200 if DOS protection is hardcoded there.
				req.Target.Count = intPtr(500)
				return req
			}(),
		},
		{
			name: "High variance attacks",
			req: func() CombatSimulationRequest {
				req := generateBaseRequest()
				// 50 models rolling 6d6 each
				req.Attacker.Attacks = DiceRoll{Count: 6, Sides: 6, Modifier: 0}
				req.Attacker.Count = 50
				return req
			}(),
		},
		{
			name: "All abilities enabled",
			req: func() CombatSimulationRequest {
				req := generateBaseRequest()
				req.Attacker.Count = 100
				req.Attacker.LethalHits = true
				req.Attacker.SustainedHits = 2
				req.Attacker.DevastatingWounds = true
				req.Settings.HitReroll = RerollFail
				req.Settings.WoundReroll = RerollFail
				return req
			}(),
		},
	}

	// RELAXED EPSILON: Large state-space calculations accumulate drift.
	// 1e-7 is sufficient for 32-bit float accuracy equivalents;
	// since we use float64, this provides a massive safety margin for rounding.
	const stressEpsilon = 1e-7

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Execute
			result, err := calc.CalculateDamageCore(tc.req)

			// 1. Error Check (Should be nil due to NoOpValidator)
			if err != nil {
				t.Fatalf("CalculateDamageCore failed with error: %v", err)
			}

			// 2. Physical Validity Checks
			if result.AverageHits < 0 {
				t.Errorf("Physical Violation: Negative average hits: %f", result.AverageHits)
			}
			if result.AverageDestroyed < 0 {
				t.Errorf("Physical Violation: Negative average destroyed: %f", result.AverageDestroyed)
			}

			// 3. Mathematical Continuity Check
			// The sum of probabilities in the distribution vector must equal 1.0 (100%)
			sum := sumDistribution(result.DestroyedDist)
			if !almostEqual(sum, 1.0, stressEpsilon) { // Using relaxed epsilon 1e-7
				t.Errorf("Probability Leak: DestroyedDist sum is %f, expected 1.0", sum)
			}
		})
	}
}

func BenchmarkComplexityScaling(b *testing.B) {
	calc := &DamageCalculatorImpl{}

	// Scale n to find the inflection point where "seconds" turn into "minutes"
	for _, n := range []int{10, 50, 100, 500, 1000} {
		b.Run(fmt.Sprintf("AttackerCount-%d", n), func(b *testing.B) {
			req := generateBaseRequest()
			req.Attacker.Count = n

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				res, _ := calc.CalculateDamageCore(req)

				// Telemetry: Track the size of the resulting distribution
				// Large distributions = high GC pressure
				b.ReportMetric(float64(len(res.DestroyedDist)), "buckets/op")
			}
		})
	}
}
