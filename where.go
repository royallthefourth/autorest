package bartlett

import (
	"encoding/csv"
	"fmt"
	sqrl "github.com/Masterminds/squirrel"
	"net/http"
	"strings"
)

type whereCond struct {
	Column    string // necessary due to the way Squirrel handles IN
	Condition string
	Value     string
}

func buildConds(t Table, r *http.Request) []whereCond {
	i := 0
	columns := make([]string, len(r.URL.Query()))
	for k := range r.URL.Query() {
		columns[i] = k
		i++
	}
	columns = t.validReadColumns(columns)
	conds := make([]whereCond, 0)

	for column, values := range r.URL.Query() {
		if sliceContains(columns, column) {
			for _, rawCond := range values {
				parsedCond, val := parseWhereCond(rawCond)
				var cond string
				if parsedCond == `in` || parsedCond == `not.in` {
					conds = append(conds, whereCond{column, parsedCond, val})
				} else {
					cond = urlToWhereCond(column, parsedCond)
					val := rectifyArg(cond, val)
					conds = append(conds, whereCond{column, cond, val})
				}
			}
		}
	}

	return conds
}

func parseOr(cond whereCond) sqrl.Or {
	if !parensMatch(cond.Value) {
		return sqrl.Or{}
	}
	// TODO handle trivial success of one pair of parens with no nesting
	// TODO split cond.Value on commas that are not between parens?
	return sqrl.Or{}
}

func parensMatch(parens string) bool {
	if parens[0] != '(' || parens[len(parens) - 1] != ')' {
		return false
	}

	foundParens := false
	weight := 0
	for _, c := range parens {
		if c == '(' {
			foundParens = true
			weight++
		} else if c == ')' {
				weight--
		}
		if weight < 0 {	// too many close parens
			return false
		}
	}

	return weight == 0 && foundParens
}

func parseSimpleWhereCond(rawCond string) (cond, val string) {
	parts := strings.Split(rawCond, `.`)
	if parts[0] == `not` {
		cond = fmt.Sprintf(`%s.%s`, parts[0], parts[1])
	} else {
		cond = parts[0]
	}

	val = strings.Replace(rawCond, cond+`.`, ``, 1)
	return cond, val
}

func parseWhereCond(rawCond string) (cond, val string) {
	if rawCond[0:2] == `or` {
		return `or`, val
	} else if rawCond[0:3] == `and` {
		return `and`, val
	} else {
		return parseSimpleWhereCond(rawCond)
	}
}

func rectifyArg(cond, val string) string {
	if strings.Contains(cond, `LIKE ?`) {
		val = strings.Replace(val, `*`, `%`, -1)
	}
	return val
}

var urlWhere map [string]string
func urlToWhereCond(column, condition string) string {
	if len(urlWhere) == 0 {
		urlWhere = make(map [string]string, 16)
		urlWhere[`eq`] = `%s = ?`
		urlWhere[`not.eq`] = `%s != ?`
		urlWhere[`neq`] = `%s != ?`
		urlWhere[`not.neq`] = `%s = ?`
		urlWhere[`gt`] = `%s > ?`
		urlWhere[`not.gt`] = `%s <= ?`
		urlWhere[`gte`] = `%s >= ?`
		urlWhere[`not.gte`] = `%s < ?`
		urlWhere[`lt`] = `%s < ?`
		urlWhere[`not.lt`] = `%s >= ?`
		urlWhere[`lte`] = `%s <= ?`
		urlWhere[`not.lte`] = `%s > ?`
		urlWhere[`like`] = `%s LIKE ?`
		urlWhere[`not.like`] = `%s NOT LIKE ?`
		urlWhere[`is`] = `%s IS ?`
		urlWhere[`not.is`] = `%s IS NOT ?`
	}
	if format, ok := urlWhere[condition]; ok {
		return fmt.Sprintf(format, column)
	}
	return ``
}

func whereIn(rawVal string) []string {
	r := csv.NewReader(strings.NewReader(strings.TrimPrefix(strings.TrimSuffix(rawVal, `)`), `(`)))
	vals, _ := r.Read()
	return vals
}
