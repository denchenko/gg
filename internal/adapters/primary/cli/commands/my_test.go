package commands

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseActivityDates(t *testing.T) {
	fixedTime := time.Date(2025, 12, 29, 10, 0, 0, 0, time.UTC)

	tests := []struct {
		name        string
		afterStr    string
		beforeStr   string
		now         time.Time
		expectError bool
		expected    ActivityDateRange
		errorMsg    string
	}{
		{
			name:      "both empty - defaults to working day logic (Monday)",
			afterStr:  "",
			beforeStr: "",
			now:       fixedTime, // Monday 2025-12-29
			expected: ActivityDateRange{
				After:  time.Date(2025, 12, 25, 0, 0, 0, 0, time.UTC), // Thursday (day before Friday)
				Before: nil,
			},
		},
		{
			name:      "both empty - Tuesday",
			afterStr:  "",
			beforeStr: "",
			now:       time.Date(2025, 12, 30, 10, 0, 0, 0, time.UTC), // Tuesday 2025-12-30
			expected: ActivityDateRange{
				After:  time.Date(2025, 12, 28, 0, 0, 0, 0, time.UTC), // Sunday (day before Monday)
				Before: nil,
			},
		},
		{
			name:      "both empty - Friday",
			afterStr:  "",
			beforeStr: "",
			now:       time.Date(2025, 12, 26, 10, 0, 0, 0, time.UTC), // Friday 2025-12-26
			expected: ActivityDateRange{
				// Wednesday (day before Thursday, which is previous working day)
				After:  time.Date(2025, 12, 24, 0, 0, 0, 0, time.UTC),
				Before: nil,
			},
		},
		{
			name:      "both empty - Saturday",
			afterStr:  "",
			beforeStr: "",
			now:       time.Date(2025, 12, 27, 10, 0, 0, 0, time.UTC), // Saturday 2025-12-27
			expected: ActivityDateRange{
				After:  time.Date(2025, 12, 25, 0, 0, 0, 0, time.UTC), // Thursday (day before Friday)
				Before: nil,
			},
		},
		{
			name:      "both empty - Sunday",
			afterStr:  "",
			beforeStr: "",
			now:       time.Date(2025, 12, 28, 10, 0, 0, 0, time.UTC), // Sunday 2025-12-28
			expected: ActivityDateRange{
				After:  time.Date(2025, 12, 25, 0, 0, 0, 0, time.UTC), // Thursday (day before Friday)
				Before: nil,
			},
		},
		{
			name:      "after provided, before empty",
			afterStr:  "2025-12-20",
			beforeStr: "",
			now:       fixedTime,
			expected: ActivityDateRange{
				After:  time.Date(2025, 12, 20, 0, 0, 0, 0, time.UTC),
				Before: nil,
			},
		},
		{
			name:      "after empty, before provided",
			afterStr:  "",
			beforeStr: "2025-12-31",
			now:       fixedTime,
			expected: ActivityDateRange{
				After:  time.Date(2025, 12, 25, 0, 0, 0, 0, time.UTC), // Default from Monday
				Before: timePtr(time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)),
			},
		},
		{
			name:      "both provided - valid range",
			afterStr:  "2025-12-20",
			beforeStr: "2025-12-25",
			now:       fixedTime,
			expected: ActivityDateRange{
				After:  time.Date(2025, 12, 20, 0, 0, 0, 0, time.UTC),
				Before: timePtr(time.Date(2025, 12, 25, 0, 0, 0, 0, time.UTC)),
			},
		},
		{
			name:        "both provided - invalid range (after after before)",
			afterStr:    "2025-12-25",
			beforeStr:   "2025-12-20",
			now:         fixedTime,
			expectError: true,
			errorMsg:    "--after date must be before --before date",
		},
		{
			name:        "invalid after format",
			afterStr:    "12/20/2025",
			beforeStr:   "",
			now:         fixedTime,
			expectError: true,
			errorMsg:    "invalid --after date",
		},
		{
			name:        "invalid before format",
			afterStr:    "",
			beforeStr:   "12/20/2025",
			now:         fixedTime,
			expectError: true,
			errorMsg:    "invalid --before date",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseActivityDates(tt.afterStr, tt.beforeStr, tt.now)

			if tt.expectError {
				require.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected.After.UTC(), result.After.UTC(), "After time mismatch")
				if tt.expected.Before == nil {
					assert.Nil(t, result.Before, "Before should be nil")
				} else {
					require.NotNil(t, result.Before, "Before should not be nil")
					assert.Equal(t, tt.expected.Before.UTC(), result.Before.UTC(), "Before time mismatch")
				}
			}
		})
	}
}

func timePtr(t time.Time) *time.Time {
	return &t
}
