package bartlett

import (
	"errors"
	"fmt"
	sqrl "github.com/Masterminds/squirrel"
	"net/http"
	"strconv"
	"strings"
)

func (b Bartlett) buildDelete(t Table, r *http.Request) (sqrl.DeleteBuilder, error) {
	query, err := deleteWhere(sqrl.Delete(t.Name), t, r)
	if err != nil {
		return query, err
	}
	query = deleteOrder(query, t, r)
	query = deleteLimit(query, r)

	if t.UserID != `` {
		userID, err := b.Users(r)
		if err != nil {
			return query, err
		}
		query = query.Where(sqrl.Eq{t.UserID: userID})
	}

	return query, nil
}

func deleteLimit(query sqrl.DeleteBuilder, r *http.Request) sqrl.DeleteBuilder {
	var (
		err   error
		limit int
	)
	if len(r.URL.Query()[`limit`]) > 0 {
		limit, err = strconv.Atoi(r.URL.Query()[`limit`][0])
		if err != nil || limit < 0 {
			limit = 0
		}
	}
	if limit > 0 {
		query = query.Limit(uint64(limit))
	}

	return query
}

func deleteOrder(query sqrl.DeleteBuilder, t Table, r *http.Request) sqrl.DeleteBuilder {
	for _, col := range parseOrder(t, r) {
		query = query.OrderBy(fmt.Sprintf(`%s %s`, col.Column, strings.ToUpper(col.Direction)))
	}

	return query
}

func deleteWhere(query sqrl.DeleteBuilder, t Table, r *http.Request) (sqrl.DeleteBuilder, error) {
	whereClauses := 0

	for _, cond := range buildConds(t, r) {
		if cond.Condition == `not.in` {
			query = query.Where(sqrl.NotEq{cond.Column: whereIn(cond.Value)})
		} else if cond.Condition == `in` {
			query = query.Where(sqrl.Eq{cond.Column: whereIn(cond.Value)})
		} else {
			query = query.Where(cond.Condition, cond.Value)
		}
		whereClauses++
	}

	var err error = nil
	if whereClauses == 0 {
		err = errors.New(`DELETE operations must have at least one WHERE clause`)
	}
	return query, err
}
