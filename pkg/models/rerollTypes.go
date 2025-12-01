package damagerequest

import (
	"encoding/json"
	"fmt"
)

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
