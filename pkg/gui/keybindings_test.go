package gui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBindingGetKeyUsesDisplayKey(t *testing.T) {
	binding := &Binding{Key: 'g', DisplayKey: "gg"}

	assert.Equal(t, "gg", binding.GetKey())
}
