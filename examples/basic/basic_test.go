package main

import (
	"fmt"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/xwb1989/sqlparser"
)

// a successful case
func TestShouldUpdateStats(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE products").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT INTO product_viewers").WithArgs(2, 3).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	// now we execute our method
	if err = recordStats(db, 2, 3); err != nil {
		t.Errorf("error was not expected while updating stats: %s", err)
	}

	// we make sure that all expectations were met
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

// a failing test case
func TestShouldRollbackStatUpdatesOnFailure(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE products").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT INTO product_viewers").
		WithArgs(2, 3).
		WillReturnError(fmt.Errorf("some error"))
	mock.ExpectRollback()

	// now we execute our method
	if err = recordStats(db, 2, 3); err == nil {
		t.Errorf("was expecting an error, but there was none")
	}

	// we make sure that all expectations were met
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestShuffleArgsWithSqlparser(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(QueryMatcherSqlParser))

	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	mock.ExpectExec("UPDATE products SET views = ?, name = ?, price = ? WHERE id = ?").
		WithArgs(3, "book", 3.5, 1).
		WillReturnResult(sqlmock.NewResult(1, 1))

	if _, err := db.Exec("UPDATE products SET name = ?, price = ?, views = ? WHERE id = ?", "book", 3.5, 3, 1); err != nil {
		t.Errorf("error was not expected: %s", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

var QueryMatcherSqlParser sqlmock.QueryMatcher = sqlmock.QueryMatcherFunc(func(expectedSQL, actualSQL string) error {
	parsedExpected, err := sqlparser.Parse(expectedSQL)
	if err != nil {
		return fmt.Errorf("expectedSQL parse error: %v", err)
	}

	parsedActual, err := sqlparser.Parse(actualSQL)
	if err != nil {
		return fmt.Errorf("actualSQL parse error: %v", err)
	}

	expected, ok1 := parsedExpected.(*sqlparser.Update)
	actual, ok2 := parsedActual.(*sqlparser.Update)

	if ok1 && ok2 && len(expected.Exprs) == len(actual.Exprs) {
		actualSetClausePosition := make(map[string]int, 0)
		for i, expr := range actual.Exprs {
			actualSetClausePosition[expr.Name.Name.String()] = i
		}

		shuffled := false
		shuffle := make(map[int]int)

		expectedSetClause := make(sqlparser.UpdateExprs, len(expected.Exprs))
		for oldPos, setClause := range expected.Exprs {
			if newPos, ok := actualSetClausePosition[setClause.Name.Name.String()]; ok {
				if oldPos != newPos {
					if v, ok := setClause.Expr.(*sqlparser.SQLVal); ok {
						shuffled = true
						v.Val = []byte(fmt.Sprintf(":v%d", newPos+1))
						expectedSetClause[newPos] = setClause
						shuffle[oldPos] = newPos
					}
				}
			}
		}

		if shuffled {
			expected.Exprs = expectedSetClause
		}

		buf1 := sqlparser.NewTrackedBuffer(nil)
		expected.Format(buf1)

		buf2 := sqlparser.NewTrackedBuffer(nil)
		actual.Format(buf2)

		if buf1.String() == buf2.String() && shuffled {
			return &sqlmock.ErrShuffle{Shuffle: shuffle}
		}
	}

	return sqlmock.QueryMatcherRegexp.Match(expectedSQL, actualSQL)
})
