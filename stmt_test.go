package prestgo

import (
	"database/sql/driver"
	"testing"
	"time"
)

func TestGetParameterOffset(t *testing.T) {
	var tests = []struct {
		name          string
		desc          string
		query         string
		expectedCount int
	}{
		{
			name:          "a",
			desc:          "1 query param at the end",
			query:         "SELECT * FROM abc WHERE blah = ?",
			expectedCount: 1,
		},
		{
			name:          "b",
			desc:          "6 query param at the end",
			query:         "SELECT * FROM abc WHERE blah = ? and foo = ? and bar = ? or (hello = ? and world= ?) or do = ?",
			expectedCount: 6,
		},
		{
			name: "c",
			desc: "1 query param with lots of suffixing clauses",
			query: `SELECT   abc,
         def,
         ghi,
         jkl,
         mno,
         pqr,
         sto,
         uvw,
         xyz,
         abc_def,
         ghi_ppp,
         Max(timestamp)
FROM     hive.infra.fp_system
WHERE    host = ?
GROUP BY (abc, def, ghi, jkl, mno, pqr, sto, uvw, xyz, abc_def, ghi_ppp)`,
			expectedCount: 1,
		},
		{
			name:          "d",
			desc:          "0 query params",
			query:         "SELECT * FROM abc",
			expectedCount: 0,
		},
	}

	for i, test := range tests {
		t.Logf("test %d: %q", i, test.desc)

		t.Run(test.name, func(t *testing.T) {
			offsets := getParameterOffsets(test.query)

			if len(offsets) != test.expectedCount {
				t.Fatalf(
					"getParameterOffsets(%q) -> expected count of %d, got %d",
					test.query,
					test.expectedCount,
					len(offsets),
				)
			}

			for _, offset := range offsets {
				if test.query[offset] != '?' {
					t.Errorf(
						"getParameterOffsets(%q) -> expected offset %d to be '?' but got %q",
						test.query,
						offset,
						test.query[offset],
					)
				}
			}
		})
	}
}

func TestQueryInterpolate(t *testing.T) {
	var happPathTests = []struct {
		name           string
		desc           string
		inputQuery     string
		queryArgs      []driver.Value
		expectedOutput string
	}{
		{
			name:           "a",
			desc:           "1 query param of type string",
			inputQuery:     "SELECT * FROM abc WHERE blah = ?",
			queryArgs:      []driver.Value{"def"},
			expectedOutput: "SELECT * FROM abc WHERE blah = 'def'",
		},
		{
			name:           "b",
			desc:           "1 query param of type int64",
			inputQuery:     "SELECT * FROM abc WHERE blah = ?",
			queryArgs:      []driver.Value{int64(1)},
			expectedOutput: "SELECT * FROM abc WHERE blah = 1",
		},
		{
			name:           "c",
			desc:           "1 query param of type bool true",
			inputQuery:     "SELECT * FROM abc WHERE blah = ?",
			queryArgs:      []driver.Value{true},
			expectedOutput: "SELECT * FROM abc WHERE blah = 1",
		},
		{
			name:           "d",
			desc:           "1 query param of type bool false",
			inputQuery:     "SELECT * FROM abc WHERE blah = ?",
			queryArgs:      []driver.Value{false},
			expectedOutput: "SELECT * FROM abc WHERE blah = 0",
		},
		{
			name:           "e",
			desc:           "1 query param of type float64",
			inputQuery:     "SELECT * FROM abc WHERE blah = ?",
			queryArgs:      []driver.Value{float64(1.01)},
			expectedOutput: "SELECT * FROM abc WHERE blah = 1.01",
		},
		{
			name:           "f",
			desc:           "1 query param of type []byte",
			inputQuery:     "SELECT * FROM abc WHERE blah = ?",
			queryArgs:      []driver.Value{[]byte("abcdef")},
			expectedOutput: "SELECT * FROM abc WHERE blah = _binary'abcdef'",
		},
		{
			name:           "g",
			desc:           "1 query param of type nil",
			inputQuery:     "SELECT * FROM abc WHERE blah = ?",
			queryArgs:      []driver.Value{nil},
			expectedOutput: "SELECT * FROM abc WHERE blah = NULL",
		},
		{
			name:           "h",
			desc:           "1 query param of type time.Time",
			inputQuery:     "SELECT * FROM abc WHERE blah = ?",
			queryArgs:      []driver.Value{time.Date(1990, time.May, 1, 1, 1, 1, 1, time.UTC)},
			expectedOutput: "SELECT * FROM abc WHERE blah = '1990-05-01 01:01:01'",
		},
		{
			name:           "i",
			desc:           "3 query param of type string, int64, and time.Time",
			inputQuery:     "SELECT * FROM abc WHERE foo = ? and bar = ? and blah = ?",
			queryArgs:      []driver.Value{"abc", int64(5), time.Date(1990, time.May, 1, 1, 1, 1, 1, time.UTC)},
			expectedOutput: "SELECT * FROM abc WHERE foo = 'abc' and bar = 5 and blah = '1990-05-01 01:01:01'",
		},
		{
			name:           "j",
			desc:           "0 query params",
			inputQuery:     "SELECT * FROM abc WHERE blah",
			queryArgs:      []driver.Value{},
			expectedOutput: "SELECT * FROM abc WHERE blah",
		},
		{
			name:           "k",
			desc:           "1 query param of type string",
			inputQuery:     "SELECT * FROM abc WHERE blah = ?",
			queryArgs:      []driver.Value{"def\r\n'"},
			expectedOutput: "SELECT * FROM abc WHERE blah = 'def\\r\\n\\''",
		},
	}

	for i, test := range happPathTests {
		t.Logf("happy path test %d: %q", i, test.desc)

		t.Run(test.name, func(t *testing.T) {
			s := &stmt{
				query:        test.inputQuery,
				paramOffsets: getParameterOffsets(test.inputQuery),
			}

			got, err := s.queryInterpolate(test.queryArgs)
			if err != nil {
				t.Fatalf(
					"stmt.queryInterpolate(%v) -> error: %v; expected nil",
					test.queryArgs,
					err,
				)
			}

			if got != test.expectedOutput {
				t.Errorf(
					"stmt.queryInterpolate(%v) -> %q; expected %q",
					test.queryArgs,
					got,
					test.expectedOutput,
				)
			}
		})
	}

	var negativeTests = []struct {
		name       string
		desc       string
		inputQuery string
		queryArgs  []driver.Value
	}{
		{
			name:       "a",
			desc:       "arg count is less than offset count",
			inputQuery: "SELECT * FROM abc WHERE blah = ?",
			queryArgs:  []driver.Value{},
		},
		{
			name:       "b",
			desc:       "arg count is greater than offset count",
			inputQuery: "SELECT * FROM abc WHERE blah = ?",
			queryArgs:  []driver.Value{"def", "efg"},
		},
		{
			name:       "c",
			desc:       "invalid arg type",
			inputQuery: "SELECT * FROM abc WHERE blah = ?",
			queryArgs:  []driver.Value{8},
		},
	}

	for i, test := range negativeTests {
		t.Logf("negative path test %d: %q", i, test.desc)

		t.Run(test.name, func(t *testing.T) {
			s := &stmt{
				query:        test.inputQuery,
				paramOffsets: getParameterOffsets(test.inputQuery),
			}

			if _, err := s.queryInterpolate(test.queryArgs); err == nil {
				t.Errorf(
					"stmt.queryInterpolate(%v) -> nil error; expected and error",
					test.queryArgs,
				)
			}
		})
	}
}
