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
	// If Sanitizer mutates + Core applies: 3 Attacks.

	tt := struct {
		name            string
		req             CombatSimulationRequest
		expectedAvgHits float64
	}{
		name: "Blast Sanitization Check: 1 Attack + Blast vs 5 Models",
		req: CombatSimulationRequest{
			Attacker: AttackerProfile{
				Count:    1,
				Attacks:  DiceRoll{Modifier: 1}, // Fixed 1 attack
				BS:       2,                     // 2+ to hit
				Strength: 4,
				AP:       0,
				Damage:   DiceRoll{Modifier: 1},
				Blast:    true, // The variable under test
			},
			Target: TargetProfile{
				Count:          intPtr(5), // Triggers +1 Bonus
				Toughness:      3,
				Save:           6,
				WoundsPerModel: 1,
			},
		},
		// Logic:
		// Base Attacks: 1
		// Blast Bonus: floor(5/5) = 1
		// Total Attacks: 2
		// Hit Chance (BS 2+): 5/6
		// Expected Avg Hits: 2 * (5/6) = 1.666...
		expectedAvgHits: 2.0 * (5.0 / 6.0),
	}

	t.Run(tt.name, func(t *testing.T) {
		calc := &DamageCalculatorImpl{}

		// 1. Explicitly run Sanitize first to ensure the bug is caught
		if err := calc.Sanitize(&tt.req); err != nil {
			t.Fatalf("Sanitize failed: %v", err)
		}

		// 2. Run Core
		resp, err := calc.CalculateDamageCore(tt.req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		verifyValue(t, "AverageHits", resp.AverageHits, tt.expectedAvgHits)
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

func TestDamageCalculatorImpl_Sanitize(t *testing.T) {
	calc := &DamageCalculatorImpl{}

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
			name: "Infinite target (nil/0) - Safe complexity",
			req: CombatSimulationRequest{
				Attacker: AttackerProfile{Count: 5, Attacks: DiceRoll{Count: 0, Sides: 0, Modifier: 1}, Damage: DiceRoll{Count: 0, Sides: 0, Modifier: 1}},
				Target:   TargetProfile{Count: nil, WoundsPerModel: 1}, // Virtual infinity
			},
			wantErr: false,
		},
		{
			name: "DOS Protection - Too many attacks",
			req: CombatSimulationRequest{
				Attacker: AttackerProfile{Count: 1000, Attacks: DiceRoll{Count: 0, Sides: 0, Modifier: 100}, Damage: DiceRoll{Count: 0, Sides: 0, Modifier: 1}},
				Target:   TargetProfile{Count: intPtr(100), WoundsPerModel: 1},
			},
			wantErr: true, // Should exceed 2,000,000 score
		},
		{
			name: "DOS Protection - Massive Health Pool",
			req: CombatSimulationRequest{
				Attacker: AttackerProfile{Count: 10, Attacks: DiceRoll{Count: 0, Sides: 0, Modifier: 10}, Damage: DiceRoll{Count: 0, Sides: 0, Modifier: 1}},
				Target:   TargetProfile{Count: intPtr(1000), WoundsPerModel: 1000},
			},
			wantErr: true,
		},
		{
			name: "Blast Feedback Loop - Large unit scaling",
			req: CombatSimulationRequest{
				// 20 attackers vs 1000 targets is a classic "horde killer" scenario
				Attacker: AttackerProfile{Count: 20, Attacks: DiceRoll{Count: 1, Sides: 6, Modifier: 0}, Damage: DiceRoll{Count: 0, Sides: 0, Modifier: 1}, Blast: true},
				Target:   TargetProfile{Count: intPtr(1000), WoundsPerModel: 1},
			},
			wantErr: true,
		},
		{
			name: "Nuclear Dice String",
			req: CombatSimulationRequest{
				Attacker: AttackerProfile{Count: 20, Attacks: DiceRoll{Count: 10000, Sides: 1000, Modifier: 0}, Damage: DiceRoll{Count: 0, Sides: 0, Modifier: 1}},
				Target:   TargetProfile{Count: intPtr(1), WoundsPerModel: 1},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := calc.Sanitize(&tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("Sanitize() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
