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
