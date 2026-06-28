package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"xorm.io/xorm/schemas"
)

func TestUrgencyProperty_NormalizedPropertyScore(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		description string
		property    UrgencyProperty
		filter      *TaskCollection
		dbType      schemas.DBType
		expectQuery string
		expectErr   string
	}{
		{
			description: "invalid property",
			property:    0,
			expectErr:   "unrecognized urgency score property: <err: invalid urgency property enum value: 0>",
		},
		{
			description: "due date",
			property:    UrgencyDueDate,
			dbType:      schemas.SQLITE,
			// Full coverage in [TestUrgencyScoreQuery]
			expectQuery: `min(1, (1 << (max(0, (- cast(unixepoch("tasks.due_date") - unixepoch('now') as int)/86400 - -8) / 2))) / 38.05)`,
		},
		{
			description: "matches filter",
			property:    UrgencyMatchesFilter,
			filter:      &TaskCollection{Filter: `done = false`},
			expectQuery: "CASE WHEN (tasks.`done`=false) THEN 1 ELSE 0 END",
		},
		{
			description: "invalid filter string",
			property:    UrgencyMatchesFilter,
			filter:      &TaskCollection{Filter: "very broken"},
			expectErr:   `could not parse filter string "very broken": Task filter expression 'very broken' is invalid [ExpressionError: expected a sign operator, got "broken" (identifier)]`,
		},
		{
			description: "percent done",
			property:    UrgencyPercentDone,
			expectQuery: `"tasks.percent_done"`,
		},
		{
			description: "priority",
			property:    UrgencyPriority,
			expectQuery: `"tasks.priority" / 5.0`,
		},
	} {
		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()
			quoter := stringBoundsQuoter{boundingString: `"`}
			query, err := tc.property.normalizedPropertyScore(tc.filter, quoter, tc.dbType)
			if tc.expectErr != "" {
				assert.EqualError(t, err, tc.expectErr)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tc.expectQuery, query)
		})
	}
}
