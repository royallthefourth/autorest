package bartlett

import (
	"errors"
	"fmt"
	sqrl "github.com/Masterminds/squirrel"
	"net/http"
	"strconv"
	"strings"
)

func (b Bartlett) buildUpdate(t Table, r *http.Request, userID interface{}, body []byte) (sqrl.UpdateBuilder, error) {
	query := t.prepareUpdate(body, userID, sqrl.Update(t.Name))
	query, err := updateWhere(query, t, r)
	if err != nil {
		return query, err
	}
	query = updateOrder(query, t, r)
	query = updateLimit(query, r)

	if t.UserID != `` && userID != nil {
		query = query.Where(sqrl.Eq{t.UserID: userID})
	}

	return query, nil
}

func updateLimit(query sqrl.UpdateBuilder, r *http.Request) sqrl.UpdateBuilder {
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

func updateOrder(query sqrl.UpdateBuilder, t Table, r *http.Request) sqrl.UpdateBuilder {
	for _, col := range parseOrder(t, r) {
		query = query.OrderBy(fmt.Sprintf(`%s %s`, col.Column, strings.ToUpper(col.Direction)))
	}

	return query
}

func updateWhere(query sqrl.UpdateBuilder, t Table, r *http.Request) (sqrl.UpdateBuilder, error) {
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
		err = errors.New(`UPDATE operations must have at least one WHERE clause`)
	}
	return query, err
}
