package panels

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContainsCaseInsensitive(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		query    string
		expected bool
	}{
		{
			name:     "uppercase query matches lowercase value",
			value:    "apischeduler-sched-6460",
			query:    "APISCHEDULER",
			expected: true,
		},
		{
			name:     "lowercase query matches uppercase value",
			value:    "TICKETS-DASHBOARD",
			query:    "tickets",
			expected: true,
		},
		{
			name:     "unicode query matches with different case",
			value:    "Réseau",
			query:    "RÉSEAU",
			expected: true,
		},
		{
			name:     "unrelated query does not match",
			value:    "accloud-lde_default",
			query:    "scheduler",
			expected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expected, containsCaseInsensitive(test.value, test.query))
		})
	}
}
