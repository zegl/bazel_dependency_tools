package maven_jar

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewestAvailable(t *testing.T) {
	newest, err := NewestAvailable("com.google.zxing:core:3.3.3")
	assert.Nil(t, err)
	assert.Equal(t, "3.4.0", newest)
}

