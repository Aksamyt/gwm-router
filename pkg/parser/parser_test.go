package parser

import (
	"reflect"
	"strings"
	"testing"
)

func indent(s string) string {
	return "    " + strings.ReplaceAll(s, "\n", "\n    ")
}

func mv(s ...string) map[string]struct{} {
	m := map[string]struct{}{}
	for _, v := range s {
		m[v] = struct{}{}
	}
	return m
}

func mid(s ...string) []string {
	return s
}

func TestExprStringer(t *testing.T) {
	for _, tt := range []struct {
		in       Expr
		expected string
	}{
		{Expr{Vars: []Var{{ID: mid("var")}}}, "{var}"},
		{Expr{Op: '+', Vars: []Var{
			{ID: mid("var")},
			{ID: mid("prefix"), Mod: ModPrefix + 12},
			{ID: mid("explode"), Mod: ModExplode},
		}}, "{+var,prefix:12,explode*}"},
	} {
		t.Run(tt.expected, func(t *testing.T) {
			got := tt.in.String()
			if got != tt.expected {
				t.Errorf("got:\n\t%q\nexpected:\n\t%q", got, tt.expected)
			}
		})
	}
}

func TestAstStringer(t *testing.T) {
	for _, tt := range []struct {
		in       Ast
		expected string
	}{
		{Ast{
			Vars: mv("a", "b", "c"),
			Parts: []interface{}{
				"raw",
				nil,
				Expr{Vars: []Var{{ID: mid("var")}}},
			},
		}, "VARS: [a b c]\n[\"raw\" / {var}]"},
	} {
		t.Run(tt.expected, func(t *testing.T) {
			got := tt.in.String()
			if got != tt.expected {
				t.Errorf("got:\n\t%q\nexpected:\n\t%q", got, tt.expected)
			}
		})
	}
}

