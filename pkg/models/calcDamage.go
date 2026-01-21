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

package damagerequest

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	calculator "github.com/AnNoName1/warhammer40k10thCalc/internal/calculator"
)

// DamageRequestDTO is the structured JSON body
type DamageRequestDTO struct {
	Attacker AttackerDTO `json:"attacker"`
	Target   TargetDTO   `json:"target"`
	Rules    RulesDTO    `json:"rules"`
}

// AttackerDTO includes weapon keywords and roll modifiers.
type AttackerDTO struct {
	NumModels     int    `json:"num_models"`
	AttacksString string `json:"attacks_string"`
	BS            int    `json:"bs"`
	S             int    `json:"s"`
	AP            int    `json:"ap"`
	D             string `json:"d"`
	// Weapon Keywords
	Blast             bool `json:"blast,omitempty"`
	SustainedHits     int  `json:"sustained_hits,omitempty"`
	LethalHits        bool `json:"lethal_hits,omitempty"`
	DevastatingWounds bool `json:"devastating_wounds,omitempty"`
	Torrent           bool `json:"torrent,omitempty"`
	// Modifiers (e.g., -1 to hit is -1)
	HitModifier   int `json:"hit_modifier,omitempty"`
	WoundModifier int `json:"wound_modifier,omitempty"`
}

// TargetDTO includes defensive layers and resilience rules.
type TargetDTO struct {
	T              int  `json:"t"`
	Save           int  `json:"save"`
	WoundsPerModel int  `json:"wounds_per_model"`
	ModelCount     *int `json:"model_count"`
	Cover          bool `json:"cover"`
	Invulnerable   *int `json:"invulnerable,omitempty"`
	FeelNoPain     *int `json:"feel_no_pain,omitempty"` // FNP (e.g., 6)
}

// RulesDTO handles rerolls, global modifiers, and critical thresholds.
type RulesDTO struct {
	HitReroll    calculator.RerollType `json:"hit_reroll,omitempty"`
	WoundReroll  calculator.RerollType `json:"wound_reroll,omitempty"`
	SaveReroll   calculator.RerollType `json:"save_reroll,omitempty"`
	SaveModifier int                   `json:"save_modifier,omitempty"`
	// Thresholds allow for rules like "Critical hits on a 5+"
	CriticalHitThreshold   int `json:"critical_hit_threshold,omitempty"`
	CriticalWoundThreshold int `json:"critical_wound_threshold,omitempty"`
}

var diceRegex = regexp.MustCompile(`(?i)^(\d*)d(\d+)\s*([+-]\s*\d+)?$`)

// ParseDiceString converts "2d6+1" or "4" into a clean DiceRoll struct
func ParseDiceString(input string) (calculator.DiceRoll, error) {
	input = strings.TrimSpace(input)

	// 1. Handle Fixed Number (e.g. "4")
	if val, err := strconv.Atoi(input); err == nil {
		return calculator.DiceRoll{Count: 0, Sides: 0, Modifier: val}, nil
	}

	// 2. Handle Dice String (e.g. "2d6+1")
	matches := diceRegex.FindStringSubmatch(input)
	if matches == nil {
		return calculator.DiceRoll{}, fmt.Errorf("invalid format '%s'", input)
	}

	// Parse Count (default to 1 if empty, e.g. "d6")
	count := 1
	if matches[1] != "" {
		var err error
		count, err = strconv.Atoi(matches[1])
		if err != nil {
			return calculator.DiceRoll{}, fmt.Errorf("invalid dice count: %w", err)
		}
	}

	// Parse Sides
	sides, err := strconv.Atoi(matches[2])
	if err != nil || sides <= 0 {
		return calculator.DiceRoll{}, fmt.Errorf("invalid die type")
	}

	// Parse Modifier
	mod := 0
	if matches[3] != "" {
		// Remove whitespace inside modifier string (e.g. "+ 1" -> "+1")
		cleanMod := strings.ReplaceAll(matches[3], " ", "")
		mod, err = strconv.Atoi(cleanMod)
		if err != nil {
			return calculator.DiceRoll{}, fmt.Errorf("invalid modifier: %w", err)
		}
	}

	return calculator.DiceRoll{
		Count:    count,
		Sides:    sides,
		Modifier: mod,
	}, nil
}

