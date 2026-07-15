// Package generateid provides a shared ID generator for provider responses.
package generateid

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// New returns a unique ID with the given prefix.
func New(prefix string) string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%s-%d-%s", prefix, time.Now().UnixNano(), hex.EncodeToString(b))
}
