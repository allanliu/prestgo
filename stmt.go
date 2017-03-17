package prestgo

import (
	"bytes"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type stmt struct {
	conn         *conn
	query        string
	paramOffsets []int
}

var _ driver.Stmt = &stmt{}

func (s *stmt) Close() error {
	return nil
}

func (s *stmt) NumInput() int {
	return -1 // TODO: parse query for parameters
}

func (s *stmt) Exec(args []driver.Value) (driver.Result, error) {
	return nil, ErrNotSupported
}

func (s *stmt) Query(args []driver.Value) (driver.Rows, error) {
	queryURL := fmt.Sprintf("http://%s/v1/statement", s.conn.addr)

	q, err := s.queryInterpolate(args)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", queryURL, strings.NewReader(q))
	if err != nil {
		return nil, err
	}
	req.Header.Add("X-Presto-User", s.conn.user)
	req.Header.Add("X-Presto-Catalog", s.conn.catalog)
	req.Header.Add("X-Presto-Schema", s.conn.schema)

	resp, err := s.conn.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Presto doesn't use the http response code, parse errors come back as 200
	if resp.StatusCode != 200 {
		return nil, ErrQueryFailed
	}

	var sresp stmtResponse
	err = json.NewDecoder(resp.Body).Decode(&sresp)
	if err != nil {
		return nil, err
	}

	if sresp.Stats.State == "FAILED" {
		return nil, sresp.Error
	}

	time.Sleep(500 * time.Millisecond)

	r := &rows{
		conn:    s.conn,
		nextURI: sresp.NextURI,
	}

	return r, nil
}

func (s *stmt) queryInterpolate(args []driver.Value) (string, error) {
	if len(args) != len(s.paramOffsets) {
		return "", driver.ErrSkip
	}

	if len(s.paramOffsets) < 1 {
		return s.query, nil
	}

	var (
		result = new(bytes.Buffer)
		i      = 0
	)

	for n, arg := range args {
		result.WriteString(s.query[i:s.paramOffsets[n]])

		switch v := arg.(type) {
		case string:
			result.WriteByte('\'')
			escapeStringBackslash(result, v)
			result.WriteByte('\'')

		case int64:
			result.WriteString(strconv.FormatInt(v, 10))

		case float64:
			result.WriteString(strconv.FormatFloat(v, 'g', -1, 64))

		case bool:
			if v {
				result.WriteByte('1')
			} else {
				result.WriteByte('0')
			}

		case nil:
			result.WriteString("NULL")

		case []byte:
			if v == nil {
				result.WriteString("NULL")

			} else {
				result.WriteString("_binary'")
				escapeBytesBackslash(result, v)
				result.WriteByte('\'')
			}

		case time.Time:
			if v.IsZero() {
				result.WriteString("'0000-00-00'")

			} else {
				v := v.In(time.UTC)
				v = v.Add(time.Nanosecond * 500) // To round under microsecond
				year := v.Year()
				year100 := year / 100
				year1 := year % 100
				month := v.Month()
				day := v.Day()
				hour := v.Hour()
				minute := v.Minute()
				second := v.Second()
				micro := v.Nanosecond() / 1000

				result.Write([]byte{
					'\'',
					digits10[year100], digits01[year100],
					digits10[year1], digits01[year1],
					'-',
					digits10[month], digits01[month],
					'-',
					digits10[day], digits01[day],
					' ',
					digits10[hour], digits01[hour],
					':',
					digits10[minute], digits01[minute],
					':',
					digits10[second], digits01[second],
				})

				if micro != 0 {
					micro10000 := micro / 10000
					micro100 := micro / 100 % 100
					micro1 := micro % 100

					result.Write([]byte{
						'.',
						digits10[micro10000], digits01[micro10000],
						digits10[micro100], digits01[micro100],
						digits10[micro1], digits01[micro1],
					})
				}
				result.WriteByte('\'')
			}

		default:
			return "", driver.ErrSkip
		}

		i = (s.paramOffsets[n] + 1)
	}

	if i < len(s.query) {
		result.WriteString(s.query[i:])
	}

	return result.String(), nil
}

func getParameterOffsets(query string) []int {
	var offsets []int

	for i := 0; i < len(query); {
		// keep looking for offsets
		pi := strings.Index(query[i:], "?")
		if pi == -1 {
			return offsets
		}
		offsets = append(offsets, pi+i)

		i += (pi + 1)
	}

	return offsets
}
