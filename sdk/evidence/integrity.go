package evidence

import (
	"crypto/sha256"
	"encoding/hex"
)

// Digester computes record digests and the chained bundle value of one integrity method (CHR-EVID-005).
type Digester interface {
	Method() string
	Digest(raw []byte) string
	Chain(digests []string) string
}

// BaseSHA256Digester implements the sha256-hash-chain method with sha256:<hex> values.
type BaseSHA256Digester struct{}

var _ Digester = BaseSHA256Digester{}

func (BaseSHA256Digester) Method() string { return "sha256-hash-chain" }

func (BaseSHA256Digester) Digest(raw []byte) string {
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func (BaseSHA256Digester) Chain(digests []string) string {
	chain := ""
	for _, d := range digests {
		sum := sha256.Sum256([]byte(chain + d))
		chain = hex.EncodeToString(sum[:])
	}
	return "sha256:" + chain
}
