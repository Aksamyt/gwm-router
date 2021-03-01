package execute

import (
	"bytes"
	"testing"

	"github.com/aksamyt/uritemplate/pkg/parser"
)

type ID struct{}

func (ID) String() string { return "270319070" }

type Name struct{}

func (Name) String() string { return "Gontrand" }

func TestReflection(t *testing.T) {
	ast, _ := parser.Parse("/hello{/id,Name}")
	expected := "/hello/270319070/Gontrand"

	for _, tt := range []struct {
		name string
		data interface{}
	}{
		{"map with string",
			map[string]string{
				"id":   "270319070",
				"Name": "Gontrand",
			},
		},
		{"map with int interface",
			map[string]interface{}{
				"id":   270319070,
				"Name": "Gontrand",
			},
		},
		{"map with Stringer",
			map[string]interface{}{
				"id":   ID{},
				"Name": "Gontrand",
			},
		},
		{"map with Stringer pointer interface",
			map[string]interface{}{
				"id":   &ID{},
				"Name": Name{},
			},
		},
		{"struct with string tag",
			struct {
				ID   string `uri:"id"`
				Name string
			}{ID: "270319070", Name: "Gontrand"},
		},
		{"struct with int interface tag",
			struct {
				ID   interface{} `uri:"id"`
				Name interface{}
			}{ID: 270319070, Name: Name{}},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			Execute(ast, &buf, tt.data)
			got := buf.String()
			if got != expected {
				t.Errorf("got:\n\t%q\nexpected:\n\t%q", got, expected)
			}
		})
	}
}

func TestNesting(t *testing.T) {
	ast, _ := parser.Parse("/hello{?person.firstName,person.lastName}")
	expected := "/hello?firstName=Gontrand&lastName=Fauxfilet"

	for _, tt := range []struct {
		name string
		data interface{}
	}{
		{"map",
			map[string]map[string]string{
				"person": {
					"firstName": "Gontrand",
					"lastName":  "Fauxfilet",
				},
			},
		},
		{"struct",
			struct {
				Person interface{} `uri:"person"`
			}{
				Person: struct {
					First string `uri:"firstName"`
					Last  string `uri:"lastName"`
				}{First: "Gontrand", Last: "Fauxfilet"},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			Execute(ast, &buf, tt.data)
			got := buf.String()
			if got != expected {
				t.Errorf("got:\n\t%q\nexpected:\n\t%q", got, expected)
			}
		})
	}
}

func TestFailSilently(t *testing.T) {
	ast, _ := parser.Parse("/hello/{name}")
	expected := "/hello/"
	var buf bytes.Buffer
	Execute(ast, &buf, struct{}{})
	got := buf.String()
	if got != expected {
		t.Errorf("got:\n\t%q\nexpected:\n\t%q", got, expected)
	}
}
