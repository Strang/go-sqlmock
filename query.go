package sqlmock

import (
	"database/sql/driver"
	"fmt"
	"regexp"
	"strings"
)

// ErrShuffle defines an error value, which can be expected in case of
// the expected args needing to be shuffled.
type ErrShuffle struct {
	Shuffle map[int]int
}

func (e *ErrShuffle) Error() string {
	return fmt.Sprintf("Shuffle: %v", e.Shuffle)
}

// ShuffleArgs shuffle the expected args according to the Shuffle map.
func (e *ErrShuffle) ShuffleArgs(args []driver.Value) {
	shuffled := make([]driver.Value, len(args))
	for oldPos, x := range args {
		if newPos, ok := e.Shuffle[oldPos]; ok {
			shuffled[newPos] = x
		} else {
			shuffled[oldPos] = x
		}
	}

	for i, x := range shuffled {
		args[i] = x
	}
}

var re = regexp.MustCompile("\\s+")

// strip out new lines and trim spaces
func stripQuery(q string) (s string) {
	return strings.TrimSpace(re.ReplaceAllString(q, " "))
}

// QueryMatcher is an SQL query string matcher interface,
// which can be used to customize validation of SQL query strings.
// As an example, external library could be used to build
// and validate SQL ast, columns selected.
//
// sqlmock can be customized to implement a different QueryMatcher
// configured through an option when sqlmock.New or sqlmock.NewWithDSN
// is called, default QueryMatcher is QueryMatcherRegexp.
type QueryMatcher interface {

	// Match expected SQL query string without whitespace to
	// actual SQL.
	Match(expectedSQL, actualSQL string) error
}

// QueryMatcherFunc type is an adapter to allow the use of
// ordinary functions as QueryMatcher. If f is a function
// with the appropriate signature, QueryMatcherFunc(f) is a
// QueryMatcher that calls f.
type QueryMatcherFunc func(expectedSQL, actualSQL string) error

// Match implements the QueryMatcher
func (f QueryMatcherFunc) Match(expectedSQL, actualSQL string) error {
	return f(expectedSQL, actualSQL)
}

// QueryMatcherRegexp is the default SQL query matcher
// used by sqlmock. It parses expectedSQL to a regular
// expression and attempts to match actualSQL.
var QueryMatcherRegexp QueryMatcher = QueryMatcherFunc(func(expectedSQL, actualSQL string) error {
	expect := stripQuery(expectedSQL)
	actual := stripQuery(actualSQL)
	re, err := regexp.Compile(expect)
	if err != nil {
		return err
	}
	if !re.MatchString(actual) {
		return fmt.Errorf(`could not match actual sql: "%s" with expected regexp "%s"`, actual, re.String())
	}
	return nil
})

// QueryMatcherEqual is the SQL query matcher
// which simply tries a case sensitive match of
// expected and actual SQL strings without whitespace.
var QueryMatcherEqual QueryMatcher = QueryMatcherFunc(func(expectedSQL, actualSQL string) error {
	expect := stripQuery(expectedSQL)
	actual := stripQuery(actualSQL)
	if actual != expect {
		return fmt.Errorf(`actual sql: "%s" does not equal to expected "%s"`, actual, expect)
	}
	return nil
})
