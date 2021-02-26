package execute

import (
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"
	"uritemplate/pkg/parser"
)

type Example struct {
	Level     int                    `json:"level"`
	Variables map[string]interface{} `json:"variables"`
	TestCases [][2]interface{}       `json:"testcases"`
}

type Fixture map[string]Example

func TestSpecExamples(t *testing.T) {
	f, err := os.Open("testdata/spec-examples-by-section.json")
	if err != nil {
		panic(err)
	}
	j, err := io.ReadAll(f)
	if err != nil {
		panic(err)
	}
	var fixture Fixture
	if err := json.Unmarshal(j, &fixture); err != nil {
		panic(err)
	}
	for k, e := range fixture {
		t.Run(k, func(t *testing.T) { runExample(t, e) })
	}
}

func runExample(t *testing.T, e Example) {
	for _, tt := range e.TestCases {
		input := tt[0].(string)
		var cases []string
		switch i := tt[1].(type) {
		case string:
			cases = append(cases, i)
		case []interface{}:
			for _, tc := range i {
				cases = append(cases, tc.(string))
			}
		}
		t.Run(input, func(t *testing.T) {
			runCases(t, input, e.Variables, cases)
		})
	}
}

func runCases(
	t *testing.T,
	input string,
	data map[string]interface{},
	cases []string,
) {
	var out strings.Builder
	ast, err := parser.Parse(input)
	if err != nil {
		t.Errorf("parser error: %v", err)
	}
	if err := Execute(ast, &out, data); err != nil {
		t.Errorf("execute error: %v", err)
	}
	got := out.String()
	for _, expected := range cases {
		if got == expected {
			return
		}
	}
	t.Errorf("got:\n\t%q\nexpected any of:\n\t%#v\ninput:\n\t%q", got, cases, input)
}

func TestInvalidWriter(t *testing.T) {
	pin, pout := io.Pipe()
	pin.Close()
	defer pout.Close()
	for _, template := range []string{
		"/",
		"test",
		"{var}",
	} {
		t.Run(template, func(t *testing.T) {
			ast, _ := parser.Parse(template)
			err := Execute(ast, pout, nil)
			if err == nil {
				t.Error("expected an error")
			}
		})
	}
}
