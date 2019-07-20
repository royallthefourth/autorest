package bartlett

import (
	"testing"
)

func TestParseSimpleWhereCond(t *testing.T) {
	cond, val := parseSimpleWhereCond(`eq.90`)
	if cond != `eq` || val != `90` {
		t.Errorf(`Expected eq, 90 but got %s, %s`, cond, val)
	}

	cond, val = parseSimpleWhereCond(`not.eq.90`)
	if cond != `not.eq` || val != `90` {
		t.Errorf(`Expected not.eq, 90 but got %s, %s`, cond, val)
	}

	cond, val = parseSimpleWhereCond(`not.eq.hello,how.are.you`)
	if cond != `not.eq` || val != `hello,how.are.you` {
		t.Errorf(`Expected not.eq, hello,how.are.you but got %s, %s`, cond, val)
	}
}

func TestWhereIn(t *testing.T) {
	vals := whereIn(`10,20,30`)
	if vals[0] != `10` {
		t.Errorf(`Expected [10 20 30] but got %+v`, vals)
	}
}

func TestParensMatch(t *testing.T) {
	tests := []struct{
		in string
		out bool
	}{
		{`(asdf`, false},
		{`)asdf`, false},
		{`(asdf)`, true},
		{`((asdf))`, true},
		{`(asdf))`, false},
		{`(asd)f`, false},
		{`(asd)f)`, false},
	}

	for _, test := range tests {
		if parensMatch(test.in) != test.out {
			t.Errorf(`Expected %t for %s`, test.out, test.in)
		}
	}
}
