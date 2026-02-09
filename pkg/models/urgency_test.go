package models

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"xorm.io/xorm/schemas"
)

type stringBoundsQuoter struct {
	boundingString string
}

func (q stringBoundsQuoter) Quote(s string) string {
	return q.boundingString + s + q.boundingString
}

func TestUrgencyScoreQuery(t *testing.T) {
	t.Parallel()
	const testQuoteBounds = "@"
	for _, tc := range []struct {
		dbType      schemas.DBType
		expectQuery string
		expectErr   string
	}{
		{
			dbType: schemas.POSTGRES,
			expectQuery: `
(
	COALESCE(least(1, (1 << (greatest(0, (- cast(extract(epoch from @tasks.due_date@ - localtimestamp) as int)/86400 - -8) / 2))) / 38.05), 0) * 0.476 +
	COALESCE(@tasks.percent_done@, 0) * 0.476 +
	COALESCE(@tasks.priority@ / 5.0, 0) * 0.048
) as urgency
			`,
		},
		{
			dbType: schemas.MYSQL,
			expectQuery: `
(
	COALESCE(least(1, (1 << (greatest(0, (- cast(extract(epoch from @tasks.due_date@ - localtimestamp) as int)/86400 - -8) / 2))) / 38.05), 0) * 0.476 +
	COALESCE(@tasks.percent_done@, 0) * 0.476 +
	COALESCE(@tasks.priority@ / 5.0, 0) * 0.048
) as urgency
			`,
		},
		{
			dbType: schemas.SQLITE,
			expectQuery: `
(
	COALESCE(min(1, (1 << (max(0, (- cast(unixepoch(@tasks.due_date@) - unixepoch('now') as int)/86400 - -8) / 2))) / 38.05), 0) * 0.476 +
	COALESCE(@tasks.percent_done@, 0) * 0.476 +
	COALESCE(@tasks.priority@ / 5.0, 0) * 0.048
) as urgency
			`,
		},
		{
			dbType:    "not a real DB",
			expectErr: "unsupported database type: not a real DB",
		},
	} {
		t.Run(string(tc.dbType), func(t *testing.T) {
			t.Parallel()
			query, err := urgencyScoreQuery(defaultWeights(), stringBoundsQuoter{boundingString: testQuoteBounds}, tc.dbType)
			if tc.expectErr != "" {
				assert.EqualError(t, err, tc.expectErr)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, strings.TrimSpace(tc.expectQuery), strings.TrimSpace(query))
		})
	}
}
