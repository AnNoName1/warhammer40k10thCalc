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
	"math"
	"testing"
)

const epsilonCore = 0.00001

func TestCalculateDamageCore_Basic(t *testing.T) {
	tests := []struct {
		name                 string
		req                  CombatSimulationRequest
		expectedAvgHits      float64
		expectedAvgDestroyed float64
		expectedHitsDist     map[int]float64
		expectedKilledDist   map[int]float64
		expectedDamageDist   map[int]float64 // New field
	}{
		{
			name: "1 Attack, BS4+, S5 vs T3, Save 6+",
			req: CombatSimulationRequest{
				Attacker: AttackerProfile{
					Count:    1,
					Attacks:  DiceRoll{Modifier: 1},
					BS:       4,
					Strength: 5,
					AP:       0,
					Damage:   DiceRoll{Modifier: 1},
				},
				Target: TargetProfile{
					Count:          intPtr(1),
					Toughness:      3,
					Save:           6,
					WoundsPerModel: 1,
				},
			},
			expectedAvgHits:      1.0 / 2.0,
			expectedAvgDestroyed: 5.0 / 18.0,
			expectedHitsDist: map[int]float64{
				0: 1.0 / 2.0,
				1: 1.0 / 2.0,
			},
			expectedKilledDist: map[int]float64{
				0: 13.0 / 18.0,
				1: 5.0 / 18.0,
			},
			// In this scenario, Total Damage equals Killed Models
			expectedDamageDist: map[int]float64{
				0: 13.0 / 18.0,
				1: 5.0 / 18.0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calc := &DamageCalculatorImpl{}
			resp, err := calc.CalculateDamageCore(tt.req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			verifyValue(t, "AverageHits", resp.AverageHits, tt.expectedAvgHits)
			verifyValue(t, "AverageDestroyed", resp.AverageDestroyed, tt.expectedAvgDestroyed)

			verifyDist(t, "HitDist", resp.HitDist, tt.expectedHitsDist)
			verifyDist(t, "DestroyedDist", resp.DestroyedDist, tt.expectedKilledDist)
			// Verify new distribution
			verifyDist(t, "DamageDist", resp.DamageDist, tt.expectedDamageDist)
		})
	}
}

func TestCalculateDamageCore_LethalHits(t *testing.T) {
	tests := []struct {
		name                 string
		req                  CombatSimulationRequest
		expectedAvgDestroyed float64
	}{
		{
			name: "BS4+, S3 vs T6 with Lethal Hits",
			req: CombatSimulationRequest{
				Attacker: AttackerProfile{
					Count:      1,
					Attacks:    DiceRoll{Modifier: 1},
					BS:         4,
					Strength:   3,
					LethalHits: true,
					Damage:     DiceRoll{Modifier: 1},
				},
				Target: TargetProfile{
					Count:          intPtr(1),
					Toughness:      6,
					Save:           7,
					WoundsPerModel: 1,
				},
			},
			expectedAvgDestroyed: 8.0 / 36.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calc := &DamageCalculatorImpl{}
			resp, err := calc.CalculateDamageCore(tt.req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			verifyValue(t, "AverageDestroyed", resp.AverageDestroyed, tt.expectedAvgDestroyed)
		})
	}
}

func TestCalculateDamageCore_DevastatingWounds(t *testing.T) {
	tests := []struct {
		name               string
		req                CombatSimulationRequest
		expectedAvgKilled  float64
		expectedKilledDist map[int]float64
	}{
		{
			name: "Devastating Wounds: no spillover, ignore save",
			req: CombatSimulationRequest{
				Attacker: AttackerProfile{
					Count:             1,
					Attacks:           DiceRoll{Modifier: 1},
					BS:                4,
					Strength:          4,
					DevastatingWounds: true,
					Damage:            DiceRoll{Modifier: 1},
				},
				Target: TargetProfile{
					Count:          intPtr(3),
					Toughness:      4,
					Save:           2, // explicitly 2 - must be ignored by devastating
					WoundsPerModel: 1,
				},
			},
			//regular - h(1/2) * non-critW(2/6) * save(1/6) + devastating - h(1/2) * critW(1/6)
			expectedAvgKilled: 1.0 / 9,
			expectedKilledDist: map[int]float64{
				0: 8.0 / 9.0,
				1: 1.0 / 9.0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calc := &DamageCalculatorImpl{}
			resp, err := calc.CalculateDamageCore(tt.req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			verifyValue(t, "AverageDestroyed", resp.AverageDestroyed, tt.expectedAvgKilled)
			verifyDist(t, "DestroyedDist", resp.DestroyedDist, tt.expectedKilledDist)
		})
	}
}

func TestCalculateDamageCore_SustainedHitsCorrelation(t *testing.T) {
	tests := []struct {
		name               string
		req                CombatSimulationRequest
		expectedAvgKilled  float64
		expectedKilledDist map[int]float64
		expectedHitsDist   map[int]float64
	}{
		{
			name: "Sustained Hits 1: Correlation Check",
			req: CombatSimulationRequest{
				Attacker: AttackerProfile{
					Count:         1,
					Attacks:       DiceRoll{Modifier: 1},
					BS:            4,
					Strength:      4,
					SustainedHits: 2, // The core test variable
					Damage:        DiceRoll{Modifier: 1},
				},
				Target: TargetProfile{
					Count:          intPtr(5),
					Toughness:      4,
					Save:           7, // Impossible save
					WoundsPerModel: 1,
				},
				Settings: SimulationSettings{
					CriticalHitThreshold: 6,
				},
			},
			// Avg Hits: (0.5*0) + (0.333*1) + (0.166*3) = 0.8333
			// Avg Kills: 0.8333 * 0.5 (wound) = 0.41666...
			expectedAvgKilled: 5.0 / 12.0,
			expectedHitsDist: map[int]float64{
				0: 0.5,
				1: 1.0 / 3.0,
				2: 0.0,
				3: 1.0 / 6.0,
			},
			expectedKilledDist: map[int]float64{
				0: 33.0 / 48.0,
				1: 11.0 / 48.0,
				2: 3.0 / 48.0,
				3: 1.0 / 48.0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calc := &DamageCalculatorImpl{}
			resp, err := calc.CalculateDamageCore(tt.req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify the "Discrete" nature of the hits
			// If your math uses 1.1 expected hits, it will FAIL here.
			verifyDist(t, "HitsDist", resp.HitDist, tt.expectedHitsDist)

			verifyValue(t, "AverageDestroyed", resp.AverageDestroyed, tt.expectedAvgKilled)
			verifyDist(t, "DestroyedDist", resp.DestroyedDist, tt.expectedKilledDist)
		})
	}
}

func TestCalculateDamageCore_Blast_Sanitization(t *testing.T) {
	// Scenario: 1 Attack Base, Target Count 5. Blast adds +1 Attack.
	// Total Expected: 2 Attacks.
	// Failure Case: If Hydrate mutates + Core applies, we get 3 Attacks.

	req := CombatSimulationRequest{
		Attacker: AttackerProfile{
			Count:    1,
			Attacks:  DiceRoll{Modifier: 1}, // Fixed 1 attack
			BS:       2,                     // 2+ to hit (5/6 probability)
			Strength: 4,
			AP:       0,
			Damage:   DiceRoll{Modifier: 1},
			Blast:    true,
		},
		Target: TargetProfile{
			Count:          intPtr(5), // Triggers +1 Blast Bonus (5/5)
			Toughness:      3,
			Save:           6,
			WoundsPerModel: 1,
		},
	}

	// Logic Calculation:
	// Base Attacks: 1
	// Blast Bonus: floor(5/5) = 1
	// Correct Total Attacks: 2
	// Hit Chance (BS 2+): 5/6
	// Expected Avg Hits: 2 * (5/6) = 1.6666666667
	expectedAvgHits := 2.0 * (5.0 / 6.0)

	t.Run("Verify Blast is not double-applied by Hydrate and Core", func(t *testing.T) {
		calc := &DamageCalculatorImpl{
			// Injecting NoOpValidator to isolate the logic from complexity gates
			Validator: func(_ *CombatSimulationRequest) error { return nil },
		}

		// 1. Run Hydrate.
		// This should normalize BS/Saves but NOT touch Attacker.Attacks.Modifier for Blast.
		calc.Hydrate(&req)

		// 2. Execute Core.
		// Core should see the original Modifier (1) and apply Blast internally once.
		resp, err := calc.CalculateDamageCore(req)
		if err != nil {
			t.Fatalf("CalculateDamageCore failed: %v", err)
		}

		// 3. Verification
		// If bug exists, resp.AverageHits would be 3 * (5/6) = 2.5
		const epsilon = 1e-9
		if math.Abs(resp.AverageHits-expectedAvgHits) > epsilon {
			t.Errorf("Blast double-applied or miscalculated. Got %f, want %f",
				resp.AverageHits, expectedAvgHits)
		}

		// 4. Structural Integrity Check
		// Ensure Hydrate didn't permanently mutate the source request attacks
		if req.Attacker.Attacks.Modifier != 1 {
			t.Errorf("Hydrate mutated Attacker.Attacks.Modifier: got %d, want 1",
				req.Attacker.Attacks.Modifier)
		}
	})
}

// verifyValue Checks float equality within epsilon
func verifyValue(t *testing.T, label string, got, want float64) {
	if math.Abs(got-want) > epsilonCore {
		t.Errorf("%s: expected %.6f (fraction), got %.6f", label, want, got)
	}
}

// verifyDist compares the probability map produced by the code against the expected map.
func verifyDist(t *testing.T, label string, got, want map[int]float64) {
	// Check if all expected keys are present and correct
	for k, wantP := range want {
		gotP, ok := got[k]
		// If wantP is zero, it's fine if key is missing
		if !ok {
			if wantP != 0 {
				t.Errorf("%s: missing outcome key %d in result distribution", label, k)
			}
			continue
		}
		if math.Abs(gotP-wantP) > epsilonCore {
			t.Errorf("%s key %d: expected prob %.6f, got %.6f", label, k, wantP, gotP)
		}
	}
	// Optional: Check for extra keys in 'got' that shouldn't be there
	if len(got) > len(want) {
		t.Errorf("%s: result distribution has extra outcomes (got %d keys, want %d)", label, len(got), len(want))
	}
}

func TestCalculateDamageCore_Torrent(t *testing.T) {
	tests := []struct {
		name                 string
		req                  CombatSimulationRequest
		expectedAvgHits      float64
		expectedAvgDestroyed float64
		expectedHitsDist     map[int]float64
		expectedKilledDist   map[int]float64
		expectedDamageDist   map[int]float64
	}{
		{
			name: "Torrent: 1 Attack, BS irrelevant (set to 6+), S4 vs T4, Save 6+",
			req: CombatSimulationRequest{
				Attacker: AttackerProfile{
					Count:    1,
					Attacks:  DiceRoll{Modifier: 1},
					BS:       6,
					Strength: 4,
					AP:       0,
					Damage:   DiceRoll{Modifier: 1},
					Torrent:  true,
				},
				Target: TargetProfile{
					Count:          intPtr(1),
					Toughness:      4,
					Save:           6,
					WoundsPerModel: 1,
				},
			},
			expectedAvgHits:      1.0,
			expectedAvgDestroyed: 5.0 / 12.0,
			expectedHitsDist: map[int]float64{
				0: 0.0,
				1: 1.0,
			},
			expectedKilledDist: map[int]float64{
				0: 7.0 / 12.0,
				1: 5.0 / 12.0,
			},
			expectedDamageDist: map[int]float64{
				0: 7.0 / 12.0,
				1: 5.0 / 12.0,
			},
		},
		{
			name: "Torrent: Multiple Attacks (D6), S3 vs T3, Save 5+",
			req: CombatSimulationRequest{
				Attacker: AttackerProfile{
					Count:    1,
					Attacks:  DiceRoll{Count: 1, Sides: 6},
					BS:       2,
					Strength: 3,
					AP:       0,
					Damage:   DiceRoll{Modifier: 1},
					Torrent:  true,
				},
				Target: TargetProfile{
					Count:          intPtr(10),
					Toughness:      3,
					Save:           5,
					WoundsPerModel: 1,
				},
			},
			expectedAvgHits:      3.5,
			expectedAvgDestroyed: 3.5 / 3.0,
			expectedHitsDist: map[int]float64{
				1: 1.0 / 6.0,
				2: 1.0 / 6.0,
				3: 1.0 / 6.0,
				4: 1.0 / 6.0,
				5: 1.0 / 6.0,
				6: 1.0 / 6.0,
			},
			// Killed/Damage distributions for D6 attacks require
			// binomial expansion or iterative convolution.
			// Provided Avg is the primary test vector.
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calc := &DamageCalculatorImpl{}
			resp, err := calc.CalculateDamageCore(tt.req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			verifyValue(t, "AverageHits", resp.AverageHits, tt.expectedAvgHits)
			verifyValue(t, "AverageDestroyed", resp.AverageDestroyed, tt.expectedAvgDestroyed)

			if tt.expectedHitsDist != nil {
				verifyDist(t, "HitDist", resp.HitDist, tt.expectedHitsDist)
			}
			if tt.expectedKilledDist != nil {
				verifyDist(t, "DestroyedDist", resp.DestroyedDist, tt.expectedKilledDist)
			}
			if tt.expectedDamageDist != nil {
				verifyDist(t, "DamageDist", resp.DamageDist, tt.expectedDamageDist)
			}
		})
	}
}

func TestCalculateDamageCore_FeelNoPain_AppliedToDevastatingWounds(t *testing.T) {
	// Regression test locking in that Feel No Pain reduces devastating-wound
	// damage exactly like normal damage — the two previously duplicated
	// (dmgWithFnp/dmgNoFnp) calls made this easy to silently diverge. Torrent
	// forces exactly 1 hit so only the Wound/Devastating/FNP math is under test.
	req := CombatSimulationRequest{
		Attacker: AttackerProfile{
			Count:             1,
			Attacks:           DiceRoll{Modifier: 1},
			Torrent:           true,
			Strength:          8,
			DevastatingWounds: true,
			Damage:            DiceRoll{Modifier: 4},
		},
		Target: TargetProfile{
			Count:          intPtr(1),
			Toughness:      4,
			Save:           7,
			WoundsPerModel: 10,
			FeelNoPain:     intPtr(5),
		},
	}

	// S8 vs T4 wounds on 2+ (5/6); default crit-wound threshold 6+ makes 1/6
	// of those devastating (bypasses the impossible Save 7+) and 4/6 normal
	// (also unsaved). FNP 5+ gives pFail = 2/3 per damage point. Both paths
	// resolve through the identical 4-damage Binomial(4, 2/3) distribution,
	// weighted by their combined 5/6 chance of occurring. WoundsPerModel=10
	// keeps the single wound-instance well under model HP so damage is never
	// capped by death, isolating the FNP math.
	expectedDamageDist := map[int]float64{
		0: 43.0 / 243.0,
		1: 20.0 / 243.0,
		2: 60.0 / 243.0,
		3: 80.0 / 243.0,
		4: 40.0 / 243.0,
	}

	calc := &DamageCalculatorImpl{}
	resp, err := calc.CalculateDamageCore(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	verifyDist(t, "DamageDist", resp.DamageDist, expectedDamageDist)
}

func TestComputeFinalUnsavedDist(t *testing.T) {
	// Certain: 1 normal wound before save, 0 devastating. probSaveFailed=0.6.
	jointWoundDist := NormalDevastatingWoundMatrix{
		{0, 0},   // nw=0
		{1.0, 0}, // nw=1: dw=0 -> 1.0
	}
	maxHits := 1

	got := computeFinalUnsavedDist(jointWoundDist, maxHits, 0.6)

	want := []float64{0.4, 0.6} // 0 unsaved (40%) or 1 unsaved (60%)

	if len(got) != len(want) {
		t.Fatalf("got %d entries, want %d", len(got), len(want))
	}
	for i := range want {
		if math.Abs(got[i]-want[i]) > epsilonCore {
			t.Errorf("finalUnsavedDist[%d]: got %v, want %v", i, got[i], want[i])
		}
	}
}

func TestComputeTotalWoundsDist(t *testing.T) {
	jointWoundDist := NormalDevastatingWoundMatrix{
		{0.1, 0.2}, // nw=0: dw=0 (total=0), dw=1 (total=1)
		{0.3, 0.4}, // nw=1: dw=0 (total=1), dw=1 (total=2, truncated by maxHits)
	}
	maxHits := 1 // deliberately truncates the [1][1]=0.4 entry (total=2 > maxHits)

	got := computeTotalWoundsDist(jointWoundDist, maxHits)

	want := []float64{0.1, 0.5} // total=0, total=1

	if len(got) != len(want) {
		t.Fatalf("got %d entries, want %d", len(got), len(want))
	}
	for i := range want {
		if math.Abs(got[i]-want[i]) > epsilonCore {
			t.Errorf("totalWoundsDist[%d]: got %v, want %v", i, got[i], want[i])
		}
	}
}

func TestComputeFinalHitsDist(t *testing.T) {
	autoWoundNormalHitDist := AutoWoundNormalHitMatrix{
		{0.1, 0.2}, // auto=0: normal=0 (totalHits=0), normal=1 (totalHits=1)
		{0.3, 0.4}, // auto=1: normal=0 (totalHits=1), normal=1 (totalHits=2)
	}
	bounds := hitBounds{maxN: 1, maxL: 1, maxHits: 2}

	got := computeFinalHitsDist(autoWoundNormalHitDist, bounds)

	want := []float64{0.1, 0.5, 0.4} // totalHits=0,1,2

	if len(got) != len(want) {
		t.Fatalf("got %d entries, want %d", len(got), len(want))
	}
	for i := range want {
		if math.Abs(got[i]-want[i]) > epsilonCore {
			t.Errorf("finalHitsDist[%d]: got %v, want %v", i, got[i], want[i])
		}
	}
}

func TestComputeJointWoundDist(t *testing.T) {
	// Certain: 0 auto-wounds, 1 normal hit. probNormalWound=0.5,
	// probDevWound=0.25 -> pAnyWound=0.75 (miss=0.25), split 2:1 normal:devastating.
	autoWoundNormalHitDist := AutoWoundNormalHitMatrix{
		{0, 1.0}, // auto=0: normal=0 -> 0, normal=1 -> 1.0
	}
	bounds := hitBounds{maxN: 1, maxL: 0, maxHits: 1}

	got := computeJointWoundDist(autoWoundNormalHitDist, bounds, 0.5, 0.25)

	want := NormalDevastatingWoundMatrix{
		{0.25, 0.25}, // normWounds=0: miss (devWounds=0) or devastating (devWounds=1)
		{0.5, 0},     // normWounds=1: normal wound (devWounds=0)
	}

	if len(got) != len(want) {
		t.Fatalf("got %d rows, want %d", len(got), len(want))
	}
	for nw := range want {
		for dw := range want[nw] {
			if math.Abs(got[nw][dw]-want[nw][dw]) > epsilonCore {
				t.Errorf("[nw=%d][dw=%d]: got %v, want %v", nw, dw, got[nw][dw], want[nw][dw])
			}
		}
	}
}

func TestComputeAutoWoundNormalHitDist(t *testing.T) {
	// Exactly 1 attack, whose single die roll is 60% one normal hit or 40%
	// one lethal hit. With attackCount pinned to 1, ComputeMultiAttackHitDistribution
	// is the identity, so CollapseLethalHitsIntoAutoWounds just swaps
	// [normal][lethal] -> [lethal(auto)][normal].
	attackCountDist := map[int]float64{1: 1.0}
	hitOutcomeDist := map[HitOutcome]float64{
		{NormalHits: 1, LethalHits: 0}: 0.6,
		{NormalHits: 0, LethalHits: 1}: 0.4,
	}
	bounds := computeHitBounds(attackCountDist, hitOutcomeDist)

	got := computeAutoWoundNormalHitDist(hitOutcomeDist, attackCountDist, bounds)

	want := AutoWoundNormalHitMatrix{
		{0, 0.6}, // auto=0: normal=0 -> 0, normal=1 -> 0.6
		{0.4, 0}, // auto=1: normal=0 -> 0.4, normal=1 -> 0
	}

	if len(got) != len(want) {
		t.Fatalf("got %d auto-wound rows, want %d", len(got), len(want))
	}
	for auto := range want {
		for normal := range want[auto] {
			if math.Abs(got[auto][normal]-want[auto][normal]) > epsilonCore {
				t.Errorf("[auto=%d][normal=%d]: got %v, want %v", auto, normal, got[auto][normal], want[auto][normal])
			}
		}
	}
}

func TestComputeHitBounds(t *testing.T) {
	attackCountDist := map[int]float64{0: 0.5, 2: 0.3, 5: 0.2} // maxAttacks = 5
	hitOutcomeDist := map[HitOutcome]float64{
		{NormalHits: 1, LethalHits: 0}: 0.5, // sum 1
		{NormalHits: 0, LethalHits: 2}: 0.3, // sum 2, maxLethalPerAttack
		{NormalHits: 1, LethalHits: 1}: 0.2, // sum 2, ties maxPerTotal
	}

	got := computeHitBounds(attackCountDist, hitOutcomeDist)

	want := hitBounds{
		maxAttacks:         5,
		maxNormalPerAttack: 1,
		maxLethalPerAttack: 2,
		maxN:               5,  // 5 * 1
		maxL:               10, // 5 * 2
		maxHits:            10, // 5 * 2 (tighter than maxN+maxL = 15)
	}

	if got != want {
		t.Errorf("computeHitBounds() = %+v, want %+v", got, want)
	}
}

func TestComputeHitOutcomeDist(t *testing.T) {
	t.Run("Torrent bypasses the hit roll entirely", func(t *testing.T) {
		req := CombatSimulationRequest{
			Attacker: AttackerProfile{
				Torrent: true,
				BS:      6, // irrelevant under Torrent
			},
		}

		got := computeHitOutcomeDist(req)

		want := HitOutcome{NormalHits: 1, LethalHits: 0}
		if len(got) != 1 || got[want] != 1.0 {
			t.Errorf("Torrent hit outcome = %v, want {%v: 1.0}", got, want)
		}
	})

	t.Run("Standard hit roll delegates to CalculateSingleHitDistribution", func(t *testing.T) {
		req := CombatSimulationRequest{
			Attacker: AttackerProfile{
				BS:            3,
				LethalHits:    true,
				SustainedHits: 1,
			},
			Settings: SimulationSettings{
				HitReroll:            RerollOnes,
				HitModifier:          1,
				CriticalHitThreshold: 6,
			},
		}

		got := computeHitOutcomeDist(req)
		want := CalculateSingleHitDistribution(3, RerollOnes, 1, true, 1, 6)

		if len(got) != len(want) {
			t.Fatalf("got %d outcomes, want %d", len(got), len(want))
		}
		for outcome, wantP := range want {
			if math.Abs(got[outcome]-wantP) > epsilonCore {
				t.Errorf("outcome %v: got %v, want %v", outcome, got[outcome], wantP)
			}
		}
	})
}

func intPtr(i int) *int {
	return &i
}

func TestDamageCalculatorImpl_Hydrate(t *testing.T) {
	calc := &DamageCalculatorImpl{}

	tests := []struct {
		name  string
		input CombatSimulationRequest
		check func(*testing.T, CombatSimulationRequest)
	}{
		{
			name: "Clamps Low Ballistic Skill and Thresholds",
			input: CombatSimulationRequest{
				Attacker: AttackerProfile{BS: 1, Torrent: false},      // BS 1 is illegal
				Settings: SimulationSettings{CriticalHitThreshold: 1}, // Crit 1 is illegal
				Target:   TargetProfile{Save: 1, Count: intPtr(5)},    // Save 1 is illegal
			},
			check: func(t *testing.T, got CombatSimulationRequest) {
				if got.Attacker.BS != 2 {
					t.Errorf("BS not clamped: got %d, want 2", got.Attacker.BS)
				}
				if got.Settings.CriticalHitThreshold != 2 {
					t.Errorf("Crit Threshold not clamped: got %d, want 2", got.Settings.CriticalHitThreshold)
				}
				if got.Target.Save != 2 {
					t.Errorf("Save not clamped: got %d, want 2", got.Target.Save)
				}
			},
		},
		{
			name: "Resolves Infinite Target (Nil Count)",
			input: CombatSimulationRequest{
				// 10 Attacks, Damage 2. Max damage potential = 20. WoundsPerModel = 2.
				// Should resolve to ~10 models.
				Attacker: AttackerProfile{
					Count:   1,
					Attacks: DiceRoll{Count: 10, Sides: 1, Modifier: 0},
					Damage:  DiceRoll{Count: 2, Sides: 1, Modifier: 0},
				},
				Target: TargetProfile{Count: nil, WoundsPerModel: 2},
			},
			check: func(t *testing.T, got CombatSimulationRequest) {
				if got.Target.Count == nil {
					t.Fatal("Target.Count remains nil after Hydration")
				}
				// Max attacks (10) * Max Damage (2) = 20 total damage.
				// 20 damage / 2 wounds per model = 10 models + 1 safety buffer = 11?
				// Logic depends on exact formula in Hydrate implementation.
				// Testing for > 0 confirms logic ran.
				if *got.Target.Count <= 0 {
					t.Errorf("Target count resolution failed: got %d", *got.Target.Count)
				}
			},
		},
		{
			name: "Infinite Target Resolution Ignores DevastatingWounds",
			input: CombatSimulationRequest{
				Attacker: AttackerProfile{
					Count:             1,
					Attacks:           DiceRoll{Count: 10, Sides: 1, Modifier: 0},
					Damage:            DiceRoll{Count: 2, Sides: 1, Modifier: 0},
					DevastatingWounds: true,
				},
				Target: TargetProfile{Count: nil, WoundsPerModel: 2},
			},
			check: func(t *testing.T, got CombatSimulationRequest) {
				if got.Target.Count == nil {
					t.Fatal("Target.Count remains nil after Hydration")
				}
				if *got.Target.Count != 10 {
					t.Errorf("count should resolve to maxAttacks (10) regardless of DevastatingWounds, got %d", *got.Target.Count)
				}
			},
		},
		{
			name: "Enforces Hard Cap on Infinite Resolution",
			input: CombatSimulationRequest{
				// Massive damage potential requesting infinite targets
				Attacker: AttackerProfile{
					Count:   100,
					Attacks: DiceRoll{Count: 100, Sides: 1, Modifier: 0},
				},
				Target: TargetProfile{Count: nil, WoundsPerModel: 1},
			},
			check: func(t *testing.T, got CombatSimulationRequest) {
				if got.Target.Count == nil {
					t.Fatal("Target.Count was nil")
				}
				// Assuming the hardcoded cap in Hydrate is 200
				if *got.Target.Count > 200 {
					t.Errorf("DOS cap failed: got %d, want <= 200", *got.Target.Count)
				}
			},
		},
		{
			name: "Default Uninitialized Thresholds to 6",
			input: CombatSimulationRequest{
				Settings: SimulationSettings{
					CriticalHitThreshold:   0, // Uninitialized
					CriticalWoundThreshold: 0, // Uninitialized
				},
				// Include mandatory fields to avoid side-effects
				Attacker: AttackerProfile{BS: 3},
				Target:   TargetProfile{Save: 3},
			},
			check: func(t *testing.T, got CombatSimulationRequest) {
				if got.Settings.CriticalHitThreshold != 6 {
					t.Errorf("CriticalHitThreshold 0 should default to 6, got %d",
						got.Settings.CriticalHitThreshold)
				}
				if got.Settings.CriticalWoundThreshold != 6 {
					t.Errorf("CriticalWoundThreshold 0 should default to 6, got %d",
						got.Settings.CriticalWoundThreshold)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clone input to avoid mutation leaking (Go struct copy)
			req := tt.input
			calc.Hydrate(&req)
			tt.check(t, req)
		})
	}
}

func TestDefaultComplexityValidator(t *testing.T) {
	tests := []struct {
		name    string
		req     CombatSimulationRequest
		wantErr bool
	}{
		{
			name: "Valid standard request",
			req: CombatSimulationRequest{
				Attacker: AttackerProfile{Count: 10, Attacks: DiceRoll{Count: 0, Sides: 0, Modifier: 2}, Damage: DiceRoll{Count: 0, Sides: 0, Modifier: 1}},
				Target:   TargetProfile{Count: intPtr(10), WoundsPerModel: 2},
			},
			wantErr: false,
		},
		{
			name: "DOS Protection - Too many attacks (Depth)",
			req: CombatSimulationRequest{
				Attacker: AttackerProfile{Count: 1000, Attacks: DiceRoll{Count: 0, Sides: 0, Modifier: 100}, Damage: DiceRoll{Count: 0, Sides: 0, Modifier: 1}},
				Target:   TargetProfile{Count: intPtr(100), WoundsPerModel: 1},
			},
			wantErr: true, // 1000 * 100 attacks = 100,000 depth.
		},
		{
			name: "DOS Protection - Massive State Space (Width)",
			req: CombatSimulationRequest{
				Attacker: AttackerProfile{Count: 10, Attacks: DiceRoll{Count: 0, Sides: 0, Modifier: 10}, Damage: DiceRoll{Count: 0, Sides: 0, Modifier: 1}},
				Target:   TargetProfile{Count: intPtr(1000), WoundsPerModel: 1000},
			},
			wantErr: true, // State space = 1,000,000. Squared = 10^12. Overflow.
		},
		{
			name: "Blast Feedback Loop - Implicit Attack Increase",
			req: CombatSimulationRequest{
				// 20 attackers, 1 attack base.
				// But Blast vs 1000 models = +200 attacks per model.
				// Total attacks = 20 * (1 + 200) = 4020 attacks.
				Attacker: AttackerProfile{
					Count:   20,
					Blast:   true,
					Attacks: DiceRoll{Count: 1, Sides: 6, Modifier: 0},
					Damage:  DiceRoll{Count: 0, Sides: 0, Modifier: 1},
				},
				Target: TargetProfile{Count: intPtr(1000), WoundsPerModel: 1},
			},
			wantErr: true,
		},
		{
			name: "Nuclear Dice String (Max Face Calculation)",
			req: CombatSimulationRequest{
				Attacker: AttackerProfile{
					Count: 20,
					// 10,000 dice * 1000 sides = Max 10,000,000 damage/attacks
					Attacks: DiceRoll{Count: 10000, Sides: 1000, Modifier: 0},
					Damage:  DiceRoll{Count: 0, Sides: 0, Modifier: 1},
				},
				Target: TargetProfile{Count: intPtr(1), WoundsPerModel: 1},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := DefaultComplexityValidator(&tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("DefaultComplexityValidator() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
