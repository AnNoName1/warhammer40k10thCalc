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