// --- Converter (The Mapper) ---
func (req *DamageRequestDTO) ToDomain() (calculator.CombatSimulationRequest, error) {
	// Logic for defaulting thresholds to 6 if not provided
	critHit := req.Rules.CriticalHitThreshold
	if critHit == 0 {
		critHit = 6
	}

	critWound := req.Rules.CriticalWoundThreshold
	if critWound == 0 {
		critWound = 6
	}

	// 1. Parse Attacks
	attacks, err := ParseDiceString(req.Attacker.AttacksString)
	if err != nil {
		return calculator.CombatSimulationRequest{}, fmt.Errorf("attacker attacks: %w", err)
	}

	// 2. Parse Damage
	damage, err := ParseDiceString(req.Attacker.D)
	if err != nil {
		return calculator.CombatSimulationRequest{}, fmt.Errorf("attacker damage: %w", err)
	}

	model := calculator.CombatSimulationRequest{
		Attacker: calculator.AttackerProfile{
			Count:             req.Attacker.NumModels,
			Attacks:           attacks,
			BS:                req.Attacker.BS,
			Strength:          req.Attacker.S,
			AP:                req.Attacker.AP,
			Damage:            damage,
			SustainedHits:     req.Attacker.SustainedHits,
			Blast:             req.Attacker.Blast,
			LethalHits:        req.Attacker.LethalHits,
			DevastatingWounds: req.Attacker.DevastatingWounds,
			Torrent:           req.Attacker.Torrent,
		},
		Target: calculator.TargetProfile{
			Count:          req.Target.ModelCount,
			Toughness:      req.Target.T,
			Save:           req.Target.Save,
			Invulnerable:   req.Target.Invulnerable,
			WoundsPerModel: req.Target.WoundsPerModel,
			FeelNoPain:     req.Target.FeelNoPain,
			HasCover:       req.Target.Cover,
		},
		Settings: calculator.SimulationSettings{
			HitReroll:              req.Rules.HitReroll,
			WoundReroll:            req.Rules.WoundReroll,
			SaveReroll:             req.Rules.SaveReroll,
			SaveModifier:           req.Rules.SaveModifier,
			CriticalHitThreshold:   critHit,
			CriticalWoundThreshold: critWound,

			HitModifier:   req.Attacker.HitModifier,
			WoundModifier: req.Attacker.WoundModifier,
		},
	}

	return model, nil
}

//now for response

type DamageResponseDTO struct {
	// Grouped or flattened as you prefer, but separate from the core
	Summary       SummaryDTO       `json:"summary"`
	Distributions DistributionsDTO `json:"distributions"`

	Message     string `json:"message"`
	RequestUUID string `json:"request_uuid,omitempty"`
}

type SummaryDTO struct {
	AverageHits      float64 `json:"average_hits"`
	AverageDestroyed float64 `json:"average_destroyed"`
}

type DistributionsDTO struct {
	Hits      map[int]float64 `json:"hits"`
	Wounds    map[int]float64 `json:"wounds"`
	Saves     map[int]float64 `json:"saves_failed"`
	Destroyed map[int]float64 `json:"models_destroyed"`
}

func MapResultToResponse(res calculator.SimulationResult, uuid string) DamageResponseDTO {
	return DamageResponseDTO{
		Summary: SummaryDTO{
			AverageHits:      res.AverageHits,
			AverageDestroyed: res.AverageDestroyed,
		},
		Distributions: DistributionsDTO{
			Hits:      res.HitDist,
			Wounds:    res.WoundDist,
			Saves:     res.PenDist,
			Destroyed: res.DestroyedDist,
		},
		Message:     "Calculation successful",
		RequestUUID: uuid,
	}
}
