/*
  This file is part of the uritemplate project.
  Copyright (C) 2021 Alexandre Szymocha (@Aksamyt).

  This Source Code Form is subject to the terms of the Mozilla Public
  License, v. 2.0. If a copy of the MPL was not distributed with this
  file, You can obtain one at http://mozilla.org/MPL/2.0/.
*/

// Package execute provides functions for rendering parsed URI templates.
package execute

import (
	"bytes"
	"fmt"
	"io"
	"reflect"
	"uritemplate/pkg/escape"
	"uritemplate/pkg/parser"
)

type exprWriter struct {
	buf    bytes.Buffer  // used to do a single write and to implement some operator’s quirks
	data   reflect.Value // the original data passed to Execute
	expr   *parser.Expr  // the expression being printed
	i      int           // the number of defined variables written
	varsep byte          // the variable separator defined by the operator
	mask   byte          // the mask given to escape.Escape defined by the operator
}

func (e *exprWriter) writeListSeparator() {
	e.buf.WriteByte(',')
}

func (e *exprWriter) writeVariableSeparator() {
	if e.i > 0 {
		e.buf.WriteByte(e.varsep)
	}
}

// formatValue is where the Prefix modifier is checked for.
func (e *exprWriter) formatValue(value reflect.Value, mod parser.Mod) {
	unescaped := fmt.Sprint(value)
	if mod&parser.ModPrefix != 0 {
		if l := int(mod ^ parser.ModPrefix); l < len(unescaped) {
			unescaped = unescaped[:l]
		}
	}
	e.buf.WriteString(escape.Escape(unescaped, e.mask))
}

// Increments the variable counter.
func (e *exprWriter) formatList(value reflect.Value, mod parser.Mod) {
	if value.Len() > 0 {
		e.formatValue(value.Index(0), mod)
		for i := 1; i < value.Len(); i++ {
			e.writeListSeparator()
			e.formatValue(value.Index(i), mod)
		}
		e.i++
	}
}

// Increments the variable counter.
func (e *exprWriter) writeVariableValue(value reflect.Value, mod parser.Mod) {
	e.formatValue(value, mod)
	e.i++
}

func (e *exprWriter) writeValueAsKey(value reflect.Value) {
	e.formatValue(value, 0)
	e.buf.WriteByte('=')
}

func (e *exprWriter) writeVariableKey(v *parser.Var) {
	e.buf.WriteString(v.ID[len(v.ID)-1])
	e.buf.WriteByte('=')
}

// writeKvVariable writes a variable’s value in a key/value context.
// Exploded iterable values are treated as if they were a collection of values
// registered under the same key, which is the variable’s name.
func (e *exprWriter) writeKvVariable(v *parser.Var) {
	value := findVariableValue(e.data, v)

	// value was probably a nil interface{}, treat it as undef
	if !value.IsValid() {
		return
	}

	switch value.Kind() {
	case reflect.Slice:
		if v.Mod&parser.ModExplode == 0 {
			e.writeVariableSeparator()
			e.writeVariableKey(v)
			e.formatList(value, v.Mod)
		} else {
			// treat each child as a separate variable
			for i := 0; i < value.Len(); i++ {
				e.writeVariableSeparator()
				e.writeVariableKey(v)
				e.writeVariableValue(value.Index(i), 0)
			}
		}
	case reflect.Map:
		if v.Mod&parser.ModExplode == 0 {
			e.writeVariableKey(v)
			for it := value.MapRange(); it.Next(); {
				if e.i > 0 {
					e.writeListSeparator()
				}
				e.writeVariableValue(it.Key(), 0)
				e.writeListSeparator()
				e.writeVariableValue(it.Value(), 0)
			}
		} else {
			for it := value.MapRange(); it.Next(); {
				e.writeVariableSeparator()
				e.writeValueAsKey(it.Key())
				e.writeVariableValue(it.Value(), 0)
			}
		}
	default:
		e.writeVariableSeparator()
		e.writeVariableKey(v)
		lenBefore := e.buf.Len()
		e.writeVariableValue(value, v.Mod)
		// path operator keys must not have an equals sign if the
		// variable is visibly empty
		if e.expr.Op == ';' && e.buf.Len() == lenBefore {
			e.buf.Truncate(lenBefore - 1)
		}
	}
}

// writeListVariable writes a variable’s value in a list context.
func (e *exprWriter) writeListVariable(v *parser.Var) {
	value := findVariableValue(e.data, v)

	// value was probably a nil interface{}, treat it as undef
	if !value.IsValid() {
		return
	}

	switch value.Kind() {
	case reflect.Slice:
		if v.Mod&parser.ModExplode == 0 {
			e.formatList(value, v.Mod)
		} else {
			// treat each child as a separate variable
			for i := 0; i < value.Len(); i++ {
				e.writeVariableSeparator()
				e.writeVariableValue(value.Index(i), 0)
			}
		}
	case reflect.Map:
		if v.Mod&parser.ModExplode == 0 {
			for it := value.MapRange(); it.Next(); {
				if e.i > 0 {
					e.writeListSeparator()
				}
				e.writeVariableValue(it.Key(), 0)
				e.writeListSeparator()
				e.writeVariableValue(it.Value(), 0)
			}
		} else {
			for it := value.MapRange(); it.Next(); {
				e.writeVariableSeparator()
				e.writeValueAsKey(it.Key())
				e.writeVariableValue(it.Value(), 0)
			}
		}
	default:
		e.writeVariableSeparator()
		e.writeVariableValue(value, v.Mod)
	}
}

// writeExpr initializes the state of its receiver and calls the right write
// function depending on the context given by the operator.
func (e *exprWriter) writeExpr() {
	switch e.expr.Op {
	case '+', '#':
		e.varsep, e.mask = ',', escape.Disallowed
	case '.':
		e.varsep, e.mask = '.', escape.Disallowed|escape.Reserved
	case '/':
		e.varsep, e.mask = '/', escape.Disallowed|escape.Reserved
	case ';':
		e.varsep, e.mask = ';', escape.Disallowed|escape.Reserved
	case '?', '&':
		e.varsep, e.mask = '&', escape.Disallowed|escape.Reserved
	default:
		e.varsep, e.mask = ',', escape.Disallowed|escape.Reserved
	}

	// almost all operators have their sign written
	if e.expr.Op != 0 && e.expr.Op != '+' {
		e.buf.WriteByte(e.expr.Op)
	}

	switch e.expr.Op {
	case ';', '?', '&':
		for i := range e.expr.Vars {
			e.writeKvVariable(&e.expr.Vars[i])
		}
	default:
		for i := range e.expr.Vars {
			e.writeListVariable(&e.expr.Vars[i])
		}
		// some operators require at least one defined variable
		if e.i == 0 && (e.expr.Op == '#' || e.expr.Op == '.') {
			e.buf.Reset()
		}
	}
}

// Execute applies a parsed uritemplate to the specified data object,
// and writes the output to w.
//
// data can be a reflect.Value.
func Execute(ast *parser.Ast, w io.Writer, data interface{}) error {
	value, ok := data.(reflect.Value)
	if !ok {
		value = reflect.ValueOf(data)
	}
	for _, part := range ast.Parts {
		switch part := part.(type) {
		case parser.Expr:
			ew := exprWriter{data: value, expr: &part}
			ew.writeExpr()
			if _, err := w.Write(ew.buf.Bytes()); err != nil {
				return err
			}
		case string:
			if _, err := w.Write([]byte(part)); err != nil {
				return err
			}
		case nil:
			if _, err := w.Write([]byte{'/'}); err != nil {
				return err
			}
		}
	}
	return nil
}