func TestAst(t *testing.T) {
	for _, tt := range []struct {
		in       string
		expected Ast
	}{
		{"hello/world", Ast{
			Vars:  mv(),
			Parts: []interface{}{"hello", nil, "world"},
		}},
		{"hello//world", Ast{
			Vars:  mv(),
			Parts: []interface{}{"hello", nil, "world"},
		}},
		{"{var}", Ast{
			Vars: mv("var"),
			Parts: []interface{}{
				Expr{Vars: []Var{{ID: mid("var")}}},
			},
		}},
		{"a//{var}/a{var}a/a", Ast{
			Vars: mv("var"),
			Parts: []interface{}{
				"a",
				nil,
				Expr{Vars: []Var{{ID: mid("var")}}},
				nil,
				"a",
				Expr{Vars: []Var{{ID: mid("var")}}},
				"a",
				nil,
				"a",
			},
		}},
		{"{+var}", Ast{
			Vars: mv("var"),
			Parts: []interface{}{
				Expr{Op: '+', Vars: []Var{{ID: mid("var")}}},
			},
		}},
		{"{+path}/here", Ast{
			Vars: mv("path"),
			Parts: []interface{}{
				Expr{Op: '+', Vars: []Var{{ID: mid("path")}}},
				nil,
				"here",
			},
		}},
		{"here?ref={+path}", Ast{
			Vars: mv("path"),
			Parts: []interface{}{
				"here?ref=",
				Expr{Op: '+', Vars: []Var{{ID: mid("path")}}},
			},
		}},
		{"map?{x,y}", Ast{
			Vars: mv("x", "y"),
			Parts: []interface{}{
				"map?",
				Expr{Vars: []Var{
					{ID: mid("x")},
					{ID: mid("y")},
				}},
			},
		}},
		{"{x,hello,y}", Ast{
			Vars: mv("x", "hello", "y"),
			Parts: []interface{}{
				Expr{Vars: []Var{
					{ID: mid("x")},
					{ID: mid("hello")},
					{ID: mid("y")},
				}},
			},
		}},
		{"{+x,hello,y}", Ast{
			Vars: mv("x", "hello", "y"),
			Parts: []interface{}{
				Expr{Op: '+', Vars: []Var{
					{ID: mid("x")},
					{ID: mid("hello")},
					{ID: mid("y")},
				}},
			},
		}},
		{"{+path,x}/here", Ast{
			Vars: mv("path", "x"),
			Parts: []interface{}{
				Expr{Op: '+', Vars: []Var{
					{ID: mid("path")},
					{ID: mid("x")},
				}},
				nil,
				"here",
			},
		}},
		{"{#x,hello,y}", Ast{
			Vars: mv("x", "hello", "y"),
			Parts: []interface{}{
				Expr{Op: '#', Vars: []Var{
					{ID: mid("x")},
					{ID: mid("hello")},
					{ID: mid("y")},
				}},
			},
		}},
		{"{#path,x}/here", Ast{
			Vars: mv("path", "x"),
			Parts: []interface{}{
				Expr{Op: '#', Vars: []Var{
					{ID: mid("path")},
					{ID: mid("x")},
				}},
				nil,
				"here",
			},
		}},
		{"X{.var}", Ast{
			Vars: mv("var"),
			Parts: []interface{}{
				"X",
				Expr{Op: '.', Vars: []Var{{ID: mid("var")}}},
			},
		}},
		{"X{.x,y}", Ast{
			Vars: mv("x", "y"),
			Parts: []interface{}{
				"X",
				Expr{Op: '.', Vars: []Var{
					{ID: mid("x")},
					{ID: mid("y")},
				}},
			},
		}},
		{"{/var}", Ast{
			Vars: mv("var"),
			Parts: []interface{}{
				Expr{Op: '/', Vars: []Var{{ID: mid("var")}}},
			},
		}},
		{"{/var,x}/here", Ast{
			Vars: mv("var", "x"),
			Parts: []interface{}{
				Expr{Op: '/', Vars: []Var{
					{ID: mid("var")},
					{ID: mid("x")},
				}},
				nil,
				"here",
			},
		}},
		{"{;x,y}", Ast{
			Vars: mv("x", "y"),
			Parts: []interface{}{
				Expr{Op: ';', Vars: []Var{
					{ID: mid("x")},
					{ID: mid("y")},
				}},
			},
		}},
		{"{?x,y}", Ast{
			Vars: mv("x", "y"),
			Parts: []interface{}{
				Expr{Op: '?', Vars: []Var{
					{ID: mid("x")},
					{ID: mid("y")},
				}},
			},
		}},
		{"?fixed=yes{&x}", Ast{
			Vars: mv("x"),
			Parts: []interface{}{
				"?fixed=yes",
				Expr{Op: '&', Vars: []Var{{ID: mid("x")}}},
			},
		}},
		{"{var:3}", Ast{
			Vars: mv("var"),
			Parts: []interface{}{
				Expr{Vars: []Var{
					{ID: mid("var"), Mod: ModPrefix + 3},
				}},
			},
		}},
		{"{var:30}", Ast{
			Vars: mv("var"),
			Parts: []interface{}{
				Expr{Vars: []Var{
					{ID: mid("var"), Mod: ModPrefix + 30},
				}},
			},
		}},
		{"{list*}", Ast{
			Vars: mv("list"),
			Parts: []interface{}{
				Expr{Vars: []Var{
					{ID: mid("list"), Mod: ModExplode},
				}},
			},
		}},
		{"{+path:6}/here", Ast{
			Vars: mv("path"),
			Parts: []interface{}{
				Expr{Op: '+', Vars: []Var{
					{ID: mid("path"), Mod: ModPrefix + 6},
				}},
				nil,
				"here",
			},
		}},
		{"{+list*}", Ast{
			Vars: mv("list"),
			Parts: []interface{}{
				Expr{Op: '+', Vars: []Var{
					{ID: mid("list"), Mod: ModExplode},
				}},
			},
		}},
		{"{#path:6}/here", Ast{
			Vars: mv("path"),
			Parts: []interface{}{
				Expr{Op: '#', Vars: []Var{
					{ID: mid("path"), Mod: ModPrefix + 6},
				}},
				nil,
				"here",
			},
		}},
		{"{#list*}", Ast{
			Vars: mv("list"),
			Parts: []interface{}{
				Expr{Op: '#', Vars: []Var{
					{ID: mid("list"), Mod: ModExplode},
				}},
			},
		}},
		{"X{.var:3}", Ast{
			Vars: mv("var"),
			Parts: []interface{}{
				"X",
				Expr{Op: '.', Vars: []Var{
					{ID: mid("var"), Mod: ModPrefix + 3},
				}},
			},
		}},
		{"X{.list*}", Ast{
			Vars: mv("list"),
			Parts: []interface{}{
				"X",
				Expr{Op: '.', Vars: []Var{
					{ID: mid("list"), Mod: ModExplode},
				}},
			},
		}},
		{"{/var:1,var}", Ast{
			Vars: mv("var"),
			Parts: []interface{}{
				Expr{Op: '/', Vars: []Var{
					{ID: mid("var"), Mod: ModPrefix + 1},
					{ID: mid("var")},
				}},
			},
		}},
		{"{/list*}", Ast{
			Vars: mv("list"),
			Parts: []interface{}{
				Expr{Op: '/', Vars: []Var{
					{ID: mid("list"), Mod: ModExplode},
				}},
			},
		}},
		{"{/list*,path:4}", Ast{
			Vars: mv("list", "path"),
			Parts: []interface{}{
				Expr{Op: '/', Vars: []Var{
					{ID: mid("list"), Mod: ModExplode},
					{ID: mid("path"), Mod: ModPrefix + 4},
				}},
			},
		}},
		{"{;hello:5}", Ast{
			Vars: mv("hello"),
			Parts: []interface{}{
				Expr{Op: ';', Vars: []Var{
					{ID: mid("hello"), Mod: ModPrefix + 5},
				}},
			},
		}},
		{"{;list*}", Ast{
			Vars: mv("list"),
			Parts: []interface{}{
				Expr{Op: ';', Vars: []Var{
					{ID: mid("list"), Mod: ModExplode},
				}},
			},
		}},
		{"{?var:3}", Ast{
			Vars: mv("var"),
			Parts: []interface{}{
				Expr{Op: '?', Vars: []Var{
					{ID: mid("var"), Mod: ModPrefix + 3},
				}},
			},
		}},
		{"{?list*}", Ast{
			Vars: mv("list"),
			Parts: []interface{}{
				Expr{Op: '?', Vars: []Var{
					{ID: mid("list"), Mod: ModExplode},
				}},
			},
		}},
		{"{&var:3}", Ast{
			Vars: mv("var"),
			Parts: []interface{}{
				Expr{Op: '&', Vars: []Var{
					{ID: mid("var"), Mod: ModPrefix + 3},
				}},
			},
		}},
		{"{&list*}", Ast{
			Vars: mv("list"),
			Parts: []interface{}{
				Expr{Op: '&', Vars: []Var{
					{ID: mid("list"), Mod: ModExplode},
				}},
			},
		}},

		{"{foo.bar}", Ast{
			Vars: mv("foo"),
			Parts: []interface{}{
				Expr{Vars: []Var{{ID: mid("foo", "bar")}}},
			},
		}},
		{"{+foo.bar*,foo.jaj:9999}", Ast{
			Vars: mv("foo"),
			Parts: []interface{}{
				Expr{Op: '+', Vars: []Var{
					{
						ID:  mid("foo", "bar"),
						Mod: ModExplode,
					},
					{
						ID:  mid("foo", "jaj"),
						Mod: ModPrefix + 9999,
					},
				}},
			},
		}},
	} {
		t.Run(tt.in, func(t *testing.T) {
			got, err := New().Parse(tt.in)
			if err != nil {
				t.Errorf("error:\n%v", err)
				return
			}
			if !reflect.DeepEqual(got, &tt.expected) {
				t.Errorf("got:\n%s\nexpected:\n%s\ninput:\n    %s", indent(got.String()), indent(tt.expected.String()), tt.in)
			}
		})
	}
}
