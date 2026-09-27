package suggest

import (
	"crypto/sha256"
	"encoding/hex"
)

// StableID computes a Suggestion.ID as a stable hash of kind+file+anchor,
// per the spec's core types ("ID: stable hash of kind+file+anchor").
// Stability matters because `integrate --suggestion <id>` needs the same
// suggestion to resolve to the same ID across runs on unchanged code.
func StableID(kind, file, anchor string) string {
	sum := sha256.Sum256([]byte(kind + "\x00" + file + "\x00" + anchor))
	return hex.EncodeToString(sum[:])[:12]
}
