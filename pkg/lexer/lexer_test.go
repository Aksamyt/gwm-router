package lexer

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
	items []Item
}

func collect(stream chan Item) (items []Item) {
	for {
		item, ok := <-stream
		if !ok {
			break
		}
		items = append(items, item)
	}
	return
}

func equal(i1, i2 []Item) bool {
	if len(i1) != len(i2) {
		return false
	}
	for k := range i1 {
		if i1[k].Typ != i2[k].Typ {
			return false
		}
		if i1[k].Val != i2[k].Val {
			return false
		}
	}
	return true
}

func sayError(t *testing.T, tt lexTest, items []Item) {
	t.Errorf("%s: got\n\t%+v\nexpected\n\t%v\ninput\n\t%q", tt.name, items, tt.items, tt.input)
}

var (
	tError   = func(msg string) Item { return Item{ItemError, msg, 0} }
	tSep     = Item{ItemSep, "/", 0}
	tLacc    = Item{ItemLacc, "{", 0}
	tRacc    = Item{ItemRacc, "}", 0}
	tOp      = func(op string) Item { return Item{ItemOp, op, 0} }
	tExplode = Item{ItemExplode, "*", 0}
	tPrefix  = Item{ItemPrefix, ":", 0}
	tLength  = func(n string) Item { return Item{ItemLength, n, 0} }
	tDot     = Item{ItemDot, ".", 0}
	tComma   = Item{ItemComma, ",", 0}
	tEOF     = Item{ItemEOF, "", 0}
	tRaw     = func(v string) Item { return Item{ItemRaw, v, 0} }
	tVar     = func(v string) Item { return Item{ItemVar, v, 0} }
)

func TestStringer(t *testing.T) {
	for _, tt := range []lexTest{
		{"invalid", "[]", []Item{{Typ: -1, Val: ""}}},
		{
			"itemError",
			fmt.Sprintf("[ERROR %s]", errorUnfinishedPercent()),
			[]Item{tError(errorUnfinishedPercent())},
		},
		{"itemSep", "[/]", []Item{tSep}},
		{"itemLacc", "[{]", []Item{tLacc}},
		{"itemRacc", "[}]", []Item{tRacc}},
		{"itemOp", "[+]", []Item{tOp("+")}},
		{"itemExplode", "[*]", []Item{tExplode}},
		{"itemPrefix", "[:]", []Item{tPrefix}},
		{"itemLength", "[1234]", []Item{tLength("1234")}},
		{"itemDot", "[.]", []Item{tDot}},
		{"itemComma", "[,]", []Item{tComma}},
		{`itemRaw("hello")`, `["hello"]`, []Item{tRaw("hello")}},
		{`itemVar("hello")`, `['hello']`, []Item{tVar("hello")}},
		{"itemEOF", "[EOF]", []Item{tEOF}},
	} {
		out := fmt.Sprintf("%+v", tt.items)
		if out != tt.input {
			t.Errorf("%s.String(): got\n\t%q\nexpected\n\t%q", tt.name, out, tt.input)
		}
	}
}

func TestSimple(t *testing.T) {
	for _, tt := range []lexTest{
		{"empty", "", []Item{tEOF}},
		{"letters", "hello", []Item{tRaw("hello"), tEOF}},
		{"number", "123", []Item{tRaw("123"), tEOF}},
		{"punctuation", "(yes)", []Item{tRaw("(yes)"), tEOF}},
	} {
		items := collect(Lex(tt.input))
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
		{"every legal raw characters", everyLegal, []Item{
			tRaw(everyLegal),
			tEOF,
		}},
	}
	for i := range everyIllegal {
		c := everyIllegal[i]
		expected := []Item{tError(errorIllegal(c))}
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
		items := collect(Lex(tt.input))
		if !equal(items, tt.items) {
			sayError(t, tt, items)
		}
	}

}

func TestRandomPercent(t *testing.T) {
	err := quick.Check(func(c byte) bool {
		input := fmt.Sprintf("%%%02x", c)
		expected := []Item{tRaw(string([]byte{c})), tEOF}
		items := collect(Lex(input))
		return equal(items, expected)
	}, nil)
	if e := (&quick.CheckError{}); errors.As(err, &e) {
		t.Errorf(`failed on input "%%%02x"`, e.In[0])
	}
}

func TestFailingPercent(t *testing.T) {
	for _, tt := range []lexTest{
		{"lonely %", "100%", []Item{
			tRaw("100"),
			tError(errorUnfinishedPercent()),
		}},
		{"unfinished %", "2%2", []Item{
			tRaw("2"),
			tError(errorUnfinishedPercent()),
		}},
		{"illegal character", "ohno%g2", []Item{
			tRaw("ohno"),
			tError(errorIllegalPercent('g')),
		}},
		{"illegal character", "%2h", []Item{
			tError(errorIllegalPercent('h')),
		}},
	} {
		items := collect(Lex(tt.input))
		if !equal(items, tt.items) {
			sayError(t, tt, items)
		}
	}
}

