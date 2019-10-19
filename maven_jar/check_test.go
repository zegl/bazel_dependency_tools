package maven_jar

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewestAvailable(t *testing.T) {
	newest, sha1, err := NewestAvailable("com.google.zxing:core:3.3.0")
	assert.Nil(t, err)
	assert.Equal(t, "3.4.0", newest)
	assert.Equal(t, "5264296c46634347890ec9250bc65f14b7362bf8", sha1)
}

func TestNewestAvailableOddJarSha1(t *testing.T) {
	newest, sha1, err := NewestAvailable("mx4j:mx4j-tools:3.0.1")
	assert.Nil(t, err)
	assert.Equal(t, "3.0.1", newest)
	assert.Equal(t, "df853af9fe34d4eb6f849a1b5936fddfcbe67751", sha1)
}

func TestNewestAvailableOroOro(t *testing.T) {
	newest, sha1, err := NewestAvailable("oro:oro:2.0.6")
	assert.Nil(t, err)
	assert.Equal(t, "2.0.8", newest)
	assert.Equal(t, "5592374f834645c4ae250f4c9fbb314c9369d698", sha1)
}
