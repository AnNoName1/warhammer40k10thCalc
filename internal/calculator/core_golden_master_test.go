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

// TRANSITIONAL: this file exists only for the CalculateDamageCore extraction
// on branch refactor/extract-damage-core-pipeline (see EXTRACTION_PLAN.md).
// It answers a narrower question than core_test.go/core_rules_test.go do:
// "did output change at all across this specific refactor's commits". It is
// not permanent coverage and is not a replacement for those suites. Remove
// this file and testdata/golden_master.json in Phase 4, once the extraction
// is complete and verified.
package calculator

import (
	"encoding/json"
	"math"
	"os"
	"testing"
)

// goldenCase pairs a descriptive name with a request. Names must be stable
// across the branch — they're the map keys in testdata/golden_master.json.
type goldenCase struct {
	name string
	req  CombatSimulationRequest
}

// goldenNoOpValidator bypasses DefaultComplexityValidator. This harness
// exercises CalculateDamageCore's math, not the validator (which has its own
// coverage in TestDefaultComplexityValidator) — some grid cases (very large
// attacker/target counts) would otherwise be rejected before reaching the
// code under test.
func goldenNoOpValidator(_ *CombatSimulationRequest) error { return nil }

// goldenMasterCases is the input grid: a baseline plus a one-parameter-at-a-time
// sweep of every boolean/toggle/reroll the request surface exposes, boundary
// BS/S/T/AP/D/Save values (including guaranteed-hit and guaranteed-miss
// cases), attack-count scaling from 0 to stress-test scale, and a few
// combined "everything enabled" scenarios to catch interaction bugs a
// one-at-a-time sweep can't.
func goldenMasterCases() []goldenCase {
	base := generateBaseRequest() // Count 10, BS3+/S4/AP1/D1, T4/Sv3+/2W, no rerolls/abilities

	withTarget := func(req CombatSimulationRequest, n int) CombatSimulationRequest {
		req.Target.Count = intPtr(n)
		return req
	}

	cases := []goldenCase{
		{"baseline", base},

		// Zero-input edges (P07 territory, but snapshotting full distributions
		// rather than just asserting the {0:1} shape).
		{"zero_attacker_count", func() CombatSimulationRequest {
			req := base
			req.Attacker.Count = 0
			return req
		}()},
		{"zero_target_count", withTarget(base, 0)},

		// Attack-count scaling.
		{"attackers_1", func() CombatSimulationRequest { req := base; req.Attacker.Count = 1; return req }()},
		{"attackers_5", func() CombatSimulationRequest { req := base; req.Attacker.Count = 5; return req }()},
		{"attackers_100", func() CombatSimulationRequest { req := base; req.Attacker.Count = 100; return req }()},
		{"attackers_1000", func() CombatSimulationRequest { req := base; req.Attacker.Count = 1000; return req }()},

		// Boolean/ability toggles, one at a time against baseline.
		{"blast", func() CombatSimulationRequest {
			req := withTarget(base, 20)
			req.Attacker.Blast = true
			return req
		}()},
		{"lethal_hits", func() CombatSimulationRequest { req := base; req.Attacker.LethalHits = true; return req }()},
		{"devastating_wounds", func() CombatSimulationRequest { req := base; req.Attacker.DevastatingWounds = true; return req }()},
		{"torrent", func() CombatSimulationRequest { req := base; req.Attacker.Torrent = true; return req }()},
		{"sustained_hits_1", func() CombatSimulationRequest { req := base; req.Attacker.SustainedHits = 1; return req }()},
		{"sustained_hits_2", func() CombatSimulationRequest { req := base; req.Attacker.SustainedHits = 2; return req }()},
		{"has_cover", func() CombatSimulationRequest { req := base; req.Target.HasCover = true; return req }()},

		// Rerolls, one axis at a time.
		{"hit_reroll_ones", func() CombatSimulationRequest { req := base; req.Settings.HitReroll = RerollOnes; return req }()},
		{"hit_reroll_fail", func() CombatSimulationRequest { req := base; req.Settings.HitReroll = RerollFail; return req }()},
		{"wound_reroll_ones", func() CombatSimulationRequest { req := base; req.Settings.WoundReroll = RerollOnes; return req }()},
		{"wound_reroll_fail", func() CombatSimulationRequest { req := base; req.Settings.WoundReroll = RerollFail; return req }()},
		{"save_reroll_ones", func() CombatSimulationRequest { req := base; req.Settings.SaveReroll = RerollOnes; return req }()},
		{"save_reroll_fail", func() CombatSimulationRequest { req := base; req.Settings.SaveReroll = RerollFail; return req }()},

		// Feel No Pain / Invulnerable, weak and strong.
		{"fnp_weak_6", func() CombatSimulationRequest { req := base; req.Target.FeelNoPain = intPtr(6); return req }()},
		{"fnp_strong_2", func() CombatSimulationRequest { req := base; req.Target.FeelNoPain = intPtr(2); return req }()},
		{"invuln_weak_5", func() CombatSimulationRequest { req := base; req.Target.Invulnerable = intPtr(5); return req }()},
		{"invuln_strong_2", func() CombatSimulationRequest { req := base; req.Target.Invulnerable = intPtr(2); return req }()},

		// Boundary BS (best/worst possible, non-Torrent).
		{"bs_best_2", func() CombatSimulationRequest { req := base; req.Attacker.BS = 2; return req }()},
		{"bs_worst_6", func() CombatSimulationRequest { req := base; req.Attacker.BS = 6; return req }()},

		// Guaranteed-wound and near-guaranteed-miss Strength/Toughness pairs.
		{"guaranteed_wound", func() CombatSimulationRequest {
			req := base
			req.Attacker.Strength = 8 // S >= 2T -> wounds on 2+
			req.Target.Toughness = 4
			return req
		}()},
		{"worst_case_wound", func() CombatSimulationRequest {
			req := base
			req.Attacker.Strength = 2 // S <= T/2 -> wounds on 6+ only
			req.Target.Toughness = 8
			return req
		}()},

		// Boundary AP and Save.
		{"ap_zero", func() CombatSimulationRequest { req := base; req.Attacker.AP = 0; return req }()},
		{"ap_high_5", func() CombatSimulationRequest { req := base; req.Attacker.AP = 5; return req }()},
		{"save_best_2", func() CombatSimulationRequest { req := base; req.Target.Save = 2; return req }()},
		{"save_impossible_7", func() CombatSimulationRequest { req := base; req.Target.Save = 7; return req }()},

		// Multi-die attacks/damage (exercises the convolution paths, not just
		// fixed-value dice).
		{"multi_die_attacks", func() CombatSimulationRequest {
			req := base
			req.Attacker.Attacks = DiceRoll{Count: 2, Sides: 6}
			return req
		}()},
		{"multi_die_damage", func() CombatSimulationRequest {
			req := base
			req.Attacker.Damage = DiceRoll{Count: 1, Sides: 6, Modifier: 1}
			return req
		}()},

		// Combined scenarios: interactions a one-at-a-time sweep can't catch.
		{"all_abilities_enabled", func() CombatSimulationRequest {
			req := withTarget(base, 20)
			req.Attacker.Count = 100
			req.Attacker.LethalHits = true
			req.Attacker.SustainedHits = 2
			req.Attacker.DevastatingWounds = true
			req.Settings.HitReroll = RerollFail
			req.Settings.WoundReroll = RerollFail
			req.Settings.SaveReroll = RerollFail
			req.Target.FeelNoPain = intPtr(5)
			return req
		}()},
		{"blast_plus_devastating_plus_fnp", func() CombatSimulationRequest {
			req := withTarget(base, 30)
			req.Attacker.Blast = true
			req.Attacker.DevastatingWounds = true
			req.Target.FeelNoPain = intPtr(5)
			req.Target.Invulnerable = intPtr(4)
			return req
		}()},
		{"worst_case_for_attacker", func() CombatSimulationRequest {
			req := base
			req.Attacker.BS = 6
			req.Attacker.Strength = 2
			req.Target.Toughness = 8
			req.Target.Save = 2
			req.Target.Invulnerable = intPtr(2)
			req.Target.FeelNoPain = intPtr(2)
			return req
		}()},
		{"best_case_for_attacker", func() CombatSimulationRequest {
			req := base
			req.Attacker.BS = 2
			req.Attacker.Strength = 8
			req.Attacker.AP = 4
			req.Attacker.DevastatingWounds = true
			req.Target.Toughness = 3
			req.Target.Save = 6
			return req
		}()},
		{"torrent_plus_devastating_plus_sustained", func() CombatSimulationRequest {
			req := base
			req.Attacker.Torrent = true
			req.Attacker.DevastatingWounds = true
			req.Attacker.SustainedHits = 2 // no-op under Torrent, verifying it stays a no-op
			return req
		}()},
	}

	return cases
}

