package security

// Sanitizer is the injectable security contract for untrusted input.
type Sanitizer interface {
	SanitizeText(field, value string, maxBytes int) (string, error)
	SanitizeMessage(value string) (string, error)
	RejectMalformedID(value string) error
	RejectInvalidUTF8(field, value string) error
	PreventPathTraversal(field, path string) (string, error)
	PreventConfigInjection(key, value string) error
	PreventSQLInjection(field, value string) error
	PreventLMDBKeyCorruption(key []byte) error
	NormalizePhone(value string) (string, error)
	NormalizeUsername(value string) (string, error)
}

const (
	// DefaultMaxMessageBytes rejects oversized user messages.
	DefaultMaxMessageBytes = 4096
	// MaxLMDBKeyBytes is the practical LMDB key size ceiling used for rejection.
	MaxLMDBKeyBytes = 511
	// MaxPathBytes rejects absurdly long path inputs.
	MaxPathBytes = 4096
)

// Standard is the default dependency-injected sanitizer.
type Standard struct {
	MaxMessageBytes int
	// AllowedRoots, when non-empty, requires cleaned paths to stay under one root.
	AllowedRoots []string
}

// New constructs a Standard sanitizer with DefaultMaxMessageBytes.
func New() *Standard {
	return &Standard{MaxMessageBytes: DefaultMaxMessageBytes}
}

// NewWithRoots constructs a sanitizer that confines paths under the given roots.
func NewWithRoots(roots ...string) *Standard {
	s := New()
	if len(roots) > 0 {
		s.AllowedRoots = append([]string(nil), roots...)
	}
	return s
}

var _ Sanitizer = (*Standard)(nil)
