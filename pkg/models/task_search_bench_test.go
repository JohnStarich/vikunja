// Vikunja is a to-do list application to facilitate your life.
// Copyright 2018-present Vikunja and contributors. All rights reserved.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package models

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"os"
	"strings"
	"testing"
	"time"

	"code.vikunja.io/api/pkg/config"
	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/user"

	"github.com/jaswdr/faker/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func initBenchmarkConfig() {
	if os.Getenv("VIKUNJA_TESTS_USE_CONFIG") == "1" {
		config.InitConfig()
	} else {
		config.InitDefaultConfig()
		config.ServiceRootpath.Set(os.Getenv("VIKUNJA_SERVICE_ROOTPATH"))
	}
}

// createBenchmarkData creates projects and tasks used for search benchmarks.
func createBenchmarkData(b *testing.B, needle string, numberOfProjects, numberOfTasks uint) *user.User {
	s := db.NewSession()
	defer s.Close()

	f := faker.New()

	u, err := user.GetUserByID(s, 1)
	if err != nil {
		b.Fatalf("get user: %v", err)
	}

	now := time.Now()
	for i := range numberOfProjects {
		p := &Project{Title: fmt.Sprintf("Project %d", i), OwnerID: u.ID}
		if _, err := s.Insert(p); err != nil {
			b.Fatalf("insert project: %v", err)
		}

		for j := range numberOfTasks {
			title := f.Lorem().Sentence(6)
			if randInt(100) == 0 {
				title += " " + needle
			}
			desc := ""
			if j%2 == 0 {
				desc = f.Lorem().Paragraph(1)
			}
			if j%100 == 0 {
				if desc == "" {
					desc = f.Lorem().Paragraph(1)
				}
				words := strings.Split(desc, " ")
				mid := len(words) / 2
				words = append(words[:mid], append([]string{needle}, words[mid:]...)...)
				desc = strings.Join(words, " ")
			}
			t := &Task{
				Title:       title,
				Description: desc,
				ProjectID:   p.ID,
				CreatedByID: u.ID,
				Index:       int64(j + 1),
				DueDate:     now.Add(dueDateJitter()),
				Priority:    randInt(5),
			}
			if _, err := s.Insert(t); err != nil {
				b.Fatalf("insert task: %v", err)
			}
		}
	}

	return u
}

func randInt(maxValue int64) int64 {
	n, err := rand.Int(rand.Reader, big.NewInt(maxValue))
	if err != nil {
		panic(err)
	}
	return n.Int64()
}

func dueDateJitter() time.Duration {
	const (
		day     = 24 * time.Hour
		minDate = -14 * day
		maxDate = 7 * day
		width   = maxDate - minDate
	)
	jitter := time.Duration(randInt(width.Nanoseconds()))
	return minDate + jitter
}

func BenchmarkTaskSearch(b *testing.B) {
	const needle = "Lorem Ipsum"

	initBenchmarkConfig()
	SetupTests()
	err := db.LoadFixtures()
	if err != nil {
		b.Fatalf("load fixtures: %v", err)
	}

	for _, tc := range []struct {
		description   string
		pickProject   bool
		opts          taskSearchOptions
		numberOfTasks []uint
	}{
		{
			description: "search",
			opts: taskSearchOptions{
				search:             needle,
				page:               1,
				perPage:            50,
				filter:             "done = false",
				filterTimezone:     "UTC",
				filterIncludeNulls: false,
			},
			numberOfTasks: []uint{2500},
		},
		{
			description: "sort by urgency",
			pickProject: true,
			opts: taskSearchOptions{
				page:    1,
				perPage: 50,
				sortby: []*sortParam{
					{
						sortBy:  taskVirtualPropertyUrgency,
						orderBy: orderDescending,
					},
				},
			},
			numberOfTasks: []uint{100, 1000, 10000, 100000},
		},
	} {
		b.Run(fmt.Sprintf("%s %s", config.DatabaseType.GetString(), tc.description), func(b *testing.B) {
			for _, numberOfTasks := range tc.numberOfTasks {
				b.Run(fmt.Sprintf("%d", numberOfTasks), func(b *testing.B) {
					auth := createBenchmarkData(b, needle, 10, numberOfTasks)

					// Get all projects for the user
					s := db.NewSession()
					projects, _, _, err := getRawProjectsForUser(
						s,
						&projectOptions{
							user: auth,
							page: -1,
						},
					)
					require.NoError(b, s.Close())
					require.NoError(b, err)
					if tc.pickProject {
						projects = projects[len(projects)-2:]
					}

					for b.Loop() {
						s := db.NewSession()
						resultSlice, _, _, err := getRawTasksForProjects(s, projects, auth, &tc.opts)
						assert.NoError(b, s.Close())

						assert.NoError(b, err)
						require.NotEmpty(b, resultSlice)
					}
				})
			}
		})
	}
}
