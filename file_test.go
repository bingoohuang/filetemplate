package filetemplate

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSubstitute(t *testing.T) {
	assert.Equal(t, "1", substitute("${PID}", map[string]string{"pid": "1"}))
	assert.Equal(t, "kill -s HUP 123", substitute("kill -s HUP ${PID}", map[string]string{"pid": "123"}))
}
