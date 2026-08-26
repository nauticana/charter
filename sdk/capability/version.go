package capability

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

var ErrIncompatibleVersion = errors.New("capability contract version is incompatible")

// VersionPolicy decides whether a caller designed against one contract version may use another (CHR-CAP-008).
type VersionPolicy interface {
	Compatible(required, offered string) error
}

// BaseVersionPolicy accepts an identical version, or the same major of a dotted numeric version when the offered minor
// is not older than the required one; an empty requirement accepts any version.
type BaseVersionPolicy struct{}

var _ VersionPolicy = BaseVersionPolicy{}

func (BaseVersionPolicy) Compatible(required, offered string) error {
	if required == "" || required == offered {
		return nil
	}
	rMajor, rMinor, rOK := parseVersion(required)
	oMajor, oMinor, oOK := parseVersion(offered)
	if !rOK || !oOK || rMajor != oMajor || rMinor > oMinor {
		return fmt.Errorf("%w: required %s, offered %s", ErrIncompatibleVersion, required, offered)
	}
	return nil
}

func parseVersion(v string) (major, minor int, ok bool) {
	parts := strings.Split(v, ".")
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, false
	}
	if len(parts) > 1 {
		if minor, err = strconv.Atoi(parts[1]); err != nil {
			return 0, 0, false
		}
	}
	return major, minor, true
}
