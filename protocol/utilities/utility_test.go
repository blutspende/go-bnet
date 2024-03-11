package utilities

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestContains(t *testing.T) {
	assert.True(t, Contains("123.123.123.123", []string{"123.123.123.123", "123.123.123.124", "123.123.123.125"}))
	assert.True(t, Contains("123.123.123.124", []string{"123.123.123.123", "123.123.123.124", "123.123.123.125"}))
	assert.True(t, Contains("123.123.123.125", []string{"123.123.123.123", "123.123.123.124", "123.123.123.125"}))
	assert.False(t, Contains("123.123.123.126", []string{"123.123.123.123", "123.123.123.124", "123.123.123.125"}))
}