const (
	goldenFixturePath   = "testdata/golden_master.json"
	goldenEpsilon       = 1e-9
	goldenStressEpsilon = 1e-7 // for attacker/target counts >= 100, matches TestStressTest_LargeNumbers
)

// isGoldenStressCase reports whether a case is large-scale enough to warrant
// the relaxed tolerance, mirroring TestStressTest_LargeNumbers' rationale
// (large state-space accumulates float drift from summation order alone).
func isGoldenStressCase(req CombatSimulationRequest) bool {
	targetCount := 0
	if req.Target.Count != nil {
		targetCount = *req.Target.Count
	}
	return req.Attacker.Count >= 100 || targetCount >= 100
}

func computeGoldenResults(t *testing.T) map[string]SimulationResult {
	t.Helper()
	calc := &DamageCalculatorImpl{Validator: goldenNoOpValidator}

	results := make(map[string]SimulationResult)
	for _, c := range goldenMasterCases() {
		res, err := calc.CalculateDamageCore(c.req)
		if err != nil {
			t.Fatalf("case %q: CalculateDamageCore failed: %v", c.name, err)
		}
		results[c.name] = res
	}
	return results
}

// TestGoldenMaster_Capture writes testdata/golden_master.json from the
// current implementation. Run manually with GOLDEN_UPDATE=1 whenever the
// fixture needs to be (re)captured; it is a no-op otherwise so it doesn't
// interfere with normal `go test ./...` runs.
func TestGoldenMaster_Capture(t *testing.T) {
	if os.Getenv("GOLDEN_UPDATE") != "1" {
		t.Skip("set GOLDEN_UPDATE=1 to (re)capture testdata/golden_master.json")
	}

	results := computeGoldenResults(t)

	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal golden results: %v", err)
	}
	if err := os.WriteFile(goldenFixturePath, data, 0644); err != nil {
		t.Fatalf("failed to write %s: %v", goldenFixturePath, err)
	}
	t.Logf("captured %d golden cases to %s", len(results), goldenFixturePath)
}

