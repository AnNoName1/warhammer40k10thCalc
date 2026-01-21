package calculator

import (
	"encoding/json"
	"fmt"
)

// CombatSimulationRequest is the clean "Internal" struct
type CombatSimulationRequest struct {
	Attacker AttackerProfile
	Target   TargetProfile
	Settings SimulationSettings
}

type AttackerProfile struct {
	Count             int
	Attacks           DiceRoll
	BS                int
	Strength          int
	AP                int
	Damage            DiceRoll
	SustainedHits     int
	Blast             bool
	LethalHits        bool
	DevastatingWounds bool
	Torrent           bool
}

type TargetProfile struct {
	Count          *int
	Toughness      int
	Save           int
	Invulnerable   *int
	WoundsPerModel int
	FeelNoPain     *int

	HasCover bool
}

type SimulationSettings struct {
	HitReroll   RerollType
	WoundReroll RerollType
	SaveReroll  RerollType
	// Thresholds
	CriticalHitThreshold   int
	CriticalWoundThreshold int
	//modifiers
	SaveModifier  int
	HitModifier   int
	WoundModifier int
}

// dice string struct - 2d6 + 1
type DiceRoll struct {
	Count    int // Number of dice (e.g., 2)
	Sides    int // Die faces (e.g., 6). If 0, it's a fixed value.
	Modifier int // Flat bonus (e.g., +1)
}

// RerollType definitions
type RerollType int

const (
	// RerollNone is the default state (0).
	RerollNone RerollType = iota
	RerollOnes
	RerollFail
	// Add more types as needed
)

// String implements the fmt.Stringer interface to provide a readable string value.
func (r RerollType) String() string {
	// Map the integer value back to a descriptive string.
	if r < 0 || int(r) >= len(rerollTypeNames) {
		return fmt.Sprintf("UnknownRerollType(%d)", r)
	}
	return rerollTypeNames[r]
}

var rerollTypeNames = map[RerollType]string{
	RerollNone: "none",
	RerollOnes: "ones",
	RerollFail: "fail",
}

// MarshalJSON ensures that the RerollType is serialized as its string value (e.g., "ones") in the JSON response,
// instead of its underlying integer value (e.g., 1).
func (r RerollType) MarshalJSON() ([]byte, error) {
	return json.Marshal(r.String())
}

// UnmarshalJSON ensures that a string value (e.g., "ones") in the request body
// is correctly converted back into the RerollType integer constant.
func (r *RerollType) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	for k, v := range rerollTypeNames {
		if v == s {
			*r = k
			return nil
		}
	}
	return fmt.Errorf("unknown RerollType: %s", s)
}

// response
type SimulationResult struct {
	AverageHits      float64
	AverageDestroyed float64
	// Using specific names that make sense to the math engine
	HitDist       map[int]float64
	WoundDist     map[int]float64
	PenDist       map[int]float64 // Armor saves failed
	DestroyedDist map[int]float64
}
