package damagerequest

type DamageRequest struct {
	NumModels     int    `json:"num_models"`
	AttacksString string `json:"attacks_string"` // e.g., "D6+2"
	BS            int    `json:"bs"`             // Ballistic Skill
	S             int    `json:"s"`              // Strength
	AP            int    `json:"ap"`             // Armor penetration
	D             string `json:"d"`              // Damage, e.g., "D3", "2"
	T             int    `json:"t"`              // Target Toughness
	Save          int    `json:"save"`           // Target Save
	// Pointers (*int) are used for optional fields. If the field is omitted in JSON, the pointer will be nil.
	Invulnerable *int `json:"invulnerable,omitempty"` // Invulnerable Save, optional
	FeelNoPain   *int `json:"feel_no_pain,omitempty"` // Feel No Pain, optional

	HitReroll   RerollType `json:"hit_reroll"`
	WoundReroll RerollType `json:"wound_reroll"`

	HitModifier   int `json:"hit_modifier,omitempty"`
	WoundModifier int `json:"wound_modifier,omitempty"`
	SaveModifier  int `json:"save_modifier,omitempty"`

	LethalHits        bool `json:"lethal_hits,omitempty"`
	DevastatingWounds bool `json:"devastating_wounds,omitempty"`
	Torrent           bool `json:"torrent,omitempty"`
}

// DamageResponse defines the structure of the JSON response sent back to the client.
type DamageResponse struct {
	AverageHits           float64         `json:"average_hits"`
	AverageDestroyed      float64         `json:"average_destroyed"`
	HitsDistribution      map[int]float64 `json:"hits_distribution"`
	PensDistribution      map[int]float64 `json:"pens_distribution"`
	WoundsDistribution    map[int]float64 `json:"wounds_distribution"`
	DestroyedDistribution map[int]float64 `json:"destroyed_distribution"`
	Message               string          `json:"message"`
}
