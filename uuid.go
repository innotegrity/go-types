package types

import (
	"encoding/hex"
	"fmt"
	"math/rand"
	"strings"

	"github.com/google/uuid"
)

// NewUUID generates a new UUID.
//
// This function first attempts to generate a v7 UUID.  If that fails, then a v8 UUID is generated instead.
func NewUUID() string {
	id, err := uuid.NewV7()
	if err != nil {
		return generateUUIDv8()
	}
	return strings.ToUpper(id.String())
}

// generateUUIDv8 generates a v8 UUID.
func generateUUIDv8() string {
	// generate 16 random bytes
	vals := make([]byte, 16)
	for i := 0; i < 16; i++ {
		vals[i] = byte(rand.Intn(255))
	}

	// replace bits 48-51 with the version (8)
	// we do this by shifting left 4, discarding the
	vals[6] = (((vals[6] << 4) & 255) >> 4) | 128

	// replace bits 64 and 65 with the variant (2)
	vals[8] = (((vals[8] << 2) & 255) >> 2) | 128

	// turn the whole thing into a string
	return strings.ToUpper(fmt.Sprintf("%s-%s-%s-%s-%s",
		hex.EncodeToString(vals[0:4]),
		hex.EncodeToString(vals[4:6]),
		hex.EncodeToString(vals[6:8]),
		hex.EncodeToString(vals[8:10]),
		hex.EncodeToString(vals[10:])))
}