// TestGoldenMaster_Verify compares the current implementation's output
// against the captured fixture, within tolerance. This is the test that
// runs during Phase 2's stage-by-stage extraction to catch any divergence.
func TestGoldenMaster_Verify(t *testing.T) {
	fixtureData, err := os.ReadFile(goldenFixturePath)
	if err != nil {
		t.Fatalf("failed to read %s (run with GOLDEN_UPDATE=1 first): %v", goldenFixturePath, err)
	}

	var golden map[string]SimulationResult
	if err := json.Unmarshal(fixtureData, &golden); err != nil {
		t.Fatalf("failed to unmarshal %s: %v", goldenFixturePath, err)
	}

	cases := goldenMasterCases()
	if len(golden) != len(cases) {
		t.Fatalf("fixture has %d cases, goldenMasterCases() has %d — re-capture with GOLDEN_UPDATE=1", len(golden), len(cases))
	}

	calc := &DamageCalculatorImpl{Validator: goldenNoOpValidator}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			want, ok := golden[c.name]
			if !ok {
				t.Fatalf("case %q missing from fixture — re-capture with GOLDEN_UPDATE=1", c.name)
			}

			got, err := calc.CalculateDamageCore(c.req)
			if err != nil {
				t.Fatalf("CalculateDamageCore failed: %v", err)
			}

			eps := goldenEpsilon
			if isGoldenStressCase(c.req) {
				eps = goldenStressEpsilon
			}

			verifyValue(t, "AverageHits", got.AverageHits, want.AverageHits)
			verifyValue(t, "AverageDestroyed", got.AverageDestroyed, want.AverageDestroyed)
			goldenVerifyDist(t, eps, "HitDist", got.HitDist, want.HitDist)
			goldenVerifyDist(t, eps, "WoundDist", got.WoundDist, want.WoundDist)
			goldenVerifyDist(t, eps, "PenDist", got.PenDist, want.PenDist)
			goldenVerifyDist(t, eps, "DamageDist", got.DamageDist, want.DamageDist)
			goldenVerifyDist(t, eps, "DestroyedDist", got.DestroyedDist, want.DestroyedDist)
		})
	}
}

// goldenVerifyDist is verifyDist with a caller-chosen epsilon instead of the
// fixed epsilonCore, since stress-scale cases need the relaxed tolerance.
func goldenVerifyDist(t *testing.T, eps float64, label string, got, want map[int]float64) {
	t.Helper()
	for k, wantP := range want {
		gotP, ok := got[k]
		if !ok {
			if wantP != 0 {
				t.Errorf("%s: missing outcome key %d in result distribution", label, k)
			}
			continue
		}
		if math.Abs(gotP-wantP) > eps {
			t.Errorf("%s key %d: golden %.9f, got %.9f (eps %.1e)", label, k, wantP, gotP, eps)
		}
	}
	if len(got) > len(want) {
		t.Errorf("%s: result distribution has extra outcomes (got %d keys, want %d)", label, len(got), len(want))
	}
}
