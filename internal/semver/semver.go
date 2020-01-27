package semver

import (
	"github.com/blang/semver"
	"strings"
)

func NormalizeNew(v string) (*semver.Version, error) {
	v = strings.TrimLeft(v, "v")
	if strings.Count(v, ".") == 1 {
		v = v + ".0"
	}
	return semver.New(v)
}
