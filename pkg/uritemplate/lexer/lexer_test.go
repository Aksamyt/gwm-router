package uritemplatelexer

import (
	"errors"
	"fmt"
	"math/rand"
	"reflect"
	"strings"
	"testing"
	"testing/quick"
)

type lexTest struct {
	name  string
	input string
	items []item
}

func collect(l *lexer) (items []item) {
	for {
		item, ok := <-l.items
		if !ok {
			break
		}
		items = append(items, item)
	}
	return
}

func equal(i1, i2 []item) bool {
	if len(i1) != len(i2) {
		return false
	}
	for k := range i1 {
		if i1[k].typ != i2[k].typ {
			return false
		}
		if i1[k].val != i2[k].val {
			return false
		}
	}
	return true
}

func sayError(t *testing.T, tt lexTest, items []item) {
	t.Errorf("%s: got\n\t%+v\nexpected\n\t%v\ninput\n\t%q", tt.name, items, tt.items, tt.input)
}

var (
	tError = func(msg string) item { return item{itemError, msg} }
	tSep   = item{itemSep, "/"}
	tLacc  = item{itemLacc, "{"}
	tRacc  = item{itemRacc, "}"}
	tOp    = func(op string) item { return item{itemOp, op} }
	tDot   = item{itemDot, "."}
	tComma = item{itemComma, ","}
	tEOF   = item{itemEOF, ""}
	tRaw   = func(v string) item { return item{itemRaw, v} }
	tVar   = func(v string) item { return item{itemVar, v} }
)

func TestStringer(t *testing.T) {
	for _, tt := range []lexTest{
		{"invalid", "[]", []item{{typ: -1, val: ""}}},
		{
			"itemError",
			fmt.Sprintf("[ERROR %s]", errorUnfinishedPercent()),
			[]item{tError(errorUnfinishedPercent())},
		},
		{"itemSep", "[/]", []item{tSep}},
		{"itemLacc", "[{]", []item{tLacc}},
		{"itemRacc", "[}]", []item{tRacc}},
		{"itemOp", "[+]", []item{tOp("+")}},
		{"itemDot", "[.]", []item{tDot}},
		{"itemComma", "[,]", []item{tComma}},
		{`itemRaw("hello")`, `["hello"]`, []item{tRaw("hello")}},
		{`itemVar("hello")`, `['hello']`, []item{tVar("hello")}},
		{"itemEOF", "[EOF]", []item{tEOF}},
	} {
		out := fmt.Sprintf("%+v", tt.items)
		if out != tt.input {
			t.Errorf("%s.String(): got\n\t%q\nexpected\n\t%q", tt.name, out, tt.input)
		}
	}
}

func TestSimple(t *testing.T) {
	for _, tt := range []lexTest{
		{"empty", "", []item{tEOF}},
		{"letters", "hello", []item{tRaw("hello"), tEOF}},
		{"number", "123", []item{tRaw("123"), tEOF}},
		{"punctuation", "(yes)", []item{tRaw("(yes)"), tEOF}},
	} {
		items := collect(lex(tt.input))
		if !equal(items, tt.items) {
			sayError(t, tt, items)
		}
	}
}

func TestEveryRawCharacter(t *testing.T) {
	var (
		everyLegal   = "!#$&()*+,-.0123456789:;=?@ABCDEFGHIJKLMNOPQRSTUVWXYZ[]_abcdefghijklmnopqrstuvwxyz~\x7f\x80\x81\x82\x83\x84\x85\x86\x87\x88\x89\x8a\x8b\x8c\x8d\x8e\x8f\x90\x91\x92\x93\x94\x95\x96\x97\x98\x99\x9a\x9b\x9c\x9d\x9e\x9f ¡¢£¤¥¦§¨©ª«¬­®¯°±²³´µ¶·¸¹º»¼½¾¿ÀÁÂÃÄÅÆÇÈÉÊËÌÍÎÏÐÑÒÓÔÕÖ×ØÙÚÛÜÝÞßàáâãäåæçèéêëìíîïðñòóôõö÷øùúûüýþÿ"
		everyIllegal = "\x00\x01\x02\x03\x04\x05\x06\x07\x08\x09\x0a\x0b\x0c\x0d\x0e\x0f\x10\x11\x12\x13\x14\x15\x16\x17\x18\x19\x1a\x1b\x1c\x1d\x1e\x1f '<>^|}`" + `\"`
	)
	tests := []lexTest{
		{"every legal raw characters", everyLegal, []item{
			tRaw(everyLegal),
			tEOF,
		}},
	}
	for i := range everyIllegal {
		c := everyIllegal[i]
		expected := []item{tError(errorIllegal(c))}
		tests = append(tests,
			lexTest{
				fmt.Sprint("illegal", c),
				fmt.Sprintf("a%c", c),
				expected,
			},
			lexTest{
				fmt.Sprint("illegal", c),
				fmt.Sprintf("%ca", c),
				expected,
			},
		)
	}

	for _, tt := range tests {
		items := collect(lex(tt.input))
		if !equal(items, tt.items) {
			sayError(t, tt, items)
		}
	}

}