func TestRandomSlashes(t *testing.T) {
	values := func(v []reflect.Value, r *rand.Rand) {
		var a []Item
		var b strings.Builder
		for r.Intn(20) > 0 {
			if r.Intn(3) > 0 {
				if len(a) > 0 && a[len(a)-1].Typ == ItemRaw {
					a[len(a)-1].Val += "o"
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
	err := quick.Check(func(expected []Item, input string) bool {
		items := collect(Lex(input))
		return equal(items, expected)
	}, &quick.Config{Values: values})
	if e := (&quick.CheckError{}); errors.As(err, &e) {
		expected := e.In[0].([]Item)
		input := e.In[1].(string)
		items := collect(Lex(input))
		sayError(t, lexTest{"random slash", input, expected}, items)
	}
}

func TestVariableList(t *testing.T) {
	for _, tt := range []lexTest{
		{"single var", "{oui}", []Item{
			tLacc,
			tVar("oui"),
			tRacc,
			tEOF,
		}},
		{"var with dots", "{foo.bar}", []Item{
			tLacc,
			tVar("foo"),
			tDot,
			tVar("bar"),
			tRacc,
			tEOF,
		}},
		{"vars with commas", "{foo,bar}", []Item{
			tLacc,
			tVar("foo"),
			tComma,
			tVar("bar"),
			tRacc,
			tEOF,
		}},
		{"lots of things", "{a,,b.c_d,.12,}", []Item{
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
		items := collect(Lex(tt.input))
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
			[]Item{
				tLacc,
				tOp(string(c)),
				tRacc,
				tEOF,
			},
		})
	}

	for _, tt := range tests {
		items := collect(Lex(tt.input))
		if !equal(items, tt.items) {
			sayError(t, tt, items)
		}
	}
}

func TestSuffixOperators(t *testing.T) {
	for _, tt := range []lexTest{
		{"explode", "{boom*}", []Item{
			tLacc,
			tVar("boom"),
			tExplode,
			tRacc,
			tEOF,
		}},
		{"prefix 1", "{a:1}", []Item{tLacc, tVar("a"), tPrefix, tLength("1"), tRacc, tEOF}},
		{"prefix 2", "{a:27}", []Item{tLacc, tVar("a"), tPrefix, tLength("27"), tRacc, tEOF}},
		{"prefix 3", "{a:031}", []Item{tLacc, tVar("a"), tPrefix, tLength("031"), tRacc, tEOF}},
		{"prefix 4", "{a:9070}", []Item{tLacc, tVar("a"), tPrefix, tLength("9070"), tRacc, tEOF}},
		{"two of them", "{/list*,path:4}", []Item{
			tLacc,
			tOp("/"),
			tVar("list"),
			tExplode,
			tComma,
			tVar("path"),
			tPrefix,
			tLength("4"),
			tRacc,
			tEOF,
		}},
	} {
		items := collect(Lex(tt.input))
		if !equal(items, tt.items) {
			sayError(t, tt, items)
		}
	}
}

func TestWrongExpr(t *testing.T) {
	tests := []lexTest{
		{"nothing", "{}", []Item{tLacc, tError(errorEmptyExpr())}},
		{"unfinished", "{", []Item{
			tLacc,
			tError(errorUnfinishedExpr()),
		}},
		{"unfinished var", "{hello", []Item{
			tLacc,
			tVar("hello"),
			tError(errorUnfinishedExpr()),
		}},
		{"unfinished explode", "{hello*", []Item{
			tLacc,
			tVar("hello"),
			tExplode,
			tError(errorUnfinishedExpr()),
		}},
		{"space", "{ ", []Item{tLacc, tError(errorUnexpected(' '))}},
		{"space var", "{oi ", []Item{
			tLacc,
			tVar("oi"),
			tError(errorUnexpected(' ')),
		}},
		{"space explode", "{oi* ", []Item{
			tLacc,
			tVar("oi"),
			tExplode,
			tError(errorUnexpected(' ')),
		}},
		{"no length", "{a:}", []Item{
			tLacc,
			tVar("a"),
			tPrefix,
			tError(errorExpectedLength()),
		}},
	}
	for _, c := range "=,!@|" {
		tests = append(tests, lexTest{
			fmt.Sprintf("reserved op %c", c),
			fmt.Sprintf("{%c}", c),
			[]Item{
				tLacc,
				tError(errorReservedOp(byte(c))),
			},
		})
	}

	for _, tt := range tests {
		items := collect(Lex(tt.input))
		if !equal(items, tt.items) {
			sayError(t, tt, items)
		}
	}
}