func TestRandomPercent(t *testing.T) {
	err := quick.Check(func(c byte) bool {
		input := fmt.Sprintf("%%%02x", c)
		expected := []item{tRaw(string([]byte{c})), tEOF}
		items := collect(lex(input))
		return equal(items, expected)
	}, nil)
	if e := (&quick.CheckError{}); errors.As(err, &e) {
		t.Errorf(`failed on input "%%%02x"`, e.In[0])
	}
}

func TestFailingPercent(t *testing.T) {
	for _, tt := range []lexTest{
		{"lonely %", "100%", []item{
			tRaw("100"),
			tError(errorUnfinishedPercent()),
		}},
		{"unfinished %", "2%2", []item{
			tRaw("2"),
			tError(errorUnfinishedPercent()),
		}},
		{"illegal character", "ohno%g2", []item{
			tRaw("ohno"),
			tError(errorIllegalPercent('g')),
		}},
		{"illegal character", "%2h", []item{
			tError(errorIllegalPercent('h')),
		}},
	} {
		items := collect(lex(tt.input))
		if !equal(items, tt.items) {
			sayError(t, tt, items)
		}
	}
}

func TestRandomSlashes(t *testing.T) {
	values := func(v []reflect.Value, r *rand.Rand) {
		var a []item
		var b strings.Builder
		for r.Intn(20) > 0 {
			if r.Intn(3) > 0 {
				if len(a) > 0 && a[len(a)-1].typ == itemRaw {
					a[len(a)-1].val += "o"
				} else {
					a = append(a, tRaw("o"))
				}
				b.WriteString("o")
			} else {
				a = append(a, tSep)
				b.WriteByte('/')
			}
		}
		if len(a) == 0 {
			a = append(a, tSep)
			b.WriteByte('/')
		}
		v[0] = reflect.ValueOf(append(a, tEOF))
		v[1] = reflect.ValueOf(b.String())
	}
	err := quick.Check(func(expected []item, input string) bool {
		items := collect(lex(input))
		return equal(items, expected)
	}, &quick.Config{Values: values})
	if e := (&quick.CheckError{}); errors.As(err, &e) {
		expected := e.In[0].([]item)
		input := e.In[1].(string)
		items := collect(lex(input))
		sayError(t, lexTest{"random slash", input, expected}, items)
	}
}

func TestVariableList(t *testing.T) {
	for _, tt := range []lexTest{
		{"single var", "{oui}", []item{
			tLacc,
			tVar("oui"),
			tRacc,
			tEOF,
		}},
		{"var with dots", "{foo.bar}", []item{
			tLacc,
			tVar("foo"),
			tDot,
			tVar("bar"),
			tRacc,
			tEOF,
		}},
		{"vars with commas", "{foo,bar}", []item{
			tLacc,
			tVar("foo"),
			tComma,
			tVar("bar"),
			tRacc,
			tEOF,
		}},
		{"lots of things", "{a,,b.c_d,.12,}", []item{
			tLacc,
			tVar("a"),
			tComma,
			tComma,
			tVar("b"),
			tDot,
			tVar("c_d"),
			tComma,
			tDot,
			tVar("12"),
			tComma,
			tRacc,
			tEOF,
		}},
	} {
		items := collect(lex(tt.input))
		if !equal(items, tt.items) {
			sayError(t, tt, items)
		}
	}
}

func TestPrefixOperators(t *testing.T) {
	var tests []lexTest
	for _, c := range "+#./;?&" {
		tests = append(tests, lexTest{
			fmt.Sprintf("op %c", c),
			fmt.Sprintf("{%c}", c),
			[]item{
				tLacc,
				tOp(string(c)),
				tRacc,
				tEOF,
			},
		})
	}

	for _, tt := range tests {
		items := collect(lex(tt.input))
		if !equal(items, tt.items) {
			sayError(t, tt, items)
		}
	}
}

func TestWrongExpr(t *testing.T) {
	tests := []lexTest{
		{"nothing", "{}", []item{tLacc, tError(errorEmptyExpr())}},
		{"unfinished", "{", []item{
			tLacc,
			tError(errorUnfinishedExpr()),
		}},
		{"unfinished 2", "{hello", []item{
			tLacc,
			tVar("hello"),
			tError(errorUnfinishedExpr()),
		}},
		{"space", "{ ", []item{tLacc, tError(errorUnexpected(' '))}},
		{"space 2", "{oi ", []item{
			tLacc,
			tVar("oi"),
			tError(errorUnexpected(' ')),
		}},
	}
	for _, c := range "=,!@|" {
		tests = append(tests, lexTest{
			fmt.Sprintf("reserved op %c", c),
			fmt.Sprintf("{%c}", c),
			[]item{
				tLacc,
				tError(errorReservedOp(byte(c))),
			},
		})
	}

	for _, tt := range tests {
		items := collect(lex(tt.input))
		if !equal(items, tt.items) {
			sayError(t, tt, items)
		}
	}
}
